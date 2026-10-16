package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRemovedMCPNamesFailBeforeAuthentication(t *testing.T) {
	server := New(Config{})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name        string
		args        map[string]any
		replacement string
	}{
		{"fibe_marquees_action", map[string]any{}, "fibe_hosts_action"},
		{"fibe_resource_get", map[string]any{"resource": "prop", "id": 1}, "repository"},
		{"fibe_schema", map[string]any{"resource": "playspec"}, "spec"},
		{"fibe_playgrounds_create", map[string]any{"marquee_id": 1}, "host_id"},
	}
	for _, tc := range cases {
		_, err := server.dispatcher.dispatch(context.Background(), tc.name, tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.replacement) {
			t.Fatalf("%s: got %v", tc.name, err)
		}
	}
}

func TestNativeMCPRejectsRemovedNamesWithoutAliases(t *testing.T) {
	server := New(Config{})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	response := server.mcp.HandleMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fibe_marquees_action","arguments":{}}}`))
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "fibe_hosts_action") || !strings.Contains(string(data), "error") {
		t.Fatalf("got %s", data)
	}
	if server.mcp.GetTool("fibe_marquees_action") != nil {
		t.Fatal("removed tool registered as an alias")
	}
}
