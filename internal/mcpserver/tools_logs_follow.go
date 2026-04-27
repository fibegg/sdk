package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerLogsFollowTool wires fibe_playgrounds_logs_follow. It streams log
// lines as MCP progress notifications and returns only when the deadline or
// max_lines is hit.
//
// The one-shot `fibe_playgrounds_logs` tool (in tools_custom.go) is the
// right call for most agents — follow mode is reserved for the cases where
// an agent genuinely needs to watch a service for a bounded window (e.g.,
// "wait until I see 'listening on :8080' in the logs").
func (s *Server) registerLogsFollowTool() {
	s.addTool(&toolImpl{
		name: "fibe_playgrounds_logs_follow", description: "[MODE:SIDEEFFECTS] Stream the live service logs from a playground as progress notifications", tier: tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runLogsFollow(ctx, c, args)
		},
	}, mcp.NewTool("fibe_playgrounds_logs_follow",
		mcp.WithDescription(`Stream playground service logs. Each new log line becomes an MCP progress notification. Returns when duration elapses or max_lines is reached.

Prefer fibe_playgrounds_logs for a one-shot snapshot. Use follow mode when you need to wait for a specific log pattern to appear.`),
		mcp.WithNumber("playground_id", mcp.Required(), mcp.Description("Playground ID")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Compose service name, for example web or worker.")),
		mcp.WithNumber("tail", mcp.Description("Initial lines from history (default: 50)")),
		mcp.WithString("duration", mcp.Description("Max follow duration (Go duration, default: 30s)")),
		mcp.WithNumber("max_lines", mcp.Description("Stop after N new lines (default: 500)")),
	))

}

func (s *Server) runLogsFollow(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	id, ok := argInt64(args, "playground_id")
	if !ok {
		return nil, fmt.Errorf("required field 'playground_id' not set")
	}
	service := argString(args, "service")
	if service == "" {
		return nil, fmt.Errorf("required field 'service' not set")
	}
	tail := 50
	if t, ok := argInt64(args, "tail"); ok && t > 0 {
		tail = int(t)
	}
	maxLines := 500
	if n, ok := argInt64(args, "max_lines"); ok && n > 0 {
		maxLines = int(n)
	}
	duration := parseDuration(argString(args, "duration"), 30*time.Second)

	streamCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	opts := &fibe.LogsStreamOptions{
		Tail:     tail,
		MaxLines: maxLines,
	}

	ch := c.Playgrounds.LogsStream(streamCtx, id, service, opts)

	progressToken := progressTokenFromCtx(ctx)
	var collected []map[string]string
	tick := 0
	for line := range ch {
		tick++
		collected = append(collected, map[string]string{"text": line.Text, "source": line.Source})
		if progressToken != nil {
			s.sendProgress(streamCtx, progressToken, float64(tick), line.Text)
		}
	}

	return map[string]any{
		"playground_id": id,
		"service":       service,
		"lines":         collected,
		"count":         len(collected),
	}, nil
}
