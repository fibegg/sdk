package mcpserver

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func enrichToolInputSchema(toolName string, tool *mcp.Tool) {
	schema := toolInputSchemaToMap(*tool)
	m, ok := schema.(map[string]any)
	if !ok {
		return
	}
	injectGlobalResponseShapeProperties(toolName, m)
	enrichSchemaProperties(toolName, m)
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	tool.InputSchema.Type = ""
	tool.RawInputSchema = data
}

func injectGlobalResponseShapeProperties(toolName string, schema map[string]any) {
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
		schema["properties"] = props
	}
	if _, exists := props[responseOnlyArg]; !exists {
		props[responseOnlyArg] = map[string]any{
			"type":        "array",
			"description": `Return only these top-level fields from each result item. Example: only: ["uuid","title","project"] on local conversations keeps envelope metadata but trims each conversation.`,
			"items": map[string]any{
				"type": "string",
			},
		}
	}
	if suppressOutputPath(toolName) {
		return
	}
	if _, exists := props[responseOutputPathArg]; !exists {
		props[responseOutputPathArg] = map[string]any{
			"type":        "string",
			"description": `JSONPath into the tool result, not a filesystem path. Example: "$.conversations[0].uuid" returns the first UUID; "$.conversations" returns only the array.`,
		}
	}
}

func suppressOutputPath(toolName string) bool {
	switch toolName {
	case "fibe_local_conversations_get_message", "fibe_memorize":
		return true
	default:
		return false
	}
}

func enrichSchemaProperties(toolName string, schema map[string]any) {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	for name, raw := range props {
		prop, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if enum := enumForToolProperty(toolName, name); len(enum) > 0 {
			if _, exists := prop["enum"]; !exists {
				prop["enum"] = enum
				prop["type"] = "string"
			}
		}
		if isNumericIDSchema(name, prop) {
			prop["minimum"] = 1
		}
		if min, ok := numericMinimumForProperty(name, prop); ok {
			if _, exists := prop["minimum"]; !exists {
				prop["minimum"] = min
			}
		}
		if max, ok := numericMaximumForProperty(name, prop); ok {
			if _, exists := prop["maximum"]; !exists {
				prop["maximum"] = max
			}
		}
		if prop["description"] == nil || strings.TrimSpace(fmt.Sprint(prop["description"])) == "" {
			prop["description"] = descriptionForToolProperty(toolName, name, prop)
		}
		enrichSchemaProperties(toolName, prop)
		if items, ok := prop["items"].(map[string]any); ok {
			enrichSchemaProperties(toolName, items)
		}
	}
}

func enumForToolProperty(toolName, name string) []string {
	switch name {
	case "action_type":
		if toolName == "fibe_playgrounds_action" {
			return fibe.ValidPlaygroundActions
		}
	case "content_type":
		if toolName == "fibe_templates_update" {
			return []string{"image/jpeg", "image/png", "image/svg+xml", "image/webp"}
		}
	case "tier":
		if toolName == "fibe_tools_catalog" {
			return []string{"meta", "base", "greenfield", "brownfield", "overseer", "local", "other", "core", "full", "all"}
		}
	case "provider":
		switch {
		case strings.HasPrefix(toolName, "fibe_agents_"):
			return fibe.ValidProviders
		case strings.HasPrefix(toolName, "fibe_props_"):
			return []string{"github", "gitea"}
		}
	case "git_provider":
		return []string{"gitea", "github"}
	case "mode":
		switch {
		case strings.HasPrefix(toolName, "fibe_agents_"):
			return []string{"oauth", "provider-api-key", "fibe-mana"}
		case toolName == "fibe_playgrounds_debug":
			return []string{"summary", "full"}
		}
	case "response_mode":
		return []string{"summary", "full"}
	}
	return nil
}

func isNumericIDSchema(name string, schema map[string]any) bool {
	if !isIDFieldName(name) {
		return false
	}
	return schemaHasType(schema, "integer") || schemaHasType(schema, "number")
}

func numericMinimumForProperty(name string, schema map[string]any) (int, bool) {
	if !(schemaHasType(schema, "integer") || schemaHasType(schema, "number")) {
		return 0, false
	}
	switch name {
	case "page", "per_page", "tail", "logs_tail", "max_lines", "max_events", "content_limit", "timeout_ms", "wait_timeout_seconds", "port":
		return 1, true
	}
	return 0, false
}

func numericMaximumForProperty(name string, schema map[string]any) (int, bool) {
	if !(schemaHasType(schema, "integer") || schemaHasType(schema, "number")) {
		return 0, false
	}
	if name == "port" {
		return 65535, true
	}
	return 0, false
}

func isIDFieldName(name string) bool {
	return name == "id" || strings.HasSuffix(name, "_id")
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

func descriptionForToolProperty(toolName, name string, schema map[string]any) string {
	if isNumericIDSchema(name, schema) {
		return idDescription(name)
	}
	if desc, ok := knownPropertyDescriptions[name]; ok {
		return desc
	}
	if strings.HasSuffix(name, "_path") {
		return fmt.Sprintf("%s path.", humanizeFieldName(name))
	}
	if strings.HasSuffix(name, "_url") {
		return fmt.Sprintf("%s URL.", humanizeFieldName(name))
	}
	return fmt.Sprintf("%s for this %s request.", humanizeFieldName(name), strings.TrimPrefix(toolName, "fibe_"))
}

func idDescription(name string) string {
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
	case "marquee_id", "target_marquee_id":
		return "Marquee ID"
	case "parent_id":
		return "Parent resource ID"
	case "pipeline_id":
		return "Pipeline ID"
	case "playground_id":
		return "Playground ID"
	case "playspec_id", "target_playspec_id":
		return "Playspec ID"
	case "prop_id", "source_prop_id":
		return "Prop ID"
	case "source_template_version_id":
		return "Source template version ID"
	case "template_id":
		return "Template ID"
	case "target_template_version_id", "version_id":
		return "Template version ID"
	case "webhook_id":
		return "Webhook endpoint ID"
	default:
		return humanizeFieldName(name)
	}
}

var knownPropertyDescriptions = map[string]string{
	"args":                   "Argument object or command tokens for the target operation.",
	"action_type":            "Lifecycle action to perform.",
	"auto_init":              "Initialize the repository with default files.",
	"auto_switch":            "Switch the target playspec to the created version after patch creation.",
	"base_compose_yaml":      "Docker Compose YAML used as the playspec base.",
	"body":                   "Text body to store.",
	"build_in_public":        "Allow the agent to build in a public playground.",
	"build_overrides_yaml":   "Per-service build override YAML values.",
	"build_platform":         "Target Docker build platform.",
	"changelog":              "Human-readable changelog for the created template version.",
	"confirm":                "Set true to confirm a destructive operation unless the server runs with --yolo.",
	"confirm_warnings":       "Set true to continue when preview warnings are present.",
	"content_base64":         "Base64-encoded file content.",
	"content_path":           "Absolute local path to read content from.",
	"content_type":           "MIME content type.",
	"cpu_limit":              "CPU limit for the agent runtime.",
	"credentials":            "Provider-specific credential payload.",
	"description":            "Optional description.",
	"dns_credentials":        "DNS provider credentials.",
	"dns_provider":           "DNS provider name.",
	"docker_compose_yaml":    "Docker Compose YAML content.",
	"dockerhub_auth_enabled": "Enable Docker Hub authentication.",
	"dockerhub_token":        "Docker Hub access token.",
	"dockerhub_username":     "Docker Hub username.",
	"domain":                 "Domain name.",
	"domains_input":          "Domain list or domain configuration input.",
	"edits":                  "Patch edits to apply.",
	"enabled":                "Whether the resource is enabled.",
	"env_overrides":          "Environment variable overrides for a playground launch.",
	"event_filters":          "Webhook event filter object.",
	"events":                 "Webhook event names.",
	"expires_at":             "Expiration time. For playgrounds, this is when the playground is automatically deleted; for API keys, this is when the key stops working.",
	"filename":               "File name.",
	"force":                  "Force the operation when the server permits it.",
	"git_provider":           "Destination git provider.",
	"granular_scopes":        "Fine-grained API key scopes keyed by resource.",
	"host":                   "Host name or IP address.",
	"image_data":             "Base64-encoded image data.",
	"include_schema":         "Include input schemas in the response.",
	"key":                    "Environment variable or secret key.",
	"label":                  "Human label shown for this API key.",
	"logs_tail":              "Number of log lines to include.",
	"max_iterations":         "Maximum number of pipeline for_each iterations.",
	"max_steps":              "Maximum number of pipeline steps.",
	"mcp_json":               "MCP configuration JSON.",
	"memory_limit":           "Memory limit for the agent runtime.",
	"mode":                   "Operation mode.",
	"model_options":          "Provider-specific model options.",
	"mount_path":             "Container mount path.",
	"mounts":                 "Mounted file specifications.",
	"name":                   "Name.",
	"name_pattern":           "Case-insensitive substring to match in tool names.",
	"never_expire":           "Prevent the playground from expiring automatically.",
	"on_error":               "Error handling mode for this pipeline step.",
	"operation":              "Schema operation name.",
	"page":                   "Page number.",
	"parallel":               "Pipeline steps to execute concurrently.",
	"params":                 "Resource-specific parameter object.",
	"patches":                "Patch operations to preview or apply.",
	"per_page":               "Number of results per page.",
	"persist_volumes":        "Persist Docker volumes for the playspec.",
	"port":                   "Network port.",
	"post_init_script":       "Shell script run after agent initialization.",
	"private":                "Whether the repository is private.",
	"provider":               "Provider name.",
	"provider_api_key_mode":  "Use provider API key mode for the agent.",
	"prompt":                 "Initial prompt or instructions for the agent.",
	"public":                 "Whether the template version is public.",
	"q":                      "Search query.",
	"query":                  "Search query.",
	"readonly":               "Mount the file as read-only.",
	"regenerate_variables":   "Template variable names to regenerate.",
	"resource":               "Resource name or alias.",
	"response_mode":          "Response detail mode.",
	"secret":                 "Whether the value is secret, or a webhook signing secret.",
	"service":                "Compose service name.",
	"service_subdomains":     "Per-service subdomain overrides.",
	"services":               "Per-service configuration.",
	"sort":                   "Sort expression.",
	"source_auto_refresh":    "Refresh from source automatically.",
	"source_auto_upgrade":    "Upgrade linked playspecs automatically.",
	"source_base64":          "Base64-encoded source content.",
	"source_path":            "Source file path.",
	"source_ref":             "Source branch, tag, or ref.",
	"source_type":            "Source object type.",
	"ssh_private_key":        "SSH private key.",
	"ssl_mode":               "SSL mode.",
	"status":                 "Resource status filter or target playground status.",
	"sync_enabled":           "Enable repository sync for the agent.",
	"sync_skills_enabled":    "Enable skill sync for the agent.",
	"syscheck_enabled":       "Enable system checks for the agent.",
	"target_services":        "Compose service names affected by this file or operation.",
	"template_body":          "Import template YAML body.",
	"template_body_path":     "Absolute local path to a template YAML file.",
	"text":                   "Message text.",
	"timeout":                "Timeout duration.",
	"timeout_ms":             "Timeout in milliseconds.",
	"tool":                   "Registered Fibe tool name.",
	"tool_filters":           "Webhook tool filter object.",
	"type":                   "Type value.",
	"url":                    "HTTP URL.",
	"user":                   "SSH username.",
	"value":                  "Secret or job environment value to store.",
	"variables":              "Template variables used while rendering a template.",
	"wait":                   "Wait for affected playground rollout status.",
	"wait_timeout":           "Maximum wait duration.",
}

func humanizeFieldName(name string) string {
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
		case "github":
			words[i] = "GitHub"
		case "json":
			words[i] = "JSON"
		case "mcp":
			words[i] = "MCP"
		case "ssh":
			words[i] = "SSH"
		case "ssl":
			words[i] = "SSL"
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
