package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestRemoteCommandContractSmoke runs every remote CLI leaf against a
// hermetic API. Snapshot tests freeze the command tree; this catches handlers
// that panic, lose their client wiring, or accidentally depend on local state.
func TestRemoteCommandContractSmoke(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var requests atomic.Int64
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="fixture.txt"`)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                1,
			"name":              "fixture",
			"status":            "running",
			"request_id":        "cli-contract-smoke",
			"data":              []any{},
			"meta":              map[string]any{"page": 1, "per_page": 25, "total": 0},
			"branches":          []any{},
			"services":          []any{},
			"events":            []any{},
			"install_url":       "https://example.com/install",
			"playground_id":     1,
			"playspec_id":       1,
			"props_created":     []any{},
			"template":          map[string]any{"id": 1},
			"latest_version":    map[string]any{"id": 1},
			"latest_version_id": 1,
		})
	}))
	defer api.Close()

	fixturePath := filepath.Join(t.TempDir(), "fixture.txt")
	if err := os.WriteFile(fixturePath, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := RootCmd()
	var executed, reachedAPI int
	for _, command := range commandLeaves(root) {
		path := command.CommandPath()
		if skipRemoteCommandSmoke(path) {
			continue
		}
		t.Run(strings.TrimPrefix(path, "fibe "), func(t *testing.T) {
			flagAPIKey = "contract-fixture"
			flagDomain = api.URL
			flagProfile = ""
			flagOutput = "json"
			flagFromFile = ""
			flagOnly = nil
			configureCommandFixture(command, path, fixturePath)

			ctx, cancel := context.WithTimeout(context.Background(), 35*time.Millisecond)
			defer cancel()
			command.SetContext(ctx)
			setCommandContext(ctx)
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(&output)

			before := requests.Load()
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("CLI handler panicked for contract-derived input: %v", recovered)
				}
			}()
			_ = command.RunE(command, commandFixtureArgs(command))
			executed++
			if requests.Load() > before {
				reachedAPI++
			}
		})
	}
	setCommandContext(context.Background())

	if executed < 140 {
		t.Fatalf("exercised %d remote command handlers, want at least 140", executed)
	}
	if reachedAPI < 100 {
		t.Fatalf("only %d remote command handlers reached the hermetic API", reachedAPI)
	}
}

func commandLeaves(root *cobra.Command) []*cobra.Command {
	var leaves []*cobra.Command
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.RunE != nil {
			leaves = append(leaves, command)
		}
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(root)
	return leaves
}

func skipRemoteCommandSmoke(path string) bool {
	for _, blocked := range []string{
		"fibe auth login", "fibe login", "fibe local playgrounds info", "fibe local playgrounds link",
		"fibe mcp serve", "fibe completion", "fibe docs",
	} {
		if path == blocked {
			return true
		}
	}
	return false
}

func configureCommandFixture(command *cobra.Command, path, fixturePath string) {
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "yes" || flag.Name == "confirm" || flag.Name == "force" {
			_ = flag.Value.Set("true")
		}
		if _, required := flag.Annotations["cobra_annotation_bash_completion_one_required_flag"]; required {
			_ = flag.Value.Set(commandFlagFixture(flag, fixturePath))
		}
	})

	values := map[string]map[string]string{
		"fibe agents add-mounted-file":     {"file": fixturePath},
		"fibe agents create":               {"name": "fixture", "provider": "opencode"},
		"fibe agents download-attachment":  {"to": "-"},
		"fibe agents upload-attachment":    {"file": fixturePath},
		"fibe api-keys create":             {"label": "fixture"},
		"fibe artefacts create":            {"name": "fixture", "file": fixturePath},
		"fibe artefacts download":          {"to": "-"},
		"fibe feedbacks create":            {"body": "fixture"},
		"fibe gitea-repos create":          {"name": "fixture"},
		"fibe github-repos create":         {"name": "fixture"},
		"fibe job-env set":                 {"key": "FIXTURE_KEY", "value": "fixture"},
		"fibe marquees create":             {"name": "fixture", "host": "127.0.0.1", "user": "fixture", "ssh-private-key": "fixture"},
		"fibe mutters create":              {"type": "message", "body": "fixture"},
		"fibe playgrounds create":          {"name": "fixture", "playspec": "1"},
		"fibe playgrounds switch-template": {"template-body": "services: {}", "mode": "preview"},
		"fibe playspecs add-mounted-file":  {"file": fixturePath},
		"fibe playspecs create":            {"name": "fixture", "compose": fixturePath},
		"fibe props attach":                {"repo": "example/fixture"},
		"fibe props create":                {"repo": "https://github.com/example/fixture"},
		"fibe props mirror":                {"url": "https://github.com/example/fixture", "name": "fixture"},
		"fibe secrets create":              {"key": "FIXTURE_KEY", "value": "fixture"},
		"fibe templates create":            {"name": "fixture", "category": "1", "template-body": "services: {}"},
		"fibe templates create-version":    {"template-body": "services: {}"},
		"fibe templates upload-image":      {"file": fixturePath},
		"fibe templates versions create":   {"template-body": "services: {}"},
		"fibe tricks trigger":              {"playspec": "1"},
		"fibe webhooks create":             {"url": "https://example.com/hook", "events": "playground.created"},
	}
	for name, value := range values[path] {
		if flag := command.Flags().Lookup(name); flag != nil {
			_ = flag.Value.Set(value)
		}
	}
}

func commandFlagFixture(flag *pflag.Flag, fixturePath string) string {
	switch flag.Value.Type() {
	case "bool":
		return "true"
	case "duration":
		return "1ms"
	case "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64":
		return "1"
	case "stringSlice", "stringArray":
		if flag.Name == "events" {
			return "playground.created"
		}
		return "fixture"
	}
	switch flag.Name {
	case "file", "path", "image", "compose":
		return fixturePath
	case "provider":
		return "opencode"
	case "action", "action-type":
		return "stop"
	case "mode":
		return "preview"
	case "repo":
		return "example/fixture"
	case "url":
		return "https://example.com/fixture"
	case "to":
		return "-"
	default:
		return "fixture"
	}
}

func commandFixtureArgs(command *cobra.Command) []string {
	fields := strings.Fields(command.Use)
	args := make([]string, 0, len(fields))
	for _, field := range fields[1:] {
		if strings.HasPrefix(field, "[") || strings.HasPrefix(field, "-") || field == "..." {
			continue
		}
		value := "1"
		lower := strings.ToLower(field)
		if strings.Contains(field, "=") {
			value = "FIXTURE_KEY=fixture"
		} else if strings.Contains(lower, "filename") {
			value = "fixture.txt"
		} else if strings.Contains(lower, "type") {
			value = "message"
		} else if strings.Contains(lower, "body") || strings.Contains(lower, "text") {
			value = "fixture"
		}
		args = append(args, value)
	}
	return args
}
