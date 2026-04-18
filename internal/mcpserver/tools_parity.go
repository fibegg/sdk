package mcpserver

// This file closes MCP↔SDK parity gaps identified during the API audit.
// Each tool here corresponds to an SDK service method that was not covered
// by tools_generated.go (uniform CRUD) or tools_custom.go (one-offs).
//
// Grouped by resource for readability. Tools that accept binary payloads
// (mounted files, artefact upload, template images) expect base64-encoded
// content under a `content_base64` arg.

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerParityTools() {
	s.registerAgentParity()
	s.registerArtefactParity()
	s.registerHunkParity()
	s.registerImportTemplateParity()
	s.registerInstallationParity()
	s.registerMarqueeParity()
	s.registerMutterParity()
	s.registerPlayspecParity()
	s.registerPropParity()
	s.registerTeamParity()
	s.registerWebhookParity()
	s.registerGitRepoParity()
}

// ---------- Agents ----------

func (s *Server) registerAgentParity() {
	// chat
	s.addTool(&toolImpl{
		name: "fibe_agents_chat", description: "Send a message to an agent", tier: tierCore,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "text", "message", "body")
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.AgentChatParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.Text == "" {
				return nil, fmt.Errorf("required field 'text' not set (also accepts 'message' as alias)")
			}
			return c.Agents.Chat(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_agents_chat",
		mcp.WithDescription("Send a message to an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Message text (alias: 'message')")),
		mcp.WithArray("images", mcp.Description("Optional list of image references"), mcp.WithStringItems()),
		mcp.WithArray("attachment_filenames", mcp.Description("Optional list of attachment filenames"), mcp.WithStringItems()),
	))

	// authenticate
	s.addTool(&toolImpl{
		name: "fibe_agents_authenticate", description: "Authenticate an agent (OAuth code/token exchange or API key)", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var code, token *string
			if v := argString(args, "code"); v != "" {
				code = &v
			}
			if v := argString(args, "token"); v != "" {
				token = &v
			}
			return c.Agents.Authenticate(ctx, id, code, token)
		},
	}, mcp.NewTool("fibe_agents_authenticate",
		mcp.WithDescription("Authenticate an agent (OAuth code/token exchange or API key)"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("code", mcp.Description("OAuth authorization code")),
		mcp.WithString("token", mcp.Description("Raw authentication token")),
	))

	// get/update messages
	s.addTool(&toolImpl{
		name: "fibe_agents_messages_get", description: "Fetch the message history for an agent", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			return c.Agents.GetMessages(ctx, id)
		},
	}, mcp.NewTool("fibe_agents_messages_get",
		mcp.WithDescription("Fetch the message history for an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
	))
	s.addTool(&toolImpl{
		name: "fibe_agents_messages_update", description: "Replace or update the message history of an agent", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			content, ok := args["content"]
			if !ok {
				return nil, fmt.Errorf("required field 'content' not set")
			}
			if err := c.Agents.UpdateMessages(ctx, id, content); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "ok": true}, nil
		},
	}, mcp.NewTool("fibe_agents_messages_update",
		mcp.WithDescription("Replace or update the message history of an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithAny("content", mcp.Description("Arbitrary message content")),
	))

	// get/update activity
	s.addTool(&toolImpl{
		name: "fibe_agents_activity_get", description: "Get granular reasoning and thinking activity of an agent", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			return c.Agents.GetActivity(ctx, id)
		},
	}, mcp.NewTool("fibe_agents_activity_get",
		mcp.WithDescription("Get granular reasoning and thinking activity of an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
	))
	s.addTool(&toolImpl{
		name: "fibe_agents_activity_update", description: "Store your own reasoning and thinking activity", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			content, ok := args["content"]
			if !ok {
				return nil, fmt.Errorf("required field 'content' not set")
			}
			if err := c.Agents.UpdateActivity(ctx, id, content); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "ok": true}, nil
		},
	}, mcp.NewTool("fibe_agents_activity_update",
		mcp.WithDescription("Store your own reasoning and thinking activity"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithAny("content", mcp.Description("Arbitrary activity payload")),
	))

	// get/update raw_providers
	s.addTool(&toolImpl{
		name: "fibe_agents_raw_providers_get", description: "Retrieve the raw AI provider configuration for an agent", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			return c.Agents.GetRawProviders(ctx, id)
		},
	}, mcp.NewTool("fibe_agents_raw_providers_get",
		mcp.WithDescription("Retrieve the raw AI provider configuration for an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
	))
	s.addTool(&toolImpl{
		name: "fibe_agents_raw_providers_update", description: "Replace the raw AI provider configuration for an agent", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			content, ok := args["content"]
			if !ok {
				return nil, fmt.Errorf("required field 'content' not set")
			}
			if err := c.Agents.UpdateRawProviders(ctx, id, content); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "ok": true}, nil
		},
	}, mcp.NewTool("fibe_agents_raw_providers_update",
		mcp.WithDescription("Replace the raw AI provider configuration for an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithAny("content", mcp.Description("Provider payload")),
	))

	// github_token
	s.addTool(&toolImpl{
		name: "fibe_agents_github_token", description: "Get the agent's scoped GitHub access token", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			repo := argString(args, "repo")
			if repo != "" {
				return c.Agents.GetGitHubTokenForRepo(ctx, id, repo)
			}
			return c.Agents.GetGitHubToken(ctx, id)
		},
	}, mcp.NewTool("fibe_agents_github_token",
		mcp.WithDescription("Get the agent's scoped GitHub access token"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("repo", mcp.Description("Optional owner/repo to scope the token")),
	))

	// gitea_token
	s.addTool(&toolImpl{
		name: "fibe_agents_gitea_token", description: "Get the agent's scoped Gitea access token", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			return c.Agents.GetGiteaToken(ctx, id)
		},
	}, mcp.NewTool("fibe_agents_gitea_token",
		mcp.WithDescription("Get the agent's scoped Gitea access token"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
	))

	// mounted_file add/update/remove
	s.addTool(&toolImpl{
		name: "fibe_agents_mounted_file_add", description: "Attach a local file mount to an agent", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "mount_path", "path")
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			filename := argString(args, "filename")
			if filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			reader, err := decodeFileSource(args)
			if err != nil {
				return nil, err
			}
			var p fibe.MountedFileParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.Agents.AddMountedFile(ctx, id, reader, filename, &p)
		},
	}, mcp.NewTool("fibe_agents_mounted_file_add",
		mcp.WithDescription("Attach a local file mount to an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Target filename")),
		mcp.WithString("content_base64", mcp.Description("Base64-encoded file content (use one of content_base64/content_path)")),
		mcp.WithString("content_path", mcp.Description("Absolute local file path to read (local MCP only)")),
		mcp.WithString("mount_path", mcp.Description("Path inside the target container (alias: 'path')")),
		mcp.WithArray("target_services", mcp.Description("Services to mount into"), mcp.WithStringItems()),
		mcp.WithBoolean("readonly", mcp.Description("Mount as read-only")),
	))
	s.addTool(&toolImpl{
		name: "fibe_agents_mounted_file_update", description: "Update the metadata of an agent's attached file mount", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "mount_path", "path")
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.MountedFileUpdateParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.Filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			return c.Agents.UpdateMountedFile(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_agents_mounted_file_update",
		mcp.WithDescription("Update the metadata of an agent's attached file mount"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Filename of the existing mounted file")),
		mcp.WithString("mount_path", mcp.Description("Path inside the target container")),
		mcp.WithArray("target_services", mcp.Description("Services to mount into"), mcp.WithStringItems()),
		mcp.WithBoolean("readonly", mcp.Description("Mount as read-only")),
	))
	s.addTool(&toolImpl{
		name: "fibe_agents_mounted_file_remove", description: "Remove an attached file mount from an agent", tier: tierFull,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			filename := argString(args, "filename")
			if filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			return c.Agents.RemoveMountedFile(ctx, id, filename)
		},
	}, mcp.NewTool("fibe_agents_mounted_file_remove",
		mcp.WithDescription("Remove an attached file mount from an agent"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Filename to remove")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))
}

// ---------- Artefacts ----------

func (s *Server) registerArtefactParity() {
	s.addTool(&toolImpl{
		name: "fibe_artefacts_create", description: "Upload and save an artefact for an agent", tier: tierCore,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "name", "title")
			aliasField(args, "content_base64", "content")
			agentID, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			filename := argString(args, "filename")
			if filename == "" {
				// Fall back to name when filename isn't provided — agents
				// often pass just `name` assuming it doubles as the filename.
				if n := argString(args, "name"); n != "" {
					filename = n
				}
			}
			if filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set (if omitted, 'name' is used as fallback)")
			}
			reader, err := decodeFileSource(args)
			if err != nil {
				return nil, err
			}
			var p fibe.ArtefactCreateParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.Artefacts.Create(ctx, agentID, &p, reader, filename)
		},
	}, mcp.NewTool("fibe_artefacts_create",
		mcp.WithDescription("Upload and save an artefact for an agent"),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("name", mcp.Description("Artefact display name (alias: 'title'). Also used as filename fallback.")),
		mcp.WithString("filename", mcp.Description("Target filename — defaults to 'name' when omitted")),
		mcp.WithString("content_base64", mcp.Description("Base64-encoded file content (alias: 'content')")),
		mcp.WithString("content_path", mcp.Description("Absolute local file path to read (local MCP only)")),
		mcp.WithString("description", mcp.Description("Optional human-readable description")),
	))

	s.addTool(&toolImpl{
		name: "fibe_artefacts_download", description: "Download an artefact's contents", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentID, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			body, filename, contentType, err := c.Artefacts.Download(ctx, agentID, id)
			if err != nil {
				return nil, err
			}
			defer body.Close()
			data, err := io.ReadAll(body)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"agent_id":       agentID,
				"id":             id,
				"filename":       filename,
				"content_type":   contentType,
				"content_base64": base64.StdEncoding.EncodeToString(data),
				"size":           len(data),
			}, nil
		},
	}, mcp.NewTool("fibe_artefacts_download",
		mcp.WithDescription("Download an artefact's contents"),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Artefact ID")),
	))
}

// ---------- Hunks ----------

func (s *Server) registerHunkParity() {
// 	s.addTool(&toolImpl{
// 		name: "fibe_hunks_ingest", description: "Trigger hunk ingestion for a prop", tier: tierFull,
// 		annotations: toolAnnotations{Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			propID, ok := argInt64(args, "prop_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'prop_id' not set")
// 			}
// 			force := argBool(args, "force")
// 			if err := c.Hunks.Ingest(ctx, propID, force); err != nil {
// 				return nil, err
// 			}
// 			return map[string]any{"prop_id": propID, "ingested": true, "force": force}, nil
// 		},
// 	}, mcp.NewTool("fibe_hunks_ingest",
// 		mcp.WithDescription("Kick off hunk ingestion for a prop. Pass force:true to re-ingest already-processed hunks."),
// 		mcp.WithNumber("prop_id", mcp.Required(), mcp.Description("Prop ID")),
// 		mcp.WithBoolean("force", mcp.Description("Force re-ingestion")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_hunks_next", description: "Fetch the next hunk awaiting processing (requires processor_name)", tier: tierFull,
// 		annotations: toolAnnotations{},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			aliasField(args, "processor_name", "processor", "worker")
// 			propID, ok := argInt64(args, "prop_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'prop_id' not set")
// 			}
// 			proc := argString(args, "processor_name")
// 			if proc == "" {
// 				return nil, fmt.Errorf("required field 'processor_name' not set — the worker claiming the hunk must identify itself (alias: 'processor')")
// 			}
// 			return c.Hunks.Next(ctx, propID, proc)
// 		},
// 	}, mcp.NewTool("fibe_hunks_next",
// 		mcp.WithDescription("Claim the next hunk awaiting processing by the named processor. 'processor_name' is REQUIRED — it's the identifier the backend uses to track which worker owns the hunk. Alias 'processor' is accepted."),
// 		mcp.WithNumber("prop_id", mcp.Required(), mcp.Description("Prop ID")),
// 		mcp.WithString("processor_name", mcp.Required(), mcp.Description("Processor identifier (required — alias: 'processor')")),
// 	))
}

// ---------- Import Templates ----------

func (s *Server) registerImportTemplateParity() {
	s.addTool(&toolImpl{
		name: "fibe_templates_search", description: "Search through the catalog of available import templates", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			q := argString(args, "query")
			var tmplID *int64
			if v, ok := argInt64(args, "template_id"); ok {
				tmplID = &v
			}
			return c.ImportTemplates.Search(ctx, q, tmplID)
		},
	}, mcp.NewTool("fibe_templates_search",
		mcp.WithDescription("Search through the catalog of available import templates"),
		mcp.WithString("query", mcp.Description("Search query")),
		mcp.WithNumber("template_id", mcp.Description("Optional template ID filter")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_versions_list", description: "List all available versions of an import template", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.ListParams
			_ = bindArgs(args, &p)
			return c.ImportTemplates.ListVersions(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_templates_versions_list",
		mcp.WithDescription("List all available versions of an import template"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Template ID")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Page size")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_versions_create", description: "Create a new version iteration for an import template", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "template_body", "body")
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.ImportTemplateVersionCreateParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.TemplateBody == "" {
				return nil, fmt.Errorf("required field 'template_body' not set (also accepts 'body' as alias)")
			}
			return c.ImportTemplates.CreateVersion(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_templates_versions_create",
		mcp.WithDescription("Create a new version iteration for an import template"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Template ID")),
		mcp.WithString("template_body", mcp.Required(), mcp.Description("Template YAML body (alias: 'body')")),
		mcp.WithBoolean("public", mcp.Description("Make this version public")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_versions_toggle_public", description: "Toggle the public visibility state of a specific template version", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "template_id", "id")
			tid, ok := argInt64(args, "template_id")
			if !ok {
				return nil, fmt.Errorf("required field 'template_id' not set (also accepts 'id' for consistency with other tools)")
			}
			vid, ok := argInt64(args, "version_id")
			if !ok {
				return nil, fmt.Errorf("required field 'version_id' not set")
			}
			return c.ImportTemplates.TogglePublic(ctx, tid, vid)
		},
	}, mcp.NewTool("fibe_templates_versions_toggle_public",
		mcp.WithDescription("Toggle the public visibility state of a specific template version"),
		mcp.WithNumber("template_id", mcp.Required(), mcp.Description("Template ID (alias: 'id')")),
		mcp.WithNumber("version_id", mcp.Required(), mcp.Description("Version ID")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_versions_destroy", description: "Delete a specific version of an import template", tier: tierFull,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "template_id", "id")
			tid, ok := argInt64(args, "template_id")
			if !ok {
				return nil, fmt.Errorf("required field 'template_id' not set")
			}
			vid, ok := argInt64(args, "version_id")
			if !ok {
				return nil, fmt.Errorf("required field 'version_id' not set")
			}
			if err := c.ImportTemplates.DestroyVersion(ctx, tid, vid); err != nil {
				return nil, err
			}
			return map[string]any{"template_id": tid, "version_id": vid, "deleted": true}, nil
		},
	}, mcp.NewTool("fibe_templates_versions_destroy",
		mcp.WithDescription("Delete a specific version of an import template"),
		mcp.WithNumber("template_id", mcp.Required(), mcp.Description("Template ID (alias: 'id')")),
		mcp.WithNumber("version_id", mcp.Required(), mcp.Description("Version ID")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_launch", description: "Bootstrap and launch a new playground directly from an import template", tier: tierCore,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.ImportTemplateLaunchParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.ImportTemplates.LaunchWithParams(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_templates_launch",
		mcp.WithDescription("Bootstrap and launch a new playground directly from an import template"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Template ID")),
		mcp.WithNumber("marquee_id", mcp.Required(), mcp.Description("Target marquee ID")),
		mcp.WithString("name", mcp.Description("Optional playground name override")),
		mcp.WithNumber("version", mcp.Description("Optional template version to launch")),
		mcp.WithObject("variables", mcp.Description("Dictionary mapping dynamically evaluated Fibe template parameters smoothly natively")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_upload_image", description: "Upload and attach a cover media image to an import template", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.UploadImageParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.ImageData == "" {
				// Allow content_base64/content_path convenience.
				if v := argString(args, "content_base64"); v != "" {
					p.ImageData = v
				} else if path := argString(args, "content_path"); path != "" {
					data, err := readLocalFileBase64(path)
					if err != nil {
						return nil, err
					}
					p.ImageData = data
				}
			}
			if p.ImageData == "" {
				return nil, fmt.Errorf("required field 'image_data' not set (or pass content_base64/content_path)")
			}
			if p.Filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			return c.ImportTemplates.UploadImage(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_templates_upload_image",
		mcp.WithDescription("Upload and attach a cover media image to an import template"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Template ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Image filename")),
		mcp.WithString("image_data", mcp.Description("Base64-encoded image data")),
		mcp.WithString("content_base64", mcp.Description("Alias for image_data")),
		mcp.WithString("content_path", mcp.Description("Absolute local path to read (local MCP only)")),
		mcp.WithString("content_type", mcp.Description("MIME type (default: image/png)")),
	))
}

// ---------- Installations ----------

func (s *Server) registerInstallationParity() {
	s.addTool(&toolImpl{
		name: "fibe_installations_repos", description: "List the GitHub repositories accessible to a specific installation", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.InstallationReposParams
			_ = bindArgs(args, &p)
			return c.Installations.Repos(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_installations_repos",
		mcp.WithDescription("List the GitHub repositories accessible to a specific installation"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Installation ID")),
		mcp.WithInputSchema[fibe.InstallationReposParams](),
	))
}

// ---------- Marquees ----------

func (s *Server) registerMarqueeParity() {
	s.addTool(&toolImpl{
		name: "fibe_marquees_autoconnect_token", description: "Generate an autoconnect token for seamless Marquee integration", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			var p fibe.AutoconnectTokenParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.Marquees.AutoconnectToken(ctx, &p)
		},
	}, mcp.NewTool("fibe_marquees_autoconnect_token",
		mcp.WithDescription("Generate an autoconnect token for seamless Marquee integration"),
		mcp.WithInputSchema[fibe.AutoconnectTokenParams](),
	))
}

// ---------- Mutters ----------

func (s *Server) registerMutterParity() {
	s.addTool(&toolImpl{
		name: "fibe_mutters_get", description: "Retrieve the full internal muttering transcript of an agent", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentID, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			var p fibe.MutterListParams
			_ = bindArgs(args, &p)
			return c.Mutters.Get(ctx, agentID, &p)
		},
	}, mcp.NewTool("fibe_mutters_get",
		mcp.WithDescription("Retrieve the full internal muttering transcript of an agent"),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithInputSchema[fibe.MutterListParams](),
	))

	s.addTool(&toolImpl{
		name: "fibe_mutters_create", description: "Append a new entry to an agent's internal muttering transcript", tier: tierCore,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentID, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			var p fibe.MutterItemParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.Mutters.CreateItem(ctx, agentID, &p)
		},
	}, mcp.NewTool("fibe_mutters_create",
		mcp.WithDescription("Append a new entry to an agent's internal muttering transcript"),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithInputSchema[fibe.MutterItemParams](),
	))
}

// ---------- Playspecs ----------

func (s *Server) registerPlayspecParity() {
	s.addTool(&toolImpl{
		name: "fibe_playspecs_services", description: "List all individual services defined within a playspec", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			svcs, err := c.Playspecs.Services(ctx, id)
			if err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "services": svcs}, nil
		},
	}, mcp.NewTool("fibe_playspecs_services",
		mcp.WithDescription("List all individual services defined within a playspec"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_switch_version_preview", description: "Preview switching a template-backed playspec to another template version", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.PlayspecTemplateVersionSwitchParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.TargetTemplateVersionID == 0 {
				return nil, fmt.Errorf("required field 'target_template_version_id' not set")
			}
			return c.Playspecs.PreviewTemplateVersionSwitch(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_playspecs_switch_version_preview",
		mcp.WithDescription("Preview switching a template-backed playspec to another template version"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithNumber("target_template_version_id", mcp.Required(), mcp.Description("Target template version ID")),
		mcp.WithAny("variables", mcp.Description("Template variable overrides as an object")),
		mcp.WithArray("regenerate_variables", mcp.Description("Random variable names to regenerate"), mcp.WithStringItems()),
		mcp.WithBoolean("confirm_warnings", mcp.Description("Preview flag mirrored from API; apply still requires confirmation for warnings")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_switch_version", description: "Switch a template-backed playspec to another template version", tier: tierCore,
		annotations: toolAnnotations{Destructive: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.PlayspecTemplateVersionSwitchParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.TargetTemplateVersionID == 0 {
				return nil, fmt.Errorf("required field 'target_template_version_id' not set")
			}
			return c.Playspecs.SwitchTemplateVersion(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_playspecs_switch_version",
		mcp.WithDescription("Switch a template-backed playspec to another template version"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithNumber("target_template_version_id", mcp.Required(), mcp.Description("Target template version ID")),
		mcp.WithAny("variables", mcp.Description("Template variable overrides as an object")),
		mcp.WithArray("regenerate_variables", mcp.Description("Random variable names to regenerate"), mcp.WithStringItems()),
		mcp.WithBoolean("confirm_warnings", mcp.Description("Required when preview reports risky changes")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_mounted_file_add", description: "Attach a local file mount to a playspec blueprint", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "mount_path", "path")
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			filename := argString(args, "filename")
			if filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			reader, err := decodeFileSource(args)
			if err != nil {
				return nil, err
			}
			var p fibe.MountedFileParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if err := c.Playspecs.AddMountedFile(ctx, id, reader, filename, &p); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "filename": filename, "ok": true}, nil
		},
	}, mcp.NewTool("fibe_playspecs_mounted_file_add",
		mcp.WithDescription("Attach a local file mount to a playspec blueprint"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Target filename")),
		mcp.WithString("content_base64", mcp.Description("Base64-encoded file content")),
		mcp.WithString("content_path", mcp.Description("Absolute local file path (local MCP only)")),
		mcp.WithString("mount_path", mcp.Description("Path inside the target container (alias: 'path')")),
		mcp.WithArray("target_services", mcp.Description("Services to mount into"), mcp.WithStringItems()),
		mcp.WithBoolean("readonly", mcp.Description("Mount as read-only")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_mounted_file_update", description: "Update the metadata of an attached file mount on a playspec", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "mount_path", "path")
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.MountedFileUpdateParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.Filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			if err := c.Playspecs.UpdateMountedFile(ctx, id, &p); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "filename": p.Filename, "ok": true}, nil
		},
	}, mcp.NewTool("fibe_playspecs_mounted_file_update",
		mcp.WithDescription("Update the metadata of an attached file mount on a playspec"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Filename of the existing mounted file")),
		mcp.WithString("mount_path", mcp.Description("Path inside the target container")),
		mcp.WithArray("target_services", mcp.Description("Services to mount into"), mcp.WithStringItems()),
		mcp.WithBoolean("readonly", mcp.Description("Mount as read-only")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_mounted_file_remove", description: "Remove an attached file mount from a playspec", tier: tierFull,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			filename := argString(args, "filename")
			if filename == "" {
				return nil, fmt.Errorf("required field 'filename' not set")
			}
			if err := c.Playspecs.RemoveMountedFile(ctx, id, filename); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "filename": filename, "removed": true}, nil
		},
	}, mcp.NewTool("fibe_playspecs_mounted_file_remove",
		mcp.WithDescription("Remove an attached file mount from a playspec"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Filename to remove")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_registry_credential_add", description: "Attach container registry authentication credentials to a playspec", tier: tierFull,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.RegistryCredentialParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			// Validate registry_type locally against the backend's known
			// enum so agents get a precise error instead of the generic
			// "Invalid credential: missing required fields" that the Rails
			// side raises when it rejects the type without naming it.
			validTypes := map[string]bool{"ghcr": true, "dockerhub": true, "aws_ecr": true}
			if p.RegistryType == "" {
				return nil, fmt.Errorf("required field 'registry_type' not set — must be one of: ghcr, dockerhub, aws_ecr")
			}
			if !validTypes[p.RegistryType] {
				return nil, fmt.Errorf("invalid registry_type %q — must be one of: ghcr, dockerhub, aws_ecr", p.RegistryType)
			}
			if p.RegistryURL == "" {
				return nil, fmt.Errorf("required field 'registry_url' not set")
			}
			if p.Username == "" {
				return nil, fmt.Errorf("required field 'username' not set")
			}
			if p.Secret == "" {
				return nil, fmt.Errorf("required field 'secret' not set")
			}
			result, err := c.Playspecs.AddRegistryCredential(ctx, id, &p)
			if err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "registry_type": p.RegistryType, "added": true, "credentials": result.Credentials}, nil
		},
	}, mcp.NewTool("fibe_playspecs_registry_credential_add",
		mcp.WithDescription("Attach container registry authentication credentials to a playspec"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithString("registry_type", mcp.Required(),
			mcp.Description("Registry type — MUST be one of: ghcr, dockerhub, aws_ecr"),
			mcp.Enum("ghcr", "dockerhub", "aws_ecr")),
		mcp.WithString("registry_url", mcp.Required(), mcp.Description("Registry URL")),
		mcp.WithString("username", mcp.Required(), mcp.Description("Registry username")),
		mcp.WithString("secret", mcp.Required(), mcp.Description("Registry password/token")),
	))

	s.addTool(&toolImpl{
		name: "fibe_playspecs_registry_credential_remove", description: "Remove registry authentication credentials from a playspec", tier: tierFull,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			credID := argString(args, "credential_id")
			if credID == "" {
				return nil, fmt.Errorf("required field 'credential_id' not set")
			}
			if err := c.Playspecs.RemoveRegistryCredential(ctx, id, credID); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "credential_id": credID, "removed": true}, nil
		},
	}, mcp.NewTool("fibe_playspecs_registry_credential_remove",
		mcp.WithDescription("Remove registry authentication credentials from a playspec"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playspec ID")),
		mcp.WithString("credential_id", mcp.Required(), mcp.Description("Credential ID")),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	))
}

// ---------- Props ----------

func (s *Server) registerPropParity() {
	s.addTool(&toolImpl{
		name: "fibe_props_with_docker_compose", description: "List all props that contain a valid Docker Compose configuration", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			var p fibe.PropListParams
			_ = bindArgs(args, &p)
			return c.Props.WithDockerCompose(ctx, &p)
		},
	}, mcp.NewTool("fibe_props_with_docker_compose",
		mcp.WithDescription("List all props that contain a valid Docker Compose configuration"),
		mcp.WithInputSchema[fibe.PropListParams](),
	))

	s.addTool(&toolImpl{
		name: "fibe_props_mirror", description: "Duplicate an external repository to create a new mirrored prop", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "source_url", "repository_url", "url")
			src := argString(args, "source_url")
			if src == "" {
				return nil, fmt.Errorf("required field 'source_url' not set (also accepts 'repository_url' as alias)")
			}
			return c.Props.Mirror(ctx, src)
		},
	}, mcp.NewTool("fibe_props_mirror",
		mcp.WithDescription("Duplicate an external repository to create a new mirrored prop"),
		mcp.WithString("source_url", mcp.Required(), mcp.Description("Source repository URL (alias: 'repository_url')")),
	))

	s.addTool(&toolImpl{
		name: "fibe_props_manual_link", description: "Manually re-link a prop to its source following an OAuth reconnection", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			return c.Props.ManualLink(ctx, id)
		},
	}, mcp.NewTool("fibe_props_manual_link",
		mcp.WithDescription("Manually re-link a prop to its source following an OAuth reconnection"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Prop ID")),
	))

	s.addTool(&toolImpl{
		name: "fibe_props_env_defaults", description: "Extract and read default environment variables from a prop's branch", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			branch := argString(args, "branch")
			envFile := argString(args, "env_file_path")
			return c.Props.EnvDefaults(ctx, id, branch, envFile)
		},
	}, mcp.NewTool("fibe_props_env_defaults",
		mcp.WithDescription("Extract and read default environment variables from a prop's branch"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Prop ID")),
		mcp.WithString("branch", mcp.Description("Branch name (default: default branch)")),
		mcp.WithString("env_file_path", mcp.Description("Path within the repo (default: .env)")),
	))
}

// ---------- Teams ----------

func (s *Server) registerTeamParity() {
// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_transfer_leadership", description: "Transfer team leadership to another member (accepts 'id' or 'team_id')", tier: tierFull,
// 		annotations: toolAnnotations{Destructive: true, Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			aliasField(args, "id", "team_id")
// 			id, ok := argInt64(args, "id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'id' not set (also accepts 'team_id' as alias)")
// 			}
// 			newLeader, ok := argInt64(args, "new_leader_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'new_leader_id' not set")
// 			}
// 			return c.Teams.TransferLeadership(ctx, id, newLeader)
// 		},
// 	}, mcp.NewTool("fibe_teams_transfer_leadership",
// 		mcp.WithDescription("Transfer team leadership to another member. Destructive: you lose admin rights. Canonical team field is 'id'; 'team_id' is accepted as an alias for consistency with fibe_teams_members_* tools."),
// 		mcp.WithNumber("id", mcp.Required(), mcp.Description("Team ID (alias: 'team_id')")),
// 		mcp.WithNumber("new_leader_id", mcp.Required(), mcp.Description("Player ID of the new leader")),
// 		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_members_invite", description: "Invite a user to a team", tier: tierFull,
// 		annotations: toolAnnotations{},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			username := argString(args, "username")
// 			if username == "" {
// 				return nil, fmt.Errorf("required field 'username' not set")
// 			}
// 			return c.Teams.InviteMember(ctx, teamID, username)
// 		},
// 	}, mcp.NewTool("fibe_teams_members_invite",
// 		mcp.WithDescription("Invite a user to a team by username."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithString("username", mcp.Required(), mcp.Description("Username (platform handle)")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_members_accept", description: "Accept a pending team invite", tier: tierFull,
// 		annotations: toolAnnotations{},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			memID, ok := argInt64(args, "membership_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'membership_id' not set")
// 			}
// 			return c.Teams.AcceptInvite(ctx, teamID, memID)
// 		},
// 	}, mcp.NewTool("fibe_teams_members_accept",
// 		mcp.WithDescription("Accept a pending team membership invite."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithNumber("membership_id", mcp.Required(), mcp.Description("Membership ID")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_members_decline", description: "Decline a pending team invite", tier: tierFull,
// 		annotations: toolAnnotations{},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			memID, ok := argInt64(args, "membership_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'membership_id' not set")
// 			}
// 			return c.Teams.DeclineInvite(ctx, teamID, memID)
// 		},
// 	}, mcp.NewTool("fibe_teams_members_decline",
// 		mcp.WithDescription("Decline a pending team membership invite."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithNumber("membership_id", mcp.Required(), mcp.Description("Membership ID")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_members_update", description: "Update a team member's role", tier: tierFull,
// 		annotations: toolAnnotations{Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			memID, ok := argInt64(args, "membership_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'membership_id' not set")
// 			}
// 			role := argString(args, "role")
// 			if role == "" {
// 				return nil, fmt.Errorf("required field 'role' not set")
// 			}
// 			return c.Teams.UpdateMember(ctx, teamID, memID, role)
// 		},
// 	}, mcp.NewTool("fibe_teams_members_update",
// 		mcp.WithDescription("Update a team member's role."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithNumber("membership_id", mcp.Required(), mcp.Description("Membership ID")),
// 		mcp.WithString("role", mcp.Required(), mcp.Description("New role (admin, member, ...)")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_members_remove", description: "Remove a member from a team", tier: tierFull,
// 		annotations: toolAnnotations{Destructive: true, Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			memID, ok := argInt64(args, "membership_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'membership_id' not set")
// 			}
// 			if err := c.Teams.RemoveMember(ctx, teamID, memID); err != nil {
// 				return nil, err
// 			}
// 			return map[string]any{"team_id": teamID, "membership_id": memID, "removed": true}, nil
// 		},
// 	}, mcp.NewTool("fibe_teams_members_remove",
// 		mcp.WithDescription("Remove a member from a team."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithNumber("membership_id", mcp.Required(), mcp.Description("Membership ID")),
// 		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_resources_list", description: "List resources owned by a team", tier: tierFull,
// 		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			var p fibe.ListParams
// 			_ = bindArgs(args, &p)
// 			return c.Teams.ListResources(ctx, teamID, &p)
// 		},
// 	}, mcp.NewTool("fibe_teams_resources_list",
// 		mcp.WithDescription("List shared resources owned by a team."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithNumber("page", mcp.Description("Page number")),
// 		mcp.WithNumber("per_page", mcp.Description("Page size")),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_resources_contribute", description: "Contribute a resource to a team", tier: tierFull,
// 		annotations: toolAnnotations{Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			var p fibe.TeamResourceParams
// 			if err := bindArgs(args, &p); err != nil {
// 				return nil, err
// 			}
// 			return c.Teams.ContributeResource(ctx, teamID, &p)
// 		},
// 	}, mcp.NewTool("fibe_teams_resources_contribute",
// 		mcp.WithDescription("Share a resource with a team."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithInputSchema[fibe.TeamResourceParams](),
// 	))

// 	s.addTool(&toolImpl{
// 		name: "fibe_teams_resources_remove", description: "Remove a shared team resource", tier: tierFull,
// 		annotations: toolAnnotations{Destructive: true, Idempotent: true},
// 		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
// 			teamID, ok := argInt64(args, "team_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'team_id' not set")
// 			}
// 			resID, ok := argInt64(args, "resource_id")
// 			if !ok {
// 				return nil, fmt.Errorf("required field 'resource_id' not set")
// 			}
// 			if err := c.Teams.RemoveResource(ctx, teamID, resID); err != nil {
// 				return nil, err
// 			}
// 			return map[string]any{"team_id": teamID, "resource_id": resID, "removed": true}, nil
// 		},
// 	}, mcp.NewTool("fibe_teams_resources_remove",
// 		mcp.WithDescription("Remove a shared resource from a team."),
// 		mcp.WithNumber("team_id", mcp.Required(), mcp.Description("Team ID")),
// 		mcp.WithNumber("resource_id", mcp.Required(), mcp.Description("Resource ID")),
// 		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
// 	))
}

// ---------- Webhooks ----------

func (s *Server) registerWebhookParity() {
	s.addTool(&toolImpl{
		name: "fibe_webhooks_event_types", description: "List all event types supported by Fibe webhooks", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			types, err := c.WebhookEndpoints.EventTypes(ctx)
			if err != nil {
				return nil, err
			}
			return map[string]any{"event_types": types}, nil
		},
	}, mcp.NewTool("fibe_webhooks_event_types",
		mcp.WithDescription("List all event types supported by Fibe webhooks"),
	))

	s.addTool(&toolImpl{
		name: "fibe_webhooks_deliveries_list", description: "List recent event delivery attempts for a webhook endpoint", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var p fibe.ListParams
			_ = bindArgs(args, &p)
			return c.WebhookEndpoints.ListDeliveries(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_webhooks_deliveries_list",
		mcp.WithDescription("List recent event delivery attempts for a webhook endpoint"),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Webhook endpoint ID")),
		mcp.WithNumber("page", mcp.Description("Page number")),
		mcp.WithNumber("per_page", mcp.Description("Page size")),
	))
}

// ---------- GitHub / Gitea Repos ----------

func (s *Server) registerGitRepoParity() {
	registerCreate(s, "fibe_github_repos_create", "Register and connect a new GitHub repository", toolOpts{Tier: tierCore},
		func(ctx context.Context, c *fibe.Client, p *fibe.GitHubRepoCreateParams) (*fibe.GitHubRepo, error) {
			return c.GitHubRepos.Create(ctx, p)
		})
	registerCreate(s, "fibe_gitea_repos_create", "Register and connect a new Gitea repository", toolOpts{Tier: tierFull},
		func(ctx context.Context, c *fibe.Client, p *fibe.GiteaRepoCreateParams) (*fibe.GiteaRepo, error) {
			return c.GiteaRepos.Create(ctx, p)
		})
}

// ---------- helpers for file-bearing tools ----------

// decodeFileSource reads either args["content_base64"] or args["content_path"]
// (local filesystem only) and returns an io.Reader suitable for multipart
// upload. One of the two must be provided.
func decodeFileSource(args map[string]any) (io.Reader, error) {
	if b := argString(args, "content_base64"); b != "" {
		data, err := base64.StdEncoding.DecodeString(b)
		if err != nil {
			return nil, fmt.Errorf("invalid content_base64: %w", err)
		}
		return bytes.NewReader(data), nil
	}
	if path := argString(args, "content_path"); path != "" {
		data, err := readLocalFile(path)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}
	return nil, fmt.Errorf("required field missing: pass content_base64 or content_path")
}
