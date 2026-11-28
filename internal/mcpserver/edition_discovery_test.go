package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Copied from fibe-community/contracts/generated/self-hosted-v1/community/
// and fibe-enterprise/contracts/generated/self-hosted-v1/ respectively.
func fixtureManifest(t *testing.T, edition string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + edition + "-route-manifest.json")
	require.NoError(t, err)
	return data
}

type editionTestAPI struct {
	server   *httptest.Server
	probes   atomic.Int32
	requests atomic.Int32
}

func newEditionTestAPI(t *testing.T, status int, body []byte, slow bool) *editionTestAPI {
	t.Helper()
	api := &editionTestAPI{}
	api.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/route-manifest.json" {
			api.requests.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[],"meta":{"page":1,"per_page":25,"total":0}}`))
			return
		}
		api.probes.Add(1)
		if r.Method != "GET" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Fibe-API-Key") != "" {
			t.Error("route discovery must use an anonymous GET")
		}
		if slow {
			<-r.Context().Done()
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(api.server.Close)
	return api
}

func newEditionTestServer(t *testing.T, domain, mode, tools string) *Server {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Domain = domain
	cfg.Edition = mode
	cfg.ToolSet = tools
	cfg.APIKey = "fixture"
	cfg.Yolo = true
	cfg.CobraRoot = &cobra.Command{Use: "fibe"}
	templates := &cobra.Command{Use: "templates", Short: "Templates"}
	templates.AddCommand(&cobra.Command{Use: "search", Run: func(*cobra.Command, []string) { t.Error("unavailable command was invoked") }})
	cfg.CobraRoot.AddCommand(templates)
	server := New(cfg)
	require.NoError(t, server.RegisterAll())
	server.baseCli = server.buildBaseClient()
	server.startEditionDiscovery()
	return server
}

func editionListedNames(t *testing.T, s *Server, ctx context.Context) []string {
	t.Helper()
	response := s.mcp.HandleMessage(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	data, err := json.Marshal(response)
	require.NoError(t, err)
	var result struct {
		Result mcp.ListToolsResult `json:"result"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	require.NotEmpty(t, result.Result.Tools)
	var names []string
	for _, tool := range result.Result.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

func editionDispatch(t *testing.T, s *Server, name string, args map[string]any) string {
	t.Helper()
	result, err := s.dispatcher.dispatch(context.Background(), name, args)
	require.NoError(t, err)
	data, err := json.Marshal(result)
	require.NoError(t, err)
	return string(data)
}

func TestMCPEditionDiscovery(t *testing.T) {
	community := fixtureManifest(t, "community")
	enterprise := fixtureManifest(t, "enterprise")
	filteredNames := []string{
		"fibe_agent_defaults_get", "fibe_agent_defaults_reset", "fibe_agent_defaults_update", "fibe_artefact_upload",
		"fibe_env_pack_attachments_get", "fibe_feedbacks_get", "fibe_feedbacks_list", "fibe_gitea_repos_create",
		"fibe_memorize", "fibe_mutter", "fibe_mutters_get", "fibe_playgrounds_switch_template",
		"fibe_team_git_installation_connect", "fibe_team_git_installation_remove", "fibe_team_git_installations_list",
		"fibe_team_principal_create", "fibe_team_principal_revoke", "fibe_team_principals_list",
		"fibe_team_repository_operation_adopt", "fibe_team_repository_operation_reconcile", "fibe_team_repository_operations_list",
		"fibe_team_runtime_principal_recover", "fibe_templates_search",
	}
	testCases := []struct {
		name   string
		status int
		body   []byte
		slow   bool
	}{
		{"community", 200, community, false},
		{"enterprise", 200, enterprise, false},
		{"404", 404, nil, false},
		{"invalid", 200, []byte(`{"edition":"community","routes":[{"path":"/api/status","methods":["GET"]}]}`), false},
		{"timeout", 200, community, true},
		{"future-edition", 200, []byte(strings.Replace(string(enterprise), `"enterprise"`, `"future"`, 1)), false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			api := newEditionTestAPI(t, tc.status, tc.body, tc.slow)
			full := newEditionTestServer(t, api.server.URL, "full", "full")
			require.Zero(t, api.probes.Load())
			start := time.Now()
			auto := newEditionTestServer(t, api.server.URL, "auto", "full")
			require.Less(t, time.Since(start), 2*time.Second)
			require.EqualValues(t, 1, api.probes.Load())

			fullNames := editionListedNames(t, full, ctx)
			autoNames := editionListedNames(t, auto, ctx)
			var omitted []string
			for _, name := range fullNames {
				if !containsName(autoNames, name) {
					omitted = append(omitted, name)
				}
			}
			t.Logf("advertised=%d full=%d omitted=%v", len(autoNames), len(fullNames), omitted)

			catalogArgs := map[string]any{"include_schema": true}
			schemaArgs := map[string]any{"resource": "list"}
			catalog := editionDispatch(t, auto, "fibe_tools_catalog", catalogArgs)
			schema := editionDispatch(t, auto, "fibe_schema", schemaArgs)
			help := editionDispatch(t, auto, "fibe_help", nil)

			if tc.name == "community" {
				require.Equal(t, filteredNames, omitted)
				require.NotContains(t, catalog, `"name":"fibe_templates_search"`)
				require.NotContains(t, schema, `"name":"template"`)
				require.Contains(t, schema, `"name":"agent"`)
				require.NotContains(t, help, "Templates")
				for _, result := range []string{catalog, schema, help} {
					require.Contains(t, result, communityNotice)
				}
				testCommunityRefusals(t, auto)
				require.Zero(t, api.requests.Load())
				// Full mode leaves the operation on its normal API path.
				editionDispatch(t, full, "fibe_resource_list", map[string]any{"resource": "templates"})
			} else {
				require.Empty(t, omitted)
				require.Equal(t, editionDispatch(t, full, "fibe_tools_catalog", catalogArgs), catalog)
				require.Equal(t, editionDispatch(t, full, "fibe_schema", schemaArgs), schema)
				require.Equal(t, editionDispatch(t, full, "fibe_help", nil), help)
				editionDispatch(t, auto, "fibe_resource_list", map[string]any{"resource": "templates"})
			}
			require.EqualValues(t, 1, api.requests.Load())
			require.EqualValues(t, 1, api.probes.Load(), "a successful or failed probe must be cached")
		})
	}
}

func testCommunityRefusals(t *testing.T, s *Server) {
	t.Helper()
	listTemplates := map[string]any{"resource": "templates"}
	step := func(tool string, args map[string]any) map[string]any {
		return map[string]any{"tool": tool, "args": args}
	}
	for _, refusal := range []struct {
		tool string
		args map[string]any
		op   string
	}{
		{"fibe_call", map[string]any{"tool": "fibe_templates_search"}, "templates search"},
		{"fibe_call", step("fibe_resource_list", listTemplates), "template list"},
		{"fibe_resource_list", listTemplates, "template list"},
		{"fibe_resource_get", map[string]any{"resource": "secrets", "id_or_key": "example"}, "secret get"},
		{"fibe_resource_delete", map[string]any{"resource": "memory", "id": 1}, "memory delete"},
		{"fibe_resource_mutate", map[string]any{"resource": "webhook", "operation": "create", "payload": map[string]any{}}, "webhook create"},
		{"fibe_run", map[string]any{"args": []any{"templates", "search"}}, "templates search"},
		{"fibe_pipeline", map[string]any{"steps": []any{step("fibe_status", nil), step("fibe_resource_list", listTemplates)}}, "template list"},
		{"fibe_schema", map[string]any{"resource": "template", "operation": "list"}, "template list"},
		{"fibe_help", map[string]any{"path": "templates search"}, "templates search"},
	} {
		_, err := s.dispatcher.dispatch(context.Background(), refusal.tool, refusal.args)
		want := refusal.op + " is not available: this server runs the Community edition."
		require.EqualError(t, err, want)
		result := toolResultFromError(refusal.tool, err)
		require.True(t, result.IsError)
		require.Equal(t, want, result.Content[0].(mcp.TextContent).Text)
	}
}

func TestMCPEditionChangesWithProfile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	community := newEditionTestAPI(t, 200, fixtureManifest(t, "community"), false)
	enterprise := newEditionTestAPI(t, 200, fixtureManifest(t, "enterprise"), false)
	s := newEditionTestServer(t, enterprise.server.URL, "auto", "core")
	ctx := context.Background()
	require.NoError(t, fibe.NewAuthProfileStore(fibe.DefaultAuthProfilePath()).SetProfile("community-fixture", community.server.URL))
	require.NoError(t, fibe.NewAuthProfileStore(fibe.DefaultAuthProfilePath()).SetProfile("enterprise-fixture", enterprise.server.URL))

	before := editionListedNames(t, s, ctx)
	require.Contains(t, before, "fibe_templates_search")
	for _, profile := range []string{"community-fixture", "enterprise-fixture", "community-fixture"} {
		editionDispatch(t, s, "fibe_auth_use", map[string]any{"profile": profile, "validate": false})
		names := editionListedNames(t, s, ctx)
		if profile == "community-fixture" {
			require.NotContains(t, names, "fibe_templates_search")
			_, err := s.dispatcher.dispatch(ctx, "fibe_resource_list", map[string]any{"resource": "template"})
			require.ErrorContains(t, err, "this server runs the Community edition")
		} else {
			require.Equal(t, before, names)
		}
	}
	require.EqualValues(t, 1, community.probes.Load())
	require.EqualValues(t, 1, enterprise.probes.Load())
	require.Zero(t, community.requests.Load())
	require.Zero(t, enterprise.requests.Load())
}

func TestMCPEditionToolAndResourceMappings(t *testing.T) {
	api := newEditionTestAPI(t, 200, fixtureManifest(t, "community"), false)
	s := newEditionTestServer(t, api.server.URL, "auto", "full")
	m := s.sessionManifest(context.Background())
	for _, name := range s.dispatcher.names() {
		tool, _ := s.dispatcher.lookup(name)
		require.NotEmpty(t, tool.routes, name)
		if !m.offersTool(tool) {
			t.Logf("hidden tool %s: %s", name, formatOperationRoutes(tool.routes))
		}
	}
	for resource, operations := range resourceschema.Registry() {
		for operation := range operations {
			routes := resourceOperationRoutes(resource, operation)
			require.NotEmpty(t, routes, "%s.%s", resource, operation)
			if !m.offers(routes) {
				t.Logf("hidden operation %s.%s: %s", resource, operation, formatOperationRoutes(routes))
			}
		}
	}
}

func TestMCPEditionPathTemplateMatching(t *testing.T) {
	route := routeKey("GET", "/api/agents/{agent}/uploads/{name}")
	require.Equal(t, route, routeKey("GET", "/api/agents/{id}/uploads/{filename}"))
	require.NotEqual(t, route, routeKey("POST", "/api/agents/{id}/uploads/{filename}"))
	require.NotEqual(t, route, routeKey("GET", "/api/agents/42/uploads/{filename}"))
	require.NotEqual(t, route, routeKey("GET", "/api/agents/{id}/uploads/{filename}/"))
	require.True(t, (&routeManifest{}).offers("UnknownService.Unknown"))
}

func containsName(names []string, name string) bool {
	i := sort.SearchStrings(names, name)
	return i < len(names) && names[i] == name
}

func formatOperationRoutes(routes operationRoutes) string {
	var out []string
	for _, clause := range strings.Fields(string(routes)) {
		for _, id := range strings.Split(clause, "+") {
			route := operationRoute(id)
			out = append(out, fmt.Sprintf("%s %s", route.method, route.path))
		}
	}
	return strings.Join(out, ", ")
}
