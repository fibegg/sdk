package fibe

import "encoding/json"

type GreenfieldCreateParams struct {
	Name                 string            `json:"name"`
	TemplateID           *int64            `json:"template_id,omitempty"`
	TemplateIdentifier   string            `json:"-"`
	TemplateVersionID    *int64            `json:"template_version_id,omitempty"`
	Version              string            `json:"version,omitempty"`
	TemplateBody         string            `json:"template_body,omitempty"`
	RepositoryURL        string            `json:"repository_url,omitempty"`
	ConfigPath           string            `json:"config_path,omitempty"`
	GitHubRef            string            `json:"github_ref,omitempty"`
	GitHubInstallationID *int64            `json:"github_installation_id,omitempty"`
	GitHubAccount        string            `json:"github_account,omitempty"`
	GitProvider          string            `json:"git_provider,omitempty"`
	Private              *bool             `json:"private,omitempty"`
	MarqueeID            *int64            `json:"marquee_id,omitempty"`
	MarqueeIdentifier    string            `json:"-"`
	PersistVolumes       *bool             `json:"persist_volumes,omitempty"`
	Variables            map[string]any    `json:"variables,omitempty"`
	EnvOverrides         map[string]string `json:"env_overrides,omitempty"`
	ServiceSubdomains    map[string]string `json:"service_subdomains,omitempty"`
	Services             map[string]any    `json:"services,omitempty"`
}

func (p *GreenfieldCreateParams) Validate() error {
	v := &validator{}
	if p.Name == "" && p.RepositoryURL == "" {
		v.required("name", p.Name)
	}
	if p.Version != "" && p.TemplateID == nil && p.TemplateIdentifier == "" {
		v.errors = append(v.errors, ValidationError{Field: "version", Message: "requires template_id_or_name"})
	}
	if p.TemplateVersionID != nil && (p.TemplateID != nil || p.TemplateIdentifier != "" || p.Version != "") {
		v.errors = append(v.errors, ValidationError{Field: "template_version_id", Message: "cannot be combined with template_id_or_name or version"})
	}
	if p.TemplateBody != "" && (p.TemplateID != nil || p.TemplateIdentifier != "" || p.TemplateVersionID != nil || p.Version != "") {
		v.errors = append(v.errors, ValidationError{Field: "template_body", Message: "cannot be combined with template_id_or_name, template_version_id, or version"})
	}
	if p.RepositoryURL != "" && (p.TemplateBody != "" || p.TemplateID != nil || p.TemplateIdentifier != "" || p.TemplateVersionID != nil || p.Version != "") {
		v.errors = append(v.errors, ValidationError{Field: "repository_url", Message: "cannot be combined with template_body, template_id_or_name, template_version_id, or version"})
	}
	return v.err()
}

func (p GreenfieldCreateParams) MarshalJSON() ([]byte, error) {
	type alias GreenfieldCreateParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.MarqueeIdentifier != "" {
		body["marquee_id"] = p.MarqueeIdentifier
	}
	if p.TemplateIdentifier != "" {
		body["template_id"] = p.TemplateIdentifier
	}
	return json.Marshal(body)
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
