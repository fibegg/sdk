// Package domainnames identifies removed FIBE vocabulary without accepting aliases.
package domainnames

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var Resources = map[string]string{
	"marque": "host", "marquee": "host", "marquees": "hosts", "mq": "hosts",
	"playspec": "spec", "playspecs": "specs", "ps": "specs",
	"prop": "repository", "props": "repositories", "pr": "repositories",
	"trick": "task", "tricks": "tasks", "tr": "tasks",
	"scrolls": "library", "genie": "agent", "genies": "agents", "crumbs": "audit_logs",
}

func Replacement(name string) (string, bool) {
	replacement, ok := Resources[strings.ToLower(strings.TrimSpace(name))]
	return replacement, ok
}

func RemovedError(name string) error {
	if replacement, ok := Replacement(name); ok {
		return fmt.Errorf("FIBE name %q was removed; use %q", name, replacement)
	}
	return nil
}

func FieldReplacement(key string) (string, bool) {
	switch key {
	case "marqueeRoot":
		return "hostRoot", true
	case "marqueeRootDomain":
		return "hostRootDomain", true
	}
	parts := strings.Split(key, "_")
	changed := false
	for i, part := range parts {
		switch part {
		case "marque", "marquee":
			parts[i] = "host"
			changed = true
		case "marquees":
			parts[i] = "hosts"
			changed = true
		case "playspec":
			parts[i] = "spec"
			changed = true
		case "playspecs":
			parts[i] = "specs"
			changed = true
		case "prop":
			parts[i] = "repository"
			changed = true
		case "props":
			parts[i] = "repositories"
			changed = true
		case "crumbs":
			parts[i] = "audit_logs"
			changed = true
		case "scrolls":
			parts[i] = "library"
			changed = true
		case "trick":
			parts[i] = "task"
			changed = true
		case "tricks":
			parts[i] = "tasks"
			changed = true
		}
	}
	return strings.Join(parts, "_"), changed
}

// RejectFields checks structural FIBE fields; application-owned extension maps
// retain their own property names and opaque values.
func RejectFields(value any) error {
	switch value := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if replacement, removed := FieldReplacement(key); removed {
				return fmt.Errorf("FIBE field %q was removed; use %q", key, replacement)
			}
			switch key {
			case "metadata", "custom_env", "env_vars", "environment", "provider_args", "properties":
				continue
			}
			if err := RejectFields(value[key]); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range value {
			if err := RejectFields(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func CheckEnvironment() error {
	replacements := map[string]string{
		"MARQUEE_ROOT": "HOST_ROOT", "MARQUEE_ROOT_DOMAIN": "HOST_ROOT_DOMAIN", "MARQUEE_URL_SCHEME": "HOST_URL_SCHEME",
		"FIBE_MARQUEE_ID": "FIBE_HOST_ID", "FIBE_PLAYSPEC_ID": "FIBE_SPEC_ID", "FIBE_PROP_ID": "FIBE_REPOSITORY_ID",
	}
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, exists := os.LookupEnv(key); exists {
			return fmt.Errorf("FIBE environment variable %s was removed; use %s", key, replacements[key])
		}
	}
	return nil
}

func ToolReplacement(name string) (string, bool) {
	parts := strings.Split(name, "_")
	changed := false
	for i, part := range parts {
		if replacement, ok := Replacement(part); ok {
			parts[i] = replacement
			changed = true
		}
	}
	return strings.Join(parts, "_"), changed
}
