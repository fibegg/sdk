package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// registerWaitTool wires fibe_playgrounds_wait. It polls until the playground
// reaches a target status, emitting progress notifications on each tick so
// hosts can display live updates.
func (s *Server) registerWaitTool() {
	s.addTool(&toolImpl{
		name: "fibe_playgrounds_wait", description: "[MODE:DIALOG] Block and poll until a playground reaches a specified target state (has timeout)", tier: tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runWait(ctx, c, args)
		},
	}, mcp.NewTool("fibe_playgrounds_wait",
		mcp.WithDescription("[MODE:DIALOG] Block and poll until a playground reaches a specified target state (has timeout)"),
		mcp.WithNumber("playground_id", mcp.Required(), mcp.Description("Playground ID")),
		mcp.WithString("status", mcp.Required(), mcp.Description("Target playground status, for example running, stopped, or has_changes.")),
		mcp.WithString("timeout", mcp.Description("Max wait duration as Go duration string (e.g. \"5m\"; default: 10m)")),
		mcp.WithString("interval", mcp.Description("Polling interval as Go duration string (default: 3s)")),
	))

}

// runWait implements the polling loop for playground_wait. It emits
// notifications/progress on every tick so hosts see status transitions
// without the agent having to loop.
func (s *Server) runWait(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	id, ok := argInt64(args, "playground_id")
	if !ok {
		return nil, fmt.Errorf("required field 'playground_id' not set")
	}
	target := argString(args, "status")
	if target == "" {
		return nil, fmt.Errorf("required field 'status' not set")
	}
	timeout := parseDuration(argString(args, "timeout"), 10*time.Minute)
	interval := parseDuration(argString(args, "interval"), 3*time.Second)

	// Extract progress token from the MCP request meta (if any). The mcp-go
	// server exposes notification helpers; we use SendNotificationToClient
	// with a progress token so hosts that track it can correlate updates.
	progressToken := progressTokenFromCtx(ctx)

	deadline := time.After(timeout)
	tick := 0
	for {
		tick++
		var (
			status    string
			payload   any
			fetchErr  error
			terminal  bool
			terminalE string
		)
		pg, err := c.Playgrounds.Status(ctx, id)
		fetchErr = err
		if err == nil {
			payload = pg
			status = pg.Status
			if status == "error" || status == "failed" || status == "destroyed" {
				if status != target {
					terminal = true
					terminalE = fibe.PlaygroundTerminalStateError(pg)
				}
			}
		}

		if fetchErr != nil {
			return nil, fetchErr
		}

		if progressToken != nil {
			s.sendProgress(ctx, progressToken, float64(tick), fmt.Sprintf("status: %s", status))
		}

		if status == target {
			return payload, nil
		}
		if terminal {
			return nil, fmt.Errorf("%s", terminalE)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("timeout after %s — last status: %s", timeout, status)
		case <-time.After(interval):
		}
	}
}

// sendProgress emits a notifications/progress message tagged with the
// provided progressToken. Errors are swallowed — missing progress
// notifications should never abort the underlying operation.
func (s *Server) sendProgress(ctx context.Context, token any, progress float64, message string) {
	params := map[string]any{
		"progressToken": token,
		"progress":      progress,
		"message":       message,
	}
	_ = s.mcp.SendNotificationToClient(ctx, "notifications/progress", params)
}

// progressTokenFromCtx retrieves the progress token the client passed in the
// _meta field of the original CallToolRequest. mcp-go stores it on the
// session context.
func progressTokenFromCtx(ctx context.Context) any {
	sess := mcpserver.ClientSessionFromContext(ctx)
	if sess == nil {
		return nil
	}
	// mcp-go v0.48.0 does not expose the meta directly on the base session
	// interface; we read it from the request meta via a typed cast where
	// available. Returning nil disables progress notifications gracefully.
	type metaHolder interface {
		RequestMeta() map[string]any
	}
	if h, ok := sess.(metaHolder); ok {
		if meta := h.RequestMeta(); meta != nil {
			return meta["progressToken"]
		}
	}
	return nil
}

// parseDuration accepts Go duration strings; falls back to def on empty/invalid.
func parseDuration(raw string, def time.Duration) time.Duration {
	if raw == "" {
		return def
	}
	// Be lenient: accept bare integers as seconds.
	if !strings.ContainsAny(raw, "mhs") {
		raw = raw + "s"
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return def
	}
	return d
}
