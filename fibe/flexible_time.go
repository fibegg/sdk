package fibe

import (
	"encoding/json"
	"fmt"
	"time"
)

var fibeTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05 MST",
	"2006-01-02 15:04:05 UTC",
	"2006-01-02 15:04:05 -0700",
}

func parseFlexibleTimeJSON(raw json.RawMessage) (*time.Time, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}

	for _, layout := range fibeTimeLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("unsupported time format %q", value)
}

func stripFlexibleTimeFields(data []byte, fieldNames ...string) ([]byte, map[string]*time.Time, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, err
	}

	parsed := make(map[string]*time.Time, len(fieldNames))
	for _, field := range fieldNames {
		if value, ok := raw[field]; ok {
			t, err := parseFlexibleTimeJSON(value)
			if err != nil {
				return nil, nil, err
			}
			parsed[field] = t
			delete(raw, field)
		}
	}

	cleaned, err := json.Marshal(raw)
	return cleaned, parsed, err
}
