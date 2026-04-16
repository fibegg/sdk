package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var rawPayload []byte

func applyFromFile(dest any) error {
	var data []byte
	var err error

	if flagFromFile != "" {
		if flagFromFile == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(flagFromFile)
		}
		if err != nil {
			return fmt.Errorf("read from-file %q: %w", flagFromFile, err)
		}
	} else {
		// Auto-detect stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read from stdin: %w", err)
			}
		} else {
			// Nothing specified and no pipe
			return nil
		}
	}

	if len(data) == 0 {
		return nil
	}
	rawPayload = data

	// Try JSON first
	err = json.Unmarshal(data, dest)
	if err == nil {
		return nil
	}

	// Fallback to YAML: decode to generic structure, marshal to JSON, then unmarshal
	// This ensures we respect the `json:"..."` struct tags present in the Fibe API payloads,
	// which yaml.v3 doesn't handle natively.
	var yamlData interface{}
	errYAML := yaml.Unmarshal(data, &yamlData)
	if errYAML == nil {
		jsonBytes, errJSON := json.Marshal(yamlData)
		if errJSON == nil {
			if errDecode := json.Unmarshal(jsonBytes, dest); errDecode == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("failed to parse file %q as JSON or YAML. JSON error: %v, YAML error: %v", flagFromFile, err, errYAML)
}

func resolveStringValue(val string) string {
	if strings.HasPrefix(val, "@") {
		data, err := os.ReadFile(strings.TrimPrefix(val, "@"))
		if err == nil {
			return string(data)
		}
	}
	return val
}
