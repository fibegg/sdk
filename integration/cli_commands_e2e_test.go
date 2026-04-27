package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cliBinPath string
	cliBinOnce sync.Once
)

func initCLIBin(t *testing.T) {
	cliBinOnce.Do(func() {
		// We use a specific temp dir to share the binary across parallel tests
		dir := filepath.Join(os.TempDir(), "fibe-cli-e2e-test")
		os.MkdirAll(dir, 0755)
		cliBinPath = filepath.Join(dir, "fibe")
		cmd := exec.Command("go", "build", "-o", cliBinPath, "../cmd/fibe")
		// The integration test runs from the integration directory, so we build ../cmd/fibe
		err := cmd.Run()
		if err != nil {
			t.Fatalf("failed to build fibe CLI: %v", err)
		}
	})
}

func runCompiledCLI(t *testing.T, args ...string) (string, error) {
	initCLIBin(t)
	// Force JSON output explicitly via flag to avoid any env var ignorance
	args = append(args, "--output", "json")
	cmd := exec.Command(cliBinPath, args...)
	cmd.Env = append(os.Environ(), "FIBE_OUTPUT=json")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func parseResourceID(t *testing.T, jsonOutput string) int64 {
	var v map[string]interface{}
	err := json.Unmarshal([]byte(jsonOutput), &v)
	require.NoError(t, err, "failed to parse json output: %s", jsonOutput)
	idFloat, ok := v["id"].(float64)
	require.True(t, ok, "id field missing or not float64 in: %s", jsonOutput)
	return int64(idFloat)
}

func TestCLI_E2E_Commands(t *testing.T) {
	adminClient(t) // Ensure server is up and env vars are set

	tests := []struct {
		name          string
		resource      string
		createCmdName string
		updateCmdName string
		createArgs    []string
		updateArgs    []string
		missingArgErr string
		cleanupDelete bool
		skipGet       bool
	}{
		{
			name:          "secrets",
			resource:      "secrets",
			createCmdName: "create",
			updateCmdName: "update",
			createArgs:    []string{"--key", uniqueName("key"), "--value", "testval"},
			updateArgs:    []string{"--value", "newval"},
			missingArgErr: "required field 'key' not set",
			cleanupDelete: true,
		},
		{
			name:          "webhooks",
			resource:      "webhooks",
			createCmdName: "create",
			updateCmdName: "update",
			createArgs:    []string{"--url", "https://example.com/hook", "--event", "artefact.created", "--secret", "webhook_secret_key_123"},
			updateArgs:    []string{"--url", "https://example.com/hook2"},
			missingArgErr: "required field 'url' not set",
			cleanupDelete: true,
		},
		{
			name:          "job-env",
			resource:      "job-env",
			createCmdName: "set",
			updateCmdName: "update",
			createArgs:    []string{strings.ReplaceAll(uniqueName("ENV_VAR"), "-", "_") + "=123"},
			updateArgs:    []string{"--value", "456"},
			missingArgErr: "accepts 1 arg",
			cleanupDelete: true,
		},
		{
			name:          "api-keys",
			resource:      "api-keys",
			createCmdName: "create",
			updateCmdName: "update",
			createArgs:    []string{"--label", uniqueName("test-key"), "--scope", "*"},
			updateArgs:    nil, // api-keys don't have update
			missingArgErr: "required field 'label' not set",
			cleanupDelete: true,
			skipGet:       true,
		},
		{
			name:          "templates",
			resource:      "templates",
			createCmdName: "create",
			updateCmdName: "update",
			createArgs:    []string{"--name", uniqueName("test-template"), "--body", "version: '3'", "--description", "cli test description long enough"},
			updateArgs:    []string{"--name", uniqueName("test-template-renamed")},
			missingArgErr: "required field 'name' not set",
			cleanupDelete: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Negative Path: Missing args
			out, err := runCompiledCLI(t, tc.resource, tc.createCmdName)
			require.Error(t, err)
			assert.True(t, strings.Contains(out, tc.missingArgErr) || strings.Contains(out, "required") || strings.Contains(out, "expected KEY=VALUE"), "output should complain about missing args: %s", out)

			// Happy Path: Create
			createCmd := append([]string{tc.resource, tc.createCmdName}, tc.createArgs...)
			out, err = runCompiledCLI(t, createCmd...)
			require.NoError(t, err, "failed to create: %s", out)
			
			id := parseResourceID(t, out)

			// Happy Path: Get
			if !tc.skipGet {
				out, err = runCompiledCLI(t, tc.resource, "get", strconv.FormatInt(id, 10))
				require.NoError(t, err, "failed to get: %s", out)
				assert.Contains(t, out, `"id"`)
			}

			// Happy Path: List
			out, err = runCompiledCLI(t, tc.resource, "list")
			require.NoError(t, err, "failed to list: %s", out)

			// Happy Path: Update
			if tc.updateArgs != nil {
				updateCmd := append([]string{tc.resource, tc.updateCmdName, strconv.FormatInt(id, 10)}, tc.updateArgs...)
				out, err = runCompiledCLI(t, updateCmd...)
				require.NoError(t, err, "failed to update: %s", out)
			}

			// Happy Path: Delete
			if tc.cleanupDelete {
				out, err = runCompiledCLI(t, tc.resource, "delete", strconv.FormatInt(id, 10))
				require.NoError(t, err, "failed to delete: %s", out)
			}
		})
	}
}

// Complex resources that need specific dependencies
func TestCLI_E2E_ComplexCommands(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	t.Run("playgrounds", func(t *testing.T) {
		t.Parallel()
		
		// Negative: no name
		out, err := runCompiledCLI(t, "playgrounds", "create")
		require.Error(t, err)
		assert.True(t, strings.Contains(out, "required field 'name' not set") || strings.Contains(out, "required"), "expected name error: %s", out)

		// Happy: create
		pgName := uniqueName("cli-e2e-pg")
		createArgs := []string{"playgrounds", "create", "--name", pgName, "--playspec-id", strconv.FormatInt(specID, 10)}
		if marqueeID > 0 {
			createArgs = append(createArgs, "--marquee-id", strconv.FormatInt(marqueeID, 10))
		}
		out, err = runCompiledCLI(t, createArgs...)
		require.NoError(t, err, "failed to create playground: %s", out)
		id := parseResourceID(t, out)

		// Happy: update
		out, err = runCompiledCLI(t, "playgrounds", "update", strconv.FormatInt(id, 10), "--name", pgName+"-renamed")
		require.NoError(t, err, "failed to update playground: %s", out)

		// Happy: get
		out, err = runCompiledCLI(t, "playgrounds", "get", strconv.FormatInt(id, 10))
		require.NoError(t, err)

		// Playgrounds debug
		out, err = runCompiledCLI(t, "playgrounds", "debug", strconv.FormatInt(id, 10))
		require.NoError(t, err, "failed to debug playground: %s", out)

		// Playgrounds logs
		out, err = runCompiledCLI(t, "playgrounds", "logs", strconv.FormatInt(id, 10), "--service", "app")
		require.NoError(t, err, "failed to get logs: %s", out)

		// Delete
		out, err = runCompiledCLI(t, "playgrounds", "delete", strconv.FormatInt(id, 10))
		require.NoError(t, err)
	})

	t.Run("playspecs", func(t *testing.T) {
		t.Parallel()
		
		// Negative: no name
		out, err := runCompiledCLI(t, "playspecs", "create")
		require.Error(t, err)
		assert.True(t, strings.Contains(out, "required field 'name' not set") || strings.Contains(out, "required"), "expected name error: %s", out)

		// Happy: create
		psName := uniqueName("cli-e2e-ps")
		composeContent := "version: '3'\nservices:\n  app:\n    image: alpine"
		payload := map[string]interface{}{
			"name":              psName,
			"base_compose_yaml": composeContent,
			"services": []map[string]interface{}{
				{"name": "app", "type": "static"},
			},
		}
		b, _ := json.Marshal(payload)
		tmpPath := filepath.Join(t.TempDir(), "payload.json")
		os.WriteFile(tmpPath, b, 0644)

		out, err = runCompiledCLI(t, "playspecs", "create", "--from-file", tmpPath)
		require.NoError(t, err, "failed to create playspec: %s", out)
		id := parseResourceID(t, out)

		// Update
		out, err = runCompiledCLI(t, "playspecs", "update", strconv.FormatInt(id, 10), "--name", psName+"-renamed")
		require.NoError(t, err)

		// Delete
		out, err = runCompiledCLI(t, "playspecs", "delete", strconv.FormatInt(id, 10))
		require.NoError(t, err)
	})

	t.Run("props", func(t *testing.T) {
		t.Parallel()
		
		// Negative: no repo url
		out, err := runCompiledCLI(t, "props", "create")
		require.Error(t, err)
		assert.True(t, strings.Contains(out, "required field 'url' not set") || strings.Contains(out, "required"), "expected repo url error: %s", out)

		// Happy: create
		out, err = runCompiledCLI(t, "props", "create", "--url", "https://github.com/fibegg/e2e-tests", "--name", uniqueName("cli-e2e-prop"))
		require.NoError(t, err, "failed to create prop: %s", out)
		id := parseResourceID(t, out)

		// Get
		out, err = runCompiledCLI(t, "props", "get", strconv.FormatInt(id, 10))
		require.NoError(t, err)

		// Delete
		out, err = runCompiledCLI(t, "props", "delete", strconv.FormatInt(id, 10))
		require.NoError(t, err)
	})

	t.Run("agents", func(t *testing.T) {
		t.Parallel()

		// Negative: no name
		out, err := runCompiledCLI(t, "agents", "create")
		require.Error(t, err)
		assert.True(t, strings.Contains(out, "required field 'name' not set") || strings.Contains(out, "required"), "expected name error: %s", out)

		// Happy: create
		agentName := uniqueName("cli-e2e-agent")
		out, err = runCompiledCLI(t, "agents", "create", "--name", agentName, "--provider", "gemini")
		require.NoError(t, err, "failed to create agent: %s", out)
		id := parseResourceID(t, out)

		// Happy: update
		out, err = runCompiledCLI(t, "agents", "update", strconv.FormatInt(id, 10), "--name", agentName+"-renamed")
		require.NoError(t, err, "failed to update agent: %s", out)

		// Happy: get
		out, err = runCompiledCLI(t, "agents", "get", strconv.FormatInt(id, 10))
		require.NoError(t, err)

		// Happy: list
		out, err = runCompiledCLI(t, "agents", "list")
		require.NoError(t, err)

		// Mutters Negative: no type
		out, err = runCompiledCLI(t, "mutters", "create", strconv.FormatInt(id, 10))
		require.Error(t, err)
		assert.True(t, strings.Contains(out, "required field 'type' not set") || strings.Contains(out, "required"), "expected type error: %s", out)

		// Mutters Happy: create
		out, err = runCompiledCLI(t, "mutters", "create", strconv.FormatInt(id, 10), "--type", "observation", "--body", "test observation")
		require.NoError(t, err, "failed to create mutter: %s", out)

		// Mutters Happy: get
		out, err = runCompiledCLI(t, "mutters", "get", strconv.FormatInt(id, 10))
		require.NoError(t, err, "failed to get mutters: %s", out)

		// Agents Duplicate
		out, err = runCompiledCLI(t, "agents", "duplicate", strconv.FormatInt(id, 10))
		require.NoError(t, err, "failed to duplicate agent: %s", out)
		dupID := parseResourceID(t, out)
		out, err = runCompiledCLI(t, "agents", "delete", strconv.FormatInt(dupID, 10))
		require.NoError(t, err)

		// Agents runtime-status
		out, err = runCompiledCLI(t, "agents", "runtime-status", strconv.FormatInt(id, 10))
		require.NoError(t, err, "failed to get agent runtime-status: %s", out)

		// Agents send-message
		out, err = runCompiledCLI(t, "agents", "send-message", strconv.FormatInt(id, 10))
		require.Error(t, err) // missing text

		out, err = runCompiledCLI(t, "agents", "send-message", strconv.FormatInt(id, 10), "--text", "hello")
		require.NoError(t, err, "failed to send message to agent: %s", out)

		// Delete
		out, err = runCompiledCLI(t, "agents", "delete", strconv.FormatInt(id, 10))
		require.NoError(t, err)
	})

	t.Run("artefacts", func(t *testing.T) {
		t.Parallel()
		agentName := uniqueName("cli-e2e-artefact-agent")
		out, err := runCompiledCLI(t, "agents", "create", "--name", agentName, "--provider", "gemini")
		require.NoError(t, err)
		agentID := parseResourceID(t, out)

		tmpFile := filepath.Join(t.TempDir(), "test.txt")
		os.WriteFile(tmpFile, []byte("hello world"), 0644)

		out, err = runCompiledCLI(t, "artefacts", "create", strconv.FormatInt(agentID, 10), "--name", "test-artefact", "--file", tmpFile)
		require.NoError(t, err, "failed to create artefact: %s", out)
		artID := parseResourceID(t, out)

		out, err = runCompiledCLI(t, "artefacts", "get", strconv.FormatInt(artID, 10))
		require.NoError(t, err)

		out, err = runCompiledCLI(t, "artefacts", "list", strconv.FormatInt(agentID, 10))
		require.NoError(t, err)

		runCompiledCLI(t, "agents", "delete", strconv.FormatInt(agentID, 10))
	})

	t.Run("templates_versions", func(t *testing.T) {
		t.Parallel()
		tmplName := uniqueName("cli-e2e-tmpl-ver")
		out, err := runCompiledCLI(t, "templates", "create", "--name", tmplName, "--body", "version: '3'")
		require.NoError(t, err)
		tmplID := parseResourceID(t, out)

		tmpFile := filepath.Join(t.TempDir(), "test.yaml")
		os.WriteFile(tmpFile, []byte("version: '3.8'"), 0644)

		out, err = runCompiledCLI(t, "templates", "versions", "create", strconv.FormatInt(tmplID, 10), "--body", "@"+tmpFile)
		require.NoError(t, err, "failed to create template version: %s", out)

		out, err = runCompiledCLI(t, "templates", "versions", "list", strconv.FormatInt(tmplID, 10))
		require.NoError(t, err)

		runCompiledCLI(t, "templates", "delete", strconv.FormatInt(tmplID, 10))
	})

	t.Run("tricks", func(t *testing.T) {
		t.Parallel()
		psName := uniqueName("cli-e2e-job")
		tmpFile := filepath.Join(t.TempDir(), "job.json")
		os.WriteFile(tmpFile, []byte("{\"name\":\""+psName+"\",\"base_compose_yaml\":\"version: '3'\",\"job_mode\":true,\"services\":[{\"name\":\"app\",\"type\":\"static\"}]}"), 0644)
		
		out, err := runCompiledCLI(t, "playspecs", "create", "--from-file", tmpFile)
		require.NoError(t, err)
		psID := parseResourceID(t, out)

		out, err = runCompiledCLI(t, "tricks", "trigger", "--playspec-id", strconv.FormatInt(psID, 10))
		require.NoError(t, err, "failed to trigger trick: %s", out)
		trickID := parseResourceID(t, out)

		out, err = runCompiledCLI(t, "tricks", "get", strconv.FormatInt(trickID, 10))
		require.NoError(t, err)

		out, err = runCompiledCLI(t, "tricks", "list", "--playspec-id", strconv.FormatInt(psID, 10))
		require.NoError(t, err)

		out, err = runCompiledCLI(t, "tricks", "delete", strconv.FormatInt(trickID, 10))
		require.NoError(t, err)

		runCompiledCLI(t, "playspecs", "delete", strconv.FormatInt(psID, 10))
	})

	t.Run("gitea_repos", func(t *testing.T) {
		t.Parallel()
		repoName := uniqueName("cli-e2e-gitea")
		out, err := runCompiledCLI(t, "gitea-repos", "create", "--name", repoName, "--private")
		require.NoError(t, err, "failed to create gitea repo: %s", out)
	})

	t.Run("monitor", func(t *testing.T) {
		t.Parallel()
		out, err := runCompiledCLI(t, "monitor", "list", "--per-page", "1")
		require.NoError(t, err, "failed to list monitor: %s", out)
	})

	t.Run("misc_commands", func(t *testing.T) {
		t.Parallel()

		out, err := runCompiledCLI(t, "status")
		require.NoError(t, err, "failed to get status: %s", out)

		out, err = runCompiledCLI(t, "doctor")
		require.NoError(t, err, "failed to run doctor: %s", out)

		out, err = runCompiledCLI(t, "schema", "--resource", "agent", "--operation", "create")
		require.NoError(t, err, "failed to run schema: %s", out)
	})
}
