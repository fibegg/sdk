package mcpserver

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func TestArtefactUploadUsesFilePayload(t *testing.T) {
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

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_artefact_upload", map[string]any{
		"name":           "report.txt",
		"content_base64": "aGVsbG8=", // "hello"
	}); err != nil {
		t.Fatalf("dispatch fibe_artefact_upload: %v", err)
	}
}

func TestArtefactUploadWorkspaceWrite(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	agentName := fmt.Sprintf("test-agent-ws-%d", time.Now().UnixNano())
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

	tmpDir := t.TempDir()
	os.Setenv("FIBE_WORKSPACE_PATH", tmpDir)
	defer os.Unsetenv("FIBE_WORKSPACE_PATH")

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_artefact_upload", map[string]any{
		"name":           "report_workspace.txt",
		"content_base64": "aGVsbG8=",
	}); err != nil {
		t.Fatalf("dispatch fibe_artefact_upload: %v", err)
	}

	content, err := os.ReadFile(tmpDir + "/report_workspace.txt")
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected content: %q", content)
	}
}
