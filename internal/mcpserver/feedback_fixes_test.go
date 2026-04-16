package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// ---------- Fix 1: empty updates rejected locally ----------

// TestEmptyUpdateRejected reproduces the Rails 400 ParameterMissing case
// the user hit on fibe_playgrounds_update. With only "id" in args, the
// outbound body would be {"playground": {}} which Rails treats as blank
// and rejects. We now short-circuit with a clear message before the HTTP
// round-trip.
func TestEmptyUpdateRejected(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Only id → no updatable fields.
	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_update", map[string]any{
		"id": 42,
	})
	if err == nil {
		t.Fatal("expected error for empty update, got nil")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("expected 'at least one field' message, got: %v", err)
	}

	// id + a real field → passes the guard (and will then fail client-side
	// on transport because we have no real server, which is fine for this
	// test — we only care that the guard didn't block it).
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_update", map[string]any{
		"id":   42,
		"name": "renamed",
	})
	if err != nil && strings.Contains(err.Error(), "at least one field") {
		t.Errorf("guard tripped on non-empty update: %v", err)
	}

	// Explicit nulls and empty strings are still treated as "no fields set".
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_update", map[string]any{
		"id":           42,
		"name":         "",
		"playspec_id":  nil,
	})
	if err == nil || !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("expected guard to trip on all-nil/empty fields, got: %v", err)
	}
}

// TestEmptyUpdateNestedRejected mirrors TestEmptyUpdateRejected for the
// nested (prop_id / agent_id scoped) update helpers.
func TestEmptyUpdateNestedRejected(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Only parent_id + id on a nested update → no fields to change.
	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_hunks_update", map[string]any{
		"prop_id": 1,
		"id":      2,
	})
	if err == nil || !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("expected empty-update guard on nested update, got: %v", err)
	}
}

// ---------- Fix 2: pipeline partial results on failure ----------

// TestPipelinePartialResultsOnFailure captures the user's biggest DX ask:
// when a later pipeline step fails without on_error:continue, earlier
// step outputs (e.g., freshly-created resource IDs) must still be
// surfaced so the agent can garbage-collect without re-running the
// pipeline blind.
func TestPipelinePartialResultsOnFailure(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4, PipelineMaxSteps: 10})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Step A succeeds and produces a fake ID the agent will want for cleanup.
	srv.dispatcher.register(&toolImpl{
		name: "test_create",
		tier: tierMeta,
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return map[string]any{"id": 99, "name": "provisional"}, nil
		},
	})
	// Step B fails with a structured API error.
	srv.dispatcher.register(&toolImpl{
		name: "test_fail",
		tier: tierMeta,
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return nil, &fibe.APIError{
				StatusCode: 422,
				Code:       fibe.ErrCodeValidationFailed,
				Message:    "nope",
				RequestID:  "req_test123",
				Details:    map[string]any{"field": []string{"required"}},
			}
		},
	})

	resp, err := srv.runPipeline(context.Background(), map[string]any{
		"steps": []any{
			map[string]any{"id": "a", "tool": "test_create", "args": map[string]any{}},
			map[string]any{"id": "b", "tool": "test_fail", "args": map[string]any{"x": "$.a.id"}},
			map[string]any{"id": "c", "tool": "test_create", "args": map[string]any{}},
		},
	})
	// Partial runs are returned as a successful pipeline response with
	// status:"partial" — the MCP call itself should not error.
	if err != nil {
		t.Fatalf("runPipeline should not error on mid-pipeline failure, got: %v", err)
	}
	m := resp.(map[string]any)

	if m["status"] != "partial" {
		t.Errorf("expected status=partial, got %v", m["status"])
	}

	// Step A's output must be present so the agent can use $.a.id for cleanup.
	steps := m["steps"].(map[string]any)
	a, ok := steps["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected step 'a' output in bindings, got %#v", steps["a"])
	}
	if a["id"].(float64) != 99 {
		t.Errorf("lost step 'a' ID in partial result: %#v", a)
	}

	// Step C should NOT have run (pipeline halts at first failure).
	if _, ran := steps["c"]; ran {
		t.Errorf("step 'c' should not have executed after 'b' failed")
	}

	// Error info must identify the failed step and include structured
	// APIError fields.
	errInfo, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error map in response, got %#v", m["error"])
	}
	if errInfo["step_id"] != "b" {
		t.Errorf("wrong failed step_id: %v", errInfo["step_id"])
	}
	if errInfo["tool"] != "test_fail" {
		t.Errorf("wrong failed tool: %v", errInfo["tool"])
	}
	if errInfo["code"] != fibe.ErrCodeValidationFailed {
		t.Errorf("expected propagated APIError code, got %v", errInfo["code"])
	}
	// Status may arrive as int (direct return) or float64 (after JSON round-trip);
	// normalize for the comparison.
	status := asInt(errInfo["status"])
	if status != 422 {
		t.Errorf("expected status=422, got %v (%T)", errInfo["status"], errInfo["status"])
	}
	if errInfo["request_id"] != "req_test123" {
		t.Errorf("expected request_id to carry through, got %v", errInfo["request_id"])
	}

	// completed_step_ids must list step 'a' but not 'b' or 'c'.
	completed, ok := m["completed_step_ids"].([]string)
	if !ok {
		// JSON round-trip through interface{} may produce []any; accept either.
		if arr, ok2 := m["completed_step_ids"].([]any); ok2 {
			completed = make([]string, len(arr))
			for i, v := range arr {
				completed[i], _ = v.(string)
			}
		} else {
			t.Fatalf("completed_step_ids not a string slice: %T", m["completed_step_ids"])
		}
	}
	if len(completed) != 1 || completed[0] != "a" {
		t.Errorf("expected completed_step_ids=[a], got %v", completed)
	}

	// Pipeline result must also be cacheable so the agent can re-query
	// the partial bindings later via fibe_pipeline_result.
	if m["pipeline_id"] == nil {
		t.Error("expected pipeline_id on partial result for later cleanup lookup")
	}
}

// ---------- Fix 3: webhook event-type hint in description/schema ----------

func TestWebhookEnumHintInToolDescription(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	tool, ok := srv.dispatcher.lookup("fibe_webhooks_create")
	if !ok {
		t.Fatal("fibe_webhooks_create not registered")
	}
	if !strings.Contains(tool.description, "fibe_webhooks_event_types") {
		t.Errorf("expected description to reference fibe_webhooks_event_types, got: %s", tool.description)
	}
	if !strings.Contains(tool.description, "agent.created") {
		t.Errorf("expected description to cite at least one concrete event type, got: %s", tool.description)
	}
}

func TestWebhookEnumHintInSchemaRegistry(t *testing.T) {
	hook, ok := schemaRegistry["webhook"]
	if !ok {
		t.Fatal("webhook entry missing from schemaRegistry")
	}
	create, ok := hook["create"].(map[string]any)
	if !ok {
		t.Fatal("webhook.create entry malformed")
	}
	props := create["properties"].(map[string]any)
	events := props["events"].(map[string]any)
	desc := events["description"].(string)
	if !strings.Contains(desc, "fibe_webhooks_event_types") {
		t.Errorf("events description should point to fibe_webhooks_event_types, got: %s", desc)
	}
	// Examples should include canonical event identifiers.
	examples, ok := events["examples"].([]string)
	if !ok || len(examples) == 0 {
		t.Errorf("expected non-empty examples on events schema, got %#v", events["examples"])
	}
}

// asInt normalizes any numeric-shaped interface value into an int. Lets
// tests be agnostic about whether the pipeline response went through JSON
// round-trip (float64) or was returned directly (int).
func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}

// Keep imports live for future assertions.
var _ = json.Marshal
var _ = errors.New
