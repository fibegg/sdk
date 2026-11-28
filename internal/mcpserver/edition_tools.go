package mcpserver

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

const communityNotice = "This server runs the Community edition."

// Use the same session/header precedence as resolveClient. Listing tools must
// also work before a client supplies its API key, and must not pin credentials.
func (s *Server) sessionManifest(ctx context.Context) *routeManifest {
	if s.editions == nil {
		return nil
	}
	st := s.sessionFor(ctx)
	st.mu.RLock()
	domain, key := st.domain, st.apiKey
	pinned := st.client != nil
	if pinned {
		baseURL := st.client.BaseURL()
		st.mu.RUnlock()
		return s.manifestFor(ctx, baseURL)
	}
	st.mu.RUnlock()
	if domain == "" && (!s.httpMode || key != "" || apiKeyFromContext(ctx) != "") {
		domain = domainFromContext(ctx)
	}
	if domain == "" || (s.httpMode && !s.domainAllowed(domain)) {
		domain = s.cfg.Domain
	}
	if domain == s.cfg.Domain {
		return s.manifestFor(ctx, s.baseCli.BaseURL())
	}
	return s.manifestFor(ctx, fibe.NewClient(fibe.WithDisableAutoConfig(), fibe.WithDomain(domain)).BaseURL())
}

func (s *Server) refreshEdition(ctx context.Context, previous *routeManifest) {
	if current := s.sessionManifest(ctx); current != previous {
		_ = s.mcp.SendNotificationToClient(ctx, mcp.MethodNotificationToolsListChanged, nil)
	}
}

func resourceOperationRoutes(resource, operation string) operationRoutes {
	key := resource + "." + operation
	if routes := mutationRoutes[key]; routes != "" {
		return routes
	}
	if routes := flatResourceRoutes[key]; routes != "" {
		return routes
	}
	return schemaOperationRoutes[key]
}

var schemaOperationRoutes = map[string]operationRoutes{
	"compose.validate":         "SpecService.ValidateComposeWithParams",
	"env_pack.attachments_get": envPackGetRoutes,
	"memory.memorize":          "MemoryService.Memorize",
	"mutter.create":            "MutterService.CreateItemByAgentIdentifier",
	"webhook.event_types":      "WebhookEndpointService.EventTypes",
}

func isResourceTool(name string) bool {
	switch name {
	case "fibe_resource_list", "fibe_resource_get", "fibe_resource_delete", "fibe_resource_watch", "fibe_resource_mutate":
		return true
	}
	return false
}

func resourceToolSupports(name, resource, operation string) bool {
	if name == "fibe_schema" {
		return true
	}
	if name == "fibe_resource_mutate" {
		_, _, _, ok := resourceschema.MutationSchemaFor(resource, operation)
		return ok
	}
	return operation == strings.TrimPrefix(name, "fibe_resource_")
}

func (m *routeManifest) resourceOperations(name, resource string) []string {
	var available []string
	for _, op := range resourceschema.SortedOperationNames(resource) {
		if resourceToolSupports(name, resource, op) && m.offers(resourceOperationRoutes(resource, op)) {
			available = append(available, op)
		}
	}
	return available
}

func (m *routeManifest) offersTool(t *toolImpl) bool {
	if m == nil || !isResourceTool(t.name) {
		return m.offers(t.routes)
	}
	for resource := range resourceschema.Registry() {
		if len(m.resourceOperations(t.name, resource)) != 0 {
			return true
		}
	}
	return false
}

func (m *routeManifest) filterToolSchema(name string, schema map[string]any) map[string]any {
	if m == nil || schema == nil || (!isResourceTool(name) && name != "fibe_schema") {
		return schema
	}
	resources := map[string]bool{}
	operations := map[string]bool{}
	for resource := range resourceschema.Registry() {
		for _, op := range m.resourceOperations(name, resource) {
			resources[resource] = true
			operations[op] = true
		}
	}

	out := maps.Clone(schema)
	properties, _ := schema["properties"].(map[string]any)
	properties = maps.Clone(properties)
	out["properties"] = properties
	for _, field := range []string{"resource", "operation"} {
		property, _ := properties[field].(map[string]any)
		values, _ := property["enum"].([]any)
		if values == nil {
			continue
		}
		filtered := make([]any, 0, len(values))
		for _, value := range values {
			selector, _ := value.(string)
			if field == "resource" {
				canonical, _ := resourceschema.CanonicalResource(selector)
				if resources[canonical] || (name == "fibe_schema" && selector == "list") {
					filtered = append(filtered, value)
				}
			} else if operations[selector] {
				filtered = append(filtered, value)
			}
		}
		property = maps.Clone(property)
		property["enum"] = filtered
		properties[field] = property
	}
	return out
}

func (s *Server) editionTools(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	manifest := s.sessionManifest(ctx)
	if manifest == nil {
		return tools
	}
	out := make([]mcp.Tool, 0, len(tools))
	for _, tool := range tools {
		t, ok := s.dispatcher.lookup(tool.Name)
		if !ok || !manifest.offersTool(t) {
			continue
		}
		if isResourceTool(tool.Name) || tool.Name == "fibe_schema" {
			schema := manifest.filterToolSchema(tool.Name, schemaForTool(s, tool.Name))
			raw, err := json.Marshal(schema)
			if err == nil {
				tool.InputSchema = mcp.ToolInputSchema{}
				tool.RawInputSchema = raw
			}
		}
		out = append(out, tool)
	}
	return out
}

func (s *Server) checkToolEdition(manifest *routeManifest, t *toolImpl, args map[string]any) error {
	if manifest == nil {
		return nil
	}
	label := strings.ReplaceAll(strings.TrimPrefix(t.name, "fibe_"), "_", " ")
	if isResourceTool(t.name) {
		op := strings.TrimPrefix(t.name, "fibe_resource_")
		payload := args
		if op == "mutate" {
			op = argString(args, "operation")
			payload, _ = args["payload"].(map[string]any)
		}
		_, resource, operation, ok := resourceschema.SchemaFor(argString(args, "resource"), op)
		if ok && resourceToolSupports(t.name, resource, operation) {
			label = strings.ReplaceAll(resource+" "+operation, "_", " ")
			routes := resourceOperationRoutes(resource, operation)
			return manifest.require(label, selectResourceRoutes(resource, operation, payload, routes))
		}
		return nil
	}
	switch t.name {
	case "fibe_call":
		if target, ok := s.dispatcher.lookup(argString(args, "tool")); ok && target.name != "fibe_call" && target.name != "fibe_pipeline" {
			subArgs, _ := args["args"].(map[string]any)
			return s.checkToolEdition(manifest, target, subArgs)
		}
	case "fibe_run":
		return s.checkRunEdition(manifest, args)
	case "fibe_launch":
		switch source, _ := mcpLaunchSource(args); source {
		case "template":
			return manifest.require("template launch", "ImportTemplateService.LaunchWithParamsByIdentifier")
		case "template_version":
			return manifest.require("template version launch", "GreenfieldService.Create")
		case "spec":
			return manifest.require("spec launch", "PlaygroundService.Create")
		case "compose", "repo":
			return manifest.require("launch", "LaunchService.Create")
		}
	case playgroundSwitchTemplateToolName:
		return manifest.require(label, selectResourceRoutes("playground", "switch_template", args, t.routes))
	case "fibe_templates_change":
		return manifest.require(label, selectResourceRoutes("template", "change", args, t.routes))
	case "fibe_env_pack_attachments_get":
		return manifest.require(label, selectResourceRoutes("env_pack", "attachments_get", args, t.routes))
	case "fibe_logs_follow":
		switch resourceschema.NormalizeResource(argString(args, "target")) {
		case "", "playground", "playgrounds":
			return manifest.require("playground logs follow", followPlaygroundLogs)
		case "task", "tasks":
			return manifest.require("task logs follow", followTaskLogs)
		}
	case "fibe_artefact_upload":
		if argString(args, "agent_id_or_name") != "" || os.Getenv("FIBE_AGENT_ID") != "" {
			return manifest.require(label, "ArtefactService.CreateByAgentIdentifier")
		}
		return manifest.require(label, "ArtefactService.CreateOwned")
	}
	return manifest.require(label, t.routes)
}

func selectResourceRoutes(resource, operation string, args map[string]any, routes operationRoutes) operationRoutes {
	mode := strings.ToLower(strings.TrimSpace(argString(args, "mode")))
	switch resource + "." + operation {
	case "playground.switch_template":
		return operationRoutes(selectTemplateMode(mode, templateSwitchPreview, templateSwitchApply))
	case "template.change":
		switch strings.ToLower(strings.TrimSpace(argString(args, "change_type"))) {
		case "switch_existing":
			return operationRoutes(selectTemplateMode(mode, templateSwitchPreview, templateSwitchApply))
		case "patch", "overwrite":
			return operationRoutes(selectTemplateMode(mode, templatePatchPreview, templatePatchApply))
		}
	case "template.update":
		payload := resourceMutationBackendPayload(resource, operation, args)
		hasMetadata := payload["name"] != nil || payload["description"] != nil || payload["category_id"] != nil
		hasImage := valuePresent(payload["image_data"]) || valuePresent(payload["content_base64"]) || valuePresent(payload["content_path"])
		switch {
		case hasMetadata && hasImage:
			return "ImportTemplateService.UpdateByIdentifier+ImportTemplateService.UploadImageByIdentifier"
		case hasMetadata:
			return "ImportTemplateService.UpdateByIdentifier"
		case hasImage:
			return "ImportTemplateService.UploadImageByIdentifier"
		}
	}
	if resource == "env_pack" && (strings.Contains(string(routes), envPackPlayground)) {
		switch argString(args, "target") {
		case "playgrounds":
			return operationRoutes(strings.Fields(string(routes))[0])
		case "specs":
			return operationRoutes(strings.Fields(string(routes))[1])
		}
	}
	return routes
}

func selectTemplateMode(mode, preview, apply string) string {
	if mode == "preview" {
		return preview
	}
	return apply
}

func (m *routeManifest) schemaRegistry() map[string]map[string]any {
	if m == nil {
		return resourceschema.Registry()
	}
	out := map[string]map[string]any{}
	for resource, schema := range resourceschema.Registry() {
		if ops := m.resourceOperations("fibe_schema", resource); len(ops) > 0 {
			out[resource] = map[string]any{}
			for _, op := range ops {
				out[resource][op] = schema[op]
			}
		}
	}
	return out
}

func (m *routeManifest) schemaCatalog() map[string]any {
	if m == nil {
		return resourceschema.CatalogResponse()
	}
	var out []resourceschema.CatalogEntry
	for _, entry := range resourceschema.Catalog() {
		entry.Operations = m.resourceOperations("fibe_schema", entry.Name)
		if len(entry.Operations) > 0 {
			out = append(out, entry)
		}
	}
	return map[string]any{"resources": out, "edition": communityNotice}
}

func (m *routeManifest) schemaNotice(result any) any {
	if m == nil {
		return result
	}
	return map[string]any{"schema": result, "edition": communityNotice}
}

func (m *routeManifest) schemaForResource(resource, operation string) (any, error) {
	if operation == "" {
		ops := m.schemaRegistry()[resource]
		if len(ops) == 0 {
			return nil, unavailable(strings.ReplaceAll(resource, "_", " "))
		}
		return m.schemaNotice(ops), nil
	}
	schema, _, op, _ := resourceschema.SchemaFor(resource, operation)
	if err := m.require(strings.ReplaceAll(resource+" "+op, "_", " "), resourceOperationRoutes(resource, op)); err != nil {
		return nil, err
	}
	return m.schemaNotice(schema), nil
}
