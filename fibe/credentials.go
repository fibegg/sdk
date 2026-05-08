package fibe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// CredentialEntry holds a stored API key for a domain and, for modern CLI
// auth, an optional named profile.
type CredentialEntry struct {
	APIKey   string `json:"api_key"`
	APIKeyID int64  `json:"api_key_id,omitempty"`
	Domain   string `json:"domain"`
	Profile  string `json:"profile,omitempty"`
}

// CredentialStore manages persistent CLI credentials.
// File layout:
// {"profiles": {"default": {...}}, "domains": {"fibe.gg": {...}}}
//
// The domains map is kept for backward compatibility with older CLIs and SDK
// ambient credential lookup. New CLI auth should prefer profile methods.
type CredentialStore struct {
	path string
}

type credentialFile struct {
	Domains  map[string]*CredentialEntry `json:"domains,omitempty"`
	Profiles map[string]*CredentialEntry `json:"profiles,omitempty"`
}

// DefaultCredentialPath returns $XDG_CONFIG_HOME/fibe/credentials.json,
// falling back to ~/.config/fibe/credentials.json.
// This ensures consistent cross-platform behavior (macOS + Linux).
func DefaultCredentialPath() string {
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		cfgDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(cfgDir, "fibe", "credentials.json")
}

// NewCredentialStore opens or creates a credential store at the given path.
func NewCredentialStore(path string) *CredentialStore {
	return &CredentialStore{path: path}
}

// Get returns the stored credential for the given domain, or nil.
func (s *CredentialStore) Get(domain string) (*CredentialEntry, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	return f.Domains[domain], nil
}

// GetProfile returns the stored credential for a named profile.
// If the credential file only has legacy domain-keyed entries, it imports the
// best available legacy entry in memory without rewriting the file.
func (s *CredentialStore) GetProfile(profile string) (*CredentialEntry, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	if entry := f.Profiles[profile]; entry != nil {
		out := *entry
		out.Profile = profile
		return &out, nil
	}
	if entry := legacyProfileEntry(f, profile); entry != nil {
		return entry, nil
	}
	return nil, nil
}

// Set stores a credential for the given domain, creating the file if needed.
func (s *CredentialStore) Set(entry *CredentialEntry) error {
	f, err := s.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if f == nil {
		f = &credentialFile{Domains: make(map[string]*CredentialEntry)}
	}
	f.Domains[entry.Domain] = entry
	return s.save(f)
}

// SetProfile stores a credential for a named profile and mirrors it by domain
// for older SDK/CLI callers that still resolve credentials by domain.
func (s *CredentialStore) SetProfile(profile string, entry *CredentialEntry) error {
	f, err := s.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if f == nil {
		f = &credentialFile{}
	}
	ensureCredentialMaps(f)
	copy := *entry
	copy.Profile = profile
	f.Profiles[profile] = &copy
	if copy.Domain != "" {
		f.Domains[copy.Domain] = &copy
	}
	return s.save(f)
}

// Delete removes the credential for the given domain.
func (s *CredentialStore) Delete(domain string) error {
	f, err := s.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	delete(f.Domains, domain)
	return s.save(f)
}

// DeleteProfile removes the credential for a named profile. It removes the
// mirrored domain entry only when it points at the same API key.
func (s *CredentialStore) DeleteProfile(profile string) error {
	f, err := s.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	entry := f.Profiles[profile]
	if entry == nil {
		delete(f.Profiles, profile)
		return s.save(f)
	}
	delete(f.Profiles, profile)
	if domainEntry := f.Domains[entry.Domain]; domainEntry != nil && domainEntry.APIKey == entry.APIKey {
		delete(f.Domains, entry.Domain)
	}
	return s.save(f)
}

// List returns all stored credentials.
func (s *CredentialStore) List() (map[string]*CredentialEntry, error) {
	f, err := s.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return f.Domains, nil
}

// ListProfiles returns profile-keyed credentials, importing legacy
// domain-keyed entries in memory when no explicit profile exists.
func (s *CredentialStore) ListProfiles() (map[string]*CredentialEntry, error) {
	f, err := s.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make(map[string]*CredentialEntry, len(f.Profiles)+len(f.Domains))
	for profile, entry := range f.Profiles {
		copy := *entry
		copy.Profile = profile
		out[profile] = &copy
	}
	for domain, entry := range f.Domains {
		profile := legacyProfileName(domain)
		if _, exists := out[profile]; exists {
			continue
		}
		copy := *entry
		copy.Domain = domain
		copy.Profile = profile
		out[profile] = &copy
	}
	return out, nil
}

func (s *CredentialStore) load() (*credentialFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var f credentialFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	ensureCredentialMaps(&f)
	return &f, nil
}

func (s *CredentialStore) save(f *credentialFile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func ensureCredentialMaps(f *credentialFile) {
	if f.Domains == nil {
		f.Domains = make(map[string]*CredentialEntry)
	}
	if f.Profiles == nil {
		f.Profiles = make(map[string]*CredentialEntry)
	}
}

func legacyProfileEntry(f *credentialFile, profile string) *CredentialEntry {
	if profile == "default" {
		if entry := f.Domains["fibe.gg"]; entry != nil {
			copy := *entry
			copy.Domain = "fibe.gg"
			copy.Profile = profile
			return &copy
		}
	}
	if entry := f.Domains[profile]; entry != nil {
		copy := *entry
		copy.Domain = profile
		copy.Profile = profile
		return &copy
	}
	return nil
}

func legacyProfileName(domain string) string {
	if domain == "fibe.gg" {
		return "default"
	}
	return domain
}
