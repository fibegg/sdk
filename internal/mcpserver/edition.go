package mcpserver

import (
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
)

const routeManifestTimeout = 350 * time.Millisecond

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

type editionDiscovery struct {
	mu      sync.Mutex
	domains map[string]func() *routeManifest
	client  http.Client
}

func (s *Server) startEditionDiscovery() {
	if s.cfg.Edition == "full" {
		return
	}
	s.editions = &editionDiscovery{
		domains: make(map[string]func() *routeManifest),
		client: http.Client{
			Timeout: routeManifestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	s.manifestFor(s.baseCli.BaseURL())
}

func (s *Server) manifestFor(baseURL string) *routeManifest {
	d := s.editions
	if d == nil {
		return nil
	}

	d.mu.Lock()
	load := d.domains[baseURL]
	if load == nil {
		load = sync.OnceValue(func() *routeManifest {
			manifest, err := d.fetch(baseURL)
			if err != nil && s.cfg.Debug {
				// An invalid URL or transport error can contain user input.
				log.Print("[fibe-mcp] DEBUG: route manifest unavailable; using the full tool surface")
			}
			return manifest
		})
		d.domains[baseURL] = load
	}
	d.mu.Unlock()
	return load()
}

func (d *editionDiscovery) fetch(baseURL string) (*routeManifest, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("invalid route manifest origin")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/route-manifest.json"
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("route manifest HTTP status %d", res.StatusCode)
	}
	const maxManifestSize = 1 << 20
	data, err := io.ReadAll(io.LimitReader(res.Body, maxManifestSize+1))
	if err != nil {
		return nil, err
	}
	var manifest routeManifest
	if len(data) > maxManifestSize || json.Unmarshal(data, &manifest) != nil ||
		(manifest.Edition != "community" && manifest.Edition != "enterprise") ||
		manifest.Profile == "" || manifest.Major < 1 || manifest.Revision == nil ||
		*manifest.Revision < 0 || len(manifest.Routes) == 0 {
		return nil, errors.New("invalid route manifest")
	}
	manifest.paths = make(map[apiRoute]bool, len(manifest.Routes))
	for _, route := range manifest.Routes {
		if !strings.HasPrefix(route.Path, "/") || strings.ContainsAny(route.Path, "?#") || len(route.Methods) == 0 {
			return nil, errors.New("invalid route manifest path")
		}
		for _, method := range route.Methods {
			if method == "" || strings.ContainsAny(method, " \t\r\n") || method != strings.ToUpper(method) {
				return nil, errors.New("invalid route manifest method")
			}
			manifest.paths[routeKey(method, route.Path)] = true
		}
	}
	return &manifest, nil
}
