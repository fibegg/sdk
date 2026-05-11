package fibe

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type ArtefactService struct {
	client *Client
}

func (s *ArtefactService) ListAll(ctx context.Context, params *ArtefactListParams) (*ListResult[Artefact], error) {
	path := "/api/artefacts"
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Artefact](s.client, ctx, path)
}

func (s *ArtefactService) GetByID(ctx context.Context, id int64) (*Artefact, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *ArtefactService) GetByIdentifier(ctx context.Context, identifier string) (*Artefact, error) {
	var result Artefact
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/artefacts", identifier), nil, &result)
	return &result, err
}

func (s *ArtefactService) List(ctx context.Context, agentID int64, params *ArtefactListParams) (*ListResult[Artefact], error) {
	return s.ListByAgentIdentifier(ctx, int64Identifier(agentID), params)
}

func (s *ArtefactService) ListByAgentIdentifier(ctx context.Context, agentIdentifier string, params *ArtefactListParams) (*ListResult[Artefact], error) {
	path := identifierPath("/api/agents", agentIdentifier) + "/artefacts"
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Artefact](s.client, ctx, path)
}

func (s *ArtefactService) Get(ctx context.Context, agentID, id int64) (*Artefact, error) {
	return s.GetByAgentAndArtefactIdentifier(ctx, int64Identifier(agentID), int64Identifier(id))
}

func (s *ArtefactService) GetByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64) (*Artefact, error) {
	return s.GetByAgentAndArtefactIdentifier(ctx, agentIdentifier, int64Identifier(id))
}

func (s *ArtefactService) GetByAgentAndArtefactIdentifier(ctx context.Context, agentIdentifier string, artefactIdentifier string) (*Artefact, error) {
	var result Artefact
	path := identifierPath(identifierPath("/api/agents", agentIdentifier)+"/artefacts", artefactIdentifier)
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *ArtefactService) Create(ctx context.Context, agentID int64, params *ArtefactCreateParams, file io.Reader, fileName string) (*Artefact, error) {
	return s.CreateByAgentIdentifier(ctx, int64Identifier(agentID), params, file, fileName)
}

func (s *ArtefactService) CreateByAgentIdentifier(ctx context.Context, agentIdentifier string, params *ArtefactCreateParams, file io.Reader, fileName string) (*Artefact, error) {
	fields := artefactCreateFields(params)
	path := identifierPath("/api/agents", agentIdentifier) + "/artefacts"
	var result Artefact
	err := s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ArtefactService) CreateOwned(ctx context.Context, params *ArtefactCreateParams, file io.Reader, fileName string) (*Artefact, error) {
	fields := artefactCreateFields(params)
	var result Artefact
	err := s.client.doMultipart(ctx, http.MethodPost, "/api/artefacts", fields, "file", fileName, file, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ArtefactService) UpdateByID(ctx context.Context, id int64, params *ArtefactUpdateParams) (*Artefact, error) {
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ArtefactService) UpdateByIdentifier(ctx context.Context, identifier string, params *ArtefactUpdateParams) (*Artefact, error) {
	var result Artefact
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/artefacts", identifier), params, &result)
	return &result, err
}

func (s *ArtefactService) UpdateByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64, params *ArtefactUpdateParams) (*Artefact, error) {
	return s.UpdateByAgentAndArtefactIdentifier(ctx, agentIdentifier, int64Identifier(id), params)
}

func (s *ArtefactService) UpdateByAgentAndArtefactIdentifier(ctx context.Context, agentIdentifier string, artefactIdentifier string, params *ArtefactUpdateParams) (*Artefact, error) {
	var result Artefact
	path := identifierPath(identifierPath("/api/agents", agentIdentifier)+"/artefacts", artefactIdentifier)
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}

func artefactCreateFields(params *ArtefactCreateParams) map[string]string {
	fields := map[string]string{
		"name": params.Name,
	}
	if params.Description != "" {
		fields["description"] = params.Description
	}
	if params.Body != "" {
		fields["body"] = params.Body
	}
	if params.PlainText != nil {
		fields["plain_text"] = fmt.Sprintf("%t", *params.PlainText)
	}
	if params.Skill != nil {
		fields["skill"] = fmt.Sprintf("%t", *params.Skill)
	}
	if params.SkillEnabled != nil {
		fields["skill_enabled"] = fmt.Sprintf("%t", *params.SkillEnabled)
	}
	if params.AgentID != nil {
		fields["agent_id"] = fmt.Sprintf("%d", *params.AgentID)
	}
	if params.AgentIdentifier != "" {
		fields["agent_id"] = params.AgentIdentifier
	}
	if params.PlaygroundID != nil {
		fields["playground_id"] = fmt.Sprintf("%d", *params.PlaygroundID)
	}
	if params.PlaygroundIdentifier != "" {
		fields["playground_id"] = params.PlaygroundIdentifier
	}
	return fields
}

func (s *ArtefactService) Download(ctx context.Context, agentID, id int64) (io.ReadCloser, string, string, error) {
	return s.DownloadByAgentAndArtefactIdentifier(ctx, int64Identifier(agentID), int64Identifier(id))
}

func (s *ArtefactService) DownloadByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64) (io.ReadCloser, string, string, error) {
	return s.DownloadByAgentAndArtefactIdentifier(ctx, agentIdentifier, int64Identifier(id))
}

func (s *ArtefactService) DownloadByAgentAndArtefactIdentifier(ctx context.Context, agentIdentifier string, artefactIdentifier string) (io.ReadCloser, string, string, error) {
	path := identifierPath(identifierPath("/api/agents", agentIdentifier)+"/artefacts", artefactIdentifier) + "/download"
	return s.client.doDownload(ctx, path)
}

func (s *ArtefactService) DownloadByID(ctx context.Context, id int64) (io.ReadCloser, string, string, error) {
	return s.DownloadByIdentifier(ctx, int64Identifier(id))
}

func (s *ArtefactService) DownloadByIdentifier(ctx context.Context, identifier string) (io.ReadCloser, string, string, error) {
	path := identifierPath("/api/artefacts", identifier) + "/download"
	return s.client.doDownload(ctx, path)
}
