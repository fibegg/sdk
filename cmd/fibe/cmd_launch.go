package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func launchCmd() *cobra.Command {
	var name, compose string
	var repoFile, repoRef, githubAccount string
	var githubInstallationID int64
	var jobMode, createPlayground, noCreatePlayground, persistVolumes bool
	var marqueeID string
	var launchVars []string
	var launchProps []string
	cmd := &cobra.Command{
		Use:   "launch [github-repo]",
		Short: "One-shot: parse compose YAML -> create playspec -> (optionally) deploy playground",
		Long: `One-shot: parse compose YAML -> create playspec -> (optionally) deploy playground on a marquee.
Fastest path from raw Docker Compose YAML to a running environment.

Deploying a playground or trick requires a funded target Marquee. If billing is
expired or missing, the server returns MARQUEE_NOT_FUNDED before deployment starts.

PLAYGROUND CREATION RULES:
  - When --marquee-id is provided, a playground is created on that marquee by default.
  - When --marquee-id is omitted, only the playspec (and any props) are created;
    no playground is deployed and the response carries playground_id=0.
  - --job-mode (trick / CI-Job) REQUIRES --marquee-id; otherwise the trick has nowhere to run.
  - Pass --no-create-playground with a marquee id to skip playground creation explicitly.

REQUIRED INPUT:
  github-repo positional argument, or --compose Docker Compose/Fibe YAML content.
  --name is optional with github-repo and inferred from the repository name.

OPTIONAL FLAGS:
  --marquee-id            Target marquee for the playground/trick
  --job-mode              Create as a trick (job-mode) instead of a playground (requires --marquee-id)
  --persist-volumes       Persist Docker volumes across trick/playground recreations
  --no-create-playground  Create only the playspec; skip playground deployment even with --marquee-id

EXAMPLES:
  fibe launch owner/repo --marquee-id 12
  fibe launch owner/repo@main --file fibe.yml --marquee-id 12
  fibe launch https://github.com/owner/repo --ref main --marquee-id 12
  fibe launch --name my-app --compose @docker-compose.yml --marquee-id 12
  fibe launch --name ci-run --compose @docker-compose.yml --marquee-id 12 --job-mode
  fibe launch owner/repo --ref unstable --marquee-id 12 --job-mode --persist-volumes
  fibe launch --name spec-only --compose @docker-compose.yml --no-create-playground` + generateSchemaDoc(&fibe.LaunchParams{}),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.LaunchParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("compose") {
				params.ComposeYAML = resolveStringValue(compose)
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
			if cmd.Flags().Changed("marquee-id") && marqueeID != "" {
				params.MarqueeIdentifier = marqueeID
			}
			if cmd.Flags().Changed("create-playground") {
				v := createPlayground
				params.CreatePlayground = &v
			}
			if cmd.Flags().Changed("no-create-playground") && noCreatePlayground {
				v := false
				params.CreatePlayground = &v
			}
			if cmd.Flags().Changed("persist-volumes") {
				v := persistVolumes
				params.PersistVolumes = &v
			}
			if cmd.Flags().Changed("var") && len(launchVars) > 0 {
				params.Variables = make(map[string]string)
				for _, v := range launchVars {
					parts := strings.SplitN(v, "=", 2)
					if len(parts) == 2 {
						key := normalizeVariableFlagKey(parts[0])
						if key != "" {
							params.Variables[key] = parts[1]
						}
					}
				}
			}
			if cmd.Flags().Changed("prop") && len(launchProps) > 0 {
				params.PropMappings = make(map[string]int64)
				params.PropMappingIdentifiers = make(map[string]string)
				for _, v := range launchProps {
					parts := strings.SplitN(v, "=", 2)
					if len(parts) == 2 {
						if pid, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
							params.PropMappings[parts[0]] = pid
						} else {
							params.PropMappingIdentifiers[parts[0]] = parts[1]
						}
					}
				}
			}

			if params.ComposeYAML == "" && len(rawPayload) > 0 {
				params.ComposeYAML = string(rawPayload)
			}
			if (len(args) > 0 || params.RepositoryURL != "") && params.ComposeYAML != "" {
				return fmt.Errorf("--compose cannot be combined with a GitHub repository argument")
			}

			repoRequest, err := resolveGitHubRepoRequest(cmd, c, args, githubRepoRequestOptions{
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
				return err
			}
			if repoRequest != nil {
				params.RepositoryURL = repoRequest.URL
				params.Name = repoRequest.Name
				params.GitHubRef = repoRequest.Ref
				params.ConfigPath = repoRequest.ConfigPath
				params.GitHubAccount = repoRequest.Account
				params.GitHubInstallationID = repoRequest.GitHubInstallationID
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.ComposeYAML == "" && params.RepositoryURL == "" {
				return fmt.Errorf("required field 'compose' not set")
			}
			if params.JobMode != nil && *params.JobMode && params.MarqueeID == nil && params.MarqueeIdentifier == "" {
				return fmt.Errorf("--job-mode requires --marquee-id (a trick has no marquee to run on otherwise)")
			}

			result, err := c.Launch.Create(ctx(), params)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name (optional with github-repo)")
	cmd.Flags().StringVar(&compose, "compose", "", "Docker-compose YAML (required unless github-repo is provided)")
	cmd.Flags().StringVar(&repoFile, "file", "", "Config file path inside the GitHub repository (optional; defaults to fibe.yml, fibe.yaml, docker-compose.yml, docker-compose.yaml)")
	cmd.Flags().StringVar(&repoRef, "ref", "", "Git branch, tag, or commit for the config file (optional)")
	cmd.Flags().StringVar(&githubAccount, "github-account", "", "GitHub App installation account owner to use when multiple installations are connected")
	cmd.Flags().Int64Var(&githubInstallationID, "github-installation-id", 0, "GitHub App installation ID to use when multiple installations are connected")
	cmd.Flags().BoolVar(&jobMode, "job-mode", false, "Create as a trick (job-mode) instead of a playground (requires --marquee-id)")
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Target marquee ID or name. Required when --job-mode is set; without it only the playspec is created.")
	cmd.Flags().BoolVar(&createPlayground, "create-playground", false, "Force playground creation. Defaults to true when --marquee-id is set, false otherwise.")
	cmd.Flags().BoolVar(&noCreatePlayground, "no-create-playground", false, "Skip playground deployment even when --marquee-id is set.")
	cmd.Flags().BoolVar(&persistVolumes, "persist-volumes", false, "Persist Docker volumes across trick/playground recreations")
	cmd.Flags().StringSliceVar(&launchVars, "var", nil, "Set template variables (e.g., --var subdomain=foo --var fibe_domain=foo.fibe.live)")
	cmd.Flags().StringSliceVar(&launchProps, "prop", nil, "Map private Git repository to Prop ID or name (e.g., --prop https://github.com/fibegg/fibe.git=my-prop)")
	return cmd
}
