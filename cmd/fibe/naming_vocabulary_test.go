package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/fibegg/sdk/internal/domainnames"
	"github.com/fibegg/sdk/internal/mcpserver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// removedVocabulary lists the pre-rename FIBE words (lower case, singular and
// plural). Matching is by identifier word: text is split on every
// non-alphanumeric character (space, "_", "-", ".", "/") and on camelCase
// boundaries, so "marquee_id", "--playspec", "PropBranch" and "fibe_props_list"
// are caught while "property", "proper", "propagate", "scrolling" or
// "tricky" are not.
var removedVocabulary = map[string]bool{
	"marque": true, "marquee": true, "marquees": true,
	"playspec": true, "playspecs": true,
	"prop": true, "props": true,
	"trick": true, "tricks": true,
	"genie": true, "genies": true,
	"scroll": true, "scrolls": true,
	"crumb": true, "crumbs": true,
}

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])|([A-Z]+)([A-Z][a-z])`)

// removedWords returns the removed words found in text, in order of appearance.
func removedWords(text string) []string {
	text = camelBoundary.ReplaceAllString(text, "${1}${3} ${2}${4}")
	words := strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var found []string
	for _, word := range words {
		if removedVocabulary[strings.ToLower(word)] {
			found = append(found, word)
		}
	}
	return found
}

type vocabularySurface struct {
	where string // human-readable location, e.g. "cli: fibe playgrounds create --host (usage)"
	text  string
}

func TestRemovedVocabularyMatcher(t *testing.T) {
	for text, want := range map[string]int{
		"marquee_id": 1, "--playspec": 1, "PropBranch": 1, "fibe_props_list": 1,
		"Trick runs": 1, "genie": 1, "Scrolls and Crumbs": 2, "playspecId": 1,
		"property": 0, "proper": 0, "propagate": 0, "scrolling": 0, "tricky": 0,
		"hosts specs repositories tasks": 0, "audit_logs library agent": 0,
	} {
		if got := len(removedWords(text)); got != want {
			t.Errorf("%q: found %d removed words; want %d", text, got, want)
		}
	}
}

// cliSurfaces walks the whole Cobra tree (hidden commands included) and
// returns every user-visible string: command path, aliases, Use, Short, Long,
// Example, Deprecated, and for every local and persistent flag its name,
// shorthand, usage, default value and annotations.
func cliSurfaces(root *cobra.Command) []vocabularySurface {
	var out []vocabularySurface
	add := func(where, text string) {
		if text != "" {
			out = append(out, vocabularySurface{where, text})
		}
	}
	seenFlags := map[*pflag.Flag]bool{}
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		path := cmd.CommandPath()
		add("cli command path: "+path, path)
		add("cli use: "+path, cmd.Use)
		for _, alias := range cmd.Aliases {
			add("cli alias: "+path, alias)
		}
		add("cli short: "+path, cmd.Short)
		add("cli long: "+path, cmd.Long)
		add("cli example: "+path, cmd.Example)
		add("cli deprecated: "+path, cmd.Deprecated)
		for _, arg := range cmd.ValidArgs {
			add("cli valid arg: "+path, arg)
		}
		for key, values := range cmd.Annotations {
			add("cli annotation key: "+path, key)
			add("cli annotation value: "+path, values)
		}
		visit := func(flag *pflag.Flag) {
			if seenFlags[flag] {
				return
			}
			seenFlags[flag] = true
			where := fmt.Sprintf("cli flag --%s (first seen on %s)", flag.Name, path)
			add(where+" name", flag.Name)
			add(where+" shorthand", flag.Shorthand)
			add(where+" usage", flag.Usage)
			add(where+" default", flag.DefValue)
			add(where+" deprecated", flag.Deprecated)
			for key, values := range flag.Annotations {
				add(where+" annotation key", key)
				add(where+" annotation values", strings.Join(values, " "))
			}
		}
		cmd.LocalFlags().VisitAll(visit)
		cmd.PersistentFlags().VisitAll(visit)
		cmd.InheritedFlags().VisitAll(visit)
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)
	return out
}

// jsonStrings flattens every object key and string value of a decoded JSON
// document into path-qualified surfaces: property names, enum members,
// descriptions, defaults and titles all count.
func jsonStrings(where string, value any, out *[]vocabularySurface) {
	switch value := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			*out = append(*out, vocabularySurface{where + " key", key})
			jsonStrings(where+"."+key, value[key], out)
		}
	case []any:
		for i, item := range value {
			jsonStrings(fmt.Sprintf("%s[%d]", where, i), item, out)
		}
	case string:
		*out = append(*out, vocabularySurface{where, value})
	}
}

// mcpSurfaces registers the complete (all tiers) MCP tool catalog exactly as
// `fibe mcp docs` does and returns each tool's name, description and JSON
// input schema.
func mcpSurfaces(t *testing.T, root *cobra.Command) []vocabularySurface {
	t.Helper()
	cfg := mcpserver.DefaultConfig()
	cfg.CobraRoot = root
	cfg.ToolSet = "full"
	server := mcpserver.New(cfg)
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	tools := server.AllTools()
	if len(tools) < 50 {
		t.Fatalf("only %d MCP tools registered; the catalog walk is not covering the real surface", len(tools))
	}
	var out []vocabularySurface
	for _, tool := range tools {
		out = append(out,
			vocabularySurface{"mcp tool name: " + tool.Name, tool.Name},
			vocabularySurface{"mcp tool description: " + tool.Name, tool.Description},
		)
		if tool.InputSchema != nil {
			data, err := json.Marshal(tool.InputSchema)
			if err != nil {
				t.Fatal(err)
			}
			var decoded any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			jsonStrings("mcp tool schema: "+tool.Name, decoded, &out)
		}
	}
	return out
}

func TestNoRemovedVocabularyInCLIAndMCPSurface(t *testing.T) {
	root := RootCmd()
	cli := cliSurfaces(root)
	if len(cli) < 1000 {
		t.Fatalf("only %d CLI strings collected; the Cobra walk is not covering the real tree", len(cli))
	}
	// Removed short aliases (mq, ps, pr, tr) are too short and too common to
	// be matched as words; compare command names and aliases to the declared
	// removed names exactly instead.
	var checkNames func(*cobra.Command)
	checkNames = func(cmd *cobra.Command) {
		for _, name := range append([]string{cmd.Name()}, cmd.Aliases...) {
			if replacement, removed := domainnames.Replacement(name); removed {
				t.Errorf("command %q uses removed name or alias %q (replacement %q)", cmd.CommandPath(), name, replacement)
			}
		}
		for _, child := range cmd.Commands() {
			checkNames(child)
		}
	}
	checkNames(root)
	surfaces := append(cli, mcpSurfaces(t, root)...)
	// There is deliberately no allow-list for these static surfaces. The
	// removed-name errors (see removed_names.go, internal/domainnames and
	// internal/mcpserver/naming_transport.go) are built at call time from the
	// caller's own input and never appear in help, flags or schemas; they are
	// exercised by naming_contract_test.go. The only static carrier of an old
	// word is Cobra's SuggestFor, checked separately below.
	for _, surface := range surfaces {
		if found := removedWords(surface.text); len(found) > 0 {
			t.Errorf("removed vocabulary %v in %s:\n    %q", found, surface.where, surface.text)
		}
	}
}

// TestSuggestForCarriesOnlyDeclaredRemovedNames is the single permitted static
// appearance of old words in the command tree. installRemovedNameErrors adds a
// removed name to the SuggestFor of the command that replaced it, so Cobra
// answers "unknown command" with the new name. SuggestFor is matched against
// the caller's input and is never rendered, so it is not user-visible text.
// Every entry must be a removed name declared in domainnames.Resources whose
// replacement is the command that carries it; anything else is a leak.
func TestSuggestForCarriesOnlyDeclaredRemovedNames(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, old := range cmd.SuggestFor {
			replacement, ok := domainnames.Replacement(old)
			if !ok || strings.ReplaceAll(replacement, "_", "-") != cmd.Name() {
				t.Errorf("%s: SuggestFor %q is not a removed name that maps to this command", cmd.CommandPath(), old)
			}
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(RootCmd())
}

// TestVocabularyWalkDetectsPlantedRemovedNames proves the walkers see every
// kind of surface they claim to: a clean tree is silent, and each planted old
// word is reported at the right place.
func TestVocabularyWalkDetectsPlantedRemovedNames(t *testing.T) {
	build := func(planted bool) *cobra.Command {
		use, alias, flag, usage, long := "hosts", "hs", "host", "Host to use", "Manages hosts."
		if planted {
			use, alias, flag, usage, long = "marquees", "mq", "playspec-id", "The prop to use", "Runs a trick."
		}
		root := &cobra.Command{Use: "fibe"}
		child := &cobra.Command{Use: use, Aliases: []string{alias}, Long: long, Run: func(*cobra.Command, []string) {}}
		child.Flags().String(flag, "", usage)
		root.AddCommand(child)
		return root
	}
	count := func(surfaces []vocabularySurface) (n int) {
		for _, surface := range surfaces {
			n += len(removedWords(surface.text))
		}
		return n
	}
	if n := count(cliSurfaces(build(false))); n != 0 {
		t.Fatalf("clean tree reported %d removed words", n)
	}
	var hits []string
	for _, surface := range cliSurfaces(build(true)) {
		if len(removedWords(surface.text)) > 0 {
			hits = append(hits, surface.where)
		}
	}
	joined := strings.Join(hits, "\n")
	for _, want := range []string{"cli use:", "cli long:", "--playspec-id (first seen on fibe marquees) name", "--playspec-id (first seen on fibe marquees) usage"} {
		if !strings.Contains(joined, want) {
			t.Errorf("planted removed name not reported at %q; got:\n%s", want, joined)
		}
	}
	var schema []vocabularySurface
	jsonStrings("schema", map[string]any{"properties": map[string]any{"marquee_id": map[string]any{"description": "Playspec", "enum": []any{"prop"}}}}, &schema)
	if n := count(schema); n != 3 {
		t.Errorf("planted schema key, description and enum member: found %d removed words; want 3", n)
	}
}
