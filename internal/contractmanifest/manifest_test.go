package contractmanifest

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestCurrentPublicAPIManifestIsExact(t *testing.T) {
	root := repositoryRoot(t)
	generated, err := Generate(filepath.Join(root, "fibe"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "contracts", "go-public-api.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, want) {
		t.Fatal("public Go API manifest is stale; run go run ./internal/cmd/apimanifest")
	}
}

func TestV03OnlyChangesApprovedDomainNames(t *testing.T) {
	root := repositoryRoot(t)
	baseline, err := Read(filepath.Join(root, "contracts", "go-public-api-v0.2.45.json"))
	if err != nil {
		t.Fatal(err)
	}
	currentData, err := Generate(filepath.Join(root, "fibe"))
	if err != nil {
		t.Fatal(err)
	}
	currentPath := filepath.Join(t.TempDir(), "current.json")
	if err := os.WriteFile(currentPath, currentData, 0o600); err != nil {
		t.Fatal(err)
	}
	current, err := Read(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	// The v0.2.45 artifact remains immutable. Its declarations are translated
	// only for the approved v0.3 domain break; all other public API stays checked.
	for i := range baseline.Declarations {
		baseline.Declarations[i].ID = canonicalDomainDeclaration(baseline.Declarations[i].ID)
		baseline.Declarations[i].Signature = strings.Join(strings.Fields(canonicalDomainDeclaration(baseline.Declarations[i].Signature)), " ")
	}
	for i := range current.Declarations {
		current.Declarations[i].Signature = strings.Join(strings.Fields(current.Declarations[i].Signature), " ")
	}
	if problems := CompatibilityErrors(baseline, current); len(problems) > 0 {
		t.Fatalf("unexpected changes outside the v0.3 domain rename:\n%s", strings.Join(problems, "\n"))
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func canonicalDomainDeclaration(value string) string {
	replacements := strings.NewReplacer(
		"Marquees", "Hosts", "Marquee", "Host", "marquees", "hosts", "marquee", "host",
		"Playspecs", "Specs", "Playspec", "Spec", "playspecs", "specs", "playspec", "spec",
		"Tricks", "Tasks", "Trick", "Task", "tricks", "tasks", "trick", "task",
		"MARQUEE", "HOST", "PLAYSPEC", "SPEC", "PROP_", "REPOSITORY_", "TRICK", "TASK",
	)
	value = replacements.Replace(value)
	value = strings.ReplaceAll(value, "prop_", "repository_")
	value = strings.ReplaceAll(value, "props_", "repositories_")
	value = strings.ReplaceAll(value, "_props", "_repositories")
	value = strings.ReplaceAll(value, "_prop", "_repository")
	value = regexp.MustCompile(`Props([A-Z]|\b)`).ReplaceAllString(value, `Repositories${1}`)
	value = regexp.MustCompile(`Prop([A-Z]|\b)`).ReplaceAllString(value, `Repository${1}`)
	value = regexp.MustCompile(`(^|_)prop_`).ReplaceAllString(value, `${1}repository_`)
	value = regexp.MustCompile(`\bprops\b`).ReplaceAllString(value, "repositories")
	value = regexp.MustCompile(`\bprop\b`).ReplaceAllString(value, "repository")
	value = regexp.MustCompile(`\bprop([A-Z])`).ReplaceAllString(value, `repository${1}`)
	return value
}
