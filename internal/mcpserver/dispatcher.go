package mcpserver

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/fibegg/sdk/fibe"
)

// Ensure the fibe import stays compile-visible even when no signature
// references it directly — handler is `func(..., *fibe.Client, ...)`.
var _ *fibe.Client

var (
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

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
	tier        toolTier // used by FIBE_MCP_TOOLS gating
	hidden      bool     // registered for dispatcher/catalog, never advertised natively

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
	tierMeta toolTier = iota
	tierBase
	tierGreenfield
	tierBrownfield
	tierOverseer
	tierLocal
	tierOther
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
	// Strip confirm so normal handlers don't have to ignore it.
	// fibe_call and fibe_pipeline need to read and forward confirm into
	// nested invocations, so we leave their args untouched.
	if !preservesConfirmArgs(t.name) {
		if _, ok := args["confirm"]; ok {
			delete(args, "confirm")
		}
		if err := validatePositiveIDArgs(args); err != nil {
			return nil, err
		}
	}

	c, err := d.srv.resolveClient(ctx)
	if err != nil {
		return nil, err
	}
	return t.handler(ctx, c, args)
}

func validatePositiveIDArgs(args map[string]any) error {
	for key, value := range args {
		if !isIDFieldName(key) || value == nil {
			continue
		}
		n, ok := valueAsInt64(value)
		if !ok {
			continue
		}
		if n <= 0 {
			return fmt.Errorf("field %q must be greater than zero", key)
		}
	}
	return nil
}

func valueAsInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int:
		return int64(x), true
	case int64:
		return x, true
	case json.Number:
		n, err := x.Int64()
		return n, err == nil
	case string:
		if x == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(x, 10, 64)
		return n, err == nil
	}
	return 0, false
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
// Slow but uniform. Before unmarshaling, it normalizes incoming arg keys and
// scalar values against the destination type so MCP hosts can pass the common
// snake_case / stringified-number shapes without losing the benefits of typed
// tool structs.
func bindArgs(args map[string]any, dest any) error {
	if dest == nil {
		return errors.New("nil destination")
	}
	normalized := any(args)
	if t := reflect.TypeOf(dest); t != nil {
		normalized = normalizeValueForType(args, t)
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func normalizeValueForType(v any, t reflect.Type) any {
	if v == nil || t == nil {
		return v
	}
	if implementsCustomUnmarshal(t) {
		return v
	}
	if t.Kind() == reflect.Pointer {
		return normalizeValueForType(v, t.Elem())
	}
	if implementsCustomUnmarshal(t) {
		return v
	}

	switch t.Kind() {
	case reflect.Struct:
		return normalizeStructValue(v, t)
	case reflect.Slice, reflect.Array:
		items, ok := v.([]any)
		if !ok {
			return v
		}
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = normalizeValueForType(item, t.Elem())
		}
		return out
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return v
		}
		m, ok := v.(map[string]any)
		if !ok {
			return v
		}
		out := make(map[string]any, len(m))
		for key, value := range m {
			out[key] = normalizeValueForType(value, t.Elem())
		}
		return out
	case reflect.String:
		if s, ok := coerceString(v); ok {
			return s
		}
	case reflect.Bool:
		if b, ok := coerceBool(v); ok {
			return b
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, ok := coerceInt64(v); ok {
			return n
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if n, ok := coerceUint64(v); ok {
			return n
		}
	case reflect.Float32, reflect.Float64:
		if f, ok := coerceFloat64(v); ok {
			return f
		}
	}
	return v
}

func normalizeStructValue(v any, t reflect.Type) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	fields := structFieldLookup(t)
	out := make(map[string]any, len(m))
	for key, value := range m {
		field, ok := fields[normalizeLookupKey(key)]
		if !ok {
			out[key] = value
			continue
		}
		out[field.outputKey] = normalizeValueForType(value, field.typ)
	}
	return out
}

type boundField struct {
	outputKey string
	typ       reflect.Type
}

func structFieldLookup(t reflect.Type) map[string]boundField {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	out := map[string]boundField{}
	if t.Kind() != reflect.Struct {
		return out
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous {
			for key, nested := range structFieldLookup(field.Type) {
				if _, exists := out[key]; !exists {
					out[key] = nested
				}
			}
			continue
		}
		if tagName(field.Tag.Get("json")) == "-" {
			continue
		}
		entry := boundField{
			outputKey: fieldOutputKey(field),
			typ:       field.Type,
		}
		for _, key := range fieldLookupKeys(field) {
			if key == "" {
				continue
			}
			out[normalizeLookupKey(key)] = entry
		}
	}
	return out
}

func fieldLookupKeys(field reflect.StructField) []string {
	keys := []string{
		field.Name,
		strings.ToLower(field.Name),
		toSnakeCase(field.Name),
	}
	if jsonName := tagName(field.Tag.Get("json")); jsonName != "" {
		keys = append(keys, jsonName)
	}
	if urlName := tagName(field.Tag.Get("url")); urlName != "" {
		keys = append(keys, urlName)
	}
	return keys
}

func fieldOutputKey(field reflect.StructField) string {
	if jsonName := tagName(field.Tag.Get("json")); jsonName != "" {
		return jsonName
	}
	return field.Name
}

func tagName(raw string) string {
	if raw == "" {
		return ""
	}
	name, _, _ := strings.Cut(raw, ",")
	return name
}

func normalizeLookupKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func implementsCustomUnmarshal(t reflect.Type) bool {
	if t == nil {
		return false
	}
	return t.Implements(jsonUnmarshalerType) ||
		t.Implements(textUnmarshalerType) ||
		reflect.PointerTo(t).Implements(jsonUnmarshalerType) ||
		reflect.PointerTo(t).Implements(textUnmarshalerType)
}

func coerceString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case json.Number:
		return x.String(), true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32), true
	case int:
		return strconv.Itoa(x), true
	case int8:
		return strconv.FormatInt(int64(x), 10), true
	case int16:
		return strconv.FormatInt(int64(x), 10), true
	case int32:
		return strconv.FormatInt(int64(x), 10), true
	case int64:
		return strconv.FormatInt(x, 10), true
	case uint:
		return strconv.FormatUint(uint64(x), 10), true
	case uint8:
		return strconv.FormatUint(uint64(x), 10), true
	case uint16:
		return strconv.FormatUint(uint64(x), 10), true
	case uint32:
		return strconv.FormatUint(uint64(x), 10), true
	case uint64:
		return strconv.FormatUint(x, 10), true
	default:
		return "", false
	}
}

func coerceBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		}
	case float64:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
	case int:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
	case int64:
		if x == 1 {
			return true, true
		}
		if x == 0 {
			return false, true
		}
	}
	return false, false
}

func coerceInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int8:
		return int64(x), true
	case int16:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		return int64(x), true
	case uint8:
		return int64(x), true
	case uint16:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		if x <= math.MaxInt64 {
			return int64(x), true
		}
	case float64:
		if math.Trunc(x) == x {
			return int64(x), true
		}
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return n, true
		}
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func coerceUint64(v any) (uint64, bool) {
	switch x := v.(type) {
	case uint:
		return uint64(x), true
	case uint8:
		return uint64(x), true
	case uint16:
		return uint64(x), true
	case uint32:
		return uint64(x), true
	case uint64:
		return x, true
	case int:
		if x >= 0 {
			return uint64(x), true
		}
	case int64:
		if x >= 0 {
			return uint64(x), true
		}
	case float64:
		if x >= 0 && math.Trunc(x) == x {
			return uint64(x), true
		}
	case json.Number:
		if n, err := strconv.ParseUint(x.String(), 10, 64); err == nil {
			return n, true
		}
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(x), 10, 64)
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func coerceFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f, true
		}
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}
