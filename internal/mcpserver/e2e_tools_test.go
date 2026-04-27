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

func TestE2EMCPTools(t *testing.T) {
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set, skipping e2e tests")
	}
	domain := os.Getenv("FIBE_DOMAIN")
	if domain == "" {
		domain = "localhost:3000" // fallback
	}

	srv := New(Config{
		APIKey:  apiKey,
		Domain:  domain,
		ToolSet: "full",
	})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	ctx := context.Background()
	client := srv.buildBaseClient()

	// 1. Test fibe_status
	t.Run("fibe_status", func(t *testing.T) {
		res, err := srv.dispatcher.dispatch(ctx, "fibe_status", map[string]any{})
		if err != nil {
			t.Fatalf("fibe_status: %v", err)
		}
		m, ok := res.(*fibe.Status)
		if !ok {
			t.Fatalf("expected *fibe.Status, got %T", res)
		}
		if m.Playgrounds.Total < 0 {
			t.Errorf("expected playgrounds total >= 0")
		}
	})

	// 1.5 Test fibe_resource_list
	t.Run("fibe_resource_list", func(t *testing.T) {
		res, err := srv.dispatcher.dispatch(ctx, "fibe_resource_list", map[string]any{
			"resource": "playground",
			"params": map[string]any{
				"page":     1,
				"per_page": 2,
			},
		})
		if err != nil {
			t.Fatalf("fibe_resource_list failed: %v", err)
		}

		list, ok := res.(*fibe.ListResult[fibe.Playground])
		if !ok {
			t.Fatalf("expected *fibe.ListResult[fibe.Playground], got %T", res)
		}
		if list.Meta.Page != 1 {
			t.Errorf("expected page 1, got %d", list.Meta.Page)
		}
	})

	// Setup: Create a temporary agent to test agent-specific tools
	agentName := fmt.Sprintf("e2e-mcp-agent-%d", time.Now().UnixNano())
	ag, err := client.Agents.Create(ctx, &fibe.AgentCreateParams{
		Name:     agentName,
		Provider: fibe.ProviderGemini,
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}
	t.Cleanup(func() {
		client.Agents.Delete(ctx, ag.ID)
	})

	agentIDStr := fmt.Sprintf("%d", ag.ID)

	// 2. Test fibe_update_name
	t.Run("fibe_update_name", func(t *testing.T) {
		// Negative: no agent ID in environment
		os.Unsetenv("FIBE_AGENT_ID")
		_, err := srv.dispatcher.dispatch(ctx, "fibe_update_name", map[string]any{
			"name": "Should Fail",
		})
		if err == nil {
			t.Errorf("expected error when FIBE_AGENT_ID is missing")
		} else if !strings.Contains(err.Error(), "FIBE_AGENT_ID") {
			t.Errorf("expected FIBE_AGENT_ID error, got: %v", err)
		}

		// Positive: correct agent ID
		os.Setenv("FIBE_AGENT_ID", agentIDStr)
		defer os.Unsetenv("FIBE_AGENT_ID")

		newName := agentName + "-updated"
		_, err = srv.dispatcher.dispatch(ctx, "fibe_update_name", map[string]any{
			"name": newName,
		})
		if err != nil {
			t.Fatalf("fibe_update_name failed: %v", err)
		}

		// Verify on backend
		updatedAg, err := client.Agents.Get(ctx, ag.ID)
		if err != nil {
			t.Fatalf("failed to fetch updated agent: %v", err)
		}
		if updatedAg.Name != newName {
			t.Errorf("expected name %q, got %q", newName, updatedAg.Name)
		}
	})

	// 3. Test fibe_artefact_upload
	t.Run("fibe_artefact_upload", func(t *testing.T) {
		os.Setenv("FIBE_AGENT_ID", agentIDStr)
		defer os.Unsetenv("FIBE_AGENT_ID")

		res, err := srv.dispatcher.dispatch(ctx, "fibe_artefact_upload", map[string]any{
			"name":           "e2e-artefact.txt",
			"content_base64": "ZTJlLWFydGVmYWN0LWNvbnRlbnQ=", // base64 of "e2e-artefact-content"
			"description":    "test description",
		})
		if err != nil {
			t.Fatalf("fibe_artefact_upload failed: %v", err)
		}
		
		resMap, ok := res.(*fibe.Artefact)
		if !ok {
			t.Fatalf("expected *fibe.Artefact, got %T", res)
		}
		
		if resMap.ID <= 0 {
			t.Fatalf("expected valid artefact ID in response, got <= 0")
		}
		
		// Wait and fetch artefact
		artefactID := resMap.ID
		art, err := client.Artefacts.Get(ctx, ag.ID, artefactID)
		if err != nil {
			t.Fatalf("failed to fetch artefact: %v", err)
		}
		if art.Name != "e2e-artefact.txt" {
			t.Errorf("expected artefact name 'e2e-artefact.txt', got %v", art.Name)
		}
		if art.Description != nil && *art.Description != "test description" {
			t.Errorf("expected description 'test description', got %v", *art.Description)
		}
	})
	
	// 4. Test fibe_mutter
	t.Run("fibe_mutter", func(t *testing.T) {
		os.Setenv("FIBE_AGENT_ID", agentIDStr)
		defer os.Unsetenv("FIBE_AGENT_ID")

		res, err := srv.dispatcher.dispatch(ctx, "fibe_mutter", map[string]any{
			"type": "thought",
			"body": "e2e mutter test",
		})
		if err != nil {
			t.Fatalf("fibe_mutter failed: %v", err)
		}
		
		resMap, ok := res.(*fibe.Mutter)
		if !ok {
			t.Fatalf("expected *fibe.Mutter, got %T", res)
		}
		
		if resMap.ID == nil || *resMap.ID <= 0 {
			t.Fatalf("expected valid mutter ID in response, got nil or <= 0")
		}
	})

}
