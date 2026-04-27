package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSchemaListPrintsResourceCatalog(t *testing.T) {
	out, err := captureStdout(func() error {
		cmd := schemaCmd()
		cmd.SetArgs([]string{"--list"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("schema --list: %v", err)
	}
	if !strings.Contains(out, `"name": "playground"`) || !strings.Contains(out, `"aliases"`) {
		t.Fatalf("expected resource catalog output, got:\n%s", out)
	}
}

func TestSchemaResourceListOperation(t *testing.T) {
	out, err := captureStdout(func() error {
		cmd := schemaCmd()
		cmd.SetArgs([]string{"playgrounds", "list"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("schema playgrounds list: %v", err)
	}
	if !strings.Contains(out, `"playground.list"`) || !strings.Contains(out, `"per_page"`) {
		t.Fatalf("expected playground list schema, got:\n%s", out)
	}
}

func TestSchemaResourceUpdateOperation(t *testing.T) {
	out, err := captureStdout(func() error {
		cmd := schemaCmd()
		cmd.SetArgs([]string{"agent", "update"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("schema agent update: %v", err)
	}
	for _, want := range []string{`"agent.update"`, `"agent_id"`, `"model_options"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %s in agent update schema, got:\n%s", want, out)
		}
	}
}

func TestSchemaResourceOperationFlags(t *testing.T) {
	out, err := captureStdout(func() error {
		cmd := schemaCmd()
		cmd.SetArgs([]string{"--resource", "agent", "--operation", "create"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("schema --resource agent --operation create: %v", err)
	}
	for _, want := range []string{`"agent.create"`, `"name"`, `"provider"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %s in schema output, got:\n%s", want, out)
		}
	}
}

func TestSchemaCustomCreateOperations(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{args: []string{"artefact", "create"}, want: []string{`"artefact.create"`, `"agent_id"`, `"playground_id"`, `"content_base64"`}},
		{args: []string{"mutter", "create"}, want: []string{`"mutter.create"`, `"agent_id"`, `"type"`, `"body"`}},
		{args: []string{"template_version", "create"}, want: []string{`"template_version.create"`, `"template_id"`, `"template_body_path"`, `"response_mode"`}},
		{args: []string{"template", "develop"}, want: []string{`"template.develop"`, `"target_type"`, `"base_version_id"`, `"post_apply"`}},
	} {
		out, err := captureStdout(func() error {
			cmd := schemaCmd()
			cmd.SetArgs(tc.args)
			return cmd.Execute()
		})
		if err != nil {
			t.Fatalf("schema %s: %v", strings.Join(tc.args, " "), err)
		}
		for _, want := range tc.want {
			if !strings.Contains(out, want) {
				t.Fatalf("expected %s in schema %s, got:\n%s", want, strings.Join(tc.args, " "), out)
			}
		}
	}
}

func TestSchemaScopedMutationActionOperations(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{args: []string{"marquee", "autoconnect_token"}, want: []string{`"marquee.autoconnect_token"`, `"ssl_mode"`, `"dns_credentials"`}},
		{args: []string{"marquee", "generate_ssh_key"}, want: []string{`"marquee.generate_ssh_key"`, `"marquee_id"`, `"minimum": 1`}},
		{args: []string{"prop", "attach"}, want: []string{`"prop.attach"`, `"repo_full_name"`}},
		{args: []string{"prop", "mirror"}, want: []string{`"prop.mirror"`, `"source_url"`}},
		{args: []string{"template", "source_set"}, want: []string{`"template.source_set"`, `"source_prop_id"`, `"source_path"`}},
		{args: []string{"template", "upgrade_playspecs"}, want: []string{`"template.upgrade_playspecs"`, `"template_id"`, `"version_id"`}},
		{args: []string{"template_version", "toggle_public"}, want: []string{`"template_version.toggle_public"`, `"template_id"`, `"version_id"`}},
		{args: []string{"trick", "trigger"}, want: []string{`"trick.trigger"`, `"playspec_id"`}},
		{args: []string{"webhook", "test"}, want: []string{`"webhook.test"`, `"webhook_id"`}},
	} {
		out, err := captureStdout(func() error {
			cmd := schemaCmd()
			cmd.SetArgs(tc.args)
			return cmd.Execute()
		})
		if err != nil {
			t.Fatalf("schema %s: %v", strings.Join(tc.args, " "), err)
		}
		for _, want := range tc.want {
			if !strings.Contains(out, want) {
				t.Fatalf("expected %s in schema %s, got:\n%s", want, strings.Join(tc.args, " "), out)
			}
		}
	}
}

func captureStdout(fn func() error) (string, error) {
	prev := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := buf.ReadFrom(r)
		done <- copyErr
	}()

	runErr := fn()
	_ = w.Close()
	os.Stdout = prev
	copyErr := <-done
	_ = r.Close()
	if runErr != nil {
		return "", runErr
	}
	return buf.String(), copyErr
}
