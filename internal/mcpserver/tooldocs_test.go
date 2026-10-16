package mcpserver

import (
	"strings"
	"testing"
)

func TestGenerateToolDocsUsesCanonicalLifecycleTools(t *testing.T) {
	srv := New(DefaultConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("register tools: %v", err)
	}

	docs := GenerateToolDocs(srv.AllTools())
	combined := docs.CatalogMarkdown + "\n" + docs.TableMarkdown
	for _, want := range []string{
		"fibe_launch",
		"fibe_playgrounds_switch_template",
		"fibe_logs_follow",
		"fibe_update_name",
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("generated tool docs missing %s", want)
		}
	}

	retiredNames := []string{
		"fibe_playgrounds_" + "transform",
		"fibe_launch_" + "create",
		"fibe_templates_" + "launch",
		"fibe_playgrounds_logs_" + "follow",
		"fibe_monitor_logs_" + "follow",
	}
	for _, retired := range retiredNames {
		if strings.Contains(combined, retired) {
			t.Fatalf("generated tool docs still include retired tool %s", retired)
		}
	}
}

func TestGenerateToolDocsIncludesLaunchTemplateVersionSelector(t *testing.T) {
	srv := New(DefaultConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("register tools: %v", err)
	}

	docs := GenerateToolDocs(srv.AllTools())
	section := toolDocsSection(docs.CatalogMarkdown, "fibe_launch")
	if !strings.Contains(section, `"version": {`) {
		t.Fatalf("fibe_launch docs missing template version selector:\n%s", section)
	}
}

func TestGenerateToolDocsSortsByToolName(t *testing.T) {
	docs := GenerateToolDocs([]ToolInfo{
		{Name: "fibe_z", Tier: "meta", Description: "z"},
		{Name: "fibe_a", Tier: "meta", Description: "a"},
	})

	first := strings.Index(docs.TableMarkdown, "`fibe_a`")
	second := strings.Index(docs.TableMarkdown, "`fibe_z`")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("table docs are not sorted by tool name:\n%s", docs.TableMarkdown)
	}
}

func toolDocsSection(markdown string, name string) string {
	start := strings.Index(markdown, "## `"+name+"`")
	if start < 0 {
		return ""
	}
	rest := markdown[start+1:]
	next := strings.Index(rest, "\n## `")
	if next < 0 {
		return markdown[start:]
	}
	return markdown[start : start+1+next]
}
