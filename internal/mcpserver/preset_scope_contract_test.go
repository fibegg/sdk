package mcpserver

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPresetsAreAbsentFromMCPDiscoveryAndDispatch(t *testing.T) {
	client := reflect.TypeOf(fibe.Client{})
	for index := 0; index < client.NumField(); index++ {
		if strings.Contains(strings.ToLower(client.Field(index).Name), "preset") {
			t.Fatalf("unexpected dedicated preset SDK service: %s", client.Field(index).Name)
		}
	}
	server := New(Config{APIKey: "scope-fixture", Domain: "https://example.invalid", ToolSet: "full"})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	for _, tool := range server.AllTools() {
		if strings.Contains(strings.ToLower(tool.Name), "preset") {
			t.Fatalf("unexpected preset MCP tool: %s", tool.Name)
		}
	}
	for _, name := range []string{"fibe_preset", "fibe_agent_preset", "fibe_agent_presets_list"} {
		if _, err := server.dispatcher.dispatch(context.Background(), name, nil); err == nil {
			t.Fatalf("unknown preset tool %s unexpectedly dispatched", name)
		}
	}
}
