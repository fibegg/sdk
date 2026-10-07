package domainnames

import (
	"strings"
	"testing"
)

func TestRemovedFieldsReportCanonicalReplacements(t *testing.T) {
	for old, replacement := range map[string]string{"marquee_id": "host_id", "playspec_id": "spec_id", "source_prop_id": "source_repository_id", "prop_mappings": "repository_mappings", "marqueeRoot": "hostRoot", "marqueeRootDomain": "hostRootDomain"} {
		t.Run(old, func(t *testing.T) {
			err := RejectFields(map[string]any{"payload": map[string]any{old: 1}})
			if err == nil || !strings.Contains(err.Error(), replacement) {
				t.Fatalf("got %v; want replacement %s", err, replacement)
			}
		})
	}
}

func TestApplicationPropertiesRemainOpaque(t *testing.T) {
	value := map[string]any{"host_id": 1, "repository_id": 2, "metadata": map[string]any{"props": "react", "marquee_id": "application-owned"}, "env_vars": map[string]any{"prop": "application-owned"}}
	if err := RejectFields(value); err != nil {
		t.Fatal(err)
	}
}

func TestComposeEnvironmentPreservesApplicationKeysButServicesRejectRemovedFields(t *testing.T) {
	value := map[string]any{"services": []any{map[string]any{"environment": map[string]any{"prop_id": "application-owned"}}}}
	if err := RejectFields(value); err != nil {
		t.Fatal(err)
	}

	value = map[string]any{"services": []any{map[string]any{"prop_id": 24}}}
	if err := RejectFields(value); err == nil || !strings.Contains(err.Error(), "repository_id") {
		t.Fatalf("got %v; want structural field replacement repository_id", err)
	}
}

func TestRemovedEnvironmentReportsReplacement(t *testing.T) {
	t.Setenv("MARQUEE_ROOT", t.TempDir())
	err := CheckEnvironment()
	if err == nil || !strings.Contains(err.Error(), "HOST_ROOT") {
		t.Fatalf("got %v", err)
	}
}
