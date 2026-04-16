package mcpserver

import "strings"

// splitContentHeader takes the single-string "filename" value the SDK's
// Download method returns (which historically stuffs both filename and
// content-type into one slot depending on how the API surfaces them) and
// separates it into a best-effort filename + content-type pair.
//
// Rules of thumb observed in the wild:
//
//   - If the value contains a slash (e.g. "text/x-python"), we treat it
//     as a content-type and leave filename empty — the caller can fall
//     back to the artefact's stored name.
//   - If the value looks like filename="foo.txt" or contains an extension,
//     we treat it as a filename.
//   - Otherwise we hand it back as filename for backwards compatibility.
func splitContentHeader(raw string) (filename, contentType string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ""
	}
	// Content-Type-ish: contains exactly one "/" and no spaces, no dots.
	if strings.Contains(s, "/") && !strings.ContainsAny(s, " .;") {
		return "", s
	}
	// Content-Disposition-ish: filename="foo.txt"
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
