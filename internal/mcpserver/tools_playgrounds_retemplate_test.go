package mcpserver

import (
	"context"
	"strings"
	"testing"
)

func TestPlaygroundsRetemplateApplyRequiresConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_retemplate", map[string]any{
		"playground_id":       1,
		"template_version_id": 2,
	})
	if err == nil || !strings.Contains(err.Error(), "confirm:true") {
		t.Fatalf("expected confirm:true error, got %v", err)
	}
}

func TestPlaygroundsRetemplatePreviewDoesNotRequireConfirm(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_retemplate", map[string]any{
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

func TestPlaygroundsRetemplateValidatesTargetSelectors(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_retemplate", map[string]any{
		"playground_id": 1,
		"confirm":       true,
	})
	if err == nil {
		t.Fatal("expected validation error when no template target provided")
	}
}
