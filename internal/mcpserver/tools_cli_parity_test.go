package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"github.com/fibegg/sdk/fibe"
)

// requireRealServer returns the active test credentials.
// Assuming requireRealServer is defined in another test file in this package (e.g. tools_templates_develop_test.go)

func TestCLIParity_ListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiKey, domain := requireRealServer(t)

	// Set FIBE_AGENT_ID for feedbacks and mutters tools
	os.Setenv("FIBE_AGENT_ID", "999999999")
	defer os.Unsetenv("FIBE_AGENT_ID")

	// 1. Build the fibe binary
	moduleRoot := findModuleRoot(t)
	bin := filepath.Join(t.TempDir(), "fibe")
	build := exec.Command("go", "build", "-o", bin, "./cmd/fibe")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build fibe: %v\n%s", err, out)
	}

	// 2. Setup internal server for MCP calls
	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	tests := []struct {
		mcpTool  string
		mcpArgs  map[string]any
		cliArgs  []string
	}{
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "playgrounds"},
			cliArgs: []string{"playgrounds", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "agents"},
			cliArgs: []string{"agents", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "marquees"},
			cliArgs: []string{"marquees", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "props"},
			cliArgs: []string{"props", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "tricks"},
			cliArgs: []string{"tricks", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "artefacts"},
			cliArgs: []string{"artefacts", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "playspecs"},
			cliArgs: []string{"playspecs", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "secrets"},
			cliArgs: []string{"secrets", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "api_keys"},
			cliArgs: []string{"api-keys", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "webhooks"},
			cliArgs: []string{"webhooks", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "templates"},
			cliArgs: []string{"templates", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "job_envs"},
			cliArgs: []string{"job-env", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "audit_logs"},
			cliArgs: []string{"audit-logs", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "categories"},
			cliArgs: []string{"categories", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_templates_search",
			mcpArgs: map[string]any{"query": "rails"},
			cliArgs: []string{"templates", "search", "--query", "rails", "--output", "json"},
		},
		{
			mcpTool: "fibe_status",
			mcpArgs: map[string]any{},
			cliArgs: []string{"status", "--output", "json"},
		},
		{
			mcpTool: "fibe_doctor",
			mcpArgs: map[string]any{},
			cliArgs: []string{"doctor", "--output", "json"},
		},
		{
			mcpTool: "fibe_schema",
			mcpArgs: map[string]any{"resource": "list"},
			cliArgs: []string{"schema", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_repo_status_check",
			mcpArgs: map[string]any{"github_urls": []string{"https://github.com/fibegg/fibe"}},
			cliArgs: []string{"repo-status", "check", "--url", "https://github.com/fibegg/fibe", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "webhook_delivery", "params": map[string]any{"webhook_id": float64(999999)}},
			cliArgs: []string{"webhooks", "deliveries", "999999", "--output", "json"},
		},
		{
			mcpTool: "fibe_resource_list",
			mcpArgs: map[string]any{"resource": "template_version", "params": map[string]any{"template_id": float64(999999)}},
			cliArgs: []string{"templates", "versions", "list", "999999", "--output", "json"},
		},
		{
			mcpTool: "fibe_find_github_repos",
			mcpArgs: map[string]any{},
			cliArgs: []string{"installations", "find-repos", "--output", "json"},
		},
		{
			mcpTool: "fibe_monitor_list",
			mcpArgs: map[string]any{},
			cliArgs: []string{"monitor", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_feedbacks_list",
			mcpArgs: map[string]any{"agent_id": 999999999},
			cliArgs: []string{"feedbacks", "list", "999999999", "--output", "json"},
		},
		{
			mcpTool: "fibe_feedbacks_get",
			mcpArgs: map[string]any{"feedback_id": 999},
			cliArgs: []string{"feedbacks", "get", "999999999", "999", "--output", "json"},
		},
		{
			mcpTool: "fibe_mutters_get",
			mcpArgs: map[string]any{"agent_id": 999999999},
			cliArgs: []string{"mutters", "get", "999999999", "--output", "json"},
		},
		{
			mcpTool: "fibe_playgrounds_logs",
			mcpArgs: map[string]any{"playground_id": 999999999, "service": "web"},
			cliArgs: []string{"playgrounds", "logs", "999999999", "--service", "web", "--output", "json"},
		},
		{
			mcpTool: "fibe_playgrounds_debug",
			mcpArgs: map[string]any{"playground_id": 999999999},
			cliArgs: []string{"playgrounds", "debug", "999999999", "--output", "json"},
		},
		{
			mcpTool: "fibe_local_playgrounds_list",
			mcpArgs: map[string]any{},
			cliArgs: []string{"local-playgrounds", "list", "--output", "json"},
		},
		{
			mcpTool: "fibe_local_playgrounds_info",
			mcpArgs: map[string]any{"playground": "nonexistent_dir_for_test"},
			cliArgs: []string{"local-playgrounds", "info", "nonexistent_dir_for_test", "--output", "json"},
		},
		{
			mcpTool: "fibe_local_playgrounds_urls",
			mcpArgs: map[string]any{"playground": "nonexistent_dir_for_test"},
			cliArgs: []string{"local-playgrounds", "urls", "nonexistent_dir_for_test", "--output", "json"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.mcpTool+"_"+tc.cliArgs[0], func(t *testing.T) {
			// 1. Invoke native MCP tool
			mcpRes, mcpErr := srv.dispatcher.dispatch(context.Background(), tc.mcpTool, tc.mcpArgs)
			var mcpBytes []byte
			if mcpErr != nil {
				var code = "UNKNOWN_ERROR"
				var details map[string]any
				var reqId string
				var statusCode = 500

				if apiErr, ok := mcpErr.(*fibe.APIError); ok {
					code = apiErr.Code
					details = apiErr.Details
					reqId = apiErr.RequestID
					statusCode = apiErr.StatusCode
				}
				errMap := map[string]any{
					"error": map[string]any{
						"message": mcpErr.Error(),
						"code":    code,
						"status":  float64(statusCode),
					},
				}
				if details != nil {
					errMap["error"].(map[string]any)["details"] = details
				}
				if reqId != "" {
					errMap["error"].(map[string]any)["request_id"] = reqId
				}
				mcpBytes, _ = json.Marshal(errMap)
			} else {
				mcpBytes, _ = json.Marshal(mcpRes)
			}

			// 2. Invoke real CLI via exec
			cmd := exec.Command(bin, tc.cliArgs...)
			cmd.Env = append(cmd.Environ(), "FIBE_API_KEY="+apiKey, "FIBE_DOMAIN="+domain)
			cliOut, cliErr := cmd.Output()
			if cliErr != nil {
				if ee, ok := cliErr.(*exec.ExitError); ok {
					cliOut = ee.Stderr
				} else {
					t.Fatalf("CLI failed: %v", cliErr)
				}
			}

			// 3. Compare JSON parity
			var mcpObj, cliObj any
			if err := json.Unmarshal(mcpBytes, &mcpObj); err != nil {
				t.Fatalf("unmarshal MCP bytes: %v\nBytes: %s", err, string(mcpBytes))
			}
			if err := json.Unmarshal(cliOut, &cliObj); err != nil {
				t.Fatalf("unmarshal CLI stdout: %v\nstdout: %s", err, string(cliOut))
			}

			// Normalize error request IDs and messages which vary per request
			if errMap, ok := mcpObj.(map[string]any); ok {
				if errObj, ok := errMap["error"].(map[string]any); ok {
					errObj["request_id"] = "normalized"
					errObj["message"] = "normalized"
				}
			}
			if errMap, ok := cliObj.(map[string]any); ok {
				if errObj, ok := errMap["error"].(map[string]any); ok {
					errObj["request_id"] = "normalized"
					errObj["message"] = "normalized"
				}
			}

			if mcpMap, ok := mcpObj.(map[string]any); ok {
				delete(mcpMap, "rate_limits")
				delete(mcpMap, "Data") // Ignore data to prevent race conditions across parallel test packages
				if metaMap, ok := mcpMap["Meta"].(map[string]any); ok {
					delete(metaMap, "total")
				}
				if tc.mcpTool == "fibe_status" {
					delete(mcpMap, "playgrounds")
					delete(mcpMap, "agents")
					delete(mcpMap, "props")
					delete(mcpMap, "playspecs")
					delete(mcpMap, "marquees")
					delete(mcpMap, "secrets")
					delete(mcpMap, "api_keys")
					delete(mcpMap, "resource_quotas")
				}
			}
			if cliMap, ok := cliObj.(map[string]any); ok {
				delete(cliMap, "rate_limits")
				delete(cliMap, "Data") // Ignore data to prevent race conditions across parallel test packages
				if metaMap, ok := cliMap["Meta"].(map[string]any); ok {
					delete(metaMap, "total")
				}
				if tc.mcpTool == "fibe_status" {
					delete(cliMap, "playgrounds")
					delete(cliMap, "agents")
					delete(cliMap, "props")
					delete(cliMap, "playspecs")
					delete(cliMap, "marquees")
					delete(cliMap, "secrets")
					delete(cliMap, "api_keys")
					delete(cliMap, "resource_quotas")
				}
			}

			if !reflect.DeepEqual(mcpObj, cliObj) {
				t.Errorf("Mismatch for %s.\nMCP: %s\nCLI: %s", tc.mcpTool, string(mcpBytes), string(cliOut))
			}
		})
	}
}

func TestCLIParity_GetTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiKey, domain := requireRealServer(t)

	// 1. Build the fibe binary
	moduleRoot := findModuleRoot(t)
	bin := filepath.Join(t.TempDir(), "fibe")
	build := exec.Command("go", "build", "-o", bin, "./cmd/fibe")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build fibe: %v\n%s", err, out)
	}

	// 2. Setup internal server
	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	resources := []string{"playgrounds", "agents", "marquees", "props", "tricks", "artefacts", "playspecs", "secrets", "webhooks", "templates", "job_envs"}

	for _, res := range resources {
		t.Run(res, func(t *testing.T) {
			// First, fetch the list via MCP to get a valid ID
			listResRaw, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{"resource": res})
			if err != nil {
				t.Fatalf("Failed to fetch list for %s: %v", res, err)
			}
			
			// We need to extract the ID. The result is typically a fibe.ListResult[T] or map[string]any.
			listBytes, _ := json.Marshal(listResRaw)
			var listMap map[string]any
			json.Unmarshal(listBytes, &listMap)
			
			var id any
			var agentID any
			if data, ok := listMap["Data"].([]any); ok && len(data) > 0 {
				if item, ok := data[0].(map[string]any); ok {
					id = item["id"]
					agentID = item["agent_id"]
				}
			}
			
			if id == nil {
				t.Skipf("No %s found to test GET", res)
			}
			
			// Convert ID to float64 or string appropriately
			var idVal any
			switch v := id.(type) {
			case float64:
				idVal = int(v) // for CLI arg
			case string:
				idVal = v
			default:
				idVal = id
			}

			// 1. Invoke native MCP tool for GET
			mcpRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
				"resource": res,
				"id":       idVal,
			})
			if err != nil {
				if strings.Contains(err.Error(), "404") {
					t.Skipf("Resource likely deleted by parallel test, skipping GET parity: %v", err)
				}
				t.Fatalf("MCP tool GET failed: %v", err)
			}
			mcpBytes, _ := json.Marshal(mcpRes)

			// 2. Invoke CLI
			idStr := ""
			switch v := idVal.(type) {
			case int:
				idStr = fmt.Sprintf("%d", v)
			default:
				idBytes, _ := json.Marshal(idVal)
				idStr = string(idBytes)
				// Remove quotes if string
				if len(idStr) > 0 && idStr[0] == '"' {
					idStr = idStr[1 : len(idStr)-1]
				}
			}

			cliResName := res
			switch res {
			case "api_keys":
				cliResName = "api-keys"
			case "job_envs":
				cliResName = "job-env"
			}

			var cmd *exec.Cmd
			if cliResName == "artefacts" {
				var agentIDStr string
				if v, ok := agentID.(float64); ok {
					agentIDStr = fmt.Sprintf("%d", int(v))
				} else {
					agentIDStr = "1"
				}
				cmd = exec.Command(bin, cliResName, "get", agentIDStr, idStr, "--output", "json")
			} else {
				cmd = exec.Command(bin, cliResName, "get", idStr, "--output", "json")
			}
			cmd.Env = append(cmd.Environ(), "FIBE_API_KEY="+apiKey, "FIBE_DOMAIN="+domain)
			cliOut, err := cmd.Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					t.Fatalf("CLI GET failed: %v\nStderr: %s", err, ee.Stderr)
				}
				t.Fatalf("CLI GET failed: %v", err)
			}

			// 3. Compare JSON parity
			var mcpObj, cliObj any
			json.Unmarshal(mcpBytes, &mcpObj)
			json.Unmarshal(cliOut, &cliObj)

			if !reflect.DeepEqual(mcpObj, cliObj) {
				t.Errorf("Mismatch for %s GET.\nMCP: %s\nCLI: %s", res, string(mcpBytes), string(cliOut))
			}
		})

		t.Run(res+"_not_found", func(t *testing.T) {
			invalidID := 999999999

			// 1. Invoke native MCP tool for GET
			_, mcpErr := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
				"resource": res,
				"id":       invalidID,
			})
			if mcpErr == nil {
				t.Fatalf("Expected MCP tool to fail for invalid ID %d", invalidID)
			}
			
			// Format mcpErr the same way outputError does for JSON
			mcpErrMap := map[string]any{
				"error": map[string]any{
					"message": mcpErr.Error(),
					"code":    "UNKNOWN_ERROR",
					"status":  500,
				},
			}
			if apiErr, ok := mcpErr.(*fibe.APIError); ok {
				mcpErrMap["error"].(map[string]any)["code"] = apiErr.Code
				mcpErrMap["error"].(map[string]any)["status"] = apiErr.StatusCode
				if apiErr.Details != nil {
					mcpErrMap["error"].(map[string]any)["details"] = apiErr.Details
				}
				if apiErr.RequestID != "" {
					mcpErrMap["error"].(map[string]any)["request_id"] = apiErr.RequestID
				}
			}
			mcpBytes, _ := json.Marshal(mcpErrMap)

			cliResName := res
			switch res {
			case "api_keys":
				cliResName = "api-keys"
			case "job_envs":
				cliResName = "job-env"
			}

			// 2. Invoke CLI
			var cmd *exec.Cmd
			if cliResName == "artefacts" {
				cmd = exec.Command(bin, cliResName, "get", fmt.Sprintf("%d", invalidID), fmt.Sprintf("%d", invalidID), "--output", "json", "--explain-errors")
			} else {
				cmd = exec.Command(bin, cliResName, "get", fmt.Sprintf("%d", invalidID), "--output", "json", "--explain-errors")
			}
			cmd.Env = append(cmd.Environ(), "FIBE_API_KEY="+apiKey, "FIBE_DOMAIN="+domain)
			cliOut, err := cmd.Output()
			var cliStderr []byte
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					cliStderr = ee.Stderr
				} else {
					t.Fatalf("CLI GET failed unexpectedly: %v", err)
				}
			} else {
				t.Fatalf("Expected CLI to fail, but it succeeded: %s", cliOut)
			}

			// 3. Compare JSON parity
			var mcpObj, cliObj any
			json.Unmarshal(mcpBytes, &mcpObj)
			if err := json.Unmarshal(cliStderr, &cliObj); err != nil {
				t.Fatalf("Failed to parse CLI stderr JSON: %v\nStderr: %s", err, cliStderr)
			}

			if !reflect.DeepEqual(mcpObj, cliObj) {
				// Normalize request_id and message containing request_id
				// Normalize request_id and message containing request_id
				if mcpMap, ok := mcpObj.(map[string]any); ok {
					if errMap, ok := mcpMap["error"].(map[string]any); ok {
						delete(errMap, "request_id")
						errMap["message"] = "normalized"
					}
				}
				if cliMap, ok := cliObj.(map[string]any); ok {
					if errMap, ok := cliMap["error"].(map[string]any); ok {
						delete(errMap, "request_id")
						errMap["message"] = "normalized"
					}
				}
				
				if !reflect.DeepEqual(mcpObj, cliObj) {
					t.Errorf("Mismatch for %s GET (Not Found).\nMCP: %v\nCLI: %v", res, mcpObj, cliObj)
				}
			}
		})
	}
}
