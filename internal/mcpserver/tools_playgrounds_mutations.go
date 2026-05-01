package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPlaygroundMutationTools() {
	s.addTool(&toolImpl{
		name: "fibe_playgrounds_action", description: "[MODE:SIDEEFFECTS] Run one playground lifecycle action: rollout, hard_restart, stop, start, or retry_compose.", tier: tierBrownfield,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "playground_id", "playground_identifier")
			if err != nil {
				return nil, err
			}
			actionType := argString(args, "action_type")
			if actionType == "" {
				return nil, fmt.Errorf("required field 'action_type' not set")
			}
			p := &fibe.PlaygroundActionParams{ActionType: actionType}
			if _, ok := args["force"]; ok {
				force := argBool(args, "force")
				p.Force = &force
			}
			return c.Playgrounds.ActionByIdentifier(ctx, identifier, p)
		},
	}, mcp.NewTool("fibe_playgrounds_action",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Run one playground lifecycle action: rollout, hard_restart, stop, start, or retry_compose."),
		mcp.WithNumber("playground_id", mcp.Description("Playground numeric ID")),
		mcp.WithString("playground_identifier", mcp.Description("Playground numeric ID or slug-safe name")),
		mcp.WithString("action_type", mcp.Required(), mcp.Description("Lifecycle action to perform.")),
		mcp.WithBoolean("force", mcp.Description("Bypass normal state guards when Rails permits forced execution.")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))
	registerPlaygroundDebugTool := func(name, desc string) {
		s.addTool(&toolImpl{
			name: name, description: desc, tier: tierBrownfield,
			annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
			handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
				identifier, err := requiredIdentifier(args, "playground_id", "playground_identifier")
				if err != nil {
					return nil, err
				}
				refresh := true
				if _, ok := args["refresh"]; ok {
					refresh = argBool(args, "refresh")
				}
				mode := argString(args, "mode")
				if mode == "" {
					mode = "summary"
				}
				logsTail := 0
				if raw, ok := argInt64(args, "logs_tail"); ok {
					logsTail = int(raw)
				}
				return c.Playgrounds.DebugWithParamsByIdentifier(ctx, identifier, &fibe.PlaygroundDebugParams{
					Mode:     mode,
					Refresh:  &refresh,
					Service:  argString(args, "service"),
					LogsTail: logsTail,
				})
			},
		}, mcp.NewTool(name,
			mcp.WithDescription(desc+" Defaults to mode=summary and refresh=true for agent diagnostics."),
			mcp.WithNumber("playground_id", mcp.Description("Playground numeric ID")),
			mcp.WithString("playground_identifier", mcp.Description("Playground numeric ID or slug-safe name")),
			mcp.WithString("mode", mcp.Description("summary (default) or full")),
			mcp.WithBoolean("refresh", mcp.Description("Refresh Docker state before reading diagnostics (default: true)")),
			mcp.WithString("service", mcp.Description("Optional Compose service name to focus diagnostics on.")),
			mcp.WithNumber("logs_tail", mcp.Description("Optional number of service log lines to include.")),
		))
	}
	registerPlaygroundDebugTool("fibe_playgrounds_debug", "[MODE:DIALOG] Retrieve comprehensive debugging and diagnostic information for a playground. Use when troubleshooting a failing deployment.")
}
