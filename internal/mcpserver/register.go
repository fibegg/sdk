package mcpserver

// registerTools wires all tools into the MCP server.
//
// The toolset is filtered by s.cfg.ToolSet ("core" or "full") so servers
// targeting token-sensitive agent sessions can ship a smaller surface. Meta
// tools (fibe_pipeline, fibe_help, fibe_run, fibe_auth_set) are always
// registered regardless of tier.
func (s *Server) registerTools() error {
	// NOTE: the toolset filter runs lazily inside addTool — see
	// applyToolSetFilter. We register all tools unconditionally here and
	// the filter drops ones that don't match the tier at list time.
	s.registerGeneratedTools()
	s.registerCustomTools()
	s.registerMetaTools()
	s.registerWaitTool()
	s.registerLogsFollowTool()
	s.registerMonitorFollowTool()
	s.registerPipelineTools()
	s.registerParityTools()
	s.registerDiscoveryTools()
	return nil
}

// registerResources wires MCP resources: schema, help, me, status, and
// cached pipeline outputs. Resources are the right vehicle for "load once
// at session start" content that doesn't warrant a tool call.
func (s *Server) registerResources() error {
	s.registerStaticResources()
	s.registerResourceTemplates()
	return nil
}
