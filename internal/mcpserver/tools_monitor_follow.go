package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerMonitorFollowTool wires the monitor list/follow tools. Monitor list
// is custom because it returns a dedicated event envelope rather than the
// generic CRUD list type.
func (s *Server) registerMonitorFollowTool() {
	s.addTool(&toolImpl{
		name:        "fibe_monitor_list",
		description: "List agent-produced monitor events",
		tier:        tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runMonitorList(ctx, c, args)
		},
	}, mcp.NewTool("fibe_monitor_list",
		mcp.WithDescription(`List agent events (messages, activities, mutters, artefacts) using standard page/per_page pagination.

Use this for request/response mode. Use fibe_monitor_follow when you need to wait for new events to arrive.`),
		mcp.WithString("agent", mcp.Description("Comma-separated agent IDs. Empty = all accessible.")),
		mcp.WithString("type", mcp.Description("Comma-separated types: message, activity, mutter, artefact")),
		mcp.WithString("since", mcp.Description("Lower bound ISO 8601")),
		mcp.WithString("q", mcp.Description("Full-text search across content")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Page size (default: 25, max: 100)")),
		mcp.WithNumber("content_limit", mcp.Description("Advanced: truncate each payload to N bytes (default: 32768, max: 131072)")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_monitor_follow",
		description: "Stream agent-produced events as MCP progress notifications",
		tier:        tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runMonitorFollow(ctx, c, args)
		},
	}, mcp.NewTool("fibe_monitor_follow",
		mcp.WithDescription(`Follow agent events (messages, activities, mutters, artefacts) by polling the list endpoint.

Each new event becomes an MCP progress notification. Returns when duration elapses or max_events is reached.

Use fibe_monitor_list for a one-shot paginated snapshot; use follow mode when you need to wait for something to happen (e.g., "tell me when the agent emits an artefact").`),
		mcp.WithString("agent", mcp.Description("Comma-separated agent IDs. Empty = all accessible.")),
		mcp.WithString("type", mcp.Description("Comma-separated types: message, activity, mutter, artefact")),
		mcp.WithString("since", mcp.Description("Lower bound ISO 8601 (default: now)")),
		mcp.WithString("q", mcp.Description("Full-text search across content")),
		mcp.WithNumber("content_limit", mcp.Description("Advanced: truncate each payload to N bytes (default: 32768, max: 131072)")),
		mcp.WithNumber("max_events", mcp.Description("Stop after N events (default: 100)")),
		mcp.WithString("duration", mcp.Description("Max follow duration as Go duration (default: 30s, max: 30m)")),
		mcp.WithString("poll_interval", mcp.Description("Polling interval as Go duration (default: 2s)")),
	))
}

func (s *Server) runMonitorList(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	params := &fibe.MonitorListParams{
		AgentIDs: argString(args, "agent"),
		Types:    argString(args, "type"),
		Since:    argString(args, "since"),
		Q:        argString(args, "q"),
	}
	if cl, ok := argInt64(args, "content_limit"); ok && cl > 0 {
		params.ContentLimit = int(cl)
	}
	if page, ok := argInt64(args, "page"); ok && page > 0 {
		params.Page = int(page)
	}
	if perPage, ok := argInt64(args, "per_page"); ok && perPage > 0 {
		params.PerPage = int(perPage)
	}
	return c.Monitor.List(ctx, params)
}

func (s *Server) runMonitorFollow(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	params := &fibe.MonitorListParams{
		AgentIDs: argString(args, "agent"),
		Types:    argString(args, "type"),
		Since:    argString(args, "since"),
		Q:        argString(args, "q"),
	}
	if cl, ok := argInt64(args, "content_limit"); ok && cl > 0 {
		params.ContentLimit = int(cl)
	}

	maxEvents := 100
	if n, ok := argInt64(args, "max_events"); ok && n > 0 {
		maxEvents = int(n)
	}
	duration := parseDuration(argString(args, "duration"), 30*time.Second)
	if duration > 30*time.Minute {
		duration = 30 * time.Minute
	}
	pollInterval := parseDuration(argString(args, "poll_interval"), 2*time.Second)

	streamCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	opts := &fibe.MonitorFollowOptions{
		PollInterval: pollInterval,
		Duration:     duration,
		MaxEvents:    maxEvents,
	}

	events, errs := c.Monitor.Follow(streamCtx, params, opts)

	progressToken := progressTokenFromCtx(ctx)
	var collected []fibe.MonitorEvent
	tick := 0
	for events != nil || errs != nil {
		select {
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			tick++
			collected = append(collected, ev)
			if progressToken != nil {
				summary := fmt.Sprintf("[%s] agent=%d %s", ev.Type, ev.AgentID, ev.ItemID)
				s.sendProgress(streamCtx, progressToken, float64(tick), summary)
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	}

	return map[string]any{
		"events": collected,
		"count":  len(collected),
	}, nil
}
