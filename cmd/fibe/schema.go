package main

import (
	"fmt"
	"reflect"
	"strings"
)

func generateSchemaDoc(model any) string {
	schema := generateSchemaStr(reflect.TypeOf(model), "")
	return fmt.Sprintf("\n\nJSON SCHEMA PAYLOAD (-f / --from-file):\n%s\n", schema)
}

func generateSchemaStr(t reflect.Type, indent string) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		var lines []string
		lines = append(lines, "{")
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			parts := strings.Split(jsonTag, ",")
			name := parts[0]
			if name == "" {
				continue
			}
			optional := ""
			for _, p := range parts[1:] {
				if p == "omitempty" {
					optional = " // optional"
				}
			}
			fieldType := generateSchemaStr(field.Type, indent+"  ")
			lines = append(lines, fmt.Sprintf("%s  \"%s\": %s,%s", indent, name, fieldType, optional))
		}
		lines = append(lines, indent+"}")
		return strings.Join(lines, "\n")
	} else if t.Kind() == reflect.Map {
		fieldType := generateSchemaStr(t.Elem(), indent+"  ")
		return fmt.Sprintf("{\n%s  \"[key: string]\": %s\n%s}", indent, fieldType, indent)
	} else if t.Kind() == reflect.Slice {
		fieldType := generateSchemaStr(t.Elem(), indent+"  ")
		return fmt.Sprintf("[\n%s  %s\n%s]", indent, fieldType, indent)
	} else if t.Kind() == reflect.Interface {
		return "\"any\""
	} else if t.String() == "fibe.PlaygroundServiceInfo" {
		// Just a hardcoded generic override for nested complex structures if needed, not usually needed
	}
	tstr := t.Kind().String()
	if t.String() == "time.Time" {
		tstr = "datetime"
	}
	return "\"" + tstr + "\""
}
