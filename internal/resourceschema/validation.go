package resourceschema

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ValidateMutationPayload(rawResource, rawOperation string, payload map[string]any) (string, string, error) {
	schema, resource, operation, ok := MutationSchemaFor(rawResource, rawOperation)
	if !ok {
		if resource == "" {
			return "", "", fmt.Errorf("unknown resource %q; supported mutation resources: %s", rawResource, MutationResourceNamesString())
		}
		ops, _, _ := MutationOperationsForResource(resource)
		if len(ops) == 0 {
			return "", "", fmt.Errorf("resource %q does not support mutation operations", resource)
		}
		return "", "", fmt.Errorf("resource %q does not support operation %q; supported operations: %s", resource, operation, strings.Join(ops, ", "))
	}
	if payload == nil {
		return "", "", fmt.Errorf("payload is required for %s.%s", resource, operation)
	}
	if err := validateObjectPayload(resource+"."+operation+".payload", payload, schema); err != nil {
		return "", "", err
	}
	if operation == "update" && !hasMutationUpdateFields(payload, requiredFields(schema)...) {
		return "", "", fmt.Errorf("%s.%s payload must include at least one field to update besides %s", resource, operation, strings.Join(requiredFields(schema), ", "))
	}
	return resource, operation, nil
}

func ValidatePayload(rawResource, rawOperation string, payload map[string]any) (string, string, error) {
	schema, resource, operation, ok := SchemaFor(rawResource, rawOperation)
	if !ok {
		if resource == "" {
			return "", "", fmt.Errorf("unknown resource %q; supported resources: %s", rawResource, ResourceNamesString())
		}
		return "", "", fmt.Errorf("resource %q does not support operation %q", resource, operation)
	}
	if payload == nil {
		return "", "", fmt.Errorf("payload is required for %s.%s", resource, operation)
	}
	if err := validateObjectPayload(resource+"."+operation+".payload", payload, cloneMap(schemaMap(schema))); err != nil {
		return "", "", err
	}
	return resource, operation, nil
}

func validateObjectPayload(path string, payload map[string]any, schema map[string]any) error {
	props, _ := schema["properties"].(map[string]any)
	required := requiredFields(schema)
	for _, field := range required {
		value, ok := payload[field]
		if !ok || value == nil {
			return fmt.Errorf("%s.%s is required", path, field)
		}
		if fieldSchema, ok := props[field].(map[string]any); ok {
			if schemaHasType(fieldSchema, "string") {
				if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
					return fmt.Errorf("%s.%s is required", path, field)
				}
			}
		}
	}
	if err := validateAnyOf(path, payload, schema); err != nil {
		return err
	}
	if additional, ok := schema["additionalProperties"].(bool); ok && !additional {
		var unknown []string
		for key := range payload {
			if _, ok := props[key]; !ok {
				unknown = append(unknown, key)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			return fmt.Errorf("%s contains unsupported field(s): %s", path, strings.Join(unknown, ", "))
		}
	}
	for key, value := range payload {
		if value == nil {
			continue
		}
		prop, ok := props[key].(map[string]any)
		if !ok {
			continue
		}
		if err := validateValue(path+"."+key, value, prop); err != nil {
			return err
		}
	}
	return nil
}

func validateAnyOf(path string, payload map[string]any, schema map[string]any) error {
	raw, ok := schema["anyOf"]
	if !ok {
		return nil
	}
	branches, ok := raw.([]any)
	if !ok || len(branches) == 0 {
		return nil
	}
	for _, branchRaw := range branches {
		branch, ok := branchRaw.(map[string]any)
		if !ok {
			continue
		}
		missing := false
		for _, field := range requiredFields(branch) {
			value, ok := payload[field]
			if !ok || value == nil {
				missing = true
				break
			}
			if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
				missing = true
				break
			}
		}
		if !missing {
			return nil
		}
	}
	return fmt.Errorf("%s must satisfy one of the required field sets in anyOf", path)
}

func validateValue(path string, value any, schema map[string]any) error {
	if enum := schemaEnum(schema); len(enum) > 0 {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a string enum value", path)
		}
		if !stringInSlice(enum, s) {
			return fmt.Errorf("%s must be one of: %s", path, strings.Join(enum, ", "))
		}
	}

	switch {
	case schemaHasType(schema, "string"):
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be a string", path)
		}
		if min, ok := numericMinimum(schema["minLength"]); ok && float64(utf8.RuneCountInString(s)) < min {
			return fmt.Errorf("%s length must be greater than or equal to %v", path, min)
		}
		if max, ok := numericMinimum(schema["maxLength"]); ok && float64(utf8.RuneCountInString(s)) > max {
			return fmt.Errorf("%s length must be less than or equal to %v", path, max)
		}
		if pattern, ok := schema["pattern"].(string); ok && pattern != "" {
			matched, err := regexp.MatchString(pattern, s)
			if err != nil {
				return fmt.Errorf("%s has invalid schema pattern %q: %w", path, pattern, err)
			}
			if !matched {
				return fmt.Errorf("%s must match pattern %s", path, pattern)
			}
		}
	case schemaHasType(schema, "integer"):
		n, ok := numericValue(value)
		if !ok || math.Trunc(n) != n {
			return fmt.Errorf("%s must be an integer", path)
		}
		if min, ok := numericMinimum(schema["minimum"]); ok && n < min {
			return fmt.Errorf("%s must be greater than or equal to %v", path, min)
		}
		if max, ok := numericMinimum(schema["maximum"]); ok && n > max {
			return fmt.Errorf("%s must be less than or equal to %v", path, max)
		}
	case schemaHasType(schema, "number"):
		n, ok := numericValue(value)
		if !ok {
			return fmt.Errorf("%s must be a number", path)
		}
		if min, ok := numericMinimum(schema["minimum"]); ok && n < min {
			return fmt.Errorf("%s must be greater than or equal to %v", path, min)
		}
		if max, ok := numericMinimum(schema["maximum"]); ok && n > max {
			return fmt.Errorf("%s must be less than or equal to %v", path, max)
		}
	case schemaHasType(schema, "boolean"):
		if _, ok := boolValue(value); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	case schemaHasType(schema, "array"):
		items, ok := arrayValues(value)
		if !ok {
			return fmt.Errorf("%s must be an array", path)
		}
		if min, ok := numericMinimum(schema["minItems"]); ok && float64(len(items)) < min {
			return fmt.Errorf("%s must contain at least %v item(s)", path, min)
		}
		if max, ok := numericMinimum(schema["maxItems"]); ok && float64(len(items)) > max {
			return fmt.Errorf("%s must contain at most %v item(s)", path, max)
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for i, item := range items {
				if item == nil {
					continue
				}
				if err := validateValue(fmt.Sprintf("%s[%d]", path, i), item, itemSchema); err != nil {
					return err
				}
			}
		}
	case schemaHasType(schema, "object"):
		obj, ok := objectValue(value)
		if !ok {
			return fmt.Errorf("%s must be an object", path)
		}
		if _, ok := schema["properties"].(map[string]any); ok {
			if err := validateObjectPayload(path, obj, schema); err != nil {
				return err
			}
		}
	}
	return nil
}

func requiredFields(schema map[string]any) []string {
	raw, ok := schema["required"]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func schemaEnum(schema map[string]any) []string {
	raw, ok := schema["enum"]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func numericValue(v any) (float64, bool) {
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
		n, err := x.Float64()
		return n, err == nil
	case string:
		if x == "" {
			return 0, false
		}
		n, err := strconv.ParseFloat(x, 64)
		return n, err == nil
	}
	return 0, false
}

func numericMinimum(v any) (float64, bool) {
	return numericValue(v)
}

func boolValue(v any) (bool, bool) {
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
	}
	return false, false
}

func arrayValues(v any) ([]any, bool) {
	switch values := v.(type) {
	case []any:
		return values, true
	case []string:
		out := make([]any, len(values))
		for i, value := range values {
			out[i] = value
		}
		return out, true
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

func objectValue(v any) (map[string]any, bool) {
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	if m, ok := v.(map[string]string); ok {
		out := make(map[string]any, len(m))
		for key, value := range m {
			out[key] = value
		}
		return out, true
	}
	return nil, false
}

func hasMutationUpdateFields(payload map[string]any, routingKeys ...string) bool {
	skip := map[string]bool{}
	for _, key := range routingKeys {
		skip[key] = true
	}
	for key, value := range payload {
		if skip[key] || value == nil {
			continue
		}
		if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		return true
	}
	return false
}

func stringInSlice(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
