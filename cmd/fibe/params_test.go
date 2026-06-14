package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func resetParamsForTest(t *testing.T) {
	t.Helper()
	oldFlag := flagFromFile
	oldRaw := rawPayload
	oldStdin := os.Stdin
	t.Cleanup(func() {
		flagFromFile = oldFlag
		rawPayload = oldRaw
		os.Stdin = oldStdin
	})
	flagFromFile = ""
	rawPayload = nil
}

func TestApplyFromFileAutoReadsRegularStdinOnly(t *testing.T) {
	resetParamsForTest(t)

	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(`{"name":"from-file-redirection"}`), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer file.Close()
	os.Stdin = file

	var payload struct {
		Name string `json:"name"`
	}
	if err := applyFromFile(&payload); err != nil {
		t.Fatalf("applyFromFile: %v", err)
	}
	if payload.Name != "from-file-redirection" {
		t.Fatalf("name = %q", payload.Name)
	}
}

func TestApplyFromFileDoesNotImplicitlyReadPipeStdin(t *testing.T) {
	resetParamsForTest(t)

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer read.Close()
	defer write.Close()
	os.Stdin = read

	done := make(chan error, 1)
	go func() {
		var payload struct {
			Name string `json:"name"`
		}
		done <- applyFromFile(&payload)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("applyFromFile: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("applyFromFile blocked on implicit pipe stdin")
	}
}

func TestApplyFromFileExplicitDashReadsPipeStdin(t *testing.T) {
	resetParamsForTest(t)

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdin = read
	flagFromFile = "-"

	if _, err := write.WriteString(`{"name":"from-explicit-stdin"}`); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if err := write.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}
	defer read.Close()

	var payload struct {
		Name string `json:"name"`
	}
	if err := applyFromFile(&payload); err != nil {
		t.Fatalf("applyFromFile: %v", err)
	}
	if payload.Name != "from-explicit-stdin" {
		t.Fatalf("name = %q", payload.Name)
	}
}
