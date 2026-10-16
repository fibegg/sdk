package fibe

import (
	"context"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type onlyFieldsCtxKey struct{}

// WithFields limits SDK responses to top-level JSON field names. Unknown names
// are ignored and an empty list disables filtering; omitted struct fields stay zero-valued.
func WithFields(ctx context.Context, fields ...string) context.Context {
	set := make(map[string]bool, len(fields))
	for _, f := range fields {
		set[f] = true
	}
	return context.WithValue(ctx, onlyFieldsCtxKey{}, set)
}

func fieldsFromCtx(ctx context.Context) map[string]bool {
	if v, ok := ctx.Value(onlyFieldsCtxKey{}).(map[string]bool); ok {
		return v
	}
	return nil
}

// ProjectFields filters a decoded struct or map to only include specified fields.
// It round-trips through map[string]any to strip unwanted keys, then
// re-decodes into the target. Returns the filtered value.
//
// If fields is nil or empty, returns the original value unchanged.
func ProjectFields[T any](v T, fields map[string]bool) T {
	if len(fields) == 0 {
		return v
	}

	data, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return v
	}

	for key := range m {
		if !fields[key] {
			delete(m, key)
		}
	}

	filtered, err := json.Marshal(m)
	if err != nil {
		return v
	}

	var result T
	if err := json.Unmarshal(filtered, &result); err != nil {
		return v
	}
	return result
}

// ProjectFieldsList applies field projection to each item in a slice.
func ProjectFieldsList[T any](items []T, fields map[string]bool) []T {
	if len(fields) == 0 {
		return items
	}
	for i := range items {
		items[i] = ProjectFields(items[i], fields)
	}
	return items
}

// ToYAML converts any value to a YAML string. Useful for LLM contexts
// where YAML is more token-efficient than JSON.
//
//	pg, _ := client.Playgrounds.Get(ctx, 42)
//	fmt.Println(fibe.ToYAML(pg))
func ToYAML(v any) string {
	// If the value is a struct with json tags, round-trip through JSON
	// to get snake_case keys instead of Go PascalCase.
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return string(data)
	}
	out, err := yaml.Marshal(normalized)
	if err != nil {
		return string(data)
	}
	return string(out)
}
