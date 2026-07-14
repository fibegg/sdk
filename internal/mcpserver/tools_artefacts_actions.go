package mcpserver

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

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

			payloadReader := reader
			var mirror *workspaceMirror
			if workspacePath := os.Getenv("FIBE_WORKSPACE_PATH"); workspacePath != "" && hasFile {
				payloadReader, mirror, err = newWorkspaceMirror(workspacePath, filename, reader)
				if err != nil {
					return nil, err
				}
				defer mirror.Abort()
			}

			backendArgs := resourceMutationBackendPayload("artefact", "create", args)
			var p fibe.ArtefactCreateParams
			if err := bindIdentifierArgs(backendArgs, &p, "playground_id"); err != nil {
				return nil, err
			}
			if body != "" {
				p.Body = body
			}

			var result *fibe.Artefact
			if agentIdentifier := argString(args, "agent_id_or_name"); agentIdentifier != "" {
				result, err = c.Artefacts.CreateByAgentIdentifier(ctx, agentIdentifier, &p, payloadReader, filename)
			} else if envAgentID := os.Getenv("FIBE_AGENT_ID"); envAgentID != "" {
				if _, parseErr := strconv.ParseInt(envAgentID, 10, 64); parseErr == nil {
					result, err = c.Artefacts.CreateByAgentIdentifier(ctx, envAgentID, &p, payloadReader, filename)
				} else {
					result, err = c.Artefacts.CreateOwned(ctx, &p, payloadReader, filename)
				}
			} else {
				result, err = c.Artefacts.CreateOwned(ctx, &p, payloadReader, filename)
			}
			if mirror != nil {
				if commitErr := mirror.Commit(); commitErr != nil {
					if err != nil {
						return nil, fmt.Errorf("upload failed (%v) and workspace commit failed: %w", err, commitErr)
					}
					return nil, fmt.Errorf("commit artefact to workspace: %w", commitErr)
				}
			}
			return result, err
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
