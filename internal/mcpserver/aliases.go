package mcpserver

// aliasField canonicalizes an arg map: if the canonical key is missing but
// any of the provided alternatives is set, the first found alternative is
// copied under the canonical name. Existing canonical values win.
//
// Purpose: the Rails API, the CLI flags, and the MCP tool inputs picked up
// slightly different field names over time (e.g. CLI --sha vs MCP
// found_commit_sha, --body vs template_body). Rather than rename fields
// and break existing pipelines, each affected tool calls aliasField to
// accept the historical alternatives and normalize them before validation
// or SDK dispatch.
func aliasField(args map[string]any, canonical string, alternatives ...string) {
	if args == nil {
		return
	}
	if v, ok := args[canonical]; ok && v != nil && v != "" {
		return
	}
	for _, alt := range alternatives {
		v, ok := args[alt]
		if !ok || v == nil || v == "" {
			continue
		}
		args[canonical] = v
		return
	}
}
