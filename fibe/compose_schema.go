package fibe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const defaultFibeSchemaURL = "https://fibe.gg/schema.json"

func (s *PlayspecService) validateComposeSchema(ctx context.Context, composeYAML string) ([]string, error) {
	var body any
	if err := yaml.Unmarshal([]byte(composeYAML), &body); err != nil {
		return []string{fmt.Sprintf("Invalid YAML: %v", err)}, nil
	}

	schemaDoc, schemaURL, err := s.fetchComposeSchema(ctx)
	if err != nil {
		return nil, err
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("fibe: add compose schema: %w", err)
	}
	compiled, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("fibe: compile compose schema: %w", err)
	}

	if err := compiled.Validate(yamlToJSONValue(body)); err != nil {
		return validationErrorMessages(err), nil
	}
	return nil, nil
}

func (s *PlayspecService) fetchComposeSchema(ctx context.Context) (any, string, error) {
	schemaURL := os.Getenv("FIBE_SCHEMA_URL")
	if schemaURL == "" {
		schemaURL = s.client.cfg.baseURL() + "/schema.json"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, schemaURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("fibe: create compose schema request: %w", err)
	}
	req.Header.Set("Accept", "application/schema+json, application/json")
	req.Header.Set("User-Agent", s.client.cfg.userAgent)

	resp, err := s.client.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fibe: fetch compose schema: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, "", fmt.Errorf("fibe: fetch compose schema: status %d", resp.StatusCode)
	}

	var schemaDoc any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&schemaDoc); err != nil {
		return nil, "", fmt.Errorf("fibe: decode compose schema: %w", err)
	}
	return yamlToJSONValue(schemaDoc), schemaURL, nil
}

func yamlToJSONValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[key] = yamlToJSONValue(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, val := range v {
			out[fmt.Sprint(key)] = yamlToJSONValue(val)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = yamlToJSONValue(val)
		}
		return out
	default:
		return value
	}
}

func validationErrorMessages(err error) []string {
	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []string{err.Error()}
	}

	output := validationErr.BasicOutput()
	units := output.Errors
	if len(units) == 0 {
		units = []jsonschema.OutputUnit{*output}
	}

	messages := make([]string, 0, len(units))
	seen := map[string]struct{}{}
	for _, unit := range units {
		if unit.Error == nil {
			continue
		}
		location := unit.InstanceLocation
		if location == "" {
			location = "/"
		}
		message := fmt.Sprintf("%s: %s", location, unit.Error.String())
		if _, exists := seen[message]; exists {
			continue
		}
		seen[message] = struct{}{}
		messages = append(messages, message)
	}
	if len(messages) == 0 {
		text := strings.TrimSpace(err.Error())
		if text == "" {
			text = "compose does not match Fibe JSON Schema"
		}
		return []string{text}
	}
	return messages
}
