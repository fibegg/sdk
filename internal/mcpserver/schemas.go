package mcpserver

// schemaRegistry mirrors the schemaRegistry in cmd/fibe/cmd_schema.go so the
// MCP server can serve fibe_schema without importing the main package. Kept
// in sync by convention; a future phase can move both copies into a shared
// package (internal/schemas) if this becomes a maintenance burden.
var schemaRegistry = map[string]map[string]any{
	"playground": {
		"create": map[string]any{
			"required": []string{"name", "playspec_id"},
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "maxLength": 255, "description": "Playground name"},
				"playspec_id": map[string]any{"type": "integer", "description": "ID of the playspec to use"},
				"marquee_id":  map[string]any{"type": "integer", "description": "ID of the marquee (server) to deploy on"},
				"services":    map[string]any{"type": "object", "description": "Per-service configuration overrides"},
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
				"name":               map[string]any{"type": "string", "maxLength": 255},
				"compose_template":   map[string]any{"type": "string", "description": "Docker Compose YAML template"},
				"port_overrides":     map[string]any{"type": "string", "description": "Comma-separated port overrides"},
				"service_subdomains": map[string]any{"type": "object", "description": "Subdomain mapping per service"},
			},
		},
	},
	"prop": {
		"create": map[string]any{
			"required": []string{"repository_url"},
			"properties": map[string]any{
				"name":           map[string]any{"type": "string", "maxLength": 255},
				"repository_url": map[string]any{"type": "string", "format": "uri", "description": "Git repository URL"},
				"default_branch": map[string]any{"type": "string", "description": "Default branch name"},
				"provider":       map[string]any{"type": "string", "enum": []string{"github", "gitea"}, "description": "Git provider"},
			},
		},
	},
	"marquee": {
		"create": map[string]any{
			"required": []string{"name", "host"},
			"properties": map[string]any{
				"name": map[string]any{"type": "string", "maxLength": 255},
				"host": map[string]any{"type": "string", "description": "SSH hostname or IP"},
				"port": map[string]any{"type": "integer", "default": 22},
				"user": map[string]any{"type": "string", "default": "root"},
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
				"url": map[string]any{"type": "string", "format": "uri", "description": "Endpoint URL to receive events"},
				"events": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Event types to subscribe to. Values are dotted identifiers like agent.created, playground.running, trick.completed. Call the fibe_webhooks_event_types tool for the authoritative live list; unknown values fail with VALIDATION_FAILED.",
					"examples": []string{
						"agent.created", "agent.updated", "agent.deleted",
						"playground.created", "playground.running", "playground.stopped", "playground.error",
						"trick.completed", "trick.failed",
					},
				},
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
