package mcpserver

// metaTools is the fixed set of tools that are always advertised regardless
// of the FIBE_MCP_TOOLS tier gate. These are the essentials for agent
// journeys and introspection.
var metaTools = map[string]bool{
	"fibe_pipeline":        true,
	"fibe_pipeline_result": true,
	"fibe_help":            true,
	"fibe_run":             true,
	"fibe_auth_set":        true,
	"fibe_me":              true,
	"fibe_status":          true,
	"fibe_limits":          true,
	"fibe_doctor":          true,
	"fibe_schema":          true,
	"fibe_tools_catalog":   true,
	"fibe_call":            true,
}

// includeTool decides whether a tool should be advertised on the mcp-go
// server given the configured toolset tier. Dispatcher registration is
// always unconditional so pipeline steps can still reach "full" tools even
// when the surface is "core".
func (s *Server) includeTool(t *toolImpl) bool {
	if metaTools[t.name] {
		return true
	}
	tier := s.cfg.ToolSet
	if tier == "" {
		tier = "full"
	}
	switch tier {
	case "full":
		return true
	case "core":
		return t.tier == tierCore || t.tier == tierMeta
	default:
		return true
	}
}
