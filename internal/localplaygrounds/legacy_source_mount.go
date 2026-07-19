package localplaygrounds

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// This parser exists only for Playgrounds created before the Core-owned source
// plan manifest. A manifest, when present, is the sole source of checkout
// identity and bypasses this compatibility path entirely.
type legacyComposeVolume struct {
	Source string
	Target string
	Type   string
}

func legacyComposeVolumes(value any) []legacyComposeVolume {
	var out []legacyComposeVolume
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			out = append(out, parseLegacyComposeVolume(item))
		}
	case []string:
		for _, item := range typed {
			out = append(out, parseLegacyComposeVolume(item))
		}
	}
	return out
}

func parseLegacyComposeVolume(value any) legacyComposeVolume {
	switch typed := value.(type) {
	case string:
		parts, ok := splitShortVolume(strings.Trim(strings.TrimSpace(typed), `"'`))
		if !ok {
			return legacyComposeVolume{}
		}
		if len(parts) == 1 {
			return legacyComposeVolume{Target: normalizeContainerTarget(parts[0])}
		}
		return legacyComposeVolume{Source: parts[0], Target: normalizeContainerTarget(parts[1])}
	case map[string]any:
		return legacyComposeVolume{
			Source: firstScalar(typed, "source", "src"),
			Target: normalizeContainerTarget(firstScalar(typed, "target", "dst", "destination")),
			Type:   scalarString(typed["type"]),
		}
	case map[any]any:
		mapped := asMap(typed)
		return legacyComposeVolume{
			Source: firstScalar(mapped, "source", "src"),
			Target: normalizeContainerTarget(firstScalar(mapped, "target", "dst", "destination")),
			Type:   scalarString(mapped["type"]),
		}
	}
	return legacyComposeVolume{}
}

func firstScalar(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := scalarString(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func splitShortVolume(value string) ([]string, bool) {
	parts := []string{}
	start := 0
	depth := 0
	for index := 0; index < len(value); index++ {
		if index+1 < len(value) && value[index] == '$' && value[index+1] == '{' {
			depth++
			index++
			continue
		}
		if depth > 0 && value[index] == '}' {
			depth--
			continue
		}
		windowsDrive := index == 1 && isASCIIAlpha(value[0]) && index+1 < len(value) && (value[index+1] == '/' || value[index+1] == '\\')
		if depth == 0 && value[index] == ':' && !windowsDrive {
			parts = append(parts, value[start:index])
			start = index + 1
		}
	}
	if depth != 0 {
		return nil, false
	}
	return append(parts, value[start:]), true
}

func isASCIIAlpha(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func normalizeContainerTarget(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "/") || strings.ContainsRune(value, '\x00') {
		return ""
	}
	return path.Clean(value)
}

func discoverLegacyServiceMount(service *Service, sourceTarget string, volumes []legacyComposeVolume, playgroundDir string) error {
	target := normalizeContainerTarget(sourceTarget)
	if target != "" {
		exact := make(map[string]bool)
		for _, volume := range volumes {
			if volume.Target != target {
				continue
			}
			if source := resolveLegacyComposeSource(volume, playgroundDir); source != "" {
				exact[source] = true
			}
		}
		if len(exact) > 1 {
			return fmt.Errorf("multiple mounts target %s with different physical sources", target)
		}
		for source := range exact {
			service.HostMount = source
			setLegacyMountMetadata(service, source)
			if service.Branch == "" && service.Repository != "" {
				service.Branch = "main"
			}
			return nil
		}
	}

	fallback := make(map[string]bool)
	for _, volume := range volumes {
		source := resolveLegacyComposeSource(volume, playgroundDir)
		if source == "" || !strings.Contains(filepath.ToSlash(source), "/props/") {
			continue
		}
		fallback[source] = true
	}
	if len(fallback) > 1 {
		return fmt.Errorf("multiple compatibility /props/ mounts resolve to different physical checkouts")
	}
	for source := range fallback {
		service.HostMount = source
		setLegacyMountMetadata(service, source)
	}
	if service.Branch == "" && service.Repository != "" && service.HostMount != "" {
		service.Branch = "main"
	}
	return nil
}

func resolveLegacyComposeSource(volume legacyComposeVolume, playgroundDir string) string {
	source := strings.TrimSpace(volume.Source)
	if source == "" || strings.Contains(source, "${") || strings.ContainsRune(source, '\x00') {
		return ""
	}
	if volume.Type != "bind" && !filepath.IsAbs(source) && !strings.HasPrefix(source, ".") {
		return ""
	}
	if filepath.IsAbs(source) {
		return filepath.Clean(source)
	}
	return filepath.Clean(filepath.Join(playgroundDir, source))
}

func normalizeLegacyRepository(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	var host, repositoryPath string
	if at := strings.Index(raw, "@"); at >= 0 {
		if colon := strings.Index(raw[at+1:], ":"); colon >= 0 && !strings.Contains(raw[:at], "://") {
			colon += at + 1
			host = raw[at+1 : colon]
			repositoryPath = raw[colon+1:]
		}
	}
	if host == "" {
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "ssh") || parsed.Hostname() == "" {
			return ""
		}
		host = parsed.Hostname()
		repositoryPath = parsed.Path
	}
	repositoryPath = strings.TrimPrefix(repositoryPath, "/")
	repositoryPath = strings.TrimSuffix(strings.TrimSuffix(repositoryPath, "/"), ".git")
	parts := strings.Split(repositoryPath, "/")
	if len(parts) < 2 {
		return ""
	}
	for _, part := range parts {
		if part == "" || strings.ContainsAny(part, " \t\r\n") {
			return ""
		}
	}
	return strings.ToLower(host) + "/" + repositoryPath
}

func setLegacyMountMetadata(service *Service, hostPath string) {
	propsIndex := strings.Index(hostPath, "/props/")
	if propsIndex == -1 {
		return
	}
	relative := hostPath[propsIndex+7:]
	parts := strings.SplitN(relative, "/", 3)
	if len(parts) < 2 {
		return
	}
	service.HostMount = hostPath
	rawProp := parts[0]
	propParts := strings.Split(rawProp, "--")
	if len(propParts) >= 3 {
		if service.Owner == "" {
			service.Owner = propParts[0]
		}
		if service.Prop == "" {
			service.Prop = strings.Join(propParts[1:len(propParts)-1], "--")
		}
	} else if service.Prop == "" {
		service.Prop = rawProp
	}
	if service.Branch == "" {
		service.Branch = parts[1]
	}
}
