package fibe

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// CredentialEntry holds a stored API key for a specific domain.
type CredentialEntry struct {
	APIKey   string `json:"api_key"`
	APIKeyID int64  `json:"api_key_id,omitempty"`
	Domain   string `json:"domain"`
}

// CredentialStore manages persistent CLI credentials keyed by domain.
// File layout: {"domains": {"fibe.gg": {...}, "next.fibe.live": {...}}}
type CredentialStore struct {
	path string
}

type credentialFile struct {
	Domains map[string]*CredentialEntry `json:"domains"`
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

func (s *CredentialStore) load() (*credentialFile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var f credentialFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Domains == nil {
		f.Domains = make(map[string]*CredentialEntry)
	}
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
