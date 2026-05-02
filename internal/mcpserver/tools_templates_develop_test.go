package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func TestE2E_TemplatesDevelopFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "full", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	mqRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "marquee",
	})
	if err != nil {
		t.Fatalf("list marquees failed: %v", err)
	}
	marquees := mqRes.(*fibe.ListResult[fibe.Marquee]).Data
	var activeMarqueeID int64
	for _, m := range marquees {
		if m.Status == "active" {
			activeMarqueeID = m.ID
			break
		}
	}
	if activeMarqueeID == 0 {
		t.Skip("no active marquees available")
	}
	t.Setenv("FIBE_MARQUEE_ID", fmt.Sprintf("%d", activeMarqueeID))
	t.Setenv("PLAYROOMS_ROOT", t.TempDir())

	// 1. Create a greenfield app to get a real template, version, playspec, and playground
	repoName := fmt.Sprintf("mcp-test-dev-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_greenfield_create", map[string]any{
		"name":         repoName,
		"wait_timeout": "0s",
		"template_body": `
x-fibe.gg:
  variables:
    app_name:
      name: "App name"
      required: true
services:
  web:
    image: nginx:alpine
    environment:
      APP_NAME: $$var__app_name
    labels:
      fibe.gg/repo_url: "https://github.com/fibegg/demo-backend"
      fibe.gg/source_mount: "/app"
      fibe.gg/expose: "external:80"
      fibe.gg/subdomain: "$$var__app_name"
`,
		"variables": map[string]any{"app_name": repoName},
	})
	if err != nil {
		t.Fatalf("greenfield create failed: %v", err)
	}
	m := res.(*fibe.GreenfieldResult)

	playspecID := int(m.Playspec.ID)
	templateID := int(*m.ImportTemplate.ID)
	baseVersionID := int(*m.ImportTemplateVersion.ID)
	playgroundID := int(m.Playground.ID)

	// 2. Test fibe_templates_develop (patch preview)
	previewRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_develop", map[string]any{
		"target_type":     "playspec",
		"target_id":       playspecID,
		"mode":            "preview",
		"change_type":     "patch",
		"base_version_id": baseVersionID,
		"patches":         []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:2", "create_missing": true}},
	})
	if err != nil {
		t.Fatalf("fibe_templates_develop patch preview failed: %v", err)
	}
	previewResTyped := previewRes.(*fibe.TemplateVersionPatchResult)
	if (*previewResTyped)["validation"] == nil {
		t.Errorf("expected validation in patch preview")
	}

	// 3. Test fibe_templates_develop (patch apply with auto_switch)
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_templates_develop", map[string]any{
		"target_type":      "playspec",
		"target_id":        playspecID,
		"mode":             "apply",
		"change_type":      "patch",
		"base_version_id":  baseVersionID,
		"patches":          []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:2", "create_missing": true}},
		"confirm_warnings": true,
	})
	if err != nil {
		t.Fatalf("fibe_templates_develop patch apply failed: %v", err)
	}

	// 4. Test creating template version with inline template_body
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "template_version",
		"operation": "create",
		"payload": map[string]any{
			"template_id":   templateID,
			"template_body": "services: {}\n",
		},
	})
	if err != nil {
		t.Fatalf("fibe_resource_mutate inline template create failed: %v", err)
	}

	// 5. Test creating template version with template_body_path
	path := filepath.Join(t.TempDir(), "template.yml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	// 5.a Rejects relative path
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "template_version",
		"operation": "create",
		"payload": map[string]any{
			"template_id":        templateID,
			"template_body_path": "template.yml",
		},
	})
	if err == nil {
		t.Fatal("expected relative path error")
	}

	// 5.b Accepts absolute path
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "template_version",
		"operation": "create",
		"payload": map[string]any{
			"template_id":        templateID,
			"template_body_path": path,
		},
	})
	if err != nil {
		t.Fatalf("fibe_resource_mutate template_body_path failed: %v", err)
	}

	// 6. Test launch with explicit template ID
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_templates_launch", map[string]any{
		"template_id": templateID,
	})
	if err != nil {
		t.Fatalf("fibe_templates_launch failed: %v", err)
	}

	// 7. Test debug tool
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_debug", map[string]any{"playground_id": playgroundID})
	if err != nil {
		t.Fatalf("fibe_playgrounds_debug failed: %v", err)
	}

	// 8. Test wait tool
	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_wait", map[string]any{"playground_id": playgroundID, "status": "running", "timeout": "10s"})
	if err != nil {
		if strings.Contains(err.Error(), "terminal state: error") || strings.Contains(err.Error(), "context deadline exceeded") {
			t.Skipf("infrastructure failure or timeout during wait, skipping remainder of E2E: %v", err)
		}
		t.Fatalf("fibe_playgrounds_wait failed: %v", err)
	}
}

func TestTemplatesDevelopApplyRequiresConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_develop", map[string]any{
		"target_type":                "playspec",
		"target_id":                  1,
		"mode":                       "apply",
		"change_type":                "switch_existing",
		"target_template_version_id": 2,
	})
	if err == nil || !strings.Contains(err.Error(), "confirm:true") {
		t.Fatalf("expected confirm:true error, got %v", err)
	}
}

func TestTemplatesDevelopPreviewDoesNotRequireConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_develop", map[string]any{
		"target_type":                "playspec",
		"target_id":                  1,
		"mode":                       "preview",
		"change_type":                "switch_existing",
		"target_template_version_id": 2,
	})
	if err == nil {
		t.Fatal("expected mock network error")
	}
	if strings.Contains(err.Error(), "confirm:true") || strings.Contains(err.Error(), "destructive") {
		t.Fatalf("preview should not require confirm, got %v", err)
	}
}

func TestTemplatesDevelopApplyAcceptsConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_develop", map[string]any{
		"target_type":                "playspec",
		"target_id":                  1,
		"mode":                       "apply",
		"change_type":                "switch_existing",
		"target_template_version_id": 2,
		"confirm":                    true,
	})
	if err == nil {
		t.Fatal("expected mock network error")
	}
	if strings.Contains(err.Error(), "confirm:true") || strings.Contains(err.Error(), "destructive") {
		t.Fatalf("confirm:true should pass the template develop gate, got %v", err)
	}
}
