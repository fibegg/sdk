package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type MutationService struct {
	client *Client
}

type MutationListParams struct {
	Status        string `url:"status,omitempty"`
	CuredAfter    string `url:"cured_after,omitempty"`
	CuredBefore   string `url:"cured_before,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}

func (s *MutationService) List(ctx context.Context, propID int64, params *MutationListParams) (*ListResult[Mutation], error) {
	path := fmt.Sprintf("/api/props/%d/mutations", propID)
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Mutation](s.client, ctx, path)
}

func (s *MutationService) Create(ctx context.Context, propID int64, params *MutationCreateParams) (*Mutation, error) {
	var result Mutation
	path := fmt.Sprintf("/api/props/%d/mutations", propID)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *MutationService) Update(ctx context.Context, propID, id int64, params *MutationUpdateParams) (*Mutation, error) {
	var result Mutation
	path := fmt.Sprintf("/api/props/%d/mutations/%d", propID, id)
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}
