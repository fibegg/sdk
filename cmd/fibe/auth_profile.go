package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

const (
	defaultProfile = "default"
	defaultDomain  = "fibe.gg"
)

type cliProfileConfig struct {
	Domain string `json:"domain"`
}

type cliConfigFile struct {
	ActiveProfile string                      `json:"active_profile,omitempty"`
	Profiles      map[string]cliProfileConfig `json:"profiles,omitempty"`
}

type cliConfigStore struct {
	path string
}

type resolvedAuth struct {
	Profile           string
	Domain            string
	APIKey            string
	APIKeyID          int64
	AuthSource        string
	DomainSource      string
	ProfileConfigured bool
	IgnoredEnv        []string
}

type authProfileRow struct {
	Name         string
	Domain       string
	Active       bool
	HasKey       bool
	MaskedKey    string
	APIKeyID     int64
	CredSource   string
	DomainSource string
}

func defaultCLIConfigPath() string {
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(cfgDir, "fibe", "config.json")
}

func newCLIConfigStore(path string) *cliConfigStore {
	return &cliConfigStore{path: path}
}

func (s *cliConfigStore) load() (*cliConfigFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cliConfigFile{Profiles: map[string]cliProfileConfig{}}, nil
		}
		return nil, err
	}
	var cfg cliConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]cliProfileConfig{}
	}
	return &cfg, nil
}

func (s *cliConfigStore) save(cfg *cliConfigFile) error {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]cliProfileConfig{}
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o644)
}

func (s *cliConfigStore) setProfile(profile, domain string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}
	cfg.Profiles[profile] = cliProfileConfig{Domain: normalizeDomainInput(domain)}
	return s.save(cfg)
}

func (s *cliConfigStore) setActive(profile string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}
	cfg.ActiveProfile = profile
	return s.save(cfg)
}

func (s *cliConfigStore) deleteProfile(profile string) error {
	cfg, err := s.load()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, profile)
	if cfg.ActiveProfile == profile {
		cfg.ActiveProfile = ""
	}
	return s.save(cfg)
}

func validateProfileName(profile string) error {
	if strings.TrimSpace(profile) == "" {
		return fmt.Errorf("profile name cannot be empty")
	}
	if strings.ContainsAny(profile, " \t\r\n") {
		return fmt.Errorf("profile name %q cannot contain whitespace", profile)
	}
	return nil
}

func selectedProfileName() string {
	if flagProfile != "" {
		return flagProfile
	}
	cfg, err := newCLIConfigStore(defaultCLIConfigPath()).load()
	if err == nil && cfg.ActiveProfile != "" {
		return cfg.ActiveProfile
	}
	return defaultProfile
}

func loginProfileName() string {
	if flagProfile != "" {
		return flagProfile
	}
	return defaultProfile
}

func resolveCLIAuth() resolvedAuth {
	cfg, _ := newCLIConfigStore(defaultCLIConfigPath()).load()
	creds := fibe.NewCredentialStore(fibe.DefaultCredentialPath())

	profile := flagProfile
	if profile == "" && cfg != nil && cfg.ActiveProfile != "" {
		profile = cfg.ActiveProfile
	}
	if profile == "" {
		profile = defaultProfile
	}

	var cfgDomain string
	var cfgHasProfile bool
	if cfg != nil {
		if p, ok := cfg.Profiles[profile]; ok {
			cfgDomain = p.Domain
			cfgHasProfile = true
		}
	}

	entry, _ := creds.GetProfile(profile)
	credHasProfile := entry != nil
	profileConfigured := flagProfile != "" || (cfg != nil && cfg.ActiveProfile != "") || cfgHasProfile || credHasProfile

	domain := defaultDomain
	domainSource := "default"
	if cfgDomain != "" {
		domain = cfgDomain
		domainSource = "profile " + profile
	} else if entry != nil && entry.Domain != "" {
		domain = entry.Domain
		domainSource = "profile " + profile
	}

	var apiKey string
	var apiKeyID int64
	authSource := "none"
	if entry != nil && entry.APIKey != "" {
		apiKey = entry.APIKey
		apiKeyID = entry.APIKeyID
		authSource = "profile " + profile
	}

	envDomain := strings.TrimSpace(os.Getenv("FIBE_DOMAIN"))
	envKey := strings.TrimSpace(os.Getenv("FIBE_API_KEY"))
	if !profileConfigured {
		if envDomain != "" {
			domain = envDomain
			domainSource = "FIBE_DOMAIN env"
		}
		if envKey != "" {
			apiKey = envKey
			authSource = "FIBE_API_KEY env"
		}
	}

	if flagDomain != "" {
		domain = flagDomain
		domainSource = "--domain flag"
	}
	if flagAPIKey != "" {
		apiKey = flagAPIKey
		apiKeyID = 0
		authSource = "--api-key flag"
	}

	var ignored []string
	if profileConfigured {
		if envDomain != "" && domainSource != "FIBE_DOMAIN env" && flagDomain == "" {
			ignored = append(ignored, "FIBE_DOMAIN")
		}
		if envKey != "" && authSource != "FIBE_API_KEY env" && flagAPIKey == "" {
			ignored = append(ignored, "FIBE_API_KEY")
		}
	}

	return resolvedAuth{
		Profile:           profile,
		Domain:            normalizeDomainInput(domain),
		APIKey:            apiKey,
		APIKeyID:          apiKeyID,
		AuthSource:        authSource,
		DomainSource:      domainSource,
		ProfileConfigured: profileConfigured,
		IgnoredEnv:        ignored,
	}
}

func saveAuthProfile(profile, domain, apiKey string, apiKeyID int64) error {
	if err := validateProfileName(profile); err != nil {
		return err
	}
	domain = normalizeDomainInput(domain)
	if err := newCLIConfigStore(defaultCLIConfigPath()).setProfile(profile, domain); err != nil {
		return err
	}
	if err := fibe.NewCredentialStore(fibe.DefaultCredentialPath()).SetProfile(profile, &fibe.CredentialEntry{
		APIKey:   apiKey,
		APIKeyID: apiKeyID,
		Domain:   domain,
	}); err != nil {
		return err
	}
	return newCLIConfigStore(defaultCLIConfigPath()).setActive(profile)
}

func deleteAuthProfile(profile string) error {
	if err := fibe.NewCredentialStore(fibe.DefaultCredentialPath()).DeleteProfile(profile); err != nil {
		return err
	}
	return newCLIConfigStore(defaultCLIConfigPath()).deleteProfile(profile)
}

func profileExists(profile string) bool {
	if profile == defaultProfile {
		return true
	}
	cfg, _ := newCLIConfigStore(defaultCLIConfigPath()).load()
	if cfg != nil {
		if _, ok := cfg.Profiles[profile]; ok {
			return true
		}
	}
	entry, _ := fibe.NewCredentialStore(fibe.DefaultCredentialPath()).GetProfile(profile)
	return entry != nil
}

func listAuthProfiles() ([]authProfileRow, error) {
	cfg, err := newCLIConfigStore(defaultCLIConfigPath()).load()
	if err != nil {
		return nil, err
	}
	creds := fibe.NewCredentialStore(fibe.DefaultCredentialPath())
	credProfiles, err := creds.ListProfiles()
	if err != nil {
		return nil, err
	}

	names := map[string]bool{}
	for name := range cfg.Profiles {
		names[name] = true
	}
	for name := range credProfiles {
		names[name] = true
	}
	if cfg.ActiveProfile != "" {
		names[cfg.ActiveProfile] = true
	}

	rows := make([]authProfileRow, 0, len(names))
	for name := range names {
		domain := ""
		domainSource := ""
		if p, ok := cfg.Profiles[name]; ok {
			domain = p.Domain
			domainSource = "config"
		}
		entry := credProfiles[name]
		if domain == "" && entry != nil {
			domain = entry.Domain
			domainSource = "credentials"
		}
		if domain == "" && name == defaultProfile {
			domain = defaultDomain
			domainSource = "default"
		}
		row := authProfileRow{
			Name:         name,
			Domain:       normalizeDomainInput(domain),
			Active:       cfg.ActiveProfile == name || (cfg.ActiveProfile == "" && name == defaultProfile),
			DomainSource: domainSource,
		}
		if entry != nil && entry.APIKey != "" {
			row.HasKey = true
			row.MaskedKey = maskKey(entry.APIKey)
			row.APIKeyID = entry.APIKeyID
			row.CredSource = "credentials"
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Active != rows[j].Active {
			return rows[i].Active
		}
		return rows[i].Name < rows[j].Name
	})
	return rows, nil
}

func normalizeDomainInput(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return defaultDomain
	}
	domain = strings.TrimRight(domain, "/")
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		if u, err := url.Parse(domain); err == nil && u.Host != "" {
			return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
		}
	}
	return domain
}

func effectiveBaseURL(domain string) string {
	return fibe.NewClient(
		fibe.WithDisableAutoConfig(),
		fibe.WithDomain(normalizeDomainInput(domain)),
	).BaseURL()
}
