package mcpserver

import (
	"bytes"
	"strings"

	"github.com/spf13/cobra"
)

var cliResources = map[string]string{
	"agents": "agent", "agents pokes": "agent_poke", "api-keys": "api_key",
	"audit-logs": "audit_log", "env-packs": "env_pack", "hosts": "host", "job-env": "job_env",
	"playgrounds": "playground", "repositories": "repository", "secrets": "secret",
	"specs": "spec", "tasks": "task", "templates": "template", "templates versions": "template_version",
	"webhooks": "webhook",
}

// These entries describe commands whose SDK operation differs from the
// resource operation of the same name. Unmapped commands remain available.
var cliRoutes = map[string]operationRoutes{
	"agents activity":                      "AgentService.GetActivityByIdentifierWithParams",
	"agents add-mounted-file":              "AgentService.AddMountedFileByIdentifier",
	"agents authenticate":                  "AgentService.AuthenticateByIdentifierWithParams",
	"agents chat":                          "AgentService.ChatByIdentifier",
	"agents create-conversation":           "AgentService.CreateConversationByIdentifier",
	"agents defaults get":                  "AgentDefaultsService.Get",
	"agents defaults reset":                "AgentDefaultsService.Reset",
	"agents defaults update":               "AgentDefaultsService.Update",
	"agents delete-conversation":           "AgentService.DeleteConversationByIdentifier",
	"agents download-attachment":           "AgentService.DownloadAttachmentByIdentifier",
	"agents duplicate":                     "AgentService.DuplicateByIdentifier",
	"agents gitea-token":                   "AgentService.GetGiteaTokenByIdentifier",
	"agents interrupt":                     "AgentService.InterruptByIdentifier",
	"agents live-state":                    "AgentService.LiveStateByIdentifier",
	"agents messages":                      "AgentService.GetMessagesByIdentifierWithParams",
	"agents purge-chat":                    "AgentService.PurgeChatByIdentifier",
	"agents remove-mounted-file":           "AgentService.RemoveMountedFileByIdentifier",
	"agents runtime-status":                "AgentService.RuntimeStatusByIdentifier",
	"agents set-activity":                  "AgentService.UpdateActivityByIdentifier",
	"agents set-messages":                  "AgentService.UpdateMessagesByIdentifier",
	"agents start-chat":                    "AgentService.StartChatByAgentIdentifier",
	"agents update-mounted-file":           "AgentService.UpdateMountedFileByIdentifier",
	"artefacts create":                     artefactUploadRoutes,
	"artefacts download":                   "GET:/api/agents/{agent}/artefacts/{id}/download",
	"artefacts get":                        "GET:/api/agents/{agent}/artefacts/{id}",
	"artefacts list":                       "ArtefactService.ListAll GET:/api/agents/{agent}/artefacts",
	"artefacts update":                     "ArtefactService.UpdateByIdentifier",
	"categories":                           "TemplateCategoryService.List",
	"env-packs attachments detach":         envPackModifyRoutes,
	"env-packs attachments get":            envPackGetRoutes,
	"env-packs attachments renew":          envPackRenewRoutes,
	"env-packs attachments reorder":        envPackModifyRoutes,
	"env-packs attachments retarget":       envPackModifyRoutes,
	"env-packs attachments set":            envPackReplaceRoutes,
	"feedbacks create":                     "FeedbackService.CreateByAgentIdentifier",
	"feedbacks delete":                     "DELETE:/api/agents/{agent}/feedbacks/{id}",
	"feedbacks get":                        "GET:/api/agents/{agent}/feedbacks/{id}",
	"feedbacks list":                       "GET:/api/agents/{agent}/feedbacks",
	"gitea-repos create":                   "GiteaRepoService.Create",
	"github apps connect":                  "GitHubAppService.ConnectInfo",
	"github-repos create":                  "GitHubRepoService.Create",
	"greenfield":                           "GreenfieldService.Create",
	"installations find-repos":             "GET:/api/github_repositories",
	"installations get-token":              "InstallationService.GetGitHubToken",
	"installations list":                   "GET:/api/installations",
	"installations repos":                  "GET:/api/installations/{id}/repos",
	"installations token":                  "GET:/api/installations/{id}/token",
	"job-env set":                          "JobEnvService.Set",
	"launch":                               launchRoutes,
	"monitor follow":                       "MonitorService.List",
	"monitor list":                         "MonitorService.List",
	"monitor logs":                         followLogsRoutes,
	"mutters create":                       "MutterService.CreateItemByAgentIdentifier",
	"mutters get":                          "GET:/api/agents/{agent}/mutter",
	"playgrounds compose":                  "PlaygroundService.ComposeByIdentifier",
	"playgrounds debug":                    "PlaygroundService.DebugWithBuildLogsByIdentifier",
	"playgrounds env":                      "PlaygroundService.EnvMetadataByIdentifier",
	"playgrounds extend":                   "PlaygroundService.ExtendExpirationByIdentifier",
	"playgrounds hard-restart":             "PlaygroundService.ActionByIdentifier",
	"playgrounds logs":                     "PlaygroundService.LogsByIdentifier",
	"playgrounds maintenance disable":      "PlaygroundService.ActionByIdentifier",
	"playgrounds maintenance enable":       "PlaygroundService.ActionByIdentifier",
	"playgrounds rollout":                  "PlaygroundService.ActionByIdentifier",
	"playgrounds start":                    "PlaygroundService.ActionByIdentifier",
	"playgrounds status":                   "PlaygroundService.RuntimeStatusByIdentifier",
	"playgrounds stop":                     "PlaygroundService.ActionByIdentifier",
	"playgrounds switch-template":          playgroundSwitchRoutes,
	"repo-status check":                    "RepoStatusService.Check",
	"repositories branches":                "GET:/api/repositories/{id}/branches",
	"repositories env-defaults":            "RepositoryService.EnvDefaultsByIdentifier",
	"server-info":                          "ServerInfoService.Get",
	"specs add-mounted-file":               "SpecService.AddMountedFileByIdentifier",
	"specs add-registry-credential":        "SpecService.AddRegistryCredentialByIdentifier",
	"specs remove-mounted-file":            "SpecService.RemoveMountedFileByIdentifier",
	"specs remove-registry-credential":     "SpecService.RemoveRegistryCredentialByIdentifier",
	"specs services":                       "SpecService.ServicesByIdentifier",
	"specs switch-template":                templateSwitchPreview + " " + templateSwitchApply,
	"specs update-mounted-file":            "SpecService.UpdateMountedFileByIdentifier",
	"specs validate-compose":               "SpecService.ValidateComposeWithParams",
	"status":                               "StatusService.Get",
	"tasks logs":                           "PlaygroundService.LogsByIdentifier",
	"tasks status":                         "PlaygroundService.StatusByIdentifier",
	"team git-installations connect":       "Client.ConnectTeamGitInstallation",
	"team git-installations list":          "Client.TeamGitInstallations",
	"team git-installations remove":        "Client.RemoveTeamGitInstallation",
	"team principals create":               "Client.CreateTeamServicePrincipal",
	"team principals list":                 "Client.TeamServicePrincipals",
	"team principals recover":              "Client.RecoverRuntimePrincipal",
	"team principals revoke":               "Client.RevokeTeamServicePrincipal",
	"team repository-operations adopt":     "Client.AdoptTeamRepositoryOperation",
	"team repository-operations list":      "Client.TeamRepositoryOperations",
	"team repository-operations reconcile": "Client.ReconcileTeamRepositoryOperation",
	"templates create-version":             "ImportTemplateService.CreateVersionByIdentifier",
	"templates destroy-version":            "ImportTemplateVersionService.DeleteByTemplateIdentifier",
	"templates search":                     "ImportTemplateService.SearchWithParams",
	"templates source clear":               "ImportTemplateService.ClearSourceByIdentifier",
	"templates source refresh":             "ImportTemplateService.RefreshSourceByIdentifier",
	"templates source set":                 "ImportTemplateService.SetSourceByIdentifier",
	"templates toggle-public":              "ImportTemplateService.TogglePublicByIdentifier",
	"templates upload-image":               "ImportTemplateService.UploadImageByIdentifier",
	"templates versions":                   "ImportTemplateService.ListVersionsByIdentifier ImportTemplateService.CreateVersionByIdentifier ImportTemplateVersionService.DeleteByTemplateIdentifier ImportTemplateService.TogglePublicByIdentifier",
	"templates versions destroy":           "ImportTemplateVersionService.DeleteByTemplateIdentifier",
	"wait":                                 "PlaygroundService.RuntimeStatusByIdentifier RepositoryService.GetMirrorStateByIdentifier PlaygroundService.StatusByIdentifier",
	"webhooks deliveries":                  "WebhookEndpointService.ListDeliveries",
}

func commandRoutes(path string) operationRoutes {
	if routes, ok := cliRoutes[path]; ok {
		return routes
	}
	if split := strings.LastIndexByte(path, ' '); split >= 0 {
		return resourceOperationRoutes(cliResources[path[:split]], strings.ReplaceAll(path[split+1:], "-", "_"))
	}
	return ""
}

func cliPath(cmd *cobra.Command) string {
	return strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
}

func (m *routeManifest) offersCommand(cmd *cobra.Command) bool {
	if m == nil || (cmd.Runnable() && m.offers(commandRoutes(cliPath(cmd)))) {
		return true
	}
	for _, child := range cmd.Commands() {
		if !child.Hidden && m.offersCommand(child) {
			return true
		}
	}
	return cmd == cmd.Root()
}

func (s *Server) checkRunEdition(manifest *routeManifest, args map[string]any) error {
	raw, ok := args["args"].([]any)
	if !ok || s.cfg.CobraRoot == nil {
		return nil
	}
	tokens := make([]string, 0, len(raw))
	for _, value := range raw {
		token, err := stringifyCLIArg(value)
		if err != nil {
			return nil
		}
		tokens = append(tokens, token)
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	command, remaining, err := s.cfg.CobraRoot.Find(tokens)
	if err != nil || command == nil {
		return nil
	}
	path := cliPath(command)
	routes := commandRoutes(path)
	switch path {
	case "launch":
		switch {
		case containsCLIFlag(tokens, "--template-version"):
			routes = "GreenfieldService.Create"
		case containsCLIFlag(tokens, "--template"):
			routes = "ImportTemplateService.LaunchWithParamsByIdentifier"
		case containsCLIFlag(tokens, "--spec"):
			routes = "PlaygroundService.Create"
		case containsCLIFlag(tokens, "--compose") || containsCLIFlag(tokens, "--repo"):
			routes = "LaunchService.Create"
		}
	case "monitor logs":
		if cliFlagValue(tokens, "--target") == "task" {
			routes = followTaskLogs
		}
	case "env-packs attachments get", "env-packs attachments set", "env-packs attachments renew",
		"env-packs attachments reorder", "env-packs attachments detach", "env-packs attachments retarget":
		if len(remaining) > 0 {
			routes = selectResourceRoutes("env_pack", "", map[string]any{"target": remaining[0]}, routes)
		}
	}
	if !manifest.offersCommand(command) {
		return unavailable(path)
	}
	return manifest.require(path, routes)
}

func (s *Server) renderCLIHelp(cmd *cobra.Command, manifest *routeManifest) string {
	// Cobra writes via the shared command tree. The caller holds runMu.
	var buf bytes.Buffer
	previous := cmd.OutOrStdout()
	cmd.SetOut(&buf)
	_ = cmd.Help()
	cmd.SetOut(previous)
	if manifest == nil {
		return buf.String()
	}

	unavailableChildren := map[string]bool{}
	for _, child := range cmd.Commands() {
		if !manifest.offersCommand(child) {
			unavailableChildren[child.Name()] = true
		}
	}
	var lines []string
	inCommands := false
	for _, line := range strings.Split(buf.String(), "\n") {
		if line == "Available Commands:" {
			inCommands = true
		} else if inCommands && !strings.HasPrefix(line, "  ") {
			inCommands = false
		}
		if inCommands {
			fields := strings.Fields(line)
			if len(fields) > 0 && unavailableChildren[fields[0]] {
				continue
			}
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func cliFlagValue(tokens []string, flag string) string {
	for i, token := range tokens {
		if token == flag && i+1 < len(tokens) {
			return tokens[i+1]
		}
		if value, ok := strings.CutPrefix(token, flag+"="); ok {
			return value
		}
	}
	return ""
}
