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
		Query:  "test",
		Sort:   "created_at_asc",
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
