package fibe

type AgentDefaults map[string]any

type AgentDefaultsPayload struct {
	AgentDefaults AgentDefaults `json:"agent_defaults"`
	Player        *Player       `json:"player,omitempty"`
}
