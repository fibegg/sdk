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

func TestPlaygrounds_RolloutWithParams(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/playgrounds/42/rollout" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		json.NewEncoder(w).Encode(Playground{ID: 42, Status: "pending"})
	})

	force := true
	_, err := c.Playgrounds.RolloutWithParams(context.Background(), 42, &PlaygroundRolloutParams{Force: &force})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["force"] != true {
		t.Fatalf("expected force=true, got %#v", body)
	}
}

func TestPlaygrounds_DebugWithParams(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/playgrounds/42/debug" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("mode") != "summary" || query.Get("refresh") != "true" || query.Get("service") != "web" || query.Get("logs_tail") != "25" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	refresh := true
	result, err := c.Playgrounds.DebugWithParams(context.Background(), 42, &PlaygroundDebugParams{
		Mode:     "summary",
		Refresh:  &refresh,
		Service:  "web",
		LogsTail: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("unexpected result: %#v", result)
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

func TestAgents_StartChat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/start_chat" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["marquee_id"] != float64(9) {
			t.Errorf("expected marquee_id 9, got %v", body["marquee_id"])
		}
		json.NewEncoder(w).Encode(AgentChatSession{ID: 123, Status: "starting"})
	})

	session, err := c.Agents.StartChat(context.Background(), 5, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 123 || session.Status != "starting" {
		t.Errorf("unexpected session: %#v", session)
	}
}

func TestAgents_RuntimeStatus(t *testing.T) {
	chatURL := "https://agent.example.test"
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/agents/5/runtime_status" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(AgentRuntimeStatus{
			ID:               123,
			Status:           "running",
			ChatURL:          &chatURL,
			RuntimeReachable: true,
			Authenticated:    true,
			IsProcessing:     false,
			QueueCount:       0,
		})
	})

	status, err := c.Agents.RuntimeStatus(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ID != 123 || status.Status != "running" || !status.RuntimeReachable || status.IsProcessing || status.QueueCount != 0 {
		t.Errorf("unexpected status: %#v", status)
	}
	if status.ChatURL == nil || *status.ChatURL != chatURL {
		t.Errorf("unexpected chat URL: %#v", status.ChatURL)
	}
}

func TestAgents_PurgeChat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/purge_chat" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(AgentChatSession{ID: 123, Status: "stopped"})
	})

	session, err := c.Agents.PurgeChat(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 123 || session.Status != "stopped" {
		t.Errorf("unexpected session: %#v", session)
	}
}

func TestAgents_CreateProviderAPIKeyModeJSON(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		agent := body["agent"]
		if agent["provider_api_key_mode"] != true {
			t.Errorf("expected provider_api_key_mode=true bool, got %#v", agent["provider_api_key_mode"])
		}
		if agent["model_options"] != "flash-lite" {
			t.Errorf("expected model_options flash-lite, got %#v", agent["model_options"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Agent{ID: 99, Name: "new-agent", Provider: ProviderGemini})
	})

	providerAPIKeyMode := true
	modelOptions := "flash-lite"
	agent, err := c.Agents.Create(context.Background(), &AgentCreateParams{
		Name:               "new-agent",
		Provider:           ProviderGemini,
		ProviderAPIKeyMode: &providerAPIKeyMode,
		ModelOptions:       &modelOptions,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != 99 {
		t.Errorf("expected ID 99, got %d", agent.ID)
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

func TestSecrets_GetReveal(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/secrets/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("reveal") != "true" {
			t.Errorf("expected reveal=true, got %q", r.URL.Query().Get("reveal"))
		}
		id := int64(42)
		value := "secret"
		json.NewEncoder(w).Encode(Secret{ID: &id, Key: "DB_URL", Value: &value})
	})

	secret, err := c.Secrets.Get(context.Background(), 42, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret.Value == nil || *secret.Value != "secret" {
		t.Errorf("expected revealed value, got %#v", secret.Value)
	}
}

func TestJobEnv_GetReveal(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/job_env/7" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("reveal") != "true" {
			t.Errorf("expected reveal=true, got %q", r.URL.Query().Get("reveal"))
		}
		id := int64(7)
		value := "env-secret"
		json.NewEncoder(w).Encode(JobEnvEntry{ID: &id, Key: "TOKEN", Value: &value, Secret: true})
	})

	entry, err := c.JobEnv.Get(context.Background(), 7, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Value == nil || *entry.Value != "env-secret" {
		t.Errorf("expected revealed value, got %#v", entry.Value)
	}
}

func TestImportTemplates_SetSourceCIFields(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/api/import_templates/11/source" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		source := body["source"]
		if source["ci_enabled"] != true {
			t.Errorf("expected ci_enabled=true, got %#v", source["ci_enabled"])
		}
		if source["ci_marquee_id"] != float64(22) {
			t.Errorf("expected ci_marquee_id=22, got %#v", source["ci_marquee_id"])
		}
		id := int64(11)
		json.NewEncoder(w).Encode(ImportTemplate{ID: &id})
	})

	ciEnabled := true
	ciMarqueeID := int64(22)
	_, err := c.ImportTemplates.SetSource(context.Background(), 11, &ImportTemplateSourceParams{
		SourcePropID: 1,
		SourcePath:   "fibe-ci.yml",
		CIEnabled:    &ciEnabled,
		CIMarqueeID:  &ciMarqueeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
