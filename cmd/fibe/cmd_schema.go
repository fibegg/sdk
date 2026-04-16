package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// schemaRegistry maps resource names to their JSON schema descriptions.
// This is the machine-parsable companion to --help.
var schemaRegistry = map[string]map[string]any{
	"playground": {
		"create": map[string]any{
			"required": []string{"name", "playspec_id"},
			"properties": map[string]any{
				"name":         map[string]any{"type": "string", "maxLength": 255, "description": "Playground name"},
				"playspec_id":  map[string]any{"type": "integer", "description": "ID of the playspec to use"},
				"marquee_id":   map[string]any{"type": "integer", "description": "ID of the marquee (server) to deploy on"},
				"services":     map[string]any{"type": "object", "description": "Per-service configuration overrides"},
			},
		},
		"update": map[string]any{
			"properties": map[string]any{
				"name": map[string]any{"type": "string", "maxLength": 255},
			},
		},
	},
	"agent": {
		"create": map[string]any{
			"required": []string{"name", "provider"},
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "maxLength": 255, "description": "Agent name"},
				"provider":    map[string]any{"type": "string", "enum": []string{"gemini", "claude-code", "openai-codex", "opencode", "aider", "custom"}, "description": "LLM provider"},
				"api_key_id":  map[string]any{"type": "integer", "description": "API key to associate"},
				"description": map[string]any{"type": "string", "description": "Agent description"},
			},
		},
	},
	"playspec": {
		"create": map[string]any{
			"required": []string{"name"},
			"properties": map[string]any{
				"name":                 map[string]any{"type": "string", "maxLength": 255},
				"compose_template":     map[string]any{"type": "string", "description": "Docker Compose YAML template"},
				"port_overrides":       map[string]any{"type": "string", "description": "Comma-separated port overrides"},
				"service_subdomains":   map[string]any{"type": "object", "description": "Subdomain mapping per service"},
			},
		},
	},
	"prop": {
		"create": map[string]any{
			"required": []string{"repository_url"},
			"properties": map[string]any{
				"name":            map[string]any{"type": "string", "maxLength": 255},
				"repository_url":  map[string]any{"type": "string", "format": "uri", "description": "Git repository URL"},
				"default_branch":  map[string]any{"type": "string", "description": "Default branch name"},
				"provider":        map[string]any{"type": "string", "enum": []string{"github", "gitea"}, "description": "Git provider"},
			},
		},
	},
	"marquee": {
		"create": map[string]any{
			"required": []string{"name", "host"},
			"properties": map[string]any{
				"name":     map[string]any{"type": "string", "maxLength": 255},
				"host":     map[string]any{"type": "string", "description": "SSH hostname or IP"},
				"port":     map[string]any{"type": "integer", "default": 22},
				"user":     map[string]any{"type": "string", "default": "root"},
			},
		},
	},
	"secret": {
		"create": map[string]any{
			"required": []string{"key", "value"},
			"properties": map[string]any{
				"key":         map[string]any{"type": "string", "maxLength": 255, "pattern": "^[a-zA-Z0-9_-]+$"},
				"value":       map[string]any{"type": "string", "maxLength": 65536},
				"description": map[string]any{"type": "string", "maxLength": 500},
			},
		},
	},
	"team": {
		"create": map[string]any{
			"required": []string{"name"},
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "maxLength": 255},
				"description": map[string]any{"type": "string"},
			},
		},
	},
	"webhook": {
		"create": map[string]any{
			"required": []string{"url"},
			"properties": map[string]any{
				"url":    map[string]any{"type": "string", "format": "uri", "description": "Endpoint URL to receive events"},
				"events": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Event types to subscribe to"},
				"secret": map[string]any{"type": "string", "description": "Shared secret for HMAC signature verification"},
			},
		},
	},
	"api_key": {
		"create": map[string]any{
			"required": []string{"label"},
			"properties": map[string]any{
				"label":      map[string]any{"type": "string", "maxLength": 255},
				"expires_at": map[string]any{"type": "string", "format": "date-time"},
				"scopes":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Permission scopes"},
			},
		},
	},
}

func schemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema <resource> [operation]",
		Short: "Show JSON Schema for a resource",
		Long: `Print the JSON Schema for resource create/update parameters.
This is the machine-parsable companion to --help.

LLM agents can use this to understand the exact shape of API payloads
without guessing or hallucinating field names.

Resources: playground, agent, playspec, prop, marquee, secret, team, webhook, api_key

Examples:
  fibe schema playground          # show all operations
  fibe schema playground create   # show create schema only
  fibe schema --list              # list all resources`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			listFlag, _ := cmd.Flags().GetBool("list")
			if listFlag || len(args) == 0 {
				resources := make([]string, 0, len(schemaRegistry))
				for k := range schemaRegistry {
					resources = append(resources, k)
				}
				fmt.Println("Available resources:", strings.Join(resources, ", "))
				return nil
			}

			resource := args[0]
			schemas, ok := schemaRegistry[resource]
			if !ok {
				return fmt.Errorf("unknown resource %q — available: %s", resource, strings.Join(resourceNames(), ", "))
			}

			if len(args) == 2 {
				op := args[1]
				schema, ok := schemas[op]
				if !ok {
					return fmt.Errorf("unknown operation %q for resource %q", op, resource)
				}
				return printJSON(map[string]any{resource + "." + op: schema})
			}

			return printJSON(schemas)
		},
	}
}

func init() {
	cmd := schemaCmd()
	cmd.Flags().Bool("list", false, "List all available resources")
}

func resourceNames() []string {
	names := make([]string, 0, len(schemaRegistry))
	for k := range schemaRegistry {
		names = append(names, k)
	}
	return names
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
