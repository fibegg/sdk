package fibe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func listEnv[T any](items []T) listEnvelope[T] {
	return listEnvelope[T]{
		Data: items,
		Meta: ListMeta{Page: 1, PerPage: 25, Total: int64(len(items))},
	}
}

func testAsyncAcceptedEndpoint(t *testing.T, postPath, statusPath string, finalPayload map[string]any) (*Client, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != postPath {
				t.Errorf("unexpected initial request %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-async",
				"status":     "queued",
			})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != statusPath {
				t.Errorf("unexpected status request %s %s", r.Method, r.URL.Path)
			}
			payload := map[string]any{
				"request_id": "req-async",
				"status":     "success",
			}
			for key, value := range finalPayload {
				payload[key] = value
			}
			json.NewEncoder(w).Encode(payload)
		default:
			t.Errorf("unexpected extra request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	return c, &calls
}

func TestPlaygrounds_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/playgrounds" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(listEnv([]Playground{
			{ID: 1, Name: "pg-1", Status: "running"},
			{ID: 2, Name: "pg-2", Status: "pending"},
		}))
	})

	result, err := c.Playgrounds.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 playgrounds, got %d", len(result.Data))
	}
	if result.Data[0].Name != "pg-1" {
		t.Errorf("expected name 'pg-1', got %q", result.Data[0].Name)
	}
	if result.Meta.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Meta.Total)
	}
}

func TestPlaygroundActionValidationIncludesMaintenanceActions(t *testing.T) {
	for _, action := range []string{PlaygroundActionEnableMaintenance, PlaygroundActionDisableMaintenance} {
		if err := (&PlaygroundActionParams{ActionType: action}).Validate(); err != nil {
			t.Fatalf("expected %s to validate: %v", action, err)
		}
	}
}

func TestPlaygrounds_Get(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playgrounds/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Playground{ID: 42, Name: "test", Status: "running"})
	})

	pg, err := c.Playgrounds.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 42 {
		t.Errorf("expected ID 42, got %d", pg.ID)
	}
}

func TestPlaygrounds_StatusByIdentifierUsesName(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/playgrounds/next/status" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":            129,
			"status":        "running",
			"state_reasons": []string{"Playguard repair: remote build script missing"},
			"build_statuses": []map[string]any{
				{
					"service_name": "web",
					"branch":       "main",
					"latest": map[string]any{
						"id":               7,
						"status":           "building",
						"commit_sha":       "abcdef1234567890",
						"short_commit_sha": "abcdef1",
					},
				},
			},
		})
	})

	status, err := c.Playgrounds.StatusByIdentifier(context.Background(), "next")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ID != 129 || status.Status != "running" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if len(status.StateReasons) != 1 || status.StateReasons[0] == "" {
		t.Fatalf("missing state reasons: %#v", status)
	}
	if len(status.BuildStatuses) != 1 || status.BuildStatuses[0].Latest == nil || status.BuildStatuses[0].Latest.CommitSHA != "abcdef1234567890" {
		t.Fatalf("missing build status: %#v", status.BuildStatuses)
	}
}

func TestPlaygrounds_WaitForStatusByIdentifier(t *testing.T) {
	var calls atomic.Int32
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/next/status":
			status := "starting"
			if calls.Add(1) >= 2 {
				status = "running"
			}
			json.NewEncoder(w).Encode(PlaygroundStatus{ID: 42, Status: status})
		case "/api/playgrounds/next":
			json.NewEncoder(w).Encode(Playground{ID: 42, Name: "next", Status: "running"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	pg, err := c.Playgrounds.WaitForStatusByIdentifier(context.Background(), "next", "running", time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 42 || calls.Load() != 2 {
		t.Fatalf("pg=%#v calls=%d, want id 42 after two status polls", pg, calls.Load())
	}
}

func TestTricks_GetByIdentifierUsesName(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/playgrounds/nightly-build" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(Playground{ID: 77, Name: "nightly-build", Status: "completed"})
	})

	trick, err := c.Tricks.GetByIdentifier(context.Background(), "nightly-build")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trick.ID != 77 || trick.Name != "nightly-build" {
		t.Fatalf("unexpected trick: %#v", trick)
	}
}

func TestPlaygrounds_Create(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		pg := body["playground"].(map[string]any)
		if pg["name"] != "new-pg" {
			t.Errorf("expected name 'new-pg', got %v", pg["name"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(Playground{ID: 99, Name: "new-pg", Status: "pending"})
	})

	pg, err := c.Playgrounds.Create(context.Background(), &PlaygroundCreateParams{
		Name:       "new-pg",
		PlayspecID: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 99 {
		t.Errorf("expected ID 99, got %d", pg.ID)
	}
}

func TestGreenfield_Create(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/greenfields" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(GreenfieldResult{
			Name:        "tower-defence",
			GitProvider: "gitea",
			Playground:  &Playground{ID: 77, Name: "tower-defence", Status: "pending"},
		})
	})

	marqueeID := int64(12)
	templateVersionID := int64(912)
	result, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{
		Name:              "tower-defence",
		TemplateBody:      "services:\n  web:\n    image: nginx\n",
		GitProvider:       "github",
		MarqueeID:         &marqueeID,
		TemplateVersionID: &templateVersionID,
		Variables:         map[string]any{"app_name": "Tower"},
	})
	if err == nil {
		t.Fatal("expected template_body/template_version_id validation error")
	}

	result, err = c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{
		Name:              "tower-defence",
		TemplateVersionID: &templateVersionID,
		GitProvider:       "github",
		MarqueeID:         &marqueeID,
		Variables:         map[string]any{"app_name": "Tower"},
		ServiceSubdomains: map[string]string{"app": "tower", "admin": "tower-admin"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Playground == nil || result.Playground.ID != 77 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if body["name"] != "tower-defence" || body["git_provider"] != "github" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if body["marquee_id"].(float64) != 12 {
		t.Fatalf("unexpected marquee id in body: %#v", body)
	}
	if body["template_version_id"].(float64) != 912 {
		t.Fatalf("unexpected template version id in body: %#v", body)
	}
	vars := body["variables"].(map[string]any)
	if vars["app_name"] != "Tower" {
		t.Fatalf("unexpected variables: %#v", vars)
	}
	serviceSubdomains := body["service_subdomains"].(map[string]any)
	if serviceSubdomains["app"] != "tower" || serviceSubdomains["admin"] != "tower-admin" {
		t.Fatalf("unexpected service_subdomains: %#v", serviceSubdomains)
	}
}

func TestGreenfield_CreateRejectsVersionWithoutTemplateID(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called")
	})

	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{Name: "todo", Version: "v1"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGreenfield_CreateRejectsTemplateBodyWithTemplateID(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called")
	})

	templateID := int64(347)
	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{Name: "todo", TemplateID: &templateID, TemplateBody: "services: {}\n"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestGreenfield_CreateWithRepositoryURL(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/greenfields" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(GreenfieldResult{Name: "repo", Playground: &Playground{ID: 77}})
	})

	marqueeID := int64(12)
	installationID := int64(123)
	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{
		RepositoryURL:        "https://github.com/owner/repo",
		ConfigPath:           "deploy/fibe.yml",
		GitHubRef:            "main",
		GitHubInstallationID: &installationID,
		MarqueeID:            &marqueeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["repository_url"] != "https://github.com/owner/repo" || body["config_path"] != "deploy/fibe.yml" || body["github_ref"] != "main" {
		t.Fatalf("unexpected repository fields: %#v", body)
	}
	if body["github_installation_id"].(float64) != 123 {
		t.Fatalf("unexpected github_installation_id: %#v", body)
	}
}

func TestGreenfield_CreateWithTemplateID(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/greenfields" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(GreenfieldResult{Name: "todo", Playground: &Playground{ID: 77}})
	})

	marqueeID := int64(12)
	templateID := int64(347)
	_, err := c.Greenfield.Create(context.Background(), &GreenfieldCreateParams{
		Name:        "todo",
		TemplateID:  &templateID,
		Version:     "v1",
		GitProvider: "gitea",
		MarqueeID:   &marqueeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["template_id"].(float64) != 347 || body["version"] != "v1" {
		t.Fatalf("unexpected template fields: %#v", body)
	}
}

func TestGitHubApps_ConnectInfo(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/github_apps/connect" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(GitHubAppConnectInfo{AppSlug: "fibe", InstallURL: "https://github.com/apps/fibe/installations/new"})
	})

	info, err := c.GitHubApps.ConnectInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.InstallURL == "" || info.AppSlug != "fibe" {
		t.Fatalf("unexpected info: %#v", info)
	}
}

func TestGiteaRepos_CreateSurfacesProp(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/gitea_repositories" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":             123,
			"name":           "bagg-app",
			"full_name":      "viktorvsk/bagg-app",
			"html_url":       "https://git-next.fibe.live/viktorvsk/bagg-app",
			"clone_url":      "https://git-next.fibe.live/viktorvsk/bagg-app.git",
			"default_branch": "main",
			"repo": map[string]any{
				"id":             123,
				"name":           "bagg-app",
				"full_name":      "viktorvsk/bagg-app",
				"html_url":       "https://git-next.fibe.live/viktorvsk/bagg-app",
				"clone_url":      "https://git-next.fibe.live/viktorvsk/bagg-app.git",
				"default_branch": "main",
			},
			"prop_id": 456,
			"prop": map[string]any{
				"id":             456,
				"name":           "bagg-app",
				"repository_url": "https://git-next.fibe.live/viktorvsk/bagg-app",
				"provider":       "gitea",
			},
		})
	})

	private := true
	result, err := c.GiteaRepos.Create(context.Background(), &GiteaRepoCreateParams{Name: "bagg-app", Private: &private})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["name"] != "bagg-app" || body["private"] != true {
		t.Fatalf("unexpected body: %#v", body)
	}
	if result.PropID != 456 || result.Prop == nil || result.Prop.ID != 456 {
		t.Fatalf("expected prop in result, got %#v", result)
	}
	if result.Repo == nil || result.Repo.HTMLURL != "https://git-next.fibe.live/viktorvsk/bagg-app" {
		t.Fatalf("expected nested repo in result, got %#v", result)
	}
	if result.DefaultBranch != "main" {
		t.Fatalf("default branch=%q", result.DefaultBranch)
	}
}

func TestPlaygrounds_Action(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/playgrounds/42/operations" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		json.NewEncoder(w).Encode(PlaygroundStatus{ID: 42, Status: "pending"})
	})

	force := true
	pg, err := c.Playgrounds.Action(context.Background(), 42, &PlaygroundActionParams{
		ActionType: PlaygroundActionRollout,
		Force:      &force,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", pg.Status)
	}
	if body["action_type"] != PlaygroundActionRollout {
		t.Fatalf("expected action_type=%q, got %#v", PlaygroundActionRollout, body)
	}
	if body["force"] != true {
		t.Fatalf("expected force=true, got %#v", body)
	}
}

func TestMarquees_UpdateSerializesDnsCredentialsForServer(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/api/marquees/1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		json.NewEncoder(w).Encode(Marquee{ID: 1, Name: "Elastic"})
	})

	provider := "cloudflare"
	_, err := c.Marquees.Update(context.Background(), 1, &MarqueeUpdateParams{
		DnsProvider:    &provider,
		DnsCredentials: map[string]string{"CF_DNS_API_TOKEN": "secret-token"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marquee := body["marquee"].(map[string]any)
	raw, ok := marquee["dns_credentials"].(string)
	if !ok {
		t.Fatalf("dns_credentials = %T (%#v), want JSON string", marquee["dns_credentials"], marquee["dns_credentials"])
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("dns_credentials should contain JSON object text: %v", err)
	}
	if decoded["CF_DNS_API_TOKEN"] != "secret-token" {
		t.Fatalf("unexpected dns_credentials payload: %#v", decoded)
	}
}

func TestMarquees_UpdateSerializesHTTPSModeAndProvidedTLS(t *testing.T) {
	var body map[string]any
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/api/marquees/1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		enabled := true
		source := "provided"
		json.NewEncoder(w).Encode(Marquee{ID: 1, Name: "Elastic", HttpsEnabled: &enabled, TlsCertificateSource: &source})
	})

	enabled := true
	source := "provided"
	cert := "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----"
	key := "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----"
	result, err := c.Marquees.Update(context.Background(), 1, &MarqueeUpdateParams{
		HttpsEnabled:         &enabled,
		TlsCertificateSource: &source,
		TlsCertificatePEM:    &cert,
		TlsPrivateKeyPEM:     &key,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	marquee := body["marquee"].(map[string]any)
	if marquee["https_enabled"] != true {
		t.Fatalf("https_enabled = %#v, want true", marquee["https_enabled"])
	}
	if marquee["tls_certificate_source"] != "provided" {
		t.Fatalf("tls_certificate_source = %#v", marquee["tls_certificate_source"])
	}
	if marquee["tls_certificate_pem"] != cert {
		t.Fatalf("tls_certificate_pem was not serialized")
	}
	if marquee["tls_private_key_pem"] != key {
		t.Fatalf("tls_private_key_pem was not serialized")
	}
	if result.TlsCertificateSource == nil || *result.TlsCertificateSource != "provided" {
		t.Fatalf("unexpected TLS source in response: %#v", result.TlsCertificateSource)
	}
}

func TestMarqueeDecodesBillingRuntimeFields(t *testing.T) {
	raw := []byte(`{
		"id": 42,
		"name": "runtime",
		"paid_until": "2026-05-22T23:59:59Z",
		"billing_requested_until": "2026-05-25T23:59:59Z",
		"billing_runtime_active": true,
		"chat_launchable": true
	}`)
	var marquee Marquee
	if err := json.Unmarshal(raw, &marquee); err != nil {
		t.Fatalf("unmarshal marquee: %v", err)
	}
	if marquee.PaidUntil == nil || marquee.PaidUntil.UTC().Format(time.RFC3339) != "2026-05-22T23:59:59Z" {
		t.Fatalf("unexpected paid_until: %#v", marquee.PaidUntil)
	}
	if marquee.BillingRequestedUntil == nil || marquee.BillingRequestedUntil.UTC().Format(time.RFC3339) != "2026-05-25T23:59:59Z" {
		t.Fatalf("unexpected billing_requested_until: %#v", marquee.BillingRequestedUntil)
	}
	if !marquee.BillingRuntimeActive || !marquee.ChatLaunchable {
		t.Fatalf("expected runtime booleans true, got billing_runtime_active=%v chat_launchable=%v", marquee.BillingRuntimeActive, marquee.ChatLaunchable)
	}
}

func TestMarquees_GenerateSSHKeyPollsAccepted(t *testing.T) {
	c, calls := testAsyncAcceptedEndpoint(
		t,
		"/api/marquees/5/ssh_keys",
		"/api/async_requests/req-async",
		map[string]any{"public_key": "ssh-ed25519 AAAA test"},
	)

	result, err := c.Marquees.GenerateSSHKey(context.Background(), 5)
	if err != nil {
		t.Fatalf("generate ssh key: %v", err)
	}
	if result.PublicKey != "ssh-ed25519 AAAA test" {
		t.Fatalf("unexpected ssh key result: %#v", result)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", calls.Load())
	}
}

func TestImportTemplates_RefreshSourcePollsAccepted(t *testing.T) {
	c, calls := testAsyncAcceptedEndpoint(
		t,
		"/api/import_templates/12/source",
		"/api/async_requests/req-async",
		map[string]any{"success": true, "version_created": true},
	)

	result, err := c.ImportTemplates.RefreshSource(context.Background(), 12)
	if err != nil {
		t.Fatalf("refresh source: %v", err)
	}
	if !result.Success || !result.VersionCreated {
		t.Fatalf("unexpected refresh result: %#v", result)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", calls.Load())
	}
}

func TestImportTemplates_UpgradeLinkedPlayspecsPollsAccepted(t *testing.T) {
	c, calls := testAsyncAcceptedEndpoint(
		t,
		"/api/import_templates/12/versions/34/upgrades",
		"/api/async_requests/req-async",
		map[string]any{"success": true, "upgraded_count": 2, "failed_count": 0},
	)

	result, err := c.ImportTemplates.UpgradeLinkedPlayspecs(context.Background(), 12, 34)
	if err != nil {
		t.Fatalf("upgrade linked playspecs: %v", err)
	}
	if !result.Success || result.UpgradedCount != 2 || result.FailedCount != 0 {
		t.Fatalf("unexpected upgrade result: %#v", result)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", calls.Load())
	}
}

func TestPlayspecs_SwitchTemplateVersionPollsAccepted(t *testing.T) {
	c, calls := testAsyncAcceptedEndpoint(
		t,
		"/api/playspecs/9/template_switches",
		"/api/async_requests/req-async",
		map[string]any{"no_op": true, "suggested_upgrade": true},
	)

	result, err := c.Playspecs.SwitchTemplateVersion(context.Background(), 9, &PlayspecTemplateVersionSwitchParams{TargetTemplateVersionID: 2})
	if err != nil {
		t.Fatalf("switch template version: %v", err)
	}
	if !result.NoOp || !result.SuggestedUpgrade {
		t.Fatalf("unexpected switch result: %#v", result)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", calls.Load())
	}
}

func TestVerifyTemplateVersionSwitchResult(t *testing.T) {
	targetID := int64(42)
	result := &PlayspecTemplateVersionSwitchResult{
		TargetTemplateVersion: &TemplateVersionRef{ID: &targetID},
		Playspec:              &Playspec{SourceTemplateVersionID: &targetID},
	}

	if err := VerifyTemplateVersionSwitchResult(result, targetID); err != nil {
		t.Fatalf("expected valid switch result: %v", err)
	}

	if err := VerifyTemplateVersionSwitchResult(&PlayspecTemplateVersionSwitchResult{}, targetID); err == nil || !strings.Contains(err.Error(), "target_template_version") {
		t.Fatalf("expected missing target error, got %v", err)
	}

	wrongID := int64(41)
	result.Playspec.SourceTemplateVersionID = &wrongID
	if err := VerifyTemplateVersionSwitchResult(result, targetID); err == nil || !strings.Contains(err.Error(), "did not apply target version") {
		t.Fatalf("expected source version mismatch error, got %v", err)
	}
}

func TestMemories_MemorizePollsAccepted(t *testing.T) {
	c, calls := testAsyncAcceptedEndpoint(
		t,
		"/api/memories",
		"/api/async_requests/req-async",
		map[string]any{"counts": map[string]any{"created": 1}},
	)

	result, err := c.Memories.Memorize(context.Background(), map[string]any{"conversation_id": "thread-1"})
	if err != nil {
		t.Fatalf("memorize: %v", err)
	}
	if result.Status != "success" || result.Counts["created"] != 1 {
		t.Fatalf("unexpected memorize result: %#v", result)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 requests, got %d", calls.Load())
	}
}

func TestPlaygrounds_DebugWithParams(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/playgrounds/42/debug" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("mode") != "summary" || query.Get("refresh") != "true" || query.Get("service") != "web" || query.Get("logs_tail") != "25" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	refresh := true
	result, err := c.Playgrounds.DebugWithParams(context.Background(), 42, &PlaygroundDebugParams{
		Mode:     "summary",
		Refresh:  &refresh,
		Service:  "web",
		LogsTail: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["ok"] != true {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestPlaygrounds_Logs(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/42/logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode logs body: %v", err)
		}
		if body["service"] != "web" || body["tail"] != float64(100) {
			t.Errorf("unexpected logs body: %#v", body)
		}
		json.NewEncoder(w).Encode(PlaygroundLogs{
			Service: "web",
			Lines:   []string{"line1", "line2"},
			Source:  "live",
		})
	})

	tail := 100
	logs, err := c.Playgrounds.Logs(context.Background(), 42, "web", &tail)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(logs.Lines))
	}
}

func TestAgents_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Agent{
			{ID: 1, Name: "agent-1", Provider: "github", Authenticated: true},
		}))
	})

	result, err := c.Agents.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 agent, got %d", len(result.Data))
	}
	if !result.Data[0].Authenticated {
		t.Error("expected authenticated=true")
	}
}

func TestAgents_ListIncludeRuntimeStatus(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/agents" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include_runtime_status"); got != "true" {
			t.Fatalf("include_runtime_status = %q", got)
		}
		json.NewEncoder(w).Encode(listEnv([]Agent{
			{
				ID:            1,
				Name:          "agent-1",
				Provider:      ProviderOpenAICodex,
				Authenticated: true,
				RuntimeStatus: &AgentRuntimeStatus{
					ID:               9,
					Status:           "running",
					RuntimeReachable: true,
					Authenticated:    true,
					IsProcessing:     true,
					QueueCount:       2,
				},
			},
		}))
	})

	include := true
	result, err := c.Agents.List(context.Background(), &AgentListParams{IncludeRuntimeStatus: &include, PerPage: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].RuntimeStatus == nil {
		t.Fatalf("missing runtime status: %#v", result)
	}
	if result.Data[0].RuntimeStatus.Status != "running" || !result.Data[0].RuntimeStatus.RuntimeReachable || result.Data[0].RuntimeStatus.QueueCount != 2 {
		t.Fatalf("unexpected runtime status: %#v", result.Data[0].RuntimeStatus)
	}
}

func TestAgents_GetByIdentifierEscapesName(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/test-agent" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(Agent{ID: 9, Name: "test-agent", Provider: ProviderOpenAICodex})
	})

	agent, err := c.Agents.GetByIdentifier(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != 9 || agent.Name != "test-agent" {
		t.Fatalf("unexpected agent: %#v", agent)
	}
}

func TestAgents_GetByIdentifierEscapesSpaces(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/test%20agent" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(Agent{ID: 10, Name: "test agent", Provider: ProviderOpenAICodex})
	})

	agent, err := c.Agents.GetByIdentifier(context.Background(), "test agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != 10 || agent.Name != "test agent" {
		t.Fatalf("unexpected agent: %#v", agent)
	}
}

func TestAgents_Chat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/messages" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		attachments, _ := body["attachmentFilenames"].([]any)
		if body["text"] != "hello" || body["conversation_id"] != "thread-1" || body["busy_policy"] != "queue" || len(attachments) != 1 || attachments[0] != "notes.txt" {
			t.Errorf("unexpected chat body: %#v", body)
		}
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
	})

	result, err := c.Agents.Chat(context.Background(), 5, &AgentChatParams{Text: "hello", ConversationID: "thread-1", BusyPolicy: "queue", AttachmentFilenames: []string{"notes.txt"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "accepted" {
		t.Errorf("expected status 'accepted', got %v", result["status"])
	}
}

func TestAgents_PokesByIdentifier(t *testing.T) {
	step := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/agents/test%20agent/pokes" {
				t.Fatalf("unexpected list request %s %s", r.Method, r.URL.EscapedPath())
			}
			if got := r.URL.Query().Get("per_page"); got != "10" {
				t.Fatalf("per_page = %q", got)
			}
			_ = json.NewEncoder(w).Encode(listEnv([]AgentPoke{{ID: 11, AgentID: 5, Schedule: "*/5 * * * *", Prompt: "keep going", Enabled: true}}))
		case 2:
			if r.Method != http.MethodPost || r.URL.EscapedPath() != "/api/agents/test%20agent/pokes" {
				t.Fatalf("unexpected create request %s %s", r.Method, r.URL.EscapedPath())
			}
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create: %v", err)
			}
			poke := body["agent_poke"]
			if poke["schedule"] != "*/5 * * * *" || poke["prompt"] != "keep going" || poke["conversation_id"] != "thread-1" {
				t.Fatalf("unexpected create body: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(AgentPoke{ID: 12, Schedule: "*/5 * * * *", Prompt: "keep going", Enabled: true})
		case 3:
			if r.Method != http.MethodPatch || r.URL.EscapedPath() != "/api/agents/test%20agent/pokes/12" {
				t.Fatalf("unexpected update request %s %s", r.Method, r.URL.EscapedPath())
			}
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update: %v", err)
			}
			if body["agent_poke"]["conversation_id"] != "" {
				t.Fatalf("expected clear conversation body, got %#v", body)
			}
			_ = json.NewEncoder(w).Encode(AgentPoke{ID: 12, Schedule: "*/5 * * * *", Prompt: "keep going", Enabled: false})
		case 4:
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/agents/test%20agent/pokes/12" {
				t.Fatalf("unexpected get request %s %s", r.Method, r.URL.EscapedPath())
			}
			_ = json.NewEncoder(w).Encode(AgentPoke{ID: 12, Schedule: "*/5 * * * *", Prompt: "keep going", Enabled: false})
		case 5:
			if r.Method != http.MethodDelete || r.URL.EscapedPath() != "/api/agents/test%20agent/pokes/12" {
				t.Fatalf("unexpected delete request %s %s", r.Method, r.URL.EscapedPath())
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
	})

	list, err := c.Agents.ListPokesByIdentifier(context.Background(), "test agent", &AgentPokeListParams{PerPage: 10})
	if err != nil || len(list.Data) != 1 || list.Data[0].ID != 11 {
		t.Fatalf("unexpected list result %#v err=%v", list, err)
	}
	created, err := c.Agents.CreatePokeByIdentifier(context.Background(), "test agent", &AgentPokeCreateParams{Schedule: "*/5 * * * *", Prompt: "keep going", ConversationID: "thread-1"})
	if err != nil || created.ID != 12 {
		t.Fatalf("unexpected create result %#v err=%v", created, err)
	}
	disabled := false
	clear := ""
	updated, err := c.Agents.UpdatePokeByIdentifier(context.Background(), "test agent", 12, &AgentPokeUpdateParams{ConversationID: &clear, Enabled: &disabled})
	if err != nil || updated.Enabled {
		t.Fatalf("unexpected update result %#v err=%v", updated, err)
	}
	got, err := c.Agents.GetPokeByIdentifier(context.Background(), "test agent", 12)
	if err != nil || got.ID != 12 {
		t.Fatalf("unexpected get result %#v err=%v", got, err)
	}
	if err := c.Agents.DeletePokeByIdentifier(context.Background(), "test agent", 12); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestAgents_Upload(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "attachment-*.webp")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := tmp.WriteString("hello"); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/test-agent/uploads" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Errorf("ParseMultipartForm: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if got := r.FormValue("conversation_id"); got != "thread-1" {
			t.Errorf("conversation_id = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("FormFile: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("ReadAll: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if string(content) != "hello" || header.Filename == "" {
			t.Errorf("unexpected uploaded file %q %q", header.Filename, string(content))
		}
		if got := header.Header.Get("Content-Type"); got != "image/webp" {
			t.Errorf("Content-Type = %q, want image/webp", got)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"filename": "runtime-file.txt"})
	})

	result, err := c.Agents.UploadByIdentifier(context.Background(), "test-agent", &AgentUploadParams{
		FilePath:       tmp.Name(),
		ConversationID: "thread-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Filename != "runtime-file.txt" {
		t.Errorf("unexpected upload result: %#v", result)
	}
}

func TestAgents_UploadReaderAndDownloadAttachment(t *testing.T) {
	step := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.Method != "POST" || r.URL.Path != "/api/agents/test-agent/uploads" {
				t.Fatalf("unexpected upload request %s %s", r.Method, r.URL.Path)
			}
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			if got := r.FormValue("conversation_id"); got != "thread-1" {
				t.Fatalf("conversation_id = %q", got)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile: %v", err)
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if string(content) != "zip-bytes" || header.Filename != "context.zip" {
				t.Fatalf("unexpected uploaded file %q %q", header.Filename, string(content))
			}
			if got := header.Header.Get("Content-Type"); got != "application/zip" {
				t.Fatalf("Content-Type = %q, want application/zip", got)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"filename": "runtime-context.zip"})
		case 2:
			if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/test-agent/uploads/runtime-context.zip" {
				t.Fatalf("unexpected download request %s %s", r.Method, r.URL.EscapedPath())
			}
			if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
				t.Fatalf("conversation_id query = %q", got)
			}
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", `inline; filename="runtime-context.zip"`)
			_, _ = w.Write([]byte("zip-bytes"))
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
	})

	upload, err := c.Agents.UploadReaderByIdentifier(context.Background(), "test-agent", strings.NewReader("zip-bytes"), "context.zip", &AgentUploadParams{ConversationID: "thread-1"})
	if err != nil {
		t.Fatalf("upload reader: %v", err)
	}
	if upload.Filename != "runtime-context.zip" {
		t.Fatalf("unexpected upload result: %#v", upload)
	}

	body, filename, contentType, err := c.Agents.DownloadAttachmentByIdentifier(context.Background(), "test-agent", upload.Filename, &AgentDataParams{ConversationID: "thread-1"})
	if err != nil {
		t.Fatalf("download attachment: %v", err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(data) != "zip-bytes" || filename != "runtime-context.zip" || contentType != "application/zip" {
		t.Fatalf("unexpected download data=%q filename=%q contentType=%q", string(data), filename, contentType)
	}
	if step != 2 {
		t.Fatalf("expected 2 requests, got %d", step)
	}
}

func TestAgents_StartChat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/chats" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["marquee_id"] != float64(9) {
			t.Errorf("expected marquee_id 9, got %v", body["marquee_id"])
		}
		json.NewEncoder(w).Encode(AgentChatSession{ID: 123, Status: "starting"})
	})

	session, err := c.Agents.StartChat(context.Background(), 5, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 123 || session.Status != "starting" {
		t.Errorf("unexpected session: %#v", session)
	}
}

func TestAgents_RestartChat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/restarts" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(AgentChatSession{ID: 123, Status: "pending"})
	})

	session, err := c.Agents.RestartChat(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 123 || session.Status != "pending" {
		t.Errorf("unexpected session: %#v", session)
	}
}

func TestAgents_RuntimeStatus(t *testing.T) {
	chatURL := "https://agent.example.test"
	lastError := "OpenCode provider quota/rate limit exhausted for provider=gemini model=gemini-2.5-flash-lite."
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/agents/5/runtime_status" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(AgentRuntimeStatus{
			ID:               123,
			Status:           "running",
			ChatURL:          &chatURL,
			RuntimeReachable: true,
			Authenticated:    true,
			IsProcessing:     false,
			QueueCount:       0,
			LastError:        &lastError,
		})
	})

	status, err := c.Agents.RuntimeStatus(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ID != 123 || status.Status != "running" || !status.RuntimeReachable || status.IsProcessing || status.QueueCount != 0 {
		t.Errorf("unexpected status: %#v", status)
	}
	if status.ChatURL == nil || *status.ChatURL != chatURL {
		t.Errorf("unexpected chat URL: %#v", status.ChatURL)
	}
	if status.LastError == nil || *status.LastError != lastError {
		t.Errorf("unexpected last error: %#v", status.LastError)
	}
}

func TestAgents_ConversationLifecycleByIdentifier(t *testing.T) {
	step := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.Method != "POST" || r.URL.EscapedPath() != "/api/agents/test%20agent/conversations" {
				t.Fatalf("unexpected create request %s %s", r.Method, r.URL.EscapedPath())
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if body["conversation_id"] != "thread-1" || body["title"] != "Project One" {
				t.Fatalf("unexpected create body: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "thread-1", "title": "Project One"})
		case 2:
			if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/test%20agent/live_state" {
				t.Fatalf("unexpected live request %s %s", r.Method, r.URL.EscapedPath())
			}
			if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
				t.Fatalf("conversation_id query = %q", got)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"content": map[string]any{
					"conversation_id": "thread-1",
					"isProcessing":    true,
					"streamText":      "partial",
					"queuedTurns":     2,
				},
			})
		case 3:
			if r.Method != "POST" || r.URL.EscapedPath() != "/api/agents/test%20agent/interrupts" {
				t.Fatalf("unexpected interrupt request %s %s", r.Method, r.URL.EscapedPath())
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode interrupt body: %v", err)
			}
			if body["conversation_id"] != "thread-1" {
				t.Fatalf("unexpected interrupt body: %#v", body)
			}
			json.NewEncoder(w).Encode(map[string]any{"interrupted": true, "conversation_id": "thread-1"})
		case 4:
			if r.Method != "DELETE" || r.URL.EscapedPath() != "/api/agents/test%20agent/conversations" {
				t.Fatalf("unexpected delete request %s %s", r.Method, r.URL.EscapedPath())
			}
			if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
				t.Fatalf("conversation_id query = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
	})

	created, err := c.Agents.CreateConversationByIdentifier(context.Background(), "test agent", &AgentConversationParams{
		ConversationID: "thread-1",
		Title:          "Project One",
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if created["id"] != "thread-1" || created["title"] != "Project One" {
		t.Fatalf("unexpected create result: %#v", created)
	}

	live, err := c.Agents.LiveStateByIdentifier(context.Background(), "test agent", &AgentDataParams{ConversationID: "thread-1"})
	if err != nil {
		t.Fatalf("live state: %v", err)
	}
	if live.ConversationID != "thread-1" || !live.IsProcessing || live.StreamText != "partial" || live.QueuedTurns != 2 {
		t.Fatalf("unexpected live state: %#v", live)
	}

	interrupted, err := c.Agents.InterruptByIdentifier(context.Background(), "test agent", &AgentConversationParams{ConversationID: "thread-1"})
	if err != nil {
		t.Fatalf("interrupt: %v", err)
	}
	if interrupted["interrupted"] != true {
		t.Fatalf("unexpected interrupt result: %#v", interrupted)
	}

	if err := c.Agents.DeleteConversationByIdentifier(context.Background(), "test agent", "thread-1"); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}
	if step != 4 {
		t.Fatalf("expected 4 requests, got %d", step)
	}
}

func TestAgents_ConversationMethodsRequireConversationID(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called")
	})

	if _, err := c.Agents.CreateConversationByIdentifier(context.Background(), "agent", nil); err == nil {
		t.Fatal("expected create validation error")
	}
	if err := c.Agents.DeleteConversationByIdentifier(context.Background(), "agent", ""); err == nil {
		t.Fatal("expected delete validation error")
	}
}

func TestAgents_DataReadsPassConversationID(t *testing.T) {
	step := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.Method != "GET" || r.URL.Path != "/api/agents/5/messages" {
				t.Fatalf("unexpected messages request %s %s", r.Method, r.URL.Path)
			}
		case 2:
			if r.Method != "GET" || r.URL.Path != "/api/agents/5/activity" {
				t.Fatalf("unexpected activity request %s %s", r.Method, r.URL.Path)
			}
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
		if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
			t.Fatalf("conversation_id query = %q", got)
		}
		json.NewEncoder(w).Encode(AgentData{Content: []any{map[string]any{"id": "item-1"}}})
	})

	if _, err := c.Agents.GetMessagesByIdentifierWithParams(context.Background(), "5", &AgentDataParams{ConversationID: "thread-1"}); err != nil {
		t.Fatalf("messages: %v", err)
	}
	if _, err := c.Agents.GetActivityByIdentifierWithParams(context.Background(), "5", &AgentDataParams{ConversationID: "thread-1"}); err != nil {
		t.Fatalf("activity: %v", err)
	}
	if step != 2 {
		t.Fatalf("expected 2 requests, got %d", step)
	}
}

func TestAgents_PurgeChat(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents/5/purges" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"id":             123,
			"status":         "stopping",
			"operation":      "purge_chat",
			"request_status": "queued",
		})
	})

	session, err := c.Agents.PurgeChat(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 123 || session.Status != "stopping" || session.Operation != "purge_chat" || session.RequestStatus != "queued" {
		t.Errorf("unexpected session: %#v", session)
	}
}

func TestAgents_CreateProviderAPIKeyModeJSON(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/agents" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		agent := body["agent"]
		if agent["provider_api_key_mode"] != true {
			t.Errorf("expected provider_api_key_mode=true bool, got %#v", agent["provider_api_key_mode"])
		}
		if agent["model_options"] != "flash-lite" {
			t.Errorf("expected model_options flash-lite, got %#v", agent["model_options"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Agent{ID: 99, Name: "new-agent", Provider: ProviderGemini})
	})

	providerAPIKeyMode := true
	modelOptions := "flash-lite"
	agent, err := c.Agents.Create(context.Background(), &AgentCreateParams{
		Name:               "new-agent",
		Provider:           ProviderGemini,
		ProviderAPIKeyMode: &providerAPIKeyMode,
		ModelOptions:       &modelOptions,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != 99 {
		t.Errorf("expected ID 99, got %d", agent.ID)
	}
}

func TestAgents_UpdateRenameContextJSON(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/api/agents/99" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["agent"]["name"] != "renamed" {
			t.Errorf("expected agent name, got %#v", body["agent"]["name"])
		}
		context := body["agent_rename_context"]
		if context["conversation_client_id"] != "thread-123" {
			t.Errorf("expected conversation context, got %#v", context)
		}
		json.NewEncoder(w).Encode(Agent{ID: 99, Name: "renamed", Provider: ProviderGemini})
	})

	name := "renamed"
	agent, err := c.Agents.Update(context.Background(), 99, &AgentUpdateParams{
		Name: &name,
		RenameContext: &AgentRenameContext{
			ConversationClientID: "thread-123",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != name {
		t.Errorf("expected name %q, got %q", name, agent.Name)
	}
}

func TestSecrets_List(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(listEnv([]Secret{{Key: "DB_URL"}}))
	})

	result, err := c.Secrets.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 secret, got %d", len(result.Data))
	}
}

func TestSecrets_GetReveal(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/secrets/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("reveal") != "true" {
			t.Errorf("expected reveal=true, got %q", r.URL.Query().Get("reveal"))
		}
		id := int64(42)
		value := "secret"
		json.NewEncoder(w).Encode(Secret{ID: &id, Key: "DB_URL", Value: &value})
	})

	secret, err := c.Secrets.Get(context.Background(), 42, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret.Value == nil || *secret.Value != "secret" {
		t.Errorf("expected revealed value, got %#v", secret.Value)
	}
}

func TestSecrets_GetByIdentifierUsesKey(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/secrets/DB_URL" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		if r.URL.Query().Get("reveal") != "true" {
			t.Errorf("expected reveal=true, got %q", r.URL.Query().Get("reveal"))
		}
		id := int64(42)
		value := "secret"
		json.NewEncoder(w).Encode(Secret{ID: &id, Key: "DB_URL", Value: &value})
	})

	secret, err := c.Secrets.GetByIdentifier(context.Background(), "DB_URL", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret.Key != "DB_URL" || secret.Value == nil || *secret.Value != "secret" {
		t.Fatalf("unexpected secret: %#v", secret)
	}
}

func TestJobEnv_GetReveal(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/job_env/7" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("reveal") != "true" {
			t.Errorf("expected reveal=true, got %q", r.URL.Query().Get("reveal"))
		}
		id := int64(7)
		value := "env-secret"
		json.NewEncoder(w).Encode(JobEnvEntry{ID: &id, Key: "TOKEN", Value: &value, Secret: true})
	})

	entry, err := c.JobEnv.Get(context.Background(), 7, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Value == nil || *entry.Value != "env-secret" {
		t.Errorf("expected revealed value, got %#v", entry.Value)
	}
}

func TestJobEnv_SetAllowsEmptyValue(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/job_env" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		params := body["job_env"]
		if params["key"] != "SERVICES_ONLY" || params["value"] != "" {
			t.Fatalf("unexpected payload: %#v", params)
		}
		id := int64(8)
		value := ""
		json.NewEncoder(w).Encode(JobEnvEntry{ID: &id, Key: "SERVICES_ONLY", Value: &value})
	})

	entry, err := c.JobEnv.Set(context.Background(), &JobEnvSetParams{Key: "SERVICES_ONLY", Value: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Value == nil || *entry.Value != "" {
		t.Fatalf("expected empty value, got %#v", entry.Value)
	}
}

func TestImportTemplates_SetSourceCIFields(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/api/import_templates/11/source" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body map[string]map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		source := body["source"]
		if source["ci_enabled"] != true {
			t.Errorf("expected ci_enabled=true, got %#v", source["ci_enabled"])
		}
		if source["ci_marquee_id"] != float64(22) {
			t.Errorf("expected ci_marquee_id=22, got %#v", source["ci_marquee_id"])
		}
		id := int64(11)
		json.NewEncoder(w).Encode(ImportTemplate{ID: &id})
	})

	ciEnabled := true
	ciMarqueeID := int64(22)
	_, err := c.ImportTemplates.SetSource(context.Background(), 11, &ImportTemplateSourceParams{
		SourcePropID: 1,
		SourcePath:   "fibe-ci.yml",
		CIEnabled:    &ciEnabled,
		CIMarqueeID:  &ciMarqueeID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportTemplates_SetSourceByIdentifierUsesName(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.EscapedPath() != "/api/import_templates/starter/source" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		var body map[string]map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		source := body["source"]
		if source["source_prop_id"] != "api-prop" {
			t.Errorf("expected source_prop_id=api-prop, got %#v", source["source_prop_id"])
		}
		id := int64(11)
		json.NewEncoder(w).Encode(ImportTemplate{ID: &id, Name: "starter"})
	})

	_, err := c.ImportTemplates.SetSourceByIdentifier(context.Background(), "starter", &ImportTemplateSourceParams{
		SourcePropIdentifier: "api-prop",
		SourcePath:           "fibe-ci.yml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportTemplates_SearchWithParamsRegex(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/import_templates" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("q") != "starter-.*" {
			t.Errorf("expected q=starter-.*, got %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("regex") != "true" {
			t.Errorf("expected regex=true, got %q", r.URL.Query().Get("regex"))
		}
		json.NewEncoder(w).Encode(listEnv([]ImportTemplate{{Name: "app-starter"}}))
	})

	result, err := c.ImportTemplates.SearchWithParams(context.Background(), &ImportTemplateSearchParams{
		Query: "starter-.*",
		Regex: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].Name != "app-starter" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestArtefacts_GetByAgentAndArtefactIdentifierUsesNames(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.EscapedPath() != "/api/agents/builder/artefacts/report" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode(Artefact{ID: 5, Name: "report"})
	})

	artefact, err := c.Artefacts.GetByAgentAndArtefactIdentifier(context.Background(), "builder", "report")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if artefact.ID != 5 || artefact.Name != "report" {
		t.Fatalf("unexpected artefact: %#v", artefact)
	}
}

func TestAPIKeys_Me(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(Player{ID: 1, Username: "testuser"})
	})

	player, err := c.APIKeys.Me(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %q", player.Username)
	}
}

func TestWebhookEndpoints_EventTypes(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/webhook_event_types" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"event_types": []string{"playground.created", "playground.status.changed"},
		})
	})

	types, err := c.WebhookEndpoints.EventTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) != 2 {
		t.Errorf("expected 2 event types, got %d", len(types))
	}
}

func TestProps_Sync(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/props/7/syncs" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"message": "Sync scheduled"})
	})

	err := c.Props.Sync(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildQuery(t *testing.T) {
	params := &ArtefactListParams{
		Query:   "test",
		Sort:    "created_at_asc",
		Page:    1,
		PerPage: 25,
	}

	q := buildQuery(params)
	if q == "" {
		t.Error("expected non-empty query string")
	}
	if q[0] != '?' {
		t.Error("expected query to start with '?'")
	}
}

func TestBuildQuery_NilParams(t *testing.T) {
	q := buildQuery(nil)
	if q != "" {
		t.Errorf("expected empty query for nil params, got %q", q)
	}
}

func TestBuildQuery_EmptyParams(t *testing.T) {
	params := &ArtefactListParams{}
	q := buildQuery(params)
	if q != "" {
		t.Errorf("expected empty query for zero-value params, got %q", q)
	}
}

func TestStatus_Get_WithLimitsSections(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/status" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		limit1000 := 1000
		limit10 := 10
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"playgrounds":  map[string]any{"total": 2, "active": 1, "stopped": 1},
			"agents":       map[string]any{"total": 3, "authenticated": 2},
			"props":        5,
			"playspecs":    4,
			"marquees":     1,
			"secrets":      0,
			"api_keys":     2,
			"subscription": map[string]any{"plan": "single", "playground_limit": 1000},
			"resource_quotas": map[string]any{
				"playgrounds": map[string]any{"used": 2, "limit": limit1000, "status": "ok"},
				"agents":      map[string]any{"used": 3, "limit": limit10, "status": "ok"},
			},
			"per_parent_caps": map[string]any{
				"mounted_files_per_agent": 5,
				"artefacts_per_agent":     100,
			},
			"rate_limits": map[string]any{
				"api": map[string]any{"limit": 5000, "remaining": 4987, "reset_seconds": 1234},
			},
		})
	})

	status, err := c.Status.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ResourceQuotas == nil || status.ResourceQuotas["playgrounds"].Used != 2 {
		t.Errorf("expected resource_quotas.playgrounds.used=2, got %+v", status.ResourceQuotas)
	}
	if status.ResourceQuotas["playgrounds"].Limit == nil || *status.ResourceQuotas["playgrounds"].Limit != 1000 {
		t.Errorf("expected playgrounds limit 1000, got %+v", status.ResourceQuotas["playgrounds"].Limit)
	}
	if status.PerParentCaps["mounted_files_per_agent"] == nil || *status.PerParentCaps["mounted_files_per_agent"] != 5 {
		t.Errorf("expected per_parent_caps.mounted_files_per_agent=5, got %+v", status.PerParentCaps)
	}
	if status.RateLimits == nil || status.RateLimits.API == nil {
		t.Fatalf("expected rate_limits.api section")
	}
	if status.RateLimits.API.Limit != 5000 || status.RateLimits.API.Remaining != 4987 {
		t.Errorf("unexpected rate limit values: %+v", status.RateLimits.API)
	}
}

func TestStatus_Get_WithoutLimitsSections(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"playgrounds":  map[string]any{"total": 0, "active": 0, "stopped": 0},
			"agents":       map[string]any{"total": 0, "authenticated": 0},
			"props":        0,
			"playspecs":    0,
			"marquees":     0,
			"secrets":      0,
			"api_keys":     0,
			"subscription": map[string]any{"plan": "free", "playground_limit": 1000},
		})
	})

	status, err := c.Status.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.ResourceQuotas != nil {
		t.Errorf("expected nil resource_quotas when omitted, got %+v", status.ResourceQuotas)
	}
	if status.PerParentCaps != nil {
		t.Errorf("expected nil per_parent_caps when omitted, got %+v", status.PerParentCaps)
	}
	if status.RateLimits != nil {
		t.Errorf("expected nil rate_limits when omitted, got %+v", status.RateLimits)
	}
}
