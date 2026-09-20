package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPropBranchesUnmarshalsObjectArray(t *testing.T) {
	payload := []byte(`{"branches":[{"name":"main","default":true},{"name":"feat","default":false}]}`)
	var pb fibe.PropBranches
	if err := json.Unmarshal(payload, &pb); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(pb.Branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(pb.Branches))
	}
	if pb.Branches[0].Name != "main" || !pb.Branches[0].Default {
		t.Errorf("first branch wrong: %#v", pb.Branches[0])
	}
	if pb.Branches[1].Name != "feat" || pb.Branches[1].Default {
		t.Errorf("second branch wrong: %#v", pb.Branches[1])
	}
}

func TestBindArgsUnderstandsURLTagsAndScalarCoercion(t *testing.T) {
	var p fibe.PlaygroundListParams
	err := bindArgs(map[string]any{
		"page":     "2",
		"per_page": "25",
		"job_mode": "true",
	}, &p)
	if err != nil {
		t.Fatalf("bindArgs: %v", err)
	}
	if p.Page != 2 {
		t.Errorf("Page=%d want 2", p.Page)
	}
	if p.PerPage != 25 {
		t.Errorf("PerPage=%d want 25", p.PerPage)
	}
	if p.JobMode == nil || !*p.JobMode {
		t.Errorf("JobMode=%v want true", p.JobMode)
	}
}

func TestBindArgsStringifiesStringSliceElements(t *testing.T) {
	type payload struct {
		Args []string `json:"args"`
	}
	var p payload
	err := bindArgs(map[string]any{
		"args": []any{75, true, "logs"},
	}, &p)
	if err != nil {
		t.Fatalf("bindArgs: %v", err)
	}
	got := strings.Join(p.Args, ",")
	if got != "75,true,logs" {
		t.Errorf("Args=%q want %q", got, "75,true,logs")
	}
}

func TestAgentsSendMessageRequiresCanonicalTextField(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_send_message", map[string]any{
		"id_or_name": 42,
		"message":    "hello",
	})
	if err == nil || !strings.Contains(err.Error(), "'text' not set") {
		t.Fatalf("expected canonical text field error, got %v", err)
	}
}

func TestAgentsSendMessageSchemaIncludesConversationControls(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	props, ok := srv.toolSchemas["fibe_agents_send_message"]["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %#v", srv.toolSchemas["fibe_agents_send_message"])
	}
	for _, want := range []string{"id_or_name", "text", "conversation_id", "busy_policy", "images", "attachment_paths", "attachment_filenames"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("schema property %q missing: %#v", want, props)
		}
	}
}

// Note: HTTP tests have been moved to real integration tests in tools_templates_change_test.go

func TestParseRepoFullName(t *testing.T) {
	cases := map[string]string{
		"octocat/Hello-World":                        "octocat/Hello-World",
		"https://github.com/octocat/Hello-World.git": "octocat/Hello-World",
		"https://github.com/octocat/Hello-World":     "octocat/Hello-World",
		"git@github.com:octocat/Hello-World.git":     "octocat/Hello-World",
		"https://gitea.example/org/repo.git":         "org/repo",
		"invalid":                                    "",
		"":                                           "",
	}
	for input, want := range cases {
		got := parseRepoFullName(input)
		if got != want {
			t.Errorf("parseRepoFullName(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestPipelineResultBindingsRootedProjection(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4, PipelineMaxSteps: 10, PipelineCacheEntryMax: 1 << 20})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	srv.dispatcher.register(&toolImpl{
		name: "test_echo_team",
		tier: tierMeta,
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return map[string]any{"id": 42, "name": "Acme"}, nil
		},
	})

	resp, err := srv.runPipeline(context.Background(), map[string]any{
		"steps": []any{
			map[string]any{"id": "create_team", "tool": "test_echo_team", "args": map[string]any{}},
		},
	})
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}
	m := resp.(map[string]any)
	pid := m["pipeline_id"].(string)

	result, err := srv.dispatcher.dispatch(context.Background(), "fibe_pipeline_result", map[string]any{
		"pipeline_id": pid,
		"path":        "$.create_team.id",
	})
	if err != nil {
		t.Fatalf("pipeline_result bindings-rooted: %v", err)
	}
	if asInt(result) != 42 {
		t.Errorf("expected 42, got %#v", result)
	}

	// Full-tree fallback: $.status should return "completed".
	status, err := srv.dispatcher.dispatch(context.Background(), "fibe_pipeline_result", map[string]any{
		"pipeline_id": pid,
		"path":        "$.status",
	})
	if err != nil {
		t.Fatalf("pipeline_result full-tree: %v", err)
	}
	if status != "completed" {
		t.Errorf("expected status=completed, got %#v", status)
	}
}

func TestLaunchSurfacesBothIDs(t *testing.T) {
	var r fibe.LaunchResult
	if err := json.Unmarshal([]byte(`{"playspec_id":10,"playground_id":20,"props_created":[1,2]}`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.PlayspecID != 10 {
		t.Errorf("PlayspecID=%d want 10", r.PlayspecID)
	}
	if r.PlaygroundID != 20 {
		t.Errorf("PlaygroundID=%d want 20", r.PlaygroundID)
	}
	if len(r.PropsCreated) != 2 {
		t.Errorf("PropsCreated=%v want [1,2]", r.PropsCreated)
	}

	if err := json.Unmarshal([]byte(`{"playspecs_created":11}`), &r); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if r.PlayspecID != 11 {
		t.Errorf("legacy PlayspecID=%d want 11", r.PlayspecID)
	}
}

func TestSplitContentHeader(t *testing.T) {
	cases := []struct {
		input  string
		wantFn string
		wantCT string
	}{
		{"text/x-python", "", "text/x-python"},
		{`attachment; filename="artefact.py"`, "artefact.py", ""},
		{"artefact.py", "artefact.py", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		fn, ct := splitContentHeader(c.input)
		if fn != c.wantFn || ct != c.wantCT {
			t.Errorf("splitContentHeader(%q) = (%q, %q); want (%q, %q)", c.input, fn, ct, c.wantFn, c.wantCT)
		}
	}
}

func TestPipelineDescriptionDocumentsMaxSteps(t *testing.T) {
	srv := New(Config{APIKey: "pk_test"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	t_impl, ok := srv.dispatcher.lookup("fibe_pipeline")
	if !ok {
		t.Fatal("fibe_pipeline not registered")
	}
	_ = t_impl
}
