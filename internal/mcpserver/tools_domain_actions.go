package mcpserver

func (s *Server) registerDomainActionTools() {
	s.registerAgentActionTools()
	s.registerAgentDefaultsTools()
	s.registerImportTemplateActionTools()
	s.registerTemplateChangeTools()
	s.registerInstallationActionTools()
	s.registerMutterActionTools()
	s.registerGitRepoActionTools()
}
