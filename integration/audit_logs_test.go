package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestAuditLogs_List(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("list all audit logs", func(t *testing.T) {
		t.Parallel()
		result, err := c.AuditLogs.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Skip("no audit logs present")
		}

		for _, log := range result.Data {
			if log.Action == "" {
				t.Error("expected action")
			}
			if log.Channel == "" {
				t.Error("expected channel")
			}
		}
	})

	t.Run("filter by channel", func(t *testing.T) {
		t.Parallel()
		result, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{
			Channel: "api",
		})
		requireNoError(t, err)

		for _, log := range result.Data {
			if log.Channel != "api" {
				t.Errorf("expected channel 'api', got %q", log.Channel)
			}
		}
	})

	t.Run("filter by resource type", func(t *testing.T) {
		t.Parallel()
		result, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{
			ResourceType: "Playground",
		})
		requireNoError(t, err)

		for _, log := range result.Data {
			if log.ResourceType != "Playground" {
				t.Errorf("expected resource_type 'Playground', got %q", log.ResourceType)
			}
		}
	})

	t.Run("scope enforcement", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "no-audit", []string{"agents:read"})
		_, err := noScope.AuditLogs.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
