package mcpserver

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// readLocalFile reads a file from the local filesystem. Used by tools that
// accept content_path as a convenience for the local-MCP-only mode. The
// path must be absolute to avoid surprises from the server's cwd.
//
// Returns ErrNotExist wrapping so callers can surface a clean message.
func readLocalFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("content_path must be absolute, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if ok := isNotExist(err); ok {
			return nil, fmt.Errorf("content_path does not exist: %s", path)
		}
		return nil, fmt.Errorf("read content_path %s: %w", path, err)
	}
	return data, nil
}

// readLocalFileBase64 is a convenience for tools that want content as
// base64-encoded string.
func readLocalFileBase64(path string) (string, error) {
	data, err := readLocalFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func isNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

// Force import of fs for future use when we add directory-level support.
var _ fs.FS
