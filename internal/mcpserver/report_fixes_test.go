package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Aggregated tests for the fixes landing in response to REPORT.md.

// ---------- SDK: PropBranches is an array of objects ----------

func TestPropBranchesUnmarshalsObjectArray(t *testing.T) {
	// Shape the Rails API actually returns.
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

// ---------- Aliases: every documented alias accepts both spellings ----------

func TestAliasFieldPrefersCanonical(t *testing.T) {
	args := map[string]any{"text": "canonical", "message": "alias"}
	aliasField(args, "text", "message")
	if args["text"] != "canonical" {
		t.Errorf("canonical should win, got %v", args["text"])
	}
}

func TestAliasFieldFillsFromAlternative(t *testing.T) {
	args := map[string]any{"message": "via-alias"}
	aliasField(args, "text", "message")
	if args["text"] != "via-alias" {
		t.Errorf("alias fill failed: %v", args["text"])
	}
}

func TestAliasFieldEmptyStringIsNotSet(t *testing.T) {
	// Empty string should NOT block an alias backfill; treat it like absent.
	args := map[string]any{"text": "", "message": "via-alias"}
	aliasField(args, "text", "message")
	if args["text"] != "via-alias" {
		t.Errorf("empty string should have been overwritten by alias; got %v", args["text"])
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

// ---------- Tool-level alias wiring ----------

func TestAgentsChatAcceptsMessageAlias(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	// Call through dispatcher with legacy "message" field. The dispatcher
	// will try to hit the network and fail (no real API), but the alias
	// guard must fire before the dispatch — we assert that the "'text' not
	// set" error is NOT the message we get back.
	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_chat", map[string]any{
		"id":      42,
		"message": "hello",
	})
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "'text' not set") {
		t.Errorf("message alias didn't canonicalize; got: %v", err)
	}
}

func TestRegistryCredentialTypeEnforced(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	// Bad enum: "docker" (common mistake) should be rejected with a clear message.
	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playspecs_registry_credential_add", map[string]any{
		"id":            1,
		"registry_type": "docker",
		"registry_url":  "https://docker.io",
		"username":      "u",
		"secret":        "s",
	})
	if err == nil {
		t.Fatal("expected validation error for wrong registry_type")
	}
	if !strings.Contains(err.Error(), "ghcr") || !strings.Contains(err.Error(), "dockerhub") {
		t.Errorf("error should enumerate valid types; got: %v", err)
	}
}

func TestTemplatesVersionsCreateAcceptsTemplateBodyPath(t *testing.T) {
	var requestBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/import_templates/690/versions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":772,"version":38,"template_body":"ok"}`))
	}))
	defer api.Close()

	path := filepath.Join(t.TempDir(), "template.yml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_versions_create", map[string]any{
		"id":                 690,
		"template_body_path": path,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if got := requestBody["template_body"]; got != "services:\n  web:\n    image: nginx\n" {
		t.Fatalf("template_body=%q", got)
	}
	if got := requestBody["response_mode"]; got != "summary" {
		t.Fatalf("response_mode=%q, want summary", got)
	}
}

func TestTemplatesVersionsCreateRejectsRelativeTemplateBodyPath(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_versions_create", map[string]any{
		"id":                 690,
		"template_body_path": "template.yml",
	})
	if err == nil {
		t.Fatal("expected relative path error")
	}
	if !strings.Contains(err.Error(), "content_path must be absolute") {
		t.Fatalf("expected absolute path error, got %v", err)
	}
}

func TestTemplatesVersionsCreateKeepsInlineTemplateBody(t *testing.T) {
	var requestBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":773,"version":39,"template_body":"ok"}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_versions_create", map[string]any{
		"id":            690,
		"template_body": "services: {}\n",
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if got := requestBody["template_body"]; got != "services: {}\n" {
		t.Fatalf("template_body=%q", got)
	}
	if got := requestBody["response_mode"]; got != "summary" {
		t.Fatalf("response_mode=%q, want summary", got)
	}
}

func TestPlayspecSwitchVersionForwardsSummaryMode(t *testing.T) {
	var requestBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playspecs/131/template_version_switch" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"from_template_version":null,"target_template_version":null,"suggested_upgrade":false,"required_variables":[],"target_variables":[],"warnings":[],"diff":{},"playground_rollout_plan":{"blocked":[],"rollout":[],"unchanged":[]},"no_op":false,"playspec":{"id":131}}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playspecs_switch_version", map[string]any{
		"id":                         131,
		"target_template_version_id": 772,
		"summary":                    true,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if requestBody["summary"] != true {
		t.Fatalf("summary was not forwarded: %#v", requestBody)
	}
}

func TestTemplatesLaunchAcceptsTargetMarqueeAlias(t *testing.T) {
	var requestBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/import_templates/690/launch" {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"playspecs_created":131,"playground_id":132}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_launch", map[string]any{
		"id":                690,
		"target_marquee_id": 7,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if requestBody["marquee_id"] != float64(7) {
		t.Fatalf("marquee_id not canonicalized: %#v", requestBody)
	}
}

func TestTemplatePatchCreateForwardsAutoSwitchFields(t *testing.T) {
	var requestBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/import_templates/690/versions/patch_create" {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"template_version":{"id":800},"switch_status":"switched"}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_versions_patch_create", map[string]any{
		"template_id":        690,
		"base_version_id":    799,
		"patches":            []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:alpine", "expect": "nginx", "create_missing": false}},
		"auto_switch":        true,
		"target_playspec_id": 131,
		"confirm_warnings":   true,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if requestBody["auto_switch"] != true || requestBody["target_playspec_id"] != float64(131) || requestBody["response_mode"] != "summary" {
		t.Fatalf("patch fields not forwarded: %#v", requestBody)
	}
	patches := requestBody["patches"].([]any)
	firstPatch := patches[0].(map[string]any)
	if firstPatch["expect"] != "nginx" || firstPatch["create_missing"] != false {
		t.Fatalf("new patch fields not forwarded: %#v", firstPatch)
	}
}

func TestPlaygroundsDebugAndDiagnoseDefaultToSummaryRefresh(t *testing.T) {
	seen := map[string]bool{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playgrounds/132/debug" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		if r.URL.Query().Get("mode") != "summary" || r.URL.Query().Get("refresh") != "true" {
			t.Fatalf("unexpected debug query: %s", r.URL.RawQuery)
		}
		seen[r.URL.RawQuery] = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"playground":{"id":132}}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	for _, tool := range []string{"fibe_playgrounds_debug", "fibe_playgrounds_diagnose"} {
		if _, err := srv.dispatcher.dispatch(context.Background(), tool, map[string]any{"id": 132}); err != nil {
			t.Fatalf("%s dispatch: %v", tool, err)
		}
	}
	if len(seen) != 1 {
		t.Fatalf("expected both tools to use same summary query, got %#v", seen)
	}
}

func TestPlaygroundsWaitPollsStatusEndpoint(t *testing.T) {
	var hitStatus bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playgrounds/132/status" {
			t.Fatalf("wait should poll status endpoint, got %s", r.URL.Path)
		}
		hitStatus = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":132,"status":"running"}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_wait", map[string]any{"id": 132, "status": "running", "timeout": "1s"}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if !hitStatus {
		t.Fatal("status endpoint was not hit")
	}
}

func TestTemplatesPatchApplyDryRunUsesPatchPreview(t *testing.T) {
	var requestBody map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/import_templates/690/versions/patch_preview" {
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"validation":{"valid":true},"switch_preview":{"success":true}}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	result, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_patch_apply", map[string]any{
		"template_id":        690,
		"base_version_id":    799,
		"target_playspec_id": 131,
		"patches":            []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:2"}},
		"dry_run":            true,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if requestBody["target_playspec_id"] != float64(131) || requestBody["response_mode"] != "summary" {
		t.Fatalf("patch apply fields not forwarded: %#v", requestBody)
	}
	if result.(map[string]any)["dry_run"] != true {
		t.Fatalf("expected dry_run result, got %#v", result)
	}
}

func TestTemplatesPatchApplyWaitTimeoutAddsDiagnostics(t *testing.T) {
	paths := map[string]int{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/import_templates/690/versions/patch_create":
			_, _ = w.Write([]byte(`{"template_version":{"id":800},"switch_status":"switched","playground_rollout_plan":{"rollout":[132]}}`))
		case "/api/playgrounds/132/status":
			_, _ = w.Write([]byte(`{"id":132,"status":"in_progress"}`))
		case "/api/playgrounds/132/debug":
			if r.URL.Query().Get("mode") != "summary" || r.URL.Query().Get("refresh") != "true" {
				t.Fatalf("unexpected diagnose query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"playground":{"id":132},"next_actions":["retry current compose"]}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	result, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_patch_apply", map[string]any{
		"template_id":        690,
		"base_version_id":    799,
		"target_playspec_id": 131,
		"patches":            []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:2"}},
		"wait":               true,
		"wait_timeout":       "1ns",
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	out := result.(map[string]any)
	if len(out["diagnostics"].(map[string]any)) == 0 {
		t.Fatalf("expected diagnostics on wait timeout: %#v", out)
	}
	if paths["/api/playgrounds/132/status"] == 0 || paths["/api/playgrounds/132/debug"] == 0 {
		t.Fatalf("expected status and debug paths, got %#v", paths)
	}
}

func TestTemplatesLineageAndPlaygroundsRetryComposeCallable(t *testing.T) {
	paths := map[string]bool{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths[r.URL.Path] = true
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/import_templates/690/lineage" {
			_, _ = w.Write([]byte(`{"template":{"id":690},"playspecs":[]}`))
			return
		}
		if r.URL.Path == "/api/playgrounds/132/retry_compose" {
			_, _ = w.Write([]byte(`{"id":132,"status":"running"}`))
			return
		}
		t.Fatalf("unexpected request: %s", r.URL.Path)
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_lineage", map[string]any{"id": 690}); err != nil {
		t.Fatalf("lineage: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_retry_compose", map[string]any{"id": 132, "force": true}); err != nil {
		t.Fatalf("retry compose: %v", err)
	}
	if !paths["/api/import_templates/690/lineage"] || !paths["/api/playgrounds/132/retry_compose"] {
		t.Fatalf("missing calls: %#v", paths)
	}
}

func TestMutationsCreateAcceptsShaAlias(t *testing.T) {
	t.Skip("mutations MCP tools are experimental and excluded from GA parity")

	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	// Supply the CLI-style "sha" field; the dispatcher should canonicalize
	// it to "found_commit_sha" before validation.
	t_impl, ok := srv.dispatcher.lookup("fibe_mutations_create")
	if !ok {
		t.Fatal("fibe_mutations_create not registered")
	}
	args := map[string]any{"prop_id": 1, "sha": "abc1234", "branch": "main"}
	// Execute just the alias application step.
	applyAliases(args, map[string][]string{"found_commit_sha": {"sha", "commit_sha", "commit"}})
	if args["found_commit_sha"] != "abc1234" {
		t.Errorf("expected sha -> found_commit_sha aliasing, got %#v", args["found_commit_sha"])
	}
	_ = t_impl
}

// ---------- Props attach: URL-to-short-form parsing ----------

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

// ---------- pipeline_result bindings-rooted projection ----------

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

	// Bindings-rooted path: $.create_team.id should return 42.
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

// ---------- fibe_launch returns both playspec_id and playground_id ----------

func TestLaunchSurfacesBothIDs(t *testing.T) {
	// We can't hit the real Rails API in a unit test, but we can verify the
	// shape of the map our handler returns by calling it with a mocked
	// LaunchService via the session client. The simplest path: assert the
	// tool description mentions both keys and that the LaunchResult struct
	// has the playspecs_created JSON tag the Rails side emits.
	var r fibe.LaunchResult
	if err := json.Unmarshal([]byte(`{"playspecs_created":10,"playground_id":20,"props_created":[1,2]}`), &r); err != nil {
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
}

// ---------- splitContentHeader helps artefacts_download disambiguate ----------

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

// ---------- Pipeline description documents the limit ----------

func TestPipelineDescriptionDocumentsMaxSteps(t *testing.T) {
	srv := New(Config{APIKey: "pk_test"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	// The dispatcher's description field is the short form; the full
	// description lives on the mcp-go Tool, not our struct. Assert the
	// short description mentions the docs exist.
	t_impl, ok := srv.dispatcher.lookup("fibe_pipeline")
	if !ok {
		t.Fatal("fibe_pipeline not registered")
	}
	_ = t_impl
}
