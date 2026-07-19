package localplaygrounds

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSourcePlanValidationMatchesTheCoreManifestBoundary(t *testing.T) {
	production := false
	pg := &Playground{
		Path: "/opt/fibe/playgrounds/demo",
		Services: map[string]*Service{
			"app": {Name: "app"},
		},
	}
	valid := sourcePlanManifest{
		Version: 1,
		Sources: []sourcePlanEntry{{
			Repository:   "localhost/acme/app",
			Branch:       "feature/topic",
			RelativePath: "props/acme--app--0123456789/feature-topic--0123456789",
			Services: []sourcePlanMember{{
				Service:     "app",
				Production:  &production,
				MountTarget: stringPointer("/workspace"),
			}},
		}},
	}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	if err := applySourcePlan(pg, data); err != nil {
		t.Fatalf("valid Core source plan rejected: %v", err)
	}

	tests := []struct {
		name     string
		manifest string
		want     string
	}{
		{
			name:     "path outside props",
			manifest: `{"version":1,"sources":[{"repository":"github.com/acme/app","branch":"main","relative_path":"other/app","services":[{"service":"app","production":false,"mount_target":"/app"}]}]}`,
			want:     "contained props path",
		},
		{
			name:     "invalid exact branch",
			manifest: `{"version":1,"sources":[{"repository":"github.com/acme/app","branch":"feature//topic","relative_path":"props/app/branch","services":[{"service":"app","production":false,"mount_target":"/app"}]}]}`,
			want:     "invalid repository or branch",
		},
		{
			name:     "duplicate logical checkout",
			manifest: `{"version":1,"sources":[{"repository":"github.com/acme/app","branch":"main","relative_path":"props/app/first","services":[{"service":"app","production":false,"mount_target":"/app"}]},{"repository":"github.com/acme/app","branch":"main","relative_path":"props/app/second","services":[{"service":"app","production":false,"mount_target":"/app"}]}]}`,
			want:     "appear more than once",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate := &Playground{Path: pg.Path, Services: map[string]*Service{"app": {Name: "app"}}}
			err := applySourcePlan(candidate, []byte(tt.manifest))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v want %q", err, tt.want)
			}
		})
	}
}

func TestSourcePlanAcceptsCoreRepositoryIdentityWithNonDefaultPort(t *testing.T) {
	production := false
	pg := &Playground{
		Path: "/opt/fibe/playgrounds/demo",
		Services: map[string]*Service{
			"app": {Name: "app"},
		},
	}
	manifest := sourcePlanManifest{
		Version: 1,
		Sources: []sourcePlanEntry{{
			Repository:   "git.example.test:8443/acme/app",
			Branch:       "main",
			RelativePath: "props/acme--app--0123456789/main--0123456789",
			Services: []sourcePlanMember{{
				Service:     "app",
				Production:  &production,
				MountTarget: stringPointer("/workspace"),
			}},
		}},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := applySourcePlan(pg, data); err != nil {
		t.Fatalf("Core non-default-port identity rejected: %v", err)
	}
	if pg.Services["app"].Repository != "git.example.test:8443/acme/app" {
		t.Fatalf("repository=%q", pg.Services["app"].Repository)
	}
}

func stringPointer(value string) *string {
	return &value
}
