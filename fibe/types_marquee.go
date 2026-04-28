package fibe

import "time"

type Marquee struct {
	ID                   int64     `json:"id"`
	Name                 string    `json:"name"`
	Host                 string    `json:"host"`
	Port                 int       `json:"port"`
	User                 string    `json:"user"`
	Status               string    `json:"status"`
	DomainsInput         *string   `json:"domains_input"`
	AcmeEmail            *string   `json:"acme_email"`
	DockerhubAuthEnabled *bool     `json:"dockerhub_auth_enabled"`
	BuildPlatform        *string   `json:"build_platform"`
	PropID               *int64    `json:"prop_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type MarqueeCreateParams struct {
	Name                 string            `json:"name"`
	Host                 string            `json:"host"`
	Port                 int               `json:"port"`
	User                 string            `json:"user"`
	SSHPrivateKey        string            `json:"ssh_private_key"`
	DomainsInput         *string           `json:"domains_input,omitempty"`
	AcmeEmail            *string           `json:"acme_email,omitempty"`
	DockerhubAuthEnabled *bool             `json:"dockerhub_auth_enabled,omitempty"`
	DockerhubUsername    *string           `json:"dockerhub_username,omitempty"`
	DockerhubToken       *string           `json:"dockerhub_token,omitempty"`
	BuildPlatform        *string           `json:"build_platform,omitempty"`
	PropID               *int64            `json:"prop_id,omitempty"`
	Status               *string           `json:"status,omitempty"`
	DnsProvider          *string           `json:"dns_provider,omitempty"`
	DnsCredentials       map[string]string `json:"dns_credentials,omitempty"`
}

func (p *MarqueeCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	v.required("host", p.Host)
	v.required("user", p.User)
	v.required("ssh_private_key", p.SSHPrivateKey)
	v.port("port", p.Port)
	if p.DockerhubAuthEnabled != nil && *p.DockerhubAuthEnabled {
		if p.DockerhubUsername == nil || *p.DockerhubUsername == "" {
			v.errors = append(v.errors, ValidationError{Field: "dockerhub_username", Message: "required when dockerhub_auth_enabled is true"})
		}
		if p.DockerhubToken == nil || *p.DockerhubToken == "" {
			v.errors = append(v.errors, ValidationError{Field: "dockerhub_token", Message: "required when dockerhub_auth_enabled is true"})
		}
	}
	return v.err()
}

type MarqueeUpdateParams struct {
	Name                 *string           `json:"name,omitempty"`
	Host                 *string           `json:"host,omitempty"`
	Port                 *int              `json:"port,omitempty"`
	User                 *string           `json:"user,omitempty"`
	SSHPrivateKey        *string           `json:"ssh_private_key,omitempty"`
	DomainsInput         *string           `json:"domains_input,omitempty"`
	AcmeEmail            *string           `json:"acme_email,omitempty"`
	DockerhubAuthEnabled *bool             `json:"dockerhub_auth_enabled,omitempty"`
	DockerhubUsername    *string           `json:"dockerhub_username,omitempty"`
	DockerhubToken       *string           `json:"dockerhub_token,omitempty"`
	BuildPlatform        *string           `json:"build_platform,omitempty"`
	PropID               *int64            `json:"prop_id,omitempty"`
	Status               *string           `json:"status,omitempty"`
	DnsProvider          *string           `json:"dns_provider,omitempty"`
	DnsCredentials       map[string]string `json:"dns_credentials,omitempty"`
}

// AutoconnectTokenParams for generating a marquee autoconnect token.
type AutoconnectTokenParams struct {
	Email          string            `json:"email,omitempty"`
	Domain         string            `json:"domain,omitempty"`
	IP             string            `json:"ip,omitempty"`
	SSLMode        string            `json:"ssl_mode,omitempty"`
	DnsProvider    string            `json:"dns_provider,omitempty"`
	DnsCredentials map[string]string `json:"dns_credentials,omitempty"`
}

type AutoconnectTokenResult struct {
	Token string `json:"token"`
}

type SSHKeyResult struct {
	PublicKey string `json:"public_key"`
}

type ConnectionTestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type MarqueeListParams struct {
	Q             string `url:"q,omitempty"`
	Status        string `url:"status,omitempty"`
	Name          string `url:"name,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}
