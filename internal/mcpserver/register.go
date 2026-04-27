package mcpserver

// registerTools wires all tools into the MCP server.
//
// The toolset is filtered by s.cfg.ToolSet so servers targeting
// token-sensitive agent sessions can ship a smaller surface. Use "full" for
// every tier, "core" for meta+base+greenfield+brownfield, or a comma list of
// named tiers such as "other,meta".
func (s *Server) registerTools() error {
	if _, err := parseToolTierSelection(s.cfg.ToolSet); err != nil {
		return err
	}
	// NOTE: the toolset filter runs lazily inside addTool. We register all
	// tools unconditionally here and the filter drops ones that don't match
	// the requested tier at MCP advertisement time.
	s.registerResourceMutationTools()
	s.registerResourceTools()
	s.registerCustomTools()
	s.registerGreenfieldTools()
	s.registerMetaTools()
	s.registerWaitTool()
	s.registerLogsFollowTool()
	s.registerMonitorFollowTool()
	s.registerPipelineTools()
	s.registerArtefactActionTools()
	s.registerDomainActionTools()
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
