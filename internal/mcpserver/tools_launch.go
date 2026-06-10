package mcpserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerLaunchTools() {
	s.addTool(&toolImpl{
		name: "fibe_launch_create", description: "[MODE:GREENFIELD] Create a playspec and optionally deploy a playground from compose YAML or a GitHub repository config file. Deployment requires a funded Marquee; unpaid Marquees return MARQUEE_NOT_FUNDED.", tier: tierGreenfield,
		annotations: toolAnnotations{Idempotent: false},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			params, err := launchArgs(ctx, c, args)
			if err != nil {
				return nil, err
			}
			return c.Launch.Create(ctx, params)
		},
	}, mcp.NewTool("fibe_launch_create",
		mcp.WithDescription("Create a playspec and optionally deploy a playground from compose YAML or a GitHub repository config file. Deployment requires a funded Marquee; unpaid Marquees return MARQUEE_NOT_FUNDED."),
		mcp.WithString("name", mcp.Description("Launch name. Optional when repository_url is provided; inferred from repo name.")),
		mcp.WithString("compose_yaml", mcp.Description("Docker Compose or Fibe YAML content. Optional when repository_url is provided.")),
		mcp.WithString("compose_yaml_path", mcp.Description("Absolute local path to Docker Compose or Fibe YAML (local MCP only). Optional when repository_url is provided.")),
		mcp.WithString("repository_url", mcp.Description("GitHub repository as owner/repo, owner/repo@ref, or https://github.com/owner/repo. Optional alternative to compose_yaml.")),
		mcp.WithString("config_path", mcp.Description("Config file path inside the GitHub repository. Optional; defaults to fibe.yml, fibe.yaml, docker-compose.yml, docker-compose.yaml.")),
		mcp.WithString("github_ref", mcp.Description("Git branch, tag, or commit for the config file. Optional.")),
		mcp.WithString("github_account", mcp.Description("GitHub App installation account owner to use when multiple installations are connected.")),
		mcp.WithNumber("github_installation_id", mcp.Description("GitHub App installation ID to use when multiple installations are connected.")),
		mcp.WithString("marquee_id_or_name", mcp.Description("Target marquee ID or name. Optional; without it only the playspec is created.")),
		mcp.WithBoolean("job_mode", mcp.Description("Create as a trick/job instead of a playground. Requires marquee_id_or_name.")),
		mcp.WithBoolean("create_playground", mcp.Description("Force playground creation. Defaults to true when marquee_id_or_name is set, false otherwise.")),
		mcp.WithBoolean("persist_volumes", mcp.Description("Persist Docker volumes across trick/playground recreations. Optional; omitted means the server infers from named compose volumes.")),
		mcp.WithObject("variables", mcp.Description("Template variables map for Fibe template compilation. Optional.")),
		mcp.WithObject("prop_mappings", mcp.Description("Map repository URL to Prop ID or name. Optional.")),
	))
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

	marqueeIdentifier := argString(args, "marquee_id_or_name")
	marqueeID, marqueeIDOK := argInt64(args, "marquee_id_or_name")
	var jobMode *bool
	if _, ok := args["job_mode"]; ok {
		value := argBool(args, "job_mode")
		jobMode = &value
	}
	if jobMode != nil && *jobMode && !marqueeIDOK && marqueeIdentifier == "" {
		return nil, fmt.Errorf("job_mode requires marquee_id_or_name")
	}
	var createPlayground *bool
	if _, ok := args["create_playground"]; ok {
		value := argBool(args, "create_playground")
		createPlayground = &value
	}
	var persistVolumes *bool
	if _, ok := args["persist_volumes"]; ok {
		value := argBool(args, "persist_volumes")
		persistVolumes = &value
	}

	params := &fibe.LaunchParams{
		ComposeYAML:            composeYAML,
		Name:                   name,
		JobMode:                jobMode,
		MarqueeIdentifier:      marqueeIdentifier,
		CreatePlayground:       createPlayground,
		PersistVolumes:         persistVolumes,
		Variables:              launchVariables(args["variables"]),
		PropMappings:           map[string]int64{},
		PropMappingIdentifiers: map[string]string{},
	}
	if marqueeIDOK && marqueeID > 0 {
		params.MarqueeID = &marqueeID
	}
	if repoRequest != nil {
		params.RepositoryURL = repoRequest.URL
		params.ConfigPath = repoRequest.ConfigPath
		params.GitHubRef = repoRequest.Ref
		params.GitHubAccount = repoRequest.Account
		params.GitHubInstallationID = repoRequest.GitHubInstallationID
	}
	applyLaunchPropMappings(params, args["prop_mappings"])
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

func applyLaunchPropMappings(params *fibe.LaunchParams, raw any) {
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
			params.PropMappings[key] = id
		} else {
			params.PropMappingIdentifiers[key] = target
		}
	}
}
