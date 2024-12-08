package readgo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTestModule(t *testing.T, dir string) {
	goModContent := `module test

go 1.22.0
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}
}

func TestAnalyzeProject(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	setupTestModule(t, tmpDir)

	// Create test files
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
		"types.go": `package main

type TestInterface interface {
	Test() error
}

type TestStruct struct {
	Field string
}`,
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	// Test cases
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
			name:    "Invalid path",
			path:    "/nonexistent/path",
			wantErr: true,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: false, // Empty path defaults to "."
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(tmpDir)
			result, err := analyzer.AnalyzeProject(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("AnalyzeProject() returned nil result")
			}
		})
	}
}

func TestAnalyzeFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	setupTestModule(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package test

func TestFunc() error {
	return nil
}

type TestType struct {
	Field string
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid file",
			path:    "test.go",
			wantErr: false,
		},
		{
			name:    "Invalid file",
			path:    "nonexistent.go",
			wantErr: true,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(tmpDir)
			result, err := analyzer.AnalyzeFile(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("AnalyzeFile() returned nil result")
			}
		})
	}
}

func TestFindType(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	setupTestModule(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "types.go")
	content := `package test

type TestType struct {
	Field string
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name      string
		pkgPath   string
		typeName  string
		wantErr   bool
		checkType bool
	}{
		{
			name:      "Valid type",
			pkgPath:   ".",
			typeName:  "TestType",
			wantErr:   false,
			checkType: true,
		},
		{
			name:     "Invalid type",
			pkgPath:  ".",
			typeName: "NonexistentType",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(tmpDir)
			result, err := analyzer.FindType(context.Background(), tt.pkgPath, tt.typeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("FindType() returned nil result")
			}
			if tt.checkType && result != nil {
				if result.Name != tt.typeName {
					t.Errorf("FindType() got type name = %v, want %v", result.Name, tt.typeName)
				}
			}
		})
	}
}

func TestFindInterface(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod file
	setupTestModule(t, tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "interfaces.go")
	content := `package test

type TestInterface interface {
	Test() error
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name          string
		pkgPath       string
		interfaceName string
		wantErr       bool
		checkType     bool
	}{
		{
			name:          "Valid interface",
			pkgPath:       ".",
			interfaceName: "TestInterface",
			wantErr:       false,
			checkType:     true,
		},
		{
			name:          "Invalid interface",
			pkgPath:       ".",
			interfaceName: "NonexistentInterface",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer(tmpDir)
			result, err := analyzer.FindInterface(context.Background(), tt.pkgPath, tt.interfaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("FindInterface() returned nil result")
			}
			if tt.checkType && result != nil {
				if result.Name != tt.interfaceName {
					t.Errorf("FindInterface() got interface name = %v, want %v", result.Name, tt.interfaceName)
				}
			}
		})
	}
}
