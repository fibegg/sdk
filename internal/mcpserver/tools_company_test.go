package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompanyMCPExplicitRecoveryAndCancellationReceipts(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("X-Fibe-Expected-Owner-ID") != "9" || r.Header.Get("X-Fibe-Expected-Owner-Type") != "Team" {
			t.Error("owner expectation lost")
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
				t.Error("explicit empty selection lost")
			}
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"principal_id":3,"replacement_agent_ids":[],"adoption":[{"agent_id":5,"status":"adopted","key_id":99}]}`))
		case "/api/company_billing/cancel_subscription":
			var p struct {
				Retry bool `json:"retry_failed"`
			}
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				t.Error(err)
			}
			if !p.Retry {
				t.Error("explicit retry lost")
			}
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`{"subscription_id":4,"cancellation_state":"pending","status":"pending"}`))
		case "/api/company_repository_operations/exact/reconcile":
			_, _ = w.Write([]byte(`{"operation_request_id":"exact","operation_state":"remote_created"}`))
		case "/api/company_git_installations":
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	srv := New(Config{APIKey: "fixture", Domain: server.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	for _, args := range []map[string]any{
		{"company_id": 9, "principal_id": 2, "replacement_agent_ids": []any{}},
		{"company_id": 9, "principal_id": 2, "replacement_agent_ids": nil, "confirm": true},
		{"company_id": 9, "principal_id": 2, "replacement_agent_ids": []any{1, 1}, "confirm": true},
		{"company_id": 9, "principal_id": 2, "replacement_agent_ids": []any{"1"}, "confirm": true},
		{"company_id": 9, "principal_id": 2, "replacement_agent_ids": []any{}, "token": "must-not-send", "confirm": true},
	} {
		if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_company_runtime_principal_recover", args); err == nil {
			t.Fatal("unconfirmed/invalid recovery accepted")
		}
	}
	if hits != 0 {
		t.Fatal("invalid recovery reached server")
	}
	result, err := srv.dispatcher.dispatch(context.Background(), "fibe_company_runtime_principal_recover", map[string]any{"company_id": 9, "principal_id": 2, "replacement_agent_ids": []any{}, "confirm": true})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "key_id") {
		t.Fatal("runtime key exposed")
	}
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"fibe_company_subscription_cancel", map[string]any{"company_id": 9, "subscription_id": 4, "retry_failed": true, "confirm": true}},
		{"fibe_company_repository_operation_reconcile", map[string]any{"company_id": 9, "request_id": "exact"}},
		{"fibe_company_git_installations_list", map[string]any{"company_id": 9}},
	} {
		if _, err := srv.dispatcher.dispatch(context.Background(), tc.name, tc.args); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 4 {
		t.Fatalf("hits %d", hits)
	}
}

func TestCompanyMCPDiscoveryAndConfirmation(t *testing.T) {
	srv := New(Config{APIKey: "fixture", Domain: "https://fibe.gg", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"fibe_company_principals_list", "fibe_company_principal_create", "fibe_company_principal_revoke", "fibe_company_runtime_principal_recover", "fibe_company_subscriptions_list", "fibe_company_subscription_cancel", "fibe_company_repository_operations_list", "fibe_company_repository_operation_reconcile", "fibe_company_repository_operation_adopt", "fibe_company_git_installations_list", "fibe_company_git_installation_connect", "fibe_company_git_installation_remove"} {
		if _, ok := srv.dispatcher.lookup(name); !ok {
			t.Fatalf("missing tool %s", name)
		}
	}
	for _, name := range []string{"fibe_company_principal_revoke", "fibe_company_runtime_principal_recover", "fibe_company_subscription_cancel", "fibe_company_repository_operation_adopt", "fibe_company_git_installation_remove"} {
		tool, _ := srv.dispatcher.lookup(name)
		if !tool.annotations.Destructive {
			t.Fatalf("confirmation not required: %s", name)
		}
	}
}
