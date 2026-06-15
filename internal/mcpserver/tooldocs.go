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
	b.WriteString(summaryMarkdown(tools))
	b.WriteString("| Tool Name | Tier | Advertised in `full` | Description |\n")
	b.WriteString("|-----------|------|----------------------|-------------|\n")
	for _, tool := range tools {
		fmt.Fprintf(
			&b,
			"| `%s` | %s | %s | %s |\n",
			tool.Name,
			markdownTableCell(tool.Tier),
			advertisedInFull(tool),
			markdownTableCell(tool.Description),
		)
	}
	return b.String()
}

func toolCatalogMarkdown(tools []ToolInfo) string {
	var b strings.Builder
	b.WriteString("# Fibe MCP Tools Catalog\n\n")
	b.WriteString(summaryMarkdown(tools))
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

func summaryMarkdown(tools []ToolInfo) string {
	hidden := 0
	core := 0
	for _, tool := range tools {
		if tool.Hidden {
			hidden++
			continue
		}
		if isCoreToolTier(tool.Tier) {
			core++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Generated from the MCP registry.\n\n")
	fmt.Fprintf(&b, "- Registered tools: %d\n", len(tools))
	fmt.Fprintf(&b, "- Advertised with `FIBE_MCP_TOOLS=full`: %d\n", len(tools)-hidden)
	fmt.Fprintf(&b, "- Advertised with `FIBE_MCP_TOOLS=core`: %d\n", core)
	fmt.Fprintf(&b, "- Hidden dispatcher-only tools: %d\n\n", hidden)
	b.WriteString("`full` advertises every non-hidden registered tool. Hidden tools remain dispatcher-reachable through `fibe_call` and `fibe_pipeline`, and `fibe_tools_catalog` reports them with `hidden:true`.\n\n")
	return b.String()
}

func advertisedInFull(tool ToolInfo) string {
	if tool.Hidden {
		return "no"
	}
	return "yes"
}

func isCoreToolTier(tier string) bool {
	switch tier {
	case "meta", "base", "greenfield", "brownfield":
		return true
	default:
		return false
	}
}

func markdownTableCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
