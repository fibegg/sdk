package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestMCPEditionStdioHelper(t *testing.T) {
	if os.Getenv("FIBE_TEST_EDITION_STDIO") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			root := RootCmd()
			root.SetArgs(os.Args[i+1:])
			if err := root.Execute(); err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
	t.Fatal("missing stdio command")
}

func TestMCPEditionReleasedOutput(t *testing.T) {
	// SHA-256 of the literal JSON-RPC result for tools/list, and the
	// literal content text for tools/call, captured from main at a3ff8f2.
	raw, err := os.ReadFile("testdata/mcp-edition-v0.3.12.json")
	require.NoError(t, err)
	var snapshots map[string]map[string]string
	require.NoError(t, json.Unmarshal(raw, &snapshots))

	fixtures := filepath.Join("..", "..", "internal", "mcpserver", "testdata")
	enterprise, err := os.ReadFile(filepath.Join(fixtures, "enterprise-route-manifest.json"))
	require.NoError(t, err)
	community, err := os.ReadFile(filepath.Join(fixtures, "community-route-manifest.json"))
	require.NoError(t, err)
	for _, scenario := range []struct {
		name, mode string
		manifest   []byte
	}{
		{"enterprise", "auto", enterprise},
		{"404", "auto", nil},
		{"community-full", "full", community},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/route-manifest.json" || scenario.mode == "full" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if scenario.manifest == nil {
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write(scenario.manifest)
			}))
			defer api.Close()

			for _, tier := range []string{"full", "core"} {
				t.Run(tier, func(t *testing.T) {
					assertMCPReleasedSnapshot(t, api.URL, tier, scenario.mode, snapshots[tier])
				})
			}
		})
	}
}

func assertMCPReleasedSnapshot(t *testing.T, domain, tier, mode string, snapshot map[string]string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// #nosec G204 -- the executable is this test binary and arguments name the local fixture.
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPEditionStdioHelper$", "--", "mcp", "serve",
		"--domain", domain, "--tools", tier, "--edition", mode)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "FIBE_") && !strings.HasPrefix(entry, "XDG_CONFIG_HOME=") {
			child.Env = append(child.Env, entry)
		}
	}
	child.Env = append(child.Env, "FIBE_TEST_EDITION_STDIO=1", "XDG_CONFIG_HOME="+t.TempDir())
	stdin, err := child.StdinPipe()
	require.NoError(t, err)
	stdout, err := child.StdoutPipe()
	require.NoError(t, err)
	var stderr bytes.Buffer
	child.Stderr = &stderr
	require.NoError(t, child.Start())
	defer func() {
		_ = stdin.Close()
		require.NoError(t, child.Wait(), stderr.String())
	}()

	cases := []struct {
		key, method, params string
	}{
		{"tools", "tools/list", `{}`},
		{"catalog", "tools/call", `{"name":"fibe_tools_catalog","arguments":{"include_schema":true}}`},
		{"schema", "tools/call", `{"name":"fibe_schema","arguments":{"resource":"list"}}`},
		{"registry", "tools/call", `{"name":"fibe_schema","arguments":{}}`},
		{"help", "tools/call", `{"name":"fibe_help","arguments":{}}`},
	}
	in := bufio.NewReader(stdout)
	for i, tc := range cases {
		req := mcp.JSONRPCRequest{JSONRPC: mcp.JSONRPC_VERSION, ID: mcp.NewRequestId(i + 1), Request: mcp.Request{Method: tc.method}}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		data = append(data[:len(data)-1], (`,"params":` + tc.params + "}\n")...)
		_, err = stdin.Write(data)
		require.NoError(t, err)
		line, err := in.ReadBytes('\n')
		require.NoError(t, err)
		var reply struct {
			Result json.RawMessage `json:"result"`
		}
		require.NoError(t, json.Unmarshal(line, &reply))
		require.NotEmpty(t, reply.Result, tc.key)
		content := []byte(reply.Result)
		if tc.method == "tools/call" {
			var result struct {
				IsError bool `json:"isError"`
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			}
			require.NoError(t, json.Unmarshal(content, &result))
			require.False(t, result.IsError)
			require.Len(t, result.Content, 1)
			content = []byte(result.Content[0].Text)
		}
		got := sha256.Sum256(normalizeReleasedMCPDescriptions(tc.key, content))
		require.Equal(t, snapshot[tc.key], hex.EncodeToString(got[:]), tc.key)
	}
}

// Keep the v0.3.12 bytes as a baseline. Restore only the two reviewed
// descriptions before comparing; every other field keeps its original bytes.
func normalizeReleasedMCPDescriptions(surface string, content []byte) []byte {
	const status = "[MODE:DIALOG] Verify repository readiness. Results describe repository access and may include runtime writability or fork and mirror guidance."
	const previousStatus = "[MODE:DIALOG] Verify repository readiness. Enterprise may include writability and fork/mirror guidance; standalone Core verifies Git read access."
	const token = "[MODE:SIDEEFFECTS] Request a GitHub credential for the named repository. The server selects the credential."
	previousToken := "[MODE:SIDEEFFECTS] Get the server-provided GitHub credential for a repository. Enterprise resolves an installation; standalone Core returns its configured credential."
	switch surface {
	case "tools":
		previousToken = "[MODE:SIDEEFFECTS] Get the server-provided GitHub credential for a repository. Enterprise resolves a GitHub App installation; standalone Core returns its configured credential. No installation ID needed."
	case "catalog":
	default:
		return content
	}
	replacements := strings.NewReplacer(
		`"description":`+strconv.Quote(status), `"description":`+strconv.Quote(previousStatus),
		`"description":`+strconv.Quote(token), `"description":`+strconv.Quote(previousToken),
	)
	return []byte(replacements.Replace(string(content)))
}

func TestMCPEditionFlagAndInstall(t *testing.T) {
	t.Setenv("FIBE_MCP_EDITION", "invalid")
	for _, command := range []string{"serve", "install"} {
		t.Run(command, func(t *testing.T) {
			root := RootCmd()
			root.SetArgs([]string{"mcp", command, "--edition", "community"})
			require.ErrorContains(t, root.Execute(), "use auto or full")
			root = RootCmd()
			root.SetArgs([]string{"mcp", command})
			require.ErrorContains(t, root.Execute(), "use auto or full")
		})
	}

	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := RootCmd()
	root.SetArgs([]string{"mcp", "install", "--client", "claude-code", "--project", project, "--edition", "full"})
	require.NoError(t, root.Execute())
	data, err := os.ReadFile(filepath.Join(project, ".mcp.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), `"FIBE_MCP_EDITION": "full"`)

	t.Setenv("FIBE_MCP_EDITION", "auto")
	root = RootCmd()
	root.SetArgs([]string{"mcp", "install", "--client", "claude-code", "--project", project})
	require.NoError(t, root.Execute())
	data, err = os.ReadFile(filepath.Join(project, ".mcp.json"))
	require.NoError(t, err)
	require.Contains(t, string(data), `"FIBE_MCP_EDITION": "auto"`)
}
