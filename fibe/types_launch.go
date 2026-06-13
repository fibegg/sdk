package fibe

import "encoding/json"

// LaunchResult captures the outcome of the POST /api/launches endpoint
// (exposed as the fibe launch CLI command). The Fibe API returns
// the freshly-created playspec ID, playground ID, and any props that were
// implicitly registered as part of the compose import.
//
// Legacy fields (ID/Status/Name) are preserved for callers written against
// an earlier SDK version, but the API does not populate them — agents
// should read PlayspecID / PlaygroundID / PropsCreated instead.
type LaunchResult struct {
	PlayspecID   int64   `json:"playspec_id,omitempty"`
	PlaygroundID int64   `json:"playground_id,omitempty"`
	PropsCreated []int64 `json:"props_created,omitempty"`

	// Legacy — kept for backwards compatibility; not populated today.
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
}

func (r *LaunchResult) UnmarshalJSON(data []byte) error {
	type alias LaunchResult
	var raw struct {
		alias
		LegacyPlayspecID int64 `json:"playspecs_created,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = LaunchResult(raw.alias)
	if r.PlayspecID == 0 {
		r.PlayspecID = raw.LegacyPlayspecID
	}
	return nil
}

type LaunchParams struct {
	ComposeYAML            string            `json:"compose_yaml"`
	Name                   string            `json:"name"`
	RepositoryURL          string            `json:"repository_url,omitempty"`
	ConfigPath             string            `json:"config_path,omitempty"`
	GitHubRef              string            `json:"github_ref,omitempty"`
	GitHubInstallationID   *int64            `json:"github_installation_id,omitempty"`
	GitHubAccount          string            `json:"github_account,omitempty"`
	JobMode                *bool             `json:"job_mode,omitempty"`
	MarqueeID              *int64            `json:"marquee_id,omitempty"`
	MarqueeIdentifier      string            `json:"-"`
	CreatePlayground       *bool             `json:"create_playground,omitempty"`
	PersistVolumes         *bool             `json:"persist_volumes,omitempty"`
	EnvOverrides           map[string]string `json:"env_overrides,omitempty"`
	ServiceSubdomains      map[string]string `json:"service_subdomains,omitempty"`
	Services               map[string]any    `json:"services,omitempty"`
	Variables              map[string]string `json:"variables,omitempty"`
	PropMappings           map[string]int64  `json:"prop_mappings,omitempty"`
	PropMappingIdentifiers map[string]string `json:"-"`
}

func (p *LaunchParams) Validate() error {
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
	if p.MarqueeIdentifier != "" {
		body["marquee_id"] = p.MarqueeIdentifier
	}
	if len(p.PropMappingIdentifiers) > 0 {
		mappings := map[string]any{}
		for k, v := range p.PropMappings {
			mappings[k] = v
		}
		for k, v := range p.PropMappingIdentifiers {
			mappings[k] = v
		}
		body["prop_mappings"] = mappings
	}
	return json.Marshal(body)
}
