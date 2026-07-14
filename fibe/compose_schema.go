package fibe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

var compiledComposeSchemas = struct {
	sync.RWMutex
	values map[[sha256.Size]byte]*jsonschema.Schema
}{values: make(map[[sha256.Size]byte]*jsonschema.Schema)}

func (s *PlayspecService) validateComposeSchema(ctx context.Context, composeYAML string) ([]string, error) {
	var body any
	if err := yaml.Unmarshal([]byte(composeYAML), &body); err != nil {
		return []string{fmt.Sprintf("Invalid YAML: %v", err)}, nil
	}

	schemaData, schemaURL, err := s.fetchComposeSchema(ctx)
	if err != nil {
		return nil, err
	}
	compiled, err := compileComposeSchema(schemaData, schemaURL)
	if err != nil {
		return nil, err
	}

	if err := compiled.Validate(yamlToJSONValue(body)); err != nil {
		return validationErrorMessages(err), nil
	}
	return nil, nil
}

func (s *PlayspecService) fetchComposeSchema(ctx context.Context) ([]byte, string, error) {
	schemaURL := os.Getenv("FIBE_SCHEMA_URL")
	if schemaURL == "" {
		schemaURL = s.client.cfg.baseURL() + "/schema.json"
	}

	parsed, err := url.Parse(schemaURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, "", fmt.Errorf("fibe: invalid compose schema URL %q", schemaURL)
	}
	// #nosec G704 -- the explicitly configured endpoint is restricted to credential-free HTTP(S) URLs above.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("fibe: create compose schema request: %w", err)
	}
	req.Header.Set("Accept", "application/schema+json, application/json")
	req.Header.Set("User-Agent", s.client.cfg.userAgent)

	// #nosec G704 -- callers opt into this schema endpoint and URL validation rejects non-HTTP(S) targets.
	resp, err := s.client.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fibe: fetch compose schema: %w", err)
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fibe: fetch compose schema: status %d", resp.StatusCode)
	}
	data, err := readLimited(resp.Body, maxResponseBody, "compose schema")
	if err != nil {
		return nil, "", fmt.Errorf("fibe: decode compose schema: %w", err)
	}
	var schemaDoc any
	if err := decodeJSONLimited(bytes.NewReader(data), int64(len(data)), "compose schema", &schemaDoc); err != nil {
		return nil, "", fmt.Errorf("fibe: decode compose schema: %w", err)
	}
	canonical, err := json.Marshal(yamlToJSONValue(schemaDoc))
	if err != nil {
		return nil, "", fmt.Errorf("fibe: normalize compose schema: %w", err)
	}
	return canonical, parsed.String(), nil
}

func compileComposeSchema(data []byte, schemaURL string) (*jsonschema.Schema, error) {
	digest := sha256.Sum256(data)
	compiledComposeSchemas.RLock()
	compiled := compiledComposeSchemas.values[digest]
	compiledComposeSchemas.RUnlock()
	if compiled != nil {
		return compiled, nil
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("fibe: decode compose schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource(schemaURL, document); err != nil {
		return nil, fmt.Errorf("fibe: add compose schema: %w", err)
	}
	compiledResult, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("fibe: compile compose schema: %w", err)
	}
	compiled = compiledResult
	compiledComposeSchemas.Lock()
	if existing := compiledComposeSchemas.values[digest]; existing != nil {
		compiled = existing
	} else {
		compiledComposeSchemas.values[digest] = compiled
	}
	compiledComposeSchemas.Unlock()
	return compiled, nil
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
