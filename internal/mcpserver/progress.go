package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

func (s *Server) sendClientProgress(ctx context.Context, event fibe.ProgressEvent) {
	progressToken := progressTokenFromCtx(ctx)
	if progressToken == nil {
		return
	}
	progress := float64(event.Attempt)
	if progress <= 0 {
		progress = 1
	}
	s.sendProgress(ctx, progressToken, progress, clientProgressMessage(event))
}

func clientProgressMessage(event fibe.ProgressEvent) string {
	status := strings.TrimSpace(event.Status)
	if status == "" {
		status = "waiting"
	}
	switch event.Operation {
	case "playground_rollout":
		return fmt.Sprintf("rollout status: %s", status)
	case "async":
		if event.RequestID != "" {
			return fmt.Sprintf("async %s: %s", event.RequestID, status)
		}
		return "async status: " + status
	default:
		if event.Operation != "" {
			return fmt.Sprintf("%s: %s", event.Operation, status)
		}
		return "status: " + status
	}
}
