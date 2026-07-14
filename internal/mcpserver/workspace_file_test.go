package mcpserver

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceMirrorCommitsAtomically(t *testing.T) {
	root := t.TempDir()
	reader, mirror, err := newWorkspaceMirror(root, "nested/report.txt", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer mirror.Abort()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		t.Fatal(err)
	}
	if err := mirror.Commit(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "nested", "report.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("workspace content=%q err=%v", data, err)
	}
	if info, err := os.Stat(filepath.Join(root, "nested", "report.txt")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("workspace mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestWorkspaceMirrorRejectsEscapeAndSymlink(t *testing.T) {
	root := t.TempDir()
	if _, _, err := newWorkspaceMirror(root, "../escape", bytes.NewReader(nil)); err == nil {
		t.Fatal("traversal accepted")
	}
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newWorkspaceMirror(root, "link", bytes.NewReader(nil)); err == nil {
		t.Fatal("symlink target accepted")
	}
}

func TestWorkspaceMirrorDoesNotCommitPartialSource(t *testing.T) {
	root := t.TempDir()
	reader, mirror, err := newWorkspaceMirror(root, "partial.txt", bytes.NewBufferString("unfinished"))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2)
	if _, err := reader.Read(buf); err != nil {
		t.Fatal(err)
	}
	if err := mirror.Commit(); err == nil {
		t.Fatal("partial source committed")
	}
	mirror.Abort()
	if _, err := os.Lstat(filepath.Join(root, "partial.txt")); !os.IsNotExist(err) {
		t.Fatalf("partial target exists: %v", err)
	}
}
