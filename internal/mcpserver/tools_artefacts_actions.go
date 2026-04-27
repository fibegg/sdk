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
			agentIDStr := os.Getenv("FIBE_AGENT_ID")
			agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
			if err != nil || agentID <= 0 {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing or invalid")
			}

			filename := argString(args, "filename")
			if filename == "" {
				filename = argString(args, "name")
			}
			if filename == "" {
				return nil, fmt.Errorf("filename is required when name is omitted")
			}

			reader, err := decodeFileSource(args)
			if err != nil {
				return nil, err
			}

			var payloadReader io.Reader = reader

			if workspacePath := os.Getenv("FIBE_WORKSPACE_PATH"); workspacePath != "" {
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

			var p fibe.ArtefactCreateParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}

			return c.Artefacts.Create(ctx, agentID, &p, payloadReader, filename)
		},
	}, mcp.NewTool("fibe_artefact_upload",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Upload and save an artefact. Useful when Player asks to create something, implicitly or explicitly"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Artefact display name (alias: 'title'). Also used as filename fallback.")),
		mcp.WithString("filename", mcp.Description("Target filename — defaults to 'name' when omitted")),
		mcp.WithString("description", mcp.Description("Optional human-readable description")),
		mcp.WithString("content_base64", mcp.Description("Base64-encoded file content (alias: 'content')")),
		mcp.WithString("content_path", mcp.Description("Absolute local file path to read (local MCP only)")),
	))
}
