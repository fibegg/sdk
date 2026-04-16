package fibe

// RepoStatus is the result of checking a GitHub repo's status.
type RepoStatus struct {
	Repos []RepoStatusEntry `json:"repos"`
}

type RepoStatusEntry struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
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
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	Private     bool   `json:"private"`
	Description string `json:"description"`
}

type GiteaRepoCreateParams struct {
	Name        string  `json:"name"`
	Private     *bool   `json:"private,omitempty"`
	AutoInit    *bool   `json:"auto_init,omitempty"`
	Description *string `json:"description,omitempty"`
}

