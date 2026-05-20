package fibe

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var githubRepoNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type GitHubRepoSource struct {
	Owner    string
	Repo     string
	FullName string
	URL      string
	Ref      string
}

type GitHubInstallationSelection struct {
	Account        string
	InstallationID int64
}

func ParseGitHubRepoSource(input string) (*GitHubRepoSource, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("github repository is required")
	}
	if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "git@") {
		return parseGitHubRepoShortSource(raw)
	}
	if strings.HasPrefix(raw, "git@") {
		return parseGitHubRepoSSHSource(raw)
	}
	return parseGitHubRepoURLSource(raw)
}

func InferNameFromGitHubRepoSource(source *GitHubRepoSource) string {
	if source == nil {
		return ""
	}
	name := strings.ToLower(source.Repo)
	name = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-._")
	return name
}

func ResolveGitHubInstallation(ctx context.Context, c *Client, selection GitHubInstallationSelection) (*Installation, error) {
	result, err := c.Installations.List(ctx)
	if err != nil {
		return nil, err
	}
	installations := result.Data
	if selection.InstallationID > 0 {
		for _, inst := range installations {
			if inst.InstallationID == selection.InstallationID || inst.ID == selection.InstallationID {
				return &inst, nil
			}
		}
		return nil, fmt.Errorf("GitHub App installation %d is not connected to this Fibe account", selection.InstallationID)
	}
	if selection.Account != "" {
		matches := make([]Installation, 0, 1)
		for _, inst := range installations {
			if inst.InstallationAccount != nil && strings.EqualFold(*inst.InstallationAccount, selection.Account) {
				matches = append(matches, inst)
			}
		}
		if len(matches) == 1 {
			return &matches[0], nil
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("multiple GitHub App installations match account %q; pass --github-installation-id. Matching installations: %s", selection.Account, installationAccountList(matches))
		}
		return nil, fmt.Errorf("no GitHub App installation is connected for account %q; connected accounts: %s", selection.Account, installationAccountList(installations))
	}
	switch len(installations) {
	case 0:
		return nil, fmt.Errorf("no GitHub App installation is connected; run `fibe github apps connect` and finish setup in the browser")
	case 1:
		return &installations[0], nil
	default:
		return nil, fmt.Errorf("multiple GitHub App installations are connected; pass --github-account or --github-installation-id. Connected accounts: %s", installationAccountList(installations))
	}
}

func parseGitHubRepoShortSource(raw string) (*GitHubRepoSource, error) {
	repoSpec := raw
	ref := ""
	if before, after, ok := strings.Cut(raw, "@"); ok {
		repoSpec = before
		ref = after
		if ref == "" {
			return nil, fmt.Errorf("github repository ref cannot be blank")
		}
	}
	parts := strings.Split(repoSpec, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("github repository must be owner/repo or https://github.com/owner/repo")
	}
	return newGitHubRepoSource(parts[0], strings.TrimSuffix(parts[1], ".git"), ref)
}

func parseGitHubRepoURLSource(raw string) (*GitHubRepoSource, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub repository URL: %w", err)
	}
	if u.Scheme != "https" || !strings.EqualFold(u.Host, "github.com") {
		return nil, fmt.Errorf("only https://github.com/owner/repo URLs are supported")
	}
	pathParts := strings.Split(strings.Trim(strings.TrimSuffix(u.Path, ".git"), "/"), "/")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("GitHub repository URL must point to https://github.com/owner/repo")
	}
	return newGitHubRepoSource(pathParts[0], pathParts[1], "")
}

func parseGitHubRepoSSHSource(raw string) (*GitHubRepoSource, error) {
	if !strings.HasPrefix(raw, "git@github.com:") {
		return nil, fmt.Errorf("only GitHub repositories are supported")
	}
	path := strings.TrimSuffix(strings.TrimPrefix(raw, "git@github.com:"), ".git")
	pathParts := strings.Split(path, "/")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("GitHub SSH repository must be git@github.com:owner/repo.git")
	}
	return newGitHubRepoSource(pathParts[0], pathParts[1], "")
}

func newGitHubRepoSource(owner, repo, ref string) (*GitHubRepoSource, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" || !githubRepoNamePattern.MatchString(owner) || !githubRepoNamePattern.MatchString(repo) {
		return nil, fmt.Errorf("invalid GitHub repository name %q/%q", owner, repo)
	}
	fullName := owner + "/" + repo
	return &GitHubRepoSource{
		Owner:    owner,
		Repo:     repo,
		FullName: fullName,
		URL:      "https://github.com/" + fullName,
		Ref:      ref,
	}, nil
}

func installationAccountList(installations []Installation) string {
	items := make([]string, 0, len(installations))
	for _, inst := range installations {
		account := "unknown"
		if inst.InstallationAccount != nil && *inst.InstallationAccount != "" {
			account = *inst.InstallationAccount
		}
		items = append(items, fmt.Sprintf("%s(%s)", account, strconv.FormatInt(inst.InstallationID, 10)))
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}
