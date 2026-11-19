// Package testedition selects reviewed live SDK tests without changing client behavior.
package testedition

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
)

const Environment = "FIBE_TEST_EDITION"
const Full = "full"
const Community = "community"

//go:embed cases.json
var files embed.FS

type Rule struct {
	Package   string            `json:"package"`
	Name      string            `json:"name"`
	Edition   string            `json:"edition"`
	Reason    string            `json:"reason"`
	Cases     map[string]string `json:"cases,omitempty"`
	Inventory []string          `json:"inventory,omitempty"`
	Review    string            `json:"dynamic_review,omitempty"`
}
type Manifest struct {
	Version int    `json:"version"`
	Rules   []Rule `json:"rules"`
}
type Case struct {
	Package string `json:"package"`
	Name    string `json:"name"`
	Edition string `json:"edition"`
	Reason  string `json:"reason"`
}

func Mode(value string) (string, error) {
	if value == "" || value == "enterprise" {
		return Full, nil
	}
	if value != Full && value != Community {
		return "", fmt.Errorf("unknown %s %q: use full, enterprise or community", Environment, value)
	}
	return value, nil
}
func Load() (Manifest, error) {
	var m Manifest
	b, err := files.ReadFile("cases.json")
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	return m, err
}
func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		b, e := os.ReadFile(filepath.Join(dir, "go.mod"))
		if e == nil && strings.Contains(string(b), "module github.com/fibegg/sdk") {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("SDK source root not found")
		}
		dir = next
	}
}

// Validate makes changed or newly introduced live tests require an explicit review.
// Only dynamic test functions need a canonical function fingerprint: their generated
// names cannot be exhaustively inferred from literal t.Run calls. Other source
// changes and formatting never require a fingerprint refresh.
func Validate(root string, m Manifest) error {
	if m.Version != 1 || len(m.Rules) == 0 {
		return fmt.Errorf("invalid or empty edition manifest")
	}
	declared := map[string]Rule{}
	for _, r := range m.Rules {
		k := r.Package + "/" + r.Name
		if _, ok := declared[k]; ok {
			return fmt.Errorf("duplicate classification %s", k)
		}
		if r.Package != "integration" && r.Package != "internal/mcpserver" {
			return fmt.Errorf("unknown package %s", r.Package)
		}
		if r.Edition != "community" && r.Edition != "enterprise" && r.Edition != "mixed" {
			return fmt.Errorf("invalid classification %s", k)
		}
		if r.Reason == "" {
			return fmt.Errorf("missing review reason %s", k)
		}
		if r.Edition == "mixed" && len(r.Cases) == 0 {
			return fmt.Errorf("mixed suite has no case decisions: %s", k)
		}
		for n, e := range r.Cases {
			if n == "" || (e != "community" && e != "enterprise" && e != "mixed") {
				return fmt.Errorf("invalid subcase %s/%s", k, n)
			}
		}
		declared[k] = r
	}
	seen := map[string]bool{}
	for _, pkg := range []string{"integration", "internal/mcpserver"} {
		entries, err := filepath.Glob(filepath.Join(root, pkg, "*_test.go"))
		if err != nil {
			return err
		}
		for _, path := range entries {
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			f, e := parser.ParseFile(token.NewFileSet(), path, b, 0)
			if e != nil {
				return e
			}

			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Name.Name == "TestMain" || !strings.HasPrefix(fn.Name.Name, "Test") {
					continue
				}
				live := pkg == "integration" || calls(fn, "requireRealServer")
				k := pkg + "/" + fn.Name.Name
				if r, ok := declared[k]; ok {
					seen[k] = true
					if pkg == "integration" && !calls(fn, "requireEdition") {
						return fmt.Errorf("live test has no edition guard: %s", k)
					}
					literals, dynamic, unguarded := subtests(fn)
					if r.Edition == "mixed" && unguarded {
						return fmt.Errorf("mixed suite has an unguarded t.Run: %s", k)
					}
					for _, name := range literals {
						if r.Edition == "mixed" {
							if _, ok := r.Cases[name]; !ok {
								return fmt.Errorf("unclassified literal subtest %s/%s", k, name)
							}
						} else {
							known := false
							for _, n := range r.Inventory {
								if normalized(n) == normalized(name) {
									known = true
								}
							}
							if !known {
								return fmt.Errorf("unlisted literal subtest %s/%s", k, name)
							}
						}
					}
					if dynamic && r.Review != fingerprint(fn) {
						return fmt.Errorf("dynamic case review is stale or missing for %s; review its generated case names, update decisions and dynamic_review", k)
					}
				} else if live {
					return fmt.Errorf("unclassified live test %s", k)
				}
			}
		}
	}
	for k := range declared {
		if !seen[k] {
			return fmt.Errorf("classification references a missing test %s", k)
		}
	}
	return nil
}

// Fingerprint prints a canonical function digest for a deliberately reviewed dynamic test.
func DynamicReviews(root string) (map[string]string, error) {
	out := map[string]string{}
	for _, pkg := range []string{"integration", "internal/mcpserver"} {
		entries, _ := filepath.Glob(filepath.Join(root, pkg, "*_test.go"))
		for _, p := range entries {
			f, e := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if e != nil {
				return nil, e
			}
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || !strings.HasPrefix(fn.Name.Name, "Test") {
					continue
				}
				_, dynamic, _ := subtests(fn)
				if dynamic {
					out[pkg+"/"+fn.Name.Name] = fingerprint(fn)
				}
			}
		}
	}
	return out, nil
}
func Fingerprint(root, qualified string) (string, error) {
	index := strings.LastIndex(qualified, "/")
	if index < 0 {
		return "", fmt.Errorf("use package/TestName")
	}
	pkg, name := qualified[:index], qualified[index+1:]
	if pkg != "integration" && pkg != "internal/mcpserver" {
		return "", fmt.Errorf("unknown test package %s", pkg)
	}
	entries, _ := filepath.Glob(filepath.Join(root, pkg, "*_test.go"))
	for _, path := range entries {
		f, e := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if e != nil {
			return "", e
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if ok && fn.Name.Name == name {
				return fingerprint(fn), nil
			}
		}
	}
	return "", fmt.Errorf("test not found: %s", qualified)
}
func fingerprint(fn *ast.FuncDecl) string {
	var b bytes.Buffer
	_ = format.Node(&b, token.NewFileSet(), fn)
	sum := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(sum[:])
}
func subtests(fn *ast.FuncDecl) ([]string, bool, bool) {
	names := []string{}
	dynamic, unguarded := false, false
	var walk func(ast.Node, string)
	walk = func(node ast.Node, parent string) {
		ast.Inspect(node, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			nameIndex := -1
			if selector, ok := c.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Run" {
				if id, ok := selector.X.(*ast.Ident); ok && id.Name == "t" {
					nameIndex = 0
					unguarded = true
				}
			}
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "runEdition" {
				nameIndex = 1
			}
			if nameIndex < 0 || len(c.Args) <= nameIndex {
				return true
			}
			literal, ok := c.Args[nameIndex].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				dynamic = true
				// Nested cases still need gates even when their parent name is generated.
				for _, arg := range c.Args[nameIndex+1:] {
					if body, ok := arg.(*ast.FuncLit); ok {
						ast.Inspect(body, func(n ast.Node) bool {
							call, ok := n.(*ast.CallExpr)
							if ok {
								if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Run" {
									if id, ok := sel.X.(*ast.Ident); ok && id.Name == "t" {
										unguarded = true
									}
								}
							}
							return true
						})
					}
				}
				return false
			}
			value, e := strconv.Unquote(literal.Value)
			if e != nil {
				return false
			}
			path := value
			if parent != "" {
				path = parent + "/" + value
			}
			names = append(names, path)
			for _, arg := range c.Args[nameIndex+1:] {
				if body, ok := arg.(*ast.FuncLit); ok {
					walk(body.Body, path)
				}
			}
			return false
		})
	}
	walk(fn.Body, "")
	return names, dynamic, unguarded
}
func calls(fn *ast.FuncDecl, name string) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if ok {
			id, ok := c.Fun.(*ast.Ident)
			if ok && id.Name == name {
				found = true
			}
		}
		return true
	})
	return found
}
func normalized(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '_'
		}
		return r
	}, s)
}
func (m Manifest) Cases() []Case {
	out := []Case{}
	for _, r := range m.Rules {
		if r.Edition == "mixed" {
			for n, e := range r.Cases {
				if e != "mixed" {
					out = append(out, Case{r.Package, r.Name + "/" + normalized(n), e, r.Reason})
				}
			}
			continue
		}
		if len(r.Inventory) == 0 {
			out = append(out, Case{r.Package, r.Name, r.Edition, r.Reason})
			continue
		}
		for _, n := range r.Inventory {
			out = append(out, Case{r.Package, r.Name + "/" + normalized(n), r.Edition, r.Reason})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Package+out[i].Name < out[j].Package+out[j].Name })
	return out
}

// splitPattern honors the unbracketed slash and alternative separators used by go test.
func splitPattern(pattern string) [][]string {
	alternatives := [][]string{}
	parts := []string{}
	start, brackets, parens := 0, 0, 0
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			i++
		case '[':
			brackets++
		case ']':
			if brackets > 0 {
				brackets--
			}
		case '(':
			if brackets == 0 {
				parens++
			}
		case ')':
			if brackets == 0 {
				parens--
			}
		case '/', '|':
			if brackets == 0 && parens == 0 {
				parts = append(parts, pattern[start:i])
				start = i + 1
				if pattern[i] == '|' {
					alternatives = append(alternatives, parts)
					parts = nil
				}
			}
		}
	}
	return append(alternatives, append(parts, pattern[start:]))
}

// Matches accepts parent selections, but never invents descendants for a misspelled leaf.
// Reviewed table names are listed explicitly; opaque runtime matrices use their root.
func Matches(pattern, name string) (bool, error) {
	if pattern == "" {
		pattern = "."
	}
	nn := strings.Split(normalized(name), "/")
	matched := false
	for _, parts := range splitPattern(pattern) {
		ok := len(parts) <= len(nn)
		for i, p := range parts {
			re, e := regexp.Compile(p)
			if e != nil {
				return false, e
			}
			if i < len(nn) && !re.MatchString(nn[i]) {
				ok = false
			}
		}
		matched = matched || ok
	}
	return matched, nil
}
func (m Manifest) Select(mode, pkg, pattern string) ([]Case, error) {
	resolved, err := Mode(mode)
	if err != nil {
		return nil, err
	}
	out := []Case{}
	for _, c := range m.Cases() {
		if pkg != "" && c.Package != pkg {
			continue
		}
		if resolved == Community && c.Edition != "community" {
			continue
		}
		match, e := Matches(pattern, c.Name)
		if e != nil {
			return nil, e
		}
		if match {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("edition selection matched no cases (edition=%s package=%s run=%q)", resolved, pkg, pattern)
	}
	return out, nil
}
func (m Manifest) Decision(pkg, name string) (string, string, error) {
	parts := strings.Split(name, "/")
	for _, r := range m.Rules {
		if r.Package != pkg || r.Name != parts[0] {
			continue
		}
		if len(parts) == 1 {
			return r.Edition, r.Reason, nil
		}
		if r.Edition != "mixed" {
			path := strings.Join(parts[1:], "/")
			for _, n := range r.Inventory {
				n = normalized(n)
				if path == n || strings.HasPrefix(path, n+"/") || strings.HasPrefix(n, path+"/") {
					return r.Edition, r.Reason, nil
				}
			}
			return "", "", fmt.Errorf("unclassified subtest %s/%s", pkg, name)
		}
		path := strings.Join(parts[1:], "/")
		best, decision := "", ""
		for n, e := range r.Cases {
			n = normalized(n)
			if (path == n || strings.HasPrefix(path, n+"/")) && len(n) > len(best) {
				best = n
				decision = e
			}
		}
		if decision == "" {
			return "", "", fmt.Errorf("unclassified mixed case %s/%s", pkg, name)
		}
		return decision, r.Reason, nil
	}
	return "", "", fmt.Errorf("unclassified test %s/%s", pkg, name)
}

var once sync.Once
var active Manifest
var activeMode string
var initError error

func Initialize() error {
	once.Do(func() {
		activeMode, initError = Mode(os.Getenv(Environment))
		if initError != nil || activeMode == Full {
			return
		}
		active, initError = Load()
		if initError != nil {
			return
		}
		root, e := Root()
		if e != nil {
			initError = e
			return
		}
		initError = Validate(root, active)
	})
	return initError
}
func Current() string {
	if err := Initialize(); err != nil {
		panic(err)
	}
	return activeMode
}

// Prepare is called by the actual Go test executable before fixture bootstrap.
// Community -list enumerates selected cases offline. The default full run is unchanged.
func Prepare(pkg string, allowUnit bool) (bool, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	if err := Initialize(); err != nil {
		return false, err
	}
	list := flag.Lookup("test.list").Value.String()
	if activeMode == Full {
		return list != "", nil
	}
	pattern := flag.Lookup("test.run").Value.String()
	if list != "" {
		pattern = list
	}
	selected, err := active.Select(activeMode, pkg, pattern)
	if err != nil && allowUnit && list == "" {
		root, e := Root()
		if e == nil && unitMatches(root, pkg, pattern) {
			return false, nil
		}
	}
	if err != nil {
		return false, err
	}
	if list != "" {
		for _, c := range selected {
			fmt.Println(c.Name)
		}
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "SDK edition=community selected=%d package=%s\n", len(selected), pkg)
	return false, nil
}
func unitMatches(root, pkg, pattern string) bool {
	entries, _ := filepath.Glob(filepath.Join(root, pkg, "*_test.go"))
	for _, p := range entries {
		f, e := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if e != nil {
			continue
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "TestMain" || !strings.HasPrefix(fn.Name.Name, "Test") || calls(fn, "requireRealServer") {
				continue
			}
			match, e := Matches(pattern, fn.Name.Name)
			if e == nil && match {
				return true
			}
		}
	}
	return false
}
func Require(t *testing.T, pkg string) {
	t.Helper()
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	if activeMode == Full {
		return
	}
	decision, reason, err := active.Decision(pkg, t.Name())
	if err != nil {
		t.Fatal(err)
	}
	if decision == "enterprise" {
		t.Skipf("edition=community: Enterprise prerequisite (%s)", reason)
	}
}
func Run(t *testing.T, pkg, name string, body func(*testing.T)) bool {
	t.Helper()
	return t.Run(name, func(c *testing.T) { Require(c, pkg); body(c) })
}

var verifyOnce sync.Once
var verifyError error

// VerifyServer prevents a Community qualification against an Enterprise executable.
// It uses the public edition manifest and never sends credentials.
func VerifyServer(t *testing.T, domain string) {
	t.Helper()
	if Current() != Community {
		return
	}
	verifyOnce.Do(func() { verifyError = CheckServer(domain) })
	if verifyError != nil {
		t.Fatal(verifyError)
	}
}
func CheckServer(domain string) error {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimRight(domain, "/") + "/route-manifest.json")
	if err != nil {
		return fmt.Errorf("verify Community server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Community route manifest returned HTTP %d", resp.StatusCode)
	}
	var manifest struct {
		Edition string `json:"edition"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&manifest); err != nil {
		return fmt.Errorf("read Community route manifest: %w", err)
	}
	if manifest.Edition != Community {
		return fmt.Errorf("Community tests require edition=community; server reports %q", manifest.Edition)
	}
	return nil
}
