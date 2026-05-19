package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestTransformRejectsRawPlayspecBeforeCreatingTemplate(t *testing.T) {
	var sawTemplateCreate bool
	playspecID := int64(9)
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/playgrounds/7":
			json.NewEncoder(w).Encode(Playground{ID: 7, Name: "raw-pg", Status: "running", PlayspecID: &playspecID})
		case r.Method == http.MethodGet && r.URL.Path == "/api/playspecs/9":
			id := int64(9)
			json.NewEncoder(w).Encode(Playspec{ID: &id, Name: "raw"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/import_templates":
			sawTemplateCreate = true
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	result, err := c.Transform(context.Background(), &PlaygroundTransformParams{
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
	playspecID := int64(9)
	sourceVersionID := int64(33)
	templateID := int64(11)
	templateVersionID := int64(22)
	var createBody map[string]any
	var switchBody map[string]any

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/playgrounds/7":
			json.NewEncoder(w).Encode(Playground{ID: 7, Name: "pg", Status: "running", PlayspecID: &playspecID})
		case r.Method == http.MethodGet && r.URL.Path == "/api/playspecs/9":
			id := int64(9)
			json.NewEncoder(w).Encode(Playspec{ID: &id, Name: "ps", SourceTemplateVersionID: &sourceVersionID})
		case r.Method == http.MethodPost && r.URL.Path == "/api/import_templates":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode template create body: %v", err)
			}
			json.NewEncoder(w).Encode(ImportTemplate{ID: &templateID, Name: "pg-transform", LatestVersionID: &templateVersionID})
		case r.Method == http.MethodPost && r.URL.Path == "/api/playspecs/9/template_switches":
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
				"playspec": map[string]any{
					"id":                         playspecID,
					"source_template_version_id": templateVersionID,
				},
				"playground_rollout_plan": map[string]any{"rollout": []int64{7}},
				"provisioned_props":       []map[string]any{{"prop_id": 44, "source_repo_url": "https://github.com/fibegg/private-api"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/playgrounds/7/status":
			json.NewEncoder(w).Encode(PlaygroundStatus{ID: 7, Status: "running"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	provisionPrivate := false
	result, err := c.Transform(context.Background(), &PlaygroundTransformParams{
		PlaygroundID:          7,
		TemplateBody:          "services:\n  web:\n    image: nginx\n",
		TemplateName:          "pg-transform",
		ProvisionMissingProps: "gitea",
		ProvisionPrivate:      &provisionPrivate,
		ReuseExistingProps:    true,
		Wait:                  true,
	})
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if result.Template == nil || result.Template.ID == nil || *result.Template.ID != templateID {
		t.Fatalf("expected created template in result, got %#v", result.Template)
	}
	if len(result.ProvisionedProps) != 1 || result.ProvisionedProps[0].PropID != 44 {
		t.Fatalf("expected provisioned prop result, got %#v", result.ProvisionedProps)
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
	if switchBody["provision_missing_props"] != "gitea" || switchBody["provision_private"] != false {
		t.Fatalf("unexpected provision switch body: %#v", switchBody)
	}
	if switchBody["reuse_existing_props"] != true {
		t.Fatalf("unexpected reuse_existing_props in switch body: %#v", switchBody)
	}
}
