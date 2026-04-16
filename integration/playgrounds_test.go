package integration

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

var (
	testMarqueeOnce sync.Once
	testMarquee     int64
	testMarqueeErr  error
)

func testMarqueeID(t *testing.T) int64 {
	t.Helper()
	testMarqueeOnce.Do(func() {
		c := adminClient(t)

		if v := os.Getenv("FIBE_TEST_MARQUEE_ID"); v != "" {
			id, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				testMarqueeErr = fmt.Errorf("invalid FIBE_TEST_MARQUEE_ID: %w", err)
				return
			}
			if _, err := c.Marquees.Get(ctx(), id); err == nil {
				testMarquee = id
				return
			}
		}

		result, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{
			Status:  "active",
			Sort:    "created_at_asc",
			PerPage: 100,
		})
		if err != nil {
			testMarqueeErr = fmt.Errorf("discover marquee: %w", err)
			return
		}
		if len(result.Data) == 0 {
			return
		}
		testMarquee = result.Data[0].ID
	})

	if testMarqueeErr != nil {
		t.Fatalf("%v", testMarqueeErr)
	}
	return testMarquee
}

func setupPlaygroundDeps(t *testing.T, c *fibe.Client) (specID, marqueeID int64) {
	t.Helper()
	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            uniqueName("pg-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err, "create playspec for playground")
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	return *spec.ID, testMarqueeID(t)
}

func TestPlaygrounds_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	var pgID int64

	t.Run("create playground", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		if marqueeID == 0 {
			t.Skip("set FIBE_TEST_MARQUEE_ID to test playground creation")
		}
		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("test-pg"),
			PlayspecID: specID,
			MarqueeID:  &marqueeID,
		})
		requireNoError(t, err)

		pgID = pg.ID
		if pg.Status == "" {
			t.Error("expected status")
		}
	})
	t.Cleanup(func() {
		if pgID > 0 {
			c.Playgrounds.Delete(ctx(), pgID)
		}
	})

	t.Run("list playgrounds", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playgrounds.List(ctx(), nil)
		requireNoError(t, err)
		if result.Meta.Page == 0 {
			t.Error("expected meta.page > 0")
		}
	})

	t.Run("get playground detail", func(t *testing.T) {
		t.Parallel()
		if pgID == 0 {
			t.Skip("no playground created")
		}
		pg, err := c.Playgrounds.Get(ctx(), pgID)
		requireNoError(t, err)

		if pg.ID != pgID {
			t.Errorf("expected ID %d, got %d", pgID, pg.ID)
		}
	})

	t.Run("get status", func(t *testing.T) {
		t.Parallel()
		if pgID == 0 {
			t.Skip("no playground created")
		}
		status, err := c.Playgrounds.Status(ctx(), pgID)
		requireNoError(t, err)

		if status.Status == "" {
			t.Error("expected status string")
		}
	})

	t.Run("update playground", func(t *testing.T) {
		t.Parallel()
		if pgID == 0 {
			t.Skip("no playground created")
		}
		newName := uniqueName("updated-pg")
		pg, err := c.Playgrounds.Update(ctx(), pgID, &fibe.PlaygroundUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)

		if pg.Name != newName {
			t.Errorf("expected name %q, got %q", newName, pg.Name)
		}
	})

	t.Run("get nonexistent playground", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Get(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestPlaygrounds_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("no scope returns 403", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "no-pg", []string{"agents:read"})
		_, err := noScope.Playgrounds.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key can list", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-read", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.List(ctx(), nil)
		requireNoError(t, err)
	})

	t.Run("read key cannot create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-read2", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       "nope",
			PlayspecID: 999,
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
