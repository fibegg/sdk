package mcpserver

import (
	"fmt"
	"strings"
)

const (
	responseOnlyArg       = "only"
	responseFieldsAlias   = "fields"
	responseOutputPathArg = "output_path"
)

type responseShape struct {
	only       []string
	outputPath string
}

func parseResponseShape(args map[string]any) (responseShape, error) {
	var shape responseShape
	if args == nil {
		return shape, nil
	}

	rawOnly, ok := args[responseOnlyArg]
	if !ok {
		rawOnly, ok = args[responseFieldsAlias]
	}
	if ok && rawOnly != nil {
		fields, err := parseOnlyFields(rawOnly)
		if err != nil {
			return responseShape{}, err
		}
		shape.only = fields
	}

	if raw, ok := args[responseOutputPathArg]; ok && raw != nil {
		path, ok := raw.(string)
		if !ok {
			return responseShape{}, fmt.Errorf("field %q must be a string", responseOutputPathArg)
		}
		shape.outputPath = strings.TrimSpace(path)
	}

	return shape, nil
}

func stripResponseShapeArgs(args map[string]any) map[string]any {
	if args == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(args))
	for key, value := range args {
		switch key {
		case responseOnlyArg, responseFieldsAlias, responseOutputPathArg:
			continue
		default:
			out[key] = value
		}
	}
	return out
}

func parseOnlyFields(raw any) ([]string, error) {
	var fields []string
	add := func(value string) {
		for _, part := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				fields = append(fields, trimmed)
			}
		}
	}

	switch value := raw.(type) {
	case string:
		add(value)
	case []string:
		for _, item := range value {
			add(item)
		}
	case []any:
		for _, item := range value {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("field %q must contain only strings", responseOnlyArg)
			}
			add(s)
		}
	default:
		return nil, fmt.Errorf("field %q must be a string or array of strings", responseOnlyArg)
	}
	return fields, nil
}

func applyResponseShape(result any, shape responseShape) (any, error) {
	if shape.outputPath == "" && len(shape.only) == 0 {
		return result, nil
	}

	normalized, err := normalizeForJSONPath(result)
	if err != nil {
		return nil, fmt.Errorf("normalize response for projection: %w", err)
	}

	if shape.outputPath != "" {
		normalized, err = projectOnMap(normalized, shape.outputPath)
		if err != nil {
			return nil, fmt.Errorf("output_path %q: %w", shape.outputPath, err)
		}
	}

	if len(shape.only) > 0 {
		normalized = projectOnlyFields(normalized, shape.only)
	}

	return normalized, nil
}

func projectOnlyFields(value any, fields []string) any {
	set := make(map[string]bool, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			set[field] = true
		}
	}
	if len(set) == 0 {
		return value
	}
	return projectOnlyValue(value, set)
}

func projectOnlyValue(value any, fields map[string]bool) any {
	switch typed := value.(type) {
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			if itemMap, ok := item.(map[string]any); ok {
				out[i] = projectRootFields(itemMap, fields)
			} else {
				out[i] = item
			}
		}
		return out
	case map[string]any:
		return projectMapFields(typed, fields)
	default:
		return value
	}
}

func projectMapFields(value map[string]any, fields map[string]bool) map[string]any {
	if key, ok := projectionEnvelopeKey(value); ok {
		out := cloneMapAny(value)
		out[key] = projectOnlyValue(out[key], fields)
		return out
	}
	return projectRootFields(value, fields)
}

func projectRootFields(value map[string]any, fields map[string]bool) map[string]any {
	out := make(map[string]any, len(fields))
	for key, keep := range fields {
		if !keep {
			continue
		}
		if v, ok := value[key]; ok {
			out[key] = v
		}
	}
	return out
}

func projectionEnvelopeKey(value map[string]any) (string, bool) {
	for _, key := range []string{"data", "Data"} {
		switch value[key].(type) {
		case []any, map[string]any:
			return key, true
		}
	}

	var arrayKey string
	for key, raw := range value {
		if _, ok := raw.([]any); !ok {
			continue
		}
		if arrayKey != "" {
			return "", false
		}
		arrayKey = key
	}
	if arrayKey == "" {
		return "", false
	}
	return arrayKey, true
}

func cloneMapAny(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}
