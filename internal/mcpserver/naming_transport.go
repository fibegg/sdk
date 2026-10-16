package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fibegg/sdk/internal/domainnames"
)

// Reject obsolete protocol names before MCP's generic lookup/schema errors.
// No removed tool or resource is registered or dispatched as an alias.
func rejectRemovedMCPRequest(_ context.Context, _ any, message any) error {
	data, ok := message.(json.RawMessage)
	if !ok {
		return nil
	}
	var request struct {
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	if json.Unmarshal(data, &request) != nil {
		return nil
	}
	switch request.Method {
	case "tools/call":
		name, _ := request.Params["name"].(string)
		if strings.HasPrefix(name, "fibe_") {
			if replacement, ok := domainnames.ToolReplacement(name); ok {
				return fmt.Errorf("FIBE tool %q was removed; use %q", name, replacement)
			}
		}
		arguments, _ := request.Params["arguments"].(map[string]any)
		if err := domainnames.RejectFields(arguments); err != nil {
			return err
		}
		resource, _ := arguments["resource"].(string)
		return domainnames.RemovedError(resource)
	case "resources/read":
		uri, _ := request.Params["uri"].(string)
		for _, prefix := range []string{"fibe://schema/", "fibe://help/"} {
			if suffix, ok := strings.CutPrefix(uri, prefix); ok {
				name, _, _ := strings.Cut(suffix, "/")
				if replacement, ok := domainnames.Replacement(name); ok {
					return fmt.Errorf("FIBE resource URI %q was removed; use %q", uri, prefix+replacement+strings.TrimPrefix(suffix, name))
				}
			}
		}
	}
	return nil
}
