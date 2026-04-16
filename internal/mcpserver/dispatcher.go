package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/fibegg/sdk/fibe"
)

// Ensure the fibe import stays compile-visible even when no signature
// references it directly — handler is `func(..., *fibe.Client, ...)`.
var _ *fibe.Client

// dispatcher is the single choke point through which every tool invocation
// passes — including steps of a fibe_pipeline. It enforces:
//
//   - destructive-op gating (confirm:true or --yolo)
//   - per-session auth resolution
//   - idempotency / rate-limit inheritance from the resolved client
//
// The actual work for each tool is performed by a toolImpl registered via
// register(). toolImpls stay free of auth/safety concerns; the dispatcher
// handles that uniformly.
type dispatcher struct {
	srv   *Server
	mu    sync.RWMutex
	tools map[string]*toolImpl
}

func newDispatcher(s *Server) *dispatcher {
	return &dispatcher{srv: s, tools: map[string]*toolImpl{}}
}

// toolImpl is the server-side execution record for a registered tool.
// Tools register themselves via dispatcher.register() and become callable
// both via direct MCP tool calls and as steps inside fibe_pipeline.
type toolImpl struct {
	name        string
	description string
	annotations toolAnnotations
	tier        toolTier // core | full — used by FIBE_MCP_TOOLS gating

	// handler performs the tool's work. It receives a live *fibe.Client
	// already resolved for this session, and the raw tool args. It must
	// return either a result (JSON-serializable) or an error.
	handler toolHandler
}

type toolAnnotations struct {
	ReadOnly    bool
	Destructive bool
	Idempotent  bool
}

type toolTier int

const (
	tierCore toolTier = iota
	tierFull
	tierMeta // pipeline, help, run, auth_set — always registered regardless of tier
)

// toolHandler is the internal handler signature. args is the raw map from
// the MCP request; handlers bind it to a typed struct themselves.
type toolHandler func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error)

func (d *dispatcher) register(t *toolImpl) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tools[t.name] = t
}

func (d *dispatcher) lookup(name string) (*toolImpl, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	t, ok := d.tools[name]
	return t, ok
}

func (d *dispatcher) names() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]string, 0, len(d.tools))
	for k := range d.tools {
		out = append(out, k)
	}
	return out
}

// dispatch runs a tool by name. This is the entry point both for MCP tool
// handlers and for fibe_pipeline steps.
//
// Destructive tools require args["confirm"] == true unless the server is
// running with --yolo. The confirm field is stripped before the handler
// sees args so tool implementations don't have to ignore it.
func (d *dispatcher) dispatch(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := d.lookup(name)
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}

	yolo := d.srv.cfg.Yolo || yoloFromContext(ctx)
	if t.annotations.Destructive && !yolo {
		if !argBool(args, "confirm") {
			return nil, &confirmRequiredError{tool: name}
		}
	}
	// Strip confirm so non-meta handlers don't have to ignore it.
	// Meta tools (fibe_call, fibe_pipeline) need to read and forward
	// confirm into nested invocations, so we leave their args untouched.
	if t.tier != tierMeta {
		if _, ok := args["confirm"]; ok {
			delete(args, "confirm")
		}
	}

	c, err := d.srv.resolveClient(ctx)
	if err != nil {
		return nil, err
	}
	return t.handler(ctx, c, args)
}

// confirmRequiredError is returned when a destructive tool is invoked without
// confirm:true and --yolo is off. Hosts like Claude Code can surface this as
// a prompt to the user.
type confirmRequiredError struct{ tool string }

func (e *confirmRequiredError) Error() string {
	return fmt.Sprintf("tool %q is destructive — pass confirm:true or run server with --yolo", e.tool)
}

func argBool(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true" || x == "1" || x == "yes"
	}
	return false
}

func argString(args map[string]any, key string) string {
	v, ok := args[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func argInt64(args map[string]any, key string) (int64, bool) {
	v, ok := args[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int:
		return int64(x), true
	case int64:
		return x, true
	case string:
		if x == "" {
			return 0, false
		}
		var n int64
		_, err := fmt.Sscanf(x, "%d", &n)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// bindArgs re-marshals the map then unmarshals into a typed destination.
// Slow but uniform and tolerant of JSON number conversions.
func bindArgs(args map[string]any, dest any) error {
	if dest == nil {
		return errors.New("nil destination")
	}
	data, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

