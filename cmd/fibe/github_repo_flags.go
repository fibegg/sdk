package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

type githubRepoRequestOptions struct {
	ExistingURL            string
	ExistingName           string
	ExistingRef            string
	ExistingConfigPath     string
	ExistingAccount        string
	ExistingInstallationID *int64
	FlagRef                string
	FlagFile               string
	FlagAccount            string
	FlagInstallationID     int64
}

type githubRepoRequest struct {
	URL                  string
	Name                 string
	Ref                  string
	ConfigPath           string
	Account              string
	GitHubInstallationID *int64
}

func resolveGitHubRepoRequest(cmd *cobra.Command, c *fibe.Client, args []string, opts githubRepoRequestOptions) (*githubRepoRequest, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("expected at most one GitHub repository argument")
	}

	repository := opts.ExistingURL
	if len(args) == 1 {
		if repository != "" {
			return nil, fmt.Errorf("pass GitHub repository either as an argument or in --from-file payload, not both")
		}
		repository = args[0]
	}

	ref := opts.ExistingRef
	if cmd.Flags().Changed("ref") {
		ref = opts.FlagRef
	}
	configPath := opts.ExistingConfigPath
	if cmd.Flags().Changed("file") {
		configPath = opts.FlagFile
	}
	account := opts.ExistingAccount
	if cmd.Flags().Changed("github-account") {
		account = opts.FlagAccount
	}
	installationID := opts.ExistingInstallationID
	if cmd.Flags().Changed("github-installation-id") {
		if opts.FlagInstallationID <= 0 {
			return nil, fmt.Errorf("--github-installation-id must be a positive integer")
		}
		id := opts.FlagInstallationID
		installationID = &id
	}

	if repository == "" {
		if cmd.Flags().Changed("ref") || cmd.Flags().Changed("file") || cmd.Flags().Changed("github-account") || cmd.Flags().Changed("github-installation-id") {
			return nil, fmt.Errorf("--ref, --file, --github-account, and --github-installation-id require a GitHub repository argument or repository_url payload")
		}
		return nil, nil
	}

	source, err := fibe.ParseGitHubRepoSource(repository)
	if err != nil {
		return nil, err
	}
	if source.Ref != "" {
		if ref != "" {
			return nil, fmt.Errorf("pass ref either as owner/repo@ref or --ref, not both")
		}
		ref = source.Ref
	}

	selection := fibe.GitHubInstallationSelection{Account: account}
	if installationID != nil {
		selection.InstallationID = *installationID
	}
	installation, err := fibe.ResolveGitHubInstallation(ctx(), c, selection)
	if err != nil {
		return nil, err
	}
	resolvedInstallationID := installation.InstallationID

	name := opts.ExistingName
	if name == "" {
		name = fibe.InferNameFromGitHubRepoSource(source)
	}

	return &githubRepoRequest{
		URL:                  source.URL,
		Name:                 name,
		Ref:                  ref,
		ConfigPath:           configPath,
		Account:              account,
		GitHubInstallationID: &resolvedInstallationID,
	}, nil
}
