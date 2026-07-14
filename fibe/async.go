package fibe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AsyncResult represents an asynchronous operation that was accepted by the API.
// The API returns HTTP 202 with a request_id that can be polled for the final result.
type AsyncResult struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"` // queued, running, success, error
	StatusURL string `json:"status_url,omitempty"`

	// Populated once the operation completes (status == "success"):
	Payload map[string]any `json:"-"`
	// Populated when the operation fails (status == "error"):
	Error        string         `json:"error,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorStatus  int            `json:"error_status,omitempty"`
	ErrorDetails map[string]any `json:"error_details,omitempty"`
}

// IsComplete returns true when the async operation has a terminal status.
func (a *AsyncResult) IsComplete() bool {
	return a.Status == "success" || a.Status == "error" || a.Status == "missing"
}

// IsPending returns true when the operation is still running.
func (a *AsyncResult) IsPending() bool {
	return a.Status == "queued" || a.Status == "running"
}

// AsyncPollOptions configures the behavior of PollAsync.
type AsyncPollOptions struct {
	// Interval between poll attempts. Default: 1s.
	Interval time.Duration
	// Timeout after which polling gives up. Default: 5m.
	Timeout time.Duration
	// Progress receives every async poll status. Defaults to the client progress hook.
	Progress ProgressFunc
}

var defaultPollOpts = AsyncPollOptions{
	Interval: 1 * time.Second,
	Timeout:  5 * time.Minute,
}

// PollAsync polls a status URL until the async operation completes or times out.
// The statusPath must be an absolute API path like "/api/async_requests/uuid".
func (c *Client) PollAsync(ctx context.Context, statusPath string, opts *AsyncPollOptions) (*AsyncResult, error) {
	if strings.TrimSpace(statusPath) == "" {
		return nil, fmt.Errorf("fibe: async status path is empty")
	}

	o := defaultPollOpts
	if opts != nil {
		if opts.Interval > 0 {
			o.Interval = opts.Interval
		}
		if opts.Timeout > 0 {
			o.Timeout = opts.Timeout
		}
		if opts.Progress != nil {
			o.Progress = opts.Progress
		}
	}
	if o.Progress == nil {
		o.Progress = c.cfg.progressHook
	}

	deadline := c.now().Add(o.Timeout)
	attempt := 0

	for {
		if c.now().After(deadline) {
			return nil, fmt.Errorf("fibe: async operation timed out after %s", o.Timeout)
		}

		attempt++
		result, err := c.pollAsyncOnce(ctx, statusPath)
		if err != nil {
			return nil, fmt.Errorf("fibe: poll async status: %w", err)
		}
		c.reportProgress(ctx, o.Progress, ProgressEvent{
			Operation:  "async",
			RequestID:  result.RequestID,
			Status:     result.Status,
			StatusURL:  result.StatusURL,
			StatusPath: statusPath,
			Attempt:    attempt,
		})

		switch result.Status {
		case "success":
			return result, nil
		case "error":
			return result, nil
		case "missing":
			return result, fmt.Errorf("fibe: async request not found")
		case "queued", "running":
			// still in progress — wait and retry
		default:
			// Unknown terminal status — treat as success payload
			result.Status = "success"
			if result.Payload == nil {
				result.Payload = map[string]any{}
			}
			return result, nil
		}

		if err := c.sleep(ctx, o.Interval); err != nil {
			return nil, err
		}
	}
}

func (c *Client) pollAsyncOnce(ctx context.Context, statusPath string) (*AsyncResult, error) {
	resp, err := c.doResponse(ctx, http.MethodGet, statusPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := readLimited(resp.Body, maxResponseBody, "async response body")
	if readErr != nil {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return nil, readErr
	}

	var raw map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, err
		}
	}
	if raw == nil {
		raw = map[string]any{}
	}

	if resp.StatusCode == http.StatusNotFound {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		return &AsyncResult{Status: "missing", Payload: raw}, nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		if c.breaker != nil {
			c.breaker.releaseProbe()
		}
		if result, ok := asyncResultFromRaw(raw); ok {
			if result.Status == "" {
				result.Status = "error"
			}
			return result, nil
		}
		return asyncErrorResultFromAPIError(resp.StatusCode, raw, resp.Header.Get("X-Request-Id")), nil
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		result, ok := asyncResultFromRaw(raw)
		if !ok {
			result = &AsyncResult{Status: "success", Payload: raw}
		}
		return result, nil
	}

	if c.breaker != nil && resp.StatusCode >= 500 {
		c.breaker.recordFailure()
	} else if c.breaker != nil {
		c.breaker.releaseProbe()
	}
	return nil, apiErrorFromRaw(resp.StatusCode, raw, resp.Header.Get("X-Request-Id"))
}

func asyncResultFromRaw(raw map[string]any) (*AsyncResult, bool) {
	status, hasStatus := raw["status"].(string)
	result := &AsyncResult{
		RequestID: stringFromMap(raw, "request_id"),
		Status:    status,
		StatusURL: stringFromMap(raw, "status_url"),
	}
	result.ErrorCode = stringFromMap(raw, "error_code")
	result.ErrorStatus = intFromMap(raw, "error_status")
	result.ErrorDetails = mapFromMap(raw, "error_details")

	if status == "success" {
		result.Payload = raw
		return result, true
	}

	if errText, ok := raw["error"].(string); ok {
		if result.Status == "" {
			result.Status = "error"
		}
		result.Error = errText
		return result, true
	}

	if errObj, ok := raw["error"].(map[string]any); ok {
		if result.Status == "" {
			result.Status = "error"
		}
		result.Error = stringFromMap(errObj, "message")
		if result.ErrorCode == "" {
			result.ErrorCode = stringFromMap(errObj, "code")
		}
		if result.ErrorDetails == nil {
			result.ErrorDetails = mapFromMap(errObj, "details")
		}
		return result, true
	}

	if hasStatus {
		if result.IsPending() {
			return result, true
		}
		result.Payload = raw
		return result, true
	}

	return result, false
}

func asyncErrorResultFromAPIError(statusCode int, raw map[string]any, requestID string) *AsyncResult {
	apiErr := apiErrorFromRaw(statusCode, raw, requestID)
	return &AsyncResult{
		Status:       "error",
		Payload:      raw,
		Error:        apiErr.Message,
		ErrorCode:    apiErr.Code,
		ErrorStatus:  apiErr.StatusCode,
		ErrorDetails: apiErr.Details,
	}
}

func apiErrorFromRaw(statusCode int, raw map[string]any, requestID string) *APIError {
	if errObj, ok := raw["error"].(map[string]any); ok {
		return &APIError{
			StatusCode: statusCode,
			Code:       stringFromMap(errObj, "code"),
			Message:    stringFromMap(errObj, "message"),
			Details:    mapFromMap(errObj, "details"),
			RequestID:  requestID,
		}
	}
	if errText, ok := raw["error"].(string); ok && errText != "" {
		return &APIError{
			StatusCode: statusCode,
			Code:       ErrCodeInternalError,
			Message:    errText,
			RequestID:  requestID,
		}
	}
	return &APIError{
		StatusCode: statusCode,
		Code:       ErrCodeInternalError,
		Message:    fmt.Sprintf("unexpected status %d", statusCode),
		RequestID:  requestID,
	}
}

// doAsync sends a request and if the API returns 202 Accepted,
// automatically polls for the final result. Otherwise behaves like do().
func (c *Client) doAsync(ctx context.Context, method, path, statusPathFmt string, body any, result any) error {
	resp, err := c.doResponse(ctx, method, path, body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusAccepted {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		// Parse the 202 response to get the request_id
		var asyncResp AsyncResult
		decErr := decodeJSONLimited(resp.Body, maxResponseBody, "async accepted response body", &asyncResp)
		drainAndClose(resp.Body)
		if decErr != nil {
			return fmt.Errorf("fibe: decode async response: %w", decErr)
		}

		statusPath, pathErr := c.asyncStatusPath(asyncResp, statusPathFmt)
		if pathErr != nil {
			return pathErr
		}
		c.reportProgress(ctx, c.cfg.progressHook, ProgressEvent{
			Operation:  "async",
			RequestID:  asyncResp.RequestID,
			Status:     asyncResp.Status,
			StatusURL:  asyncResp.StatusURL,
			StatusPath: statusPath,
			Attempt:    0,
		})

		// Poll until completion
		final, pollErr := c.PollAsync(ctx, statusPath, nil)
		if pollErr != nil {
			return pollErr
		}

		if final.Status == "error" {
			statusCode := final.ErrorStatus
			if statusCode == 0 {
				statusCode = http.StatusUnprocessableEntity
			}
			code := final.ErrorCode
			if code == "" {
				code = "REMOTE_REQUEST_FAILED"
			}
			return &APIError{
				StatusCode: statusCode,
				Code:       code,
				Message:    final.Error,
				Details:    final.ErrorDetails,
			}
		}

		// Decode the final payload into the caller's result type
		if result != nil && final.Payload != nil {
			data, err := json.Marshal(final.Payload)
			if err != nil {
				return fmt.Errorf("fibe: marshal async result: %w", err)
			}
			return decodeJSONLimitedProjected(ctx, bytes.NewReader(data), maxResponseBody, "async final response body", result)
		}
		return nil
	}

	// Non-202 — standard response handling
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if c.breaker != nil {
			c.breaker.recordSuccess()
		}
		c.storeRequestID(resp)
		if resp.StatusCode == 204 || result == nil {
			drainAndClose(resp.Body)
			return nil
		}
		err := decodeJSONLimitedProjected(ctx, resp.Body, maxResponseBody, "async response body", result)
		drainAndClose(resp.Body)
		return err
	}

	apiErr := c.parseError(resp)
	drainAndClose(resp.Body)
	if c.breaker != nil && resp.StatusCode >= 500 {
		c.breaker.recordFailure()
	} else if c.breaker != nil {
		c.breaker.releaseProbe()
	}
	return apiErr
}

func (c *Client) reportProgress(ctx context.Context, hook ProgressFunc, event ProgressEvent) {
	if hook == nil {
		return
	}
	hook(ctx, event)
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func mapFromMap(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func intFromMap(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func (c *Client) asyncStatusPath(result AsyncResult, statusPathFmt string) (string, error) {
	if result.StatusURL != "" {
		return normalizeStatusPath(result.StatusURL, c.cfg.baseURL())
	}
	if result.RequestID == "" {
		return "", fmt.Errorf("fibe: async response missing request_id")
	}
	return fmt.Sprintf(statusPathFmt, result.RequestID), nil
}

func normalizeStatusPath(statusURL, baseURL string) (string, error) {
	u, err := url.Parse(statusURL)
	if err != nil {
		return "", fmt.Errorf("fibe: invalid async status_url: %w", err)
	}
	if u.IsAbs() {
		base, err := url.Parse(baseURL)
		if err != nil {
			return "", fmt.Errorf("fibe: invalid base URL for async status: %w", err)
		}
		if !strings.EqualFold(u.Scheme, base.Scheme) || !strings.EqualFold(u.Host, base.Host) {
			return "", fmt.Errorf("fibe: async status_url must use the configured API origin")
		}
	}
	if u.Fragment != "" {
		return "", fmt.Errorf("fibe: async status_url must not include a fragment")
	}
	path := u.EscapedPath()
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	if path == "" {
		return "", fmt.Errorf("fibe: async status_url has no path")
	}
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("fibe: async status_url must be an absolute API path")
	}
	return path, nil
}
