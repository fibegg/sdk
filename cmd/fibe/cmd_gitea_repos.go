package main

import (
	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func giteaReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitea-repos",
		Short: "Manage Gitea repositories",
		Long: `Create Gitea repositories via the Fibe API and register them as Props.

SUBCOMMANDS:
  create    Create a new Gitea repository and Prop`,
	}
	cmd.AddCommand(giteaRepoCreateCmd())
	return cmd
}

func giteaRepoCreateCmd() *cobra.Command {
	var name, description string
	var private, autoInit bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Gitea repository and Prop",
		Long: `Create a new Gitea repository and register it as a Fibe Prop.

REQUIRED FLAGS:
  --name    Repository name

OPTIONAL FLAGS:
  --private     Create as private repository
  --auto-init   Initialize with README
  --description Repository description

EXAMPLES:
  fibe gitea-repos create --name my-repo
  fibe gitea-repos create --name my-repo --private --auto-init --description "My project"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.GiteaRepoCreateParams{Name: name}
			if cmd.Flags().Changed("private") {
				params.Private = &private
			}
			if cmd.Flags().Changed("auto-init") {
				params.AutoInit = &autoInit
			}
			if cmd.Flags().Changed("description") {
				params.Description = &description
			}
			repo, err := c.GiteaRepos.Create(ctx(), params)
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
