package fibe

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultDomain     = "fibe.gg"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

var (
	sdkVersion       = "devel"
	defaultUserAgent = "fibe-go/" + sdkVersion
)

type Option func(*clientConfig)

// ProgressEvent describes a long-running SDK operation tick. It is intentionally
// transport-neutral: CLIs can render it, MCP servers can forward it as progress
// notifications, and library callers can ignore it.
type ProgressEvent struct {
	Operation  string
	RequestID  string
	Status     string
	StatusURL  string
	StatusPath string
	Attempt    int
}

// ProgressFunc receives progress events from long-running SDK operations.
type ProgressFunc func(context.Context, ProgressEvent)

type clientConfig struct {
	domain            string
	apiKey            string
	httpClient        *http.Client
	timeout           time.Duration
	timeoutSet        bool
	userAgent         string
	maxRetries        int
	retryBaseDelay    time.Duration
	retryMaxDelay     time.Duration
	breaker           *CircuitBreakerConfig
	logger            *slog.Logger
	debug             bool
	rateLimitWait     bool
	requestHook       func(req *http.Request) error
	responseHook      func(res *http.Response) error
	progressHook      ProgressFunc
	disableAutoConfig bool
}

func defaultConfig() *clientConfig {
	return &clientConfig{
		domain:         defaultDomain,
		timeout:        defaultTimeout,
		userAgent:      defaultUserAgent,
		maxRetries:     defaultMaxRetries,
		retryBaseDelay: 500 * time.Millisecond,
		retryMaxDelay:  30 * time.Second,
	}
}

func (c *clientConfig) baseURL() string {
	baseURL, err := resolveBaseURL(c.domain)
	if err == nil {
		return baseURL
	}
	// BaseURL cannot report an error without changing its public signature.
	// Request construction uses resolveBaseURL directly and returns the error.
	return strings.TrimRight(c.domain, "/")
}

func isLocalDomain(d string) bool {
	u, err := url.Parse("//" + strings.TrimSpace(d))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	ip := net.ParseIP(host)
	return host == "localhost" ||
		(ip != nil && ip.IsLoopback()) ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".test") ||
		strings.HasSuffix(host, ".internal")
}

func resolveBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("fibe: base URL is empty")
	}

	candidate := raw
	if !strings.Contains(candidate, "://") {
		scheme := "https://"
		if isLocalDomain(candidate) {
			scheme = "http://"
		}
		candidate = scheme + candidate
	}

	u, err := url.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("fibe: invalid base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("fibe: base URL must use http or https")
	}
	if u.Host == "" || u.Hostname() == "" {
		return "", fmt.Errorf("fibe: base URL must include a host")
	}
	if u.User != nil {
		return "", fmt.Errorf("fibe: base URL must not include credentials")
	}
	if u.Fragment != "" {
		return "", fmt.Errorf("fibe: base URL must not include a fragment")
	}
	if u.RawQuery != "" {
		return "", fmt.Errorf("fibe: base URL must not include a query")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return strings.TrimRight(u.String(), "/"), nil
}

func WithAPIKey(key string) Option {
	return func(c *clientConfig) { c.apiKey = key }
}

func WithDomain(domain string) Option {
	return func(c *clientConfig) { c.domain = domain }
}

// WithBaseURL sets the full base URL (scheme + host + optional port).
// Alias for WithDomain — accepts both "fibe.gg" and "http://localhost:3000".
func WithBaseURL(url string) Option {
	return func(c *clientConfig) { c.domain = url }
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = client }
}

func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = d
		c.timeoutSet = true
	}
}

func WithUserAgent(ua string) Option {
	return func(c *clientConfig) { c.userAgent = ua }
}

func WithMaxRetries(n int) Option {
	return func(c *clientConfig) { c.maxRetries = n }
}

func WithRetryDelay(base, max time.Duration) Option {
	return func(c *clientConfig) {
		c.retryBaseDelay = base
		c.retryMaxDelay = max
	}
}

func WithCircuitBreaker(cfg CircuitBreakerConfig) Option {
	return func(c *clientConfig) { c.breaker = &cfg }
}

func WithLogger(l *slog.Logger) Option {
	return func(c *clientConfig) { c.logger = l }
}

func WithDebug() Option {
	return func(c *clientConfig) { c.debug = true }
}

func WithRateLimitAutoWait() Option {
	return func(c *clientConfig) { c.rateLimitWait = true }
}

// WithRequestHook adds a callback that executes immediately before the HTTP request is sent
func WithRequestHook(hook func(req *http.Request) error) Option {
	return func(c *clientConfig) { c.requestHook = hook }
}

// WithResponseHook adds a callback that executes immediately after the HTTP response is received,
// before the body is read or parsed.
func WithResponseHook(hook func(res *http.Response) error) Option {
	return func(c *clientConfig) { c.responseHook = hook }
}

// WithProgress installs a callback for long-running SDK operations such as
// async request polling and rollout waits.
func WithProgress(hook ProgressFunc) Option {
	return func(c *clientConfig) { c.progressHook = hook }
}

// WithDisableAutoConfig prevents NewClient from reading FIBE_API_KEY,
// FIBE_DOMAIN, or the CLI credential store. CLI commands use this after
// resolving profiles explicitly; SDK callers keep the existing behavior by
// default.
func WithDisableAutoConfig() Option {
	return func(c *clientConfig) { c.disableAutoConfig = true }
}
