package restcontract

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRESTContractIsCurrent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	got, err := Generate(filepath.Join(root, "fibe"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "contracts", "rest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("REST contract is stale; run go run ./internal/cmd/restcontract")
	}
}
