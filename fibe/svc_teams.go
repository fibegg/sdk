package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type TeamService struct {
	client *Client
}

func (s *TeamService) List(ctx context.Context, params *TeamListParams) (*ListResult[Team], error) {
	path := "/api/teams" + buildQuery(params)
	return doList[Team](s.client, ctx, path)
}

func (s *TeamService) Get(ctx context.Context, id int64) (*Team, error) {
	var result Team
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/teams/%d", id), nil, &result)
	return &result, err
}

func (s *TeamService) Create(ctx context.Context, params *TeamCreateParams) (*Team, error) {
	var result Team
	body := map[string]any{"team": params}
	err := s.client.do(ctx, http.MethodPost, "/api/teams", body, &result)
	return &result, err
}

func (s *TeamService) Update(ctx context.Context, id int64, params *TeamUpdateParams) (*Team, error) {
	var result Team
	body := map[string]any{"team": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/teams/%d", id), body, &result)
	return &result, err
}

func (s *TeamService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/teams/%d", id), nil, nil)
}

func (s *TeamService) TransferLeadership(ctx context.Context, id, newLeaderID int64) (*Team, error) {
	var result Team
	body := map[string]any{"new_leader_id": newLeaderID}
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/teams/%d/transfer_leadership", id), body, &result)
	return &result, err
}

func (s *TeamService) InviteMember(ctx context.Context, teamID int64, username string) (*TeamMembership, error) {
	var result TeamMembership
	body := map[string]any{"username": username}
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/teams/%d/memberships/invite", teamID), body, &result)
	return &result, err
}

func (s *TeamService) AcceptInvite(ctx context.Context, teamID, membershipID int64) (*TeamMembership, error) {
	var result TeamMembership
	path := fmt.Sprintf("/api/teams/%d/memberships/%d/accept", teamID, membershipID)
	err := s.client.do(ctx, http.MethodPatch, path, nil, &result)
	return &result, err
}

func (s *TeamService) DeclineInvite(ctx context.Context, teamID, membershipID int64) (*TeamMembership, error) {
	var result TeamMembership
	path := fmt.Sprintf("/api/teams/%d/memberships/%d/decline", teamID, membershipID)
	err := s.client.do(ctx, http.MethodPatch, path, nil, &result)
	return &result, err
}

func (s *TeamService) UpdateMember(ctx context.Context, teamID, membershipID int64, role string) (*TeamMembership, error) {
	var result TeamMembership
	body := map[string]any{"role": role}
	path := fmt.Sprintf("/api/teams/%d/memberships/%d", teamID, membershipID)
	err := s.client.do(ctx, http.MethodPatch, path, body, &result)
	return &result, err
}

func (s *TeamService) RemoveMember(ctx context.Context, teamID, membershipID int64) error {
	path := fmt.Sprintf("/api/teams/%d/memberships/%d", teamID, membershipID)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil)
}

func (s *TeamService) Leave(ctx context.Context, teamID int64) error {
	path := fmt.Sprintf("/api/teams/%d/memberships/leave", teamID)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil)
}

func (s *TeamService) ListResources(ctx context.Context, teamID int64, params *ListParams) (*ListResult[TeamResource], error) {
	path := fmt.Sprintf("/api/teams/%d/resources", teamID) + buildQuery(params)
	return doList[TeamResource](s.client, ctx, path)
}

func (s *TeamService) ContributeResource(ctx context.Context, teamID int64, params *TeamResourceParams) (*TeamResource, error) {
	var result TeamResource
	path := fmt.Sprintf("/api/teams/%d/resources", teamID)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *TeamService) RemoveResource(ctx context.Context, teamID, resourceID int64) error {
	path := fmt.Sprintf("/api/teams/%d/resources/%d", teamID, resourceID)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil)
}
