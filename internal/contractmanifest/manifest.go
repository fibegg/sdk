// Package contractmanifest renders a deterministic description of the public
// Go API. It intentionally records source-level declarations so struct field
// order, tags, awkward names, and public Unwrap methods remain visible.
package contractmanifest

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
	"sort"
	"strings"
)

type Manifest struct {
	Version      int           `json:"version"`
	Package      string        `json:"package"`
	Declarations []Declaration `json:"declarations"`
}

type Declaration struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Signature string `json:"signature"`
}

func Generate(dir string) ([]byte, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make(map[string]*ast.File)
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
		if file.Name.Name == "fibe" {
			files[path] = file
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("package fibe not found in %s", dir)
	}
	manifest := Manifest{Version: 1, Package: "github.com/fibegg/sdk/fibe"}
	for _, filename := range sortedFileNames(files) {
		file := files[filename]
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.GenDecl:
				manifest.Declarations = append(manifest.Declarations, declarationsFromGen(fset, typed)...)
			case *ast.FuncDecl:
				if declaration, ok := declarationFromFunc(fset, typed); ok {
					manifest.Declarations = append(manifest.Declarations, declaration)
				}
			}
		}
	}
	sort.Slice(manifest.Declarations, func(i, j int) bool {
		return manifest.Declarations[i].ID < manifest.Declarations[j].ID
	})
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func declarationsFromGen(fset *token.FileSet, decl *ast.GenDecl) []Declaration {
	var declarations []Declaration
	for _, raw := range decl.Specs {
		switch spec := raw.(type) {
		case *ast.TypeSpec:
			if !spec.Name.IsExported() {
				continue
			}
			publicSpec := publicTypeSpec(spec)
			declarations = append(declarations, Declaration{
				ID:        "type." + spec.Name.Name,
				Kind:      "type",
				Signature: "type " + renderNode(fset, publicSpec),
			})
		case *ast.ValueSpec:
			for index, name := range spec.Names {
				if !name.IsExported() {
					continue
				}
				var signature strings.Builder
				signature.WriteString(decl.Tok.String())
				signature.WriteByte(' ')
				signature.WriteString(name.Name)
				if spec.Type != nil {
					signature.WriteByte(' ')
					signature.WriteString(renderNode(fset, spec.Type))
				}
				if index < len(spec.Values) {
					signature.WriteString(" = ")
					signature.WriteString(renderNode(fset, spec.Values[index]))
				} else if len(spec.Values) == 1 {
					signature.WriteString(" = ")
					signature.WriteString(renderNode(fset, spec.Values[0]))
				}
				declarations = append(declarations, Declaration{
					ID:        decl.Tok.String() + "." + name.Name,
					Kind:      decl.Tok.String(),
					Signature: signature.String(),
				})
			}
		}
	}
	return declarations
}

func publicTypeSpec(spec *ast.TypeSpec) *ast.TypeSpec {
	copy := *spec
	structure, ok := spec.Type.(*ast.StructType)
	if !ok {
		return &copy
	}
	structureCopy := *structure
	fieldsCopy := *structure.Fields
	fieldsCopy.List = make([]*ast.Field, 0, len(structure.Fields.List))
	for _, field := range structure.Fields.List {
		if publicStructField(field) {
			fieldsCopy.List = append(fieldsCopy.List, field)
		}
	}
	structureCopy.Fields = &fieldsCopy
	copy.Type = &structureCopy
	return &copy
}

func publicStructField(field *ast.Field) bool {
	if len(field.Names) > 0 {
		for _, name := range field.Names {
			if name.IsExported() {
				return true
			}
		}
		return false
	}
	return ast.IsExported(receiverName(field.Type))
}

func declarationFromFunc(fset *token.FileSet, decl *ast.FuncDecl) (Declaration, bool) {
	if !decl.Name.IsExported() {
		return Declaration{}, false
	}
	kind := "func"
	id := "func." + decl.Name.Name
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		receiver := receiverName(decl.Recv.List[0].Type)
		if receiver == "" || !ast.IsExported(receiver) {
			return Declaration{}, false
		}
		kind = "method"
		id = "method." + receiver + "." + decl.Name.Name
	}
	copy := *decl
	copy.Doc = nil
	copy.Body = nil
	return Declaration{ID: id, Kind: kind, Signature: renderNode(fset, &copy)}, true
}

func receiverName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return receiverName(typed.X)
	case *ast.IndexExpr:
		return receiverName(typed.X)
	case *ast.IndexListExpr:
		return receiverName(typed.X)
	default:
		return ""
	}
}

func renderNode(fset *token.FileSet, node any) string {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		panic(err)
	}
	return buffer.String()
}

func sortedFileNames(files map[string]*ast.File) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func CompatibilityErrors(baseline, current *Manifest) []string {
	currentByID := make(map[string]Declaration, len(current.Declarations))
	for _, declaration := range current.Declarations {
		currentByID[declaration.ID] = declaration
	}
	var errors []string
	for _, declaration := range baseline.Declarations {
		candidate, ok := currentByID[declaration.ID]
		if !ok {
			errors = append(errors, declaration.ID+" was removed")
			continue
		}
		if candidate.Signature != declaration.Signature {
			errors = append(errors, declaration.ID+" changed")
		}
	}
	sort.Strings(errors)
	return errors
}
