package mcpserver

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func namedResource(resource string) bool {
	switch resource {
	case "playground", "trick", "playspec", "prop", "marquee":
		return true
	default:
		return false
	}
}

func argIdentifier(args map[string]any, idKey, identifierKey string) (string, bool) {
	if identifierKey != "" {
		if value := strings.TrimSpace(argString(args, identifierKey)); value != "" {
			return value, true
		}
	}
	if value := strings.TrimSpace(argString(args, idKey)); value != "" {
		return value, true
	}
	if id, ok := argInt64(args, idKey); ok {
		return strconv.FormatInt(id, 10), true
	}
	return "", false
}

func requiredIdentifier(args map[string]any, idKey, identifierKey string) (string, error) {
	identifier, ok := argIdentifier(args, idKey, identifierKey)
	if !ok {
		if identifierKey == "" {
			return "", fmt.Errorf("required field %q not set", idKey)
		}
		return "", fmt.Errorf("required field %q or %q not set", idKey, identifierKey)
	}
	return identifier, nil
}

func requiredPositiveID(args map[string]any, idKey string) (int64, error) {
	id, ok := argInt64(args, idKey)
	if !ok {
		return 0, fmt.Errorf("required field %q not set", idKey)
	}
	if id <= 0 {
		return 0, fmt.Errorf("field %q must be greater than zero", idKey)
	}
	return id, nil
}

func parsePositiveIdentifierID(identifier, field string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(identifier), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("field %q must be a numeric ID for this resource", field)
	}
	if id <= 0 {
		return 0, fmt.Errorf("field %q must be greater than zero", field)
	}
	return id, nil
}

func bindIdentifierArgs(args map[string]any, dest any, fields ...string) error {
	cleaned := make(map[string]any, len(args))
	for key, value := range args {
		cleaned[key] = value
	}
	identifiers := map[string]string{}
	for _, field := range fields {
		value, ok := stringIdentifierValue(args[field])
		if !ok {
			continue
		}
		identifiers[field] = value
		delete(cleaned, field)
	}
	if err := bindArgs(cleaned, dest); err != nil {
		return err
	}
	for field, value := range identifiers {
		if err := setIdentifierField(dest, field, value); err != nil {
			return err
		}
	}
	return nil
}

func stringIdentifierValue(value any) (string, bool) {
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return "", false
	}
	return s, true
}

func setIdentifierField(dest any, idField, value string) error {
	name, ok := identifierStructField(idField)
	if !ok {
		return nil
	}
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("identifier destination must be a non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("identifier destination must point to a struct")
	}
	field := v.FieldByName(name)
	if !field.IsValid() {
		return nil
	}
	if !field.CanSet() || field.Kind() != reflect.String {
		return fmt.Errorf("cannot set identifier field %s", name)
	}
	field.SetString(value)
	return nil
}

func identifierStructField(idField string) (string, bool) {
	switch idField {
	case "build_in_public_playground_id":
		return "BuildInPublicPlaygroundIdentifier", true
	case "ci_marquee_id":
		return "CIMarqueeIdentifier", true
	case "marquee_id":
		return "MarqueeIdentifier", true
	case "playground_id":
		return "PlaygroundIdentifier", true
	case "playspec_id":
		return "PlayspecIdentifier", true
	case "prop_id":
		return "PropIdentifier", true
	case "source_prop_id":
		return "SourcePropIdentifier", true
	case "target_playground_id":
		return "TargetPlaygroundIdentifier", true
	case "target_playspec_id":
		return "TargetPlayspecIdentifier", true
	default:
		return "", false
	}
}
