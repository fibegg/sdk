package mcpserver

import "strings"

// splitContentHeader separates the SDK's legacy combined filename/content-type
// slot. MIME-looking values become content types; all other values remain filenames.
func splitContentHeader(raw string) (filename, contentType string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	if strings.Contains(s, "/") && !strings.ContainsAny(s, " .;") {
		return "", s
	}
	if idx := strings.Index(s, "filename="); idx != -1 {
		rest := s[idx+len("filename="):]
		rest = strings.Trim(rest, "\"';,")
		if semi := strings.Index(rest, ";"); semi != -1 {
			rest = rest[:semi]
		}
		return strings.TrimSpace(rest), ""
	}
	return s, ""
}
