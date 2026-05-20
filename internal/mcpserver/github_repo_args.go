package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
)

type mcpGitHubRepoRequest struct {
	URL                  string
	Name                 string
	Ref                  string
	ConfigPath           string
	Account              string
	GitHubInstallationID *int64
}

func resolveMCPGitHubRepoRequest(ctx context.Context, c *fibe.Client, args map[string]any, existingName string) (*mcpGitHubRepoRequest, error) {
	repository := firstStringArg(args, "repository_url", "repository", "repo")
	ref := firstStringArg(args, "github_ref", "ref")
	configPath := firstStringArg(args, "config_path", "file")
	account := argString(args, "github_account")
	installationID, hasInstallationID := argInt64(args, "github_installation_id")
	if hasInstallationID && installationID <= 0 {
		return nil, fmt.Errorf("github_installation_id must be a positive integer")
	}

	if repository == "" {
		if ref != "" || configPath != "" || account != "" || hasInstallationID {
			return nil, fmt.Errorf("github_ref, config_path, github_account, and github_installation_id require repository_url")
		}
		return nil, nil
	}
	if c == nil {
		return nil, fmt.Errorf("Fibe client is required to resolve GitHub App installations")
	}

	source, err := fibe.ParseGitHubRepoSource(repository)
	if err != nil {
		return nil, err
	}
	if source.Ref != "" {
		if ref != "" {
			return nil, fmt.Errorf("pass ref either as owner/repo@ref or github_ref, not both")
		}
		ref = source.Ref
	}

	selection := fibe.GitHubInstallationSelection{Account: account}
	if hasInstallationID {
		selection.InstallationID = installationID
	}
	installation, err := fibe.ResolveGitHubInstallation(ctx, c, selection)
	if err != nil {
		return nil, err
	}
	resolvedInstallationID := installation.InstallationID

	name := existingName
	if name == "" {
		name = fibe.InferNameFromGitHubRepoSource(source)
	}

	return &mcpGitHubRepoRequest{
		URL:                  source.URL,
		Name:                 name,
		Ref:                  ref,
		ConfigPath:           configPath,
		Account:              account,
		GitHubInstallationID: &resolvedInstallationID,
	}, nil
}

func firstStringArg(args map[string]any, keys ...string) string {
	for _, key := range keys {
		value := argString(args, key)
		if value != "" {
			return value
		}
	}
	return ""
}
