package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// companyClient adds an explicit expectation; it never changes server authority
// or discards a profile's existing owner proof.
func (c *Client) companyClient(companyID int64) (*Client, error) {
	if companyID <= 0 {
		return nil, fmt.Errorf("company ID must be positive")
	}
	forked := *c.cfg
	WithOwnerContext(OwnerContext{OwnerType: "Team", OwnerID: companyID})(&forked)
	return newClientFromConfig(&forked), nil
}
func positiveCompanyResource(id int64) error {
	if id <= 0 {
		return fmt.Errorf("resource ID must be positive")
	}
	return nil
}
func (c *Client) TeamServicePrincipals(ctx context.Context, companyID int64) ([]TeamServicePrincipal, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []TeamServicePrincipal `json:"data"`
	}
	err = client.do(ctx, http.MethodGet, "/api/team_service_principals", nil, &result)
	return result.Data, err
}
func (c *Client) CreateTeamServicePrincipal(ctx context.Context, companyID int64, name string) (*TeamServicePrincipal, error) {
	client, err := c.companyClient(companyID)
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
func (c *Client) RevokeTeamServicePrincipal(ctx context.Context, companyID, principalID int64) error {
	client, err := c.companyClient(companyID)
	if err != nil {
		return err
	}
	if err = positiveCompanyResource(principalID); err != nil {
		return err
	}
	return client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/team_service_principals/%d", principalID), nil, nil)
}

// RecoverRuntimePrincipal requires a live human Company-admin credential.
// A non-nil empty selection recovers without rebinding agents; nil is invalid.
// An identical completed selection replays; a changed selection conflicts.
func (c *Client) RecoverRuntimePrincipal(ctx context.Context, companyID, principalID int64, agentIDs []int64) (*RuntimePrincipalRecovery, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	if err = positiveCompanyResource(principalID); err != nil {
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
func (c *Client) CompanySubscriptions(ctx context.Context, companyID int64) ([]CompanySubscription, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	var result []CompanySubscription
	err = client.do(ctx, http.MethodGet, "/api/company_billing/subscriptions", nil, &result)
	return result, err
}

// CancelCompanySubscription accepts HTTP 202 as a durable request receipt.
// retryFailed is an explicit retry of failed intent; read subscriptions for status.
func (c *Client) CancelCompanySubscription(ctx context.Context, companyID, subscriptionID int64, retryFailed bool) (*CompanyCancellation, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	if err = positiveCompanyResource(subscriptionID); err != nil {
		return nil, err
	}
	var result CompanyCancellation
	err = client.do(ctx, http.MethodPost, "/api/company_billing/cancel_subscription", map[string]any{"subscription_id": subscriptionID, "retry_failed": retryFailed}, &result)
	return &result, err
}
func (c *Client) CompanyRepositoryOperations(ctx context.Context, companyID int64) ([]CompanyRepositoryOperation, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	var result []CompanyRepositoryOperation
	err = client.do(ctx, http.MethodGet, "/api/company_repository_operations", nil, &result)
	return result, err
}
func validateCompanyOperationRequestID(requestID string) error {
	if strings.TrimSpace(requestID) == "" || strings.ContainsAny(requestID, "/?#\\\x00") || requestID == "." || requestID == ".." {
		return fmt.Errorf("a single Company operation request ID is required")
	}
	return nil
}
func (c *Client) ReconcileCompanyRepositoryOperation(ctx context.Context, companyID int64, requestID string, connectionID *int64) (*CompanyRepositoryOperation, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	if err = validateCompanyOperationRequestID(requestID); err != nil {
		return nil, err
	}
	if connectionID != nil {
		if err = positiveCompanyResource(*connectionID); err != nil {
			return nil, err
		}
	}
	var result CompanyRepositoryOperation
	err = client.do(ctx, http.MethodPost, fmt.Sprintf("/api/company_repository_operations/%s/reconcile", url.PathEscape(requestID)), struct {
		ConnectionID *int64 `json:"connection_id,omitempty"`
	}{connectionID}, &result)
	return &result, err
}
func (c *Client) AdoptCompanyRepositoryOperation(ctx context.Context, companyID int64, requestID string, connectionID *int64) (*Repository, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	if err = validateCompanyOperationRequestID(requestID); err != nil {
		return nil, err
	}
	if connectionID != nil {
		if err = positiveCompanyResource(*connectionID); err != nil {
			return nil, err
		}
	}
	var result Repository
	err = client.do(ctx, http.MethodPost, fmt.Sprintf("/api/company_repository_operations/%s/adopt", url.PathEscape(requestID)), struct {
		ConnectionID *int64 `json:"connection_id,omitempty"`
	}{connectionID}, &result)
	return &result, err
}
func (c *Client) CompanyGitInstallations(ctx context.Context, companyID int64) ([]CompanyGitInstallation, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data []CompanyGitInstallation `json:"data"`
	}
	err = client.do(ctx, http.MethodGet, "/api/company_git_installations", nil, &result)
	return result.Data, err
}

// ConnectCompanyGitInstallation verifies the existing human connection; no token is exported.
func (c *Client) ConnectCompanyGitInstallation(ctx context.Context, companyID, installationID, verificationConnectionID int64) (*CompanyGitInstallation, error) {
	client, err := c.companyClient(companyID)
	if err != nil {
		return nil, err
	}
	if err = positiveCompanyResource(installationID); err != nil {
		return nil, err
	}
	if err = positiveCompanyResource(verificationConnectionID); err != nil {
		return nil, err
	}
	var result CompanyGitInstallation
	err = client.do(ctx, http.MethodPost, "/api/company_git_installations", map[string]any{"company_git_installation": map[string]int64{"installation_id": installationID, "verification_connection_id": verificationConnectionID}}, &result)
	return &result, err
}
func (c *Client) RemoveCompanyGitInstallation(ctx context.Context, companyID, connectionID int64) error {
	client, err := c.companyClient(companyID)
	if err != nil {
		return err
	}
	if err = positiveCompanyResource(connectionID); err != nil {
		return err
	}
	return client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/company_git_installations/%d", connectionID), nil, nil)
}
