package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// AuditLog emits one JSON-line record per tool call. Useful for debugging
// agent journeys and for compliance in multi-tenant deployments.
//
// Enable by setting FIBE_MCP_AUDIT_LOG=<path> (or "stderr") before starting
// the server. The file is opened in append mode and never rotated by us —
// callers are expected to wire logrotate if they care.
type AuditLog struct {
	mu   sync.Mutex
	w    io.Writer
	path string
}

// newAuditLog resolves the target from the env var. Returns nil when
// logging is disabled so callers pay zero cost on the hot path.
func newAuditLog() *AuditLog {
	target := os.Getenv("FIBE_MCP_AUDIT_LOG")
	if target == "" {
		return nil
	}
	if target == "stderr" {
		return &AuditLog{w: os.Stderr, path: "stderr"}
	}
	f, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: FIBE_MCP_AUDIT_LOG=%s: %v (audit disabled)\n", target, err)
		return nil
	}
	return &AuditLog{w: f, path: target}
}

func (a *AuditLog) write(entry map[string]any) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	enc := json.NewEncoder(a.w)
	_ = enc.Encode(entry)
}

// auditLog writes a structured line per tool invocation. Args values are
// redacted for keys that look sensitive (api_key, secret, token, password).
func (s *Server) auditLog(ctx context.Context, tool string, args map[string]any, err error, dur time.Duration) {
	if s.audit == nil {
		return
	}
	sess := "default"
	if cs := mcpserver.ClientSessionFromContext(ctx); cs != nil {
		sess = cs.SessionID()
	}
	entry := map[string]any{
		"ts":         time.Now().UTC().Format(time.RFC3339Nano),
		"session_id": sess,
		"tool":       tool,
		"duration_ms": dur.Milliseconds(),
		"args":       redactSensitive(args),
	}
	if err != nil {
		entry["error"] = err.Error()
	}
	s.audit.write(entry)
}

// sensitiveArgKeys are redacted before hitting the audit log. We log
// argument keys (so you can tell what was called with what flags) but
// replace sensitive values with "[redacted]".
var sensitiveArgKeys = map[string]bool{
	"api_key":         true,
	"secret":          true,
	"token":           true,
	"password":        true,
	"content_base64":  true, // may be secrets-in-files
	"image_data":      true,
}

func redactSensitive(args map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		if sensitiveArgKeys[k] {
			out[k] = "[redacted]"
			continue
		}
		out[k] = v
	}
	return out
}
