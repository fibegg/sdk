package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type FeedbackService struct {
	client *Client
}

func (s *FeedbackService) List(ctx context.Context, agentID int64, params *FeedbackListParams) (*ListResult[Feedback], error) {
	return s.ListByAgentIdentifier(ctx, int64Identifier(agentID), params)
}

func (s *FeedbackService) ListByAgentIdentifier(ctx context.Context, agentIdentifier string, params *FeedbackListParams) (*ListResult[Feedback], error) {
	path := identifierPath("/api/agents", agentIdentifier) + "/feedbacks"
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Feedback](s.client, ctx, path)
}

func (s *FeedbackService) Get(ctx context.Context, agentID, id int64) (*Feedback, error) {
	return s.GetByAgentIdentifier(ctx, int64Identifier(agentID), id)
}

func (s *FeedbackService) GetByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64) (*Feedback, error) {
	var result Feedback
	path := fmt.Sprintf("%s/feedbacks/%d", identifierPath("/api/agents", agentIdentifier), id)
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *FeedbackService) Create(ctx context.Context, agentID int64, params *FeedbackCreateParams) (*Feedback, error) {
	return s.CreateByAgentIdentifier(ctx, int64Identifier(agentID), params)
}

func (s *FeedbackService) CreateByAgentIdentifier(ctx context.Context, agentIdentifier string, params *FeedbackCreateParams) (*Feedback, error) {
	var result Feedback
	path := identifierPath("/api/agents", agentIdentifier) + "/feedbacks"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *FeedbackService) Update(ctx context.Context, agentID, id int64, params *FeedbackUpdateParams) (*Feedback, error) {
	return s.UpdateByAgentIdentifier(ctx, int64Identifier(agentID), id, params)
}

func (s *FeedbackService) UpdateByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64, params *FeedbackUpdateParams) (*Feedback, error) {
	var result Feedback
	path := fmt.Sprintf("%s/feedbacks/%d", identifierPath("/api/agents", agentIdentifier), id)
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}

func (s *FeedbackService) Delete(ctx context.Context, agentID, id int64) error {
	return s.DeleteByAgentIdentifier(ctx, int64Identifier(agentID), id)
}

func (s *FeedbackService) DeleteByAgentIdentifier(ctx context.Context, agentIdentifier string, id int64) error {
	path := fmt.Sprintf("%s/feedbacks/%d", identifierPath("/api/agents", agentIdentifier), id)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil)
}
