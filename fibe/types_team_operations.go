package fibe

import "time"

// RuntimePrincipalRecovery deliberately exposes no runtime credential IDs or bytes.
type RuntimePrincipalRecovery struct {
	PrincipalID         int64                      `json:"principal_id"`
	SupersedesID        int64                      `json:"supersedes_id"`
	ReplacementAgentIDs []int64                    `json:"replacement_agent_ids"`
	RecoveryStatus      string                     `json:"recovery_status"`
	Status              int                        `json:"status"`
	Adoption            []RuntimePrincipalAdoption `json:"adoption"`
}
type RuntimePrincipalAdoption struct {
	AgentID          int64      `json:"agent_id"`
	Status           string     `json:"status"`
	ChatID           *int64     `json:"chat_id,omitempty"`
	HostID           *int64     `json:"host_id,omitempty"`
	ChatStartedAt    *time.Time `json:"chat_started_at,omitempty"`
	RedeployEligible *bool      `json:"redeploy_eligible,omitempty"`
}
type TeamServicePrincipal struct {
	OwnershipMetadata
	ID        int64      `json:"id"`
	Subject   string     `json:"subject"`
	Name      string     `json:"name"`
	RevokedAt *time.Time `json:"revoked_at"`
}
type TeamSubscription struct {
	OwnerContext
	ID                       int64      `json:"id"`
	PlanType                 string     `json:"plan_type"`
	BillingInterval          string     `json:"billing_interval"`
	Status                   string     `json:"status"`
	BillingProvider          string     `json:"billing_provider"`
	CurrentPeriodStart       *time.Time `json:"current_period_start"`
	CurrentPeriodEnd         *time.Time `json:"current_period_end"`
	CanceledAt               *time.Time `json:"canceled_at"`
	CancellationState        string     `json:"cancellation_state"`
	CancellationRequestID    *string    `json:"cancellation_request_id"`
	CancellationRequestedAt  *time.Time `json:"cancellation_requested_at"`
	CancellationReconciledAt *time.Time `json:"cancellation_reconciled_at"`
	CancellationErrorCode    *string    `json:"cancellation_error_code"`
}

// TeamCancellation is a durable intent receipt, not proof of provider cancellation.
type TeamCancellation struct {
	OwnerContext
	ID                       int64      `json:"id"`
	SubscriptionID           int64      `json:"subscription_id"`
	Status                   string     `json:"status"`
	CancellationState        string     `json:"cancellation_state"`
	CancellationRequestID    *string    `json:"cancellation_request_id"`
	CancellationRequestedAt  *time.Time `json:"cancellation_requested_at"`
	CancellationReconciledAt *time.Time `json:"cancellation_reconciled_at"`
	CancellationErrorCode    *string    `json:"cancellation_error_code"`
}
type TeamRepositoryOperation struct {
	OwnerContext
	ID                int64                      `json:"team_repository_operation_id"`
	RequestID         string                     `json:"operation_request_id"`
	RequestedFullName string                     `json:"requested_full_name"`
	State             string                     `json:"operation_state"`
	OutcomeUnknown    bool                       `json:"outcome_unknown"`
	Destination       *TeamRepositoryDestination `json:"destination"`
	RecoveryAction    string                     `json:"recovery_action"`
	ErrorCode         *string                    `json:"error_code"`
	RequestedAt       *time.Time                 `json:"requested_at"`
	ReconciledAt      *time.Time                 `json:"reconciled_at"`
	LocalRepositoryID *int64                     `json:"local_repository_id"`
}
type TeamRepositoryDestination struct {
	ID            int64  `json:"id"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}
type TeamGitInstallation struct {
	OwnershipMetadata
	ID                int64                  `json:"id"`
	Provider          string                 `json:"provider"`
	InstallationID    int64                  `json:"installation_id"`
	Organization      string                 `json:"organization"`
	VerifiedAuthority TeamGitAuthority       `json:"verified_authority"`
	AuthorityStatus   TeamGitAuthorityStatus `json:"authority_status"`
}
type TeamGitAuthority struct {
	InstallationID      string            `json:"installation_id"`
	OrganizationID      string            `json:"organization_id"`
	OrganizationLogin   string            `json:"organization_login"`
	AppID               string            `json:"app_id"`
	Permissions         map[string]string `json:"permissions"`
	RepositorySelection string            `json:"repository_selection"`
}
type TeamGitAuthorityStatus struct {
	Available           bool   `json:"available"`
	RepositorySelection string `json:"repository_selection,omitempty"`
	ErrorCode           string `json:"error_code,omitempty"`
}
