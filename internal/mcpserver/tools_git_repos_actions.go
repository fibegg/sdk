package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func (s *Server) registerGitRepoActionTools() {
	registerCreate(s, "fibe_github_repos_create", "[MODE:GREENFIELD] Register and connect a new GitHub repository", toolOpts{Tier: tierGreenfield},
		func(ctx context.Context, c *fibe.Client, p *fibe.GitHubRepoCreateParams) (*fibe.GitHubRepo, error) {
			return c.GitHubRepos.Create(ctx, p)
		})
	registerCreate(s, "fibe_gitea_repos_create", "[MODE:GREENFIELD] Register and connect a new Gitea repository and Prop", toolOpts{Tier: tierGreenfield},
		func(ctx context.Context, c *fibe.Client, p *fibe.GiteaRepoCreateParams) (*fibe.GiteaRepo, error) {
			return c.GiteaRepos.Create(ctx, p)
		})
}

func rolloutIDsFromPatchResult(result *fibe.TemplateVersionPatchResult) []int64 {
	if result == nil {
		return nil
	}
	plan, ok := (*result)["playground_rollout_plan"].(map[string]any)
	if !ok {
		return nil
	}
	return anyInt64Slice(plan["rollout"])
}

func anyInt64Slice(raw any) []int64 {
	switch v := raw.(type) {
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			switch x := item.(type) {
			case float64:
				out = append(out, int64(x))
			case int:
				out = append(out, int64(x))
			case int64:
				out = append(out, x)
			}
		}
		return out
	case []int64:
		return v
	case []int:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			out = append(out, int64(item))
		}
		return out
	default:
		return nil
	}
}

func waitForTemplatePatchRollouts(ctx context.Context, c *fibe.Client, ids []int64, timeout time.Duration, diagnose bool) ([]map[string]any, map[string]any) {
	results := make([]map[string]any, 0, len(ids))
	diagnostics := map[string]any{}
	for _, id := range ids {
		result := waitForSingleTemplatePatchRollout(ctx, c, id, timeout)
		results = append(results, result)
		if diagnose && result["success"] != true {
			refresh := true
			debug, err := c.Playgrounds.DebugWithParams(ctx, id, &fibe.PlaygroundDebugParams{Mode: "summary", Refresh: &refresh, LogsTail: 50})
			if err != nil {
				diagnostics[fmt.Sprintf("%d", id)] = map[string]any{"error": err.Error()}
			} else {
				diagnostics[fmt.Sprintf("%d", id)] = debug
			}
		}
	}
	return results, diagnostics
}

func waitForSingleTemplatePatchRollout(ctx context.Context, c *fibe.Client, id int64, timeout time.Duration) map[string]any {
	deadline := time.Now().Add(timeout)
	var lastStatus string
	for {
		status, err := c.Playgrounds.Status(ctx, id)
		if err != nil {
			return map[string]any{"id": id, "success": false, "error": err.Error(), "last_status": lastStatus}
		}
		lastStatus = status.Status
		if status.Status == "running" || status.Status == "completed" {
			return map[string]any{"id": id, "success": true, "status": status.Status}
		}
		if status.Status == "error" || status.Status == "failed" || status.Status == "destroyed" {
			return map[string]any{"id": id, "success": false, "status": status.Status, "failure_diagnostics": status.FailureDiagnostics}
		}
		if time.Now().After(deadline) {
			return map[string]any{"id": id, "success": false, "status": status.Status, "error": fmt.Sprintf("timeout after %s", timeout)}
		}
		select {
		case <-ctx.Done():
			return map[string]any{"id": id, "success": false, "status": status.Status, "error": ctx.Err().Error()}
		case <-time.After(3 * time.Second):
		}
	}
}

// ---------- helpers for file-bearing tools ----------

// decodeFileSource reads either args["content_base64"] or args["content_path"]
// (local filesystem only) and returns an io.Reader suitable for multipart
// upload. One of the two must be provided.
func decodeFileSource(args map[string]any) (io.Reader, error) {
	if b := argString(args, "content_base64"); b != "" {
		data, err := base64.StdEncoding.DecodeString(b)
		if err != nil {
			return nil, fmt.Errorf("invalid content_base64: %w", err)
		}
		return bytes.NewReader(data), nil
	}
	if path := argString(args, "content_path"); path != "" {
		data, err := readLocalFile(path)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}
	return nil, fmt.Errorf("required field missing: pass content_base64 or content_path")
}
