package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnvPackNilEmptyTargetsAndValues(t *testing.T) {
	empty := []EnvPackAttachmentInput{}
	for _, test := range []struct {
		p    EnvPackAttachmentsParams
		want string
	}{{EnvPackAttachmentsParams{}, `{}`}, {EnvPackAttachmentsParams{Attachments: &empty}, `{"env_pack_attachments":[]}`}} {
		bytes, err := json.Marshal(test.p)
		if err != nil || string(bytes) != test.want {
			t.Fatalf("marshal %s %v", bytes, err)
		}
	}
	targets := []string{}
	rows := []EnvPackAttachmentInput{{EnvPackID: 1, ServiceNames: &targets}}
	if (&EnvPackAttachmentsParams{Attachments: &rows}).Validate() == nil {
		t.Fatal("empty selected target list accepted")
	}
	rows[0].ServiceNames = nil
	if err := (&EnvPackAttachmentsParams{Attachments: &rows}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (&EnvPackCreateParams{Name: "readable", Env: map[string]string{"EMPTY": "", "MULTILINE": "line\n\"quoted\""}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if (&EnvPackCreateParams{Name: "bad", Env: map[string]string{"NOT-A-KEY": "value"}}).Validate() == nil {
		t.Fatal("invalid key accepted")
	}
}
func TestEnvPackSDKRequestsAndServerRerun(t *testing.T) {
	paths := []string{}
	bodies := []map[string]any{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		body := map[string]any{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/playgrounds/7" {
			_, _ = w.Write([]byte(`{"id":7,"spec_id":9,"name":"source"}`))
		} else {
			_, _ = w.Write([]byte(`{"id":8,"name":"created","env":{"EMPTY":"","TEXT":"line\n\"quoted\""},"version":1}`))
		}
	}))
	defer api.Close()
	c := NewClient(WithDomain(api.URL), WithAPIKey("fixture"), WithDisableAutoConfig())
	ctx := context.Background()
	p, err := c.EnvPacks.Create(ctx, &EnvPackCreateParams{Name: "readable", Env: map[string]string{"EMPTY": "", "TEXT": "line\n\"quoted\""}})
	if err != nil || p.Env["TEXT"] != "line\n\"quoted\"" {
		t.Fatalf("response %v %v", p, err)
	}
	empty := []EnvPackAttachmentInput{}
	if _, err = c.EnvPacks.ReplaceAttachments(ctx, "playgrounds", 7, &EnvPackAttachmentsParams{Attachments: &empty}); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Tasks.Rerun(ctx, 7); err != nil {
		t.Fatal(err)
	}
	expected := []string{"POST /api/env_packs", "PATCH /api/playgrounds/7/env_pack_attachments", "GET /api/playgrounds/7", "POST /api/playgrounds/7/rerun"}
	if strings.Join(paths, "|") != strings.Join(expected, "|") {
		t.Fatalf("paths: %v", paths)
	}
	if len(bodies[1]["env_pack_attachments"].([]any)) != 0 {
		t.Fatal("empty detach lost")
	}
	if _, ok := bodies[3]["env_pack_attachments"]; ok {
		t.Fatal("rerun fabricated attachment selection")
	}
}

func TestEnvPackLaunchTransportPreservesRecipientSelection(t *testing.T) {
	var requests []map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/launches" && r.URL.Path != "/api/import_templates/12/launches" {
			t.Fatalf("unexpected launch path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		requests = append(requests, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"spec_id":7,"playground_id":8}`))
	}))
	defer api.Close()
	client := NewClient(WithDomain(api.URL), WithAPIKey("fixture"), WithDisableAutoConfig())
	empty := []EnvPackAttachmentInput{}
	names := []string{"web"}
	selected := []EnvPackAttachmentInput{{EnvPackID: 9, ServiceNames: &names}}
	for _, refs := range []*[]EnvPackAttachmentInput{nil, &empty, &selected} {
		_, err := client.Launch.Create(context.Background(), &LaunchParams{ComposeYAML: "services:\n  web:\n    image: alpine\n", Name: "fixture", EnvPackAttachments: refs})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.ImportTemplates.LaunchWithParams(context.Background(), 12, &ImportTemplateLaunchParams{HostID: 21, EnvPackAttachments: refs})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(requests) != 6 {
		t.Fatalf("requests %d", len(requests))
	}
	for _, index := range []int{0, 1} {
		if _, exists := requests[index]["env_pack_attachments"]; exists {
			t.Fatal("omitted selection changed")
		}
	}
	for _, index := range []int{2, 3} {
		if len(requests[index]["env_pack_attachments"].([]any)) != 0 {
			t.Fatal("explicit empty selection lost")
		}
	}
	for _, index := range []int{4, 5} {
		row := requests[index]["env_pack_attachments"].([]any)[0].(map[string]any)
		if row["env_pack_id"] != float64(9) || row["service_names"].([]any)[0] != "web" {
			t.Fatal("recipient selection changed")
		}
	}
	invalidTargets := []string{}
	invalid := []EnvPackAttachmentInput{{EnvPackID: 9, ServiceNames: &invalidTargets}}
	if _, err := client.Launch.Create(context.Background(), &LaunchParams{ComposeYAML: "services: {}", Name: "invalid", EnvPackAttachments: &invalid}); err == nil {
		t.Fatal("invalid launch reached server")
	}
	if _, err := client.ImportTemplates.LaunchWithParams(context.Background(), 12, &ImportTemplateLaunchParams{HostID: 21, EnvPackAttachments: &invalid}); err == nil {
		t.Fatal("invalid template launch reached server")
	}
	if len(requests) != 6 {
		t.Fatal("invalid refs caused a request")
	}
}
