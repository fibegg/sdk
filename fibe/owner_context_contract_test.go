package fibe

import (
	"context"
	"net/http"
	"net/http/httptest"
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
