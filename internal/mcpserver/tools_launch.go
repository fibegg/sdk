package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerLaunchTools() {
	s.addTool(&toolImpl{
		name: "fibe_launch", description: "[MODE:GREENFIELD] Launch from exactly one source: template, template version, spec, compose YAML, or repository config. Deployment requires a funded Host; unpaid Hosts return HOST_NOT_FUNDED.", tier: tierGreenfield,
		annotations: toolAnnotations{Idempotent: false},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runLaunch(ctx, c, args)
		},
	}, mcp.NewTool("fibe_launch",
		mcp.WithDescription("Launch from exactly one source: template, template version, spec, compose YAML, or repository config. Template source without version uses the latest template version. Deployment requires a funded Host; unpaid Hosts return HOST_NOT_FUNDED."),
		mcp.WithString("template_id_or_name", mcp.Description("Template ID or name. Mutually exclusive with other source fields; without version/template_version_id, latest version is used.")),
		mcp.WithNumber("version", mcp.Description("Template version number for template_id_or_name. Optional; defaults to latest available version.")),
		mcp.WithNumber("template_version_id", mcp.Description("Exact template version ID. Mutually exclusive with other source fields.")),
		mcp.WithString("spec_id_or_name", mcp.Description("Existing spec ID or name. Mutually exclusive with other source fields.")),
		mcp.WithString("name", mcp.Description("Launch name. Optional when repository_url is provided; inferred from repo name.")),
		mcp.WithString("compose_yaml", mcp.Description("Docker Compose or Fibe YAML content. Optional when repository_url is provided.")),
		mcp.WithString("compose_yaml_path", mcp.Description("Absolute local path to Docker Compose or Fibe YAML (local MCP only). Optional when repository_url is provided.")),
		mcp.WithString("repository_url", mcp.Description("GitHub repository as owner/repo, owner/repo@ref, or https://github.com/owner/repo. Optional alternative to compose_yaml.")),
		mcp.WithString("config_path", mcp.Description("Config file path inside the GitHub repository. Optional; defaults to fibe.yml, fibe.yaml, docker-compose.yml, docker-compose.yaml.")),
		mcp.WithString("github_ref", mcp.Description("Git branch, tag, or commit for the config file. Optional.")),
		mcp.WithString("github_account", mcp.Description("GitHub App installation account owner to use when multiple installations are connected.")),
		mcp.WithNumber("github_installation_id", mcp.Description("GitHub App installation ID to use when multiple installations are connected.")),
		mcp.WithString("host_id_or_name", mcp.Description("Target host ID or name. Required for template/spec launch; compose/repo can omit to create only the spec unless create_playground is true.")),
		mcp.WithBoolean("job_mode", mcp.Description("Create as a task/job instead of a playground. Requires host_id_or_name.")),
		mcp.WithBoolean("create_playground", mcp.Description("Force playground creation. Defaults to true when host_id_or_name is set, false otherwise.")),
		mcp.WithBoolean("persist_volumes", mcp.Description("Persist Docker volumes across task/playground recreations. Optional; omitted means the server infers from named compose volumes.")),
		mcp.WithObject("variables", mcp.Description("Template variables map for Fibe template compilation. Optional.")),
		mcp.WithObject("env_overrides", mcp.Description("Runtime environment overrides for the created Playground.")),
		mcp.WithObject("service_subdomains", mcp.Description("Per-service subdomain overrides.")),
		mcp.WithObject("services", mcp.Description("Per-service runtime Playground configuration overrides.")),
		mcp.WithBoolean("wait", mcp.Description("Wait for created playground to reach running where supported.")),
		mcp.WithNumber("wait_timeout_seconds", mcp.Description("Wait timeout in seconds.")),
		mcp.WithBoolean("diagnose_on_failure", mcp.Description("Diagnose failed waits where supported.")),
		mcp.WithString("response_mode", mcp.Description("Server response detail mode where supported.")),
		mcp.WithObject("repository_mappings", mcp.Description("Map repository URL to Repository ID or name. Optional.")),
	))
}

func (s *Server) runLaunch(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	source, err := mcpLaunchSource(args)
	if err != nil {
		return nil, err
	}
	switch source {
	case "template":
		return launchTemplateArgs(ctx, c, args)
	case "template_version":
		return launchTemplateVersionArgs(ctx, c, args)
	case "spec":
		return launchSpecArgs(ctx, c, args)
	default:
		params, err := launchArgs(ctx, c, args)
		if err != nil {
			return nil, err
		}
		return c.Launch.Create(ctx, params)
	}
}

func mcpLaunchSource(args map[string]any) (string, error) {
	fields := map[string]string{
		"template_id_or_name": "template",
		"template_version_id": "template_version",
		"spec_id_or_name":     "spec",
		"compose_yaml":        "compose",
		"compose_yaml_path":   "compose",
		"repository_url":      "repo",
	}
	selected := ""
	for field, source := range fields {
		if valuePresent(args[field]) {
			if selected != "" && selected != source {
				return "", fmt.Errorf("provide exactly one launch source")
			}
			selected = source
		}
	}
	if selected == "" {
		return "", fmt.Errorf("provide exactly one launch source")
	}
	return selected, nil
}

func valuePresent(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case float64:
		return v > 0
	case int64:
		return v > 0
	case int:
		return v > 0
	default:
		return true
	}
}

func launchTemplateArgs(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	identifier, err := requiredIdentifier(args, "template_id_or_name", "")
	if err != nil {
		return nil, err
	}
	hostID, hostIdentifier, err := resolveMCPHost(ctx, c, args)
	if err != nil {
		return nil, err
	}
	params := &fibe.ImportTemplateLaunchParams{
		HostIdentifier:    hostIdentifier,
		Name:              argString(args, "name"),
		Variables:         argMap(args, "variables"),
		EnvOverrides:      argStringMap(args, "env_overrides"),
		ServiceSubdomains: argStringMap(args, "service_subdomains"),
		Services:          argMap(args, "services"),
	}
	if hostID != nil {
		params.HostID = *hostID
	}
	if v, ok := argInt64(args, "version"); ok && v > 0 {
		params.Version = &v
	}
	if _, ok := args["persist_volumes"]; ok {
		v := argBool(args, "persist_volumes")
		params.PersistVolumes = &v
	}
	return c.ImportTemplates.LaunchWithParamsByIdentifier(ctx, identifier, params)
}

func launchTemplateVersionArgs(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	versionID, ok := argInt64(args, "template_version_id")
	if !ok || versionID <= 0 {
		return nil, fmt.Errorf("template_version_id must be a positive integer")
	}
	hostID, hostIdentifier, err := resolveMCPHost(ctx, c, args)
	if err != nil {
		return nil, err
	}
	params := &fibe.GreenfieldCreateParams{
		Name:              argString(args, "name"),
		TemplateVersionID: &versionID,
		HostIdentifier:    hostIdentifier,
		Variables:         argMap(args, "variables"),
		EnvOverrides:      argStringMap(args, "env_overrides"),
		ServiceSubdomains: argStringMap(args, "service_subdomains"),
		Services:          argMap(args, "services"),
	}
	params.HostID = hostID
	if _, ok := args["persist_volumes"]; ok {
		v := argBool(args, "persist_volumes")
		params.PersistVolumes = &v
	}
	return c.Greenfield.Create(ctx, params)
}

func launchSpecArgs(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
	identifier, err := requiredIdentifier(args, "spec_id_or_name", "")
	if err != nil {
		return nil, err
	}
	params := &fibe.PlaygroundCreateParams{
		Name:           argString(args, "name"),
		SpecIdentifier: identifier,
		Services:       serviceConfigArgs(args["services"]),
	}
	hostID, hostIdentifier, err := resolveMCPHost(ctx, c, args)
	if err != nil {
		return nil, err
	}
	params.HostID = hostID
	params.HostIdentifier = hostIdentifier
	return c.Playgrounds.Create(ctx, params)
}

func argStringMap(args map[string]any, key string) map[string]string {
	raw, ok := args[key].(map[string]any)
	if !ok || raw == nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func serviceConfigArgs(raw any) map[string]*fibe.ServiceConfig {
	values, ok := raw.(map[string]any)
	if !ok || values == nil {
		return nil
	}
	out := make(map[string]*fibe.ServiceConfig, len(values))
	for name, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}
		var cfg fibe.ServiceConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			out[name] = &cfg
		}
	}
	return out
}

func launchArgs(ctx context.Context, c *fibe.Client, args map[string]any) (*fibe.LaunchParams, error) {
	composeYAML, err := readInlineOrPathTextArgOptional(args, "compose_yaml", "compose_yaml_path")
	if err != nil {
		return nil, err
	}
	name := argString(args, "name")
	repoRequest, err := resolveMCPGitHubRepoRequest(ctx, c, args, name)
	if err != nil {
		return nil, err
	}
	if repoRequest != nil {
		if composeYAML != "" {
			return nil, fmt.Errorf("compose_yaml cannot be combined with repository_url")
		}
		name = repoRequest.Name
	}
	if name == "" {
		return nil, fmt.Errorf("required field 'name' not set")
	}
	if composeYAML == "" && repoRequest == nil {
		return nil, fmt.Errorf("required field 'compose_yaml' not set")
	}

	var jobMode *bool
	if _, ok := args["job_mode"]; ok {
		value := argBool(args, "job_mode")
		jobMode = &value
	}
	var createPlayground *bool
	if _, ok := args["create_playground"]; ok {
		value := argBool(args, "create_playground")
		createPlayground = &value
	}
	var hostID *int64
	var hostIdentifier string
	if createPlayground == nil || *createPlayground || (jobMode != nil && *jobMode) {
		var err error
		hostID, hostIdentifier, err = resolveMCPHost(ctx, c, args)
		if err != nil {
			return nil, err
		}
	}
	var persistVolumes *bool
	if _, ok := args["persist_volumes"]; ok {
		value := argBool(args, "persist_volumes")
		persistVolumes = &value
	}

	params := &fibe.LaunchParams{
		ComposeYAML:                  composeYAML,
		Name:                         name,
		JobMode:                      jobMode,
		HostIdentifier:               hostIdentifier,
		CreatePlayground:             createPlayground,
		PersistVolumes:               persistVolumes,
		EnvOverrides:                 argStringMap(args, "env_overrides"),
		ServiceSubdomains:            argStringMap(args, "service_subdomains"),
		Services:                     argMap(args, "services"),
		Variables:                    launchVariables(args["variables"]),
		RepositoryMappings:           map[string]int64{},
		RepositoryMappingIdentifiers: map[string]string{},
	}
	params.HostID = hostID
	if repoRequest != nil {
		params.RepositoryURL = repoRequest.URL
		params.ConfigPath = repoRequest.ConfigPath
		params.GitHubRef = repoRequest.Ref
		params.GitHubAccount = repoRequest.Account
		params.GitHubInstallationID = repoRequest.GitHubInstallationID
	}
	applyLaunchRepositoryMappings(params, args["repository_mappings"])
	return params, nil
}

func launchVariables(raw any) map[string]string {
	out := map[string]string{}
	values, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for key, value := range values {
		normalized := normalizeVariableKey(key)
		item := strings.TrimSpace(fmt.Sprint(value))
		if normalized != "" && item != "" && item != "<nil>" {
			out[normalized] = item
		}
	}
	return out
}

func applyLaunchRepositoryMappings(params *fibe.LaunchParams, raw any) {
	values, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for key, value := range values {
		target := strings.TrimSpace(fmt.Sprint(value))
		if target == "" || target == "<nil>" {
			continue
		}
		if id, err := strconv.ParseInt(target, 10, 64); err == nil {
			params.RepositoryMappings[key] = id
		} else {
			params.RepositoryMappingIdentifiers[key] = target
		}
	}
}
