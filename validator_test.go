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
)

func TestValidateFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建测试文件
	validContent := `package test
func Valid() {}`
	validFile := filepath.Join(tmpDir, "valid.go")
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	invalidContent := `package test
func Invalid() {
	var x string = 123 // Type error
}`
	invalidFile := filepath.Join(tmpDir, "invalid.go")
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator(tmpDir)

	tests := []struct {
		name    string
		file    string
		level   ValidationLevel
		wantErr bool
		check   func(t *testing.T, result *ValidationResult)
	}{
		{
			name:    "Basic validation",
			file:    "valid.go",
			level:   ValidationLevelBasic,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.Valid {
					t.Error("Expected valid result")
				}
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			},
		},
		{
			name:    "Standard validation",
			file:    "valid.go",
			level:   ValidationLevelStandard,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.Valid {
					t.Error("Expected valid result")
				}
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			},
		},
		{
			name:    "Strict validation",
			file:    "invalid.go",
			level:   ValidationLevelStrict,
			wantErr: true,
			check: func(t *testing.T, result *ValidationResult) {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}
			},
		},
		{
			name:    "Non-existent file",
			file:    "nonexistent.go",
			level:   ValidationLevelBasic,
			wantErr: true,
			check: func(t *testing.T, result *ValidationResult) {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateFile(context.Background(), tt.file, tt.level)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ValidateFile() unexpected error = %v", err)
				}
				return
			}

			if tt.wantErr {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestValidatePackage(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建测试包结构
	pkgFiles := map[string]string{
		"valid/main.go": `package valid
func Valid() {}`,
		"valid/types.go": `package valid
type ValidType struct{}`,
		"invalid/main.go": `package invalid
func Invalid() {
	var x string = 123 // Type error
}`,
		"invalid/types.go": `package invalid
type InvalidType struct {
	Field int = "string" // Type error
}`,
		"empty/empty.go": "",
	}

	for name, content := range pkgFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	validator := NewValidator(tmpDir)

	tests := []struct {
		name    string
		pkg     string
		level   ValidationLevel
		wantErr bool
		check   func(t *testing.T, result *ValidationResult)
	}{
		{
			name:    "Basic validation",
			pkg:     "valid",
			level:   ValidationLevelBasic,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.Valid {
					t.Error("Expected valid result")
				}
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			},
		},
		{
			name:    "Standard validation",
			pkg:     "valid",
			level:   ValidationLevelStandard,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.Valid {
					t.Error("Expected valid result")
				}
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			},
		},
		{
			name:    "Strict validation",
			pkg:     "invalid",
			level:   ValidationLevelStrict,
			wantErr: true,
			check: func(t *testing.T, result *ValidationResult) {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}
			},
		},
		{
			name:    "Empty package check",
			pkg:     "empty",
			level:   ValidationLevelBasic,
			wantErr: true,
			check: func(t *testing.T, result *ValidationResult) {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}
			},
		},
		{
			name:    "Non-existent package",
			pkg:     "nonexistent",
			level:   ValidationLevelBasic,
			wantErr: true,
			check: func(t *testing.T, result *ValidationResult) {
				if result != nil && result.Valid {
					t.Error("Expected invalid result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidatePackage(context.Background(), tt.pkg, tt.level)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ValidatePackage() unexpected error = %v", err)
				}
				return
			}

			if tt.wantErr {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestValidateProject(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建测试项目结构
	projectFiles := map[string]string{
		"go.mod": `module testproject
go 1.22.0`,
		"pkg1/main.go": `package pkg1

import "fmt"

func Valid() {
	fmt.Println("Hello")
}`,
		"pkg1/types.go": `package pkg1

type ValidType struct {
	Name string
}`,
		"pkg2/invalid.go": `package pkg2

func Invalid() {
	var x string = 123 // Type error
}`,
		"pkg2/types.go": `package pkg2

type InvalidType struct {
	Field int = "string" // Type error
}`,
		"pkg3/empty.go": `package pkg3`,
	}

	for name, content := range projectFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	validator := NewValidator(tmpDir)

	tests := []struct {
		name    string
		level   ValidationLevel
		wantErr bool
		check   func(t *testing.T, result *ValidationResult)
	}{
		{
			name:    "Basic validation",
			level:   ValidationLevelBasic,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.Valid {
					t.Error("Expected valid result")
				}
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			},
		},
		{
			name:    "Standard validation",
			level:   ValidationLevelStandard,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.Valid {
					t.Error("Expected valid result")
				}
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			},
		},
		{
			name:    "Strict validation",
			level:   ValidationLevelStrict,
			wantErr: true,
			check: func(t *testing.T, result *ValidationResult) {
				if result.Valid {
					t.Error("Expected invalid result")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected validation errors")
				}

				// 检查是否检测到类型错误
				var hasTypeError bool
				for _, err := range result.Errors {
					if err.Code == "TYPE_ERROR" {
						hasTypeError = true
						break
					}
				}
				if !hasTypeError {
					t.Error("Expected to find type error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateProject(context.Background(), tt.level)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ValidateProject() unexpected error = %v", err)
				}
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestValidateDependencies(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建依赖测试结构
	files := map[string]string{
		"go.mod": `module testproject
go 1.22.0
require (
	golang.org/x/tools v0.28.0
)`,
		"pkg/main.go": `package pkg
import (
	"golang.org/x/tools/go/packages"
)
func LoadPackage(path string) (*packages.Package, error) {
	cfg := &packages.Config{Mode: packages.NeedTypes}
	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		return nil, err
	}
	return pkgs[0], nil
}`,
		"internal/util.go": `package internal
func Helper() {}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	validator := NewValidator(tmpDir)

	tests := []struct {
		name    string
		pkg     string
		wantErr bool
	}{
		{
			name:    "Valid external dependency",
			pkg:     "pkg",
			wantErr: false,
		},
		{
			name:    "Internal package",
			pkg:     "internal",
			wantErr: false,
		},
		{
			name:    "Non-existent package",
			pkg:     "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateDependencies(context.Background(), tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result for successful validation")
			}
		})
	}
}

// 测试第三方包依赖验证
func TestValidateExternalDependencies(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建项目目录
	projectDir := filepath.Join(tmpDir, "testproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建一个包含外部依赖的项目结构
	files := map[string]string{
		"go.mod": `module testproject
go 1.22.0
require (
	golang.org/x/tools v0.28.0
	golang.org/x/sync v0.10.0
)`,
		"pkg/analyzer.go": `package pkg

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
)

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
}`,
		"pkg/sync.go": `package pkg

import (
	"context"
	"golang.org/x/sync/errgroup"
)

func Process(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return nil
	})
	return g.Wait()
}`,
	}

	for name, content := range files {
		path := filepath.Join(projectDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 初始化依赖
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to initialize dependencies: %v", err)
	}

	validator := NewValidator(projectDir)

	tests := []struct {
		name    string
		level   ValidationLevel
		wantErr bool
		check   func(t *testing.T, result *ValidationResult)
	}{
		{
			name:    "Basic dependency validation",
			level:   ValidationLevelBasic,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.HasExternalDeps {
					t.Error("Expected external dependencies to be detected")
				}
				if len(result.ExternalDeps) == 0 {
					t.Error("Expected non-empty external dependencies list")
				}
			},
		},
		{
			name:    "Strict dependency validation",
			level:   ValidationLevelStrict,
			wantErr: false,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.HasExternalDeps {
					t.Error("Expected external dependencies to be detected")
				}

				// 检查是否检测到所有外部依赖
				deps := []string{"golang.org/x/tools", "golang.org/x/sync"}
				for _, dep := range deps {
					found := false
					for _, extDep := range result.ExternalDeps {
						if strings.Contains(extDep, dep) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find dependency %s", dep)
					}
				}

				// 检查导入分析
				imports := []string{
					"golang.org/x/tools/go/analysis",
					"golang.org/x/sync/errgroup",
				}
				for _, imp := range imports {
					found := false
					for _, resultImp := range result.Imports {
						if strings.Contains(resultImp, imp) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find import %s", imp)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateProject(context.Background(), tt.level)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ValidateProject() unexpected error = %v", err)
				}
				return
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// 测试循环依赖检测
func TestCheckCircularDependencies(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建项目目录
	projectDir := filepath.Join(tmpDir, "testproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 创建一个包含循环依赖的项目结构
	files := map[string]string{
		"go.mod": `module testproject
go 1.22.0`,
		"pkg1/pkg1.go": `package pkg1

import "testproject/pkg2"

type Type1 struct{}

func (t *Type1) Method1() {
	pkg2.NewType2().Method2()
}

func NewType1() *Type1 {
	return &Type1{}
}`,
		"pkg2/pkg2.go": `package pkg2

import "testproject/pkg3"

type Type2 struct{}

func (t *Type2) Method2() {
	pkg3.NewType3().Method3()
}

func NewType2() *Type2 {
	return &Type2{}
}`,
		"pkg3/pkg3.go": `package pkg3

import "testproject/pkg1"

type Type3 struct{}

func (t *Type3) Method3() {
	pkg1.NewType1().Method1()
}

func NewType3() *Type3 {
	return &Type3{}
}`,
	}

	for name, content := range files {
		path := filepath.Join(projectDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	validator := NewValidator(projectDir)

	tests := []struct {
		name      string
		pkgPath   string
		wantCycle bool
		check     func(t *testing.T, result *ValidationResult)
	}{
		{
			name:      "Detect cycle starting from pkg1",
			pkgPath:   "pkg1",
			wantCycle: true,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.HasCircularDeps {
					t.Error("Expected to detect circular dependencies")
				}
				if len(result.CircularDeps) == 0 {
					t.Error("Expected non-empty circular dependencies list")
				}
				// 验证循环依赖链
				cycle := []string{"pkg1", "pkg2", "pkg3", "pkg1"}
				for i, pkg := range cycle[:len(cycle)-1] {
					found := false
					for _, dep := range result.CircularDeps {
						if strings.Contains(dep, pkg) && strings.Contains(dep, cycle[i+1]) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find dependency from %s to %s", pkg, cycle[i+1])
					}
				}
			},
		},
		{
			name:      "Detect cycle starting from pkg2",
			pkgPath:   "pkg2",
			wantCycle: true,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.HasCircularDeps {
					t.Error("Expected to detect circular dependencies")
				}
				if len(result.CircularDeps) == 0 {
					t.Error("Expected non-empty circular dependencies list")
				}
				// 验证循环依赖链
				cycle := []string{"pkg2", "pkg3", "pkg1", "pkg2"}
				for i, pkg := range cycle[:len(cycle)-1] {
					found := false
					for _, dep := range result.CircularDeps {
						if strings.Contains(dep, pkg) && strings.Contains(dep, cycle[i+1]) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find dependency from %s to %s", pkg, cycle[i+1])
					}
				}
			},
		},
		{
			name:      "Detect cycle starting from pkg3",
			pkgPath:   "pkg3",
			wantCycle: true,
			check: func(t *testing.T, result *ValidationResult) {
				if !result.HasCircularDeps {
					t.Error("Expected to detect circular dependencies")
				}
				if len(result.CircularDeps) == 0 {
					t.Error("Expected non-empty circular dependencies list")
				}
				// 验证循环依赖链
				cycle := []string{"pkg3", "pkg1", "pkg2", "pkg3"}
				for i, pkg := range cycle[:len(cycle)-1] {
					found := false
					for _, dep := range result.CircularDeps {
						if strings.Contains(dep, pkg) && strings.Contains(dep, cycle[i+1]) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find dependency from %s to %s", pkg, cycle[i+1])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.CheckCircularDependencies(context.Background(), tt.pkgPath)
			if err != nil {
				t.Fatalf("CheckCircularDependencies() error = %v", err)
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// 测试并发验证
func TestConcurrentValidation(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建多个包进行并发验证
	for i := 0; i < 5; i++ {
		pkgName := fmt.Sprintf("pkg%d", i)
		pkgPath := filepath.Join(tmpDir, pkgName)
		if err := os.MkdirAll(pkgPath, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		content := fmt.Sprintf(`package %s
func Func%d() {}`, pkgName, i)

		if err := os.WriteFile(filepath.Join(pkgPath, "main.go"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	validator := NewValidator(tmpDir)
	var wg sync.WaitGroup
	errs := make(chan error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pkgPath := fmt.Sprintf("pkg%d", i)
			_, err := validator.ValidatePackage(context.Background(), pkgPath, ValidationLevelStrict)
			if err != nil {
				errs <- fmt.Errorf("failed to validate package %s: %v", pkgPath, err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}
