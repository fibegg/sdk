package fibe

import "encoding/json"

// LaunchResult captures the outcome of the POST /api/launch endpoint
// (exposed as the fibe launch CLI command). The Rails controller returns
// the freshly-created playspec ID, playground ID, and any props that were
// implicitly registered as part of the compose import.
//
// Legacy fields (ID/Status/Name) are preserved for callers written against
// an earlier SDK version, but the API does not populate them — agents
// should read PlayspecID / PlaygroundID / PropsCreated instead.
type LaunchResult struct {
	PlayspecID   int64   `json:"playspecs_created,omitempty"`
	PlaygroundID int64   `json:"playground_id,omitempty"`
	PropsCreated []int64 `json:"props_created,omitempty"`

	// Legacy — kept for backwards compatibility; not populated today.
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
}

type LaunchParams struct {
	ComposeYAML            string            `json:"compose_yaml"`
	Name                   string            `json:"name"`
	JobMode                *bool             `json:"job_mode,omitempty"`
	MarqueeID              *int64            `json:"marquee_id,omitempty"`
	MarqueeIdentifier      string            `json:"-"`
	CreatePlayground       *bool             `json:"create_playground,omitempty"`
	Variables              map[string]string `json:"variables,omitempty"`
	PropMappings           map[string]int64  `json:"prop_mappings,omitempty"`
	PropMappingIdentifiers map[string]string `json:"-"`
}

func (p *LaunchParams) Validate() error {
	v := &validator{}
	v.required("compose_yaml", p.ComposeYAML)
	v.required("name", p.Name)
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
