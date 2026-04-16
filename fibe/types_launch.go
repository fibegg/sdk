package fibe

// LaunchResult captures the outcome of the POST /api/launch endpoint
// (exposed as fibe launch / fibe_launch). The Rails controller returns
// the freshly-created playspec ID, playground ID, and any props that were
// implicitly registered as part of the compose import.
//
// Legacy fields (ID/Status/Name) are preserved for callers written against
// an earlier SDK version, but the API does not populate them — agents
// should read PlayspecID / PlaygroundID / PropsCreated instead.
type LaunchResult struct {
	PlayspecID    int64   `json:"playspecs_created,omitempty"`
	PlaygroundID  int64   `json:"playground_id,omitempty"`
	PropsCreated  []int64 `json:"props_created,omitempty"`

	// Legacy — kept for backwards compatibility; not populated today.
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
}

type LaunchParams struct {
	ComposeYAML      string `json:"compose_yaml"`
	Name             string `json:"name"`
	JobMode          *bool  `json:"job_mode,omitempty"`
	MarqueeID        *int64 `json:"marquee_id,omitempty"`
	CreatePlayground *bool  `json:"create_playground,omitempty"`
}


func (p *LaunchParams) Validate() error {
	v := &validator{}
	v.required("compose_yaml", p.ComposeYAML)
	v.required("name", p.Name)
	return v.err()
}

