package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeGoFile создаёт временный Go-файл с заданным содержимым.
func writeGoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// hasGenerateResetComment

func TestHasGenerateResetComment_Nil(t *testing.T) {
	assert.False(t, hasGenerateResetComment(nil))
}

func TestHasGenerateResetComment_WithComment(t *testing.T) {
	cg := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// generate:reset"}},
	}
	assert.True(t, hasGenerateResetComment(cg))
}

func TestHasGenerateResetComment_WithoutComment(t *testing.T) {
	cg := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "// just a comment"}},
	}
	assert.False(t, hasGenerateResetComment(cg))
}

// findStructsInFile

func TestFindStructsInFile_WithAnnotation(t *testing.T) {
	src := `package testpkg

// generate:reset
type MyStruct struct {
	Name string
	Age  int
}
`
	dir := t.TempDir()
	path := writeGoFile(t, dir, "test.go", src)

	structs, pkg, err := findStructsInFile(path)
	require.NoError(t, err)
	assert.Equal(t, "testpkg", pkg)
	assert.Len(t, structs, 1)
	assert.Equal(t, "MyStruct", structs[0].Name)
}

func TestFindStructsInFile_NoAnnotation(t *testing.T) {
	src := `package testpkg

type Plain struct {
	Value float64
}
`
	dir := t.TempDir()
	path := writeGoFile(t, dir, "plain.go", src)

	structs, pkg, err := findStructsInFile(path)
	require.NoError(t, err)
	assert.Equal(t, "testpkg", pkg)
	assert.Empty(t, structs)
}

func TestFindStructsInFile_InvalidFile(t *testing.T) {
	_, _, err := findStructsInFile("/nonexistent/file.go")
	assert.Error(t, err)
}

func TestFindStructsInFile_NotAStruct(t *testing.T) {
	src := `package testpkg

// generate:reset
type MyAlias = string
`
	dir := t.TempDir()
	path := writeGoFile(t, dir, "alias.go", src)

	structs, _, err := findStructsInFile(path)
	require.NoError(t, err)
	assert.Empty(t, structs)
}

// generateResetMethod

func TestGenerateResetMethod_BasicFields(t *testing.T) {
	src := `package p
type S struct {
	Name string
	Age  int
	Flag bool
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "s.go", src, 0)
	require.NoError(t, err)

	var fields []*ast.Field
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					fields = st.Fields.List
				}
			}
		}
	}

	var buf bytes.Buffer
	err = generateResetMethod(&buf, structInfo{Name: "S", Fields: fields})
	require.NoError(t, err)

	code := buf.String()
	assert.Contains(t, code, "func (s *S) Reset()")
	assert.Contains(t, code, `s.Name = ""`)
	assert.Contains(t, code, "s.Age = 0")
}

// generateResetFile

func TestGenerateResetFile(t *testing.T) {
	src := `package mypkg

// generate:reset
type Config struct {
	Host string
	Port int
}
`
	dir := t.TempDir()
	srcPath := writeGoFile(t, dir, "config.go", src)

	structs, _, err := findStructsInFile(srcPath)
	require.NoError(t, err)
	require.Len(t, structs, 1)

	pkg := &packageStructs{
		PackageName: "mypkg",
		PackagePath: dir,
		Structs:     structs,
		Imports:     make(map[string]bool),
	}

	err = generateResetFile(pkg)
	require.NoError(t, err)

	generated := filepath.Join(dir, "reset.gen.go")
	data, err := os.ReadFile(generated)
	require.NoError(t, err)
	assert.Contains(t, string(data), "func (c *Config) Reset()")
}

// collectImports

func TestCollectImports_Primitive(t *testing.T) {
	src := `package p
type S struct { Name string }
`
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "f.go", src, 0)

	var fields []*ast.Field
	for _, d := range f.Decls {
		if gd, ok := d.(*ast.GenDecl); ok {
			for _, s := range gd.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						fields = st.Fields.List
					}
				}
			}
		}
	}

	imports := make(map[string]bool)
	collectImports(fields, imports)
	// Примитивные типы не требуют импортов
	assert.Empty(t, imports)
}

// formatType

func TestFormatType_Ident(t *testing.T) {
	result := formatType(&ast.Ident{Name: "string"})
	assert.Equal(t, "string", result)
}

func TestFormatType_StarIdent(t *testing.T) {
	result := formatType(&ast.StarExpr{X: &ast.Ident{Name: "Config"}})
	assert.Equal(t, "*Config", result)
}

func TestFormatType_ArrayType(t *testing.T) {
	result := formatType(&ast.ArrayType{Elt: &ast.Ident{Name: "int"}})
	assert.Equal(t, "[]int", result)
}

func TestFormatType_MapType(t *testing.T) {
	result := formatType(&ast.MapType{
		Key:   &ast.Ident{Name: "string"},
		Value: &ast.Ident{Name: "int"},
	})
	assert.Equal(t, "map[string]int", result)
}

func TestFormatType_SelectorExpr(t *testing.T) {
	result := formatType(&ast.SelectorExpr{
		X:   &ast.Ident{Name: "sync"},
		Sel: &ast.Ident{Name: "Mutex"},
	})
	assert.Equal(t, "sync.Mutex", result)
}

func TestFormatType_Unknown(t *testing.T) {
	result := formatType(&ast.BasicLit{})
	assert.Equal(t, "unknown", result)
}

// generateFieldReset

func TestGenerateFieldReset_AllFieldTypes(t *testing.T) {
	src := `package p
import ("time"; "sync")
type S struct {
	Name     string
	Count    int
	Flag     bool
	Score    float64
	Items    []string
	Arr      [3]int
	Meta     map[string]int
	Created  time.Time
	Mu       sync.Mutex
	Ch       chan int
	Fn       func()
	Iface    interface{}
	PtrStr   *string
	PtrTime  *time.Time
	Unknown  S
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "s.go", src, parser.ParseComments)
	require.NoError(t, err)

	var fields []*ast.Field
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					fields = st.Fields.List
				}
			}
		}
	}

	for _, field := range fields {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			result := generateFieldReset("s", name.Name, field.Type)
			assert.NotEmpty(t, result, "field %s should produce non-empty reset code", name.Name)
		}
	}
}

// generatePointerReset

func TestGeneratePointerReset_Primitive(t *testing.T) {
	result := generatePointerReset("s", "Name", "s.Name", &ast.Ident{Name: "string"})
	assert.Contains(t, result, "*s.Name")
}

func TestGeneratePointerReset_NamedType(t *testing.T) {
	result := generatePointerReset("s", "Val", "s.Val", &ast.Ident{Name: "MyStruct"})
	assert.Contains(t, result, "resetter")
}

func TestGeneratePointerReset_SelectorExpr(t *testing.T) {
	sel := &ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}}
	result := generatePointerReset("s", "Created", "s.Created", sel)
	assert.Contains(t, result, "time.Time{}")
}

func TestGeneratePointerReset_StructType(t *testing.T) {
	result := generatePointerReset("s", "Nested", "s.Nested", &ast.StructType{Fields: &ast.FieldList{}})
	assert.Contains(t, result, "*s.Nested")
}

func TestGeneratePointerReset_Default(t *testing.T) {
	// Тип по умолчанию (не примитив, не selector, не struct)
	result := generatePointerReset("s", "Items", "s.Items", &ast.ArrayType{Elt: &ast.Ident{Name: "string"}})
	assert.Contains(t, result, "resetter")
}

// collectImportsFromType

func TestCollectImportsFromType_Time(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}}, imports)
	assert.True(t, imports["time"])
}

func TestCollectImportsFromType_Context(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.SelectorExpr{X: &ast.Ident{Name: "context"}, Sel: &ast.Ident{Name: "Context"}}, imports)
	assert.True(t, imports["context"])
}

func TestCollectImportsFromType_Sync(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.SelectorExpr{X: &ast.Ident{Name: "sync"}, Sel: &ast.Ident{Name: "Mutex"}}, imports)
	assert.True(t, imports["sync"])
}

func TestCollectImportsFromType_UnknownPackage(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.SelectorExpr{X: &ast.Ident{Name: "custom"}, Sel: &ast.Ident{Name: "T"}}, imports)
	assert.Empty(t, imports)
}

func TestCollectImportsFromType_StarExpr(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.StarExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}}}, imports)
	assert.True(t, imports["time"])
}

func TestCollectImportsFromType_ArrayType(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.ArrayType{Elt: &ast.SelectorExpr{X: &ast.Ident{Name: "sync"}, Sel: &ast.Ident{Name: "Mutex"}}}, imports)
	assert.True(t, imports["sync"])
}

func TestCollectImportsFromType_MapType(t *testing.T) {
	imports := make(map[string]bool)
	collectImportsFromType(&ast.MapType{
		Key:   &ast.Ident{Name: "string"},
		Value: &ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}},
	}, imports)
	assert.True(t, imports["time"])
}

// generateResetFile with imports

func TestGenerateResetFile_WithTimeField(t *testing.T) {
	src := `package mypkg

import "time"

// generate:reset
type Event struct {
	Name    string
	Created time.Time
}
`
	dir := t.TempDir()
	srcPath := writeGoFile(t, dir, "event.go", src)

	structs, _, err := findStructsInFile(srcPath)
	require.NoError(t, err)
	require.Len(t, structs, 1)

	pkg := &packageStructs{
		PackageName: "mypkg",
		PackagePath: dir,
		Structs:     structs,
		Imports:     make(map[string]bool),
	}

	err = generateResetFile(pkg)
	require.NoError(t, err)

	generated := filepath.Join(dir, "reset.gen.go")
	data, err := os.ReadFile(generated)
	require.NoError(t, err)
	assert.Contains(t, string(data), "func (e *Event) Reset()")
	assert.Contains(t, string(data), `"time"`)
}

// run (integration-like)

func TestRun_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	err := run()
	// Нет файлов с generate:reset — должно завершиться без ошибки
	assert.NoError(t, err)
}

func TestRun_WithAnnotatedStruct(t *testing.T) {
	dir := t.TempDir()
	src := `package main

// generate:reset
type Point struct {
	X float64
	Y float64
}
`
	writeGoFile(t, dir, "point.go", src)

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	err := run()
	require.NoError(t, err)

	generated := filepath.Join(dir, "reset.gen.go")
	data, err := os.ReadFile(generated)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "func (p *Point) Reset()"))
}
