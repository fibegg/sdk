package mcpserver

import (
	"context"
	"testing"

	"github.com/fibegg/sdk/internal/testedition"
	"github.com/stretchr/testify/require"
)

func TestMCPEditionLiveSurface(t *testing.T) {
	key, domain := requireRealServer(t)
	cfg := DefaultConfig()
	cfg.APIKey = key
	cfg.Domain = domain
	cfg.ToolSet = "core"
	s := New(cfg)
	require.NoError(t, s.RegisterAll())
	s.baseCli = s.buildBaseClient()
	ctx := context.Background()
	t.Cleanup(s.startEditionDiscovery(ctx))

	names := editionListedNames(t, s, ctx)
	require.Contains(t, names, "fibe_status")
	require.Contains(t, names, "fibe_resource_list")
	community := s.sessionManifest(ctx) != nil
	switch testedition.Current() {
	case testedition.Community:
		require.True(t, community, "the live Community route manifest must be discovered")
	case testedition.Enterprise:
		require.False(t, community, "the live Enterprise surface must remain complete")
	}
	if community {
		require.NotContains(t, names, "fibe_templates_search")
		_, err := s.dispatcher.dispatch(ctx, "fibe_call", map[string]any{"tool": "fibe_templates_search"})
		require.EqualError(t, err, "templates search is not available: this server runs the Community edition.")
	} else {
		require.Contains(t, names, "fibe_templates_search")
	}
	editionDispatch(t, s, "fibe_resource_list", map[string]any{"resource": "agents"})
}
