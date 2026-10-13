package fibe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestRepositoryMirrorWaitsForCompanyResult(t *testing.T) {
	for _, withState := range []bool{false, true} {
		t.Run(map[bool]string{false: "Mirror", true: "MirrorWithState"}[withState], func(t *testing.T) {
			posts, polls := 0, 0
			c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/repositories/mirrors":
					posts++
					w.WriteHeader(http.StatusAccepted)
					json.NewEncoder(w).Encode(map[string]any{"request_id": "company-import", "status": "queued", "status_url": "/api/company_repository_operations/company-import"})
				case r.Method == http.MethodGet && r.URL.Path == "/api/company_repository_operations/company-import":
					polls++
					json.NewEncoder(w).Encode(map[string]any{"request_id": "company-import", "status": "success", "repository": map[string]any{"id": 42, "name": "imported", "owner_type": "Team", "owner_id": 9, "provider": "github", "status": "active"}})
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			})
			var repo *Repository
			var err error
			if withState {
				var result *RepositoryMirrorState
				result, err = c.Repositories.MirrorWithState(context.Background(), "https://github.com/upstream/imported", "")
				if result != nil {
					repo = &result.Repository
				}
			} else {
				repo, err = c.Repositories.Mirror(context.Background(), "https://github.com/upstream/imported", "")
			}
			if err != nil || repo == nil || repo.ID != 42 || repo.OwnerType != "Team" || repo.OwnerID != 9 || repo.Status != "active" || posts != 1 || polls != 1 {
				t.Fatalf("expected one completed Company import: repo=%+v error=%v posts=%d polls=%d", repo, err, posts, polls)
			}
		})
	}
}

func TestRepositoryMirrorPreservesPersonalResponse(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected poll for synchronous Personal response")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 43, "provider": "gitea", "status": "active", "mirror_status": "pending"})
	})
	result, err := c.Repositories.MirrorWithState(context.Background(), "https://github.com/upstream/imported", "")
	if err != nil || result.ID != 43 || result.MirrorStatus != "pending" {
		t.Fatalf("Personal mirror changed: %+v %v", result, err)
	}
}

func TestRepositoryMirrorReportsRetainedReconciliationError(t *testing.T) {
	posts := 0
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{"request_id": "unknown-import", "status": "queued", "status_url": "/api/company_repository_operations/unknown-import"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "error", "error_code": "DESTINATION_RECONCILIATION_REQUIRED", "error": "Inspect the retained operation"})
	})
	_, err := c.Repositories.Mirror(context.Background(), "https://github.com/upstream/imported", "")
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Code != "DESTINATION_RECONCILIATION_REQUIRED" || posts != 1 {
		t.Fatalf("expected retained error without another POST: err=%v posts=%d", err, posts)
	}
}

func TestRepositoryMirrorSelectsExplicitCompanyInstallation(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["github_installation_id"] != float64(501) || body["name"] != "destination" || body["source_url"] != "https://github.com/upstream/source" {
			t.Errorf("authored import changed: %+v", body)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 44, "provider": "github"})
	})
	result, err := c.Repositories.MirrorWithParams(context.Background(), &RepositoryMirrorParams{SourceURL: "https://github.com/upstream/source", Name: "destination", GitHubInstallationID: 501})
	if err != nil || result.ID != 44 {
		t.Fatalf("explicit installation import: %+v %v", result, err)
	}
}

func TestRepositoryMirrorRejectsInvalidParametersBeforeHTTP(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("invalid import reached HTTP") })
	for _, p := range []*RepositoryMirrorParams{nil, {}, {SourceURL: "https://github.com/upstream/source", GitHubInstallationID: -1}} {
		if _, err := c.Repositories.MirrorWithParams(context.Background(), p); err == nil {
			t.Fatalf("accepted invalid parameters %+v", p)
		}
	}
}

func TestRepositoryMirrorProjectsTheCompletedRepository(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{"request_id": "projected-import", "status": "queued", "status_url": "/api/company_repository_operations/projected-import"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "repository": map[string]any{"id": 45, "name": "projected", "provider": "github", "status": "active"}})
	})
	result, err := c.Repositories.MirrorWithState(WithFields(context.Background(), "id", "name"), "https://github.com/upstream/source", "")
	if err != nil || result.ID != 45 || result.Name != "projected" || result.Provider != "" {
		t.Fatalf("projection did not apply to final Repository: %+v %v", result, err)
	}
}
