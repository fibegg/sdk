package mcpserver

import (
	"fmt"
	"sort"
	"strings"
)

var allToolTiers = []toolTier{
	tierMeta,
	tierBase,
	tierGreenfield,
	tierBrownfield,
	tierOverseer,
	tierLocal,
	tierOther,
}

var coreToolTiers = []toolTier{
	tierMeta,
	tierBase,
	tierGreenfield,
	tierBrownfield,
}

var toolTierNames = map[toolTier]string{
	tierMeta:       "meta",
	tierBase:       "base",
	tierGreenfield: "greenfield",
	tierBrownfield: "brownfield",
	tierOverseer:   "overseer",
	tierLocal:      "local",
	tierOther:      "other",
}

var toolTierByName = map[string]toolTier{
	"meta":       tierMeta,
	"base":       tierBase,
	"greenfield": tierGreenfield,
	"brownfield": tierBrownfield,
	"overseer":   tierOverseer,
	"local":      tierLocal,
	"other":      tierOther,
}

var confirmForwardingTools = map[string]bool{
	"fibe_call":     true,
	"fibe_pipeline": true,
}

// includeTool decides whether a tool should be advertised on the mcp-go
// server given the configured tool surface. Dispatcher registration is
// always unconditional so fibe_call and fibe_pipeline can still reach tools
// that are not currently advertised.
func (s *Server) includeTool(t *toolImpl) bool {
	if t.hidden {
		return false
	}
	tiers, err := parseToolTierSelection(s.cfg.ToolSet)
	if err != nil {
		return true
	}
	return tiers[t.tier]
}

func preservesConfirmArgs(name string) bool {
	return confirmForwardingTools[name]
}

func parseToolTierSelection(raw string) (map[toolTier]bool, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "full"
	}

	selected := map[toolTier]bool{}
	for _, part := range strings.Split(raw, ",") {
		token := normalizeToolTierToken(part)
		if token == "" {
			continue
		}
		switch token {
		case "full", "all":
			return tierSet(allToolTiers...), nil
		case "core":
			addTiers(selected, coreToolTiers...)
		default:
			tier, ok := toolTierByName[token]
			if !ok {
				return nil, fmt.Errorf("unknown MCP tool tier %q (valid: %s)", part, strings.Join(toolTierSelectionNames(), ", "))
			}
			selected[tier] = true
		}
	}
	if len(selected) == 0 {
		return tierSet(allToolTiers...), nil
	}
	return selected, nil
}

func toolTierSelectionNames() []string {
	names := []string{"core", "full", "all"}
	for _, tier := range allToolTiers {
		names = append(names, toolTierNames[tier])
	}
	sort.Strings(names)
	return names
}

func normalizeToolTierToken(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	token = strings.ReplaceAll(token, "-", "_")
	return token
}

func tierSet(tiers ...toolTier) map[toolTier]bool {
	out := map[toolTier]bool{}
	addTiers(out, tiers...)
	return out
}

func addTiers(out map[toolTier]bool, tiers ...toolTier) {
	for _, tier := range tiers {
		out[tier] = true
	}
}

func tierToString(t toolTier) string {
	if name, ok := toolTierNames[t]; ok {
		return name
	}
	return "unknown"
}
