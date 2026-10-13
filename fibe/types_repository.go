package fibe

import "time"

type Repository struct {
	OwnershipMetadata
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	RepositoryURL string     `json:"repository_url"`
	Private       bool       `json:"private"`
	DefaultBranch string     `json:"default_branch"`
	Status        string     `json:"status"`
	Provider      string     `json:"provider"`
	LastSyncedAt  *time.Time `json:"last_synced_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	Branches              []string `json:"branches,omitempty"`
	OriginalRepositoryURL *string  `json:"original_repository_url,omitempty"`
	HasCredentials        *bool    `json:"has_credentials,omitempty"`
	ServiceNames          []string `json:"service_names,omitempty"`
}

type RepositoryCreateParams struct {
	RepositoryURL string         `json:"repository_url"`
	Name          *string        `json:"name,omitempty"`
	Private       *bool          `json:"private,omitempty"`
	DefaultBranch *string        `json:"default_branch,omitempty"`
	Provider      *string        `json:"provider,omitempty"`
	Credentials   map[string]any `json:"credentials,omitempty"`
}

func (p *RepositoryCreateParams) Validate() error {
	v := &validator{}
	v.required("repository_url", p.RepositoryURL)
	return v.err()
}

// RepositoryAttachParams selects an existing Repository in the current owner's installation.
type RepositoryAttachParams struct {
	RepoFullName         string `json:"repo_full_name"`
	GitHubInstallationID int64  `json:"github_installation_id,omitempty"`
}

func (p *RepositoryAttachParams) Validate() error {
	v := &validator{}
	v.required("repo_full_name", p.RepoFullName)
	if p.GitHubInstallationID < 0 {
		v.errors = append(v.errors, ValidationError{Field: "github_installation_id", Message: "must be positive when supplied"})
	}
	return v.err()
}

// RepositoryMirrorParams selects the source and, for Companies, its verified installation.
// Company imports require repositories:read and repositories:write for completion polling.
type RepositoryMirrorParams struct {
	SourceURL            string `json:"source_url"`
	Name                 string `json:"name,omitempty"`
	GitHubInstallationID int64  `json:"github_installation_id,omitempty"`
}

func (p *RepositoryMirrorParams) Validate() error {
	v := &validator{}
	v.required("source_url", p.SourceURL)
	if p.GitHubInstallationID < 0 {
		v.errors = append(v.errors, ValidationError{Field: "github_installation_id", Message: "must be positive when supplied"})
	}
	return v.err()
}

type RepositoryUpdateParams struct {
	Name          *string        `json:"name,omitempty"`
	RepositoryURL *string        `json:"repository_url,omitempty"`
	Private       *bool          `json:"private,omitempty"`
	DefaultBranch *string        `json:"default_branch,omitempty"`
	Provider      *string        `json:"provider,omitempty"`
	Credentials   map[string]any `json:"credentials,omitempty"`
}

type RepositoryBranches struct {
	Branches []RepositoryBranch `json:"branches"`
}

// RepositoryBranch is one entry in a RepositoryBranches response. The API returns
// metadata (default marker, ahead/behind counts, last-commit details) in
// addition to the branch name, so RepositoryBranch intentionally captures only
// the stable fields: anything the server adds later surfaces via the
// Extra map without breaking existing callers.
type RepositoryBranch struct {
	Name    string         `json:"name"`
	Default bool           `json:"default"`
	Extra   map[string]any `json:"-"`
}

type RepositoryEnvDefaults struct {
	Defaults map[string]string `json:"defaults"`
}

type RepositoryListParams struct {
	Q             string `url:"q,omitempty"`
	Status        string `url:"status,omitempty"`
	Provider      string `url:"provider,omitempty"`
	Name          string `url:"name,omitempty"`
	Private       *bool  `url:"private,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}
