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
	"net/textproto"
	"net/url"
	"os"
	"path"
	"reflect"
	"sort"
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
	now           func() time.Time
	sleep         func(context.Context, time.Duration) error
	lastRequestID atomic.Value // stores string

	Playgrounds            *PlaygroundService
	Tricks                 *TrickService
	Agents                 *AgentService
	Cable                  *CableService
	AgentDefaults          *AgentDefaultsService
	Artefacts              *ArtefactService
	Playspecs              *PlayspecService
	Props                  *PropService
	Marquees               *MarqueeService
	Secrets                *SecretService
	JobEnv                 *JobEnvService
	APIKeys                *APIKeyService
	ImportTemplates        *ImportTemplateService
	ImportTemplateVersions *ImportTemplateVersionService
	WebhookEndpoints       *WebhookEndpointService
	Feedbacks              *FeedbackService
	Mutters                *MutterService
	Memories               *MemoryService
	AuditLogs              *AuditLogService
	Monitor                *MonitorService
	Greenfield             *GreenfieldService
	GitHubApps             *GitHubAppService
	GitHubRepos            *GitHubRepoService
	GiteaRepos             *GiteaRepoService
	Installations          *InstallationService
	Launch                 *LaunchService
	RepoStatus             *RepoStatusService
	TemplateCategories     *TemplateCategoryService
	Status                 *StatusService
	ServerInfo             *ServerInfoService
}

func NewClient(opts ...Option) *Client {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	if !cfg.disableAutoConfig && cfg.apiKey == "" {
		if key := os.Getenv("FIBE_API_KEY"); key != "" {
			cfg.apiKey = key
		}
	}

	if !cfg.disableAutoConfig && cfg.domain == defaultDomain {
		if domain := os.Getenv("FIBE_DOMAIN"); domain != "" {
			cfg.domain = domain
		}
	}

	// Lowest priority: credential store from `fibe auth login`
	if !cfg.disableAutoConfig && cfg.apiKey == "" {
		store := NewCredentialStore(DefaultCredentialPath())
		if entry, err := store.Get(cfg.domain); err == nil && entry != nil {
			cfg.apiKey = entry.APIKey
		}
	}

	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{Timeout: cfg.timeout}
	} else {
		client := *cfg.httpClient
		if cfg.timeoutSet {
			client.Timeout = cfg.timeout
		}
		cfg.httpClient = &client
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
	if cfg.maxRetries < 0 {
		cfg.maxRetries = 0
	}
	if cfg.retryBaseDelay < 0 {
		cfg.retryBaseDelay = 0
	}
	if cfg.retryMaxDelay < 0 {
		cfg.retryMaxDelay = 0
	}
	if cfg.retryMaxDelay > 0 && cfg.retryMaxDelay < cfg.retryBaseDelay {
		cfg.retryMaxDelay = cfg.retryBaseDelay
	}
	c := &Client{
		cfg:       cfg,
		http:      cfg.httpClient,
		rateLimit: &rateLimitTracker{},
		retry: &retryPolicy{
			maxRetries: cfg.maxRetries,
			baseDelay:  cfg.retryBaseDelay,
			maxDelay:   cfg.retryMaxDelay,
		},
		now:   time.Now,
		sleep: waitForRetry,
	}

	if cfg.breaker != nil {
		c.breaker = newCircuitBreaker(*cfg.breaker)
	}

	c.Playgrounds = &PlaygroundService{client: c}
	c.Tricks = &TrickService{client: c}
	c.Agents = &AgentService{client: c}
	c.Cable = &CableService{client: c}
	c.AgentDefaults = &AgentDefaultsService{client: c}
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
	c.Memories = &MemoryService{client: c}
	c.AuditLogs = &AuditLogService{client: c}
	c.Monitor = &MonitorService{client: c}
	c.Greenfield = &GreenfieldService{client: c}
	c.GitHubApps = &GitHubAppService{client: c}
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
	if isMutationMethod(method) && idempotencyKeyFromCtx(ctx) == "" {
		ctx = WithIdempotencyKey(ctx, NewIdempotencyKey())
	}

	if err := c.waitForRateLimit(ctx); err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= c.retry.maxRetries; attempt++ {
		if attempt > 0 {
			c.cfg.logger.Debug("retrying request", "attempt", attempt, "path", path)
		}

		resp, err := c.doOnce(ctx, method, path, body)
		if err != nil {
			lastErr = err
			if c.breaker != nil {
				if c.retry.isTransientError(err) {
					c.breaker.recordFailure()
				} else {
					c.breaker.releaseProbe()
				}
			}
			if c.retry.shouldRetryError(attempt, err) {
				delay := c.retry.delay(attempt, 0)
				if err := c.sleep(ctx, delay); err != nil {
					return err
				}
				continue
			}
			break
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if c.breaker != nil {
				c.breaker.recordSuccess()
			}
			c.storeRequestID(resp)
			if resp.StatusCode == 204 || result == nil {
				drainAndClose(resp.Body)
				return nil
			}
			err := decodeJSONLimitedProjected(ctx, resp.Body, maxResponseBody, "response body", result)
			drainAndClose(resp.Body)
			if err != nil {
				return err
			}
			return nil
		}

		apiErr := c.parseError(resp)
		drainAndClose(resp.Body)

		if c.retry.shouldRetry(attempt, resp.StatusCode) {
			if c.breaker != nil && resp.StatusCode >= 500 {
				c.breaker.recordFailure()
			} else if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			retryAfter := parseRetryAfterAt(resp, c.now())
			delay := c.retry.delay(attempt, retryAfter)
			lastErr = apiErr
			if err := c.sleep(ctx, delay); err != nil {
				return err
			}
			continue
		}

		if c.breaker != nil && resp.StatusCode >= 500 {
			c.breaker.recordFailure()
		} else if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return apiErr
	}
	return lastErr
}

// doResponse executes a request through the shared reliability policy while
// leaving the final response body to the caller. It is used by protocols such
// as async polling that must interpret multiple non-error HTTP statuses.
func (c *Client) doResponse(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if c.breaker != nil && !c.breaker.allow() {
		return nil, &CircuitOpenError{Resource: path}
	}
	if isMutationMethod(method) && idempotencyKeyFromCtx(ctx) == "" {
		ctx = WithIdempotencyKey(ctx, NewIdempotencyKey())
	}
	if err := c.waitForRateLimit(ctx); err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return nil, err
	}

	for attempt := 0; attempt <= c.retry.maxRetries; attempt++ {
		resp, err := c.doOnce(ctx, method, path, body)
		if err != nil {
			if c.retry.shouldRetryError(attempt, err) {
				if c.breaker != nil {
					c.breaker.recordFailure()
				}
				if err := c.sleep(ctx, c.retry.delay(attempt, 0)); err != nil {
					return nil, err
				}
				continue
			}
			if c.breaker != nil {
				if c.retry.isTransientError(err) {
					c.breaker.recordFailure()
				} else {
					c.breaker.releaseProbe()
				}
			}
			return nil, err
		}
		if c.retry.shouldRetry(attempt, resp.StatusCode) {
			retryAfter := parseRetryAfterAt(resp, c.now())
			drainAndClose(resp.Body)
			if c.breaker != nil && resp.StatusCode >= 500 {
				c.breaker.recordFailure()
			}
			if err := c.sleep(ctx, c.retry.delay(attempt, retryAfter)); err != nil {
				return nil, err
			}
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("fibe: request failed without a response")
}

func (c *Client) doOnce(ctx context.Context, method, path string, body any) (*http.Response, error) {
	baseURL, err := resolveBaseURL(c.cfg.domain)
	if err != nil {
		return nil, &permanentRequestError{err: err}
	}
	u := baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, &permanentRequestError{err: fmt.Errorf("fibe: marshal request body: %w", err)}
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, &permanentRequestError{err: fmt.Errorf("fibe: create request: %w", err)}
	}

	if c.cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
	}
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key := idempotencyKeyFromCtx(ctx); key != "" {
		req.Header.Set("Idempotency-Key", key)
	} else if isMutationMethod(method) {
		req.Header.Set("Idempotency-Key", NewIdempotencyKey())
	}

	if c.cfg.debug {
		c.cfg.logger.Debug("request", "method", method, "url", u)
	}

	if c.cfg.requestHook != nil {
		if err := c.cfg.requestHook(req); err != nil {
			return nil, &permanentRequestError{err: fmt.Errorf("fibe: request hook: %w", err)}
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	c.rateLimit.update(resp)
	c.storeRequestID(resp)

	if c.cfg.responseHook != nil {
		if err := c.cfg.responseHook(resp); err != nil {
			drainAndClose(resp.Body)
			return nil, &permanentRequestError{err: fmt.Errorf("fibe: response hook: %w", err)}
		}
	}

	return resp, nil
}

func isMutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (c *Client) doMultipart(ctx context.Context, method, path string, fields map[string]string, fileField, fileName string, fileReader io.Reader, result any) error {
	if c.breaker != nil && !c.breaker.allow() {
		return &CircuitOpenError{Resource: path}
	}
	baseURL, err := resolveBaseURL(c.cfg.domain)
	if err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return err
	}
	if isMutationMethod(method) && idempotencyKeyFromCtx(ctx) == "" {
		ctx = WithIdempotencyKey(ctx, NewIdempotencyKey())
	}
	if err := c.waitForRateLimit(ctx); err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return err
	}

	seeker, replayable := fileReader.(io.ReadSeeker)
	var startOffset int64
	if replayable {
		startOffset, err = seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			replayable = false
		}
	}
	if fileReader == nil {
		replayable = true
	}

	var lastErr error
	for attempt := 0; attempt <= c.retry.maxRetries; attempt++ {
		if attempt > 0 {
			if !replayable {
				break
			}
			if seeker != nil {
				if _, err := seeker.Seek(startOffset, io.SeekStart); err != nil {
					return fmt.Errorf("fibe: rewind multipart file: %w", err)
				}
			}
		}

		body, contentType := streamMultipartBody(fields, fileField, fileName, fileReader)
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
		if err != nil {
			_ = body.Close()
			return fmt.Errorf("fibe: create request: %w", err)
		}
		if c.cfg.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
		}
		req.Header.Set("User-Agent", c.cfg.userAgent)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKeyFromCtx(ctx))

		if c.cfg.requestHook != nil {
			if err := c.cfg.requestHook(req); err != nil {
				_ = body.Close()
				if c.breaker != nil {
					c.breaker.releaseProbe()
				}
				return fmt.Errorf("fibe: request hook: %w", err)
			}
		}

		resp, err := c.http.Do(req)
		if err != nil {
			_ = body.Close()
			lastErr = fmt.Errorf("fibe: execute request: %w", err)
			if c.breaker != nil {
				if c.retry.isTransientError(err) {
					c.breaker.recordFailure()
				} else {
					c.breaker.releaseProbe()
				}
			}
			if !replayable || !c.retry.shouldRetryError(attempt, err) {
				return lastErr
			}
			if err := c.sleep(ctx, c.retry.delay(attempt, 0)); err != nil {
				return err
			}
			continue
		}
		c.rateLimit.update(resp)
		c.storeRequestID(resp)
		if c.cfg.responseHook != nil {
			if err := c.cfg.responseHook(resp); err != nil {
				drainAndClose(resp.Body)
				if c.breaker != nil {
					c.breaker.releaseProbe()
				}
				return fmt.Errorf("fibe: response hook: %w", err)
			}
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if c.breaker != nil {
				c.breaker.recordSuccess()
			}
			if resp.StatusCode == http.StatusNoContent || result == nil {
				drainAndClose(resp.Body)
				return nil
			}
			err := decodeJSONLimitedProjected(ctx, resp.Body, maxResponseBody, "multipart response body", result)
			drainAndClose(resp.Body)
			if err != nil {
				return fmt.Errorf("fibe: decode multipart response: %w", err)
			}
			return nil
		}

		apiErr := c.parseError(resp)
		drainAndClose(resp.Body)
		lastErr = apiErr
		if replayable && c.retry.shouldRetry(attempt, resp.StatusCode) {
			if c.breaker != nil && resp.StatusCode >= 500 {
				c.breaker.recordFailure()
			}
			if err := c.sleep(ctx, c.retry.delay(attempt, parseRetryAfterAt(resp, c.now()))); err != nil {
				return err
			}
			continue
		}
		if c.breaker != nil && resp.StatusCode >= 500 {
			c.breaker.recordFailure()
		} else if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return apiErr
	}
	return lastErr
}

func streamMultipartBody(fields map[string]string, fileField, fileName string, fileReader io.Reader) (io.ReadCloser, string) {
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	contentType := multipartWriter.FormDataContentType()
	go func() {
		var writeErr error
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if writeErr = multipartWriter.WriteField(key, fields[key]); writeErr != nil {
				break
			}
		}
		if writeErr == nil && fileReader != nil {
			var part io.Writer
			part, writeErr = createMultipartFilePart(multipartWriter, fileField, fileName)
			if writeErr == nil {
				_, writeErr = io.Copy(part, fileReader)
			}
		}
		if closeErr := multipartWriter.Close(); writeErr == nil {
			writeErr = closeErr
		}
		_ = writer.CloseWithError(writeErr)
	}()
	return reader, contentType
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) waitForRateLimit(ctx context.Context) error {
	if !c.cfg.rateLimitWait {
		return nil
	}
	wait := c.rateLimit.waitTimeAt(c.now())
	if wait <= 0 {
		return nil
	}
	c.cfg.logger.Debug("rate limit wait", "duration", wait)
	return c.sleep(ctx, wait)
}

func createMultipartFilePart(writer *multipart.Writer, fileField, fileName string) (io.Writer, error) {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, multipartQuote(fileField), multipartQuote(fileName)))
	contentType := mime.TypeByExtension(path.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	return writer.CreatePart(header)
}

func multipartQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func (c *Client) doDownload(ctx context.Context, path string) (io.ReadCloser, string, string, error) {
	if c.breaker != nil && !c.breaker.allow() {
		return nil, "", "", &CircuitOpenError{Resource: path}
	}
	if err := c.waitForRateLimit(ctx); err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return nil, "", "", err
	}

	baseURL, err := resolveBaseURL(c.cfg.domain)
	if err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return nil, "", "", err
	}
	noRedirectClient := *c.http
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	u := baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return nil, "", "", err
	}
	if c.cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
	}
	req.Header.Set("User-Agent", c.cfg.userAgent)
	req.Header.Set("Accept", "application/octet-stream, */*")
	if c.cfg.requestHook != nil {
		if err := c.cfg.requestHook(req); err != nil {
			if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			return nil, "", "", fmt.Errorf("fibe: request hook: %w", err)
		}
	}

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		if c.breaker != nil {
			if c.retry.isTransientError(err) {
				c.breaker.recordFailure()
			} else {
				c.breaker.releaseProbe()
			}
		}
		return nil, "", "", err
	}
	c.rateLimit.update(resp)
	c.storeRequestID(resp)
	if c.cfg.responseHook != nil {
		if err := c.cfg.responseHook(resp); err != nil {
			drainAndClose(resp.Body)
			if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			return nil, "", "", fmt.Errorf("fibe: response hook: %w", err)
		}
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		filename := filenameFromContentDisposition(resp.Header.Get("Content-Disposition"))
		drainAndClose(resp.Body)
		if loc == "" {
			if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			return nil, "", "", &APIError{StatusCode: resp.StatusCode, Code: ErrCodeInternalError, Message: "redirect without Location"}
		}
		redirectURL, err := url.Parse(loc)
		if err != nil || !redirectURL.IsAbs() || (redirectURL.Scheme != "http" && redirectURL.Scheme != "https") || redirectURL.User != nil {
			if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			return nil, "", "", fmt.Errorf("fibe: invalid download redirect URL")
		}
		redirReq, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL.String(), nil)
		if err != nil {
			if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			return nil, "", "", err
		}
		apiURL, _ := url.Parse(baseURL)
		if sameHTTPOrigin(apiURL, redirectURL) && c.cfg.apiKey != "" {
			redirReq.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
		}
		redirReq.Header.Set("User-Agent", c.cfg.userAgent)
		redirReq.Header.Set("Accept", "application/octet-stream, */*")
		redirectClient := *c.http
		callerCheckRedirect := redirectClient.CheckRedirect
		redirectClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
			if len(via) > 3 {
				return fmt.Errorf("fibe: download redirect limit exceeded")
			}
			if next.URL.Scheme != "http" && next.URL.Scheme != "https" {
				return fmt.Errorf("fibe: download redirect must use http or https")
			}
			if sameHTTPOrigin(apiURL, next.URL) && c.cfg.apiKey != "" {
				// #nosec G119 -- credentials are restored only for the normalized configured API origin; cross-origin redirects delete them below.
				next.Header.Set("Authorization", "Bearer "+c.cfg.apiKey)
			} else {
				next.Header.Del("Authorization")
			}
			if callerCheckRedirect != nil {
				return callerCheckRedirect(next, via)
			}
			return nil
		}
		redirResp, err := redirectClient.Do(redirReq)
		if err != nil {
			if c.breaker != nil {
				c.breaker.releaseProbe()
			}
			return nil, "", "", err
		}
		if redirResp.StatusCode >= 200 && redirResp.StatusCode < 300 {
			if c.breaker != nil {
				c.breaker.recordSuccess()
			}
			if filename == "" {
				filename = filenameFromContentDisposition(redirResp.Header.Get("Content-Disposition"))
			}
			if filename == "" {
				filename = filenameFromURL(loc)
			}
			return redirResp.Body, filename, redirResp.Header.Get("Content-Type"), nil
		}
		drainAndClose(redirResp.Body)
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return nil, "", "", &APIError{StatusCode: redirResp.StatusCode, Code: ErrCodeInternalError, Message: fmt.Sprintf("download from redirect failed: %d", redirResp.StatusCode)}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		filename := filenameFromContentDisposition(resp.Header.Get("Content-Disposition"))
		return resp.Body, filename, resp.Header.Get("Content-Type"), nil
	}

	apiErr := c.parseError(resp)
	drainAndClose(resp.Body)
	if c.breaker != nil && resp.StatusCode >= 500 {
		c.breaker.recordFailure()
	} else if c.breaker != nil {
		c.breaker.releaseProbe()
	}
	return nil, "", "", apiErr
}

func sameHTTPOrigin(a, b *url.URL) bool {
	if a == nil || b == nil || !strings.EqualFold(a.Scheme, b.Scheme) || !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	port := func(u *url.URL) string {
		if value := u.Port(); value != "" {
			return value
		}
		if strings.EqualFold(u.Scheme, "https") {
			return "443"
		}
		return "80"
	}
	return port(a) == port(b)
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

func (c *Client) parseError(resp *http.Response) error {
	var errResp apiErrorResponse
	if err := decodeJSONLimited(resp.Body, maxErrorBody, "error response body", &errResp); err != nil {
		if strings.Contains(err.Error(), "exceeds") {
			return err
		}
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
		apiErr.RetryAfter = parseRetryAfterAt(resp, c.now())
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

func applyProjection(ctx context.Context, result any) error {
	fields := fieldsFromCtx(ctx)
	if len(fields) == 0 {
		return nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("fibe: marshal projected response: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("fibe: decode projected response: %w", err)
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
		return fmt.Errorf("fibe: marshal projected fields: %w", err)
	}

	rv := reflect.ValueOf(result)
	if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct {
		rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
	}
	if err := json.Unmarshal(filtered, result); err != nil {
		return fmt.Errorf("fibe: apply projected fields: %w", err)
	}
	return nil
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
