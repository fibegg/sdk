package testedition

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModes(t *testing.T) {
	for _, value := range []string{"", Full, "enterprise", Community} {
		mode, e := Mode(value)
		if e != nil || mode == "" {
			t.Fatalf("%q: %s %v", value, mode, e)
		}
	}
	for _, value := range []string{"comunity", "Community", "enterprise ", "none"} {
		if _, e := Mode(value); e == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}
func TestSelectionAndMixedDecisions(t *testing.T) {
	m := Manifest{Rules: []Rule{{Package: "integration", Name: "TestMixed", Edition: "mixed", Reason: "review", Cases: map[string]string{"core operation": "community", "vault operation": "enterprise", "nested": "mixed", "nested/core": "community"}}}}
	cases, e := m.Select(Community, "integration", "^TestMixed$")
	if e != nil || len(cases) != 2 {
		t.Fatalf("%v %v", cases, e)
	}
	for _, pattern := range []string{"Misspelled", "^TestMixed$/vault_operation", "^TestMixed$/core_operatioN", "^TestMixed$/core_operation/unknown", "["} {
		if _, e := m.Select(Community, "integration", pattern); e == nil {
			t.Fatalf("accepted empty selection %q", pattern)
		}
	}
	if _, _, e := m.Decision("integration", "TestMixed/unknown"); e == nil {
		t.Fatal("accepted unknown dynamic child")
	}
	if _, _, e := m.Decision("integration", "TestNew"); e == nil {
		t.Fatal("accepted unknown test")
	}
	if e, _, err := m.Decision("integration", "TestMixed/nested/core"); err != nil || e != Community {
		t.Fatalf("nested: %s %v", e, err)
	}
}
func TestGoRunPatterns(t *testing.T) {
	for _, tc := range []struct {
		pattern, name string
		want          bool
	}{
		{"^TestA$", "TestA/space name", true},
		{"^TestA$/space_name$", "TestA/space name", true},
		{"^TestA$/space_name$/typo", "TestA/space name", false},
		{"^TestB$|^TestA$/space", "TestA/space name", true},
		{"^Test(A|B)$/[a/b]", "TestA/a", true},
		{"^TestA$/nope", "TestA/space name", false},
	} {
		got, e := Matches(tc.pattern, tc.name)
		if e != nil || got != tc.want {
			t.Fatalf("%q %q got %v %v", tc.pattern, tc.name, got, e)
		}
	}
}
func TestClassificationCoverage(t *testing.T) {
	root, e := Root()
	if e != nil {
		t.Fatal(e)
	}
	m, e := Load()
	if e != nil {
		t.Fatal(e)
	}
	if e = Validate(root, m); e != nil {
		t.Fatal(e)
	}
}
func TestNewAndUnclassifiedCasesFail(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "integration")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	write := func(source string) {
		t.Helper()
		if e := os.WriteFile(filepath.Join(dir, "new_test.go"), []byte(source), 0600); e != nil {
			t.Fatal(e)
		}
	}
	m := Manifest{Version: 1, Rules: []Rule{{Package: "integration", Name: "TestKnown", Edition: Community, Reason: "core", Inventory: []string{"known"}}}}
	write(`package integration; import "testing";func TestKnown(t *testing.T){requireEdition(t);t.Run("known",func(t *testing.T){})};func TestNew(t *testing.T){}`)
	if e := Validate(root, m); e == nil || !strings.Contains(e.Error(), "unclassified live test") {
		t.Fatalf("new test: %v", e)
	}
	write(`package integration; import "testing";func TestKnown(t *testing.T){requireEdition(t);t.Run("new",func(t *testing.T){})}`)
	if e := Validate(root, m); e == nil || !strings.Contains(e.Error(), "unlisted literal") {
		t.Fatalf("new literal: %v", e)
	}
	m.Rules[0].Edition = "mixed"
	m.Rules[0].Cases = map[string]string{"known": Community}
	write(`package integration; import "testing";func TestKnown(t *testing.T){requireEdition(t);t.Run("known",func(t *testing.T){})}`)
	if e := Validate(root, m); e == nil || !strings.Contains(e.Error(), "unguarded") {
		t.Fatalf("unguarded mixed: %v", e)
	}
	write(`package integration; import "testing";func TestKnown(t *testing.T){requireEdition(t);name:="known";runEdition(t,name,func(t *testing.T){})}`)
	if e := Validate(root, m); e == nil || !strings.Contains(e.Error(), "dynamic case review") {
		t.Fatalf("unreviewed dynamic: %v", e)
	}
}
func TestCommunityServerIdentity(t *testing.T) {
	for _, edition := range []string{"community", "enterprise", ""} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/route-manifest.json" || r.Header.Get("Authorization") != "" {
				t.Error("unexpected manifest request")
			}
			_, _ = w.Write([]byte(`{"edition":"` + edition + `"}`))
		}))
		err := CheckServer(srv.URL)
		srv.Close()
		if (err == nil) != (edition == Community) {
			t.Fatalf("edition %q: %v", edition, err)
		}
	}
}
