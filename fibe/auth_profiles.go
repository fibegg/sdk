package fibe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

const (
	DefaultProfileName   = "default"
	DefaultProfileDomain = "fibe.gg"
)

type AuthProfile struct {
	Domain string `json:"domain"`
}

type AuthProfileConfig struct {
	ActiveProfile string                 `json:"active_profile,omitempty"`
	Profiles      map[string]AuthProfile `json:"profiles,omitempty"`
}

type AuthProfileStore struct {
	path string
}

func DefaultAuthProfilePath() string {
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(cfgDir, "fibe", "config.json")
}

func NewAuthProfileStore(path string) *AuthProfileStore {
	return &AuthProfileStore{path: path}
}

func (s *AuthProfileStore) Load() (*AuthProfileConfig, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &AuthProfileConfig{Profiles: map[string]AuthProfile{}}, nil
		}
		return nil, err
	}
	var cfg AuthProfileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]AuthProfile{}
	}
	return &cfg, nil
}

func (s *AuthProfileStore) Save(cfg *AuthProfileConfig) error {
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]AuthProfile{}
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

func (s *AuthProfileStore) SetProfile(profile, domain string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	cfg.Profiles[profile] = AuthProfile{Domain: domain}
	return s.Save(cfg)
}

func (s *AuthProfileStore) SetActive(profile string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	cfg.ActiveProfile = profile
	return s.Save(cfg)
}

func (s *AuthProfileStore) DeleteProfile(profile string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	delete(cfg.Profiles, profile)
	if cfg.ActiveProfile == profile {
		cfg.ActiveProfile = ""
	}
	return s.Save(cfg)
}

func (s *AuthProfileStore) ActiveProfile() string {
	cfg, err := s.Load()
	if err == nil && cfg.ActiveProfile != "" {
		return cfg.ActiveProfile
	}
	return DefaultProfileName
}

func (s *AuthProfileStore) ProfileDomain(profile string) (string, bool) {
	cfg, err := s.Load()
	if err != nil {
		return "", false
	}
	p, ok := cfg.Profiles[profile]
	return p.Domain, ok
}

func (s *AuthProfileStore) ProfileNames() []string {
	cfg, err := s.Load()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(cfg.Profiles)+1)
	seen := map[string]bool{}
	for name := range cfg.Profiles {
		names = append(names, name)
		seen[name] = true
	}
	if cfg.ActiveProfile != "" && !seen[cfg.ActiveProfile] {
		names = append(names, cfg.ActiveProfile)
		seen[cfg.ActiveProfile] = true
	}
	if !seen[DefaultProfileName] {
		names = append(names, DefaultProfileName)
	}
	sort.Strings(names)
	return names
}
