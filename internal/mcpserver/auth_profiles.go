package mcpserver

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/fibegg/sdk/fibe"
)

type mcpAuthProfile struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	BaseURL    string `json:"base_url"`
	Active     bool   `json:"active"`
	HasKey     bool   `json:"has_key"`
	APIKeyID   int64  `json:"api_key_id,omitempty"`
	MaskedKey  string `json:"masked_key,omitempty"`
	CredSource string `json:"credential_source,omitempty"`
}

func listMCPAuthProfiles() ([]mcpAuthProfile, error) {
	store := fibe.NewAuthProfileStore(fibe.DefaultAuthProfilePath())
	cfg, err := store.Load()
	if err != nil {
		return nil, err
	}
	creds, err := fibe.NewCredentialStore(fibe.DefaultCredentialPath()).ListProfiles()
	if err != nil {
		return nil, err
	}

	names := map[string]bool{fibe.DefaultProfileName: true}
	if cfg.ActiveProfile != "" {
		names[cfg.ActiveProfile] = true
	}
	for name := range cfg.Profiles {
		names[name] = true
	}
	for name := range creds {
		names[name] = true
	}

	out := make([]mcpAuthProfile, 0, len(names))
	active := cfg.ActiveProfile
	if active == "" {
		active = fibe.DefaultProfileName
	}
	for name := range names {
		domain := ""
		if p, ok := cfg.Profiles[name]; ok {
			domain = p.Domain
		}
		cred := creds[name]
		if domain == "" && cred != nil {
			domain = cred.Domain
		}
		if domain == "" {
			domain = fibe.DefaultProfileDomain
		}
		row := mcpAuthProfile{
			Name:    name,
			Domain:  normalizeMCPDomain(domain),
			BaseURL: mcpBaseURL(domain),
			Active:  name == active,
		}
		if cred != nil && cred.APIKey != "" {
			row.HasKey = true
			row.APIKeyID = cred.APIKeyID
			row.MaskedKey = maskMCPKey(cred.APIKey)
			row.CredSource = "credentials"
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Active != out[j].Active {
			return out[i].Active
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func resolveMCPAuthProfile(profile string) (domain string, apiKey string, apiKeyID int64, ok bool, err error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "", "", 0, false, fmt.Errorf("profile is required")
	}
	store := fibe.NewAuthProfileStore(fibe.DefaultAuthProfilePath())
	domain, hasConfig := store.ProfileDomain(profile)
	entry, credErr := fibe.NewCredentialStore(fibe.DefaultCredentialPath()).GetProfile(profile)
	if credErr != nil && !os.IsNotExist(credErr) {
		return "", "", 0, false, credErr
	}
	if domain == "" && entry != nil {
		domain = entry.Domain
	}
	if domain == "" && profile == fibe.DefaultProfileName {
		domain = fibe.DefaultProfileDomain
	}
	if entry != nil {
		apiKey = entry.APIKey
		apiKeyID = entry.APIKeyID
	}
	ok = hasConfig || entry != nil || profile == fibe.DefaultProfileName
	return normalizeMCPDomain(domain), apiKey, apiKeyID, ok, nil
}

func normalizeMCPDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fibe.DefaultProfileDomain
	}
	domain = strings.TrimRight(domain, "/")
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		if u, err := url.Parse(domain); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return domain
}

func mcpBaseURL(domain string) string {
	return fibe.NewClient(
		fibe.WithDisableAutoConfig(),
		fibe.WithDomain(normalizeMCPDomain(domain)),
	).BaseURL()
}

func maskMCPKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "***" + key[len(key)-4:]
}
