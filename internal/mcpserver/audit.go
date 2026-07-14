package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	mu      sync.Mutex
	w       io.Writer
	closers []io.Closer
	path    string
	err     error
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
	dir := filepath.Dir(target)
	base := filepath.Base(target)
	root, err := os.OpenRoot(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: FIBE_MCP_AUDIT_LOG=%s: %v (audit disabled)\n", target, err)
		return nil
	}
	if info, err := root.Lstat(base); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			_ = root.Close()
			fmt.Fprintf(os.Stderr, "WARN: FIBE_MCP_AUDIT_LOG=%s: target must be a regular non-symlink file (audit disabled)\n", target)
			return nil
		}
	} else if !os.IsNotExist(err) {
		_ = root.Close()
		fmt.Fprintf(os.Stderr, "WARN: FIBE_MCP_AUDIT_LOG=%s: %v (audit disabled)\n", target, err)
		return nil
	}
	f, err := root.OpenFile(base, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		_ = root.Close()
		fmt.Fprintf(os.Stderr, "WARN: FIBE_MCP_AUDIT_LOG=%s: %v (audit disabled)\n", target, err)
		return nil
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = root.Close()
		fmt.Fprintf(os.Stderr, "WARN: FIBE_MCP_AUDIT_LOG=%s: tighten permissions: %v (audit disabled)\n", target, err)
		return nil
	}
	return &AuditLog{w: f, closers: []io.Closer{f, root}, path: target}
}

// Close reports any persistent audit-write error together with failures from
// closing the file and its confined filesystem root.
func (a *AuditLog) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var closeErr error
	for _, closer := range a.closers {
		closeErr = errors.Join(closeErr, closer.Close())
	}
	return errors.Join(a.err, closeErr)
}

func (a *AuditLog) write(entry map[string]any) error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return a.err
	}
	enc := json.NewEncoder(a.w)
	a.err = enc.Encode(entry)
	return a.err
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
	sessionHash := sha256.Sum256([]byte(sess))
	entry := map[string]any{
		"ts":          time.Now().UTC().Format(time.RFC3339Nano),
		"session_id":  hex.EncodeToString(sessionHash[:8]),
		"tool":        tool,
		"duration_ms": dur.Milliseconds(),
		"args":        redactSensitive(args),
	}
	if err != nil {
		entry["error"] = limitAuditString(err.Error())
	}
	if writeErr := s.audit.write(entry); writeErr != nil {
		fmt.Fprintf(os.Stderr, "WARN: MCP audit write failed for %s: %v\n", s.audit.path, writeErr)
	}
}

// sensitiveArgKeys are redacted before hitting the audit log. We log
// argument keys (so you can tell what was called with what flags) but
// replace sensitive values with "[redacted]".
var sensitiveArgKeys = map[string]bool{
	"api_key":           true,
	"access_token":      true,
	"authorization":     true,
	"client_secret":     true,
	"compose_yaml":      true,
	"content":           true,
	"content_base64":    true,
	"content_path":      true,
	"credential":        true,
	"credentials":       true,
	"file_content":      true,
	"image_data":        true,
	"password":          true,
	"pem":               true,
	"private_key":       true,
	"raw":               true,
	"refresh_token":     true,
	"secret":            true,
	"source":            true,
	"source_code":       true,
	"template_body":     true,
	"token":             true,
	"workspace_content": true,
}

func redactSensitive(args map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		if isSensitiveAuditKey(k) {
			out[k] = "[redacted]"
			continue
		}
		out[k] = redactAuditValue(v)
	}
	return out
}

func isSensitiveAuditKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
	if sensitiveArgKeys[key] {
		return true
	}
	for _, suffix := range []string{"_api_key", "_token", "_password", "_secret", "_credential", "_credentials", "_private_key", "_pem", "_content", "_content_base64", "_source"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func redactAuditValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return redactSensitive(value)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = redactAuditValue(value[i])
		}
		return out
	case string:
		return limitAuditString(value)
	default:
		return value
	}
}

func limitAuditString(value string) string {
	const maxAuditString = 4096
	if len(value) <= maxAuditString {
		return value
	}
	return value[:maxAuditString] + "...[truncated]"
}
