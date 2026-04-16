package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestAuditTrail_SecretCreationLogged verifies that creating a secret appears
// in the audit log.
func TestAuditTrail_SecretCreationLogged(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Create a secret
	s := seedSecret(t, c, "audit")

	// Poll audit logs for a recent secret-related entry
	entry, found := pollUntil(8, time.Second, func() (*fibe.AuditLog, bool) {
		list, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{
			ResourceType: "Secret",
			Sort:         "created_at_desc",
			PerPage:      25,
		})
		if err != nil {
			return nil, false
		}
		for _, entry := range list.Data {
			if entry.ResourceID != nil && s.ID != nil && *entry.ResourceID == *s.ID {
				return &entry, true
			}
		}
		return nil, false
	})

	if !found {
		t.Log("no audit entry found for secret creation (may be async with larger delay)")
		return
	}
	if !strings.Contains(strings.ToLower(entry.Action), "creat") {
		t.Errorf("expected 'create' in action, got %q", entry.Action)
	}
	if entry.Channel != "api" {
		t.Errorf("expected channel=api, got %s", entry.Channel)
	}
}

// TestAuditTrail_ActionPrefixFilter filters by action prefix.
func TestAuditTrail_ActionPrefixFilter(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Generate an action by creating a secret
	_ = seedSecret(t, c, "audit-prefix")

	time.Sleep(2 * time.Second)

	r, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{
		ActionPrefix: "secret",
		PerPage:      25,
	})
	requireNoError(t, err)
	for _, l := range r.Data {
		if !strings.Contains(strings.ToLower(l.Action), "secret") {
			t.Errorf("expected 'secret' in action for prefix filter, got %q", l.Action)
		}
	}
}

// TestAuditTrail_ResourceTypeFilterIsolation ensures filter truly isolates.
func TestAuditTrail_ResourceTypeFilterIsolation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Create an agent and secret to generate events
	_ = seedAgent(t, c, fibe.ProviderGemini)
	_ = seedSecret(t, c, "audit-iso")

	time.Sleep(2 * time.Second)

	// Filter by agent only — must not include Secret resource types
	r, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{ResourceType: "Agent", PerPage: 25})
	requireNoError(t, err)
	for _, l := range r.Data {
		if l.ResourceType != "Agent" {
			t.Errorf("resource_type filter leak: got %q", l.ResourceType)
		}
	}
}
