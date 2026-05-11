package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerArtefactActionTools() {
	s.addTool(&toolImpl{
		name:        "fibe_artefact_upload",
		description: "[MODE:SIDEEFFECTS] Upload and save an artefact. Useful when Player asks to create something, implicitly or explicitly",
		tier:        tierBase,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			filename := argString(args, "filename")
			if filename == "" {
				filename = argString(args, "name")
			}

			reader, hasFile, err := artefactUploadReader(args)
			if err != nil {
				return nil, err
			}
			body := argString(args, "body")
			if body == "" {
				body = argString(args, "content_text")
			}
			if !hasFile && body == "" {
				return nil, fmt.Errorf("content_base64, content_path, body, or content_text is required")
			}
			if filename == "" {
				filename = "artefact.md"
			}

			var payloadReader io.Reader = reader

			if workspacePath := os.Getenv("FIBE_WORKSPACE_PATH"); workspacePath != "" && hasFile {
				content, err := io.ReadAll(reader)
				if err != nil {
					return nil, fmt.Errorf("failed to read artefact content for workspace: %w", err)
				}
				cleanFilename := filepath.Clean(filename)
				if strings.HasPrefix(cleanFilename, "..") || filepath.IsAbs(cleanFilename) {
					return nil, fmt.Errorf("invalid filename for workspace: must be relative path without traversal")
				}
				targetPath := filepath.Join(workspacePath, cleanFilename)
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return nil, fmt.Errorf("failed to create directory for artefact: %w", err)
				}
				if err := os.WriteFile(targetPath, content, 0644); err != nil {
					return nil, fmt.Errorf("failed to write artefact to workspace: %w", err)
				}
				payloadReader = bytes.NewReader(content)
			}

			backendArgs := resourceMutationBackendPayload("artefact", "create", args)
			var p fibe.ArtefactCreateParams
			if err := bindIdentifierArgs(backendArgs, &p, "playground_id"); err != nil {
				return nil, err
			}
			if body != "" {
				p.Body = body
			}

			if agentIdentifier := argString(args, "agent_id_or_name"); agentIdentifier != "" {
				return c.Artefacts.CreateByAgentIdentifier(ctx, agentIdentifier, &p, payloadReader, filename)
			}
			if envAgentID := os.Getenv("FIBE_AGENT_ID"); envAgentID != "" {
				if _, err := strconv.ParseInt(envAgentID, 10, 64); err == nil {
					return c.Artefacts.CreateByAgentIdentifier(ctx, envAgentID, &p, payloadReader, filename)
				}
			}
			return c.Artefacts.CreateOwned(ctx, &p, payloadReader, filename)
		},
	}, mcp.NewTool("fibe_artefact_upload",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Upload and save an artefact. Useful when Player asks to create something, implicitly or explicitly"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Artefact display name (alias: 'title'). Also used as filename fallback.")),
		mcp.WithString("agent_id_or_name", mcp.Description("Optional agent id or name; defaults to FIBE_AGENT_ID when available, otherwise creates a player-owned artefact")),
		mcp.WithString("playground_id_or_name", mcp.Description("Optional playground ID or name to associate with the artefact")),
		mcp.WithString("filename", mcp.Description("Target filename — defaults to 'name' when omitted")),
		mcp.WithString("description", mcp.Description("Optional human-readable description")),
		mcp.WithString("content_base64", mcp.Description("Base64-encoded file content (alias: 'content')")),
		mcp.WithString("content_path", mcp.Description("Absolute local file path to read (local MCP only)")),
		mcp.WithString("body", mcp.Description("Inline body for body-only artefacts")),
		mcp.WithString("content_text", mcp.Description("Alias for body")),
		mcp.WithBoolean("skill", mcp.Description("Expose this artefact as a skill")),
		mcp.WithBoolean("skill_enabled", mcp.Description("Enable this artefact skill by default")),
	))
}

func artefactUploadReader(args map[string]any) (io.Reader, bool, error) {
	if argString(args, "content_base64") == "" && argString(args, "content") == "" && argString(args, "content_path") == "" {
		return nil, false, nil
	}
	reader, err := decodeFileSource(args)
	return reader, err == nil, err
}
