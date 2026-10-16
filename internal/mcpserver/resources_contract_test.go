package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

func TestAllMCPResourcesAreReadableThroughJSONRPC(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"fixture","data":[],"meta":{"page":1,"per_page":25,"total":0}}`))
	}))
	defer api.Close()

	root := &cobra.Command{Use: "fibe"}
	root.AddCommand(&cobra.Command{Use: "playgrounds", Long: "Playground help fixture"})
	server := New(Config{
		APIKey:                "fixture",
		Domain:                api.URL,
		ToolSet:               "full",
		CobraRoot:             root,
		PipelineCacheSize:     4,
		PipelineCacheEntryMax: 1024,
	})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	session := server.sessionFor(ctx)
	pipelineID, _, err := server.cache.Put(session.sessionID, map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}

	for _, uri := range []string{
		"fibe://me",
		"fibe://status",
		"fibe://schema",
		"fibe://pipeline/schema",
		"fibe://schema/list",
		"fibe://schema/playground",
		"fibe://schema/playground/create",
		"fibe://help/playgrounds",
		"fibe://pipelines/" + pipelineID,
		"fibe://pipelines/missing",
	} {
		uri := uri
		t.Run(strings.TrimPrefix(uri, "fibe://"), func(t *testing.T) {
			request, err := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "resources/read",
				"params":  map[string]any{"uri": uri},
			})
			if err != nil {
				t.Fatal(err)
			}
			response := server.mcp.HandleMessage(ctx, request)
			switch typed := response.(type) {
			case mcp.JSONRPCError:
				t.Fatalf("resource %s failed: %s", uri, typed.Error.Message)
			case *mcp.JSONRPCError:
				t.Fatalf("resource %s failed: %s", uri, typed.Error.Message)
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(encoded), `"result"`) {
				t.Fatalf("resource %s response = %s", uri, encoded)
			}
		})
	}
}

func TestMCPResourceHelpersRejectInvalidInputs(t *testing.T) {
	if path, ok := parseURIPath("fibe://schema/playground", "fibe://schema/"); !ok || path != "playground" {
		t.Fatalf("path=%q ok=%v", path, ok)
	}
	if _, ok := parseURIPath("https://example.com", "fibe://schema/"); ok {
		t.Fatal("unexpected URI prefix match")
	}
	contents := jsonResource("fibe://fixture", make(chan int))
	if len(contents) != 1 {
		t.Fatalf("contents=%d", len(contents))
	}
}
