package fibe

import (
	"context"
	"encoding/json"
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
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *MarqueeService) GetByIdentifier(ctx context.Context, identifier string) (*Marquee, error) {
	var result Marquee
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/marquees", identifier), nil, &result)
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
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *MarqueeService) UpdateByIdentifier(ctx context.Context, identifier string, params *MarqueeUpdateParams) (*Marquee, error) {
	var result Marquee
	body, err := marqueeRequestBody(params)
	if err != nil {
		return nil, err
	}
	err = s.client.do(ctx, http.MethodPatch, identifierPath("/api/marquees", identifier), body, &result)
	return &result, err
}

func (s *MarqueeService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *MarqueeService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/marquees", identifier), nil, nil)
}

func (s *MarqueeService) GenerateSSHKey(ctx context.Context, id int64) (*SSHKeyResult, error) {
	return s.GenerateSSHKeyByIdentifier(ctx, int64Identifier(id))
}

func (s *MarqueeService) GenerateSSHKeyByIdentifier(ctx context.Context, identifier string) (*SSHKeyResult, error) {
	var result SSHKeyResult
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/marquees", identifier)+"/generate_ssh_key", nil, &result)
	return &result, err
}

// TestConnection tests SSH connectivity to the marquee host.
// The API returns 202 Accepted; this method auto-polls for the result.
func (s *MarqueeService) TestConnection(ctx context.Context, id int64) (*ConnectionTestResult, error) {
	return s.TestConnectionByIdentifier(ctx, int64Identifier(id))
}

func (s *MarqueeService) TestConnectionByIdentifier(ctx context.Context, identifier string) (*ConnectionTestResult, error) {
	var result ConnectionTestResult
	path := identifierPath("/api/marquees", identifier)
	err := s.client.doAsync(ctx, http.MethodPost, path+"/test_connection", path+"/test_connection/%s", nil, &result)
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
