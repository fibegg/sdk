package fibe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	cfg           *clientConfig
	http          *http.Client
	rateLimit     *rateLimitTracker
	breaker       *circuitBreaker
	retry         *retryPolicy
	lastRequestID atomic.Value // stores string

	Playgrounds        *PlaygroundService
	Tricks             *TrickService
	Agents             *AgentService
	Artefacts          *ArtefactService
	Playspecs          *PlayspecService
	Props              *PropService
	Marquees           *MarqueeService
	Secrets            *SecretService
	JobEnv             *JobEnvService
	APIKeys                *APIKeyService
	ImportTemplates        *ImportTemplateService
	ImportTemplateVersions *ImportTemplateVersionService
	WebhookEndpoints       *WebhookEndpointService
	Feedbacks              *FeedbackService
	Mutters            *MutterService
	AuditLogs          *AuditLogService
	Monitor            *MonitorService
	Greenfield         *GreenfieldService
	GitHubRepos        *GitHubRepoService
	GiteaRepos         *GiteaRepoService
	Installations      *InstallationService
	Launch             *LaunchService
	RepoStatus         *RepoStatusService
	TemplateCategories *TemplateCategoryService
	Status             *StatusService
	ServerInfo         *ServerInfoService
}

func NewClient(opts ...Option) *Client {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	if cfg.apiKey == "" {
		if key := os.Getenv("FIBE_API_KEY"); key != "" {
			cfg.apiKey = key
		}
	}

	if cfg.domain == defaultDomain {
		if domain := os.Getenv("FIBE_DOMAIN"); domain != "" {
			cfg.domain = domain
		}
	}

	// Lowest priority: credential store from `fibe auth login`
	if cfg.apiKey == "" {
		store := NewCredentialStore(DefaultCredentialPath())
		if entry, err := store.Get(cfg.domain); err == nil && entry != nil {
			cfg.apiKey = entry.APIKey
		}
	}

	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{Timeout: cfg.timeout}
	}

	if cfg.logger == nil {
		cfg.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
		if cfg.debug {
			cfg.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}
	}

	return newClientFromConfig(cfg)
}

func newClientFromConfig(cfg *clientConfig) *Client {
	c := &Client{
		cfg:       cfg,
		http:      cfg.httpClient,
		rateLimit: &rateLimitTracker{},
		retry: &retryPolicy{
			maxRetries: cfg.maxRetries,
			baseDelay:  cfg.retryBaseDelay,
			maxDelay:   cfg.retryMaxDelay,
		},
	}

	if cfg.breaker != nil {
		c.breaker = newCircuitBreaker(*cfg.breaker)
	}

	c.Playgrounds = &PlaygroundService{client: c}
	c.Tricks = &TrickService{client: c}
	c.Agents = &AgentService{client: c}
	c.Artefacts = &ArtefactService{client: c}
	c.Playspecs = &PlayspecService{client: c}
	c.Props = &PropService{client: c}
	c.Marquees = &MarqueeService{client: c}
	c.Secrets = &SecretService{client: c}
	c.JobEnv = &JobEnvService{client: c}
	c.APIKeys = &APIKeyService{client: c}
	c.ImportTemplates = &ImportTemplateService{client: c}
	c.ImportTemplateVersions = &ImportTemplateVersionService{client: c}
	c.WebhookEndpoints = &WebhookEndpointService{client: c}
	c.Feedbacks = &FeedbackService{client: c}
	c.Mutters = &MutterService{client: c}
	c.AuditLogs = &AuditLogService{client: c}
	c.Monitor = &MonitorService{client: c}
	c.Greenfield = &GreenfieldService{client: c}
	c.GitHubRepos = &GitHubRepoService{client: c}
	c.GiteaRepos = &GiteaRepoService{client: c}
	c.Installations = &InstallationService{client: c}
	c.Launch = &LaunchService{client: c}
	c.RepoStatus = &RepoStatusService{client: c}
	c.TemplateCategories = &TemplateCategoryService{client: c}
	c.Status = &StatusService{client: c}
	c.ServerInfo = &ServerInfoService{client: c}

	return c
}

// WithKey returns a new Client that uses a different API key but shares
// the same base URL, HTTP transport, logger, and all other configuration.
// The new client gets its own rate limit tracker and circuit breaker state.
//
// This is the primary mechanism for multi-key e2e testing:
//
//	admin := fibe.NewClient(fibe.WithAPIKey(adminKey))
//	reader := admin.WithKey(readerKey)
//	other := admin.WithKey(otherPlayerKey)
//
//	admin.Playgrounds.Create(ctx, params)   // creates as admin
//	_, err := other.Playgrounds.Get(ctx, id) // should 404
//	pg, _ := reader.Playgrounds.Get(ctx, id) // should succeed
func (c *Client) WithKey(apiKey string) *Client {
	forked := *c.cfg
	forked.apiKey = apiKey
	return newClientFromConfig(&forked)
}

// Ping provides a fast, cheap way to verify that the CLI/SDK can reach Fibe servers
// and that the provided API key is valid. It returns nil if the connection and
// authentication succeeded.
func (c *Client) Ping(ctx context.Context) error {
	var result struct {
		ID int64 `json:"id"`
	}
	return c.do(ctx, http.MethodGet, "/api/me", nil, &result)
}

// RateLimit returns the current active rate limit state seen from the Fibe API.
// This is automatically updated via X-RateLimit headers on every request.
func (c *Client) RateLimit() RateLimit {
	return c.rateLimit.current()
}

// BaseURL returns the resolved base URL this client targets.
func (c *Client) BaseURL() string {
	return c.cfg.baseURL()
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	if c.breaker != nil && !c.breaker.allow() {
		return &CircuitOpenError{Resource: path}
	}

	if c.cfg.rateLimitWait {
		if wait := c.rateLimit.waitTime(); wait > 0 {
			c.cfg.logger.Debug("rate limit wait", "duration", wait)
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.retry.maxRetries; attempt++ {
		if attempt > 0 {
			c.cfg.logger.Debug("retrying request", "attempt", attempt, "path", path)
		}

		resp, err := c.doOnce(ctx, method, path, body)
		if err != nil {
			lastErr = err
			if attempt < c.retry.maxRetries {
				delay := c.retry.delay(attempt, 0)
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-timer.C:
				}
				continue
			}
			break
		}
		c.rateLimit.update(resp)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if c.breaker != nil {
				c.breaker.recordSuccess()
			}
			c.storeRequestID(resp)
			if resp.StatusCode == 204 || result == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				return nil
			}
			err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(result)
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}
			applyProjection(ctx, result)
			return nil
		}

		apiErr := c.parseError(resp)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if c.retry.shouldRetry(attempt, resp.StatusCode) {
			if c.breaker != nil {
				c.breaker.recordFailure()
			}
			retryAfter := parseRetryAfter(resp)
			delay := c.retry.delay(attempt, retryAfter)
			lastErr = apiErr
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}

		if c.breaker != nil && resp.StatusCode >= 500 {
			c.breaker.recordFailure()
		}
		return apiErr
	}
	return lastErr
}

func (c *Client) doOnce(ctx context.Context, method, path string, body any) (*http.Response, error) {
	u := c.cfg.baseURL() + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("fibe: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("fibe: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key := idempotencyKeyFromCtx(ctx); key != "" {
		req.Header.Set("Idempotency-Key", key)
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		req.Header.Set("Idempotency-Key", NewIdempotencyKey())
	}

	if c.cfg.debug {
		c.cfg.logger.Debug("request", "method", method, "url", u)
	}

	if c.cfg.requestHook != nil {
		if err := c.cfg.requestHook(req); err != nil {
			return nil, fmt.Errorf("fibe: request hook: %w", err)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if c.cfg.responseHook != nil {
		if err := c.cfg.responseHook(resp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("fibe: response hook: %w", err)
		}
	}

	return resp, nil
}

func (c *Client) doMultipart(ctx context.Context, method, path string, fields map[string]string, fileField, fileName string, fileReader io.Reader, result any) error {
	if c.breaker != nil && !c.breaker.allow() {
		return &CircuitOpenError{Resource: path}
	}

	u := c.cfg.baseURL() + path

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return fmt.Errorf("fibe: write field %s: %w", k, err)
		}
	}

	if fileReader != nil {
		part, err := writer.CreateFormFile(fileField, fileName)
		if err != nil {
			return fmt.Errorf("fibe: create form file: %w", err)
		}
		if _, err := io.Copy(part, fileReader); err != nil {
			return fmt.Errorf("fibe: copy file: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("fibe: close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, &buf)
	if err != nil {
		return fmt.Errorf("fibe: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	if key := idempotencyKeyFromCtx(ctx); key != "" {
		req.Header.Set("Idempotency-Key", key)
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete {
		req.Header.Set("Idempotency-Key", NewIdempotencyKey())
	}

	if c.cfg.requestHook != nil {
		if err := c.cfg.requestHook(req); err != nil {
			return fmt.Errorf("fibe: request hook: %w", err)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fibe: execute request: %w", err)
	}
	defer resp.Body.Close()

	if c.cfg.responseHook != nil {
		if err := c.cfg.responseHook(resp); err != nil {
			return fmt.Errorf("fibe: response hook: %w", err)
		}
	}

	c.rateLimit.update(resp)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		c.storeRequestID(resp)
		if resp.StatusCode == 204 || result == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(result); err != nil {
			return fmt.Errorf("fibe: decode multipart response: %w", err)
		}
		return nil
	}

	if c.breaker != nil && resp.StatusCode >= 500 {
		c.breaker.recordFailure()
	}
	return c.parseError(resp)
}

func (c *Client) doStream(ctx context.Context, method, path string, body any) (io.ReadCloser, error) {
	if c.breaker != nil && !c.breaker.allow() {
		return nil, &CircuitOpenError{Resource: path}
	}

	resp, err := c.doOnce(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	c.rateLimit.update(resp)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		return resp.Body, nil
	}

	defer resp.Body.Close()
	if c.breaker != nil && resp.StatusCode >= 500 {
		c.breaker.recordFailure()
	}
	return nil, c.parseError(resp)
}

func (c *Client) doDownload(ctx context.Context, path string) (io.ReadCloser, string, string, error) {
	if c.breaker != nil && !c.breaker.allow() {
		return nil, "", "", &CircuitOpenError{Resource: path}
	}

	noRedirectClient := &http.Client{
		Timeout:   c.cfg.timeout,
		Transport: c.http.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	u := c.cfg.baseURL() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Accept", "application/octet-stream, */*")

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, "", "", err
	}

	c.rateLimit.update(resp)

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		filename := filenameFromContentDisposition(resp.Header.Get("Content-Disposition"))
		resp.Body.Close()
		if loc == "" {
			return nil, "", "", &APIError{StatusCode: resp.StatusCode, Code: ErrCodeInternalError, Message: "redirect without Location"}
		}
		redirReq, err := http.NewRequestWithContext(ctx, http.MethodGet, loc, nil)
		if err != nil {
			return nil, "", "", err
		}
		redirResp, err := http.DefaultClient.Do(redirReq)
		if err != nil {
			return nil, "", "", err
		}
		if redirResp.StatusCode >= 200 && redirResp.StatusCode < 300 {
			if filename == "" {
				filename = filenameFromContentDisposition(redirResp.Header.Get("Content-Disposition"))
			}
			if filename == "" {
				filename = filenameFromURL(loc)
			}
			return redirResp.Body, filename, redirResp.Header.Get("Content-Type"), nil
		}
		defer redirResp.Body.Close()
		return nil, "", "", &APIError{StatusCode: redirResp.StatusCode, Code: ErrCodeInternalError, Message: fmt.Sprintf("download from redirect failed: %d", redirResp.StatusCode)}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		filename := filenameFromContentDisposition(resp.Header.Get("Content-Disposition"))
		return resp.Body, filename, resp.Header.Get("Content-Type"), nil
	}

	defer resp.Body.Close()
	return nil, "", "", c.parseError(resp)
}

func filenameFromContentDisposition(header string) string {
	if header == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(header)
	if err != nil {
		return ""
	}
	if v, ok := params["filename*"]; ok && v != "" {
		if i := strings.Index(v, "''"); i != -1 {
			if decoded, err := url.QueryUnescape(v[i+2:]); err == nil {
				return decoded
			}
			return v[i+2:]
		}
		return v
	}
	if v, ok := params["filename"]; ok {
		return v
	}
	return ""
}

func filenameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	base := path.Base(u.Path)
	if base == "/" || base == "." {
		return ""
	}
	return base
}

func (c *Client) parseError(resp *http.Response) *APIError {
	var errResp apiErrorResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1*1024*1024)).Decode(&errResp); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       ErrCodeInternalError,
			Message:    fmt.Sprintf("unexpected status %d", resp.StatusCode),
			RequestID:  resp.Header.Get("X-Request-Id"),
		}
	}

	apiErr := &APIError{
		StatusCode:         resp.StatusCode,
		Code:               errResp.Error.Code,
		Message:            errResp.Error.Message,
		Details:            errResp.Error.Details,
		RequestID:          resp.Header.Get("X-Request-Id"),
		IdempotentReplayed: resp.Header.Get("X-Idempotent-Replayed") == "true",
	}

	if resp.StatusCode == 429 {
		apiErr.Code = ErrCodeRateLimited
		apiErr.RetryAfter = parseRetryAfter(resp)
	}

	c.storeRequestID(resp)
	return apiErr
}

func (c *Client) storeRequestID(resp *http.Response) {
	if id := resp.Header.Get("X-Request-Id"); id != "" {
		c.lastRequestID.Store(id)
	}
}

// LastRequestID returns the X-Request-Id from the most recent API response.
// Useful for support tickets and debugging.
func (c *Client) LastRequestID() string {
	v, _ := c.lastRequestID.Load().(string)
	return v
}

func applyProjection(ctx context.Context, result any) {
	fields := fieldsFromCtx(ctx)
	if len(fields) == 0 {
		return
	}

	data, err := json.Marshal(result)
	if err != nil {
		return
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	if dataSlice, ok := raw["data"].([]any); ok {
		for i, item := range dataSlice {
			if m, ok := item.(map[string]any); ok {
				dataSlice[i] = filterMap(m, fields)
			}
		}
		raw["data"] = dataSlice
	} else {
		filterMap(raw, fields)
	}

	filtered, err := json.Marshal(raw)
	if err != nil {
		return
	}

	rv := reflect.ValueOf(result)
	if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct {
		rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
	}
	json.Unmarshal(filtered, result)
}

func filterMap(m map[string]any, fields map[string]bool) map[string]any {
	for key := range m {
		if !fields[key] {
			delete(m, key)
		}
	}
	return m
}

func buildQuery(params any) string {
	if params == nil {
		return ""
	}
	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	t := v.Type()

	q := url.Values{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		omitEmpty := len(parts) > 1 && parts[1] == "omitempty"

		fv := v.Field(i)
		var str string
		switch fv.Kind() {
		case reflect.String:
			str = fv.String()
		case reflect.Int, reflect.Int64:
			if fv.Int() != 0 {
				str = fmt.Sprintf("%d", fv.Int())
			}
		case reflect.Ptr:
			if !fv.IsNil() && fv.Elem().Kind() == reflect.Bool {
				str = fmt.Sprintf("%t", fv.Elem().Bool())
			}
		}
		if omitEmpty && str == "" {
			continue
		}
		if str != "" {
			q.Set(name, str)
		}
	}

	encoded := q.Encode()
	if encoded == "" {
		return ""
	}
	return "?" + encoded
}
