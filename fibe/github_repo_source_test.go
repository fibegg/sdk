package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestParseGitHubRepoSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fullName string
		ref      string
	}{
		{name: "short", input: "owner/repo", fullName: "owner/repo"},
		{name: "short with ref", input: "owner/repo@feature/foo", fullName: "owner/repo", ref: "feature/foo"},
		{name: "https", input: "https://github.com/owner/repo", fullName: "owner/repo"},
		{name: "https git suffix", input: "https://github.com/owner/repo.git", fullName: "owner/repo"},
		{name: "ssh", input: "git@github.com:owner/repo.git", fullName: "owner/repo"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source, err := ParseGitHubRepoSource(tc.input)
			if err != nil {
				t.Fatalf("ParseGitHubRepoSource: %v", err)
			}
			if source.FullName != tc.fullName || source.Ref != tc.ref || source.URL != "https://github.com/"+tc.fullName {
				t.Fatalf("unexpected source: %#v", source)
			}
		})
	}
}

func TestParseGitHubRepoSourceRejectsUnsupportedURL(t *testing.T) {
	if _, err := ParseGitHubRepoSource("https://gitlab.com/owner/repo"); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}

func TestInferNameFromGitHubRepoSource(t *testing.T) {
	source, err := ParseGitHubRepoSource("Owner/My_App.git")
	if err != nil {
		t.Fatalf("ParseGitHubRepoSource: %v", err)
	}
	if got := InferNameFromGitHubRepoSource(source); got != "my_app" {
		t.Fatalf("name=%q want my_app", got)
	}
}

func TestResolveGitHubInstallation(t *testing.T) {
	account := "fibegg"
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/installations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(listEnv([]Installation{{
			ID:                  7,
			Provider:            "github",
			InstallationID:      123,
			InstallationAccount: &account,
		}}))
	})

	inst, err := ResolveGitHubInstallation(context.Background(), c, GitHubInstallationSelection{})
	if err != nil {
		t.Fatalf("ResolveGitHubInstallation: %v", err)
	}
	if inst.InstallationID != 123 {
		t.Fatalf("installation_id=%d want 123", inst.InstallationID)
	}
}

func TestResolveGitHubInstallationRequiresExplicitSelectionWhenAmbiguous(t *testing.T) {
	a := "one"
	b := "two"
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(listEnv([]Installation{
			{ID: 1, InstallationID: 101, InstallationAccount: &a},
			{ID: 2, InstallationID: 202, InstallationAccount: &b},
		}))
	})

	if _, err := ResolveGitHubInstallation(context.Background(), c, GitHubInstallationSelection{}); err == nil {
		t.Fatal("expected ambiguity error")
	}
}

func TestResolveGitHubInstallationRejectsDuplicateAccountAlias(t *testing.T) {
	account := "fibegg"
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(listEnv([]Installation{
			{ID: 1, InstallationID: 101, InstallationAccount: &account},
			{ID: 2, InstallationID: 202, InstallationAccount: &account},
		}))
	})

	if _, err := ResolveGitHubInstallation(context.Background(), c, GitHubInstallationSelection{Account: account}); err == nil {
		t.Fatal("expected duplicate account ambiguity error")
	}
}
