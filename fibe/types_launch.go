package fibe

import "encoding/json"

// LaunchResult contains the created spec, playground, and imported Repositories.
// Legacy fields remain for source compatibility but are not populated by the API.
type LaunchResult struct {
	SpecID              int64   `json:"spec_id,omitempty"`
	PlaygroundID        int64   `json:"playground_id,omitempty"`
	RepositoriesCreated []int64 `json:"repositories_created,omitempty"`

	// Legacy fields retained for compatibility.
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
}

func (r *LaunchResult) UnmarshalJSON(data []byte) error {
	type alias LaunchResult
	var raw struct {
		alias
		LegacySpecID int64 `json:"specs_created,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = LaunchResult(raw.alias)
	if r.SpecID == 0 {
		r.SpecID = raw.LegacySpecID
	}
	return nil
}

type LaunchParams struct {
	EnvPackAttachments           *[]EnvPackAttachmentInput `json:"env_pack_attachments,omitempty"`
	ComposeYAML                  string                    `json:"compose_yaml"`
	Name                         string                    `json:"name"`
	RepositoryURL                string                    `json:"repository_url,omitempty"`
	ConfigPath                   string                    `json:"config_path,omitempty"`
	GitHubRef                    string                    `json:"github_ref,omitempty"`
	GitHubInstallationID         *int64                    `json:"github_installation_id,omitempty"`
	GitHubAccount                string                    `json:"github_account,omitempty"`
	JobMode                      *bool                     `json:"job_mode,omitempty"`
	HostID                       *int64                    `json:"host_id,omitempty"`
	HostIdentifier               string                    `json:"-"`
	CreatePlayground             *bool                     `json:"create_playground,omitempty"`
	PersistVolumes               *bool                     `json:"persist_volumes,omitempty"`
	EnvOverrides                 map[string]string         `json:"env_overrides,omitempty"`
	ServiceSubdomains            map[string]string         `json:"service_subdomains,omitempty"`
	Services                     map[string]any            `json:"services,omitempty"`
	Variables                    map[string]string         `json:"variables,omitempty"`
	RepositoryMappings           map[string]int64          `json:"repository_mappings,omitempty"`
	RepositoryMappingIdentifiers map[string]string         `json:"-"`
}

func (p *LaunchParams) Validate() error {
	if err := (&EnvPackAttachmentsParams{Attachments: p.EnvPackAttachments}).Validate(); err != nil {
		return err
	}
	v := &validator{}
	if p.ComposeYAML == "" && p.RepositoryURL == "" {
		v.required("compose_yaml", p.ComposeYAML)
	}
	if p.Name == "" && p.RepositoryURL == "" {
		v.required("name", p.Name)
	}
	if p.ComposeYAML != "" && p.RepositoryURL != "" {
		v.errors = append(v.errors, ValidationError{Field: "repository_url", Message: "cannot be combined with compose_yaml"})
	}
	return v.err()
}

func (p LaunchParams) MarshalJSON() ([]byte, error) {
	type alias LaunchParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.HostIdentifier != "" {
		body["host_id"] = p.HostIdentifier
	}
	if len(p.RepositoryMappingIdentifiers) > 0 {
		mappings := map[string]any{}
		for k, v := range p.RepositoryMappings {
			mappings[k] = v
		}
		for k, v := range p.RepositoryMappingIdentifiers {
			mappings[k] = v
		}
		body["repository_mappings"] = mappings
	}
	return json.Marshal(body)
}
