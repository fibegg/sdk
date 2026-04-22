package fibe

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type AgentService struct {
	client *Client
}

func (s *AgentService) List(ctx context.Context, params *AgentListParams) (*ListResult[Agent], error) {
	path := "/api/agents" + buildQuery(params)
	return doList[Agent](s.client, ctx, path)
}

func (s *AgentService) Get(ctx context.Context, id int64) (*Agent, error) {
	var result Agent
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d", id), nil, &result)
	return &result, err
}

func (s *AgentService) Create(ctx context.Context, params *AgentCreateParams) (*Agent, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	prepared, err := prepareAgentCreateParams(params)
	if err != nil {
		return nil, err
	}
	var result Agent
	body := map[string]any{"agent": prepared}
	err = s.client.do(ctx, http.MethodPost, "/api/agents", body, &result)
	return &result, err
}

func (s *AgentService) Update(ctx context.Context, id int64, params *AgentUpdateParams) (*Agent, error) {
	var result Agent
	body := map[string]any{"agent": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/agents/%d", id), body, &result)
	return &result, err
}

func (s *AgentService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/agents/%d", id), nil, nil)
}

func (s *AgentService) Chat(ctx context.Context, id int64, params *AgentChatParams) (map[string]any, error) {
	var result map[string]any
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/chat", id), params, &result)
	return result, err
}

func (s *AgentService) Authenticate(ctx context.Context, id int64, code, token *string) (*Agent, error) {
	body := map[string]any{}
	if code != nil {
		body["code"] = *code
	}
	if token != nil {
		body["token"] = *token
	}
	var result Agent
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/authenticate", id), body, &result)
	return &result, err
}

func (s *AgentService) StartChat(ctx context.Context, id, marqueeID int64) (*AgentChatSession, error) {
	var result AgentChatSession
	body := map[string]any{"marquee_id": marqueeID}
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/start_chat", id), body, &result)
	return &result, err
}

func (s *AgentService) RuntimeStatus(ctx context.Context, id int64) (*AgentRuntimeStatus, error) {
	var result AgentRuntimeStatus
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/runtime_status", id), nil, &result)
	return &result, err
}

func (s *AgentService) PurgeChat(ctx context.Context, id int64) (*AgentChatSession, error) {
	var result AgentChatSession
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/purge_chat", id), nil, &result)
	return &result, err
}

func (s *AgentService) RevokeGitHubToken(ctx context.Context, id int64) (*Agent, error) {
	var result Agent
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/revoke_github_token", id), nil, &result)
	return &result, err
}

func (s *AgentService) Duplicate(ctx context.Context, id int64) (*Agent, error) {
	var result Agent
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/duplicate", id), nil, &result)
	return &result, err
}

func (s *AgentService) AddMountedFile(ctx context.Context, id int64, file io.Reader, fileName string, params *MountedFileParams) (*Agent, error) {
	fields := map[string]string{
		"mount_path": params.MountPath,
	}
	if params.ReadOnly != nil {
		if *params.ReadOnly {
			fields["readonly"] = "true"
		} else {
			fields["readonly"] = "false"
		}
	}
	for _, svc := range params.TargetServices {
		fields["target_services[]"] = svc
	}
	path := fmt.Sprintf("/api/agents/%d/add_mounted_file", id)
	var result Agent
	err := s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, &result)
	if err != nil {
		return nil, err
	}
	if result.ID != 0 {
		return &result, nil
	}
	return s.Get(ctx, id)
}

func (s *AgentService) AddMountedFileFromArtefact(ctx context.Context, id int64, artefactID int64, params *MountedFileParams) (*Agent, error) {
	body := map[string]any{
		"artefact_id": artefactID,
		"mount_path":  params.MountPath,
	}
	if params.ReadOnly != nil {
		body["readonly"] = *params.ReadOnly
	}
	if len(params.TargetServices) > 0 {
		body["target_services"] = params.TargetServices
	}
	path := fmt.Sprintf("/api/agents/%d/add_mounted_file", id)
	var result Agent
	err := s.client.do(ctx, http.MethodPost, path, body, &result)
	return &result, err
}

func (s *AgentService) UpdateMountedFile(ctx context.Context, id int64, params *MountedFileUpdateParams) (*Agent, error) {
	var result Agent
	path := fmt.Sprintf("/api/agents/%d/update_mounted_file", id)
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}

func (s *AgentService) RemoveMountedFile(ctx context.Context, id int64, filename string) (*Agent, error) {
	var result Agent
	path := fmt.Sprintf("/api/agents/%d/remove_mounted_file", id)
	body := map[string]any{"filename": filename}
	err := s.client.do(ctx, http.MethodDelete, path, body, &result)
	return &result, err
}

func (s *AgentService) GetMessages(ctx context.Context, id int64) (*AgentData, error) {
	var result AgentData
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/messages", id), nil, &result)
	return &result, err
}

func (s *AgentService) UpdateMessages(ctx context.Context, id int64, content any) error {
	body := map[string]any{"content": content}
	return s.client.do(ctx, http.MethodPut, fmt.Sprintf("/api/agents/%d/messages", id), body, nil)
}

func (s *AgentService) GetActivity(ctx context.Context, id int64) (*AgentData, error) {
	var result AgentData
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/activity", id), nil, &result)
	return &result, err
}

func (s *AgentService) UpdateActivity(ctx context.Context, id int64, content any) error {
	body := map[string]any{"content": content}
	return s.client.do(ctx, http.MethodPut, fmt.Sprintf("/api/agents/%d/activity", id), body, nil)
}

func (s *AgentService) GetGitHubToken(ctx context.Context, id int64) (*GitHubToken, error) {
	var result GitHubToken
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/github_token", id), nil, &result)
	return &result, err
}

func (s *AgentService) GetGitHubTokenForRepo(ctx context.Context, id int64, repo string) (*GitHubToken, error) {
	var result GitHubToken
	values := url.Values{}
	values.Set("repo", repo)
	path := fmt.Sprintf("/api/agents/%d/github_token?%s", id, values.Encode())
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *AgentService) GetGiteaToken(ctx context.Context, id int64) (*GiteaToken, error) {
	var result GiteaToken
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/gitea_token", id), nil, &result)
	return &result, err
}

func (s *AgentService) GetRawProviders(ctx context.Context, id int64) (*AgentData, error) {
	var result AgentData
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/agents/%d/raw_providers", id), nil, &result)
	return &result, err
}

func prepareAgentCreateParams(params *AgentCreateParams) (*AgentCreateParams, error) {
	prepared := *params
	if len(params.Mounts) == 0 {
		return &prepared, nil
	}
	prepared.Mounts = make([]AgentMountSpec, len(params.Mounts))
	for i, mount := range params.Mounts {
		next := mount
		if next.ContentPath != "" {
			data, err := os.ReadFile(next.ContentPath)
			if err != nil {
				return nil, fmt.Errorf("read mount content_path %s: %w", next.ContentPath, err)
			}
			next.ContentBase64 = base64.StdEncoding.EncodeToString(data)
			if next.Filename == "" {
				next.Filename = filepath.Base(next.ContentPath)
			}
			next.ContentPath = ""
		}
		prepared.Mounts[i] = next
	}
	return &prepared, nil
}

func (s *AgentService) UpdateRawProviders(ctx context.Context, id int64, content any) error {
	body := map[string]any{"content": content}
	return s.client.do(ctx, http.MethodPut, fmt.Sprintf("/api/agents/%d/raw_providers", id), body, nil)
}
