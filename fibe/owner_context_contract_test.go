package fibe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnerContextProfileDoesNotReplacePersonalDomainCredential(t *testing.T) {
	store := NewCredentialStore(filepath.Join(t.TempDir(), "credentials.json"))
	personal := &CredentialEntry{APIKey: "fixture-personal", Domain: "example.test"}
	if err := store.SetProfile("personal", personal); err != nil {
		t.Fatal(err)
	}
	company := &CredentialEntry{APIKey: "fixture-company", Domain: "example.test", CredentialContext: CredentialContext{
		OwnerContext: OwnerContext{OwnerType: "Team", OwnerID: 19}, PrincipalType: "TeamServicePrincipal", PrincipalID: 31, AuthorizationVersion: 2}}
	if err := store.SetProfile("company", company); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get("example.test")
	if err != nil || got.APIKey != "fixture-personal" {
		t.Fatal("company profile replaced personal domain lookup")
	}
	stored, err := store.GetProfile("company")
	if err != nil || stored.OwnerType != "Team" || stored.OwnerID != 19 || stored.PrincipalID != 31 {
		t.Fatal("owner binding was not retained")
	}
}

func TestOwnerContextProofMatchesCredentialResponse(t *testing.T) {
	for _, actual := range []string{"19", "20", ""} {
		t.Run(actual, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Fibe-Expected-Owner-Type") != "Team" || r.Header.Get("X-Fibe-Expected-Owner-ID") != "19" {
					t.Error("request omitted profile binding")
				}
				w.Header().Set("X-Fibe-Owner-Type", "Team")
				w.Header().Set("X-Fibe-Owner-ID", actual)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"owner_type":"Team","owner_id":19,"principal_type":"TeamServicePrincipal","principal_id":31,"authorization_version":2}`))
			}))
			defer server.Close()
			client := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL), WithAPIKey("fixture"), WithMaxRetries(0), WithOwnerContext(OwnerContext{OwnerType: "Team", OwnerID: 19}))
			result, err := client.OwnerContext(context.Background())
			if actual == "19" && (err != nil || result.PrincipalType != "TeamServicePrincipal") {
				t.Fatalf("matching owner rejected: %v", err)
			}
			if actual != "19" && err == nil {
				t.Fatal("mismatched or missing proof was accepted")
			}
		})
	}
}

// sharedOwnerContextFixtureSHA256 pins the credential-metadata fixture shared with the Rails contract and
// fibe-agent. Every repository pins the same digest; change all copies together.
const sharedOwnerContextFixtureSHA256 = "b0fdea43bb72126b5bf7dd744eb41cdc748db25775f385c29ebee762705d1085"

type sharedOwnerContextCase struct {
	Name            string            `json:"name"`
	Credential      CredentialContext `json:"credential"`
	APIResponse     json.RawMessage   `json:"api_response"`
	ContainerEnv    map[string]string `json:"container_env"`
	ProofHeaders    map[string]string `json:"proof_headers"`
	ResponseHeaders map[string]string `json:"response_headers"`
}

func loadSharedOwnerContextFixture(t *testing.T) []sharedOwnerContextCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "owner_context_metadata_v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != sharedOwnerContextFixtureSHA256 {
		t.Fatalf("shared owner-context fixture digest = %s, want %s (update every repository together)", got, sharedOwnerContextFixtureSHA256)
	}
	var document struct {
		Cases []sharedOwnerContextCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &document); err != nil || len(document.Cases) != 4 {
		t.Fatalf("fixture cases = %d, err = %v", len(document.Cases), err)
	}
	return document.Cases
}

func TestOwnerContextDecodesSharedCredentialMetadataFixture(t *testing.T) {
	for _, entry := range loadSharedOwnerContextFixture(t) {
		t.Run(entry.Name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for name, value := range entry.ProofHeaders {
					if r.Header.Get(name) != value {
						t.Errorf("proof header %s = %q, want %q", name, r.Header.Get(name), value)
					}
				}
				for name, value := range entry.ResponseHeaders {
					w.Header().Set(name, value)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(entry.APIResponse)
			}))
			defer server.Close()
			client := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL), WithAPIKey("fixture"), WithMaxRetries(0),
				WithOwnerContext(entry.Credential.OwnerContext))
			got, err := client.OwnerContext(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.OwnerContext != entry.Credential.OwnerContext || got.PrincipalType != entry.Credential.PrincipalType ||
				got.PrincipalID != entry.Credential.PrincipalID || got.AuthorizationVersion != entry.Credential.AuthorizationVersion {
				t.Fatalf("decoded %+v, want %+v", got, entry.Credential)
			}
			if got.AuthorityRevision == nil {
				t.Fatal("authority_revision was not decoded")
			}
		})
	}
}
