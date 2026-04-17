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

// readInlineOrPathTextArg returns inline text from inlineKey or reads it from
// an absolute local file path in pathKey. Used by local-MCP tools that accept
// large text payloads but also need a file-path escape hatch to avoid bloating
// the JSON-RPC request.
func readInlineOrPathTextArg(args map[string]any, inlineKey, pathKey string) (string, error) {
	if text := argString(args, inlineKey); text != "" {
		return text, nil
	}
	if path := argString(args, pathKey); path != "" {
		data, err := readLocalFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", fmt.Errorf("required field missing: pass %s or %s", inlineKey, pathKey)
}

func isNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

// Force import of fs for future use when we add directory-level support.
var _ fs.FS
