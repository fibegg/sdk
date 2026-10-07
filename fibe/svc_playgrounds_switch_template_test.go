package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestTransformRejectsRawSpecBeforeCreatingTemplate(t *testing.T) {
	var sawTemplateCreate bool
	specID := int64(9)
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/playgrounds/7":
			json.NewEncoder(w).Encode(Playground{ID: 7, Name: "raw-pg", Status: "running", SpecID: &specID})
		case r.Method == http.MethodGet && r.URL.Path == "/api/specs/9":
			id := int64(9)
			json.NewEncoder(w).Encode(Spec{ID: &id, Name: "raw"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/import_templates":
			sawTemplateCreate = true
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	result, err := c.SwitchPlaygroundTemplate(WithFields(context.Background(), "mode"), &PlaygroundTemplateSwitchParams{
		PlaygroundID: 7,
		TemplateBody: "services: {}\n",
	})
	if err == nil || !strings.Contains(err.Error(), "was not launched from a template version") {
		t.Fatalf("expected template-origin error, got result=%#v err=%v", result, err)
	}
	if result == nil || result.Playground == nil || result.Playground.ID != 7 {
		t.Fatalf("expected partial result with playground, got %#v", result)
	}
	if sawTemplateCreate {
		t.Fatal("Transform created an import template before checking template origin")
	}
}

func TestTransformCreatesTemplateSwitchesAndWaits(t *testing.T) {
	specID := int64(9)
	sourceVersionID := int64(33)
	templateID := int64(11)
	templateVersionID := int64(22)
	var createBody map[string]any
	var switchBody map[string]any

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/playgrounds/7":
			json.NewEncoder(w).Encode(Playground{ID: 7, Name: "pg", Status: "running", SpecID: &specID})
		case r.Method == http.MethodGet && r.URL.Path == "/api/specs/9":
			id := int64(9)
			json.NewEncoder(w).Encode(Spec{ID: &id, Name: "ps", SourceTemplateVersionID: &sourceVersionID})
		case r.Method == http.MethodPost && r.URL.Path == "/api/import_templates":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode template create body: %v", err)
			}
			json.NewEncoder(w).Encode(ImportTemplate{ID: &templateID, Name: "pg-transform", LatestVersionID: &templateVersionID})
		case r.Method == http.MethodPost && r.URL.Path == "/api/specs/9/template_switches":
			if err := json.NewDecoder(r.Body).Decode(&switchBody); err != nil {
				t.Fatalf("decode switch body: %v", err)
			}
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{"request_id": "req-switch", "status": "queued"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/async_requests/req-switch":
			json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-switch",
				"status":     "success",
				"target_template_version": map[string]any{
					"id": templateVersionID,
				},
				"spec": map[string]any{
					"id":                         specID,
					"source_template_version_id": templateVersionID,
				},
				"playground_rollout_plan":  map[string]any{"rollout": []int64{7}},
				"provisioned_repositories": []map[string]any{{"repository_id": 44, "source_repo_url": "https://github.com/fibegg/private-api"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/playgrounds/7/status":
			json.NewEncoder(w).Encode(PlaygroundStatus{ID: 7, Status: "running", Services: []PlaygroundServiceInfo{{Name: "web", Status: "running", Running: true}}})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	provisionPrivate := false
	result, err := c.SwitchPlaygroundTemplate(WithFields(context.Background(), "mode"), &PlaygroundTemplateSwitchParams{
		PlaygroundID:                 7,
		TemplateBody:                 "services:\n  web:\n    image: nginx\n",
		TemplateName:                 "pg-transform",
		ProvisionMissingRepositories: "gitea",
		ProvisionPrivate:             &provisionPrivate,
		ReuseExistingRepositories:    true,
		Wait:                         true,
	})
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if result.Template == nil || result.Template.ID == nil || *result.Template.ID != templateID {
		t.Fatalf("expected created template in result, got %#v", result.Template)
	}
	if len(result.ProvisionedRepositories) != 1 || result.ProvisionedRepositories[0].RepositoryID != 44 {
		t.Fatalf("expected provisioned repository result, got %#v", result.ProvisionedRepositories)
	}
	if len(result.WaitResults) != 1 || result.WaitResults[0]["success"] != true {
		t.Fatalf("expected successful wait result, got %#v", result.WaitResults)
	}

	templatePayload := createBody["import_template"].(map[string]any)
	if templatePayload["name"] != "pg-transform" || createBody["template_body"] == "" {
		t.Fatalf("unexpected template create body: %#v", createBody)
	}
	if switchBody["target_template_version_id"].(float64) != float64(templateVersionID) {
		t.Fatalf("unexpected switch target: %#v", switchBody)
	}
	if switchBody["rollout_mode"] != "target" || switchBody["target_playground_id"].(float64) != 7 {
		t.Fatalf("unexpected rollout switch body: %#v", switchBody)
	}
	if switchBody["provision_missing_repositories"] != "gitea" || switchBody["provision_private"] != false {
		t.Fatalf("unexpected provision switch body: %#v", switchBody)
	}
	if switchBody["reuse_existing_repositories"] != true {
		t.Fatalf("unexpected reuse_existing_repositories in switch body: %#v", switchBody)
	}
}
