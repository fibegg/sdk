package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompanyCLIExplicitOwnerRecoveryCancellationAndRepositoryReceipt(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("X-Fibe-Expected-Owner-ID") != "9" || r.Header.Get("X-Fibe-Expected-Owner-Type") != "Team" {
			t.Error("Company expectation lost")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Fibe-Owner-Type", "Team")
		w.Header().Set("X-Fibe-Owner-ID", "9")
		switch r.URL.Path {
		case "/api/team_service_principals/2/recover":
			var p struct {
				IDs []int64 `json:"replacement_agent_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				t.Error(err)
			}
			if p.IDs == nil || len(p.IDs) != 0 {
				t.Error("explicit [] lost")
			}
			_, _ = w.Write([]byte(`{"principal_id":3,"replacement_agent_ids":[],"adoption":[]}`))
		case "/api/company_billing/cancel_subscription":
			var p struct {
				ID    int64 `json:"subscription_id"`
				Retry bool  `json:"retry_failed"`
			}
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				t.Error(err)
			}
			if p.ID != 4 || !p.Retry {
				t.Error("retry intent lost")
			}
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`{"subscription_id":4,"status":"pending","cancellation_state":"pending"}`))
		case "/api/company_repository_operations/exact/adopt":
			_, _ = w.Write([]byte(`{"id":18,"owner_type":"Team","owner_id":9,"repository_url":"https://github.com/org/repo"}`))
		case "/api/company_git_installations":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("FIBE_DOMAIN", server.URL)
	t.Setenv("FIBE_API_KEY", "fixture")
	for _, args := range [][]string{
		{"company", "--company-id", "9", "principals", "recover", "2", "--agent-ids", "[]"},
		{"company", "--company-id", "9", "subscriptions", "cancel", "4", "--retry-failed"},
		{"company", "--company-id", "9", "repository-operations", "adopt", "exact"},
		{"company", "--company-id", "9", "git-installations", "list"},
	} {
		command := RootCmd()
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	if len(requests) != 4 {
		t.Fatalf("requests %v", requests)
	}
	for _, args := range [][]string{
		{"company", "principals", "list"},
		{"company", "--company-id", "9", "principals", "recover", "2"},
		{"company", "--company-id", "9", "principals", "recover", "2", "--agent-ids", "null"},
		{"company", "--company-id", "9", "principals", "recover", "2", "--agent-ids", "[1,1]"},
		{"company", "--company-id", "9", "principals", "recover", "2", "--agent-ids", "[1.5]"},
	} {
		command := RootCmd()
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Fatalf("invalid command accepted: %v", args)
		}
	}
	if len(requests) != 4 {
		t.Fatal("invalid command reached server")
	}
}

func TestCompanyCLICommandsRegistered(t *testing.T) {
	for _, path := range [][]string{
		{"principals", "list"}, {"principals", "create"}, {"principals", "revoke"}, {"principals", "recover"},
		{"subscriptions", "list"}, {"subscriptions", "cancel"},
		{"repository-operations", "list"}, {"repository-operations", "reconcile"}, {"repository-operations", "adopt"},
		{"git-installations", "list"}, {"git-installations", "connect"}, {"git-installations", "remove"},
	} {
		found, _, err := RootCmd().Find(append([]string{"company"}, path...))
		if err != nil || found.Name() != path[1] {
			t.Fatalf("missing Company command %v", path)
		}
	}
}
