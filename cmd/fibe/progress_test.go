package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStatusLineInteractiveRewritesAndClearsSingleLine(t *testing.T) {
	var out bytes.Buffer
	interactive := true
	line := newStatusLine(&out, statusLineOptions{
		Interactive: &interactive,
		Interval:    time.Hour,
		Frames:      []string{"-"},
	})

	line.Start("status: in_progress")
	line.Update("status: running")
	line.Stop()

	got := out.String()
	if strings.Contains(got, "\n") {
		t.Fatalf("interactive status line should not emit newlines: %q", got)
	}
	if !strings.Contains(got, "\r- status: in_progress") || !strings.Contains(got, "\r- status: running") {
		t.Fatalf("missing interactive rewrites: %q", got)
	}
	if !strings.HasSuffix(got, "\r") {
		t.Fatalf("status line should end cleared at column zero: %q", got)
	}
}

func TestStatusLineInteractivePadsShorterMessages(t *testing.T) {
	var out bytes.Buffer
	interactive := true
	line := newStatusLine(&out, statusLineOptions{
		Interactive: &interactive,
		Interval:    time.Hour,
		Frames:      []string{"-"},
	})

	line.Start("status: in_progress")
	line.Update("ok")
	line.Stop()

	got := out.String()
	if !strings.Contains(got, "\r- ok                 ") {
		t.Fatalf("shorter update should clear previous characters: %q", got)
	}
}

func TestStatusLineNonInteractiveFallbackUpdates(t *testing.T) {
	var out bytes.Buffer
	interactive := false
	line := newStatusLine(&out, statusLineOptions{
		Interactive:     &interactive,
		FallbackUpdates: true,
	})

	line.Update("status: pending")
	line.Update("status: running")
	line.Stop()

	want := "status: pending\nstatus: running\n"
	if out.String() != want {
		t.Fatalf("fallback output = %q, want %q", out.String(), want)
	}
}

func TestStatusLineNonInteractiveSilentByDefault(t *testing.T) {
	var out bytes.Buffer
	interactive := false
	line := newStatusLine(&out, statusLineOptions{Interactive: &interactive})

	line.Start("working")
	line.Update("still working")
	line.Stop()

	if out.Len() != 0 {
		t.Fatalf("expected silent non-interactive status line, got %q", out.String())
	}
}
