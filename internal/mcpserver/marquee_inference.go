package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

func resolveMCPMarquee(ctx context.Context, c *fibe.Client, args map[string]any) (*int64, string, error) {
	id, identifier := explicitMCPMarquee(args)
	if id != nil || identifier != "" {
		return id, identifier, nil
	}

	if envID, err := parseMarqueeIDEnv(); err == nil {
		return &envID, "", nil
	} else if !strings.Contains(err.Error(), "FIBE_MARQUEE_ID is not set") {
		return nil, "", err
	}

	if c == nil {
		return nil, "", fmt.Errorf("marquee_id is required when FIBE_MARQUEE_ID is not set")
	}
	result, err := c.Marquees.List(ctx, &fibe.MarqueeListParams{PerPage: 100})
	if err != nil {
		return nil, "", err
	}
	var candidates []fibe.Marquee
	for _, marquee := range result.Data {
		if marquee.ChatLaunchable || marquee.BillingRuntimeActive {
			candidates = append(candidates, marquee)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, "", fmt.Errorf("marquee_id is required; no launchable Marquees are available")
	case 1:
		id := candidates[0].ID
		return &id, "", nil
	default:
		return nil, "", fmt.Errorf("marquee_id is required; multiple launchable Marquees are available: %s", mcpMarqueeCandidateNames(candidates))
	}
}

func explicitMCPMarquee(args map[string]any) (*int64, string) {
	for _, key := range []string{"marquee_id_or_name", "marquee_id"} {
		if id, ok := argInt64(args, key); ok && id > 0 {
			return &id, ""
		}
		if identifier := strings.TrimSpace(argString(args, key)); identifier != "" {
			return nil, identifier
		}
	}
	return nil, ""
}

func mcpMarqueeCandidateNames(candidates []fibe.Marquee) string {
	names := make([]string, 0, len(candidates))
	for _, marquee := range candidates {
		label := marquee.Name
		if label == "" {
			label = strconv.FormatInt(marquee.ID, 10)
		}
		names = append(names, label)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
