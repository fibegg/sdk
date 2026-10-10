package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

// Companion to cmd/fibe/naming_vocabulary_test.go (acceptance row N01-008).
// That test walks the Cobra tree and every MCP tool name and input schema;
// this one covers the MCP protocol surfaces it cannot reach from package
// main: resources, resource templates, the on-the-wire tools/list, and the
// bodies of the static schema resources.
//
// There is deliberately no allow-list: the removed-name errors
// (domainnames.RemovedError, rejectRemovedMCPRequest) are produced at call
// time from the caller's own input and never appear in a listing or schema.

var removedWordSet = map[string]bool{
	"marque": true, "marquee": true, "marquees": true,
	"playspec": true, "playspecs": true,
	"prop": true, "props": true,
	"trick": true, "tricks": true,
	"genie": true, "genies": true,
	"scroll": true, "scrolls": true,
	"crumb": true, "crumbs": true,
}

var camelWordBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])|([A-Z]+)([A-Z][a-z])`)

func removedWordsIn(text string) []string {
	text = camelWordBoundary.ReplaceAllString(text, "${1}${3} ${2}${4}")
	var found []string
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if removedWordSet[strings.ToLower(word)] {
			found = append(found, word)
		}
	}
	return found
}

func TestNoRemovedVocabularyInMCPProtocolSurface(t *testing.T) {
	cfg := mockServerConfig()
	cfg.ToolSet = "full"
	server := New(cfg)
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	call := func(id int, method, params string) string {
		t.Helper()
		message := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q,"params":%s}`, id, method, params)
		data, err := json.Marshal(server.mcp.HandleMessage(context.Background(), json.RawMessage(message)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), `"error"`) {
			t.Fatalf("%s failed: %s", method, data)
		}
		return string(data)
	}
	for name, body := range map[string]string{
		"tools/list":               call(1, "tools/list", `{}`),
		"resources/list":           call(2, "resources/list", `{}`),
		"resources/templates/list": call(3, "resources/templates/list", `{}`),
		"fibe://schema":            call(4, "resources/read", `{"uri":"fibe://schema"}`),
		"fibe://pipeline/schema":   call(5, "resources/read", `{"uri":"fibe://pipeline/schema"}`),
	} {
		if len(body) < 200 {
			t.Fatalf("%s returned %d bytes; the surface walk is not covering anything", name, len(body))
		}
		// Resource bodies are JSON strings inside the envelope; decode the
		// envelope so escaped content is scanned as plain text.
		var decoded any
		if err := json.Unmarshal([]byte(body), &decoded); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var texts []string
		collectJSONText(decoded, &texts)
		for _, text := range texts {
			if found := removedWordsIn(text); len(found) > 0 {
				t.Errorf("removed vocabulary %v in MCP %s:\n    %q", found, name, truncateForReport(text))
			}
			// A resource body is itself JSON: scan its keys and values too.
			var inner any
			if strings.HasPrefix(strings.TrimSpace(text), "{") && json.Unmarshal([]byte(text), &inner) == nil {
				var innerTexts []string
				collectJSONText(inner, &innerTexts)
				for _, innerText := range innerTexts {
					if found := removedWordsIn(innerText); len(found) > 0 {
						t.Errorf("removed vocabulary %v in MCP %s body:\n    %q", found, name, truncateForReport(innerText))
					}
				}
			}
		}
	}
}

func collectJSONText(value any, out *[]string) {
	switch value := value.(type) {
	case map[string]any:
		for key, item := range value {
			*out = append(*out, key)
			collectJSONText(item, out)
		}
	case []any:
		for _, item := range value {
			collectJSONText(item, out)
		}
	case string:
		*out = append(*out, value)
	}
}

func truncateForReport(text string) string {
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}

func TestRemovedWordMatcherIsIdentifierAware(t *testing.T) {
	for text, want := range map[string]int{
		"marquee_id": 1, "playspecId": 1, "PropBranch": 1, "fibe_tricks_list": 1, "Scrolls and Crumbs": 2,
		"property": 0, "proper": 0, "propagate": 0, "scrolling": 0, "tricky": 0, "hosts specs repositories tasks": 0,
	} {
		if got := len(removedWordsIn(text)); got != want {
			t.Errorf("%q: found %d; want %d", text, got, want)
		}
	}
}
