package mcpserver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

var workspaceTempCounter atomic.Uint64

type workspaceMirror struct {
	root      *os.Root
	tempName  string
	target    string
	file      *os.File
	source    *completionReader
	committed bool
}

type completionReader struct {
	reader io.Reader
	eof    bool
}

func (r *completionReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err == io.EOF {
		r.eof = true
	}
	return n, err
}

func newWorkspaceMirror(workspacePath, filename string, source io.Reader) (io.Reader, *workspaceMirror, error) {
	clean := filepath.Clean(filename)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, nil, fmt.Errorf("invalid filename for workspace: must be relative path without traversal")
	}
	root, err := os.OpenRoot(workspacePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open workspace root: %w", err)
	}
	dir := filepath.Dir(clean)
	if err := root.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("create workspace directory: %w", err), root.Close())
	}
	if info, err := root.Lstat(clean); err == nil {
		if !info.Mode().IsRegular() {
			return nil, nil, errors.Join(fmt.Errorf("workspace target must be a regular file: %s", filename), root.Close())
		}
	} else if !os.IsNotExist(err) {
		return nil, nil, errors.Join(fmt.Errorf("inspect workspace target: %w", err), root.Close())
	}
	tempName := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d-%d", filepath.Base(clean), os.Getpid(), workspaceTempCounter.Add(1)))
	file, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("create workspace temporary file: %w", err), root.Close())
	}
	tracked := &completionReader{reader: io.LimitReader(source, maxLocalFile+1)}
	mirror := &workspaceMirror{root: root, tempName: tempName, target: clean, file: file, source: tracked}
	return io.TeeReader(tracked, file), mirror, nil
}

func (m *workspaceMirror) Commit() error {
	if m == nil || m.committed {
		return nil
	}
	if !m.source.eof {
		return fmt.Errorf("workspace artefact source was not read completely")
	}
	if info, err := m.file.Stat(); err != nil {
		return err
	} else if info.Size() > maxLocalFile {
		return fmt.Errorf("workspace artefact exceeds %d bytes", maxLocalFile)
	}
	if err := m.file.Sync(); err != nil {
		return err
	}
	if err := m.file.Close(); err != nil {
		return err
	}
	if err := m.root.Rename(m.tempName, m.target); err != nil {
		return err
	}
	dir, err := m.root.Open(filepath.Dir(m.target))
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		return errors.Join(err, dir.Close())
	}
	if err := dir.Close(); err != nil {
		return err
	}
	m.committed = true
	return m.root.Close()
}

func (m *workspaceMirror) Abort() {
	if m == nil || m.committed {
		return
	}
	_ = m.file.Close()
	_ = m.root.Remove(m.tempName)
	_ = m.root.Close()
}
