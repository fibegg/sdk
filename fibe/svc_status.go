package fibe

import (
	"context"
	"net/http"
)

// StatusService provides access to the dashboard status endpoint.
type StatusService struct {
	client *Client
}

// Status contains a summary of the authenticated user's resources.
type Status struct {
	Playgrounds struct {
		Total   int `json:"total"`
		Active  int `json:"active"`
		Stopped int `json:"stopped"`
	} `json:"playgrounds"`
	Agents struct {
		Total         int `json:"total"`
		Authenticated int `json:"authenticated"`
	} `json:"agents"`
	Props        int `json:"props"`
	Playspecs    int `json:"playspecs"`
	Marquees     int `json:"marquees"`
	Secrets      int `json:"secrets"`
	Teams        int `json:"teams"`
	APIKeys      int `json:"api_keys"`
	Subscription struct {
		Plan            string `json:"plan"`
		PlaygroundLimit int    `json:"playground_limit"`
	} `json:"subscription"`

	ResourceQuotas map[string]ResourceQuotaEntry `json:"resource_quotas,omitempty"`
	PerParentCaps  map[string]*int               `json:"per_parent_caps,omitempty"`
	RateLimits     *RateLimitsSection            `json:"rate_limits,omitempty"`
}

// ResourceQuotaEntry reports usage, limit, and status for one resource type.
// Limit is nil when the quota is unlimited.
type ResourceQuotaEntry struct {
	Used   int    `json:"used"`
	Limit  *int   `json:"limit"`
	Status string `json:"status"`
}

// RateLimitsSection carries rate-limit state for the authenticated player.
// Only populated when the request is authenticated via API key.
type RateLimitsSection struct {
	API *RateLimitInfo `json:"api,omitempty"`
}

// RateLimitInfo describes a single rate-limit bucket.
type RateLimitInfo struct {
	Limit        int `json:"limit"`
	Remaining    int `json:"remaining"`
	ResetSeconds int `json:"reset_seconds"`
}

// Get returns a summary of all the authenticated user's resources.
// This is designed for LLM agents to gather context in a single request.
func (s *StatusService) Get(ctx context.Context) (*Status, error) {
	var result Status
	err := s.client.do(ctx, http.MethodGet, "/api/status", nil, &result)
	return &result, err
}
