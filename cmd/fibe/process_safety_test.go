package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnlyMainUsesOSExit(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob go files: %v", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "main.go" {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(data), "os.Exit(") {
			t.Fatalf("%s uses os.Exit; command handlers must return errors so MCP sessions survive failures", file)
		}
	}
}
