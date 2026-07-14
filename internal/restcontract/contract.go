// Package restcontract derives a deterministic, language-neutral inventory of
// SDK service operations from the Go implementation.
package restcontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Contract is the portable REST and serialization contract shared by SDKs.
type Contract struct {
	Version         int            `json:"version"`
	Authentication  Authentication `json:"authentication"`
	Serialization   Serialization  `json:"serialization"`
	Operations      []Operation    `json:"operations"`
	CustomJSONTypes []string       `json:"custom_json_types"`
}

// Authentication records headers applied by the shared executor.
type Authentication struct {
	Header            string `json:"header"`
	Scheme            string `json:"scheme"`
	OmitWhenKeyEmpty  bool   `json:"omit_when_key_empty"`
	UserAgentPrefix   string `json:"user_agent_prefix"`
	IdempotencyHeader string `json:"idempotency_header"`
	RequestIDHeader   string `json:"request_id_header"`
}

// Serialization records behavior common to all operations.
type Serialization struct {
	RequestContentType  string `json:"request_content_type"`
	ResponseContentType string `json:"response_content_type"`
	UnknownFields       string `json:"unknown_fields"`
	TrailingJSON        string `json:"trailing_json"`
	QueryEncoding       string `json:"query_encoding"`
	SuccessBodyLimit    int64  `json:"success_body_limit_bytes"`
	ErrorBodyLimit      int64  `json:"error_body_limit_bytes"`
}

// Operation describes one exported service method.
type Operation struct {
	Name           string `json:"name"`
	HTTPMethod     string `json:"http_method,omitempty"`
	PathTemplate   string `json:"path_template,omitempty"`
	PathExpression string `json:"path_expression,omitempty"`
	Codec          string `json:"codec"`
	RequestType    string `json:"request_type,omitempty"`
	ResponseType   string `json:"response_type,omitempty"`
	DelegatesTo    string `json:"delegates_to,omitempty"`
}

// Generate renders the REST contract from a fibe package directory.
func Generate(dir string) ([]byte, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	contract := Contract{
		Version: 1,
		Authentication: Authentication{
			Header: "Authorization", Scheme: "Bearer", OmitWhenKeyEmpty: true,
			UserAgentPrefix: "fibe-go/", IdempotencyHeader: "Idempotency-Key", RequestIDHeader: "X-Request-Id",
		},
		Serialization: Serialization{
			RequestContentType: "application/json", ResponseContentType: "application/json",
			UnknownFields: "accepted", TrailingJSON: "rejected", QueryEncoding: "RFC 3986 via net/url.Values.Encode",
			SuccessBodyLimit: 10 << 20, ErrorBodyLimit: 1 << 20,
		},
	}
	custom := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, raw := range file.Decls {
			fn, ok := raw.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			receiver := receiverName(fn.Recv.List[0].Type)
			if receiver == "" {
				continue
			}
			if fn.Name.Name == "MarshalJSON" || fn.Name.Name == "UnmarshalJSON" {
				custom[receiver] = true
			}
			if !strings.HasSuffix(receiver, "Service") {
				continue
			}
			contract.Operations = append(contract.Operations, operationFromFunc(fset, receiver, fn))
		}
	}
	for name := range custom {
		contract.CustomJSONTypes = append(contract.CustomJSONTypes, name)
	}
	sort.Strings(contract.CustomJSONTypes)
	sort.Slice(contract.Operations, func(i, j int) bool { return contract.Operations[i].Name < contract.Operations[j].Name })
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func operationFromFunc(fset *token.FileSet, receiver string, fn *ast.FuncDecl) Operation {
	op := Operation{
		Name:         receiver + "." + fn.Name.Name,
		Codec:        "delegated",
		RequestType:  requestType(fset, fn.Type.Params),
		ResponseType: renderResults(fset, fn.Type.Results),
	}
	vars := localExpressions(fn.Body)
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if helper, ok := transportHelper(selector); ok && op.HTTPMethod == "" {
			methodIndex, pathIndex := 1, 2
			op.Codec = helper
			if selector.Sel.Name == "doDownload" {
				op.HTTPMethod, methodIndex, pathIndex = "GET", -1, 1
			}
			if methodIndex >= 0 && len(call.Args) > methodIndex {
				op.HTTPMethod = httpMethod(call.Args[methodIndex])
			}
			if len(call.Args) > pathIndex {
				op.PathExpression = render(fset, call.Args[pathIndex])
				op.PathTemplate = templateFor(call.Args[pathIndex], vars)
			}
			return true
		}
		if callName(call.Fun) == "doList" && op.HTTPMethod == "" && len(call.Args) > 2 {
			op.HTTPMethod = "GET"
			op.Codec = "paginated-json"
			op.PathExpression = render(fset, call.Args[2])
			op.PathTemplate = templateFor(call.Args[2], vars)
			return true
		}
		if op.DelegatesTo == "" {
			if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == receiverVariable(fn) && selector.Sel.IsExported() {
				op.DelegatesTo = receiver + "." + selector.Sel.Name
			} else if target := delegatedService(selector, receiverVariable(fn)); target != "" {
				op.DelegatesTo = target
			}
		}
		return true
	})
	if op.HTTPMethod == "" && strings.Contains(fn.Name.Name, "Stream") {
		op.Codec = "websocket"
	}
	if op.HTTPMethod != "" {
		op.DelegatesTo = ""
	}
	return op
}

func callName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.IndexExpr:
		return callName(value.X)
	case *ast.IndexListExpr:
		return callName(value.X)
	default:
		return ""
	}
}

func delegatedService(selector *ast.SelectorExpr, receiver string) string {
	resource, ok := selector.X.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	client, ok := resource.X.(*ast.SelectorExpr)
	if !ok || client.Sel.Name != "client" {
		return ""
	}
	root, ok := client.X.(*ast.Ident)
	if !ok || root.Name != receiver {
		return ""
	}
	return resource.Sel.Name + "." + selector.Sel.Name
}

func transportHelper(selector *ast.SelectorExpr) (string, bool) {
	inner, ok := selector.X.(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != "client" {
		return "", false
	}
	switch selector.Sel.Name {
	case "do":
		return "json", true
	case "doAsync":
		return "async-json", true
	case "doMultipart":
		return "multipart", true
	case "doDownload":
		return "download", true
	default:
		return "", false
	}
}

func localExpressions(body *ast.BlockStmt) map[string]ast.Expr {
	values := map[string]ast.Expr{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch statement := node.(type) {
		case *ast.AssignStmt:
			for i, left := range statement.Lhs {
				if ident, ok := left.(*ast.Ident); ok && i < len(statement.Rhs) {
					values[ident.Name] = statement.Rhs[i]
				}
			}
		case *ast.DeclStmt:
			if declaration, ok := statement.Decl.(*ast.GenDecl); ok {
				for _, raw := range declaration.Specs {
					if spec, ok := raw.(*ast.ValueSpec); ok {
						for i, name := range spec.Names {
							if i < len(spec.Values) {
								values[name.Name] = spec.Values[i]
							}
						}
					}
				}
			}
		}
		return true
	})
	return values
}

func templateFor(expr ast.Expr, vars map[string]ast.Expr) string {
	switch value := expr.(type) {
	case *ast.BasicLit:
		if value.Kind == token.STRING {
			text, _ := strconv.Unquote(value.Value)
			return text
		}
	case *ast.Ident:
		if resolved, ok := vars[value.Name]; ok && resolved != expr {
			return templateFor(resolved, vars)
		}
		return "{" + value.Name + "}"
	case *ast.BinaryExpr:
		if value.Op == token.ADD {
			return templateFor(value.X, vars) + templateFor(value.Y, vars)
		}
	case *ast.CallExpr:
		if ident, ok := value.Fun.(*ast.Ident); ok {
			switch ident.Name {
			case "identifierPath":
				if len(value.Args) == 2 {
					return templateFor(value.Args[0], vars) + "/" + templateParameter(value.Args[1])
				}
			case "buildQuery":
				return "{?query}"
			}
		}
		if selector, ok := value.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "Sprintf" && len(value.Args) > 0 {
			formatText := templateFor(value.Args[0], vars)
			for _, arg := range value.Args[1:] {
				match := regexp.MustCompile(`%[-+#0-9.]*[a-zA-Z]`).FindStringIndex(formatText)
				if match == nil {
					break
				}
				formatText = formatText[:match[0]] + templateParameter(arg) + formatText[match[1]:]
			}
			return formatText
		}
	case *ast.SelectorExpr:
		if value.Sel.Name == "Encode" {
			return "{?query}"
		}
	}
	return "{" + strings.ReplaceAll(render(token.NewFileSet(), expr), " ", "") + "}"
}

func templateParameter(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return "{" + ident.Name + "}"
	}
	return "{value}"
}

func httpMethod(expr ast.Expr) string {
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		return strings.TrimPrefix(strings.ToUpper(selector.Sel.Name), "METHOD")
	}
	if literal, ok := expr.(*ast.BasicLit); ok {
		value, _ := strconv.Unquote(literal.Value)
		return value
	}
	return render(token.NewFileSet(), expr)
}

func requestType(fset *token.FileSet, fields *ast.FieldList) string {
	if fields == nil {
		return ""
	}
	for i := len(fields.List) - 1; i >= 0; i-- {
		field := fields.List[i]
		if rendered := render(fset, field.Type); rendered != "context.Context" && rendered != "string" && rendered != "int64" && rendered != "int" && rendered != "bool" && rendered != "time.Duration" {
			return rendered
		}
	}
	return ""
}

func renderResults(fset *token.FileSet, fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields.List))
	for _, field := range fields.List {
		parts = append(parts, render(fset, field.Type))
	}
	return strings.Join(parts, ", ")
}

func receiverVariable(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 && len(fn.Recv.List[0].Names) > 0 {
		return fn.Recv.List[0].Names[0].Name
	}
	return ""
}

func receiverName(expr ast.Expr) string {
	if pointer, ok := expr.(*ast.StarExpr); ok {
		expr = pointer.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func render(fset *token.FileSet, node any) string {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		return fmt.Sprint(node)
	}
	return buffer.String()
}
