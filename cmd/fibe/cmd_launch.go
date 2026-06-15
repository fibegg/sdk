package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func launchCmd() *cobra.Command {
	var (
		name                 string
		template             string
		templateVersion      string
		playspec             string
		compose              string
		repo                 string
		repoFile             string
		repoRef              string
		githubAccount        string
		githubInstallationID int64
		marquee              string
		version              int64
		jobMode              bool
		createPlayground     bool
		noCreatePlayground   bool
		persistVolumes       bool
		wait                 bool
		waitTimeout          time.Duration
		launchVars           []string
		launchProps          []string
		envFlags             []string
		subdomainFlags       []string
		serviceFlags         []string
	)

	cmd := &cobra.Command{
		Use:   "launch [source]",
		Short: "Launch a playspec, template, compose file, or GitHub repository",
		Long: `Launch a Fibe app from one explicit source.

SOURCE SELECTION:
  --template <id-or-name>          Launch latest or selected version of a template
  --template-version <id>          Launch an exact template version ID
  --playspec <id-or-name>          Create a playground from an existing playspec
  --compose <yaml-or-@file>        Create a playspec and playground from compose YAML
  --repo <owner/repo[@ref]|url>    Create a playspec and playground from repository config

If a template is selected and neither --version nor --template-version is set,
the latest template version is used. If --marquee is omitted, the SDK uses
FIBE_MARQUEE_ID or infers the only launchable Marquee.

SERVICE OVERRIDES:
  --service SERVICE.FIELD=VALUE applies runtime Playground config only. It does
  not edit the compose file, TemplateVersion, or Playspec.

EXAMPLES:
  fibe launch --template billing-app --name billing-staging --marquee next --persist-volumes --subdomain web=billing-staging --service web.exposure_port=3000 --service web.exposure_visibility=external
  fibe launch --template-version 912 --name branch-a --marquee next --subdomain web=branch-a --wait
  fibe launch --playspec starter --name demo --marquee next --service worker.env_vars.QUEUE=critical
  fibe launch --compose @docker-compose.yml --name demo --marquee next --persist-volumes
  fibe launch --repo owner/repo@main --name demo --marquee next`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			var filePayload map[string]any
			if err := applyFromFile(&filePayload); err != nil {
				return err
			}
			if compose == "" && len(rawPayload) > 0 && len(args) == 0 {
				compose = string(rawPayload)
			}
			source, err := detectLaunchSource(cmd, c, args, launchSourceFlagValues{
				Template:        template,
				TemplateVersion: templateVersion,
				Playspec:        playspec,
				Compose:         compose,
				Repo:            repo,
			})
			if err != nil {
				return err
			}

			serviceOverrides, err := parsePlaygroundServiceOverrides(serviceFlags)
			if err != nil {
				return err
			}
			subdomains, err := parseSubdomainFlags(subdomainFlags)
			if err != nil {
				return err
			}
			envOverrides, err := parseEnvFlags(envFlags)
			if err != nil {
				return err
			}
			variables, err := parseKeyValueFlags(launchVars)
			if err != nil {
				return err
			}
			marqueeIdentifier, err := resolveLaunchMarqueeIdentifier(c, marquee)
			if err != nil {
				return err
			}

			var result any
			var playgroundID int64
			switch source.Kind {
			case launchSourceTemplate:
				if templateVersion != "" {
					return fmt.Errorf("--template cannot be combined with --template-version; use one source selector")
				}
				result, playgroundID, err = runTemplateLaunch(c, source.Value, name, marqueeIdentifier, version, persistVolumes, cmd.Flags().Changed("persist-volumes"), variables, envOverrides, subdomains, serviceOverrides)
			case launchSourceTemplateVersion:
				result, playgroundID, err = runTemplateVersionLaunch(c, source.Value, name, marqueeIdentifier, persistVolumes, cmd.Flags().Changed("persist-volumes"), variables, envOverrides, subdomains, serviceOverrides)
			case launchSourcePlayspec:
				result, playgroundID, err = runPlayspecLaunch(c, source.Value, name, marqueeIdentifier, serviceOverrides)
			case launchSourceCompose, launchSourceRepo:
				result, playgroundID, err = runComposeOrRepoLaunch(cmd, c, source, args, name, repoFile, repoRef, githubAccount, githubInstallationID, marqueeIdentifier, jobMode, createPlayground, noCreatePlayground, persistVolumes, cmd.Flags().Changed("persist-volumes"), launchProps, variables, envOverrides, subdomains, serviceOverrides)
			default:
				err = fmt.Errorf("unsupported launch source %q", source.Kind)
			}
			if err != nil {
				return err
			}
			if wait {
				if err := waitForCreatedPlayground(c, playgroundID, waitTimeout); err != nil {
					return err
				}
			}
			outputJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&template, "template", "", "Template ID or name")
	cmd.Flags().StringVar(&templateVersion, "template-version", "", "Exact template version ID")
	cmd.Flags().StringVar(&playspec, "playspec", "", "Playspec ID or name")
	cmd.Flags().StringVar(&compose, "compose", "", "Docker Compose or Fibe YAML content, or @path")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repository as owner/repo, owner/repo@ref, or URL")
	cmd.Flags().StringVar(&name, "name", "", "Playground/app name")
	cmd.Flags().StringVar(&marquee, "marquee", "", "Target Marquee ID or name; inferred when exactly one launchable Marquee exists")
	cmd.Flags().Int64Var(&version, "version", 0, "Template version number when --template is used; omitted means latest")
	cmd.Flags().StringVar(&repoFile, "file", "", "Config file path inside the GitHub repository")
	cmd.Flags().StringVar(&repoRef, "ref", "", "Git branch, tag, or commit for repository config")
	cmd.Flags().StringVar(&githubAccount, "github-account", "", "GitHub App installation account owner")
	cmd.Flags().Int64Var(&githubInstallationID, "github-installation-id", 0, "GitHub App installation ID")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Create as a trick/job for compose or repo sources")
	cmd.Flags().BoolVar(&createPlayground, "create-playground", false, "Force playground creation for compose or repo sources")
	cmd.Flags().BoolVar(&noCreatePlayground, "no-create-playground", false, "Create only the playspec for compose or repo sources")
	cmd.Flags().BoolVar(&persistVolumes, "persist-volumes", false, "Persist Docker volumes across playground recreations")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for created playground to reach running")
	cmd.Flags().DurationVar(&waitTimeout, "wait-timeout", 10*time.Minute, "Maximum wait duration")
	cmd.Flags().StringArrayVar(&launchVars, "var", nil, "Template variable override as key=value (repeatable)")
	cmd.Flags().StringArrayVar(&launchProps, "prop", nil, "Map private Git repository to Prop ID or name for compose/repo sources")
	cmd.Flags().StringArrayVar(&envFlags, "env", nil, "Runtime environment override as KEY=VALUE (repeatable)")
	cmd.Flags().StringArrayVar(&subdomainFlags, "subdomain", nil, "Service subdomain override as SERVICE=SUBDOMAIN (repeatable)")
	cmd.Flags().StringArrayVar(&serviceFlags, "service", nil, "Runtime service override as SERVICE.FIELD=VALUE (repeatable)")
	return cmd
}

func runTemplateLaunch(c *fibe.Client, template, name, marquee string, version int64, persistVolumes bool, persistChanged bool, variables map[string]any, env map[string]string, subdomains map[string]string, services map[string]*fibe.ServiceConfig) (*fibe.LaunchResult, int64, error) {
	params := &fibe.ImportTemplateLaunchParams{
		MarqueeIdentifier: marquee,
		Name:              name,
		Variables:         variables,
		EnvOverrides:      env,
		ServiceSubdomains: subdomains,
		Services:          serviceConfigMapAny(services),
	}
	if version > 0 {
		params.Version = &version
	}
	if persistChanged {
		params.PersistVolumes = &persistVolumes
	}
	result, err := c.ImportTemplates.LaunchWithParamsByIdentifier(ctx(), template, params)
	if err != nil {
		return nil, 0, err
	}
	return result, result.PlaygroundID, nil
}

func runTemplateVersionLaunch(c *fibe.Client, templateVersion, name, marquee string, persistVolumes bool, persistChanged bool, variables map[string]any, env map[string]string, subdomains map[string]string, services map[string]*fibe.ServiceConfig) (*fibe.GreenfieldResult, int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(templateVersion), 10, 64)
	if err != nil || id <= 0 {
		return nil, 0, fmt.Errorf("--template-version must be a positive integer ID")
	}
	params := &fibe.GreenfieldCreateParams{
		Name:              name,
		TemplateVersionID: &id,
		MarqueeIdentifier: marquee,
		Variables:         variables,
		EnvOverrides:      env,
		ServiceSubdomains: subdomains,
		Services:          serviceConfigMapAny(services),
	}
	if persistChanged {
		params.PersistVolumes = &persistVolumes
	}
	result, err := c.Greenfield.Create(ctx(), params)
	if err != nil {
		return nil, 0, err
	}
	var playgroundID int64
	if result.Playground != nil {
		playgroundID = result.Playground.ID
	}
	return result, playgroundID, nil
}

func runPlayspecLaunch(c *fibe.Client, playspec, name, marquee string, services map[string]*fibe.ServiceConfig) (*fibe.Playground, int64, error) {
	if name == "" {
		return nil, 0, fmt.Errorf("required field 'name' not set")
	}
	ps, err := c.Playspecs.GetByIdentifier(ctx(), playspec)
	if err != nil {
		return nil, 0, err
	}
	if err := validateServiceOverrideNames(playspecServiceNames(ps), services); err != nil {
		return nil, 0, err
	}
	params := &fibe.PlaygroundCreateParams{
		Name:               name,
		PlayspecIdentifier: playspec,
		MarqueeIdentifier:  marquee,
		Services:           services,
	}
	result, err := c.Playgrounds.Create(ctx(), params)
	if err != nil {
		return nil, 0, err
	}
	return result, result.ID, nil
}

func runComposeOrRepoLaunch(cmd *cobra.Command, c *fibe.Client, source launchSource, args []string, name, repoFile, repoRef, githubAccount string, githubInstallationID int64, marquee string, jobMode, createPlayground, noCreatePlayground, persistVolumes, persistChanged bool, launchProps []string, variables map[string]any, env map[string]string, subdomains map[string]string, services map[string]*fibe.ServiceConfig) (*fibe.LaunchResult, int64, error) {
	params := &fibe.LaunchParams{Name: name, MarqueeIdentifier: marquee}
	if source.Kind == launchSourceCompose {
		params.ComposeYAML = resolveStringValue(source.Value)
		if params.ComposeYAML == "" && len(rawPayload) > 0 {
			params.ComposeYAML = string(rawPayload)
		}
	}
	if source.Kind == launchSourceRepo {
		params.RepositoryURL = source.Value
	}
	if cmd.Flags().Changed("file") {
		params.ConfigPath = repoFile
	}
	if cmd.Flags().Changed("ref") {
		params.GitHubRef = repoRef
	}
	if cmd.Flags().Changed("github-account") {
		params.GitHubAccount = githubAccount
	}
	if cmd.Flags().Changed("github-installation-id") && githubInstallationID > 0 {
		id := githubInstallationID
		params.GitHubInstallationID = &id
	}
	if cmd.Flags().Changed("job-mode") && jobMode {
		t := true
		params.JobMode = &t
	}
	if cmd.Flags().Changed("create-playground") {
		v := createPlayground
		params.CreatePlayground = &v
	}
	if cmd.Flags().Changed("no-create-playground") && noCreatePlayground {
		v := false
		params.CreatePlayground = &v
	}
	if persistChanged {
		v := persistVolumes
		params.PersistVolumes = &v
	}
	params.EnvOverrides = env
	params.ServiceSubdomains = subdomains
	params.Services = serviceConfigMapAny(services)
	params.Variables = mapStringAnyToString(variables)
	applyLaunchProps(params, launchProps)
	if source.Kind == launchSourceRepo {
		repoRequest, err := resolveGitHubRepoRequest(cmd, c, nil, githubRepoRequestOptions{
			ExistingURL:            params.RepositoryURL,
			ExistingName:           params.Name,
			ExistingRef:            params.GitHubRef,
			ExistingConfigPath:     params.ConfigPath,
			ExistingAccount:        params.GitHubAccount,
			ExistingInstallationID: params.GitHubInstallationID,
			FlagRef:                repoRef,
			FlagFile:               repoFile,
			FlagAccount:            githubAccount,
			FlagInstallationID:     githubInstallationID,
		})
		if err != nil {
			return nil, 0, err
		}
		if repoRequest != nil {
			params.RepositoryURL = repoRequest.URL
			params.Name = repoRequest.Name
			params.GitHubRef = repoRequest.Ref
			params.ConfigPath = repoRequest.ConfigPath
			params.GitHubAccount = repoRequest.Account
			params.GitHubInstallationID = repoRequest.GitHubInstallationID
		}
	}
	if params.Name == "" {
		return nil, 0, fmt.Errorf("required field 'name' not set")
	}
	if params.ComposeYAML != "" {
		if err := validateServiceOverrideNames(composeServiceNames(params.ComposeYAML), services); err != nil {
			return nil, 0, err
		}
	}
	result, err := c.Launch.Create(ctx(), params)
	if err != nil {
		return nil, 0, err
	}
	return result, result.PlaygroundID, nil
}

func applyLaunchProps(params *fibe.LaunchParams, values []string) {
	if len(values) == 0 {
		return
	}
	params.PropMappings = make(map[string]int64)
	params.PropMappingIdentifiers = make(map[string]string)
	for _, value := range values {
		key, target, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		if id, err := strconv.ParseInt(target, 10, 64); err == nil {
			params.PropMappings[key] = id
		} else {
			params.PropMappingIdentifiers[key] = target
		}
	}
}

func mapStringAnyToString(values map[string]any) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = fmt.Sprint(value)
	}
	return out
}
