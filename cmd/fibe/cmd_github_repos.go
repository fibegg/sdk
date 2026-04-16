package main

import (
	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func githubReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github-repos",
		Short: "Manage GitHub repositories",
		Long: `Create GitHub repositories via the Fibe API.

SUBCOMMANDS:
  create    Create a new GitHub repository`,
	}
	cmd.AddCommand(githubRepoCreateCmd())
	return cmd
}

func githubRepoCreateCmd() *cobra.Command {
	var name, description string
	var private, autoInit bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new GitHub repository",
		Long: `Create a new GitHub repository.

REQUIRED FLAGS:
  --name    Repository name

OPTIONAL FLAGS:
  --private      Create as private repository
  --auto-init    Initialize with README
  --description  Repository description

EXAMPLES:
  fibe github-repos create --name my-repo
  fibe github-repos create --name my-repo --private --auto-init --description "My project"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.GitHubRepoCreateParams{Name: name}
			if cmd.Flags().Changed("private") {
				params.Private = &private
			}
			if cmd.Flags().Changed("auto-init") {
				params.AutoInit = &autoInit
			}
			if cmd.Flags().Changed("description") {
				params.Description = &description
			}
			repo, err := c.GitHubRepos.Create(ctx(), params)
			if err != nil {
				return err
			}
			outputJSON(repo)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Repository name (required)")
	cmd.Flags().BoolVar(&private, "private", false, "Create as private")
	cmd.Flags().BoolVar(&autoInit, "auto-init", false, "Initialize with README")
	cmd.Flags().StringVar(&description, "description", "", "Repository description")
	return cmd
}
