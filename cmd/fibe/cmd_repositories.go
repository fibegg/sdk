package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func repositoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repositories",
		Aliases: []string{"repository", "repos"},
		Short:   "Manage repositories (linked repositories)",
		Long: `Manage Fibe repositories: linked Git repositories.

Repositories connect your GitHub or Gitea repositories to Fibe for automatic
syncing, branch tracking, and environment variable detection.

PROVIDERS:
  github    GitHub repository
  gitea     Gitea repository (self-hosted)

SUBCOMMANDS:
  list              List all repositories
  get <id-or-name> Show repository details
  create            Create a new repository
  update <id-or-name>       Update repository settings
  delete <id-or-name>       Delete a repository
  attach            Attach a GitHub repo by name
  mirror            Mirror a GitHub repo to Gitea
  sync <id-or-name>         Trigger repository sync
  branches <id-or-name>     List branches
  env-defaults <id-or-name> Get env defaults for a branch`,
	}

	cmd.AddCommand(
		repositoryListCmd(), repositoryGetCmd(), repositoryCreateCmd(), repositoryUpdateCmd(),
		repositoryDeleteCmd(), repositoryAttachCmd(), repositoryMirrorCmd(), repositorySyncCmd(),
		repositoryBranchesCmd(), repositoryEnvDefaultsCmd(),
	)
	return cmd
}

func repositoryListCmd() *cobra.Command {
	var query, status, provider, name, sort, createdAfter, createdBefore, private string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all repositories",
		Long: `List all repositories (repositories) accessible to the authenticated user.

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
  fibe repositories list
  fibe repos list -q "fibe" --status active
  fibe repositories list --provider github --sort name_asc
  fibe repositories list --private true -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.RepositoryListParams{}
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
			repositories, err := c.Repositories.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(repositories)
				return nil
			}
			headers := []string{"ID", "NAME", "URL", "PROVIDER", "STATUS", "SYNCED"}
			rows := make([][]string, len(repositories.Data))
			for i, p := range repositories.Data {
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

func repositoryGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Show repository details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			repository, err := c.Repositories.GetByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(repository)
				return nil
			}
			fmt.Printf("ID:       %d\n", repository.ID)
			fmt.Printf("Name:     %s\n", repository.Name)
			fmt.Printf("URL:      %s\n", repository.RepositoryURL)
			fmt.Printf("Provider: %s\n", repository.Provider)
			fmt.Printf("Branch:   %s\n", repository.DefaultBranch)
			fmt.Printf("Private:  %s\n", fmtBool(repository.Private))
			fmt.Printf("Status:   %s\n", repository.Status)
			fmt.Printf("Synced:   %s\n", fmtTime(repository.LastSyncedAt))
			return nil
		},
	}
}

func repositoryCreateCmd() *cobra.Command {
	var repoURL, name, defaultBranch, provider string
	var private bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new repository from a repository URL",
		Long: `Link a Git repository to Fibe.

NOTE ON PRIVATE REPOSITORIES:
  - Private GitHub repositories must be attached natively via the GitHub App (fibe repositories attach).
  - Private repositories hosted elsewhere CANNOT be publicly synced. They must first be mirrored via Fibe's internal Gitea (fibe repositories mirror).
  - Branch resolution logic: If source defaults to 'master', it will sync 'master'.

REQUIRED FLAGS:
  --url               Repository URL (HTTPS or SSH)

OPTIONAL FLAGS:
  --name              Display name (defaults to repo name)
  --private           Mark as private (default: false)
  --default-branch    Default branch name
  --provider          Provider: github or gitea

For 'credentials' (nested map) use --from-file with JSON.

EXAMPLES:
  fibe repositories create --url https://github.com/org/repo
  fibe repos create --url git@github.com:org/repo.git --name my-repo --default-branch main
  fibe repositories create --url https://gitea.example.com/org/repo --provider gitea --private` + generateSchemaDoc(&fibe.RepositoryCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.RepositoryCreateParams{}
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
			if params.RepositoryURL == "" {
				return fmt.Errorf("required field 'url' not set")
			}
			repository, err := c.Repositories.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(repository)
				return nil
			}
			fmt.Printf("Created repository %d (%s)\n", repository.ID, repository.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&repoURL, "url", "", "Repository URL (required)")
	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().BoolVar(&private, "private", false, "Mark as private")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "Default branch name")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider (github, gitea)")
	return cmd
}

func repositoryUpdateCmd() *cobra.Command {
	var name, repoURL, defaultBranch, provider string
	var private bool

	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update repository settings",
		Long: `Update an existing repository's internal settings.

OPTIONAL FLAGS:
  --name              New display name
  --url               New repository URL
  --private           Update private setting
  --default-branch    Update default branch
  --provider          Update provider (github, gitea)

EXAMPLES:
  fibe repositories update 5 --name renamed
  fibe repositories update 5 --default-branch main --private` + generateSchemaDoc(&fibe.RepositoryUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.RepositoryUpdateParams{}
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
			repository, err := c.Repositories.UpdateByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(repository)
				return nil
			}
			fmt.Printf("Updated repository %d (%s)\n", repository.ID, repository.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&repoURL, "url", "", "New repository URL")
	cmd.Flags().BoolVar(&private, "private", false, "Update private setting")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "Update default branch")
	cmd.Flags().StringVar(&provider, "provider", "", "Update provider (github, gitea)")
	return cmd
}

func repositoryDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if err := c.Repositories.DeleteByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Repository %s deleted\n", args[0])
			return nil
		},
	}
}

func repositoryAttachCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:   "attach",
		Short: "Attach a GitHub repo by full name",
		Long: `Attach an existing GitHub repository by its full name (owner/repo).

Requires GitHub App installation on the repository.

REQUIRED FLAGS:
  --repo   GitHub repo full name (e.g., org/repo)

EXAMPLES:
  fibe repositories attach --repo myorg/myrepo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			repository, err := c.Repositories.Attach(ctx(), repo)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(repository)
				return nil
			}
			fmt.Printf("Attached repository %d (%s)\n", repository.ID, repository.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo full name (required)")
	mustMarkFlagRequired(cmd, "repo")
	return cmd
}

func repositoryMirrorCmd() *cobra.Command {
	var sourceURL string

	cmd := &cobra.Command{
		Use:   "mirror",
		Short: "Mirror a GitHub repo to Gitea",
		Long: `Create a mirrored copy of a GitHub repository in the internal Gitea instance.

REQUIRED FLAGS:
  --url   GitHub repository URL

OPTIONAL FLAGS:
  --name  Name for the mirrored repository

EXAMPLES:
  fibe repositories mirror --url https://github.com/org/repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			nameFlag, _ := cmd.Flags().GetString("name")
			repository, err := c.Repositories.MirrorWithState(ctx(), sourceURL, nameFlag)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(repository)
				return nil
			}
			fmt.Printf("Mirroring started: repository %d (%s)\n", repository.ID, repository.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceURL, "url", "", "Source GitHub URL (required)")
	cmd.Flags().String("name", "", "Name for the mirrored repository (optional)")
	mustMarkFlagRequired(cmd, "url")
	return cmd
}

func repositorySyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <id-or-name>",
		Short: "Trigger repository sync",
		Long: `Trigger an immediate sync of the repository.

Fetches latest branches, commits, and file changes from the remote.

EXAMPLES:
  fibe repositories sync 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if err := c.Repositories.SyncByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(map[string]any{"ok": true, "repository": args[0], "message": "sync scheduled"})
				return nil
			}
			fmt.Printf("Sync scheduled for repository %s\n", args[0])
			return nil
		},
	}
}

func repositoryBranchesCmd() *cobra.Command {
	var query string
	var limit int

	cmd := &cobra.Command{
		Use:   "branches <id-or-name>",
		Short: "List repository branches",
		Long: `List branches of a linked repository.

OPTIONAL FLAGS:
  --query   Filter branches by name
  --limit   Max results (default: 20, max: 50)

EXAMPLES:
  fibe repositories branches 7
  fibe repos branches 7 --query feat`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			result, err := c.Repositories.BranchesByIdentifier(ctx(), args[0], query, limit)
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

func repositoryEnvDefaultsCmd() *cobra.Command {
	var branch, envFile string

	cmd := &cobra.Command{
		Use:   "env-defaults <id-or-name>",
		Short: "Get environment variable defaults for a branch",
		Long: `Extract default environment variables from a branch's .env file.

REQUIRED FLAGS:
  --branch   Branch name to read defaults from

OPTIONAL FLAGS:
  --env-file   Path to env file (default: .env)

EXAMPLES:
  fibe repositories env-defaults 7 --branch main
  fibe repos env-defaults 7 --branch develop --env-file .env.example`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			result, err := c.Repositories.EnvDefaultsByIdentifier(ctx(), args[0], branch, envFile)
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
	cmd.Flags().StringVar(&envFile, "env-file", ".env", "Env file path")
	mustMarkFlagRequired(cmd, "branch")
	return cmd
}
