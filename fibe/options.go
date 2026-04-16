package fibe

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	defaultDomain    = "fibe.gg"
	defaultTimeout   = 30 * time.Second
	defaultMaxRetries = 3
	defaultUserAgent = "fibe-go/0.1.0"
)

type Option func(*clientConfig)

type clientConfig struct {
	domain         string
	apiKey         string
	httpClient     *http.Client
	timeout        time.Duration
	userAgent      string
	maxRetries     int
	retryBaseDelay time.Duration
	retryMaxDelay  time.Duration
	breaker        *CircuitBreakerConfig
	logger         *slog.Logger
	debug          bool
	rateLimitWait  bool
	requestHook    func(req *http.Request) error
	responseHook   func(res *http.Response) error
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
	d := c.domain
	if strings.HasPrefix(d, "http://") || strings.HasPrefix(d, "https://") {
		return strings.TrimRight(d, "/")
	}
	if isLocalDomain(d) {
		return "http://" + d
	}
	return "https://" + d
}

func isLocalDomain(d string) bool {
	host := d
	if i := strings.IndexByte(host, ':'); i != -1 {
		host = host[:i]
	}
	return host == "localhost" ||
		strings.HasPrefix(host, "127.") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".test") ||
		strings.HasSuffix(host, ".internal")
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
	return func(c *clientConfig) { c.timeout = d }
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
