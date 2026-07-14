package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type failingDownloadReader struct {
	sent bool
}

func (r *failingDownloadReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		return copy(p, "partial"), nil
	}
	return 0, errors.New("download interrupted")
}

func TestWriteDownloadAtomicallyReplacesDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.bin")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := writeDownload(&failingDownloadReader{}, path); err == nil {
		t.Fatal("interrupted download succeeded")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "old" {
		t.Fatalf("destination changed after failure: data=%q err=%v", data, err)
	}
	if n, err := writeDownload(io.NopCloser(&fixedReader{data: []byte("new")}), path); err != nil || n != 3 {
		t.Fatalf("writeDownload n=%d err=%v", n, err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "new" {
		t.Fatalf("destination=%q err=%v", data, err)
	}
}

type fixedReader struct {
	data []byte
}

func (r *fixedReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestWriteDownloadRejectsSymlinkDestination(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := writeDownload(&fixedReader{data: []byte("replace")}, link); err == nil {
		t.Fatal("symlink destination accepted")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "keep" {
		t.Fatalf("symlink target changed: data=%q err=%v", data, err)
	}
}
