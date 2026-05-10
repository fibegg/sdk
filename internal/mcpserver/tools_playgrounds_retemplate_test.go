package mcpserver

import (
	"context"
	"strings"
	"testing"
)

func TestPlaygroundsTransformApplyRequiresConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_transform", map[string]any{
		"playground_id":       1,
		"template_version_id": 2,
	})
	if err == nil || !strings.Contains(err.Error(), "confirm:true") {
		t.Fatalf("expected confirm:true error, got %v", err)
	}
}

func TestPlaygroundsTransformPreviewDoesNotRequireConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_transform", map[string]any{
		"playground_id":       1,
		"template_version_id": 2,
		"mode":                "preview",
	})
	if err == nil {
		t.Fatal("expected mock network error")
	}
	if strings.Contains(err.Error(), "confirm:true") || strings.Contains(err.Error(), "destructive") {
		t.Fatalf("preview should not require confirm, got %v", err)
	}
}

func TestPlaygroundsTransformValidatesTargetSelectors(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_transform", map[string]any{
		"playground_id": 1,
		"confirm":       true,
	})
	if err == nil {
		t.Fatal("expected validation error when no template target provided")
	}
}

func TestBuildRetemplateParamsDefaultsAgentFacingApplyBehavior(t *testing.T) {
	params, err := buildRetemplateParams(map[string]any{
		"playground_id":        7,
		"template_version_id":  22,
		"reuse_existing_props": true,
	}, "apply")
	if err != nil {
		t.Fatalf("buildRetemplateParams: %v", err)
	}
	if params.PlaygroundID != 7 || params.PlaygroundIdentifier != "7" {
		t.Fatalf("unexpected playground identifiers: %#v", params)
	}
	if params.TemplateVersionID != 22 {
		t.Fatalf("template_version_id=%d want 22", params.TemplateVersionID)
	}
	if !params.Wait {
		t.Fatal("transform apply should wait by default")
	}
	if params.ProvisionMissingProps != "gitea" {
		t.Fatalf("provision_missing_props=%q want gitea", params.ProvisionMissingProps)
	}
	if !params.ReuseExistingProps {
		t.Fatal("reuse_existing_props=false want true")
	}
}

func TestPlaygroundsRetemplateAliasIsCallableThroughFibeCall(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_call", map[string]any{
		"tool": "fibe_playgrounds_retemplate",
		"args": map[string]any{
			"playground_id":       1,
			"template_version_id": 2,
		},
		"confirm": true,
	})
	if err == nil {
		t.Fatal("expected mock network error")
	}
	if strings.Contains(err.Error(), "confirm:true") {
		t.Fatalf("fibe_call should forward confirm to legacy alias, got %v", err)
	}
}
