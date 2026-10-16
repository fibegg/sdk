package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// teamClient adds an explicit expectation; it never changes server authority
// or discards a profile's existing owner proof.
func (c *Client) teamClient(teamID int64) (*Client, error) {
	if teamID <= 0 {
		return nil, fmt.Errorf("team ID must be positive")
	}
	forked := *c.cfg
	WithOwnerContext(OwnerContext{OwnerType: "Team", OwnerID: teamID})(&forked)
	return newClientFromConfig(&forked), nil
}
func positiveTeamResource(id int64) error {
	if id <= 0 {
		return fmt.Errorf("resource ID must be positive")
	}
	return nil
}
func (c *Client) TeamServicePrincipals(ctx context.Context, teamID int64) ([]TeamServicePrincipal, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []TeamServicePrincipal `json:"data"`
	}
	err = client.do(ctx, http.MethodGet, "/api/team_service_principals", nil, &result)
	return result.Data, err
}
func (c *Client) CreateTeamServicePrincipal(ctx context.Context, teamID int64, name string) (*TeamServicePrincipal, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("principal name is required")
	}
	var result TeamServicePrincipal
	err = client.do(ctx, http.MethodPost, "/api/team_service_principals", map[string]any{"service_principal": map[string]string{"name": name}}, &result)
	return &result, err
}
func (c *Client) RevokeTeamServicePrincipal(ctx context.Context, teamID, principalID int64) error {
	client, err := c.teamClient(teamID)
	if err != nil {
		return err
	}
	if err = positiveTeamResource(principalID); err != nil {
		return err
	}
	return client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/team_service_principals/%d", principalID), nil, nil)
}

// RecoverRuntimePrincipal requires a live human Team-admin credential.
// A non-nil empty selection recovers without rebinding agents; nil is invalid.
// An identical completed selection replays; a changed selection conflicts.
func (c *Client) RecoverRuntimePrincipal(ctx context.Context, teamID, principalID int64, agentIDs []int64) (*RuntimePrincipalRecovery, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	if err = positiveTeamResource(principalID); err != nil {
		return nil, err
	}
	if agentIDs == nil {
		return nil, fmt.Errorf("replacement_agent_ids must be an explicit list; use [] for no replacements")
	}
	seen := map[int64]bool{}
	for _, id := range agentIDs {
		if id <= 0 || seen[id] {
			return nil, fmt.Errorf("replacement_agent_ids must contain distinct positive Agent IDs")
		}
		seen[id] = true
	}
	var result RuntimePrincipalRecovery
	err = client.do(ctx, http.MethodPost, fmt.Sprintf("/api/team_service_principals/%d/recover", principalID), map[string]any{"replacement_agent_ids": agentIDs}, &result)
	return &result, err
}
func (c *Client) TeamSubscriptions(ctx context.Context, teamID int64) ([]TeamSubscription, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	var result []TeamSubscription
	err = client.do(ctx, http.MethodGet, "/api/team_billing/subscriptions", nil, &result)
	return result, err
}

// CancelTeamSubscription accepts HTTP 202 as a durable request receipt.
// retryFailed is an explicit retry of failed intent; read subscriptions for status.
func (c *Client) CancelTeamSubscription(ctx context.Context, teamID, subscriptionID int64, retryFailed bool) (*TeamCancellation, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	if err = positiveTeamResource(subscriptionID); err != nil {
		return nil, err
	}
	var result TeamCancellation
	err = client.do(ctx, http.MethodPost, "/api/team_billing/cancel_subscription", map[string]any{"subscription_id": subscriptionID, "retry_failed": retryFailed}, &result)
	return &result, err
}
func (c *Client) TeamRepositoryOperations(ctx context.Context, teamID int64) ([]TeamRepositoryOperation, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	var result []TeamRepositoryOperation
	err = client.do(ctx, http.MethodGet, "/api/team_repository_operations", nil, &result)
	return result, err
}
func validateTeamOperationRequestID(requestID string) error {
	if strings.TrimSpace(requestID) == "" || strings.ContainsAny(requestID, "/?#\\\x00") || requestID == "." || requestID == ".." {
		return fmt.Errorf("a single Team operation request ID is required")
	}
	return nil
}
func (c *Client) ReconcileTeamRepositoryOperation(ctx context.Context, teamID int64, requestID string, connectionID *int64) (*TeamRepositoryOperation, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	if err = validateTeamOperationRequestID(requestID); err != nil {
		return nil, err
	}
	if connectionID != nil {
		if err = positiveTeamResource(*connectionID); err != nil {
			return nil, err
		}
	}
	var result TeamRepositoryOperation
	err = client.do(ctx, http.MethodPost, fmt.Sprintf("/api/team_repository_operations/%s/reconcile", url.PathEscape(requestID)), struct {
		ConnectionID *int64 `json:"connection_id,omitempty"`
	}{connectionID}, &result)
	return &result, err
}
func (c *Client) AdoptTeamRepositoryOperation(ctx context.Context, teamID int64, requestID string, connectionID *int64) (*Repository, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	if err = validateTeamOperationRequestID(requestID); err != nil {
		return nil, err
	}
	if connectionID != nil {
		if err = positiveTeamResource(*connectionID); err != nil {
			return nil, err
		}
	}
	var result Repository
	err = client.do(ctx, http.MethodPost, fmt.Sprintf("/api/team_repository_operations/%s/adopt", url.PathEscape(requestID)), struct {
		ConnectionID *int64 `json:"connection_id,omitempty"`
	}{connectionID}, &result)
	return &result, err
}
func (c *Client) TeamGitInstallations(ctx context.Context, teamID int64) ([]TeamGitInstallation, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []TeamGitInstallation `json:"data"`
	}
	err = client.do(ctx, http.MethodGet, "/api/team_git_installations", nil, &result)
	return result.Data, err
}

// ConnectTeamGitInstallation verifies the existing human connection; no token is exported.
func (c *Client) ConnectTeamGitInstallation(ctx context.Context, teamID, installationID, verificationConnectionID int64) (*TeamGitInstallation, error) {
	client, err := c.teamClient(teamID)
	if err != nil {
		return nil, err
	}
	if err = positiveTeamResource(installationID); err != nil {
		return nil, err
	}
	if err = positiveTeamResource(verificationConnectionID); err != nil {
		return nil, err
	}
	var result TeamGitInstallation
	err = client.do(ctx, http.MethodPost, "/api/team_git_installations", map[string]any{"team_git_installation": map[string]int64{"installation_id": installationID, "verification_connection_id": verificationConnectionID}}, &result)
	return &result, err
}
func (c *Client) RemoveTeamGitInstallation(ctx context.Context, teamID, connectionID int64) error {
	client, err := c.teamClient(teamID)
	if err != nil {
		return err
	}
	if err = positiveTeamResource(connectionID); err != nil {
		return err
	}
	return client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/team_git_installations/%d", connectionID), nil, nil)
}
