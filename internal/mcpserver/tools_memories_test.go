package mcpserver

import "testing"

func TestMemorizeToolIsRegisteredInBaseTier(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	tool, ok := srv.dispatcher.lookup("fibe_memorize")
	if !ok {
		t.Fatalf("fibe_memorize not registered")
	}
	if tool.tier != tierBase {
		t.Fatalf("fibe_memorize tier = %v, want base", tool.tier)
	}
	if !advertisedToolNames(srv)["fibe_memorize"] {
		t.Fatalf("fibe_memorize should be advertised in core mode")
	}

	for _, removed := range []string{"fibe_memories_search", "fibe_memories_get", "fibe_memories_sync", "fibe_memories_delete"} {
		if _, ok := srv.dispatcher.lookup(removed); ok {
			t.Fatalf("%s should be replaced by generic resource tools or fibe_memorize", removed)
		}
	}
}

func TestMemoryToolSchemas(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	schema := srv.toolSchemas["fibe_memorize"]
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("memorize schema properties missing: %#v", schema)
	}
	for _, want := range []string{"conversation_id", "content", "tags", "confidence", "groundings", "only"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("memorize schema property %q missing: %#v", want, props)
		}
	}
	if _, ok := props["conversation"]; ok {
		t.Fatalf("memorize schema should not expose conversation object: %#v", props)
	}
	for _, disallowed := range []string{"messages", "raw_content", "provider"} {
		if _, ok := props[disallowed]; ok {
			t.Fatalf("memorize schema should not ask agents for %s: %#v", disallowed, props)
		}
	}
	if _, ok := props["output_path"]; ok {
		t.Fatalf("memorize schema should not expose output_path: %#v", props)
	}
}

func TestApplyDefaultMemoryAgentID(t *testing.T) {
	t.Setenv("FIBE_AGENT_ID", "123")
	memory := map[string]any{"content": "one"}

	if err := applyDefaultMemoryAgentID(memory); err != nil {
		t.Fatalf("applyDefaultMemoryAgentID: %v", err)
	}

	if got := memory["agent_id"]; got != int64(123) {
		t.Fatalf("agent_id = %#v", got)
	}

	memory = map[string]any{"content": "two", "agent_id": int64(456)}
	if err := applyDefaultMemoryAgentID(memory); err != nil {
		t.Fatalf("applyDefaultMemoryAgentID existing: %v", err)
	}
	if got := memory["agent_id"]; got != int64(456) {
		t.Fatalf("existing agent_id = %#v", got)
	}
}
