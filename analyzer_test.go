package readgo

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAnalyzeProject(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid project",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "Non-existent project",
			path:    "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(WithWorkDir("testdata/basic"))
			result, err := analyzer.AnalyzeProject(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != nil {
				if result.Name != "main" {
					t.Errorf("Expected project name to be 'main', got %q", result.Name)
				}
				if len(result.Types) == 0 {
					t.Error("Expected types to be found")
				}
				if len(result.Functions) == 0 {
					t.Error("Expected functions to be found")
				}
			}
		})
	}
}

func TestAnalyzeProjectWithDirectories(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir := t.TempDir()

	// 创建嵌套的目录结构
	dirs := []string{
		filepath.Join(tmpDir, "src"),
		filepath.Join(tmpDir, "src", "models"),
		filepath.Join(tmpDir, "src", "handlers"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		assertNoError(t, err)
		assertDirExists(t, dir)
	}

	// 在目录中创建一些 Go 文件，确保包名和内容格式正确
	files := map[string]string{
		filepath.Join(tmpDir, "src", "models", "user.go"): `// Package models contains data models
package models

// User represents a user in the system
type User struct {
	ID   int
	Name string
}`,
		filepath.Join(tmpDir, "src", "handlers", "handler.go"): `// Package handlers contains HTTP handlers
package handlers

// Handler represents an HTTP handler
type Handler struct {
	Path string
}`,
		filepath.Join(tmpDir, "go.mod"): `module testproject

go 1.16
`,
	}

	for path, content := range files {
		err := os.WriteFile(path, []byte(content), 0644)
		assertNoError(t, err)
		assertFileExists(t, path)
	}

	// 使用 Analyzer 分析项目，使用正确的相对路径
	analyzer := NewAnalyzer(WithWorkDir(tmpDir))
	result, err := analyzer.AnalyzeProject(context.Background(), "src")
	assertNoError(t, err)

	if result == nil {
		t.Fatal("Expected analysis result, got nil")
	}

	// 验证是否找到了所有类型
	foundTypes := make(map[string]bool)
	for _, typ := range result.Types {
		foundTypes[typ.Name] = true
	}

	expectedTypes := []string{"User", "Handler"}
	for _, typeName := range expectedTypes {
		if !foundTypes[typeName] {
			t.Errorf("Expected to find type %s in analysis results", typeName)
		}
	}

	// 验证包名，添加调试信息
	packages := make(map[string]bool)
	for _, typ := range result.Types {
		t.Logf("Found type %s in package %s", typ.Name, typ.Package)
		// 从完整包路径中提取最后一个部分作为包名
		pkgParts := strings.Split(typ.Package, "/")
		pkgName := pkgParts[len(pkgParts)-1]
		packages[pkgName] = true
	}

	expectedPackages := []string{"models", "handlers"}
	for _, pkgName := range expectedPackages {
		if !packages[pkgName] {
			t.Errorf("Expected to find package %s in analysis results", pkgName)
		}
	}
}

func TestAnalyzeFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	// 创建一个超大文件
	largePath := filepath.Join(tmpDir, "large.go")
	largeContent := make([]byte, maxFileSize+1)
	err := os.WriteFile(largePath, largeContent, 0600)
	assertNoError(t, err)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid file",
			filePath: filepath.Join("testdata", "basic", "main.go"),
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			filePath: "nonexistent.go",
			wantErr:  true,
		},
		{
			name:     "File too large",
			filePath: largePath,
			wantErr:  true,
		},
		{
			name:     "Invalid extension",
			filePath: "test.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.AnalyzeFile(context.Background(), tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				assertFileExists(t, filepath.Join(tmpDir, tt.filePath))

				// Check if we found the expected types
				var foundUser, foundInterface bool
				for _, typ := range result.Types {
					switch typ.Name {
					case "User":
						foundUser = true
						if typ.Type != "struct{ID int; Name string}" {
							t.Errorf("User type has wrong structure: %s", typ.Type)
						}
					case "ComplexInterface":
						foundInterface = true
						if !strings.Contains(typ.Type, "interface") {
							t.Errorf("ComplexInterface is not an interface type: %s", typ.Type)
						}
					}
				}

				if !foundUser {
					t.Error("User type not found in analysis results")
				}
				if !foundInterface {
					t.Error("ComplexInterface not found in analysis results")
				}
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestFindType(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name     string
		pkgPath  string
		typeName string
		wantErr  bool
	}{
		{
			name:     "Valid type",
			pkgPath:  "./testdata/basic",
			typeName: "User",
			wantErr:  false,
		},
		{
			name:     "Non-existent type",
			pkgPath:  "./testdata/basic",
			typeName: "NonExistentType",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.FindType(context.Background(), tt.pkgPath, tt.typeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				if result.Name != tt.typeName {
					t.Errorf("FindType() got type name %q, want %q", result.Name, tt.typeName)
				}
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestFindInterface(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name          string
		pkgPath       string
		interfaceName string
		wantErr       bool
	}{
		{
			name:          "Valid interface",
			pkgPath:       "./testdata/basic",
			interfaceName: "ComplexInterface",
			wantErr:       false,
		},
		{
			name:          "Non-existent interface",
			pkgPath:       "./testdata/basic",
			interfaceName: "NonExistentInterface",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(
				WithWorkDir(tmpDir),
				WithCacheTTL(time.Minute),
			)
			result, err := analyzer.FindInterface(context.Background(), tt.pkgPath, tt.interfaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assertNoError(t, err)
				if result.Name != tt.interfaceName {
					t.Errorf("FindInterface() got interface name %q, want %q", result.Name, tt.interfaceName)
				}
			} else {
				assertError(t, err)
			}
		})
	}
}

func TestCacheEffectiveness(t *testing.T) {
	analyzer := NewAnalyzer(
		WithWorkDir("testdata/basic"),
		WithCacheTTL(time.Minute),
	)

	// First call
	start := time.Now()
	result1, err := analyzer.FindType(context.Background(), ".", "TestType")
	if err != nil {
		t.Fatalf("First FindType() failed: %v", err)
	}
	firstDuration := time.Since(start)

	// Second call (should be cached)
	start = time.Now()
	result2, err := analyzer.FindType(context.Background(), ".", "TestType")
	if err != nil {
		t.Fatalf("Second FindType() failed: %v", err)
	}
	secondDuration := time.Since(start)

	// Verify results are the same
	if !reflect.DeepEqual(result1, result2) {
		t.Error("Cache returned different results")
	}

	// Check cache stats
	stats := analyzer.GetCacheStats()
	if hits, ok := stats["hits"].(int64); !ok || hits == 0 {
		t.Error("Expected cache hits > 0")
	}

	// The second call should be significantly faster
	if secondDuration > firstDuration/2 {
		t.Logf("First call: %v", firstDuration)
		t.Logf("Second call: %v", secondDuration)
		t.Skip("Cache performance test skipped - results may vary on different machines")
	}
}

// Helper functions for assertions
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("Directory %s does not exist: %v", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("Path %s exists but is not a directory", path)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("File %s does not exist: %v", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("Path %s exists but is a directory", path)
	}
}

func TestIsGeneratedFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "Generated file",
			content:  "// Code generated by protoc-gen-go. DO NOT EDIT.\npackage main",
			expected: true,
		},
		{
			name:     "Normal file",
			content:  "package main\n\nfunc main() {}",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeneratedFile([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("isGeneratedFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsAllowedExtension(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		expected bool
	}{
		{
			name:     "Go file",
			ext:      ".go",
			expected: true,
		},
		{
			name:     "Text file",
			ext:      ".txt",
			expected: false,
		},
		{
			name:     "No extension",
			ext:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllowedExtension(tt.ext)
			if result != tt.expected {
				t.Errorf("isAllowedExtension() = %v, want %v", result, tt.expected)
			}
		})
	}
}
