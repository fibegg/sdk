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

func TestProjectForOutputFiltersTopLevelSlices(t *testing.T) {
	got := projectForOutput([]map[string]any{
		{
			"first_user_message_sentence": "hello",
			"uuid":                        "conversation-id",
		},
	}, []string{"first_user_message_sentence"})

	rows, ok := got.([]any)
	if !ok {
		t.Fatalf("projected type = %T", got)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len = %d", len(rows))
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatalf("row type = %T", rows[0])
	}
	if row["first_user_message_sentence"] != "hello" {
		t.Fatalf("first sentence = %v", row["first_user_message_sentence"])
	}
	if _, ok := row["uuid"]; ok {
		t.Fatalf("uuid was not filtered: %#v", row)
	}
}

func TestLocalConversationTableColumnsHonorOnly(t *testing.T) {
	prev := flagOnly
	flagOnly = []string{"first_user_message_sentence"}
	t.Cleanup(func() { flagOnly = prev })

	columns := selectedLocalConversationColumns()
	if len(columns) != 1 {
		t.Fatalf("columns len = %d", len(columns))
	}
	if columns[0].field != "first_user_message_sentence" {
		t.Fatalf("field = %q", columns[0].field)
	}
}

func TestLocalCommandPaths(t *testing.T) {
	root := RootCmd()
	for _, args := range [][]string{
		{"local", "playgrounds", "list"},
		{"local", "conversations", "list"},
	} {
		cmd, remaining, err := root.Find(args)
		if err != nil {
			t.Fatalf("find %v: %v", args, err)
		}
		if cmd == nil {
			t.Fatalf("find %v returned nil command", args)
		}
		if len(remaining) != 0 {
			t.Fatalf("find %v remaining = %v", args, remaining)
		}
	}
}

func TestLegacyLocalCommandPathsAreNotRegistered(t *testing.T) {
	root := RootCmd()
	for _, name := range []string{"local-playgrounds", "local-conversations"} {
		for _, cmd := range root.Commands() {
			if cmd.Name() == name {
				t.Fatalf("legacy command %q is still registered", name)
			}
			for _, alias := range cmd.Aliases {
				if alias == name {
					t.Fatalf("legacy command %q is still registered as an alias", name)
				}
			}
		}
	}
}
