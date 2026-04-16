package mcpserver

import (
	"net/url"
	"strings"
)

// parseRepoFullName extracts "owner/repo" from a variety of forms callers
// might pass to fibe_props_attach: a full HTTPS URL, an SSH URL, or the
// short form itself. Returns empty string when the input can't be mapped
// cleanly — in which case the caller falls back to whatever was supplied.
//
// Examples:
//
//	parseRepoFullName("octocat/Hello-World")                → "octocat/Hello-World"
//	parseRepoFullName("https://github.com/octocat/Hello-World.git") → "octocat/Hello-World"
//	parseRepoFullName("git@github.com:octocat/Hello-World.git")     → "octocat/Hello-World"
func parseRepoFullName(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	// Short form already?
	if !strings.Contains(s, "://") && !strings.HasPrefix(s, "git@") {
		s = strings.TrimSuffix(s, ".git")
		if strings.Count(s, "/") == 1 {
			return s
		}
		return ""
	}
	// SSH form: git@host:owner/repo(.git)
	if strings.HasPrefix(s, "git@") {
		if i := strings.Index(s, ":"); i != -1 {
			s = strings.TrimSuffix(s[i+1:], ".git")
			if strings.Count(s, "/") == 1 {
				return s
			}
		}
		return ""
	}
	// URL form.
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	p := strings.TrimPrefix(u.Path, "/")
	p = strings.TrimSuffix(p, ".git")
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}
