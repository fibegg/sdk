package mcpserver

func (s *Server) registerDomainActionTools() {
	s.registerCompanyTools()
	s.registerAgentActionTools()
	s.registerAgentDefaultsTools()
	s.registerImportTemplateActionTools()
	s.registerTemplateChangeTools()
	s.registerInstallationActionTools()
	s.registerMutterActionTools()
	s.registerGitRepoActionTools()
}
