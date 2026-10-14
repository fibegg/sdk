package fibe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTeamOperationsTransportAndPublicReceipts(t *testing.T) {
	cases := []struct {
		name, method, path, request, response string
		status                                int
		call                                  func(*Client) (any, error)
	}{
		{"principals", "GET", "/api/team_service_principals", "", `{"data":[{"id":2,"owner_type":"Team","owner_id":9,"name":"runtime"}]}`, 200, func(c *Client) (any, error) { return c.TeamServicePrincipals(context.Background(), 9) }},
		{"principal-create", "POST", "/api/team_service_principals", `{"service_principal":{"name":"service"}}`, `{"id":3,"name":"service","owner_type":"Team","owner_id":9}`, 201, func(c *Client) (any, error) { return c.CreateTeamServicePrincipal(context.Background(), 9, "service") }},
		{"principal-revoke", "DELETE", "/api/team_service_principals/2", "", "", 204, func(c *Client) (any, error) { return nil, c.RevokeTeamServicePrincipal(context.Background(), 9, 2) }},
		{"recovery", "POST", "/api/team_service_principals/2/recover", `{"replacement_agent_ids":[]}`, `{"principal_id":3,"supersedes_id":2,"replacement_agent_ids":[],"recovery_status":"completed","status":201,"adoption":[{"agent_id":5,"status":"redeploy_pending","key_id":77,"previous_key_id":66,"token":"synthetic-secret"}],"original_authority":{"api_key_id":99}}`, 201, func(c *Client) (any, error) { return c.RecoverRuntimePrincipal(context.Background(), 9, 2, []int64{}) }},
		{"subscriptions", "GET", "/api/team_billing/subscriptions", "", `[{"id":4,"owner_type":"Team","owner_id":9,"status":"active","cancellation_state":"failed","cancellation_error_code":"AUTHORITY_LOST"}]`, 200, func(c *Client) (any, error) { return c.TeamSubscriptions(context.Background(), 9) }},
		{"cancel", "POST", "/api/team_billing/cancel_subscription", `{"subscription_id":4,"retry_failed":true}`, `{"id":4,"subscription_id":4,"owner_type":"Team","owner_id":9,"status":"pending","cancellation_state":"pending","cancellation_request_id":"cancel-exact"}`, 202, func(c *Client) (any, error) { return c.CancelTeamSubscription(context.Background(), 9, 4, true) }},
		{"operations", "GET", "/api/team_repository_operations", "", `[{"team_repository_operation_id":8,"operation_request_id":"exact-id","operation_state":"unknown","outcome_unknown":true,"owner_type":"Team","owner_id":9}]`, 200, func(c *Client) (any, error) { return c.TeamRepositoryOperations(context.Background(), 9) }},
		{"reconcile", "POST", "/api/team_repository_operations/exact-id/reconcile", `{"connection_id":6}`, `{"team_repository_operation_id":8,"operation_request_id":"exact-id","operation_state":"remote_created","destination":{"id":100,"full_name":"org/repo","html_url":"https://github.com/org/repo"}}`, 200, func(c *Client) (any, error) {
			id := int64(6)
			return c.ReconcileTeamRepositoryOperation(context.Background(), 9, "exact-id", &id)
		}},
		{"adopt", "POST", "/api/team_repository_operations/exact-id/adopt", `{}`, `{"id":18,"owner_type":"Team","owner_id":9,"repository_url":"https://github.com/org/repo"}`, 201, func(c *Client) (any, error) {
			return c.AdoptTeamRepositoryOperation(context.Background(), 9, "exact-id", nil)
		}},
		{"installations", "GET", "/api/team_git_installations", "", `{"data":[{"id":6,"installation_id":123,"organization":"org","owner_type":"Team","owner_id":9,"verified_authority":{"installation_id":"123","organization_id":"789","app_id":"456","organization_login":"org","permissions":{"contents":"write"}},"authority_status":{"available":false,"error_code":"TEAM_INSTALLATION_AUTHORITY_UNAVAILABLE"},"token":"synthetic-secret"}]}`, 200, func(c *Client) (any, error) { return c.TeamGitInstallations(context.Background(), 9) }},
		{"connect", "POST", "/api/team_git_installations", `{"team_git_installation":{"installation_id":123,"verification_connection_id":7}}`, `{"id":6,"installation_id":123,"organization":"org"}`, 201, func(c *Client) (any, error) { return c.ConnectTeamGitInstallation(context.Background(), 9, 123, 7) }},
		{"remove", "DELETE", "/api/team_git_installations/6", "", "", 204, func(c *Client) (any, error) { return nil, c.RemoveTeamGitInstallation(context.Background(), 9, 6) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hits := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("X-Fibe-Expected-Owner-Type") != "Team" || r.Header.Get("X-Fibe-Expected-Owner-ID") != "9" {
					t.Error("missing explicit Team proof")
				}
				if tc.request != "" {
					var actual, expected any
					if err := json.NewDecoder(r.Body).Decode(&actual); err != nil {
						t.Error(err)
					}
					_ = json.Unmarshal([]byte(tc.request), &expected)
					a, _ := json.Marshal(actual)
					b, _ := json.Marshal(expected)
					if string(a) != string(b) {
						t.Errorf("request payload %s want %s", a, b)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Fibe-Owner-Type", "Team")
				w.Header().Set("X-Fibe-Owner-ID", "9")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			client := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL), WithAPIKey("fixture"), WithMaxRetries(0))
			result, err := tc.call(client)
			if err != nil {
				t.Fatal(err)
			}
			if hits != 1 {
				t.Fatalf("hits %d", hits)
			}
			encoded, _ := json.Marshal(result)
			for _, private := range []string{"key_id", "previous_key_id", "api_key_id", "synthetic-secret", "original_authority"} {
				if strings.Contains(string(encoded), private) {
					t.Fatalf("private field escaped typed receipt: %s", private)
				}
			}
		})
	}
}

func TestTeamRecoveryExplicitSelectionsAndReplayErrors(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Fibe-Owner-Type", "Team")
		w.Header().Set("X-Fibe-Owner-ID", "9")
		if hits == 3 {
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"error":{"code":"RECOVERY_CONFLICT","message":"Different selection"}}`))
			return
		}
		if hits == 1 {
			w.WriteHeader(201)
		}
		_, _ = w.Write([]byte(`{"principal_id":3,"supersedes_id":2,"replacement_agent_ids":[],"recovery_status":"completed","status":200,"adoption":[]}`))
	}))
	defer server.Close()
	c := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL), WithAPIKey("fixture"), WithMaxRetries(0))
	for _, invalid := range [][]int64{nil, {0}, {-1}, {5, 5}} {
		if _, err := c.RecoverRuntimePrincipal(context.Background(), 9, 2, invalid); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
	if hits != 0 {
		t.Fatal("invalid selection reached server")
	}
	first, err := c.RecoverRuntimePrincipal(context.Background(), 9, 2, []int64{})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := c.RecoverRuntimePrincipal(context.Background(), 9, 2, []int64{})
	if err != nil || replay.PrincipalID != first.PrincipalID {
		t.Fatal("completed replay changed", err)
	}
	_, err = c.RecoverRuntimePrincipal(context.Background(), 9, 2, []int64{5})
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != 409 || apiError.Code != "RECOVERY_CONFLICT" {
		t.Fatalf("conflict lost: %v", err)
	}
	if _, err = c.TeamSubscriptions(context.Background(), 0); err == nil {
		t.Fatal("invalid Team accepted")
	}
	if _, err = c.ReconcileTeamRepositoryOperation(context.Background(), 9, "../another", nil); err == nil {
		t.Fatal("operation traversal accepted")
	}
	if hits != 3 {
		t.Fatalf("unexpected requests %d", hits)
	}
}

func TestTeamOperationsRejectForeignResponseOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Fibe-Owner-Type", "Team")
		w.Header().Set("X-Fibe-Owner-ID", "10")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	c := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL), WithAPIKey("fixture"), WithMaxRetries(0))
	if _, err := c.TeamSubscriptions(context.Background(), 9); err == nil {
		t.Fatal("foreign response owner accepted")
	}
}

func TestTeamOperationsRetainExistingProfileOwnerProof(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Fibe-Expected-Owner-ID") != "9" {
			t.Error("explicit Team expectation lost")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Fibe-Owner-Type", "Team")
		w.Header().Set("X-Fibe-Owner-ID", "9")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL), WithAPIKey("fixture"), WithMaxRetries(0),
		WithOwnerContext(OwnerContext{OwnerType: "Team", OwnerID: 10}))
	if _, err := client.TeamSubscriptions(context.Background(), 9); err == nil {
		t.Fatal("explicit Team expectation discarded the conflicting profile proof")
	}
}
