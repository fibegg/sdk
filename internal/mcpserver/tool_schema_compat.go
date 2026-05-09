package mcpserver

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

var forbiddenTopLevelToolSchemaKeywords = []string{
	"oneOf",
	"anyOf",
	"allOf",
	"enum",
	"not",
}

// makeToolInputSchemaClientCompatible keeps the advertised MCP tool schema in
// the conservative object-root shape accepted by OpenAI/Codex-style clients.
// Domain validation still happens in each handler, so removing root-level
// validation combinators only avoids client registration failures.
func makeToolInputSchemaClientCompatible(tool *mcp.Tool) {
	schema := toolInputSchemaToMap(*tool)
	m, ok := schema.(map[string]any)
	if !ok {
		return
	}
	m["type"] = "object"
	if _, ok := m["properties"].(map[string]any); !ok {
		m["properties"] = map[string]any{}
	}
	for _, key := range forbiddenTopLevelToolSchemaKeywords {
		delete(m, key)
	}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	tool.InputSchema.Type = ""
	tool.RawInputSchema = data
}
