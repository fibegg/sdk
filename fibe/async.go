package fibe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Error string `json:"error,omitempty"`
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
}

var defaultPollOpts = AsyncPollOptions{
	Interval: 1 * time.Second,
	Timeout:  5 * time.Minute,
}

// PollAsync polls a status URL until the async operation completes or times out.
// The statusPath must be an absolute API path like "/api/playgrounds/123/action/uuid".
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
	}

	deadline := time.Now().Add(o.Timeout)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("fibe: async operation timed out after %s", o.Timeout)
		}

		result, err := c.pollAsyncOnce(ctx, statusPath)
		if err != nil {
			return nil, fmt.Errorf("fibe: poll async status: %w", err)
		}

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

		timer := time.NewTimer(o.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) pollAsyncOnce(ctx context.Context, statusPath string) (*AsyncResult, error) {
	resp, err := c.doOnce(ctx, http.MethodGet, statusPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.rateLimit.update(resp)
	c.storeRequestID(resp)

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if readErr != nil {
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
		return &AsyncResult{Status: "missing", Payload: raw}, nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		if result, ok := asyncResultFromRaw(raw); ok {
			if result.Status == "" {
				result.Status = "error"
			}
			return result, nil
		}
		return asyncErrorResultFromAPIError(resp.StatusCode, raw, resp.Header.Get("X-Request-Id")), nil
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result, ok := asyncResultFromRaw(raw)
		if !ok {
			result = &AsyncResult{Status: "success", Payload: raw}
		}
		return result, nil
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
		Status:  "error",
		Payload: raw,
		Error:   apiErr.Message,
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
	resp, err := c.doOnce(ctx, method, path, body)
	if err != nil {
		return err
	}
	c.rateLimit.update(resp)

	if resp.StatusCode == http.StatusAccepted {
		// Parse the 202 response to get the request_id
		var asyncResp AsyncResult
		decErr := json.NewDecoder(resp.Body).Decode(&asyncResp)
		resp.Body.Close()
		if decErr != nil {
			return fmt.Errorf("fibe: decode async response: %w", decErr)
		}

		statusPath, pathErr := asyncStatusPath(asyncResp, statusPathFmt)
		if pathErr != nil {
			return pathErr
		}

		// Poll until completion
		final, pollErr := c.PollAsync(ctx, statusPath, nil)
		if pollErr != nil {
			return pollErr
		}

		if final.Status == "error" {
			return &APIError{
				StatusCode: 422,
				Code:       "REMOTE_REQUEST_FAILED",
				Message:    final.Error,
			}
		}

		// Decode the final payload into the caller's result type
		if result != nil && final.Payload != nil {
			data, _ := json.Marshal(final.Payload)
			return json.Unmarshal(data, result)
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
			resp.Body.Close()
			return nil
		}
		err := json.NewDecoder(resp.Body).Decode(result)
		resp.Body.Close()
		return err
	}

	apiErr := c.parseError(resp)
	resp.Body.Close()
	if c.breaker != nil && resp.StatusCode >= 500 {
		c.breaker.recordFailure()
	}
	return apiErr
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func mapFromMap(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func asyncStatusPath(result AsyncResult, statusPathFmt string) (string, error) {
	if result.StatusURL != "" {
		return normalizeStatusPath(result.StatusURL)
	}
	if result.RequestID == "" {
		return "", fmt.Errorf("fibe: async response missing request_id")
	}
	return fmt.Sprintf(statusPathFmt, result.RequestID), nil
}

func normalizeStatusPath(statusURL string) (string, error) {
	if strings.HasPrefix(statusURL, "http://") || strings.HasPrefix(statusURL, "https://") {
		u, err := url.Parse(statusURL)
		if err != nil {
			return "", fmt.Errorf("fibe: invalid async status_url: %w", err)
		}
		path := u.EscapedPath()
		if u.RawQuery != "" {
			path += "?" + u.RawQuery
		}
		if path == "" {
			return "", fmt.Errorf("fibe: async status_url has no path")
		}
		return path, nil
	}
	return statusURL, nil
}
