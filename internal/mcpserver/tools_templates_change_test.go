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

func TestE2E_TemplatesChangeFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "full", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	mqRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "host",
	})
	if err != nil {
		t.Fatalf("list hosts failed: %v", err)
	}
	hosts := mqRes.(*fibe.ListResult[fibe.Host]).Data
	var activeHostID int64
	for _, m := range hosts {
		if m.Status == "active" {
			activeHostID = m.ID
			break
		}
	}
	if activeHostID == 0 {
		t.Skip("no active hosts available")
	}
	t.Setenv("FIBE_HOST_ID", fmt.Sprintf("%d", activeHostID))
	t.Setenv("HOST_ROOT", t.TempDir())

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
      fibe.gg/port: "80"
      fibe.gg/visibility: "external"
      fibe.gg/subdomain: "$$var__app_name"
`,
		"variables": map[string]any{"app_name": repoName},
	})
	if err != nil {
		t.Fatalf("greenfield create failed: %v", err)
	}
	m := res.(*fibe.GreenfieldResult)

	specID := int(m.Spec.ID)
	templateID := int(*m.ImportTemplate.ID)
	baseVersionID := int(*m.ImportTemplateVersion.ID)
	playgroundID := int(m.Playground.ID)

	previewRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_change", map[string]any{
		"target_type":       "spec",
		"target_id_or_name": specID,
		"mode":              "preview",
		"change_type":       "patch",
		"base_version_id":   baseVersionID,
		"patches":           []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:2", "create_missing": true}},
	})
	if err != nil {
		t.Fatalf("fibe_templates_change patch preview failed: %v", err)
	}
	previewResTyped := previewRes.(*fibe.TemplateVersionPatchResult)
	if (*previewResTyped)["validation"] == nil {
		t.Errorf("expected validation in patch preview")
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_templates_change", map[string]any{
		"target_type":       "spec",
		"target_id_or_name": specID,
		"mode":              "apply",
		"change_type":       "patch",
		"base_version_id":   baseVersionID,
		"patches":           []any{map[string]any{"path": "services.web.image", "op": "set", "value": "nginx:2", "create_missing": true}},
		"confirm_warnings":  true,
	})
	if err != nil {
		t.Fatalf("fibe_templates_change patch apply failed: %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "template_version",
		"operation": "create",
		"payload": map[string]any{
			"template_id_or_name": templateID,
			"template_body":       "services: {}\n",
		},
	})
	if err != nil {
		t.Fatalf("fibe_resource_mutate inline template create failed: %v", err)
	}

	path := filepath.Join(t.TempDir(), "template.yml")
	if err := os.WriteFile(path, []byte("services:\n  web:\n    image: nginx\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "template_version",
		"operation": "create",
		"payload": map[string]any{
			"template_id_or_name": templateID,
			"template_body_path":  "template.yml",
		},
	})
	if err == nil {
		t.Fatal("expected relative path error")
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "template_version",
		"operation": "create",
		"payload": map[string]any{
			"template_id_or_name": templateID,
			"template_body_path":  path,
		},
	})
	if err != nil {
		t.Fatalf("fibe_resource_mutate template_body_path failed: %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_launch", map[string]any{
		"template_id_or_name": templateID,
		"host_id_or_name":     activeHostID,
	})
	if err != nil {
		t.Fatalf("fibe_launch failed: %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_debug", map[string]any{"id_or_name": playgroundID})
	if err != nil {
		t.Fatalf("fibe_playgrounds_debug failed: %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_wait", map[string]any{"id_or_name": playgroundID, "status": "running", "timeout": "10s"})
	if err != nil {
		if strings.Contains(err.Error(), "terminal state: error") || strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout after") {
			t.Skipf("infrastructure failure or timeout during wait, skipping remainder of E2E: %v", err)
		}
		t.Fatalf("fibe_playgrounds_wait failed: %v", err)
	}
}

func TestTemplatesChangeApplyRequiresConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_change", map[string]any{
		"target_type":                "spec",
		"target_id_or_name":          1,
		"mode":                       "apply",
		"change_type":                "switch_existing",
		"target_template_version_id": 2,
	})
	if err == nil || !strings.Contains(err.Error(), "confirm:true") {
		t.Fatalf("expected confirm:true error, got %v", err)
	}
}

func TestTemplatesChangePreviewDoesNotRequireConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_change", map[string]any{
		"target_type":                "spec",
		"target_id_or_name":          1,
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

func TestTemplatesChangeApplyAcceptsConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_templates_change", map[string]any{
		"target_type":                "spec",
		"target_id_or_name":          1,
		"mode":                       "apply",
		"change_type":                "switch_existing",
		"target_template_version_id": 2,
		"confirm":                    true,
	})
	if err == nil {
		t.Fatal("expected mock network error")
	}
	if strings.Contains(err.Error(), "confirm:true") || strings.Contains(err.Error(), "destructive") {
		t.Fatalf("confirm:true should pass the template change gate, got %v", err)
	}
}

func TestTemplatesDevelopAliasIsRemoved(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, ok := srv.dispatcher.lookup("fibe_templates_develop"); ok {
		t.Fatal("fibe_templates_develop should not be registered")
	}
}
