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

// registerWaitTool wires fibe_playgrounds_wait and fibe_tricks_wait. These
// tools poll a resource until it reaches a target status, emitting progress
// notifications on each tick so hosts can display live updates.
func (s *Server) registerWaitTool() {
	s.addTool(&toolImpl{
		name:        "fibe_playgrounds_wait",
		description: "Poll a playground until it reaches a target status",
		tier:        tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runWait(ctx, c, args, waitResourcePlayground)
		},
	}, mcp.NewTool("fibe_playgrounds_wait",
		mcp.WithDescription("Poll a playground until it reaches a target status. Emits MCP progress notifications on every tick. Useful to replace agent-side retry loops — let the server do the polling."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playground ID")),
		mcp.WithString("status", mcp.Required(), mcp.Description("Target status: running, stopped, has_changes, etc.")),
		mcp.WithString("timeout", mcp.Description("Max wait duration as Go duration string (e.g. \"5m\"; default: 10m)")),
		mcp.WithString("interval", mcp.Description("Polling interval as Go duration string (default: 3s)")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_tricks_wait",
		description: "Poll a trick until it reaches a target status (e.g., completed)",
		tier:        tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runWait(ctx, c, args, waitResourceTrick)
		},
	}, mcp.NewTool("fibe_tricks_wait",
		mcp.WithDescription("Poll a trick until it reaches a target status (e.g. completed). Emits MCP progress notifications on every tick."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Trick ID")),
		mcp.WithString("status", mcp.Required(), mcp.Description("Target status: completed, error, running, etc.")),
		mcp.WithString("timeout", mcp.Description("Max wait duration (default: 10m)")),
		mcp.WithString("interval", mcp.Description("Polling interval (default: 3s)")),
	))
}

type waitResource int

const (
	waitResourcePlayground waitResource = iota
	waitResourceTrick
)

// runWait implements the polling loop shared by playground_wait and
// trick_wait. Emits notifications/progress on every tick so hosts see
// status transitions without the agent having to loop.
func (s *Server) runWait(ctx context.Context, c *fibe.Client, args map[string]any, kind waitResource) (any, error) {
	id, ok := argInt64(args, "id")
	if !ok {
		return nil, fmt.Errorf("required field 'id' not set")
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
		switch kind {
		case waitResourcePlayground:
			pg, err := c.Playgrounds.Get(ctx, id)
			fetchErr = err
			if err == nil {
				payload = pg
				status = pg.Status
				if status == "error" || status == "failed" || status == "destroyed" {
					if status != target {
						terminal = true
						terminalE = status
					}
				}
			}
		case waitResourceTrick:
			pg, err := c.Tricks.Get(ctx, id)
			fetchErr = err
			if err == nil {
				payload = pg
				status = pg.Status
				if status == "completed" && target != "completed" {
					terminal = true
					terminalE = status
				}
				if status == "error" || status == "failed" || status == "destroyed" {
					terminal = true
					terminalE = status
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
			return nil, fmt.Errorf("resource reached terminal state: %s", terminalE)
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
	if err != nil || d <= 0 {
		return def
	}
	return d
}
