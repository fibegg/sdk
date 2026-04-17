package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func propsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "props",
		Aliases: []string{"repos"},
		Short:   "Manage props (linked repositories)",
		Long: `Manage Fibe props — linked Git repositories.

Props connect your GitHub or Gitea repositories to Fibe for automatic
syncing, branch tracking, and environment variable detection.

PROVIDERS:
  github    GitHub repository
  gitea     Gitea repository (self-hosted)

SUBCOMMANDS:
  list              List all props
  get <id>          Show prop details
  create            Create a new prop
  update <id>       Update prop settings
  delete <id>       Delete a prop
  attach            Attach a GitHub repo by name
  mirror            Mirror a GitHub repo to Gitea
  sync <id>         Trigger repository sync
  branches <id>     List branches
  env-defaults <id> Get env defaults for a branch
  with-compose      List props that have docker-compose`,
	}

	cmd.AddCommand(
		propListCmd(), propGetCmd(), propCreateCmd(), propUpdateCmd(),
		propDeleteCmd(), propAttachCmd(), propMirrorCmd(), propSyncCmd(),
		propBranchesCmd(), propEnvDefaultsCmd(), propWithComposeCmd(),
	)
	return cmd
}

func propListCmd() *cobra.Command {
	var query, status, provider, name, sort, createdAfter, createdBefore, private string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all props",
		Long: `List all props (repositories) accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name, repository_url (substring match)
  --status              Filter by exact status. Values: active, syncing, error
  --provider            Filter by provider. Values: github, gitea
  --name                Filter by name (substring match)
  --private             Filter by visibility. Values: true, false

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, NAME, URL, PROVIDER, STATUS, SYNCED
  Use --output json for full details.

EXAMPLES:
  fibe props list
  fibe repos list -q "fibe" --status active
  fibe props list --provider github --sort name_asc
  fibe props list --private true -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PropListParams{}
			if query != "" {
				params.Q = query
			}
			if status != "" {
				params.Status = status
			}
			if provider != "" {
				params.Provider = provider
			}
			if name != "" {
				params.Name = name
			}
			if private == "true" {
				t := true
				params.Private = &t
			} else if private == "false" {
				f := false
				params.Private = &f
			}
			if createdAfter != "" {
				params.CreatedAfter = createdAfter
			}
			if createdBefore != "" {
				params.CreatedBefore = createdBefore
			}
			if sort != "" {
				params.Sort = sort
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			props, err := c.Props.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(props)
				return nil
			}
			headers := []string{"ID", "NAME", "URL", "PROVIDER", "STATUS", "SYNCED"}
			rows := make([][]string, len(props.Data))
			for i, p := range props.Data {
				rows[i] = []string{
					fmtInt64(p.ID), p.Name, p.RepositoryURL,
					p.Provider, p.Status, fmtTime(p.LastSyncedAt),
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name, repository URL")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider (github, gitea)")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&private, "private", "", "Filter by visibility (true/false)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func propGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show prop details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			prop, err := c.Props.Get(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(prop)
				return nil
			}
			fmt.Printf("ID:       %d\n", prop.ID)
			fmt.Printf("Name:     %s\n", prop.Name)
			fmt.Printf("URL:      %s\n", prop.RepositoryURL)
			fmt.Printf("Provider: %s\n", prop.Provider)
			fmt.Printf("Branch:   %s\n", prop.DefaultBranch)
			fmt.Printf("Private:  %s\n", fmtBool(prop.Private))
			fmt.Printf("Status:   %s\n", prop.Status)
			fmt.Printf("Synced:   %s\n", fmtTime(prop.LastSyncedAt))
			return nil
		},
	}
}

func propCreateCmd() *cobra.Command {
	var repoURL, name, defaultBranch, provider, dockerCompose string
	var private bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new prop from a repository URL",
		Long: `Link a Git repository to Fibe.

NOTE ON PRIVATE REPOSITORIES:
  - Private GitHub repositories must be attached natively via the GitHub App (fibe props attach).
  - Private repositories hosted elsewhere CANNOT be publicly synced. They must first be mirrored via Fibe's internal Gitea (fibe props mirror).
  - Branch resolution logic: If source defaults to 'master', it will sync 'master'.

REQUIRED FLAGS:
  --url               Repository URL (HTTPS or SSH)

OPTIONAL FLAGS:
  --name              Display name (defaults to repo name)
  --private           Mark as private (default: false)
  --default-branch    Default branch name
  --provider          Provider: github or gitea
  --docker-compose    Inline docker-compose YAML (use @file.yml to read from disk)

For 'credentials' (nested map) use --from-file with JSON.

EXAMPLES:
  fibe props create --url https://github.com/org/repo
  fibe repos create --url git@github.com:org/repo.git --name my-repo --default-branch main
  fibe props create --url https://gitea.example.com/org/repo --provider gitea --private` + generateSchemaDoc(&fibe.PropCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PropCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("url") {
				params.RepositoryURL = repoURL
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("private") {
				params.Private = &private
			}
			if cmd.Flags().Changed("default-branch") {
				params.DefaultBranch = &defaultBranch
			}
			if cmd.Flags().Changed("provider") {
				params.Provider = &provider
			}
			if cmd.Flags().Changed("docker-compose") {
				v := resolveStringValue(dockerCompose)
				params.DockerComposeYAML = &v
			}
			if params.RepositoryURL == "" {
				return fmt.Errorf("required field 'url' not set")
			}
			prop, err := c.Props.Create(ctx(), params)
			if err != nil {
				return err
			}
			fmt.Printf("Created prop %d (%s)\n", prop.ID, prop.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoURL, "url", "", "Repository URL (required)")
	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().BoolVar(&private, "private", false, "Mark as private")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "Default branch name")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider (github, gitea)")
	cmd.Flags().StringVar(&dockerCompose, "docker-compose", "", "Inline docker-compose YAML (use @file)")
	return cmd
}

func propUpdateCmd() *cobra.Command {
	var name, repoURL, defaultBranch, provider, dockerCompose string
	var private bool

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update prop settings",
		Long: `Update an existing prop's internal settings.

OPTIONAL FLAGS:
  --name              New display name
  --url               New repository URL
  --private           Update private setting
  --default-branch    Update default branch
  --provider          Update provider (github, gitea)
  --docker-compose    Update inline compose YAML (use @file)

EXAMPLES:
  fibe props update 5 --name renamed
  fibe props update 5 --default-branch main --private` + generateSchemaDoc(&fibe.PropUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.PropUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("url") {
				params.RepositoryURL = &repoURL
			}
			if cmd.Flags().Changed("private") {
				params.Private = &private
			}
			if cmd.Flags().Changed("default-branch") {
				params.DefaultBranch = &defaultBranch
			}
			if cmd.Flags().Changed("provider") {
				params.Provider = &provider
			}
			if cmd.Flags().Changed("docker-compose") {
				v := resolveStringValue(dockerCompose)
				params.DockerComposeYAML = &v
			}
			prop, err := c.Props.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			fmt.Printf("Updated prop %d (%s)\n", prop.ID, prop.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&repoURL, "url", "", "New repository URL")
	cmd.Flags().BoolVar(&private, "private", false, "Update private setting")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "Update default branch")
	cmd.Flags().StringVar(&provider, "provider", "", "Update provider (github, gitea)")
	cmd.Flags().StringVar(&dockerCompose, "docker-compose", "", "Update docker-compose YAML (use @file)")
	return cmd
}

func propDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a prop",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Props.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Prop %d deleted\n", id)
			return nil
		},
	}
}

func propAttachCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:   "attach",
		Short: "Attach a GitHub repo by full name",
		Long: `Attach an existing GitHub repository by its full name (owner/repo).

Requires GitHub App installation on the repository.

REQUIRED FLAGS:
  --repo   GitHub repo full name (e.g., org/repo)

EXAMPLES:
  fibe props attach --repo myorg/myrepo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			prop, err := c.Props.Attach(ctx(), repo)
			if err != nil {
				return err
			}
			fmt.Printf("Attached prop %d (%s)\n", prop.ID, prop.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo full name (required)")
	cmd.MarkFlagRequired("repo")
	return cmd
}

func propMirrorCmd() *cobra.Command {
	var sourceURL string

	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Mirror a GitHub repo to Gitea",
		Long: `Create a mirrored copy of a GitHub repository in the internal Gitea instance.

REQUIRED FLAGS:
  --url   GitHub repository URL

EXAMPLES:
  fibe props mirror --url https://github.com/org/repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			prop, err := c.Props.Mirror(ctx(), sourceURL)
			if err != nil {
				return err
			}
			fmt.Printf("Mirroring started — prop %d (%s)\n", prop.ID, prop.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceURL, "url", "", "Source GitHub URL (required)")
	cmd.MarkFlagRequired("url")
	return cmd
}

func propSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <id>",
		Short: "Trigger repository sync",
		Long: `Trigger an immediate sync of the repository.

Fetches latest branches, commits, and file changes from the remote.

EXAMPLES:
  fibe props sync 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Props.Sync(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Sync scheduled for prop %d\n", id)
			return nil
		},
	}
}

func propBranchesCmd() *cobra.Command {
	var query string
	var limit int

	cmd := &cobra.Command{
		Use:   "branches <id>",
		Short: "List repository branches",
		Long: `List branches of a linked repository.

OPTIONAL FLAGS:
  --query   Filter branches by name
  --limit   Max results (default: 20, max: 50)

EXAMPLES:
  fibe props branches 7
  fibe repos branches 7 --query feat`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			result, err := c.Props.Branches(ctx(), id, query, limit)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			for _, b := range result.Branches {
				if b.Default {
					fmt.Printf("* %s\n", b.Name)
				} else {
					fmt.Printf("  %s\n", b.Name)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max results")
	return cmd
}

func propEnvDefaultsCmd() *cobra.Command {
	var branch, envFile string

	cmd := &cobra.Command{
		Use:   "env-defaults <id>",
		Short: "Get environment variable defaults for a branch",
		Long: `Extract default environment variables from a branch's .env file.

REQUIRED FLAGS:
  --branch   Branch name to read defaults from

OPTIONAL FLAGS:
  --env-file   Path to env file (default: .env)

EXAMPLES:
  fibe props env-defaults 7 --branch main
  fibe repos env-defaults 7 --branch develop --env-file .env.example`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			result, err := c.Props.EnvDefaults(ctx(), id, branch, envFile)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			for k, v := range result.Defaults {
				fmt.Printf("%s=%s\n", k, v)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "Branch name (required)")
	cmd.Flags().StringVar(&envFile, "env-file", "", "Env file path")
	cmd.MarkFlagRequired("branch")
	return cmd
}

func propWithComposeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "with-compose",
		Short: "List props that have docker-compose files",
		Long: `List all props that have a docker-compose.yml detected in their repository.

EXAMPLES:
  fibe props with-compose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			props, err := c.Props.WithDockerCompose(ctx(), nil)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(props)
				return nil
			}
			headers := []string{"ID", "NAME", "URL", "PROVIDER"}
			rows := make([][]string, len(props.Data))
			for i, p := range props.Data {
				rows[i] = []string{fmtInt64(p.ID), p.Name, p.RepositoryURL, p.Provider}
			}
			outputTable(headers, rows)
			return nil
		},
	}
}
