package fibe

import (
	"context"
	"encoding/json"
	"net/http"
)

type HostService struct {
	client *Client
}

func (s *HostService) List(ctx context.Context, params *HostListParams) (*ListResult[Host], error) {
	path := "/api/hosts" + buildQuery(params)
	return doList[Host](s.client, ctx, path)
}

func (s *HostService) Get(ctx context.Context, id int64) (*Host, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *HostService) GetByIdentifier(ctx context.Context, identifier string) (*Host, error) {
	var result Host
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/hosts", identifier), nil, &result)
	return &result, err
}

func (s *HostService) Create(ctx context.Context, params *HostCreateParams) (*Host, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Host
	body, err := hostRequestBody(params)
	if err != nil {
		return nil, err
	}
	err = s.client.do(ctx, http.MethodPost, "/api/hosts", body, &result)
	return &result, err
}

func (s *HostService) Update(ctx context.Context, id int64, params *HostUpdateParams) (*Host, error) {
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *HostService) UpdateByIdentifier(ctx context.Context, identifier string, params *HostUpdateParams) (*Host, error) {
	var result Host
	body, err := hostRequestBody(params)
	if err != nil {
		return nil, err
	}
	err = s.client.do(ctx, http.MethodPatch, identifierPath("/api/hosts", identifier), body, &result)
	return &result, err
}

func (s *HostService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *HostService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/hosts", identifier), nil, nil)
}

func (s *HostService) GenerateSSHKey(ctx context.Context, id int64) (*SSHKeyResult, error) {
	return s.GenerateSSHKeyByIdentifier(ctx, int64Identifier(id))
}

func (s *HostService) GenerateSSHKeyByIdentifier(ctx context.Context, identifier string) (*SSHKeyResult, error) {
	var result SSHKeyResult
	path := identifierPath("/api/hosts", identifier)
	err := s.client.doAsync(ctx, http.MethodPost, path+"/ssh_keys", "/api/async_requests/%s", nil, &result)
	return &result, err
}

// TestConnection tests SSH connectivity to the host host.
// The API returns 202 Accepted; this method auto-polls for the result.
func (s *HostService) TestConnection(ctx context.Context, id int64) (*ConnectionTestResult, error) {
	return s.TestConnectionByIdentifier(ctx, int64Identifier(id))
}

func (s *HostService) TestConnectionByIdentifier(ctx context.Context, identifier string) (*ConnectionTestResult, error) {
	var result ConnectionTestResult
	path := identifierPath("/api/hosts", identifier)
	err := s.client.doAsync(ctx, http.MethodPost, path+"/connection_tests", "/api/async_requests/%s", nil, &result)
	return &result, err
}

func (s *HostService) AutoconnectToken(ctx context.Context, params *AutoconnectTokenParams) (*AutoconnectTokenResult, error) {
	var result AutoconnectTokenResult
	err := s.client.do(ctx, http.MethodPost, "/api/autoconnect_tokens", params, &result)
	return &result, err
}

func hostRequestBody(params any) (map[string]any, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	var host map[string]any
	if err := json.Unmarshal(data, &host); err != nil {
		return nil, err
	}
	if creds, ok := host["dns_credentials"]; ok && creds != nil {
		encoded, err := json.Marshal(creds)
		if err != nil {
			return nil, err
		}
		host["dns_credentials"] = string(encoded)
	}
	return map[string]any{"host": host}, nil
}
