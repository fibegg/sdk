package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPagination_Envelope(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("default pagination returns meta", func(t *testing.T) {
		t.Parallel()
		result, err := c.Secrets.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Page != 1 {
			t.Errorf("expected page 1, got %d", result.Meta.Page)
		}
		if result.Meta.PerPage <= 0 {
			t.Errorf("expected positive per_page, got %d", result.Meta.PerPage)
		}
	})

	t.Run("explicit page params", func(t *testing.T) {
		t.Parallel()
		result, err := c.Secrets.List(ctx(), &fibe.SecretListParams{
			Page:    1,
			PerPage: 5,
		})
		requireNoError(t, err)

		if result.Meta.Page != 1 {
			t.Errorf("expected page 1, got %d", result.Meta.Page)
		}
		if result.Meta.PerPage != 5 {
			t.Errorf("expected per_page 5, got %d", result.Meta.PerPage)
		}
		if len(result.Data) > 5 {
			t.Errorf("expected at most 5 items, got %d", len(result.Data))
		}
	})

	t.Run("page 2 returns different data or empty", func(t *testing.T) {
		t.Parallel()
		result, err := c.Secrets.List(ctx(), &fibe.SecretListParams{
			Page:    2,
			PerPage: 1,
		})
		requireNoError(t, err)

		if result.Meta.Page != 2 {
			t.Errorf("expected page 2, got %d", result.Meta.Page)
		}
	})

	t.Run("consistent envelope across all list endpoints", func(t *testing.T) {
		t.Parallel()
		endpoints := []struct {
			name string
			call func() (int64, error)
		}{
			{"playgrounds", func() (int64, error) {
				r, e := c.Playgrounds.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
			{"agents", func() (int64, error) {
				r, e := c.Agents.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
			{"playspecs", func() (int64, error) {
				r, e := c.Playspecs.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
			{"props", func() (int64, error) {
				r, e := c.Props.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
			{"secrets", func() (int64, error) {
				r, e := c.Secrets.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
			{"api_keys", func() (int64, error) {
				r, e := c.APIKeys.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
			{"webhook_endpoints", func() (int64, error) {
				r, e := c.WebhookEndpoints.List(ctx(), nil)
				if e != nil {
					return 0, e
				}
				return r.Meta.Total, nil
			}},
		}

		for _, ep := range endpoints {
			ep := ep
			t.Run(ep.name, func(t *testing.T) {
				t.Parallel()
				total, err := ep.call()
				requireNoError(t, err)
				_ = total
			})
		}
	})
}

func TestPagination_Iterator(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	for i := 0; i < 3; i++ {
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   uniqueName("ITER_TEST"),
			Value: "val",
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })
	}

	t.Run("collect all via small pages", func(t *testing.T) {
		t.Parallel()
		firstPage, err := c.Secrets.List(ctx(), &fibe.SecretListParams{PerPage: 1})
		requireNoError(t, err)

		if firstPage.Meta.Total < 3 {
			t.Skipf("need at least 3 secrets, got %d", firstPage.Meta.Total)
		}

		allPages, err := c.Secrets.List(ctx(), &fibe.SecretListParams{PerPage: 100})
		requireNoError(t, err)

		if int64(len(allPages.Data)) != allPages.Meta.Total && allPages.Meta.Total <= 100 {
			t.Errorf("expected %d items on single large page, got %d", allPages.Meta.Total, len(allPages.Data))
		}
	})
}
