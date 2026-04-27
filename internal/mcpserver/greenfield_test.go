package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGreenfieldArgsUseEnvMarqueeID(t *testing.T) {
	t.Setenv("FIBE_MARQUEE_ID", "88")

	params, timeout, err := greenfieldArgs(map[string]any{
		"name":      "tower-defence",
		"variables": map[string]any{"app_name": "Tower", "--subdomain": "x1"},
	})
	if err != nil {
		t.Fatalf("greenfieldArgs: %v", err)
	}
	if params.MarqueeID == nil || *params.MarqueeID != 88 {
		t.Fatalf("marquee_id=%v want 88", params.MarqueeID)
	}
	if params.GitProvider != "gitea" {
		t.Fatalf("git_provider=%q want gitea", params.GitProvider)
	}
	if timeout.String() != "10m0s" {
		t.Fatalf("timeout=%s want 10m0s", timeout)
	}
	if params.Variables["app_name"] != "Tower" || params.Variables["subdomain"] != "x1" {
		t.Fatalf("variables=%#v", params.Variables)
	}
}

func TestGreenfieldArgsAcceptTemplateIDAndVersion(t *testing.T) {
	t.Setenv("FIBE_MARQUEE_ID", "88")

	params, _, err := greenfieldArgs(map[string]any{
		"name":        "todo",
		"template_id": float64(347),
		"version":     "v1",
	})
	if err != nil {
		t.Fatalf("greenfieldArgs: %v", err)
	}
	if params.TemplateID == nil || *params.TemplateID != 347 {
		t.Fatalf("template_id=%v want 347", params.TemplateID)
	}
	if params.Version != "v1" {
		t.Fatalf("version=%q want v1", params.Version)
	}
}

func TestGreenfieldArgsAcceptTemplateBody(t *testing.T) {
	t.Setenv("FIBE_MARQUEE_ID", "88")

	params, _, err := greenfieldArgs(map[string]any{
		"name":          "todo",
		"template_body": "services:\n  web:\n    image: nginx\n",
	})
	if err != nil {
		t.Fatalf("greenfieldArgs: %v", err)
	}
	if params.TemplateBody != "services:\n  web:\n    image: nginx\n" {
		t.Fatalf("template_body=%q", params.TemplateBody)
	}
}

func TestGreenfieldArgsAcceptTemplateBodyPath(t *testing.T) {
	t.Setenv("FIBE_MARQUEE_ID", "88")

	body := "services:\n  web:\n    image: nginx\n"
	path := filepath.Join(t.TempDir(), "template.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	params, _, err := greenfieldArgs(map[string]any{
		"name":               "todo",
		"template_body_path": path,
	})
	if err != nil {
		t.Fatalf("greenfieldArgs: %v", err)
	}
	if params.TemplateBody != body {
		t.Fatalf("template_body=%q want %q", params.TemplateBody, body)
	}
}

func TestGreenfieldArgsRejectTemplateBodyWithTemplateID(t *testing.T) {
	t.Setenv("FIBE_MARQUEE_ID", "88")

	_, _, err := greenfieldArgs(map[string]any{
		"name":          "todo",
		"template_id":   float64(347),
		"template_body": "services: {}\n",
	})
	if err == nil || !strings.Contains(err.Error(), "template_body cannot be combined") {
		t.Fatalf("expected template_body conflict error, got %v", err)
	}
}

func TestGreenfieldToolRegisteredAsGreenfield(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	tool, ok := srv.dispatcher.lookup("fibe_greenfield_create")
	if !ok {
		t.Fatal("fibe_greenfield_create not registered")
	}
	if tool.tier != tierGreenfield {
		t.Fatalf("tier=%v want greenfield", tool.tier)
	}
}
