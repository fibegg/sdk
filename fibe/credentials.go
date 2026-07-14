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
	return withStoreLock(s.path, func() (*CredentialEntry, error) {
		f, err := s.loadUnlocked()
		if err != nil {
			return nil, err
		}
		return cloneCredential(f.Domains[domain]), nil
	})
}

// GetProfile returns the stored credential for a named profile.
// If the credential file only has legacy domain-keyed entries, it imports the
// best available legacy entry in memory without rewriting the file.
func (s *CredentialStore) GetProfile(profile string) (*CredentialEntry, error) {
	return withStoreLock(s.path, func() (*CredentialEntry, error) {
		f, err := s.loadUnlocked()
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
	})
}

// Set stores a credential for the given domain, creating the file if needed.
func (s *CredentialStore) Set(entry *CredentialEntry) error {
	return withStoreLockError(s.path, func() error {
		f, err := s.loadUnlocked()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if f == nil {
			f = &credentialFile{Domains: make(map[string]*CredentialEntry)}
		}
		f.Domains[entry.Domain] = cloneCredential(entry)
		return s.saveUnlocked(f)
	})
}

// SetProfile stores a credential for a named profile and mirrors it by domain
// for older SDK/CLI callers that still resolve credentials by domain.
func (s *CredentialStore) SetProfile(profile string, entry *CredentialEntry) error {
	return withStoreLockError(s.path, func() error {
		f, err := s.loadUnlocked()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if f == nil {
			f = &credentialFile{}
		}
		ensureCredentialMaps(f)
		cloned := *entry
		cloned.Profile = profile
		f.Profiles[profile] = &cloned
		if cloned.Domain != "" {
			f.Domains[cloned.Domain] = &cloned
		}
		return s.saveUnlocked(f)
	})
}

// Delete removes the credential for the given domain.
func (s *CredentialStore) Delete(domain string) error {
	return withStoreLockError(s.path, func() error {
		f, err := s.loadUnlocked()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		delete(f.Domains, domain)
		return s.saveUnlocked(f)
	})
}

// DeleteProfile removes the credential for a named profile. It removes the
// mirrored domain entry only when it points at the same API key.
func (s *CredentialStore) DeleteProfile(profile string) error {
	return withStoreLockError(s.path, func() error {
		f, err := s.loadUnlocked()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		entry := f.Profiles[profile]
		if entry == nil {
			delete(f.Profiles, profile)
			return s.saveUnlocked(f)
		}
		delete(f.Profiles, profile)
		if domainEntry := f.Domains[entry.Domain]; domainEntry != nil && domainEntry.APIKey == entry.APIKey {
			delete(f.Domains, entry.Domain)
		}
		return s.saveUnlocked(f)
	})
}

// List returns all stored credentials.
func (s *CredentialStore) List() (map[string]*CredentialEntry, error) {
	return withStoreLock(s.path, func() (map[string]*CredentialEntry, error) {
		f, err := s.loadUnlocked()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		return cloneCredentialMap(f.Domains), nil
	})
}

// ListProfiles returns profile-keyed credentials, importing legacy
// domain-keyed entries in memory when no explicit profile exists.
func (s *CredentialStore) ListProfiles() (map[string]*CredentialEntry, error) {
	return withStoreLock(s.path, func() (map[string]*CredentialEntry, error) {
		f, err := s.loadUnlocked()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		out := make(map[string]*CredentialEntry, len(f.Profiles)+len(f.Domains))
		for profile, entry := range f.Profiles {
			cloned := *entry
			cloned.Profile = profile
			out[profile] = &cloned
		}
		for domain, entry := range f.Domains {
			profile := legacyProfileName(domain)
			if _, exists := out[profile]; exists {
				continue
			}
			cloned := *entry
			cloned.Domain = domain
			cloned.Profile = profile
			out[profile] = &cloned
		}
		return out, nil
	})
}

func (s *CredentialStore) loadUnlocked() (*credentialFile, error) {
	data, err := readStoreFile(s.path)
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

func (s *CredentialStore) saveUnlocked(f *credentialFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(s.path, append(data, '\n'), 0o600)
}

func cloneCredential(entry *CredentialEntry) *CredentialEntry {
	if entry == nil {
		return nil
	}
	cloned := *entry
	return &cloned
}

func cloneCredentialMap(entries map[string]*CredentialEntry) map[string]*CredentialEntry {
	out := make(map[string]*CredentialEntry, len(entries))
	for key, entry := range entries {
		out[key] = cloneCredential(entry)
	}
	return out
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
			cloned := *entry
			cloned.Domain = "fibe.gg"
			cloned.Profile = profile
			return &cloned
		}
	}
	if entry := f.Domains[profile]; entry != nil {
		cloned := *entry
		cloned.Domain = profile
		cloned.Profile = profile
		return &cloned
	}
	return nil
}

func legacyProfileName(domain string) string {
	if domain == "fibe.gg" {
		return "default"
	}
	return domain
}
