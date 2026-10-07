package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

func resolveMCPHost(ctx context.Context, c *fibe.Client, args map[string]any) (*int64, string, error) {
	id, identifier := explicitMCPHost(args)
	if id != nil || identifier != "" {
		return id, identifier, nil
	}

	if envID, err := parseHostIDEnv(); err == nil {
		return &envID, "", nil
	} else if !strings.Contains(err.Error(), "FIBE_HOST_ID is not set") {
		return nil, "", err
	}

	if c == nil {
		return nil, "", fmt.Errorf("host_id is required when FIBE_HOST_ID is not set")
	}
	result, err := c.Hosts.List(ctx, &fibe.HostListParams{PerPage: 100})
	if err != nil {
		return nil, "", err
	}
	var candidates []fibe.Host
	for _, host := range result.Data {
		if host.ChatLaunchable || host.BillingRuntimeActive {
			candidates = append(candidates, host)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, "", fmt.Errorf("host_id is required; no launchable Hosts are available")
	case 1:
		id := candidates[0].ID
		return &id, "", nil
	default:
		return nil, "", fmt.Errorf("host_id is required; multiple launchable Hosts are available: %s", mcpHostCandidateNames(candidates))
	}
}

func explicitMCPHost(args map[string]any) (*int64, string) {
	for _, key := range []string{"host_id_or_name", "host_id"} {
		if id, ok := argInt64(args, key); ok && id > 0 {
			return &id, ""
		}
		if identifier := strings.TrimSpace(argString(args, key)); identifier != "" {
			return nil, identifier
		}
	}
	return nil, ""
}

func mcpHostCandidateNames(candidates []fibe.Host) string {
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
