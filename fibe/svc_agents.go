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
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) GetByIdentifier(ctx context.Context, identifier string) (*Agent, error) {
	var result Agent
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/agents", identifier), nil, &result)
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
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *AgentService) UpdateByIdentifier(ctx context.Context, identifier string, params *AgentUpdateParams) (*Agent, error) {
	var result Agent
	body := map[string]any{"agent": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/agents", identifier), body, &result)
	return &result, err
}

func (s *AgentService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/agents", identifier), nil, nil)
}

func (s *AgentService) Chat(ctx context.Context, id int64, params *AgentChatParams) (map[string]any, error) {
	return s.ChatByIdentifier(ctx, int64Identifier(id), params)
}

func (s *AgentService) ChatByIdentifier(ctx context.Context, identifier string, params *AgentChatParams) (map[string]any, error) {
	var result map[string]any
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/agents", identifier)+"/chat", params, &result)
	return result, err
}

func (s *AgentService) Authenticate(ctx context.Context, id int64, code, token *string) (*Agent, error) {
	return s.AuthenticateByIdentifier(ctx, int64Identifier(id), code, token)
}

func (s *AgentService) AuthenticateByIdentifier(ctx context.Context, identifier string, code, token *string) (*Agent, error) {
	body := map[string]any{}
	if code != nil {
		body["code"] = *code
	}
	if token != nil {
		body["token"] = *token
	}
	var result Agent
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/agents", identifier)+"/authenticate", body, &result)
	return &result, err
}

func (s *AgentService) StartChat(ctx context.Context, id, marqueeID int64) (*AgentChatSession, error) {
	var result AgentChatSession
	body := map[string]any{"marquee_id": marqueeID}
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/agents/%d/start_chat", id), body, &result)
	return &result, err
}

func (s *AgentService) StartChatByIdentifier(ctx context.Context, id int64, marqueeIdentifier string) (*AgentChatSession, error) {
	return s.StartChatByAgentIdentifier(ctx, int64Identifier(id), marqueeIdentifier)
}

func (s *AgentService) StartChatByAgentIdentifier(ctx context.Context, agentIdentifier string, marqueeIdentifier string) (*AgentChatSession, error) {
	var result AgentChatSession
	body := map[string]any{"marquee_id": marqueeIdentifier}
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/agents", agentIdentifier)+"/start_chat", body, &result)
	return &result, err
}

func (s *AgentService) RestartChat(ctx context.Context, id int64) (*AgentChatSession, error) {
	return s.RestartChatByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) RestartChatByIdentifier(ctx context.Context, identifier string) (*AgentChatSession, error) {
	var result AgentChatSession
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/agents", identifier)+"/restart_chat", nil, &result)
	return &result, err
}

func (s *AgentService) RuntimeStatus(ctx context.Context, id int64) (*AgentRuntimeStatus, error) {
	return s.RuntimeStatusByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) RuntimeStatusByIdentifier(ctx context.Context, identifier string) (*AgentRuntimeStatus, error) {
	var result AgentRuntimeStatus
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/agents", identifier)+"/runtime_status", nil, &result)
	return &result, err
}

func (s *AgentService) PurgeChat(ctx context.Context, id int64) (*AgentChatSession, error) {
	return s.PurgeChatByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) PurgeChatByIdentifier(ctx context.Context, identifier string) (*AgentChatSession, error) {
	var result AgentChatSession
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/agents", identifier)+"/purge_chat", nil, &result)
	return &result, err
}

func (s *AgentService) Duplicate(ctx context.Context, id int64) (*Agent, error) {
	return s.DuplicateByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) DuplicateByIdentifier(ctx context.Context, identifier string) (*Agent, error) {
	var result Agent
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/agents", identifier)+"/duplicate", nil, &result)
	return &result, err
}

func (s *AgentService) AddMountedFile(ctx context.Context, id int64, file io.Reader, fileName string, params *MountedFileParams) (*Agent, error) {
	return s.AddMountedFileByIdentifier(ctx, int64Identifier(id), file, fileName, params)
}

func (s *AgentService) AddMountedFileByIdentifier(ctx context.Context, identifier string, file io.Reader, fileName string, params *MountedFileParams) (*Agent, error) {
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
	path := identifierPath("/api/agents", identifier) + "/add_mounted_file"
	var result Agent
	err := s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, &result)
	if err != nil {
		return nil, err
	}
	if result.ID != 0 {
		return &result, nil
	}
	return s.GetByIdentifier(ctx, identifier)
}

func (s *AgentService) AddMountedFileFromArtefact(ctx context.Context, id int64, artefactID int64, params *MountedFileParams) (*Agent, error) {
	return s.AddMountedFileFromArtefactByIdentifier(ctx, int64Identifier(id), artefactID, params)
}

func (s *AgentService) AddMountedFileFromArtefactByIdentifier(ctx context.Context, identifier string, artefactID int64, params *MountedFileParams) (*Agent, error) {
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
	path := identifierPath("/api/agents", identifier) + "/add_mounted_file"
	var result Agent
	err := s.client.do(ctx, http.MethodPost, path, body, &result)
	return &result, err
}

func (s *AgentService) UpdateMountedFile(ctx context.Context, id int64, params *MountedFileUpdateParams) (*Agent, error) {
	return s.UpdateMountedFileByIdentifier(ctx, int64Identifier(id), params)
}

func (s *AgentService) UpdateMountedFileByIdentifier(ctx context.Context, identifier string, params *MountedFileUpdateParams) (*Agent, error) {
	var result Agent
	path := identifierPath("/api/agents", identifier) + "/update_mounted_file"
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}

func (s *AgentService) RemoveMountedFile(ctx context.Context, id int64, filename string) (*Agent, error) {
	return s.RemoveMountedFileByIdentifier(ctx, int64Identifier(id), filename)
}

func (s *AgentService) RemoveMountedFileByIdentifier(ctx context.Context, identifier string, filename string) (*Agent, error) {
	var result Agent
	path := identifierPath("/api/agents", identifier) + "/remove_mounted_file"
	body := map[string]any{"filename": filename}
	err := s.client.do(ctx, http.MethodDelete, path, body, &result)
	return &result, err
}

func (s *AgentService) GetMessages(ctx context.Context, id int64) (*AgentData, error) {
	return s.GetMessagesByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) GetMessagesByIdentifier(ctx context.Context, identifier string) (*AgentData, error) {
	var result AgentData
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/agents", identifier)+"/messages", nil, &result)
	return &result, err
}

func (s *AgentService) UpdateMessages(ctx context.Context, id int64, content any) error {
	return s.UpdateMessagesByIdentifier(ctx, int64Identifier(id), content)
}

func (s *AgentService) UpdateMessagesByIdentifier(ctx context.Context, identifier string, content any) error {
	body := map[string]any{"content": content}
	return s.client.do(ctx, http.MethodPut, identifierPath("/api/agents", identifier)+"/messages", body, nil)
}

func (s *AgentService) GetActivity(ctx context.Context, id int64) (*AgentData, error) {
	return s.GetActivityByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) GetActivityByIdentifier(ctx context.Context, identifier string) (*AgentData, error) {
	var result AgentData
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/agents", identifier)+"/activity", nil, &result)
	return &result, err
}

func (s *AgentService) UpdateActivity(ctx context.Context, id int64, content any) error {
	return s.UpdateActivityByIdentifier(ctx, int64Identifier(id), content)
}

func (s *AgentService) UpdateActivityByIdentifier(ctx context.Context, identifier string, content any) error {
	body := map[string]any{"content": content}
	return s.client.do(ctx, http.MethodPut, identifierPath("/api/agents", identifier)+"/activity", body, nil)
}

func (s *AgentService) GetGiteaToken(ctx context.Context, id int64) (*GiteaToken, error) {
	return s.GetGiteaTokenByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) GetGiteaTokenByIdentifier(ctx context.Context, identifier string) (*GiteaToken, error) {
	var result GiteaToken
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/agents", identifier)+"/gitea_token", nil, &result)
	return &result, err
}

func (s *AgentService) GetGitHubTokenForRepo(ctx context.Context, id int64, repo string) (*GitHubToken, error) {
	values := url.Values{}
	values.Set("repo", repo)
	var result GitHubToken
	err := s.client.do(ctx, http.MethodGet, "/api/github_token?"+values.Encode(), nil, &result)
	return &result, err
}

func (s *AgentService) RevokeGitHubToken(ctx context.Context, id int64) (map[string]any, error) {
	return s.RevokeGitHubTokenByIdentifier(ctx, int64Identifier(id))
}

func (s *AgentService) RevokeGitHubTokenByIdentifier(ctx context.Context, identifier string) (map[string]any, error) {
	var result map[string]any
	err := s.client.do(ctx, http.MethodDelete, identifierPath("/api/agents", identifier)+"/github_token", nil, &result)
	return result, err
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
