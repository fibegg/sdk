package fibe

// RepoStatus is the result of checking a GitHub repo's status.
type RepoStatus struct {
	Repos []RepoStatusEntry `json:"repos"`
}

type RepoStatusEntry struct {
	URL                  string `json:"url"`
	Status               string `json:"status"`
	Error                string `json:"error,omitempty"`
	PropID               int64  `json:"prop_id,omitempty"`
	PropName             string `json:"prop_name,omitempty"`
	GitHubURL            string `json:"github_url,omitempty"`
	PlayerRepo           string `json:"player_repo,omitempty"`
	PlayerRepoURL        string `json:"player_repo_url,omitempty"`
	DefaultBranch        string `json:"default_branch,omitempty"`
	ForkURL              string `json:"fork_url,omitempty"`
	SourceFullName       string `json:"source_full_name,omitempty"`
	Mirrorable           bool   `json:"mirrorable,omitempty"`
	RuntimeWritable      *bool  `json:"runtime_writable,omitempty"`
	RuntimeAccessSource  string `json:"runtime_access_source,omitempty"`
	RuntimeAccessMessage string `json:"runtime_access_message,omitempty"`
	GitHubAppInstalled   bool   `json:"github_app_installed,omitempty"`
	OAuthAccessible      bool   `json:"oauth_accessible,omitempty"`
	RequiresFork         bool   `json:"requires_fork,omitempty"`
}

// GitHubRepo is the result of creating a new GitHub repo.
type GitHubRepo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

type GitHubRepoCreateParams struct {
	Name        string  `json:"name"`
	Private     *bool   `json:"private,omitempty"`
	AutoInit    *bool   `json:"auto_init,omitempty"`
	Description *string `json:"description,omitempty"`
}

// GiteaRepo is the result of creating a new Gitea repo.
type GiteaRepo struct {
	ID            int64             `json:"id"`
	Name          string            `json:"name"`
	FullName      string            `json:"full_name"`
	HTMLURL       string            `json:"html_url"`
	CloneURL      string            `json:"clone_url"`
	SSHURL        string            `json:"ssh_url"`
	Private       bool              `json:"private"`
	Description   string            `json:"description"`
	DefaultBranch string            `json:"default_branch,omitempty"`
	Repo          *GiteaRepoSummary `json:"repo,omitempty"`
	PropID        int64             `json:"prop_id,omitempty"`
	Prop          *Prop             `json:"prop,omitempty"`
}

type GiteaRepoSummary struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Private       bool   `json:"private"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

type GiteaRepoCreateParams struct {
	Name        string  `json:"name"`
	Private     *bool   `json:"private,omitempty"`
	AutoInit    *bool   `json:"auto_init,omitempty"`
	Description *string `json:"description,omitempty"`
}
