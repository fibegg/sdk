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
	path := fmt.Sprintf("/api/agents/%d/feedbacks", agentID)
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Feedback](s.client, ctx, path)
}

func (s *FeedbackService) Get(ctx context.Context, agentID, id int64) (*Feedback, error) {
	var result Feedback
	path := fmt.Sprintf("/api/agents/%d/feedbacks/%d", agentID, id)
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *FeedbackService) Create(ctx context.Context, agentID int64, params *FeedbackCreateParams) (*Feedback, error) {
	var result Feedback
	path := fmt.Sprintf("/api/agents/%d/feedbacks", agentID)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *FeedbackService) Update(ctx context.Context, agentID, id int64, params *FeedbackUpdateParams) (*Feedback, error) {
	var result Feedback
	path := fmt.Sprintf("/api/agents/%d/feedbacks/%d", agentID, id)
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}

func (s *FeedbackService) Delete(ctx context.Context, agentID, id int64) error {
	path := fmt.Sprintf("/api/agents/%d/feedbacks/%d", agentID, id)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil)
}
