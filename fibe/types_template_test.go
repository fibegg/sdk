package fibe

import (
	"encoding/json"
	"testing"
)

func TestImportTemplateUnmarshalAcceptsRailsTimestampFormat(t *testing.T) {
	body := []byte(`{
		"id": 1,
		"name": "greenfield",
		"created_at": "2026-04-30 13:05:15 UTC",
		"updated_at": "2026-04-30 13:06:15 UTC",
		"source": {
			"path": "docker-compose.yml",
			"ref": "main",
			"last_refreshed_at": "2026-04-30 13:07:15 UTC"
		},
		"versions": [{
			"id": 2,
			"template_body": "services: {}",
			"created_at": "2026-04-30 13:08:15 UTC"
		}]
	}`)

	var template ImportTemplate
	if err := json.Unmarshal(body, &template); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if template.CreatedAt == nil || template.CreatedAt.UTC().Format("2006-01-02T15:04:05Z") != "2026-04-30T13:05:15Z" {
		t.Fatalf("unexpected created_at: %#v", template.CreatedAt)
	}
	if template.UpdatedAt == nil || template.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z") != "2026-04-30T13:06:15Z" {
		t.Fatalf("unexpected updated_at: %#v", template.UpdatedAt)
	}
	if template.Source == nil || template.Source.LastRefreshedAt == nil {
		t.Fatalf("expected source last_refreshed_at to parse")
	}
	if len(template.Versions) != 1 || template.Versions[0].CreatedAt == nil {
		t.Fatalf("expected version created_at to parse")
	}
}
