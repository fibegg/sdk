package mcpserver

import (
	"net/url"
	"strings"
)

// parseRepoFullName normalizes short, HTTPS, and SSH GitHub references to owner/repo.
func parseRepoFullName(input string) string {
	s := strings.TrimSpace(input)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") && !strings.HasPrefix(s, "git@") {
		s = strings.TrimSuffix(s, ".git")
		if strings.Count(s, "/") == 1 {
			return s
		}
		return ""
	}
	if strings.HasPrefix(s, "git@") {
		if i := strings.Index(s, ":"); i != -1 {
			s = strings.TrimSuffix(s[i+1:], ".git")
			if strings.Count(s, "/") == 1 {
				return s
			}
		}
		return ""
	}
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
