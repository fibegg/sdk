package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMCPConfigWriteIsAtomicAndPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "mcp.json")
	const writers = 16
	var group sync.WaitGroup
	for i := range writers {
		group.Add(1)
		go func() {
			defer group.Done()
			data, _ := json.Marshal(map[string]int{"writer": i})
			if err := writeMCPConfigFile(path, data); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	group.Wait()
	data, err := readMCPConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]int
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("partial config: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}

func TestMCPConfigRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := writeMCPConfigFile(link, []byte("{}")); err == nil {
		t.Fatal("expected symlink rejection")
	}
}
