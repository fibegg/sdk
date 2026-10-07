package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

func resolveLaunchHostIdentifier(c *fibe.Client, explicit string) (string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit, nil
	}
	if raw := strings.TrimSpace(os.Getenv("FIBE_HOST_ID")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return "", fmt.Errorf("FIBE_HOST_ID must be a positive integer")
		}
		return raw, nil
	}
	result, err := c.Hosts.List(ctx(), &fibe.HostListParams{PerPage: 100})
	if err != nil {
		return "", err
	}
	var candidates []fibe.Host
	for _, host := range result.Data {
		if host.ChatLaunchable || host.BillingRuntimeActive {
			candidates = append(candidates, host)
		}
	}
	switch len(candidates) {
	case 0:
		return "", fmt.Errorf("--host is required; no launchable Hosts are available")
	case 1:
		if candidates[0].Name != "" {
			return candidates[0].Name, nil
		}
		return strconv.FormatInt(candidates[0].ID, 10), nil
	default:
		return "", fmt.Errorf("--host is required; multiple launchable Hosts are available: %s", hostCandidateNames(candidates))
	}
}

func hostCandidateNames(candidates []fibe.Host) string {
	names := make([]string, 0, len(candidates))
	for _, host := range candidates {
		label := host.Name
		if label == "" {
			label = strconv.FormatInt(host.ID, 10)
		}
		names = append(names, label)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
