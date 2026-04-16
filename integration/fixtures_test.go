package integration

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// realComposeYAML returns a syntactically-valid multi-service docker-compose
// that exercises real features the backend can parse.
func realComposeYAML() string {
	return `services:
  web:
    image: nginx:alpine
    ports:
      - "80"
    environment:
      NGINX_PORT: "80"
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: app
  cache:
    image: redis:7-alpine
`
}

// minimalComposeYAML returns a single-service compose (for speed).
func minimalComposeYAML() string {
	return "services:\n  web:\n    image: nginx:alpine\n"
}

// uniqueHost returns a fake but unique IP address for marquee tests.
// The backend enforces host uniqueness, and many tests run in parallel,
// so we derive an address from the unique name counter.
func uniqueHost() string {
	n := nameCounter.Add(1)
	ts := time.Now().UnixNano() & 0xFF
	return fmt.Sprintf("10.%d.%d.%d", (n>>16)&0xFF, (n>>8)&0xFF, (n&0xFF)^ts)
}

// jobComposeYAML returns a compose suitable for job-mode playspecs (tricks).
func jobComposeYAML() string {
	return `services:
  worker:
    image: alpine:3
    command: ["sh", "-c", "echo done"]
`
}

// jobWatchedService returns a service def with JobWatch=true, which is required
// for backends that enforce "job_mode requires at least one watched service".
func jobWatchedService(name string) fibe.PlayspecServiceDef {
	t := true
	return fibe.PlayspecServiceDef{
		Name:     name,
		Type:     fibe.ServiceTypeStatic,
		JobWatch: &t,
	}
}

// seedPlayspec creates a real playspec, registers cleanup, returns ID.
func seedPlayspec(t *testing.T, c *fibe.Client, opts ...func(*fibe.PlayspecCreateParams)) *fibe.Playspec {
	t.Helper()
	params := &fibe.PlayspecCreateParams{
		Name:            uniqueName("fx-spec"),
		BaseComposeYAML: realComposeYAML(),
		Services: []fibe.PlayspecServiceDef{
			{Name: "web", Type: fibe.ServiceTypeStatic},
			{Name: "db", Type: fibe.ServiceTypeStatic},
			{Name: "cache", Type: fibe.ServiceTypeStatic},
		},
	}
	for _, o := range opts {
		o(params)
	}
	spec, err := c.Playspecs.Create(ctx(), params)
	requireNoError(t, err, "seed playspec")
	if spec.ID == nil || *spec.ID == 0 {
		t.Fatal("expected playspec ID")
	}
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })
	return spec
}

// seedAgent creates an agent and registers cleanup.
func seedAgent(t *testing.T, c *fibe.Client, provider string) *fibe.Agent {
	t.Helper()
	if provider == "" {
		provider = fibe.ProviderGemini
	}
	ag, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("fx-agent"),
		Provider: provider,
	})
	requireNoError(t, err, "seed agent")
	t.Cleanup(func() { c.Agents.Delete(ctx(), ag.ID) })
	return ag
}

// seedSecret creates a secret and registers cleanup.
func seedSecret(t *testing.T, c *fibe.Client, keySuffix string) *fibe.Secret {
	t.Helper()
	key := fmt.Sprintf("FX_%s_%d", keySuffix, time.Now().UnixNano())
	s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
		Key:   key,
		Value: "fx-value-" + keySuffix,
	})
	requireNoError(t, err, "seed secret")
	t.Cleanup(func() {
		if s.ID != nil {
			c.Secrets.Delete(ctx(), *s.ID)
		}
	})
	return s
}

// seedTeam creates a team (skips test if teams feature disabled) and registers cleanup.
func seedTeam(t *testing.T, c *fibe.Client) *fibe.Team {
	t.Helper()
	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("fx-team"),
	})
	if err != nil {
		if apiErr, ok := err.(*fibe.APIError); ok && apiErr.Code == fibe.ErrCodeFeatureDisabled {
			t.Skip("teams feature disabled for this account")
		}
		requireNoError(t, err, "seed team")
	}
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })
	return team
}

// skipIfFeatureDisabled returns true (and skips) if the error is FEATURE_DISABLED.
func skipIfFeatureDisabled(t *testing.T, err error, feature string) bool {
	t.Helper()
	if err == nil {
		return false
	}
	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		return false
	}
	if apiErr.Code == fibe.ErrCodeFeatureDisabled {
		t.Skipf("%s feature disabled: %s", feature, apiErr.Message)
		return true
	}
	return false
}

// requireSortedByString asserts slice is sorted ascending or descending.
// Uses Go's byte-wise comparison, which differs from PostgreSQL's default locale-aware
// collation (which ignores punctuation). Callers that sort against PG should use
// requireSortedByStringLocaleAware which normalizes punctuation out.
func requireSortedByString(t *testing.T, label string, values []string, ascending bool) {
	t.Helper()
	if len(values) < 2 {
		return
	}
	check := sort.SliceIsSorted(values, func(i, j int) bool {
		if ascending {
			return values[i] < values[j]
		}
		return values[i] > values[j]
	})
	if !check {
		dir := "descending"
		if ascending {
			dir = "ascending"
		}
		t.Errorf("%s: expected %s order, got %v", label, dir, values)
	}
}

// localeNormalize strips punctuation & casefolds — approximates PostgreSQL's
// default collation treatment of punctuation.
func localeNormalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// requireSortedByStringLocaleAware asserts ordering using a locale-aware
// approximation (strip punctuation, casefold). Matches PostgreSQL default behavior.
func requireSortedByStringLocaleAware(t *testing.T, label string, values []string, ascending bool) {
	t.Helper()
	if len(values) < 2 {
		return
	}
	normalized := make([]string, len(values))
	for i, v := range values {
		normalized[i] = localeNormalize(v)
	}
	check := sort.SliceIsSorted(normalized, func(i, j int) bool {
		if ascending {
			return normalized[i] < normalized[j]
		}
		return normalized[i] > normalized[j]
	})
	if !check {
		dir := "descending"
		if ascending {
			dir = "ascending"
		}
		t.Errorf("%s: expected %s locale-aware order, got raw=%v normalized=%v", label, dir, values, normalized)
	}
}

// requireSortedByTime asserts slice is sorted ascending or descending by time.
func requireSortedByTime(t *testing.T, label string, values []time.Time, ascending bool) {
	t.Helper()
	if len(values) < 2 {
		return
	}
	for i := 1; i < len(values); i++ {
		if ascending && values[i-1].After(values[i]) {
			t.Errorf("%s: expected ascending order at index %d: %v > %v", label, i, values[i-1], values[i])
			return
		}
		if !ascending && values[i-1].Before(values[i]) {
			t.Errorf("%s: expected descending order at index %d: %v < %v", label, i, values[i-1], values[i])
			return
		}
	}
}
