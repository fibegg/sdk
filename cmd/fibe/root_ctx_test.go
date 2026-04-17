package main

import (
	"context"
	"testing"
)

type ctxKey string

func TestCtxUsesCurrentCommandContext(t *testing.T) {
	setCommandContext(context.WithValue(context.Background(), ctxKey("trace"), "ok"))
	t.Cleanup(func() { setCommandContext(nil) })

	if got := ctx().Value(ctxKey("trace")); got != "ok" {
		t.Fatalf("ctx value = %v, want ok", got)
	}
}
