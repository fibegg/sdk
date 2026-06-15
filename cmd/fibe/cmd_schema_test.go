package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSchemaCustomCreateOperations(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{args: []string{"artefact", "create"}, want: []string{`"artefact.create"`, `"agent_id_or_name"`, `"playground_id_or_name"`, `"content_base64"`}},
		{args: []string{"mutter", "create"}, want: []string{`"mutter.create"`, `"agent_id_or_name"`, `"type"`, `"body"`}},
		{args: []string{"template_version", "create"}, want: []string{`"template_version.create"`, `"template_id_or_name"`, `"template_body_path"`, `"response_mode"`}},
		{args: []string{"template", "change"}, want: []string{`"template.change"`, `"target_type"`, `"base_version_id"`, `"post_apply"`}},
		{args: []string{"playground", "switch_template"}, want: []string{`"playground.switch_template"`, `"id_or_name"`, `"provision_missing_props"`}},
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
		{args: []string{"marquee", "generate_ssh_key"}, want: []string{`"marquee.generate_ssh_key"`, `"id_or_name"`, `"minimum": 1`}},
		{args: []string{"prop", "attach"}, want: []string{`"prop.attach"`, `"repo_full_name"`}},
		{args: []string{"prop", "mirror"}, want: []string{`"prop.mirror"`, `"source_url"`}},
		{args: []string{"template", "source_set"}, want: []string{`"template.source_set"`, `"source_prop_id_or_name"`, `"source_path"`}},
		{args: []string{"template", "upgrade_playspecs"}, want: []string{`"template.upgrade_playspecs"`, `"template_id_or_name"`, `"version_id"`}},
		{args: []string{"template_version", "toggle_public"}, want: []string{`"template_version.toggle_public"`, `"template_id_or_name"`, `"version_id"`}},
		{args: []string{"trick", "trigger"}, want: []string{`"trick.trigger"`, `"playspec_id_or_name"`}},
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
