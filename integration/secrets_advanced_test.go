package integration

import (
	"fmt"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 43-secrets-crud.spec.js
func TestSecrets_Pagination(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prefix := uniqueName("PAGE_TEST")
	var ids []int64
	for i := 0; i < 3; i++ {
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   fmt.Sprintf("%s_%d", prefix, i),
			Value: "val",
		})
		requireNoError(t, err)
		ids = append(ids, *s.ID)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			c.Secrets.Delete(ctx(), id)
		}
	})

	t.Run("page 1 with per_page=1", func(t *testing.T) {
		t.Parallel()
		result, err := c.Secrets.List(ctx(), &fibe.SecretListParams{Key: prefix, Page: 1, PerPage: 1})
		requireNoError(t, err)

		if len(result.Data) != 1 {
			t.Errorf("expected 1 item, got %d", len(result.Data))
		}
		if result.Meta.Total < 3 {
			t.Errorf("expected total >= 3, got %d", result.Meta.Total)
		}
	})

	t.Run("page 2 with per_page=1 returns different item", func(t *testing.T) {
		t.Parallel()
		r1, _ := c.Secrets.List(ctx(), &fibe.SecretListParams{Key: prefix, Page: 1, PerPage: 1})
		r2, _ := c.Secrets.List(ctx(), &fibe.SecretListParams{Key: prefix, Page: 2, PerPage: 1})

		if len(r1.Data) > 0 && len(r2.Data) > 0 {
			if r1.Data[0].ID != nil && r2.Data[0].ID != nil && *r1.Data[0].ID == *r2.Data[0].ID {
				t.Error("page 1 and page 2 should return different items")
			}
		}
	})
}

// Migrated from: 43-secrets-crud.spec.js
func TestSecrets_EncryptionVerification(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	value := "super-secret-password-123!@#"
	s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
		Key:   uniqueName("ENCRYPT_TEST"),
		Value: value,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })

	t.Run("value readable via get", func(t *testing.T) {
		// Parallel disabled: updates run concurrently will mutate
		got, err := c.Secrets.Get(ctx(), *s.ID)
		requireNoError(t, err)

		if got.Value == nil || *got.Value != value {
			t.Errorf("expected value %q, got %v", value, got.Value)
		}
	})

	t.Run("update preserves encryption", func(t *testing.T) {
		// Parallel disabled: sequential dependency
		newVal := "updated-secret-456"
		_, err := c.Secrets.Update(ctx(), *s.ID, &fibe.SecretUpdateParams{
			Value: &newVal,
		})
		requireNoError(t, err)

		got, err := c.Secrets.Get(ctx(), *s.ID)
		requireNoError(t, err)
		if got.Value == nil || *got.Value != newVal {
			t.Error("updated value should be readable")
		}
	})
}

// Migrated from: 44-audit-logs-api.spec.js
func TestAuditLogs_AfterSecretOperations(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
		Key:   uniqueName("AUDIT_TEST"),
		Value: "audited",
	})
	requireNoError(t, err)
	c.Secrets.Delete(ctx(), *s.ID)

	t.Run("audit logs contain API channel entries", func(t *testing.T) {
		t.Parallel()
		result, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{
			Channel: "api",
		})
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected audit log entries from API operations")
		}

		for _, log := range result.Data {
			if log.Channel != "api" {
				t.Errorf("expected channel 'api', got %q", log.Channel)
			}
		}
	})
}
