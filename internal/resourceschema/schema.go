package resourceschema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/fibegg/sdk/fibe"
)

type CatalogEntry struct {
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases,omitempty"`
	Operations []string `json:"operations"`
}

type resourceDef struct {
	name       string
	aliases    []string
	operations []string
	listSchema map[string]any
	get        bool
	delete     bool
}

var flatResources = []resourceDef{
	{name: "playground", aliases: []string{"playgrounds"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.PlaygroundListParams](), get: true, delete: true},
	{name: "trick", aliases: []string{"tricks"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.PlaygroundListParams](), get: true, delete: true},
	{name: "agent", aliases: []string{"agents"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.AgentListParams](), get: true, delete: true},
	{name: "artefact", aliases: []string{"artefacts"}, operations: []string{"list", "get"}, listSchema: listParamsSchema[fibe.ArtefactListParams](), get: true},
	{name: "artefact_attachment", aliases: []string{"artefact_attachments"}, operations: []string{"get"}, get: true},
	{name: "playspec", aliases: []string{"playspecs"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.PlayspecListParams](), get: true, delete: true},
	{name: "prop", aliases: []string{"props"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.PropListParams](), get: true, delete: true},
	{name: "marquee", aliases: []string{"marquees"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.MarqueeListParams](), get: true, delete: true},
	{name: "secret", aliases: []string{"secrets"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.SecretListParams](), get: true, delete: true},
	{name: "api_key", aliases: []string{"api_keys"}, operations: []string{"list", "delete"}, listSchema: listParamsSchema[fibe.APIKeyListParams](), delete: true},
	{name: "webhook", aliases: []string{"webhooks", "webhook_endpoint", "webhook_endpoints"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.WebhookEndpointListParams](), get: true, delete: true},
	{name: "webhook_delivery", aliases: []string{"webhook_deliveries"}, operations: []string{"list"}, listSchema: webhookDeliveryListParamsSchema()},
	{name: "template", aliases: []string{"templates", "import_template", "import_templates"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.ImportTemplateListParams](), get: true, delete: true},
	{name: "template_version", aliases: []string{"template_versions"}, operations: []string{"list", "delete"}, listSchema: templateVersionListParamsSchema(), delete: true},
	{name: "template_source", aliases: []string{"template_sources"}, operations: []string{"delete"}, delete: true},
	{name: "job_env", aliases: []string{"job_envs", "job_environment", "job_environments"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.JobEnvListParams](), get: true, delete: true},
	{name: "audit_log", aliases: []string{"audit_logs"}, operations: []string{"list"}, listSchema: listParamsSchema[fibe.AuditLogListParams]()},
	{name: "memory", aliases: []string{"memories"}, operations: []string{"list", "get", "delete"}, listSchema: listParamsSchema[fibe.MemoryListParams](), get: true, delete: true},
	{name: "category", aliases: []string{"categories", "template_category", "template_categories"}, operations: []string{"list"}, listSchema: listParamsSchema[fibe.ListParams]()},
}

var schemaOnlyResources = []resourceDef{
	{name: "compose", aliases: []string{"composes", "docker_compose", "docker_compose_yaml"}},
	{name: "mutter", aliases: []string{"mutters"}},
}

var aliasToCanonical = buildAliasMap()

var registry = buildRegistry()

func Catalog() []CatalogEntry {
	resources := schemaResources()
	out := make([]CatalogEntry, 0, len(resources))
	for _, r := range resources {
		aliases := append([]string(nil), r.aliases...)
		operations := SortedOperationNames(r.name)
		out = append(out, CatalogEntry{
			Name:       r.name,
			Aliases:    aliases,
			Operations: operations,
		})
	}
	return out
}

func Registry() map[string]map[string]any {
	return registry
}

func CanonicalResource(raw string) (string, bool) {
	key := NormalizeResource(raw)
	if key == "" {
		return "", false
	}
	name, ok := aliasToCanonical[key]
	return name, ok
}

func NormalizeResource(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.ReplaceAll(raw, "-", "_")
	raw = strings.ReplaceAll(raw, " ", "_")
	for strings.Contains(raw, "__") {
		raw = strings.ReplaceAll(raw, "__", "_")
	}
	return raw
}

func ResourceNames() []string {
	return resourceNames(schemaResources())
}

func FlatResourceNames() []string {
	return resourceNames(flatResources)
}

func ResourceSelectors() []string {
	return resourceSelectorsFor(flatResources, func(resourceDef) bool { return true })
}

func ResourceSelectorsForOperation(operation string) []string {
	op := NormalizeResource(operation)
	return resourceSelectorsFor(flatResources, func(r resourceDef) bool {
		_, ok := registry[r.name][op]
		return ok
	})
}

func SchemaResourceSelectors() []string {
	values := append([]string{"list"}, resourceSelectorsFor(schemaResources(), func(resourceDef) bool { return true })...)
	sort.Strings(values)
	return values
}

func OperationNames() []string {
	seen := map[string]bool{}
	for _, ops := range registry {
		for op := range ops {
			seen[op] = true
		}
	}
	values := make([]string, 0, len(seen))
	for op := range seen {
		values = append(values, op)
	}
	sort.Strings(values)
	return values
}

func resourceSelectorsFor(resources []resourceDef, include func(resourceDef) bool) []string {
	seen := map[string]bool{}
	var values []string
	add := func(value string) {
		for _, candidate := range []string{value, strings.ReplaceAll(value, "_", "-")} {
			if candidate != "" && !seen[candidate] {
				values = append(values, candidate)
				seen[candidate] = true
			}
		}
	}
	for _, r := range resources {
		if !include(r) {
			continue
		}
		add(r.name)
		for _, alias := range r.aliases {
			add(NormalizeResource(alias))
		}
	}
	sort.Strings(values)
	return values
}

func ResourceNamesString() string {
	return strings.Join(ResourceNames(), ", ")
}

func FlatResourceNamesString() string {
	return strings.Join(FlatResourceNames(), ", ")
}

func resourceNames(resources []resourceDef) []string {
	names := make([]string, 0, len(resources))
	for _, r := range resources {
		names = append(names, r.name)
	}
	return names
}

func SchemasFor(raw string) (map[string]any, string, bool) {
	name, ok := CanonicalResource(raw)
	if !ok {
		return nil, "", false
	}
	schemas, ok := registry[name]
	return schemas, name, ok
}

func SchemaFor(rawResource, rawOperation string) (any, string, string, bool) {
	schemas, name, ok := SchemasFor(rawResource)
	if !ok {
		return nil, "", "", false
	}
	op := NormalizeResource(rawOperation)
	schema, ok := schemas[op]
	return schema, name, op, ok
}

func CatalogResponse() map[string]any {
	return map[string]any{
		"resources": Catalog(),
	}
}

func buildAliasMap() map[string]string {
	out := map[string]string{}
	for _, r := range schemaResources() {
		out[r.name] = r.name
		for _, alias := range r.aliases {
			out[NormalizeResource(alias)] = r.name
		}
	}
	return out
}

func schemaResources() []resourceDef {
	out := make([]resourceDef, 0, len(flatResources)+len(schemaOnlyResources))
	out = append(out, flatResources...)
	out = append(out, schemaOnlyResources...)
	return out
}

func buildRegistry() map[string]map[string]any {
	out := map[string]map[string]any{
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
		"webhook": {
			"create": map[string]any{
				"required": []string{"url"},
				"properties": map[string]any{
					"url": map[string]any{"type": "string", "format": "uri", "description": "Endpoint URL to receive events"},
					"events": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Event types to subscribe to. Inspect fibe_schema(resource:webhook, operation:event_types) for the SDK's static enum; unknown values fail with VALIDATION_FAILED.",
						"examples": []string{
							"agent.created", "agent.updated", "agent.destroyed",
							"playground.created", "playground.status.changed", "playground.error",
							"playground.completed", "webhook.test",
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

	for _, r := range schemaResources() {
		if out[r.name] == nil {
			out[r.name] = map[string]any{}
		}
	}

	out["playground"]["create"] = paramsSchema[fibe.PlaygroundCreateParams]("name", "playspec_id")
	out["playground"]["update"] = updateParamsSchemaFor[fibe.PlaygroundUpdateParams]("playground_id")
	out["playground"]["action"] = map[string]any{
		"type":     "object",
		"required": []string{"playground_id", "action_type"},
		"properties": map[string]any{
			"playground_id": namedIdentifierSchema("playground_id", "Playground ID or slug-safe name."),
			"action_type":   map[string]any{"type": "string", "enum": fibe.ValidPlaygroundActions, "description": "Lifecycle action to perform."},
			"force":         map[string]any{"type": "boolean", "description": "Bypass normal state guards when Rails permits forced execution."},
		},
	}
	agentCreate := withPropertyEnum(paramsSchema[fibe.AgentCreateParams]("name", "provider"), "provider", fibe.ValidProviders)
	out["agent"]["create"] = withPropertyEnum(agentCreate, "mode", []string{"oauth", "provider-api-key", "fibe-mana"})
	out["agent"]["update"] = withPropertyEnum(updateParamsSchemaFor[fibe.AgentUpdateParams]("agent_id"), "mode", []string{"oauth", "provider-api-key", "fibe-mana"})
	out["artefact"]["create"] = artefactCreateSchema()
	out["mutter"]["create"] = mutterCreateSchema()
	out["playspec"]["create"] = paramsSchema[fibe.PlayspecCreateParams]("name", "base_compose_yaml")
	out["playspec"]["update"] = updateParamsSchemaFor[fibe.PlayspecUpdateParams]("playspec_id")
	out["prop"]["create"] = withPropertyEnum(paramsSchema[fibe.PropCreateParams]("repository_url"), "provider", []string{"github", "gitea"})
	out["prop"]["update"] = withPropertyEnum(updateParamsSchemaFor[fibe.PropUpdateParams]("prop_id"), "provider", []string{"github", "gitea"})
	out["marquee"]["create"] = marqueeCreateSchema()
	out["marquee"]["update"] = updateParamsSchemaFor[fibe.MarqueeUpdateParams]("marquee_id")
	out["marquee"]["autoconnect_token"] = marqueeAutoconnectTokenSchema()
	out["marquee"]["generate_ssh_key"] = resourceActionIDSchema("marquee_id", "Marquee ID used to generate a new SSH keypair.")
	out["marquee"]["test_connection"] = resourceActionIDSchema("marquee_id", "Marquee ID used to run the connection test.")
	out["prop"]["attach"] = propAttachSchema()
	out["prop"]["mirror"] = propMirrorSchema()
	out["prop"]["sync"] = resourceActionIDSchema("prop_id", "Prop ID to synchronize with its git remote.")
	out["secret"]["create"] = paramsSchema[fibe.SecretCreateParams]("key", "value")
	out["secret"]["update"] = updateParamsSchemaFor[fibe.SecretUpdateParams]("secret_id")
	out["api_key"]["create"] = paramsSchema[fibe.APIKeyCreateParams]("label")
	out["webhook"]["create"] = webhookCreateSchema()
	out["webhook"]["update"] = webhookUpdateSchema()
	out["webhook"]["event_types"] = webhookEventTypesSchema()
	out["webhook"]["test"] = resourceActionIDSchema("webhook_id", "Webhook endpoint ID to receive a simulated test delivery.")
	out["compose"]["validate"] = composeValidateSchema()
	out["template"]["create"] = paramsSchema[fibe.ImportTemplateCreateParams]("name", "category_id", "template_body")
	out["template"]["update"] = templateUpdateSchema()
	out["template"]["develop"] = templateDevelopSchema()
	out["template"]["fork"] = resourceActionIDSchema("template_id", "Template ID to fork into a new standalone template.")
	out["template"]["source_refresh"] = resourceActionIDSchema("template_id", "Template ID whose tracked source file should be refreshed.")
	out["template"]["source_set"] = templateSourceSetSchema()
	out["template"]["upgrade_playspecs"] = templateUpgradePlayspecsSchema()
	out["template_version"]["create"] = templateVersionCreateSchema()
	out["template_version"]["toggle_public"] = templateVersionTogglePublicSchema()
	out["trick"]["trigger"] = paramsSchema[fibe.TrickTriggerParams]("playspec_id")
	out["trick"]["rerun"] = resourceActionIDSchema("trick_id", "Source trick ID to rerun.")
	out["job_env"]["create"] = paramsSchema[fibe.JobEnvSetParams]("key", "value")
	out["job_env"]["update"] = updateParamsSchemaFor[fibe.JobEnvUpdateParams]("job_env_id")
	out["memory"]["memorize"] = MemoryMemorizeSchema()

	for _, r := range flatResources {
		if r.listSchema != nil {
			out[r.name]["list"] = resourceListSchema(r.name, r.listSchema)
		}
		if r.get {
			out[r.name]["get"] = resourceIDSchema(r.name, "fibe_resource_get")
		}
		if r.delete {
			out[r.name]["delete"] = resourceIDSchema(r.name, "fibe_resource_delete")
		}
	}
	overrideResourceIDDescription(out, "artefact_attachment", "get", "Artefact ID whose single file attachment should be downloaded.")
	overrideResourceIDDescription(out, "template_source", "delete", "Template ID whose tracked source configuration should be cleared.")
	overrideResourceIDDescription(out, "template_version", "delete", "Template version ID to delete.")
	return out
}

func paramsSchema[P any](required ...string) map[string]any {
	var p P
	return structSchema(reflect.TypeOf(p), required...)
}

func updateParamsSchema[P any]() map[string]any {
	return updateParamsSchemaFor[P]("id")
}

func updateParamsSchemaFor[P any](idField string) map[string]any {
	if idField == "" {
		idField = "id"
	}
	schema := paramsSchema[P]()
	props := schema["properties"].(map[string]any)
	if namedIdentifierField(idField) {
		props[idField] = namedIdentifierSchema(idField, schemaIDDescription(idField)+" or slug-safe name.")
	} else {
		props[idField] = map[string]any{"type": "integer", "description": schemaIDDescription(idField) + ".", "minimum": 1}
	}
	schema["required"] = []string{idField}
	return schema
}

func artefactCreateSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"agent_id"},
		"properties": map[string]any{
			"agent_id":       map[string]any{"type": "integer", "description": "Agent ID that owns the artefact.", "minimum": 1},
			"name":           map[string]any{"type": "string", "description": "Artefact display name. Also used as the filename fallback by artefact.create."},
			"filename":       map[string]any{"type": "string", "description": "Filename for the uploaded artefact. Defaults to name when omitted."},
			"content_base64": map[string]any{"type": "string", "description": "Base64-encoded file content. Use either content_base64 or content_path."},
			"content_path":   map[string]any{"type": "string", "description": "Absolute local file path to read and upload. Use either content_path or content_base64."},
			"description":    map[string]any{"type": "string", "description": "Optional human-readable artefact description."},
			"playground_id":  namedIdentifierSchema("playground_id", "Optional playground ID or slug-safe name to associate with the artefact."),
		},
	}
}

func mutterCreateSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"agent_id", "type", "body"},
		"properties": map[string]any{
			"agent_id":      map[string]any{"type": "integer", "description": "Agent ID that owns the mutter.", "minimum": 1},
			"type":          map[string]any{"type": "string", "description": "Mutter type label. Common values are info, warning, error, and success; Server accepts arbitrary strings."},
			"body":          map[string]any{"type": "string", "description": "Mutter body text."},
			"playground_id": namedIdentifierSchema("playground_id", "Optional playground ID or slug-safe name to associate with the mutter."),
		},
	}
}

func marqueeCreateSchema() map[string]any {
	schema := paramsSchema[fibe.MarqueeCreateParams]("name", "host", "port", "user", "ssh_private_key")
	props := schema["properties"].(map[string]any)
	if port, ok := props["port"].(map[string]any); ok {
		port["description"] = "SSH port. The Rails UI defaults to 22."
		port["minimum"] = 1
		port["maximum"] = 65535
		port["default"] = 22
	}
	return schema
}

func marqueeAutoconnectTokenSchema() map[string]any {
	schema := paramsSchema[fibe.AutoconnectTokenParams]()
	props := schema["properties"].(map[string]any)
	props["email"].(map[string]any)["description"] = "Email address to place in the generated autoconnect token payload."
	props["domain"].(map[string]any)["description"] = "Domain name to place in the generated autoconnect token payload."
	props["ip"].(map[string]any)["description"] = "Public IP address to place in the generated autoconnect token payload."
	if sslMode, ok := props["ssl_mode"].(map[string]any); ok {
		sslMode["enum"] = []string{"http", "dns"}
		sslMode["description"] = "SSL setup mode for the generated connect script."
	}
	props["dns_provider"].(map[string]any)["description"] = "DNS provider name to place in the generated autoconnect token payload."
	props["dns_credentials"].(map[string]any)["description"] = "DNS provider credential values to include in the generated autoconnect token payload."
	return schema
}

func propAttachSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"repo_full_name"},
		"properties": map[string]any{
			"repo_full_name": map[string]any{"type": "string", "description": "GitHub repository owner/name to attach, for example octocat/Hello-World."},
		},
	}
}

func propMirrorSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"source_url"},
		"properties": map[string]any{
			"source_url": map[string]any{"type": "string", "format": "uri", "description": "Source GitHub repository URL to mirror."},
			"name":       map[string]any{"type": "string", "description": "Optional name for the mirrored repository. If omitted, Rails infers it from the URL."},
		},
	}
}

func templateSourceSetSchema() map[string]any {
	schema := paramsSchema[fibe.ImportTemplateSourceParams]("source_prop_id", "source_path")
	props := schema["properties"].(map[string]any)
	props["template_id"] = map[string]any{"type": "integer", "description": "Template ID whose tracked source should be configured.", "minimum": 1}
	props["source_prop_id"] = namedIdentifierSchema("source_prop_id", "Source Prop ID or slug-safe name containing the template YAML file.")
	props["source_path"].(map[string]any)["description"] = "Path to the source YAML file inside the Prop repository."
	props["source_ref"].(map[string]any)["description"] = "Source branch, tag, or git ref."
	props["source_auto_refresh"].(map[string]any)["description"] = "Refresh the template when matching source changes are detected."
	props["source_auto_upgrade"].(map[string]any)["description"] = "Automatically upgrade linked job Playspecs after a source refresh creates a new version."
	props["ci_enabled"].(map[string]any)["description"] = "Enable CI workflow sync for this template source."
	props["ci_marquee_id"] = namedIdentifierSchema("ci_marquee_id", "Marquee ID or slug-safe name used by CI workflow sync.")
	delete(props, "marquee_id")
	schema["required"] = []string{"template_id", "source_prop_id", "source_path"}
	return schema
}

func templateUpgradePlayspecsSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"template_id", "version_id"},
		"properties": map[string]any{
			"template_id": map[string]any{"type": "integer", "description": "Template ID whose linked job Playspecs should be upgraded.", "minimum": 1},
			"version_id":  map[string]any{"type": "integer", "description": "Target template version ID.", "minimum": 1},
		},
	}
}

func templateVersionTogglePublicSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"template_id", "version_id"},
		"properties": map[string]any{
			"template_id": map[string]any{"type": "integer", "description": "Template ID that owns the version.", "minimum": 1},
			"version_id":  map[string]any{"type": "integer", "description": "Template version ID whose public visibility should be toggled.", "minimum": 1},
		},
	}
}

func resourceActionIDSchema(idField, description string) map[string]any {
	prop := map[string]any{"type": "integer", "description": description, "minimum": 1}
	if namedIdentifierField(idField) {
		prop = namedIdentifierSchema(idField, strings.Replace(description, " ID", " ID or slug-safe name", 1))
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{idField},
		"properties": map[string]any{
			idField: prop,
		},
	}
}

func MemoryMemorizeSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"conversation_id", "content"},
		"properties": map[string]any{
			"conversation_id": map[string]any{
				"type":        "string",
				"minLength":   1,
				"description": "Stable local source conversation UUID. This is not a Rails database ID.",
			},
			"content": map[string]any{
				"type":        "string",
				"minLength":   1,
				"description": "Durable memory text.",
			},
			"agent_id": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "Optional Rails Agent ID that created the memory. fibe_memorize fills this from FIBE_AGENT_ID when available.",
			},
			"tags": map[string]any{
				"type":        "array",
				"description": "Memory tags. The server normalizes tags to lowercase slug-like strings.",
				"items":       map[string]any{"type": "string"},
			},
			"confidence": map[string]any{
				"type":        "number",
				"minimum":     0,
				"maximum":     1,
				"description": "Confidence from 0 to 1.",
			},
			"memory_key": map[string]any{
				"type":        "string",
				"description": "Optional exact idempotency key. Omit this and Rails computes one from content, tags, conversation_id, and groundings.",
			},
			"metadata": map[string]any{
				"type":        "object",
				"description": "Optional memory metadata.",
			},
			"groundings": memoryGroundingsSchema(),
		},
	}
}

func memoryGroundingsSchema() map[string]any {
	return map[string]any{
		"type":        "array",
		"description": "Proof references into normalized messages or raw provider events.",
		"items": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"message_position": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"description": "Zero-based source message position.",
				},
				"provider_message_uuid": map[string]any{
					"type":        "string",
					"description": "Provider message UUID when available.",
				},
				"raw_event_index": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"description": "Zero-based raw event index when grounding points to raw events.",
				},
				"start_character": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"description": "Start character offset within the normalized message content.",
				},
				"end_character": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"description": "End character offset within the normalized message content.",
				},
				"raw_start_character": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"description": "Start character offset within raw content or raw event text.",
				},
				"raw_end_character": map[string]any{
					"type":        "integer",
					"minimum":     0,
					"description": "End character offset within raw content or raw event text.",
				},
				"quote": map[string]any{
					"type":        "string",
					"description": "Short proof excerpt. Maximum 2000 characters on the server.",
				},
				"metadata": map[string]any{
					"type":        "object",
					"description": "Optional grounding metadata.",
				},
			},
		},
	}
}

func templateUpdateSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"template_id"},
		"properties": map[string]any{
			"template_id":    map[string]any{"type": "integer", "description": "Template ID.", "minimum": 1},
			"name":           map[string]any{"type": "string", "description": "New template name."},
			"description":    map[string]any{"type": "string", "description": "New template description."},
			"category_id":    map[string]any{"type": "integer", "description": "New template category ID.", "minimum": 1},
			"filename":       map[string]any{"type": "string", "description": "Image filename when uploading a cover image. Defaults to cover.png in template.update."},
			"image_data":     map[string]any{"type": "string", "description": "Base64-encoded cover image data."},
			"content_base64": map[string]any{"type": "string", "description": "Alias for image_data."},
			"content_path":   map[string]any{"type": "string", "description": "Absolute local path to a cover image file."},
			"content_type":   map[string]any{"type": "string", "enum": []string{"image/jpeg", "image/png", "image/svg+xml", "image/webp"}, "description": "Cover image MIME type. If omitted, Rails infers it from filename."},
		},
	}
}

func templateVersionCreateSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"template_id"},
		"anyOf": []any{
			map[string]any{"required": []string{"template_body"}},
			map[string]any{"required": []string{"template_body_path"}},
		},
		"properties": map[string]any{
			"template_id":        map[string]any{"type": "integer", "description": "Template ID.", "minimum": 1},
			"template_body":      map[string]any{"type": "string", "description": "Template YAML body."},
			"template_body_path": map[string]any{"type": "string", "description": "Absolute local path to a template YAML file. Use either template_body or template_body_path."},
			"public":             map[string]any{"type": "boolean", "description": "Make this version public."},
			"changelog":          map[string]any{"type": "string", "description": "Human-readable changelog for the created template version."},
			"response_mode":      map[string]any{"type": "string", "enum": []string{"summary", "full"}, "description": "Response detail mode."},
		},
	}
}

func templateVersionPatchCreateSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"template_id", "base_version_id"},
		"properties": map[string]any{
			"template_id":          map[string]any{"type": "integer", "description": "Template ID.", "minimum": 1},
			"base_version_id":      map[string]any{"type": "integer", "description": "Exact template version ID to patch.", "minimum": 1},
			"patches":              map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Patch entries: YAML path set/remove or exact search/replace. YAML path entries support expect, create_missing, and allow_missing."},
			"edits":                map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Alias for patches."},
			"public":               map[string]any{"type": "boolean", "description": "Make the created version public."},
			"changelog":            map[string]any{"type": "string", "description": "Human-readable changelog for the created template version."},
			"target_playspec_id":   namedIdentifierSchema("target_playspec_id", "Target playspec ID or slug-safe name for optional auto-switch."),
			"target_playground_id": namedIdentifierSchema("target_playground_id", "Target playground ID or slug-safe name for optional rollout."),
			"switch_variables":     map[string]any{"type": "object", "description": "Variables to pass to version switch."},
			"regenerate_variables": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Template variable names to regenerate during switch."},
			"confirm_warnings":     map[string]any{"type": "boolean", "description": "Set true to continue when preview warnings are present."},
			"auto_switch":          map[string]any{"type": "boolean", "description": "Switch target_playspec_id to the created version."},
			"response_mode":        map[string]any{"type": "string", "enum": []string{"summary", "full"}, "description": "Response detail mode."},
		},
	}
}

func webhookEventTypesSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"event_type": map[string]any{
				"type":        "string",
				"enum":        append([]string(nil), fibe.WebhookKnownEvents...),
				"description": "One webhook event type accepted by webhook create/update events.",
			},
			"event_types": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string", "enum": append([]string(nil), fibe.WebhookKnownEvents...)},
				"description": "All webhook event types currently known by this SDK build.",
			},
		},
		"event_types": append([]string(nil), fibe.WebhookKnownEvents...),
	}
}

func composeValidateSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"anyOf": []any{
			map[string]any{"required": []string{"compose_yaml"}},
			map[string]any{"required": []string{"compose_path"}},
		},
		"properties": map[string]any{
			"compose_yaml": map[string]any{"type": "string", "description": "Docker Compose YAML content to validate."},
			"compose_path": map[string]any{"type": "string", "description": "Absolute local path to a Docker Compose YAML file. Local MCP only."},
			"target_type":  map[string]any{"type": "string", "enum": []string{"playspec", "template", "trick"}, "description": "Validation target. Use trick for job-mode playspec/template constraints."},
			"job_mode":     map[string]any{"type": "boolean", "description": "Apply job-mode validation rules: at least one watched service and no exposed services."},
		},
	}
}

func templateDevelopSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"target_type", "target_id", "mode", "change_type"},
		"properties": map[string]any{
			"target_type":                map[string]any{"type": "string", "enum": []string{"template", "playspec", "playground", "trick"}, "description": "Object to start from: template, playspec, normal playground, or job-mode trick."},
			"target_id":                  map[string]any{"type": "integer", "description": "ID of the target object.", "minimum": 1},
			"mode":                       map[string]any{"type": "string", "enum": []string{"preview", "apply"}, "description": "Preview validates and diffs without writes; apply creates/switches resources."},
			"change_type":                map[string]any{"type": "string", "enum": []string{"patch", "overwrite", "switch_existing"}, "description": "Template change workflow to run."},
			"base_version_id":            map[string]any{"type": "integer", "description": "Base template version ID for patch or overwrite. Defaults from target when possible.", "minimum": 1},
			"target_template_version_id": map[string]any{"type": "integer", "description": "Existing template version ID for switch_existing.", "minimum": 1},
			"patches":                    map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Patch entries: YAML path set/remove or exact search/replace."},
			"edits":                      map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Alias for patches."},
			"template_body":              map[string]any{"type": "string", "description": "Full replacement template YAML for overwrite."},
			"template_body_path":         map[string]any{"type": "string", "description": "Absolute local path to full replacement template YAML."},
			"changelog":                  map[string]any{"type": "string", "description": "Human-readable changelog for a created template version."},
			"public":                     map[string]any{"type": "boolean", "description": "Make the created template version public."},
			"switch_variables":           map[string]any{"type": "object", "description": "Template variables to use when switching a playspec."},
			"regenerate_variables":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Variable names to regenerate while switching."},
			"confirm_warnings":           map[string]any{"type": "boolean", "description": "Allow apply when preview reports switch warnings."},
			"post_apply":                 map[string]any{"type": "string", "enum": []string{"none", "rollout_target", "rollout_all", "trigger_trick"}, "description": "Optional action after apply. Tricks should use trigger_trick; normal playgrounds can use rollout_target or rollout_all."},
			"wait":                       map[string]any{"type": "boolean", "description": "Wait for rollout targets or triggered trick completion."},
			"wait_timeout_seconds":       map[string]any{"type": "integer", "description": "Maximum seconds to wait.", "minimum": 1},
			"diagnose_on_failure":        map[string]any{"type": "boolean", "description": "Attach playground diagnostics when a wait fails."},
			"response_mode":              map[string]any{"type": "string", "enum": []string{"summary", "full"}, "description": "Response detail mode."},
		},
	}
}

func overrideResourceIDDescription(registry map[string]map[string]any, resource, operation, description string) {
	schema, ok := registry[resource][operation].(map[string]any)
	if !ok {
		return
	}
	props, _ := schema["properties"].(map[string]any)
	id, _ := props["id"].(map[string]any)
	if id != nil {
		id["description"] = description
	}
}

func schemaMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = cloneValue(item)
		}
		return out
	case []string:
		return append([]string(nil), x...)
	case []int:
		return append([]int(nil), x...)
	default:
		return x
	}
}

func stringSliceToAny(values []string) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func webhookCreateSchema() map[string]any {
	schema := paramsSchema[fibe.WebhookEndpointCreateParams]("url", "events")
	return withWebhookEventDescription(schema)
}

func webhookUpdateSchema() map[string]any {
	schema := updateParamsSchemaFor[fibe.WebhookEndpointUpdateParams]("webhook_id")
	return withWebhookEventDescription(schema)
}

func withWebhookEventDescription(schema map[string]any) map[string]any {
	props, _ := schema["properties"].(map[string]any)
	events, _ := props["events"].(map[string]any)
	if events == nil {
		return schema
	}
	events["description"] = "Event types to subscribe to. Inspect fibe_schema(resource:webhook, operation:event_types) for the SDK's static enum; unknown values fail with VALIDATION_FAILED."
	events["items"] = map[string]any{"type": "string", "enum": append([]string(nil), fibe.WebhookKnownEvents...)}
	events["examples"] = []string{
		"agent.created", "agent.updated", "agent.destroyed",
		"playground.created", "playground.status.changed", "playground.error",
		"playground.completed", "webhook.test",
	}
	return schema
}

func withPropertyEnum(schema map[string]any, property string, values []string) map[string]any {
	props, _ := schema["properties"].(map[string]any)
	prop, _ := props[property].(map[string]any)
	if prop == nil {
		return schema
	}
	prop["enum"] = append([]string(nil), values...)
	prop["type"] = "string"
	return schema
}

func resourceListSchema(resource string, params map[string]any) map[string]any {
	required := []string{"resource"}
	if len(requiredFields(params)) > 0 {
		required = append(required, "params")
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties": map[string]any{
			"resource": map[string]any{
				"type":        "string",
				"enum":        []string{resource},
				"description": "Canonical resource name for fibe_resource_list.",
			},
			"params": withDescription(params, "Resource-specific list filters."),
		},
	}
}

func withDescription(schema map[string]any, description string) map[string]any {
	if schema["description"] == nil {
		schema["description"] = description
	}
	return schema
}

func resourceIDSchema(resource, tool string) map[string]any {
	required := []string{"resource", "id"}
	properties := map[string]any{
		"resource": map[string]any{
			"type":        "string",
			"enum":        []string{resource},
			"description": fmt.Sprintf("Canonical resource name for %s.", tool),
		},
		"id": map[string]any{
			"type":        "integer",
			"description": "ID of the selected resource.",
			"minimum":     1,
		},
	}
	if namedResource(resource) {
		required = []string{"resource"}
		properties["identifier"] = map[string]any{
			"type":        "string",
			"minLength":   1,
			"description": "Numeric ID or slug-safe resource name. Use when identifying playgrounds, tricks, playspecs, props, and marquees by name.",
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties":           properties,
	}
	if namedResource(resource) {
		schema["anyOf"] = []any{
			map[string]any{"required": []string{"id"}},
			map[string]any{"required": []string{"identifier"}},
		}
	}
	return schema
}

func listParamsSchema[P any]() map[string]any {
	var p P
	return structSchema(reflect.TypeOf(p))
}

func templateVersionListParamsSchema() map[string]any {
	schema := listParamsSchema[fibe.ListParams]()
	props := schema["properties"].(map[string]any)
	props["template_id"] = map[string]any{
		"type":        "integer",
		"description": "Import template ID whose versions should be listed.",
		"minimum":     1,
	}
	schema["required"] = []string{"template_id"}
	return schema
}

func webhookDeliveryListParamsSchema() map[string]any {
	schema := listParamsSchema[fibe.ListParams]()
	props := schema["properties"].(map[string]any)
	props["webhook_id"] = map[string]any{
		"type":        "integer",
		"description": "Webhook endpoint ID whose delivery attempts should be listed.",
		"minimum":     1,
	}
	schema["required"] = []string{"webhook_id"}
	return schema
}

func structSchema(t reflect.Type, required ...string) map[string]any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	props := map[string]any{}
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			name := fieldSchemaName(field)
			if name == "" || name == "-" {
				continue
			}
			prop := schemaForType(field.Type)
			enrichPropertySchema(name, prop)
			props[name] = prop
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           props,
	}
	if len(required) > 0 {
		schema["required"] = append([]string(nil), required...)
	}
	return schema
}

func enrichPropertySchema(name string, prop map[string]any) {
	if namedIdentifierField(name) && (schemaHasType(prop, "integer") || schemaHasType(prop, "number") || schemaHasType(prop, "string")) {
		convertNamedIdentifierSchema(name, prop)
	}
	if schemaFieldIsNumericID(name, prop) {
		prop["minimum"] = 1
	}
	if min, ok := schemaFieldMinimum(name, prop); ok {
		if _, exists := prop["minimum"]; !exists {
			prop["minimum"] = min
		}
	}
	if max, ok := schemaFieldMaximum(name, prop); ok {
		if _, exists := prop["maximum"]; !exists {
			prop["maximum"] = max
		}
	}
	if prop["description"] == nil {
		prop["description"] = schemaFieldDescription(name, prop)
	}
	if items, ok := prop["items"].(map[string]any); ok {
		if props, ok := items["properties"].(map[string]any); ok {
			for childName, raw := range props {
				if child, ok := raw.(map[string]any); ok {
					enrichPropertySchema(childName, child)
				}
			}
		}
	}
}

func namedResource(resource string) bool {
	switch resource {
	case "playground", "trick", "playspec", "prop", "marquee":
		return true
	default:
		return false
	}
}

func namedIdentifierField(name string) bool {
	switch name {
	case "playground_id", "target_playground_id", "build_in_public_playground_id",
		"trick_id",
		"playspec_id", "target_playspec_id",
		"prop_id", "source_prop_id",
		"marquee_id", "ci_marquee_id", "target_marquee_id":
		return true
	default:
		return false
	}
}

func namedIdentifierSchema(name string, description string) map[string]any {
	if description == "" {
		description = schemaIDDescription(name) + " or slug-safe name."
	}
	return map[string]any{
		"oneOf": []any{
			map[string]any{"type": "integer", "minimum": 1},
			map[string]any{"type": "string", "minLength": 1},
		},
		"description": description,
	}
}

func convertNamedIdentifierSchema(name string, prop map[string]any) {
	desc, _ := prop["description"].(string)
	if desc == "" {
		desc = schemaIDDescription(name) + " or slug-safe name."
	}
	for _, key := range []string{"type", "minimum", "maximum", "format"} {
		delete(prop, key)
	}
	prop["oneOf"] = []any{
		map[string]any{"type": "integer", "minimum": 1},
		map[string]any{"type": "string", "minLength": 1},
	}
	prop["description"] = desc
}

func schemaFieldIsNumericID(name string, prop map[string]any) bool {
	if name != "id" && !strings.HasSuffix(name, "_id") {
		return false
	}
	return schemaHasType(prop, "integer") || schemaHasType(prop, "number")
}

func schemaFieldMinimum(name string, prop map[string]any) (int, bool) {
	if !(schemaHasType(prop, "integer") || schemaHasType(prop, "number")) {
		return 0, false
	}
	switch name {
	case "page", "per_page", "tail", "logs_tail", "max_lines", "max_events", "content_limit", "timeout_ms", "wait_timeout_seconds", "port":
		return 1, true
	}
	return 0, false
}

func schemaFieldMaximum(name string, prop map[string]any) (int, bool) {
	if !(schemaHasType(prop, "integer") || schemaHasType(prop, "number")) {
		return 0, false
	}
	if name == "port" {
		return 65535, true
	}
	return 0, false
}

func schemaHasType(schema map[string]any, want string) bool {
	switch typ := schema["type"].(type) {
	case string:
		return typ == want
	case []any:
		for _, value := range typ {
			if value == want {
				return true
			}
		}
	case []string:
		for _, value := range typ {
			if value == want {
				return true
			}
		}
	}
	return false
}

func schemaFieldDescription(name string, prop map[string]any) string {
	if schemaFieldIsNumericID(name, prop) {
		return schemaIDDescription(name) + "."
	}
	if desc, ok := schemaFieldDescriptions[name]; ok {
		return desc
	}
	if strings.HasSuffix(name, "_path") {
		return humanizeSchemaFieldName(name) + " path."
	}
	if strings.HasSuffix(name, "_url") {
		return humanizeSchemaFieldName(name) + " URL."
	}
	return humanizeSchemaFieldName(name) + "."
}

func schemaIDDescription(name string) string {
	switch name {
	case "id":
		return "ID of the selected resource"
	case "api_key_id":
		return "API key ID"
	case "agent_id":
		return "Agent ID"
	case "artefact_id":
		return "Artefact ID"
	case "base_version_id":
		return "Base template version ID"
	case "build_in_public_playground_id":
		return "Build-in-public playground ID"
	case "category_id":
		return "Template category ID"
	case "ci_marquee_id":
		return "CI marquee ID"
	case "feedback_id":
		return "Feedback ID"
	case "job_env_id":
		return "Job environment variable ID"
	case "marquee_id", "target_marquee_id":
		return "Marquee ID"
	case "playground_id":
		return "Playground ID"
	case "playspec_id", "target_playspec_id":
		return "Playspec ID"
	case "prop_id", "source_prop_id":
		return "Prop ID"
	case "secret_id":
		return "Secret ID"
	case "source_template_version_id":
		return "Source template version ID"
	case "template_id":
		return "Template ID"
	case "target_template_version_id", "version_id":
		return "Template version ID"
	case "trick_id":
		return "Trick ID"
	case "webhook_id":
		return "Webhook endpoint ID"
	default:
		return humanizeSchemaFieldName(name)
	}
}

var schemaFieldDescriptions = map[string]string{
	"agent_accessible":       "Whether the API key is accessible to agents.",
	"auto_switch":            "Switch the target playspec to the created version after patch creation.",
	"base_compose_yaml":      "Docker Compose YAML used as the playspec base.",
	"build_in_public":        "Allow the agent to build in a public playground.",
	"build_overrides_yaml":   "Per-service build override YAML values.",
	"build_platform":         "Target Docker build platform.",
	"changelog":              "Human-readable changelog for the created template version.",
	"confirm_warnings":       "Set true to continue when preview warnings are present.",
	"content_base64":         "Base64-encoded file content.",
	"content_path":           "Absolute local path to read content from.",
	"content_type":           "MIME content type.",
	"cpu_limit":              "CPU limit for the agent runtime.",
	"cli_version":            "Fibe CLI version pin for the agent runtime.",
	"credentials":            "Provider-specific credential payload.",
	"custom_env":             "KEY=VALUE lines injected into the agent runtime.",
	"default_branch":         "Default git branch.",
	"description":            "Optional description.",
	"dns_credentials":        "DNS provider credentials.",
	"dns_provider":           "DNS provider name.",
	"docker_compose_yaml":    "Docker Compose YAML content.",
	"dockerhub_auth_enabled": "Enable Docker Hub authentication.",
	"dockerhub_token":        "Docker Hub access token.",
	"dockerhub_username":     "Docker Hub username.",
	"domains_input":          "Domain list or domain configuration input.",
	"enabled":                "Whether the resource is enabled.",
	"env_overrides":          "Environment variable overrides for a playground launch.",
	"event_filters":          "Webhook event filter object.",
	"events":                 "Webhook event names.",
	"expires_at":             "Expiration time. For playgrounds, this is when the playground is automatically deleted; for API keys, this is when the key stops working.",
	"filename":               "File name.",
	"granular_scopes":        "Fine-grained API key scopes keyed by resource.",
	"host":                   "Host name or IP address.",
	"key":                    "Environment variable or secret key.",
	"label":                  "Human label shown for this API key.",
	"mcp_json":               "MCP configuration JSON.",
	"memory_limit":           "Memory limit for the agent runtime.",
	"mode":                   "Agent authentication mode.",
	"model_options":          "Provider-specific model options.",
	"mount_path":             "Container mount path.",
	"mounts":                 "Mounted file specifications.",
	"name":                   "Name.",
	"never_expire":           "Prevent the playground from expiring automatically.",
	"page":                   "Page number.",
	"patches":                "Patch operations to preview or apply.",
	"per_page":               "Number of results per page.",
	"persist_volumes":        "Persist Docker volumes for the playspec.",
	"port":                   "Network port.",
	"post_init_script":       "Shell script run after agent initialization.",
	"private":                "Whether the repository is private.",
	"provider":               "Provider name.",
	"provider_api_key_mode":  "Use provider API key mode for the agent.",
	"provider_args":          "Provider-specific runtime arguments.",
	"provider_args_cli":      "Provider runtime arguments encoded as CLI flags.",
	"prompt":                 "Initial prompt or instructions for the agent.",
	"public":                 "Whether the template version is public.",
	"q":                      "Search query.",
	"query":                  "Search query.",
	"readonly":               "Mount the file as read-only.",
	"regenerate_variables":   "Template variable names to regenerate.",
	"response_mode":          "Response detail mode.",
	"scopes":                 "API key scopes.",
	"secret":                 "Whether the value is secret, or a webhook signing secret.",
	"service_subdomains":     "Per-service subdomain overrides.",
	"service":                "Compose service name.",
	"services":               "Per-service configuration.",
	"sort":                   "Sort expression.",
	"source_type":            "Source object type.",
	"ssh_private_key":        "SSH private key.",
	"status":                 "Resource status filter or target playground status.",
	"sync_enabled":           "Enable repository sync for the agent.",
	"sync_skills_enabled":    "Enable skill sync for the agent.",
	"syscheck_enabled":       "Enable system checks for the agent.",
	"skill_toggles":          "Per-skill enabled/disabled overrides.",
	"target_services":        "Compose service names affected by this file or operation.",
	"template_body":          "Import template YAML body.",
	"tool_filters":           "Webhook tool filter object.",
	"trigger_config":         "Trigger configuration object.",
	"url":                    "HTTP URL.",
	"user":                   "SSH username.",
	"variables":              "Template variables used while rendering a template.",
	"type":                   "Type value.",
	"value":                  "Secret or job environment value to store.",
}

func fieldSchemaName(field reflect.StructField) string {
	if name := tagName(field.Tag.Get("url")); name != "" {
		return name
	}
	if name := tagName(field.Tag.Get("json")); name != "" {
		return name
	}
	return toSnakeCase(field.Name)
}

func schemaForType(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return schemaForType(t.Elem())
	}
	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaForType(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object"}
	case reflect.Struct:
		return structSchema(t)
	default:
		return map[string]any{"type": "string"}
	}
}

func tagName(raw string) string {
	if raw == "" {
		return ""
	}
	name, _, _ := strings.Cut(raw, ",")
	return name
}

func toSnakeCase(s string) string {
	var b strings.Builder
	var prevLower bool
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && prevLower {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
			prevLower = false
		} else {
			prevLower = unicode.IsLower(r) || unicode.IsDigit(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func humanizeSchemaFieldName(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	for i, word := range words {
		switch strings.ToLower(word) {
		case "id":
			words[i] = "ID"
		case "api":
			words[i] = "API"
		case "ci":
			words[i] = "CI"
		case "cpu":
			words[i] = "CPU"
		case "dns":
			words[i] = "DNS"
		case "json":
			words[i] = "JSON"
		case "mcp":
			words[i] = "MCP"
		case "ssh":
			words[i] = "SSH"
		case "url":
			words[i] = "URL"
		default:
			if word == "" {
				continue
			}
			runes := []rune(strings.ToLower(word))
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

func MarshalJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func SortedOperationNames(resource string) []string {
	schemas, ok := registry[resource]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(schemas))
	for op := range schemas {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}
