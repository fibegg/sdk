package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func listEnv[T any](items []T) listEnvelope[T] {
	return listEnvelope[T]{
		Data: items,
		Meta: ListMeta{Page: 1, PerPage: 25, Total: int64(len(items))},
	}
}

func TestPlaygrounds_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/playgrounds" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(listEnv([]Playground{
			{ID: 1, Name: "pg-1", Status: "running"},
			{ID: 2, Name: "pg-2", Status: "pending"},
		}))
	})

	result, err := c.Playgrounds.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 playgrounds, got %d", len(result.Data))
	}
	if result.Data[0].Name != "pg-1" {
		t.Errorf("expected name 'pg-1', got %q", result.Data[0].Name)
	}
	if result.Meta.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Meta.Total)
	}
}

func TestPlaygrounds_Get(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playgrounds/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Playground{ID: 42, Name: "test", Status: "running"})
	})

	pg, err := c.Playgrounds.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 42 {
		t.Errorf("expected ID 42, got %d", pg.ID)
	}
}

func TestPlaygrounds_Create(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		pg := body["playground"].(map[string]any)
		if pg["name"] != "new-pg" {
			t.Errorf("expected name 'new-pg', got %v", pg["name"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(Playground{ID: 99, Name: "new-pg", Status: "pending"})
	})

	pg, err := c.Playgrounds.Create(context.Background(), &PlaygroundCreateParams{
		Name:       "new-pg",
		PlayspecID: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 99 {
		t.Errorf("expected ID 99, got %d", pg.ID)
	}
}

func TestPlaygrounds_Rollout(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/playgrounds/42/rollout" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(Playground{ID: 42, Status: "pending"})
	})

	pg, err := c.Playgrounds.Rollout(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", pg.Status)
	}
}

func TestPlaygrounds_Logs(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playgrounds/42/logs/web" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("tail") != "100" {
			t.Errorf("expected tail=100, got %q", r.URL.Query().Get("tail"))
		}
		json.NewEncoder(w).Encode(PlaygroundLogs{
			Service: "web",
			Lines:   []string{"line1", "line2"},
			Source:  "live",
		})
	})

	tail := 100
	logs, err := c.Playgrounds.Logs(context.Background(), 42, "web", &tail)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(logs.Lines))
	}
}

func TestAgents_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Agent{
			{ID: 1, Name: "agent-1", Provider: "github", Authenticated: true},
		}))
	})

	result, err := c.Agents.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 agent, got %d", len(result.Data))
	}
	if !result.Data[0].Authenticated {
		t.Error("expected authenticated=true")
	}
}

func TestAgents_Chat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/chat" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["text"] != "hello" {
			t.Errorf("expected text 'hello', got %v", body["text"])
		}
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
	})

	result, err := c.Agents.Chat(context.Background(), 5, &AgentChatParams{Text: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got %v", result["status"])
	}
}

func TestSecrets_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Secret{{Key: "DB_URL"}}))
	})

	result, err := c.Secrets.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 secret, got %d", len(result.Data))
	}
}

func TestAPIKeys_Me(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Player{ID: 1, Username: "testuser"})
	})

	player, err := c.APIKeys.Me(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", player.Username)
	}
}

func TestTeams_CRUD(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/teams", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Team{{ID: 1, Name: "team-1", MembersCount: 3}}))
	})
	mux.HandleFunc("POST /api/teams", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(Team{ID: 2, Name: "new-team"})
	})
	mux.HandleFunc("DELETE /api/teams/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := NewClient(WithAPIKey("test"), WithBaseURL(srv.URL), WithMaxRetries(0))

	teams, err := c.Teams.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(teams.Data) != 1 {
		t.Errorf("expected 1 team, got %d", len(teams.Data))
	}

	team, err := c.Teams.Create(context.Background(), &TeamCreateParams{Name: "new-team"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if team.Name != "new-team" {
		t.Errorf("expected name 'new-team', got %q", team.Name)
	}

	err = c.Teams.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestWebhookEndpoints_EventTypes(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/webhook_endpoints/event_types" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"event_types": []string{"playground.created", "playground.status.changed"},
		})
	})

	types, err := c.WebhookEndpoints.EventTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) != 2 {
		t.Errorf("expected 2 event types, got %d", len(types))
	}
}

func TestProps_Sync(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/props/7/sync" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"message": "Sync scheduled"})
	})

	err := c.Props.Sync(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildQuery(t *testing.T) {
	params := &ArtefactListParams{
		Query:   "test",
		Sort:    "created_at_asc",
		Page:    1,
		PerPage: 25,
	}

	q := buildQuery(params)
	if q == "" {
		t.Error("expected non-empty query string")
	}
	if q[0] != '?' {
		t.Error("expected query to start with '?'")
	}
}

func TestBuildQuery_NilParams(t *testing.T) {
	q := buildQuery(nil)
	if q != "" {
		t.Errorf("expected empty query for nil params, got %q", q)
	}
}

func TestBuildQuery_EmptyParams(t *testing.T) {
	params := &ArtefactListParams{}
	q := buildQuery(params)
	if q != "" {
		t.Errorf("expected empty query for zero-value params, got %q", q)
	}
}

func TestStatus_Get_WithLimitsSections(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/status" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		limit1000 := 1000
		limit10 := 10
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"playgrounds":  map[string]any{"total": 2, "active": 1, "stopped": 1},
			"agents":       map[string]any{"total": 3, "authenticated": 2},
			"props":        5,
			"playspecs":    4,
			"marquees":     1,
			"secrets":      0,
			"teams":        0,
			"api_keys":     2,
			"subscription": map[string]any{"plan": "single", "playground_limit": 1000},
			"resource_quotas": map[string]any{
				"playgrounds": map[string]any{"used": 2, "limit": limit1000, "status": "ok"},
				"agents":      map[string]any{"used": 3, "limit": limit10, "status": "ok"},
			},
			"per_parent_caps": map[string]any{
				"mounted_files_per_agent": 5,
				"artefacts_per_agent":     100,
			},
			"rate_limits": map[string]any{
				"api": map[string]any{"limit": 5000, "remaining": 4987, "reset_seconds": 1234},
			},
		})
	})

	status, err := c.Status.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ResourceQuotas == nil || status.ResourceQuotas["playgrounds"].Used != 2 {
		t.Errorf("expected resource_quotas.playgrounds.used=2, got %+v", status.ResourceQuotas)
	}
	if status.ResourceQuotas["playgrounds"].Limit == nil || *status.ResourceQuotas["playgrounds"].Limit != 1000 {
		t.Errorf("expected playgrounds limit 1000, got %+v", status.ResourceQuotas["playgrounds"].Limit)
	}
	if status.PerParentCaps["mounted_files_per_agent"] == nil || *status.PerParentCaps["mounted_files_per_agent"] != 5 {
		t.Errorf("expected per_parent_caps.mounted_files_per_agent=5, got %+v", status.PerParentCaps)
	}
	if status.RateLimits == nil || status.RateLimits.API == nil {
		t.Fatalf("expected rate_limits.api section")
	}
	if status.RateLimits.API.Limit != 5000 || status.RateLimits.API.Remaining != 4987 {
		t.Errorf("unexpected rate limit values: %+v", status.RateLimits.API)
	}
}

func TestStatus_Get_WithoutLimitsSections(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"playgrounds":  map[string]any{"total": 0, "active": 0, "stopped": 0},
			"agents":       map[string]any{"total": 0, "authenticated": 0},
			"props":        0,
			"playspecs":    0,
			"marquees":     0,
			"secrets":      0,
			"teams":        0,
			"api_keys":     0,
			"subscription": map[string]any{"plan": "free", "playground_limit": 1000},
		})
	})

	status, err := c.Status.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ResourceQuotas != nil {
		t.Errorf("expected nil resource_quotas when omitted, got %+v", status.ResourceQuotas)
	}
	if status.PerParentCaps != nil {
		t.Errorf("expected nil per_parent_caps when omitted, got %+v", status.PerParentCaps)
	}
	if status.RateLimits != nil {
		t.Errorf("expected nil rate_limits when omitted, got %+v", status.RateLimits)
	}
}
