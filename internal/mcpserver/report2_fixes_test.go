package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRun2_FibeCallForwardsConfirm verifies that fibe_call(tool: "<destructive>",
// confirm: true) reaches the dispatcher with confirm true (so the destructive
// gate does NOT trip). Regression for Run 2 NEW-3.
func TestRun2_FibeCallForwardsConfirm(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	ctx := context.Background()

	// Top-level confirm should be forwarded into args.
	_, err := srv.dispatcher.dispatch(ctx, "fibe_call", map[string]any{
		"tool":    "fibe_playgrounds_delete",
		"args":    map[string]any{"id": 42},
		"confirm": true,
	})
	if err != nil {
		if _, ok := err.(*confirmRequiredError); ok {
			t.Fatalf("top-level confirm:true on fibe_call should NOT trigger confirm-required gate, got: %v", err)
		}
		if strings.Contains(err.Error(), "confirm:true") || strings.Contains(err.Error(), "destructive") {
			t.Fatalf("top-level confirm:true on fibe_call must bypass destructive gate, got: %v", err)
		}
	}

	// confirm inside args should also work.
	_, err = srv.dispatcher.dispatch(ctx, "fibe_call", map[string]any{
		"tool": "fibe_playgrounds_delete",
		"args": map[string]any{"id": 42, "confirm": true},
	})
	if err != nil {
		if _, ok := err.(*confirmRequiredError); ok {
			t.Fatalf("nested args.confirm:true on fibe_call should NOT trigger confirm-required gate, got: %v", err)
		}
	}

	// Without confirm anywhere, gate must trip.
	_, err = srv.dispatcher.dispatch(ctx, "fibe_call", map[string]any{
		"tool": "fibe_playgrounds_delete",
		"args": map[string]any{"id": 42},
	})
	if err == nil {
		t.Fatalf("fibe_call without confirm on a destructive tool should error")
	}
	if _, ok := err.(*confirmRequiredError); !ok {
		if !strings.Contains(err.Error(), "confirm:true") && !strings.Contains(err.Error(), "destructive") {
			t.Fatalf("expected confirm-required error, got: %v", err)
		}
	}
}

// TestRun2_TemplatesVersionsHelpResolvesCreate verifies that
// fibe_help(path: "templates versions create") returns the create cobra Short,
// not the list cmd's Short. Regression for Run 2 NEW-2 (Option B fix:
// templates restructured into a "versions" subcommand group).
func TestRun2_TemplatesVersionsHelpResolvesCreate(t *testing.T) {
	root := buildRootForHelpTest(t)
	cmd, _, err := root.Find(strings.Fields("templates versions create"))
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected to find a command")
	}
	if cmd.Name() != "create" {
		t.Errorf("expected resolved command name 'create', got %q (full path: %q)", cmd.Name(), cmd.CommandPath())
	}
	if !strings.Contains(strings.ToLower(cmd.Short), "create") || strings.Contains(strings.ToLower(cmd.Short), "list template versions") {
		t.Errorf("expected create-version short, got %q", cmd.Short)
	}
}

// TestRun2_TemplatesVersionsDestroyAcceptsIDAlias verifies that
// aliasField copies 'id' into 'template_id' the way the
// fibe_templates_versions_destroy handler does. Regression for Run 2 P-2.
func TestRun2_TemplatesVersionsDestroyAcceptsIDAlias(t *testing.T) {
	args := map[string]any{"id": 5, "version_id": 12}
	aliasField(args, "template_id", "id")
	if args["template_id"] != 5 {
		t.Fatalf("expected template_id=5 after alias, got %v", args["template_id"])
	}
	tid, ok := argInt64(args, "template_id")
	if !ok || tid != 5 {
		t.Fatalf("expected argInt64 to read template_id=5, got tid=%d ok=%v", tid, ok)
	}

	// Existing canonical wins.
	args = map[string]any{"id": 99, "template_id": 5, "version_id": 12}
	aliasField(args, "template_id", "id")
	if args["template_id"] != 5 {
		t.Fatalf("canonical template_id should win, got %v", args["template_id"])
	}
}

func buildRootForHelpTest(t *testing.T) *cobra.Command {
	t.Helper()
	// Build a minimal cobra tree mirroring the production layout for the
	// templates command, since we can't import the main package from here.
	root := &cobra.Command{Use: "fibe"}
	templates := &cobra.Command{Use: "templates"}
	versions := &cobra.Command{Use: "versions", Short: "Manage template versions"}
	versions.AddCommand(&cobra.Command{Use: "list", Short: "List template versions"})
	versions.AddCommand(&cobra.Command{Use: "create", Short: "Create a new version for an import template"})
	versions.AddCommand(&cobra.Command{Use: "destroy", Short: "Delete a template version"})
	versions.AddCommand(&cobra.Command{Use: "toggle-public", Short: "Toggle a version's public visibility"})
	templates.AddCommand(versions)
	root.AddCommand(templates)
	return root
}
