package fibe

import (
	"context"
	"encoding/json"
	"net/http"
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

func TestGreenfield_Create(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/greenfield" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(GreenfieldResult{
			Name:        "tower-defence",
			GitProvider: "gitea",
			Playground:  &Playground{ID: 77, Name: "tower-defence", Status: "pending"},
		})
	})

	marqueeID := int64(12)
	result, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{
		Name:         "tower-defence",
		TemplateBody: "services:\n  web:\n    image: nginx\n",
		GitProvider:  "github",
		MarqueeID:    &marqueeID,
		Variables:    map[string]any{"app_name": "Tower"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Playground == nil || result.Playground.ID != 77 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if body["name"] != "tower-defence" || body["git_provider"] != "github" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if body["marquee_id"].(float64) != 12 {
		t.Fatalf("unexpected marquee id in body: %#v", body)
	}
	if body["template_body"] != "services:\n  web:\n    image: nginx\n" {
		t.Fatalf("unexpected template body in body: %#v", body)
	}
	vars := body["variables"].(map[string]any)
	if vars["app_name"] != "Tower" {
		t.Fatalf("unexpected variables: %#v", vars)
	}
}

func TestGreenfield_CreateRejectsVersionWithoutTemplateID(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called")
	})

	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{Name: "todo", Version: "v1"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGreenfield_CreateRejectsTemplateBodyWithTemplateID(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called")
	})

	templateID := int64(347)
	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{Name: "todo", TemplateID: &templateID, TemplateBody: "services: {}\n"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGreenfield_CreateWithTemplateID(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/greenfield" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(GreenfieldResult{Name: "todo", Playground: &Playground{ID: 77}})
	})

	marqueeID := int64(12)
	templateID := int64(347)
	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{
		Name:        "todo",
		TemplateID:  &templateID,
		Version:     "v1",
		GitProvider: "gitea",
		MarqueeID:   &marqueeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["template_id"].(float64) != 347 || body["version"] != "v1" {
		t.Fatalf("unexpected template fields: %#v", body)
	}
}

func TestGiteaRepos_CreateSurfacesProp(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/gitea_repos" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             123,
			"name":           "bagg-app",
			"full_name":      "viktorvsk/bagg-app",
			"html_url":       "https://git-next.fibe.live/viktorvsk/bagg-app",
			"clone_url":      "https://git-next.fibe.live/viktorvsk/bagg-app.git",
			"default_branch": "main",
			"repo": map[string]any{
				"id":             123,
				"name":           "bagg-app",
				"full_name":      "viktorvsk/bagg-app",
				"html_url":       "https://git-next.fibe.live/viktorvsk/bagg-app",
				"clone_url":      "https://git-next.fibe.live/viktorvsk/bagg-app.git",
				"default_branch": "main",
			},
			"prop_id": 456,
			"prop": map[string]any{
				"id":             456,
				"name":           "bagg-app",
				"repository_url": "https://git-next.fibe.live/viktorvsk/bagg-app",
				"provider":       "gitea",
			},
		})
	})

	private := true
	result, err := c.GiteaRepos.Create(context.Background(), &GiteaRepoCreateParams{Name: "bagg-app", Private: &private})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["name"] != "bagg-app" || body["private"] != true {
		t.Fatalf("unexpected body: %#v", body)
	}
	if result.PropID != 456 || result.Prop == nil || result.Prop.ID != 456 {
		t.Fatalf("expected prop in result, got %#v", result)
	}
	if result.Repo == nil || result.Repo.HTMLURL != "https://git-next.fibe.live/viktorvsk/bagg-app" {
		t.Fatalf("expected nested repo in result, got %#v", result)
	}
	if result.DefaultBranch != "main" {
		t.Fatalf("default branch=%q", result.DefaultBranch)
	}
}

func TestPlaygrounds_Action(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/playgrounds/42/action" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		json.NewEncoder(w).Encode(PlaygroundStatus{ID: 42, Status: "pending"})
	})

	force := true
	pg, err := c.Playgrounds.Action(context.Background(), 42, &PlaygroundActionParams{
		ActionType: PlaygroundActionRollout,
		Force:      &force,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", pg.Status)
	}
	if body["action_type"] != PlaygroundActionRollout {
		t.Fatalf("expected action_type=%q, got %#v", PlaygroundActionRollout, body)
	}
	if body["force"] != true {
		t.Fatalf("expected force=true, got %#v", body)
	}
}

func TestMarquees_UpdateSerializesDnsCredentialsForRails(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/api/marquees/1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		json.NewEncoder(w).Encode(Marquee{ID: 1, Name: "Elastic"})
	})

	provider := "cloudflare"
	_, err := c.Marquees.Update(context.Background(), 1, &MarqueeUpdateParams{
		DnsProvider:    &provider,
		DnsCredentials: map[string]string{"CF_DNS_API_TOKEN": "secret-token"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marquee := body["marquee"].(map[string]any)
	raw, ok := marquee["dns_credentials"].(string)
	if !ok {
		t.Fatalf("dns_credentials = %T (%#v), want JSON string", marquee["dns_credentials"], marquee["dns_credentials"])
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("dns_credentials should contain JSON object text: %v", err)
	}
	if decoded["CF_DNS_API_TOKEN"] != "secret-token" {
		t.Fatalf("unexpected dns_credentials payload: %#v", decoded)
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

func TestAgents_GetByIdentifierEscapesName(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/test-agent" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(Agent{ID: 9, Name: "test-agent", Provider: ProviderOpenAICodex})
	})

	agent, err := c.Agents.GetByIdentifier(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != 9 || agent.Name != "test-agent" {
		t.Fatalf("unexpected agent: %#v", agent)
	}
}

func TestAgents_GetByIdentifierEscapesSpaces(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/test%20agent" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(Agent{ID: 10, Name: "test agent", Provider: ProviderOpenAICodex})
	})

	agent, err := c.Agents.GetByIdentifier(context.Background(), "test agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != 10 || agent.Name != "test agent" {
		t.Fatalf("unexpected agent: %#v", agent)
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

func TestAgents_RestartChat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/restart_chat" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(AgentChatSession{ID: 123, Status: "pending"})
	})

	session, err := c.Agents.RestartChat(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 123 || session.Status != "pending" {
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

func TestAgents_UpdateRenameContextJSON(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/api/agents/99" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["agent"]["name"] != "renamed" {
			t.Errorf("expected agent name, got %#v", body["agent"]["name"])
		}
		context := body["agent_rename_context"]
		if context["conversation_client_id"] != "thread-123" {
			t.Errorf("expected conversation context, got %#v", context)
		}
		json.NewEncoder(w).Encode(Agent{ID: 99, Name: "renamed", Provider: ProviderGemini})
	})

	name := "renamed"
	agent, err := c.Agents.Update(context.Background(), 99, &AgentUpdateParams{
		Name: &name,
		RenameContext: &AgentRenameContext{
			ConversationClientID: "thread-123",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != name {
		t.Errorf("expected name %q, got %q", name, agent.Name)
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

func TestImportTemplates_SearchWithParamsRegex(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/import_templates/search" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("q") != "rails-.*" {
			t.Errorf("expected q=rails-.*, got %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("regex") != "true" {
			t.Errorf("expected regex=true, got %q", r.URL.Query().Get("regex"))
		}
		json.NewEncoder(w).Encode(listEnv([]ImportTemplate{{Name: "rails-starter"}}))
	})

	result, err := c.ImportTemplates.SearchWithParams(context.Background(), &ImportTemplateSearchParams{
		Query: "rails-.*",
		Regex: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].Name != "rails-starter" {
		t.Fatalf("unexpected result: %#v", result)
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
