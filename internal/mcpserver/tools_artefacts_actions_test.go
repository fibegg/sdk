package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestArtefactUploadUsesFilePayload(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/agents/12/artefacts" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("name"); got != "report.txt" {
			t.Fatalf("name form value = %q", got)
		}
		if got := r.FormValue("playground_id"); got != "9" {
			t.Fatalf("playground_id form value = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		_ = file.Close()
		if header.Filename != "report.txt" {
			t.Fatalf("filename = %q", header.Filename)
		}
		_, _ = w.Write([]byte(`{"id":55,"agent_id":12,"name":"report.txt"}`))
	}))
	defer api.Close()

	os.Setenv("FIBE_AGENT_ID", "12")
	defer os.Unsetenv("FIBE_AGENT_ID")

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_artefact_upload", map[string]any{
		"name":           "report.txt",
		"content_base64": "aGVsbG8=",
		"playground_id":  9,
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
}

func TestArtefactUploadWorkspaceWrite(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":55,"agent_id":12,"name":"report.txt"}`))
	}))
	defer api.Close()

	os.Setenv("FIBE_AGENT_ID", "12")
	defer os.Unsetenv("FIBE_AGENT_ID")

	tmpDir := t.TempDir()
	os.Setenv("FIBE_WORKSPACE_PATH", tmpDir)
	defer os.Unsetenv("FIBE_WORKSPACE_PATH")

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_artefact_upload", map[string]any{
		"name":           "report.txt",
		"content_base64": "aGVsbG8=",
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	content, err := os.ReadFile(tmpDir + "/report.txt")
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected content: %q", content)
	}
}

