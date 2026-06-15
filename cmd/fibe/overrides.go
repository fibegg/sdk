package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"gopkg.in/yaml.v3"
)

func parseKeyValueFlags(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, raw := range values {
		key, value, ok := strings.Cut(raw, "=")
		key = normalizeVariableFlagKey(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --var %q, expected key=value", raw)
		}
		out[key] = value
	}
	return out, nil
}

func parseStringMapFlags(flagName string, values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, raw := range values {
		key, value, ok := strings.Cut(raw, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --%s %q, expected key=value", flagName, raw)
		}
		out[key] = value
	}
	return out, nil
}

func parseSubdomainFlags(values []string) (map[string]string, error) {
	return parseStringMapFlags("subdomain", values)
}

func parseEnvFlags(values []string) (map[string]string, error) {
	return parseStringMapFlags("env", values)
}

func parsePlaygroundServiceOverrides(values []string) (map[string]*fibe.ServiceConfig, error) {
	services := map[string]*fibe.ServiceConfig{}
	for _, value := range values {
		if err := applyPlaygroundServiceOverride(services, value); err != nil {
			return nil, err
		}
	}
	return services, nil
}

func applyPlaygroundServiceOverrides(params *fibe.PlaygroundCreateParams, values []string) error {
	services, err := parsePlaygroundServiceOverrides(values)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}
	if params.Services == nil {
		params.Services = map[string]*fibe.ServiceConfig{}
	}
	for name, cfg := range services {
		mergeServiceConfig(params.Services, name, cfg)
	}
	return nil
}

func applyPlaygroundServiceOverride(services map[string]*fibe.ServiceConfig, raw string) error {
	left, value, ok := strings.Cut(raw, "=")
	if !ok {
		return fmt.Errorf("--service must use SERVICE.FIELD=VALUE")
	}
	serviceName, field, ok := strings.Cut(strings.TrimSpace(left), ".")
	if !ok || strings.TrimSpace(serviceName) == "" || strings.TrimSpace(field) == "" {
		return fmt.Errorf("--service must use SERVICE.FIELD=VALUE")
	}
	serviceName = strings.TrimSpace(serviceName)
	field = strings.TrimSpace(field)
	cfg := services[serviceName]
	if cfg == nil {
		cfg = &fibe.ServiceConfig{}
		services[serviceName] = cfg
	}

	switch {
	case field == "subdomain":
		cfg.Subdomain = value
	case field == "exposure_port":
		port, err := parsePlaygroundServicePort(value)
		if err != nil {
			return fmt.Errorf("--service %s.exposure_port: %w", serviceName, err)
		}
		cfg.ExposurePort = &port
	case field == "exposure_visibility":
		if value != "internal" && value != "external" {
			return fmt.Errorf("--service %s.exposure_visibility must be internal or external", serviceName)
		}
		cfg.ExposureVisibility = value
	case field == "path_rule":
		cfg.PathRule = value
	case field == "start_command":
		cfg.StartCommand = value
	case field == "image":
		cfg.Image = value
	case field == "dockerfile_path":
		cfg.DockerfilePath = value
	case field == "env_file_path":
		cfg.EnvFilePath = value
	case field == "healthcheck_path":
		cfg.HealthcheckPath = value
	case strings.HasPrefix(field, "env_vars."):
		key := strings.TrimPrefix(field, "env_vars.")
		if key == "" {
			return fmt.Errorf("--service env_vars key cannot be blank")
		}
		if cfg.EnvVars == nil {
			cfg.EnvVars = map[string]string{}
		}
		cfg.EnvVars[key] = value
	case strings.HasPrefix(field, "git_config."):
		if err := applyPlaygroundGitConfigOverride(cfg, serviceName, strings.TrimPrefix(field, "git_config."), value); err != nil {
			return err
		}
	default:
		if field == "port_mappings" || strings.HasPrefix(field, "port_mappings.") {
			return fmt.Errorf("--service does not support port_mappings; use -f JSON/YAML for port mappings")
		}
		if strings.Contains(field, ".") {
			return fmt.Errorf("--service does not support service names with dots; use -f JSON/YAML")
		}
		return fmt.Errorf("--service field %q is not supported", field)
	}
	return nil
}

func parsePlaygroundServicePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("must be an integer from 1 to 65535")
	}
	return port, nil
}

func applyPlaygroundGitConfigOverride(cfg *fibe.ServiceConfig, serviceName, field, value string) error {
	if cfg.GitConfig == nil {
		cfg.GitConfig = &fibe.GitConfig{}
	}
	switch field {
	case "branch_name":
		cfg.GitConfig.BranchName = value
	case "base_branch_name":
		cfg.GitConfig.BaseBranchName = value
	case "create_branch":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("--service %s.git_config.create_branch must be true or false", serviceName)
		}
		cfg.GitConfig.CreateBranch = parsed
	default:
		return fmt.Errorf("--service git_config field %q is not supported", field)
	}
	return nil
}

func mergeServiceConfig(dst map[string]*fibe.ServiceConfig, name string, src *fibe.ServiceConfig) {
	if src == nil {
		return
	}
	current := dst[name]
	if current == nil {
		dst[name] = src
		return
	}
	if src.Subdomain != "" {
		current.Subdomain = src.Subdomain
	}
	if src.ExposureVisibility != "" {
		current.ExposureVisibility = src.ExposureVisibility
	}
	if src.PathRule != "" {
		current.PathRule = src.PathRule
	}
	if src.StartCommand != "" {
		current.StartCommand = src.StartCommand
	}
	if src.DockerfilePath != "" {
		current.DockerfilePath = src.DockerfilePath
	}
	if src.EnvFilePath != "" {
		current.EnvFilePath = src.EnvFilePath
	}
	if src.HealthcheckPath != "" {
		current.HealthcheckPath = src.HealthcheckPath
	}
	if src.Image != "" {
		current.Image = src.Image
	}
	if src.ExposurePort != nil {
		current.ExposurePort = src.ExposurePort
	}
	if src.EnvVars != nil {
		if current.EnvVars == nil {
			current.EnvVars = map[string]string{}
		}
		for key, value := range src.EnvVars {
			current.EnvVars[key] = value
		}
	}
	if src.GitConfig != nil {
		current.GitConfig = src.GitConfig
	}
	if src.Exposure != nil {
		current.Exposure = src.Exposure
	}
	if len(src.PortMappings) > 0 {
		current.PortMappings = src.PortMappings
	}
}

func serviceConfigMapAny(services map[string]*fibe.ServiceConfig) map[string]any {
	if len(services) == 0 {
		return nil
	}
	out := make(map[string]any, len(services))
	for name, cfg := range services {
		out[name] = cfg
	}
	return out
}

func validateServiceOverrideNames(known []string, services map[string]*fibe.ServiceConfig) error {
	if len(services) == 0 || len(known) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, name := range known {
		if name = strings.TrimSpace(name); name != "" {
			allowed[name] = true
		}
	}
	for name := range services {
		if !allowed[name] {
			return fmt.Errorf("unknown service %q (available: %s)", name, strings.Join(known, ", "))
		}
	}
	return nil
}

func playspecServiceNames(ps *fibe.Playspec) []string {
	if ps == nil {
		return nil
	}
	return serviceNamesFromAnySlice(ps.Services)
}

func serviceNamesFromAnySlice(raw []any) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range raw {
		name := serviceNameFromAny(item)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

func serviceNameFromAny(item any) string {
	switch v := item.(type) {
	case map[string]any:
		return strings.TrimSpace(fmt.Sprint(v["name"]))
	case map[string]string:
		return strings.TrimSpace(v["name"])
	case fibe.PlayspecServiceDef:
		return v.Name
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(decoded["name"]))
	}
}

func composeServiceNames(composeYAML string) []string {
	var decoded struct {
		Services map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(composeYAML), &decoded); err != nil {
		return nil
	}
	out := make([]string, 0, len(decoded.Services))
	for name := range decoded.Services {
		out = append(out, name)
	}
	return out
}

func resolveCategoryID(c *fibe.Client, selector string) (int64, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return 0, nil
	}
	if id, err := strconv.ParseInt(selector, 10, 64); err == nil {
		return id, nil
	}
	cats, err := c.TemplateCategories.List(ctx(), &fibe.ListParams{PerPage: 100})
	if err != nil {
		return 0, err
	}
	var matches []fibe.TemplateCategory
	for _, cat := range cats.Data {
		if strings.EqualFold(cat.Name, selector) || strings.EqualFold(cat.Slug, selector) {
			matches = append(matches, cat)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("category %q not found", selector)
	case 1:
		return matches[0].ID, nil
	default:
		return 0, fmt.Errorf("category %q is ambiguous", selector)
	}
}

func resolveCredentialID(c *fibe.Client, playspecIdentifier, selector string) (string, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return "", nil
	}
	ps, err := c.Playspecs.GetByIdentifier(ctx(), playspecIdentifier)
	if err != nil {
		return "", err
	}
	credentials, err := playspecCredentials(ps)
	if err != nil {
		return "", err
	}
	var matches []fibe.RegistryCredentialInfo
	for _, credential := range credentials {
		composite := strings.Join([]string{credential.RegistryType, credential.RegistryURL, credential.Username}, "/")
		if credential.ID == selector ||
			credential.RegistryURL == selector ||
			credential.Username == selector ||
			composite == selector {
			matches = append(matches, credential)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("credential %q not found on playspec %s", selector, playspecIdentifier)
	case 1:
		return matches[0].ID, nil
	default:
		return "", fmt.Errorf("credential %q is ambiguous on playspec %s", selector, playspecIdentifier)
	}
}

func playspecCredentials(ps *fibe.Playspec) ([]fibe.RegistryCredentialInfo, error) {
	if ps == nil || ps.Credentials == nil {
		return nil, nil
	}
	data, err := json.Marshal(ps.Credentials)
	if err != nil {
		return nil, err
	}
	var direct []fibe.RegistryCredentialInfo
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}
	var wrapped fibe.RegistryCredentialResult
	if err := json.Unmarshal(data, &wrapped); err == nil {
		return wrapped.Credentials, nil
	}
	return nil, fmt.Errorf("could not parse playspec credentials")
}
