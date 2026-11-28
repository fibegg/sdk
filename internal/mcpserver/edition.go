package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	routeManifestTimeout       = 2 * time.Second
	routeManifestRetryInterval = 30 * time.Second
)

type routeManifest struct {
	Edition  string `json:"edition"`
	Profile  string `json:"profile"`
	Major    int    `json:"major"`
	Revision *int   `json:"revision"`
	Routes   []struct {
		Path    string   `json:"path"`
		Methods []string `json:"methods"`
	} `json:"routes"`
	paths map[apiRoute]bool
}

type apiRoute struct {
	method string
	path   string
}

func routeKey(method, path string) apiRoute {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if len(segment) > 2 && strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			segments[i] = "{}"
		}
	}
	return apiRoute{method, strings.Join(segments, "/")}
}

type editionProbe struct {
	done     chan struct{}
	manifest *routeManifest
	retryAt  time.Time
}

type editionDiscovery struct {
	mu           sync.Mutex
	domains      map[string]*editionProbe
	fallbackSeen map[string]string // active MCP session ID to the domain that timed out or failed
	client       http.Client
	ctx          context.Context
}

func (s *Server) startEditionDiscovery(ctx context.Context) context.CancelFunc {
	if s.cfg.Edition == "full" {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	s.editions = &editionDiscovery{
		ctx:          ctx,
		domains:      make(map[string]*editionProbe),
		fallbackSeen: make(map[string]string),
		client: http.Client{
			Timeout: routeManifestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	s.startManifestProbe(s.baseCli.BaseURL())
	return cancel
}

func (s *Server) startManifestProbe(baseURL string) *editionProbe {
	d := s.editions
	d.mu.Lock()
	defer d.mu.Unlock()
	probe := d.domains[baseURL]
	if probe != nil && (probe.retryAt.IsZero() || time.Now().Before(probe.retryAt)) {
		return probe
	}
	probe = &editionProbe{done: make(chan struct{})}
	d.domains[baseURL] = probe
	go s.loadManifestProbe(d, baseURL, probe)
	return probe
}

func (s *Server) loadManifestProbe(d *editionDiscovery, baseURL string, probe *editionProbe) {
	manifest, answered, err := d.fetch(baseURL)
	if err != nil && s.cfg.Debug {
		// An invalid URL or transport error can contain user input.
		log.Print("[fibe-mcp] DEBUG: route manifest unavailable; using the full tool surface")
	}
	d.mu.Lock()
	if !answered {
		probe.retryAt = time.Now().Add(routeManifestRetryInterval)
	}
	var listeners []string
	if manifest != nil && manifest.Edition == "community" {
		probe.manifest = manifest
	}
	if answered {
		for session, domain := range d.fallbackSeen {
			if domain != baseURL {
				continue
			}
			delete(d.fallbackSeen, session)
			if probe.manifest != nil {
				listeners = append(listeners, session)
			}
		}
	}
	close(probe.done)
	d.mu.Unlock()
	for _, session := range listeners {
		_ = s.mcp.SendNotificationToSpecificClient(session, mcp.MethodNotificationToolsListChanged, nil)
	}
}

func (s *Server) manifestFor(ctx context.Context, baseURL string) *routeManifest {
	d := s.editions
	if d == nil {
		return nil
	}
	probe := s.startManifestProbe(baseURL)
	select {
	case <-ctx.Done():
		return nil
	case <-probe.done:
	}
	if session := mcpserver.ClientSessionFromContext(ctx); session != nil {
		d.mu.Lock()
		if !probe.retryAt.IsZero() {
			// A very late reader can arrive after a new probe has completed.
			// In that case the new result already owns notification state.
			if d.domains[baseURL] == probe {
				d.fallbackSeen[session.SessionID()] = baseURL
			}
		} else {
			delete(d.fallbackSeen, session.SessionID())
		}
		d.mu.Unlock()
	}
	return probe.manifest
}

func (d *editionDiscovery) fetch(baseURL string) (*routeManifest, bool, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "http" && u.Scheme != "https") {
		return nil, false, errors.New("invalid route manifest origin")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/route-manifest.json"
	req, err := http.NewRequestWithContext(d.ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, false, err
	}
	res, err := d.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, true, fmt.Errorf("route manifest HTTP status %d", res.StatusCode)
	}
	const maxManifestSize = 1 << 20
	data, err := io.ReadAll(io.LimitReader(res.Body, maxManifestSize+1))
	if err != nil {
		return nil, false, err
	}
	var manifest routeManifest
	if len(data) > maxManifestSize || json.Unmarshal(data, &manifest) != nil ||
		manifest.Edition == "" ||
		manifest.Profile == "" || manifest.Major < 1 || manifest.Revision == nil ||
		*manifest.Revision < 0 || len(manifest.Routes) == 0 {
		return nil, true, errors.New("invalid route manifest")
	}
	manifest.paths = make(map[apiRoute]bool, len(manifest.Routes))
	for _, route := range manifest.Routes {
		if !strings.HasPrefix(route.Path, "/") || strings.ContainsAny(route.Path, "?#") || len(route.Methods) == 0 {
			return nil, true, errors.New("invalid route manifest path")
		}
		for _, method := range route.Methods {
			if method == "" || strings.ContainsAny(method, " \t\r\n") || method != strings.ToUpper(method) {
				return nil, true, errors.New("invalid route manifest method")
			}
			manifest.paths[routeKey(method, route.Path)] = true
		}
	}
	return &manifest, true, nil
}
