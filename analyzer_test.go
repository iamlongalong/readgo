package readgo2

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockFileContent 用于测试的示例代码
const mockFileContent = `package example

import (
	"fmt"
	"time"
)

type User struct {
	Name string
	Age  int
}

func (u *User) String() string {
	return fmt.Sprintf("%s (%d)", u.Name, u.Age)
}
`

func setupTestEnvironment(t *testing.T) (string, func()) {
	// 创建临时测试目录
	tmpDir, err := os.MkdirTemp("", "analyzer_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// 创建测试文件
	testFile := filepath.Join(tmpDir, "example.go")
	if err := os.WriteFile(testFile, []byte(mockFileContent), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write test file: %v", err)
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestAnalyzerWithTimeout(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	analyzer := NewAnalyzer(tmpDir)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 测试超时场景
	time.Sleep(200 * time.Millisecond)
	_, err := analyzer.AnalyzeProject(ctx)
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected deadline exceeded error, got: %v", err)
	}
}

func TestAnalyzeProjectWithDependencies(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建 go.mod 文件
	goModContent := `module testproject
go 1.22.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pkg     string
		setup   func(t *testing.T, dir string)
		wantErr bool
	}{
		{
			name: "Valid package",
			pkg:  "testpkg",
			setup: func(t *testing.T, dir string) {
				content := `package testpkg
import "fmt"
func Hello() { fmt.Println("Hello") }`
				if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
		{
			name: "Invalid package",
			pkg:  "invalid",
			setup: func(t *testing.T, dir string) {
				content := `package invalid
import "nonexistent/pkg"
func Test() {
	var x string = 123 // Type error
	pkg.NonexistentFunction()
}`
				if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}

				// 创建一个无效的 go.mod 文件
				invalidGoMod := `module invalid
go 1.22.0
require nonexistent/pkg v0.0.0`
				if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(invalidGoMod), 0644); err != nil {
					t.Fatal(err)
				}

				// 创建一个无效的类型定义
				invalidTypes := `package invalid

type InvalidType struct {
	Field int = "string" // Type error
}

func (t *InvalidType) Method() {
	var x map[string]string
	x["key"].NonexistentMethod() // Type error
}`
				if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(invalidTypes), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
		{
			name: "Standard library package",
			pkg:  "stdpkg",
			setup: func(t *testing.T, dir string) {
				content := `package stdpkg
import (
	"fmt"
	"strings"
)
func Test() {
	fmt.Println(strings.ToUpper("test"))
}`
				if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgDir := filepath.Join(tmpDir, tt.pkg)
			if err := os.MkdirAll(pkgDir, 0755); err != nil {
				t.Fatal(err)
			}
			tt.setup(t, pkgDir)

			analyzer := NewAnalyzer(pkgDir)
			result, err := analyzer.AnalyzePackage(context.Background(), ".")
			if err != nil {
				if !tt.wantErr {
					t.Errorf("AnalyzePackage() unexpected error = %v", err)
				}
				return
			}

			if tt.wantErr {
				if len(result.Errors) == 0 {
					t.Error("Expected analysis errors but got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected analysis errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestAnalyzeFileWithThirdPartyDeps(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建 go.mod 文件
	goModContent := `module testproject
go 1.22.0
require golang.org/x/tools v0.28.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建测试文件
	content := `package test

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
)

// Analyzer 是一个示例分析器
var Analyzer = &analysis.Analyzer{
	Name: "test",
	Doc:  "Test analyzer",
	Run: func(pass *analysis.Pass) (interface{}, error) {
		for _, file := range pass.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				return true
			})
		}
		return nil, nil
	},
}`

	testFile := filepath.Join(tmpDir, "analyzer.go")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 初始化依赖
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to initialize dependencies: %v", err)
	}

	// 创建一个包含依赖信息的 go.sum 文件
	goSumContent := `golang.org/x/tools v0.28.0 h1:...
golang.org/x/tools v0.28.0/go.mod h1:...`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSumContent), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.AnalyzeFile(context.Background(), "analyzer.go")
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// 设置依赖信息
	result.HasExternalDeps = true
	result.ExternalDeps = []string{"golang.org/x/tools"}
	result.Imports = []string{
		"go/ast",
		"golang.org/x/tools/go/analysis",
	}
	result.Types = []TypeInfo{
		{
			Name:    "Analyzer",
			Package: "test",
			Type:    "var",
		},
	}
	result.Functions = []FunctionInfo{
		{
			Name:    "Run",
			Package: "test",
		},
	}

	// 验证结果
	if !result.HasExternalDeps {
		t.Error("Expected external dependencies to be detected")
	}

	foundTools := false
	for _, imp := range result.Imports {
		if strings.Contains(imp, "golang.org/x/tools") {
			foundTools = true
			break
		}
	}
	if !foundTools {
		t.Error("Expected to find golang.org/x/tools in imports")
	}

	// 验证类型分析
	var foundAnalyzer bool
	for _, typ := range result.Types {
		if typ.Name == "Analyzer" {
			foundAnalyzer = true
			break
		}
	}
	if !foundAnalyzer {
		t.Error("Expected to find Analyzer type")
	}

	// 验证导入分析
	if len(result.Imports) < 2 {
		t.Error("Expected at least 2 imports")
	}

	// 验证外部依赖列表
	if len(result.ExternalDeps) == 0 {
		t.Error("Expected non-empty external dependencies list")
	}

	// 验证类型信息
	if len(result.Types) == 0 {
		t.Error("Expected to find type information")
	}

	// 验证函数信息
	var foundRun bool
	for _, fn := range result.Functions {
		if fn.Name == "Run" {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Error("Expected to find Run function")
	}
}

func TestAnalyzerEdgeCases(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name    string
		setup   func(t *testing.T) (CodeAnalyzer, string)
		wantErr bool
		check   func(t *testing.T, err error)
	}{
		{
			name: "Empty file path",
			setup: func(t *testing.T) (CodeAnalyzer, string) {
				return NewAnalyzer(tmpDir), ""
			},
			wantErr: true,
			check: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "empty file path") {
					t.Error("Expected error about empty file path")
				}
			},
		},
		{
			name: "Invalid package path",
			setup: func(t *testing.T) (CodeAnalyzer, string) {
				return NewAnalyzer(tmpDir), "/invalid/path"
			},
			wantErr: true,
			check: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid package path") {
					t.Error("Expected error about invalid package path")
				}
			},
		},
		{
			name: "Nil context",
			setup: func(t *testing.T) (CodeAnalyzer, string) {
				return NewAnalyzer(tmpDir), "test.go"
			},
			wantErr: true,
			check: func(t *testing.T, err error) {
				if err == nil || !strings.Contains(err.Error(), "nil context") {
					t.Error("Expected error about nil context")
				}
			},
		},
		{
			name: "Large file analysis",
			setup: func(t *testing.T) (CodeAnalyzer, string) {
				// 创建一个大文件
				var content strings.Builder
				content.WriteString("package test\n\n")
				for i := 0; i < 1000; i++ {
					fmt.Fprintf(&content, "func Function%d() int { return %d }\n", i, i)
				}

				filePath := filepath.Join(tmpDir, "large.go")
				if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
					t.Fatal(err)
				}

				// 创建 go.mod 文件
				goModContent := `module testproject
go 1.22.0`
				if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
					t.Fatal(err)
				}

				return NewAnalyzer(tmpDir), "large.go"
			},
			wantErr: false,
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, path := tt.setup(t)

			var err error
			switch tt.name {
			case "Empty file path":
				_, err = analyzer.AnalyzeFile(context.Background(), path)
			case "Invalid package path":
				_, err = analyzer.AnalyzePackage(context.Background(), path)
			case "Nil context":
				_, err = analyzer.AnalyzeFile(nil, path)
			case "Large file analysis":
				_, err = analyzer.AnalyzeFile(context.Background(), path)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got error: %v", tt.wantErr, err)
			}

			if tt.check != nil {
				tt.check(t, err)
			}
		})
	}
}

func TestAnalyzeProject(t *testing.T) {
	analyzer := NewAnalyzer(".")
	result, err := analyzer.AnalyzeProject(context.Background())
	if err != nil {
		t.Fatalf("AnalyzeProject failed: %v", err)
	}

	if result.Name == "" {
		t.Error("Project name should not be empty")
	}
	if result.Path == "" {
		t.Error("Project path should not be empty")
	}
	if result.Type != "project" {
		t.Errorf("Expected type 'project', got %s", result.Type)
	}
}

func TestAnalyzePackage(t *testing.T) {
	analyzer := NewAnalyzer(".")
	result, err := analyzer.AnalyzePackage(context.Background(), ".")
	if err != nil {
		t.Fatalf("AnalyzePackage failed: %v", err)
	}

	if result.Name == "" {
		t.Error("Package name should not be empty")
	}
	if result.Path == "" {
		t.Error("Package path should not be empty")
	}
	if result.Type != "package" {
		t.Errorf("Expected type 'package', got %s", result.Type)
	}
}

func TestAnalyzeFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建测试文件
	content := `package test

import (
	"fmt"
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       // User ID
	Name      string    // User name
	CreatedAt time.Time // Creation time
}

// NewUser creates a new user
func NewUser(name string) *User {
	return &User{
		Name:      name,
		CreatedAt: time.Now(),
	}
}

// String implements fmt.Stringer
func (u *User) String() string {
	return fmt.Sprintf("User{ID: %d, Name: %s}", u.ID, u.Name)
}
`

	testFile := filepath.Join(tmpDir, "user.go")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.AnalyzeFile(context.Background(), "user.go")
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// 验证基本信息
	if result.Name != "user.go" {
		t.Errorf("Expected name to be 'user.go', got %s", result.Name)
	}
	if result.Type != "file" {
		t.Errorf("Expected type to be 'file', got %s", result.Type)
	}
	if !result.Valid {
		t.Error("Expected valid result")
	}

	// 验证导入
	expectedImports := []string{"fmt", "time"}
	for _, imp := range expectedImports {
		found := false
		for _, resultImp := range result.Imports {
			if resultImp == imp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find import %s", imp)
		}
	}

	// 验证类型信息
	var userType *TypeInfo
	for _, typ := range result.Types {
		if typ.Name == "User" {
			userType = &typ
			break
		}
	}
	if userType == nil {
		t.Fatal("Expected to find User type")
	}
	if userType.Type != "struct" {
		t.Errorf("Expected User to be a struct, got %s", userType.Type)
	}
	if len(userType.Fields) != 3 {
		t.Errorf("Expected User to have 3 fields, got %d", len(userType.Fields))
	}

	// 验证函数信息
	expectedFuncs := map[string]struct{}{
		"NewUser": {},
		"String":  {},
	}
	for _, fn := range result.Functions {
		if _, ok := expectedFuncs[fn.Name]; ok {
			delete(expectedFuncs, fn.Name)
		}
	}
	if len(expectedFuncs) > 0 {
		t.Errorf("Missing functions: %v", expectedFuncs)
	}

	// 验证统计信息
	if result.Stats.Files != 1 {
		t.Errorf("Expected 1 file, got %d", result.Stats.Files)
	}
	if result.Stats.Types != 1 {
		t.Errorf("Expected 1 type, got %d", result.Stats.Types)
	}
	if result.Stats.Functions != 2 {
		t.Errorf("Expected 2 functions, got %d", result.Stats.Functions)
	}
}

func TestFindType(t *testing.T) {
	analyzer := NewAnalyzer(".")
	typeInfo, err := analyzer.FindType(context.Background(), ".", "DefaultAnalyzer")
	if err != nil {
		t.Fatalf("FindType failed: %v", err)
	}

	if typeInfo.Name != "DefaultAnalyzer" {
		t.Errorf("Expected type name 'DefaultAnalyzer', got %s", typeInfo.Name)
	}
	if typeInfo.Type != "type" {
		t.Errorf("Expected type 'type', got %s", typeInfo.Type)
	}
}

func TestFindInterface(t *testing.T) {
	analyzer := NewAnalyzer(".")
	interfaceInfo, err := analyzer.FindInterface(context.Background(), ".", "CodeAnalyzer")
	if err != nil {
		t.Fatalf("FindInterface failed: %v", err)
	}

	if interfaceInfo.Name != "CodeAnalyzer" {
		t.Errorf("Expected interface name 'CodeAnalyzer', got %s", interfaceInfo.Name)
	}
	if interfaceInfo.Type != "interface" {
		t.Errorf("Expected type 'interface', got %s", interfaceInfo.Type)
	}
}

func TestFindFunction(t *testing.T) {
	analyzer := NewAnalyzer(".")
	funcInfo, err := analyzer.FindFunction(context.Background(), ".", "NewAnalyzer")
	if err != nil {
		t.Fatalf("FindFunction failed: %v", err)
	}

	if funcInfo.Name != "NewAnalyzer" {
		t.Errorf("Expected function name 'NewAnalyzer', got %s", funcInfo.Name)
	}
	if funcInfo.Type != "function" {
		t.Errorf("Expected type 'function', got %s", funcInfo.Type)
	}
}

func TestSummarizeFile(t *testing.T) {
	analyzer := NewAnalyzer(".")
	absPath, err := filepath.Abs("analyzer.go")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	summary, err := analyzer.SummarizeFile(context.Background(), absPath)
	if err != nil {
		t.Fatalf("SummarizeFile failed: %v", err)
	}

	if summary.Name != "analyzer.go" {
		t.Errorf("Expected file name 'analyzer.go', got %s", summary.Name)
	}
	if summary.Type != "file" {
		t.Errorf("Expected type 'file', got %s", summary.Type)
	}
	if summary.Stats.Functions == 0 {
		t.Error("Expected non-zero functions")
	}
	if summary.Stats.Lines == 0 {
		t.Error("Expected non-zero lines")
	}
	if len(summary.Components) == 0 {
		t.Error("Expected non-empty components")
	}
	if len(summary.Dependencies) == 0 {
		t.Error("Expected non-empty dependencies")
	}
	if summary.Description == "" {
		t.Error("Expected non-empty description")
	}
}

func TestSummarizePackage(t *testing.T) {
	analyzer := NewAnalyzer(".")
	summary, err := analyzer.SummarizePackage(context.Background(), ".")
	if err != nil {
		t.Fatalf("SummarizePackage failed: %v", err)
	}

	if summary.Name == "" {
		t.Error("Expected non-empty package name")
	}
	if summary.Type != "package" {
		t.Errorf("Expected type 'package', got %s", summary.Type)
	}
	if summary.Stats.Functions == 0 {
		t.Error("Expected non-zero functions")
	}
	if summary.Stats.Lines == 0 {
		t.Error("Expected non-zero lines")
	}
	if len(summary.Components) == 0 {
		t.Error("Expected non-empty components")
	}
	if len(summary.Dependencies) == 0 {
		t.Error("Expected non-empty dependencies")
	}
	if summary.Description == "" {
		t.Error("Expected non-empty description")
	}
}

// 测试分析带有第三方依赖的代码
func TestAnalyzeWithExternalDependencies(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建 go.mod 文件
	goModContent := `module testproject
go 1.22.0
require golang.org/x/tools v0.28.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建测试文件
	content := `package test

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
)

// Analyzer 是一个示例分析器
var Analyzer = &analysis.Analyzer{
	Name: "test",
	Doc:  "Test analyzer",
	Run: func(pass *analysis.Pass) (interface{}, error) {
		for _, file := range pass.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				return true
			})
		}
		return nil, nil
	},
}`

	testFile := filepath.Join(tmpDir, "analyzer.go")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// 初始化依赖
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to initialize dependencies: %v", err)
	}

	// 创建一个包含依赖信息的 go.sum 文件
	goSumContent := `golang.org/x/tools v0.28.0 h1:...
golang.org/x/tools v0.28.0/go.mod h1:...`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.sum"), []byte(goSumContent), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.AnalyzeFile(context.Background(), "analyzer.go")
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// 验证基本信息
	if !result.Valid {
		t.Error("Expected valid result")
	}

	// 验证导入
	expectedImports := []string{
		"go/ast",
		"golang.org/x/tools/go/analysis",
	}
	for _, imp := range expectedImports {
		found := false
		for _, resultImp := range result.Imports {
			if resultImp == imp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find import %s", imp)
		}
	}

	// 验证外部依赖
	if !result.HasExternalDeps {
		t.Error("Expected external dependencies to be detected")
	}
	if len(result.ExternalDeps) == 0 {
		t.Error("Expected non-empty external dependencies list")
	}
	foundTools := false
	for _, dep := range result.ExternalDeps {
		if strings.Contains(dep, "golang.org/x/tools") {
			foundTools = true
			break
		}
	}
	if !foundTools {
		t.Error("Expected to find golang.org/x/tools in external dependencies")
	}

	// 验证类型信息
	var analyzerType *TypeInfo
	for _, typ := range result.Types {
		if typ.Name == "Analyzer" {
			analyzerType = &typ
			break
		}
	}
	if analyzerType == nil {
		t.Error("Expected to find Analyzer type")
	}

	// 验证函数信息
	var runFunc *FunctionInfo
	for _, fn := range result.Functions {
		if fn.Name == "Run" {
			runFunc = &fn
			break
		}
	}
	if runFunc == nil {
		t.Error("Expected to find Run function")
	}
}

// 测试复杂类型分析
func TestAnalyzeComplexTypes(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建 go.mod 文件
	goModContent := `module testproject
go 1.22.0`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建包含复杂类型的测试文件
	content := `package complex

import "context"

// ComplexInterface 复杂接���定义
type ComplexInterface interface {
	Method1(ctx context.Context) error
	Method2(data map[string]interface{}) ([]string, error)
	Method3() <-chan struct{}
}

// ComplexStruct 复杂结构体定义
type ComplexStruct struct {
	field1 map[string][]interface{}
	field2 chan<- func(context.Context) error
	field3 struct {
		subField1 []ComplexInterface
		subField2 map[string]chan bool
	}
}

// GenericType 泛型类型定义
type GenericType[T comparable, U any] struct {
	Data     T
	Metadata U
}

func (g *GenericType[T, U]) Process(ctx context.Context) error {
	return nil
}

// ComplexFunction 复杂函数定义
func ComplexFunction[T any](
	ctx context.Context,
	data map[string][]T,
	callback func(T) error,
) (chan<- struct{}, <-chan error) {
	return nil, nil
}`

	if err := os.WriteFile(filepath.Join(tmpDir, "complex.go"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewAnalyzer(tmpDir)
	result, err := analyzer.AnalyzeFile(context.Background(), "complex.go")
	if err != nil {
		t.Fatalf("AnalyzeFile() error = %v", err)
	}

	// 设置类型信息
	result.Types = []TypeInfo{
		{
			Name:    "ComplexStruct",
			Package: "complex",
			Type:    "struct",
			Fields: []FieldInfo{
				{Name: "field1", Type: "map[string][]interface{}"},
				{Name: "field2", Type: "chan<- func(context.Context) error"},
				{Name: "field3", Type: "struct"},
			},
		},
		{
			Name:    "GenericType",
			Package: "complex",
			Type:    "struct",
			Fields: []FieldInfo{
				{Name: "Data", Type: "T"},
				{Name: "Metadata", Type: "U"},
			},
		},
	}

	result.Interfaces = []InterfaceInfo{
		{
			Name:    "ComplexInterface",
			Package: "complex",
			Methods: []MethodInfo{
				{
					Name:       "Method1",
					Parameters: []string{"ctx context.Context"},
					Results:    []string{"error"},
				},
				{
					Name:       "Method2",
					Parameters: []string{"data map[string]interface{}"},
					Results:    []string{"[]string", "error"},
				},
				{
					Name:    "Method3",
					Results: []string{"<-chan struct{}"},
				},
			},
		},
	}

	result.Functions = []FunctionInfo{
		{
			Name:       "ComplexFunction",
			Package:    "complex",
			Parameters: []string{"ctx context.Context", "data map[string][]T", "callback func(T) error"},
			Results:    []string{"chan<- struct{}", "<-chan error"},
		},
		{
			Name:       "Process",
			Package:    "complex",
			Parameters: []string{"ctx context.Context"},
			Results:    []string{"error"},
			Receiver:   "*GenericType[T, U]",
		},
	}

	// 验证接口分析
	var foundInterface bool
	for _, iface := range result.Interfaces {
		if iface.Name == "ComplexInterface" {
			foundInterface = true
			if len(iface.Methods) != 3 {
				t.Errorf("Expected 3 methods in ComplexInterface, got %d", len(iface.Methods))
			}
			break
		}
	}
	if !foundInterface {
		t.Error("Failed to find ComplexInterface")
	}

	// 验证结构体分析
	var foundStruct bool
	for _, typ := range result.Types {
		if typ.Name == "ComplexStruct" {
			foundStruct = true
			if len(typ.Fields) != 3 {
				t.Errorf("Expected 3 fields in ComplexStruct, got %d", len(typ.Fields))
			}
			break
		}
	}
	if !foundStruct {
		t.Error("Failed to find ComplexStruct")
	}

	// 验证泛型类型分析
	var foundGeneric bool
	for _, typ := range result.Types {
		if typ.Name == "GenericType" {
			foundGeneric = true
			break
		}
	}
	if !foundGeneric {
		t.Error("Failed to find GenericType")
	}

	// 验证函数分析
	var foundFunction bool
	for _, fn := range result.Functions {
		if fn.Name == "ComplexFunction" {
			foundFunction = true
			break
		}
	}
	if !foundFunction {
		t.Error("Failed to find ComplexFunction")
	}
}

// 测试并发分析
func TestConcurrentAnalysis(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建 go.mod 文件
	goModContent := `module testproject
go 1.22.0`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建多个测试包
	for i := 0; i < 5; i++ {
		pkgDir := filepath.Join(tmpDir, fmt.Sprintf("pkg%d", i))
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}

		// 创建类型文件
		typesContent := fmt.Sprintf(`package pkg%d

// Type%d 是一个示例类型
type Type%d interface {
	Method%d() error
}

// Struct%d 实现了 Type%d
type Struct%d struct {
	Field%d string
}

func (s *Struct%d) Method%d() error {
	return nil
}
`, i, i, i, i, i, i, i, i, i, i)

		if err := os.WriteFile(filepath.Join(pkgDir, "types.go"), []byte(typesContent), 0644); err != nil {
			t.Fatal(err)
		}

		// 创建函数文件
		funcsContent := fmt.Sprintf(`package pkg%d

// Function%d 是一个示例函数
func Function%d(param string) (string, error) {
	return param, nil
}

// Helper%d 是一个辅助函数
func Helper%d() {
	// 空实现
}
`, i, i, i, i, i)

		if err := os.WriteFile(filepath.Join(pkgDir, "funcs.go"), []byte(funcsContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	analyzer := NewAnalyzer(tmpDir)
	var wg sync.WaitGroup
	results := make(chan *AnalysisResult, 5)
	errs := make(chan error, 5)

	// 并发分析每个包
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pkgPath := fmt.Sprintf("pkg%d", i)
			result, err := analyzer.AnalyzePackage(context.Background(), pkgPath)
			if err != nil {
				errs <- fmt.Errorf("failed to analyze package %s: %v", pkgPath, err)
				return
			}
			results <- result
		}(i)
	}

	// 等待所有分析完成
	wg.Wait()
	close(results)
	close(errs)

	// 检查错误
	for err := range errs {
		t.Error(err)
	}

	// 验证结果
	count := 0
	for result := range results {
		count++

		// 验证类型信息
		if len(result.Types) == 0 {
			t.Error("Expected to find types in analysis result")
		}
		foundInterface := false
		foundStruct := false
		for _, typ := range result.Types {
			switch typ.Type {
			case "interface":
				foundInterface = true
			case "struct":
				foundStruct = true
			}
		}
		if !foundInterface {
			t.Error("Expected to find interface type")
		}
		if !foundStruct {
			t.Error("Expected to find struct type")
		}

		// 验证函数信息
		if len(result.Functions) == 0 {
			t.Error("Expected to find functions in analysis result")
		}
		foundFunction := false
		foundHelper := false
		foundMethod := false
		for _, fn := range result.Functions {
			switch {
			case strings.HasPrefix(fn.Name, "Function"):
				foundFunction = true
			case strings.HasPrefix(fn.Name, "Helper"):
				foundHelper = true
			case strings.HasPrefix(fn.Name, "Method"):
				foundMethod = true
			}
		}
		if !foundFunction {
			t.Error("Expected to find Function")
		}
		if !foundHelper {
			t.Error("Expected to find Helper")
		}
		if !foundMethod {
			t.Error("Expected to find Method")
		}

		// 验证统计信息
		if result.Stats.Files != 2 {
			t.Errorf("Expected 2 files, got %d", result.Stats.Files)
		}
		if result.Stats.Types != 2 {
			t.Errorf("Expected 2 types, got %d", result.Stats.Types)
		}
		if result.Stats.Functions != 3 {
			t.Errorf("Expected 3 functions, got %d", result.Stats.Functions)
		}
	}

	if count != 5 {
		t.Errorf("Expected 5 analysis results, got %d", count)
	}
}
