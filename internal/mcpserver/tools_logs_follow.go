package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerLogsFollowTool() {
	s.addTool(&toolImpl{
		name: "fibe_monitor_logs_follow", description: "[MODE:BROWNFIELD] Stream live playground or trick logs as progress notifications. Omitting service streams all services.", tier: tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runLogsFollow(ctx, c, args)
		},
	}, mcp.NewTool("fibe_monitor_logs_follow",
		mcp.WithDescription(`Stream playground or trick logs. Omitting service streams all services. Each new log line becomes an MCP progress notification. Returns when duration elapses or max_lines is reached.

Prefer fibe_playgrounds_logs for a one-shot snapshot. Use follow mode when you need to wait for a specific log pattern to appear. Unpaid Marquees fail with MARQUEE_NOT_FUNDED.`),
		mcp.WithString("id_or_name", mcp.Required(), mcp.Description("Playground or trick numeric ID or slug-safe name")),
		mcp.WithString("target", mcp.Enum("playground", "trick"), mcp.Description("Target type (default: playground).")),
		mcp.WithString("service", mcp.Description("Optional Compose service name, for example web or worker. Omit to stream all services.")),
		mcp.WithNumber("tail", mcp.Description("Initial lines from history (default: 50)")),
		mcp.WithString("duration", mcp.Description("Max follow duration (Go duration, default: 30s)")),
		mcp.WithNumber("max_lines", mcp.Description("Stop after N new lines (default: 500)")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playgrounds_logs_follow", description: "[MODE:BROWNFIELD] Compatibility alias for fibe_monitor_logs_follow with target=playground. Omitting service streams all services.", tier: tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			args["target"] = "playground"
			return s.runLogsFollow(ctx, c, args)
		},
	}, mcp.NewTool("fibe_playgrounds_logs_follow",
		mcp.WithDescription(`Compatibility alias for fibe_monitor_logs_follow with target=playground. Streams playground logs as MCP progress notifications. Omitting service streams all services.`),
		mcp.WithString("id_or_name", mcp.Required(), mcp.Description("Playground numeric ID or slug-safe name")),
		mcp.WithString("service", mcp.Description("Optional Compose service name, for example web or worker. Omit to stream all services.")),
		mcp.WithNumber("tail", mcp.Description("Initial lines from history (default: 50)")),
		mcp.WithString("duration", mcp.Description("Max follow duration (Go duration, default: 30s)")),
		mcp.WithNumber("max_lines", mcp.Description("Stop after N new lines (default: 500)")),
	))
}

func (s *Server) runLogsFollow(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	identifier, err := requiredIdentifier(args, "id_or_name", "")
	if err != nil {
		return nil, err
	}
	service := argString(args, "service")
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

	target := strings.ToLower(strings.TrimSpace(argString(args, "target")))
	var ch <-chan fibe.LogStreamEvent
	var errs <-chan error
	switch target {
	case "", "playground", "playgrounds":
		ch, errs = c.Playgrounds.LogStreamByIdentifier(streamCtx, identifier, service, opts)
	case "trick", "tricks":
		ch, errs = c.Tricks.LogStreamByIdentifier(streamCtx, identifier, service, opts)
	default:
		return nil, fmt.Errorf("target must be playground or trick")
	}

	progressToken := progressTokenFromCtx(ctx)
	var collected []fibe.LogStreamEvent
	var lines []map[string]string
	tick := 0
	for ch != nil || errs != nil {
		select {
		case ev, ok := <-ch:
			if !ok {
				ch = nil
				continue
			}
			collected = append(collected, ev)
			if ev.Type == "log" {
				tick++
				lines = append(lines, map[string]string{"text": ev.Line, "source": ev.Stream, "service": ev.Service})
				if progressToken != nil {
					s.sendProgress(streamCtx, progressToken, float64(tick), ev.Line)
				}
			} else if ev.Type == "status" && progressToken != nil {
				tick++
				s.sendProgress(streamCtx, progressToken, float64(tick), ev.Status)
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return nil, err
			}
		case <-streamCtx.Done():
			ch = nil
			errs = nil
		}
	}

	return map[string]any{
		"id_or_name": identifier,
		"target":     target,
		"service":    service,
		"events":     collected,
		"lines":      lines,
		"count":      len(collected),
		"line_count": len(lines),
	}, nil
}
