package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

const testFibeComposeSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "x-fibe.gg": { "version": "1" },
  "type": "object",
  "required": ["services"],
  "properties": {
    "services": {
      "type": "object",
      "additionalProperties": { "$ref": "#/$defs/service" }
    }
  },
  "additionalProperties": true,
  "$defs": {
    "service": {
      "type": "object",
      "properties": {
        "labels": { "$ref": "#/$defs/labels" }
      },
      "additionalProperties": true
    },
    "labels": {
      "oneOf": [
        { "$ref": "#/$defs/labelsObject" },
        { "$ref": "#/$defs/labelsArray" }
      ]
    },
    "labelsObject": {
      "type": "object",
      "propertyNames": { "$ref": "#/$defs/labelName" },
      "additionalProperties": true
    },
    "labelsArray": {
      "type": "array",
      "items": {
        "anyOf": [
          { "pattern": "^fibe\\.gg/expose(?:=.*)?$" },
          { "type": "string", "not": { "pattern": "^fibe\\.gg/" } }
        ]
      }
    },
    "labelName": {
      "anyOf": [
        { "enum": ["fibe.gg/expose"] },
        { "not": { "pattern": "^fibe\\.gg/" } }
      ]
    }
  }
}`

func TestValidateComposeWithParamsHonorsFIBESchemaURL(t *testing.T) {
	var schemaRequests atomic.Int64
	schemaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		schemaRequests.Add(1)
		w.Header().Set("Content-Type", "application/schema+json")
		_, _ = w.Write([]byte(testFibeComposeSchema))
	}))
	defer schemaServer.Close()
	t.Setenv("FIBE_SCHEMA_URL", schemaServer.URL)

	var apiRequests atomic.Int64
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequests.Add(1)
		if r.URL.Path != "/api/playspecs/validate_compose" {
			t.Fatalf("unexpected API path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ComposeValidation{Valid: true})
	}))
	defer apiServer.Close()

	client := NewClient(WithAPIKey("test"), WithBaseURL(apiServer.URL), WithMaxRetries(0))
	result, err := client.Playspecs.ValidateCompose(context.Background(), "services:\n  web:\n    image: nginx\n")
	if err != nil {
		t.Fatalf("ValidateCompose returned error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid result, got %#v", result)
	}
	if schemaRequests.Load() != 1 {
		t.Fatalf("expected one schema request, got %d", schemaRequests.Load())
	}
	if apiRequests.Load() != 1 {
		t.Fatalf("expected one API request, got %d", apiRequests.Load())
	}
}

func TestValidateComposeWithParamsStopsBeforeRailsOnSchemaFailure(t *testing.T) {
	schemaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/schema+json")
		_, _ = w.Write([]byte(testFibeComposeSchema))
	}))
	defer schemaServer.Close()
	t.Setenv("FIBE_SCHEMA_URL", schemaServer.URL)

	var apiRequests atomic.Int64
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiServer.Close()

	client := NewClient(WithAPIKey("test"), WithBaseURL(apiServer.URL), WithMaxRetries(0))
	result, err := client.Playspecs.ValidateCompose(context.Background(), "services:\n  web:\n    image: nginx\n    labels:\n      fibe.gg/unknown: nope\n")
	if err != nil {
		t.Fatalf("ValidateCompose returned error: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid schema preflight result")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected schema errors")
	}
	if apiRequests.Load() != 0 {
		t.Fatalf("expected no API requests after schema failure, got %d", apiRequests.Load())
	}
}
