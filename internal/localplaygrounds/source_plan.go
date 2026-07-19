package localplaygrounds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

const sourcePlanFilename = ".fibe-source-plan.json"
const maxSourcePlanBytes = 1 << 20

type sourcePlanManifest struct {
	Version int               `json:"version"`
	Sources []sourcePlanEntry `json:"sources"`
}

type sourcePlanEntry struct {
	Repository   string             `json:"repository"`
	Branch       string             `json:"branch"`
	RelativePath string             `json:"relative_path"`
	Services     []sourcePlanMember `json:"services"`
}

type sourcePlanMember struct {
	Service     string  `json:"service"`
	Production  *bool   `json:"production"`
	MountTarget *string `json:"mount_target"`
}

func applySourcePlan(pg *Playground, data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest sourcePlanManifest
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("invalid source-plan manifest: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return err
	}
	if manifest.Version != 1 {
		return fmt.Errorf("unsupported source-plan manifest version %d", manifest.Version)
	}
	for _, service := range pg.Services {
		service.HostMount = ""
		service.CheckoutServices = nil
	}
	seenServices := make(map[string]bool)
	seenCheckouts := make(map[string]string)
	seenPaths := make(map[string]string)
	for _, entry := range manifest.Sources {
		if !validNormalizedRepository(entry.Repository) || !validSourcePlanBranch(entry.Branch) {
			return fmt.Errorf("source-plan entry has invalid repository or branch")
		}
		logicalCheckout := entry.Repository + "\x00" + entry.Branch
		if previous, exists := seenCheckouts[logicalCheckout]; exists {
			return fmt.Errorf("source-plan repository and branch appear more than once (%s and %s)", previous, entry.RelativePath)
		}
		if previous, exists := seenPaths[entry.RelativePath]; exists {
			return fmt.Errorf("source-plan checkout path is reused by %s and %s", previous, logicalCheckout)
		}
		if len(entry.Services) == 0 {
			return fmt.Errorf("source-plan entry has no member services")
		}
		seenCheckouts[logicalCheckout] = entry.RelativePath
		seenPaths[entry.RelativePath] = logicalCheckout
		checkout, err := resolveSourcePlanPath(pg.Path, entry.RelativePath)
		if err != nil {
			return err
		}
		allServices := make([]string, 0, len(entry.Services))
		for _, member := range entry.Services {
			if member.Production == nil {
				return fmt.Errorf("source-plan service %q has no production flag", member.Service)
			}
			if seenServices[member.Service] {
				return fmt.Errorf("source-plan service %q belongs to multiple checkouts", member.Service)
			}
			service, exists := pg.Services[member.Service]
			if !exists {
				return fmt.Errorf("source-plan service %q is absent from Compose", member.Service)
			}
			seenServices[member.Service] = true
			allServices = append(allServices, member.Service)
			service.Repository = entry.Repository
			service.Prop, service.Owner = repositoryNameAndOwner(entry.Repository)
			service.Branch = entry.Branch
			if !*member.Production && member.MountTarget != nil {
				if normalizeContainerTarget(*member.MountTarget) == "" {
					return fmt.Errorf("source-plan service %q has invalid mount target", member.Service)
				}
				service.HostMount = checkout
			}
		}
		sort.Strings(allServices)
		for _, member := range entry.Services {
			if service := pg.Services[member.Service]; service.HostMount == checkout {
				service.CheckoutServices = append([]string(nil), allServices...)
			}
		}
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("source-plan manifest contains trailing JSON")
		}
		return fmt.Errorf("invalid source-plan manifest: %w", err)
	}
	return nil
}

func validNormalizedRepository(value string) bool {
	parts := strings.Split(value, "/")
	if len(parts) < 3 || parts[0] != strings.ToLower(parts[0]) || !validRepositoryHost(parts[0]) {
		return false
	}
	for _, part := range parts[1:] {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, " \t\r\n\x00:@\\") {
			return false
		}
	}
	return true
}

func validRepositoryHost(value string) bool {
	host := value
	if separator := strings.LastIndexByte(value, ':'); separator >= 0 {
		host = value[:separator]
		port := value[separator+1:]
		if port == "" || len(port) > 5 {
			return false
		}
		portNumber := 0
		for index := range len(port) {
			if port[index] < '0' || port[index] > '9' {
				return false
			}
			portNumber = portNumber*10 + int(port[index]-'0')
		}
		if portNumber < 1 || portNumber > 65535 {
			return false
		}
	}
	if len(host) == 0 || len(host) > 253 || strings.Contains(host, "..") || !isASCIIAlphaNumeric(host[0]) || !isASCIIAlphaNumeric(host[len(host)-1]) {
		return false
	}
	for index := 1; index < len(host)-1; index++ {
		if !isASCIIAlphaNumeric(host[index]) && host[index] != '.' && host[index] != '-' {
			return false
		}
	}
	return true
}

func validSourcePlanBranch(value string) bool {
	if len(value) == 0 || len(value) > 255 || value[0] == '-' || !isASCIIAlphaNumeric(value[0]) || strings.Contains(value, "..") || strings.Contains(value, "@{") || strings.Contains(value, "//") || strings.HasSuffix(value, ".") || strings.HasSuffix(value, "/") {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if !isASCIIAlphaNumeric(character) && character != '.' && character != '_' && character != '/' && character != '-' {
			return false
		}
	}
	return true
}

func isASCIIAlphaNumeric(value byte) bool {
	return isASCIIAlpha(value) || value >= '0' && value <= '9'
}

func resolveSourcePlanPath(playgroundDir, relative string) (string, error) {
	if !validSourcePlanRelativePath(relative) {
		return "", fmt.Errorf("source-plan checkout path must be a contained props path")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	checkout := filepath.Join(playgroundDir, clean)
	relativeToPlayground, err := filepath.Rel(playgroundDir, checkout)
	if err != nil || relativeToPlayground == ".." || strings.HasPrefix(relativeToPlayground, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source-plan checkout path escapes the Playground")
	}
	return filepath.Clean(checkout), nil
}

func validSourcePlanRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") || !strings.HasPrefix(value, "props/") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || !isASCIIAlphaNumeric(segment[0]) {
			return false
		}
		for index := 1; index < len(segment); index++ {
			character := segment[index]
			if !isASCIIAlphaNumeric(character) && character != '.' && character != '_' && character != '-' {
				return false
			}
		}
	}
	return true
}
