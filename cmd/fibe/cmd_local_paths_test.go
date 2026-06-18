package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalScreenshotPathJSONIsNonMutatingByDefault(t *testing.T) {
	linkDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(linkDir, ".current_playground.json"), []byte(`{"name":"Demo Playground"}`), 0o644); err != nil {
		t.Fatalf("write current state: %v", err)
	}

	payload := runLocalScreenshotPathCommand(t, linkDir, "Home Page", false)

	if payload.Type != "screenshot" || payload.Name != "Home Page" || payload.Playground != "Demo Playground" {
		t.Fatalf("payload = %#v", payload)
	}
	wantPath := filepath.Join(linkDir, ".artifacts", "screenshots", "Demo-Playground", "Home-Page.png")
	if payload.Path != wantPath || payload.Dir != filepath.Dir(wantPath) {
		t.Fatalf("path=%q dir=%q want %q", payload.Path, payload.Dir, wantPath)
	}
	if payload.CreatedDir {
		t.Fatal("CreatedDir=true, want false")
	}
	if _, err := os.Stat(payload.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dir stat err=%v want not exist", err)
	}
}

func TestLocalScreenshotPathMkdirCreatesParent(t *testing.T) {
	linkDir := t.TempDir()

	payload := runLocalScreenshotPathCommand(t, linkDir, "report/preview", true)

	if !payload.CreatedDir {
		t.Fatal("CreatedDir=false, want true")
	}
	if payload.Path != filepath.Join(linkDir, ".artifacts", "screenshots", "report-preview.png") {
		t.Fatalf("path=%q", payload.Path)
	}
	info, err := os.Stat(payload.Dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", payload.Dir)
	}
}

func runLocalScreenshotPathCommand(t *testing.T, linkDir string, name string, mkdir bool) localResolvedPath {
	t.Helper()

	oldOutput := flagOutput
	flagOutput = "json"
	t.Cleanup(func() { flagOutput = oldOutput })

	args := []string{"--name", name, "--link-dir", linkDir}
	if mkdir {
		args = append(args, "--mkdir")
	}
	cmd := localScreenshotPathCmd()
	cmd.SetArgs(args)

	stdout, err := captureLocalPathsStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var payload localResolvedPath
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json %q: %v", stdout, err)
	}
	return payload
}

func captureLocalPathsStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer

	runErr := fn()
	closeErr := writer.Close()
	os.Stdout = oldStdout

	output, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("read stdout: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close stdout: %v", closeErr)
	}
	return string(output), runErr
}
