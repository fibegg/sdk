package fibe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type MarqueeService struct {
	client *Client
}

func (s *MarqueeService) List(ctx context.Context, params *MarqueeListParams) (*ListResult[Marquee], error) {
	path := "/api/marquees" + buildQuery(params)
	return doList[Marquee](s.client, ctx, path)
}

func (s *MarqueeService) Get(ctx context.Context, id int64) (*Marquee, error) {
	var result Marquee
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/marquees/%d", id), nil, &result)
	return &result, err
}

func (s *MarqueeService) Create(ctx context.Context, params *MarqueeCreateParams) (*Marquee, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Marquee
	body, err := marqueeRequestBody(params)
	if err != nil {
		return nil, err
	}
	err = s.client.do(ctx, http.MethodPost, "/api/marquees", body, &result)
	return &result, err
}

func (s *MarqueeService) Update(ctx context.Context, id int64, params *MarqueeUpdateParams) (*Marquee, error) {
	var result Marquee
	body, err := marqueeRequestBody(params)
	if err != nil {
		return nil, err
	}
	err = s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/marquees/%d", id), body, &result)
	return &result, err
}

func (s *MarqueeService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/marquees/%d", id), nil, nil)
}

func (s *MarqueeService) GenerateSSHKey(ctx context.Context, id int64) (*SSHKeyResult, error) {
	var result SSHKeyResult
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/marquees/%d/generate_ssh_key", id), nil, &result)
	return &result, err
}

// TestConnection tests SSH connectivity to the marquee host.
// The API returns 202 Accepted; this method auto-polls for the result.
func (s *MarqueeService) TestConnection(ctx context.Context, id int64) (*ConnectionTestResult, error) {
	var result ConnectionTestResult
	statusFmt := fmt.Sprintf("/api/marquees/%d/test_connection/%%s", id)
	err := s.client.doAsync(ctx, http.MethodPost, fmt.Sprintf("/api/marquees/%d/test_connection", id), statusFmt, nil, &result)
	return &result, err
}

func (s *MarqueeService) AutoconnectToken(ctx context.Context, params *AutoconnectTokenParams) (*AutoconnectTokenResult, error) {
	var result AutoconnectTokenResult
	err := s.client.do(ctx, http.MethodPost, "/api/marquees/autoconnect_token", params, &result)
	return &result, err
}

func marqueeRequestBody(params any) (map[string]any, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	var marquee map[string]any
	if err := json.Unmarshal(data, &marquee); err != nil {
		return nil, err
	}
	if creds, ok := marquee["dns_credentials"]; ok && creds != nil {
		encoded, err := json.Marshal(creds)
		if err != nil {
			return nil, err
		}
		marquee["dns_credentials"] = string(encoded)
	}
	return map[string]any{"marquee": marquee}, nil
}
