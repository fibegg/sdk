package mcpserver

// This file wires domain-specific MCP actions that are not covered by the
// generic flat resource tools. Tools that accept binary payloads
// (mounted files, artefact upload, template images) expect base64-encoded
// content under a `content_base64` arg.

func (s *Server) registerDomainActionTools() {
	s.registerAgentActionTools()
	s.registerImportTemplateActionTools()
	s.registerTemplateDevelopTools()
	s.registerInstallationActionTools()
	s.registerMutterActionTools()
	s.registerGitRepoActionTools()
}
