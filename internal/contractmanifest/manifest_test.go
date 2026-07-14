package contractmanifest

import (
	"bytes"
	"os"
	"path/filepath"
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

func TestV0245PublicAPIIsCompatible(t *testing.T) {
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
	if problems := CompatibilityErrors(baseline, current); len(problems) > 0 {
		t.Fatalf("v0.2.45 compatibility failures:\n%s", strings.Join(problems, "\n"))
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
