package fibe

import "time"

type Prop struct {
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

	// Detail fields
	Branches              []string `json:"branches,omitempty"`
	OriginalRepositoryURL *string  `json:"original_repository_url,omitempty"`
	HasCredentials        *bool    `json:"has_credentials,omitempty"`
	DockerComposeYAML     *string  `json:"docker_compose_yaml,omitempty"`
}

type PropCreateParams struct {
	RepositoryURL     string         `json:"repository_url"`
	Name              *string        `json:"name,omitempty"`
	Private           *bool          `json:"private,omitempty"`
	DefaultBranch     *string        `json:"default_branch,omitempty"`
	Provider          *string        `json:"provider,omitempty"`
	DockerComposeYAML *string        `json:"docker_compose_yaml,omitempty"`
	Credentials       map[string]any `json:"credentials,omitempty"`
}

func (p *PropCreateParams) Validate() error {
	v := &validator{}
	v.required("repository_url", p.RepositoryURL)
	return v.err()
}

type PropUpdateParams struct {
	Name              *string        `json:"name,omitempty"`
	RepositoryURL     *string        `json:"repository_url,omitempty"`
	Private           *bool          `json:"private,omitempty"`
	DefaultBranch     *string        `json:"default_branch,omitempty"`
	Provider          *string        `json:"provider,omitempty"`
	DockerComposeYAML *string        `json:"docker_compose_yaml,omitempty"`
	Credentials       map[string]any `json:"credentials,omitempty"`
}

type PropBranches struct {
	Branches []PropBranch `json:"branches"`
}

// PropBranch is one entry in a PropBranches response. The API returns
// metadata (default marker, ahead/behind counts, last-commit details) in
// addition to the branch name, so PropBranch intentionally captures only
// the stable fields — anything the server adds later surfaces via the
// Extra map without breaking existing callers.
type PropBranch struct {
	Name    string         `json:"name"`
	Default bool           `json:"default"`
	Extra   map[string]any `json:"-"`
}

type PropEnvDefaults struct {
	Defaults map[string]string `json:"defaults"`
}

type PropListParams struct {
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
