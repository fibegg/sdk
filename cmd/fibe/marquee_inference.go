package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

func resolveLaunchMarqueeIdentifier(c *fibe.Client, explicit string) (string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit, nil
	}
	if raw := strings.TrimSpace(os.Getenv("FIBE_MARQUEE_ID")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return "", fmt.Errorf("FIBE_MARQUEE_ID must be a positive integer")
		}
		return raw, nil
	}
	result, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{PerPage: 100})
	if err != nil {
		return "", err
	}
	var candidates []fibe.Marquee
	for _, marquee := range result.Data {
		if marquee.ChatLaunchable || marquee.BillingRuntimeActive {
			candidates = append(candidates, marquee)
		}
	}
	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("--marquee is required; no launchable Marquees are available")
	case 1:
		if candidates[0].Name != "" {
			return candidates[0].Name, nil
		}
		return strconv.FormatInt(candidates[0].ID, 10), nil
	default:
		return "", fmt.Errorf("--marquee is required; multiple launchable Marquees are available: %s", marqueeCandidateNames(candidates))
	}
}

func marqueeCandidateNames(candidates []fibe.Marquee) string {
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
