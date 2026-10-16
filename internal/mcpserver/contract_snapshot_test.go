package mcpserver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMCPToolContract(t *testing.T) {
	server := New(Config{APIKey: "contract-fixture", Domain: "https://fibe.gg", ToolSet: "full"})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	contract := struct {
		Version int        `json:"version"`
		Tools   []ToolInfo `json:"tools"`
	}{Version: 1, Tools: server.AllTools()}
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	path := filepath.Join(mcpRepositoryRoot(t), "contracts", "mcp-tools.json")
	if os.Getenv("UPDATE_CONTRACTS") == "1" {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, want) {
		t.Fatal("MCP tool contract is stale; run UPDATE_CONTRACTS=1 go test ./internal/mcpserver -run TestMCPToolContract")
	}
}

func mcpRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
