package fibe

type GreenfieldCreateParams struct {
	Name         string         `json:"name"`
	TemplateID   *int64         `json:"template_id,omitempty"`
	Version      string         `json:"version,omitempty"`
	TemplateBody string         `json:"template_body,omitempty"`
	GitProvider  string         `json:"git_provider,omitempty"`
	MarqueeID    *int64         `json:"marquee_id,omitempty"`
	Variables    map[string]any `json:"variables,omitempty"`
}

func (p *GreenfieldCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	if p.Version != "" && p.TemplateID == nil {
		v.errors = append(v.errors, ValidationError{Field: "version", Message: "requires template_id"})
	}
	if p.TemplateBody != "" && (p.TemplateID != nil || p.Version != "") {
		v.errors = append(v.errors, ValidationError{Field: "template_body", Message: "cannot be combined with template_id or version"})
	}
	return v.err()
}

type GreenfieldResult struct {
	Name                    string                 `json:"name"`
	GitProvider             string                 `json:"git_provider"`
	SourceTemplateVersionID *int64                 `json:"source_template_version_id,omitempty"`
	Repo                    *GreenfieldRepo        `json:"repo,omitempty"`
	Repos                   []GreenfieldRepo       `json:"repos,omitempty"`
	Prop                    *Prop                  `json:"prop,omitempty"`
	Props                   []Prop                 `json:"props,omitempty"`
	ImportTemplate          *ImportTemplate        `json:"import_template,omitempty"`
	ImportTemplateVersion   *ImportTemplateVersion `json:"import_template_version,omitempty"`
	Playspec                *GreenfieldIDName      `json:"playspec,omitempty"`
	Playground              *Playground            `json:"playground,omitempty"`
	ServiceURLs             []GreenfieldServiceURL `json:"service_urls,omitempty"`
	Link                    *GreenfieldLinkResult  `json:"link,omitempty"`
}

type GreenfieldRepo struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	HTMLURL       string   `json:"html_url"`
	CloneURL      string   `json:"clone_url"`
	SSHURL        string   `json:"ssh_url"`
	Private       bool     `json:"private"`
	Description   string   `json:"description"`
	DefaultBranch string   `json:"default_branch"`
	Provider      string   `json:"provider"`
	RepositoryURL string   `json:"repository_url"`
	SourceRepoURL string   `json:"source_repo_url,omitempty"`
	ServiceNames  []string `json:"service_names,omitempty"`
}

type GreenfieldIDName struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type GreenfieldServiceURL struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	URL          string `json:"url"`
	Visibility   string `json:"visibility"`
	AuthRequired bool   `json:"auth_required"`
}

type GreenfieldLinkResult struct {
	LinkDir    string                 `json:"link_dir"`
	Playground string                 `json:"playground"`
	StateFile  string                 `json:"state_file"`
	Links      []GreenfieldLinkedPath `json:"links"`
}

type GreenfieldLinkedPath struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Target  string `json:"target"`
	Service string `json:"service,omitempty"`
	Prop    string `json:"prop,omitempty"`
	Branch  string `json:"branch,omitempty"`
}
