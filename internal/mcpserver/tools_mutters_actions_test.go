package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func TestMutterToolCreatesAgentMutter(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	agentName := fmt.Sprintf("test-agent-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     agentName,
			"provider": "openai-codex",
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	m := res.(*fibe.Agent)
	agentID := int(m.ID)

	os.Setenv("FIBE_AGENT_ID", fmt.Sprintf("%d", agentID))
	defer os.Unsetenv("FIBE_AGENT_ID")

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type": "proof",
		"body": "Verified rollout completed.",
	}); err != nil {
		t.Fatalf("fibe_mutter dispatch: %v", err)
	}
}

func TestMutterToolValidatesBeforeAPI(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	os.Setenv("FIBE_AGENT_ID", "bad")
	defer os.Unsetenv("FIBE_AGENT_ID")

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type": "proof",
		"body": "bad id",
	})
	if err == nil || !strings.Contains(err.Error(), "missing or invalid") {
		t.Fatalf("expected missing or invalid environment variable error, got %v", err)
	}

	os.Setenv("FIBE_AGENT_ID", "7")

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type":  "proof",
		"body":  "extra",
		"extra": true,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported field") {
		t.Fatalf("expected local unsupported field error, got %v", err)
	}
}

func TestMuttersGetSchemaRequiresAgentID(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	schema := srv.toolSchemas["fibe_mutters_get"]
	props := schema["properties"].(map[string]any)
	if _, ok := props["id_or_name"]; !ok {
		t.Fatalf("fibe_mutters_get schema missing id_or_name: %#v", schema)
	}
	for _, bad := range []string{"PlaygroundID", "Query", "PerPage"} {
		if _, ok := props[bad]; ok {
			t.Fatalf("fibe_mutters_get schema should use snake_case, found %q in %#v", bad, schema)
		}
	}
	if playground, ok := props["playground_id_or_name"].(map[string]any); !ok {
		t.Fatalf("fibe_mutters_get playground_id_or_name should be identifier schema: %#v", props["playground_id_or_name"])
	} else if _, ok := playground["oneOf"]; !ok {
		t.Fatalf("fibe_mutters_get playground_id_or_name should accept ID or name: %#v", playground)
	}
	required, _ := schema["required"].([]any)
	if !containsAnyString(required, "id_or_name") {
		t.Fatalf("fibe_mutters_get schema should require id_or_name: %#v", schema)
	}
	if len(required) != 1 {
		t.Fatalf("fibe_mutters_get should only require id_or_name: %#v", schema)
	}
}

func TestMuttersGetUsesAgentIDAndFilters(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	agentName := fmt.Sprintf("test-agent-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     agentName,
			"provider": "openai-codex",
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	m := res.(*fibe.Agent)
	agentID := int(m.ID)

	os.Setenv("FIBE_AGENT_ID", fmt.Sprintf("%d", agentID))
	defer os.Unsetenv("FIBE_AGENT_ID")

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type": "proof",
		"body": "deploying highly critical update",
	}); err != nil {
		t.Fatalf("dispatch fibe_mutter: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutters_get", map[string]any{
		"id_or_name": agentID,
		"query":      "deploy",
		"page":       1,
		"per_page":   10,
	}); err != nil {
		t.Fatalf("fibe_mutters_get dispatch: %v", err)
	}
}

func containsAnyString(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
