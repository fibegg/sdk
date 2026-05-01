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
	var result Artefact
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/artefacts/%d", id), nil, &result)
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
	return s.GetByAgentIdentifier(ctx, int64Identifier(agentID), id)
}

func (s *ArtefactService) GetByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64) (*Artefact, error) {
	var result Artefact
	path := fmt.Sprintf("%s/artefacts/%d", identifierPath("/api/agents", agentIdentifier), id)
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *ArtefactService) Create(ctx context.Context, agentID int64, params *ArtefactCreateParams, file io.Reader, fileName string) (*Artefact, error) {
	return s.CreateByAgentIdentifier(ctx, int64Identifier(agentID), params, file, fileName)
}

func (s *ArtefactService) CreateByAgentIdentifier(ctx context.Context, agentIdentifier string, params *ArtefactCreateParams, file io.Reader, fileName string) (*Artefact, error) {
	fields := map[string]string{
		"name": params.Name,
	}
	if params.Description != "" {
		fields["description"] = params.Description
	}
	if params.PlaygroundID != nil {
		fields["playground_id"] = fmt.Sprintf("%d", *params.PlaygroundID)
	}
	if params.PlaygroundIdentifier != "" {
		fields["playground_id"] = params.PlaygroundIdentifier
	}
	path := identifierPath("/api/agents", agentIdentifier) + "/artefacts"
	var result Artefact
	err := s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ArtefactService) Download(ctx context.Context, agentID, id int64) (io.ReadCloser, string, string, error) {
	return s.DownloadByAgentIdentifier(ctx, int64Identifier(agentID), id)
}

func (s *ArtefactService) DownloadByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64) (io.ReadCloser, string, string, error) {
	path := fmt.Sprintf("%s/artefacts/%d/download", identifierPath("/api/agents", agentIdentifier), id)
	return s.client.doDownload(ctx, path)
}

func (s *ArtefactService) DownloadByID(ctx context.Context, id int64) (io.ReadCloser, string, string, error) {
	path := fmt.Sprintf("/api/artefacts/%d/download", id)
	return s.client.doDownload(ctx, path)
}
