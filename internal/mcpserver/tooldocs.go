package mcpserver

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ToolDocs struct {
	CatalogMarkdown string
	TableMarkdown   string
}

func GenerateToolDocs(tools []ToolInfo) ToolDocs {
	sorted := sortedToolInfos(tools)
	return ToolDocs{
		CatalogMarkdown: toolCatalogMarkdown(sorted),
		TableMarkdown:   toolTableMarkdown(sorted),
	}
}

func sortedToolInfos(tools []ToolInfo) []ToolInfo {
	out := append([]ToolInfo(nil), tools...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func toolTableMarkdown(tools []ToolInfo) string {
	var b strings.Builder
	b.WriteString("# Fibe MCP Tools Table\n\n")
	b.WriteString("| Tool Name | Tier | Description |\n")
	b.WriteString("|-----------|------|-------------|\n")
	for _, tool := range tools {
		fmt.Fprintf(
			&b,
			"| `%s` | %s | %s |\n",
			tool.Name,
			markdownTableCell(tool.Tier),
			markdownTableCell(tool.Description),
		)
	}
	return b.String()
}

func toolCatalogMarkdown(tools []ToolInfo) string {
	var b strings.Builder
	b.WriteString("# Fibe MCP Tools Catalog\n\n")
	for _, tool := range tools {
		fmt.Fprintf(&b, "## `%s`\n", tool.Name)
		fmt.Fprintf(
			&b,
			"**Tier:** %s | **Hidden:** %t | **Destructive:** %t | **Idempotent:** %t | **Read-only:** %t\n\n",
			tool.Tier,
			tool.Hidden,
			tool.Destructive,
			tool.Idempotent,
			tool.ReadOnly,
		)
		b.WriteString("### Description\n")
		b.WriteString(strings.TrimSpace(tool.Description))
		b.WriteString("\n")
		if tool.InputSchema != nil {
			schemaJSON, err := json.MarshalIndent(tool.InputSchema, "", "  ")
			if err == nil {
				b.WriteString("\n### Input Schema\n")
				b.WriteString("```json\n")
				b.Write(schemaJSON)
				b.WriteString("\n```\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func markdownTableCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
