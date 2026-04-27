package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResourceMutateDispatchesCreateAndUpdate(t *testing.T) {
	var seen []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		switch r.Method + " " + r.URL.Path {
		case "POST /api/agents":
			agent := body["agent"].(map[string]any)
			if agent["name"] != "builder" || agent["provider"] != "openai-codex" {
				t.Fatalf("unexpected create body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":11,"name":"builder","provider":"openai-codex"}`))
		case "PATCH /api/agents/11":
			agent := body["agent"].(map[string]any)
			if agent["name"] != "renamed" {
				t.Fatalf("unexpected update body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":11,"name":"renamed","provider":"openai-codex"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     "builder",
			"provider": "openai-codex",
		},
	}); err != nil {
		t.Fatalf("create dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agent",
		"operation": "update",
		"payload": map[string]any{
			"agent_id": 11,
			"name":     "renamed",
		},
	}); err != nil {
		t.Fatalf("update dispatch: %v", err)
	}
	if len(seen) != 2 {
		t.Fatalf("expected two API calls, got %#v", seen)
	}
}

func TestResourceMutateRejectsInvalidPayloadBeforeAPI(t *testing.T) {
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Fatalf("unexpected API call: %s %s", r.Method, r.URL.Path)
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "api_key",
		"operation": "update",
		"payload": map[string]any{
			"api_key_id": 1,
			"label":      "bad",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "does not support operation") {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agent",
		"operation": "create",
		"payload": map[string]any{
			"name": "missing provider",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("expected local required-field error, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("invalid payloads should not hit API, got %d call(s)", calls)
	}
}

func TestResourceMutateDryRunValidatesWithoutAPI(t *testing.T) {
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Fatalf("dry run should not call API: %s %s", r.Method, r.URL.Path)
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "props",
		"operation": "attach",
		"dry_run":   true,
		"payload": map[string]any{
			"repo_full_name": "octocat/Hello-World",
		},
	})
	if err != nil {
		t.Fatalf("dry run dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["resource"] != "prop" || result["operation"] != "attach" || result["dry_run"] != true || result["valid"] != true {
		t.Fatalf("unexpected dry run result: %#v", result)
	}
	if calls != 0 {
		t.Fatalf("dry run made %d API call(s)", calls)
	}
}

func TestResourceMutateDispatchesScopedActions(t *testing.T) {
	seen := map[string]int{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		seen[key]++

		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		switch key {
		case "POST /api/marquees/autoconnect_token":
			_, _ = w.Write([]byte(`{"token":"tok"}`))
		case "POST /api/marquees/3/generate_ssh_key":
			_, _ = w.Write([]byte(`{"public_key":"ssh-rsa test"}`))
		case "POST /api/marquees/3/test_connection":
			_, _ = w.Write([]byte(`{"success":true,"message":"ok"}`))
		case "POST /api/props/attach":
			if body["repo_full_name"] != "octocat/Hello-World" {
				t.Fatalf("unexpected attach body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":31,"name":"Hello-World"}`))
		case "POST /api/props/mirror":
			if body["source_url"] != "https://github.com/octocat/Hello-World" || body["name"] != "mirror" {
				t.Fatalf("unexpected mirror body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":32,"name":"mirror"}`))
		case "POST /api/props/3/sync":
			_, _ = w.Write([]byte(`{"message":"Sync scheduled"}`))
		case "POST /api/import_templates/4/fork":
			_, _ = w.Write([]byte(`{"id":44,"name":"forked"}`))
		case "POST /api/import_templates/4/source/refresh":
			_, _ = w.Write([]byte(`{"success":true,"version_created":false}`))
		case "PUT /api/import_templates/4/source":
			source := body["source"].(map[string]any)
			if source["source_prop_id"] != float64(3) || source["source_path"] != "template.yml" || source["ci_marquee_id"] != float64(3) {
				t.Fatalf("unexpected source body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":4,"name":"template"}`))
		case "POST /api/import_templates/4/versions/5/upgrade_linked_playspecs":
			_, _ = w.Write([]byte(`{"success":true,"upgraded_count":1,"failed_count":0}`))
		case "PATCH /api/import_templates/4/toggle_public":
			if body["version_id"] != float64(5) {
				t.Fatalf("unexpected toggle body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":5,"version":1,"public":true}`))
		case "POST /api/playgrounds":
			_, _ = w.Write([]byte(`{"id":71,"name":"trick","job_mode":true}`))
		case "GET /api/playgrounds/8":
			_, _ = w.Write([]byte(`{"id":8,"name":"source","playspec_id":7,"marquee_id":3,"job_mode":true}`))
		case "GET /api/playspecs/7":
			_, _ = w.Write([]byte(`{"id":7,"name":"job"}`))
		case "POST /api/webhook_endpoints/9/test":
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected request: %s with body %#v", key, body)
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	cases := []struct {
		resource  string
		operation string
		payload   map[string]any
	}{
		{resource: "marquee", operation: "autoconnect_token", payload: map[string]any{"ssl_mode": "http"}},
		{resource: "marquee", operation: "generate_ssh_key", payload: map[string]any{"marquee_id": 3}},
		{resource: "marquee", operation: "test_connection", payload: map[string]any{"marquee_id": 3}},
		{resource: "prop", operation: "attach", payload: map[string]any{"repo_full_name": "octocat/Hello-World"}},
		{resource: "prop", operation: "mirror", payload: map[string]any{"source_url": "https://github.com/octocat/Hello-World", "name": "mirror"}},
		{resource: "prop", operation: "sync", payload: map[string]any{"prop_id": 3}},
		{resource: "template", operation: "fork", payload: map[string]any{"template_id": 4}},
		{resource: "template", operation: "source_refresh", payload: map[string]any{"template_id": 4}},
		{resource: "template", operation: "source_set", payload: map[string]any{"template_id": 4, "source_prop_id": 3, "source_path": "template.yml", "ci_marquee_id": 3}},
		{resource: "template", operation: "upgrade_playspecs", payload: map[string]any{"template_id": 4, "version_id": 5}},
		{resource: "template_version", operation: "toggle_public", payload: map[string]any{"template_id": 4, "version_id": 5}},
		{resource: "trick", operation: "trigger", payload: map[string]any{"playspec_id": 7, "marquee_id": 3, "name": "trick"}},
		{resource: "trick", operation: "rerun", payload: map[string]any{"trick_id": 8}},
		{resource: "webhook", operation: "test", payload: map[string]any{"webhook_id": 9}},
	}

	for _, tc := range cases {
		t.Run(tc.resource+"."+tc.operation, func(t *testing.T) {
			if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
				"resource":  tc.resource,
				"operation": tc.operation,
				"payload":   tc.payload,
			}); err != nil {
				t.Fatalf("dispatch: %v", err)
			}
		})
	}

	for _, want := range []string{
		"POST /api/marquees/autoconnect_token",
		"POST /api/marquees/3/generate_ssh_key",
		"POST /api/marquees/3/test_connection",
		"POST /api/props/attach",
		"POST /api/props/mirror",
		"POST /api/props/3/sync",
		"POST /api/import_templates/4/fork",
		"POST /api/import_templates/4/source/refresh",
		"PUT /api/import_templates/4/source",
		"POST /api/import_templates/4/versions/5/upgrade_linked_playspecs",
		"PATCH /api/import_templates/4/toggle_public",
		"GET /api/playgrounds/8",
		"GET /api/playspecs/7",
		"POST /api/webhook_endpoints/9/test",
	} {
		if seen[want] == 0 {
			t.Fatalf("expected request %s, saw %#v", want, seen)
		}
	}
	if seen["POST /api/playgrounds"] != 2 {
		t.Fatalf("expected two playground creates from trick trigger/rerun, saw %#v", seen)
	}
}

func TestResourceMutateScopedActionValidationBeforeAPI(t *testing.T) {
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Fatalf("unexpected API call: %s %s", r.Method, r.URL.Path)
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, tc := range []struct {
		name      string
		resource  string
		operation string
		payload   map[string]any
		want      string
	}{
		{name: "bad id", resource: "template", operation: "source_refresh", payload: map[string]any{"template_id": 0}, want: "greater than or equal to 1"},
		{name: "unknown field", resource: "prop", operation: "mirror", payload: map[string]any{"source_url": "https://github.com/o/r", "extra": true}, want: "unsupported field"},
		{name: "old prop alias rejected", resource: "prop", operation: "attach", payload: map[string]any{"repository_url": "https://github.com/o/r"}, want: "repo_full_name is required"},
		{name: "old template id alias rejected", resource: "template", operation: "source_set", payload: map[string]any{"id": 1, "source_prop_id": 2, "source_path": "template.yml"}, want: "template_id is required"},
		{name: "unsupported pair", resource: "webhook", operation: "source_set", payload: map[string]any{"webhook_id": 1}, want: "does not support operation"},
		{name: "dedicated mutter tool", resource: "mutter", operation: "create", payload: map[string]any{"type": "proof", "body": "done"}, want: "does not support mutation operations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
				"resource":  tc.resource,
				"operation": tc.operation,
				"payload":   tc.payload,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}

	if calls != 0 {
		t.Fatalf("invalid payloads should not hit API, got %d call(s)", calls)
	}
}

func TestResourceMutateListedConcreteToolsRemoved(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, name := range []string{
		"fibe_marquees_autoconnect_token",
		"fibe_marquees_generate_ssh_key",
		"fibe_marquees_test_connection",
		"fibe_props_attach",
		"fibe_props_mirror",
		"fibe_props_sync",
		"fibe_templates_fork",
		"fibe_templates_source_refresh",
		"fibe_templates_source_set",
		"fibe_templates_upgrade_playspecs",
		"fibe_templates_versions_toggle_public",
		"fibe_tricks_rerun",
		"fibe_tricks_trigger",
		"fibe_webhooks_test",
		"fibe_job_env_set",
		"fibe_agents_raw_providers_get",
		"fibe_agents_raw_providers_update",
		"fibe_templates_lineage",
		"fibe_marquees_generate_ssh_key_status",
		"fibe_marquees_test_connection_status",
		"fibe_feedbacks_create",
		"fibe_feedbacks_delete",
		"fibe_feedbacks_update",
		"fibe_mutters_create",
	} {
		if _, ok := srv.dispatcher.lookup(name); ok {
			t.Fatalf("%s should not be registered", name)
		}
	}

	for _, name := range []string{
		"fibe_templates_launch",
		"fibe_feedbacks_get",
		"fibe_feedbacks_list",
		"fibe_mutter",
	} {
		if _, ok := srv.dispatcher.lookup(name); !ok {
			t.Fatalf("%s should remain registered", name)
		}
	}
}
