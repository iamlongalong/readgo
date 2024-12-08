package readgo

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"valid.go": `package test

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
		"invalid.go": `package test

import "fmt"

func main() {
	fmt.Println("Hello, World!"
}`, // Missing closing parenthesis
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	validator := NewValidator(tmpDir)

	tests := []struct {
		name          string
		file          string
		wantErr       bool
		wantValidErrs bool // true if we expect validation errors in the result
	}{
		{
			name:          "Valid file",
			file:          "valid.go",
			wantErr:       false,
			wantValidErrs: false,
		},
		{
			name:          "Invalid file",
			file:          "invalid.go",
			wantErr:       false,
			wantValidErrs: true,
		},
		{
			name:    "Non-existent file",
			file:    "nonexistent.go",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateFile(context.Background(), tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.wantValidErrs && len(result.Errors) == 0 {
					t.Error("ValidateFile() expected validation errors but got none")
				}
				if !tt.wantValidErrs && len(result.Errors) > 0 {
					t.Errorf("ValidateFile() got unexpected errors = %v", result.Errors)
				}
			}
		})
	}
}

func TestValidatePackage(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test package structure
	pkgPath := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create test package: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"main.go": `package testpkg

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
		"types.go": `package testpkg

type User struct {
	Name string
}`,
	}

	for name, content := range testFiles {
		path := filepath.Join(pkgPath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	validator := NewValidator(tmpDir)

	tests := []struct {
		name    string
		pkg     string
		wantErr bool
	}{
		{
			name:    "Valid package",
			pkg:     "testpkg",
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
			result, err := validator.ValidatePackage(context.Background(), tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result.Errors) > 0 {
				t.Errorf("ValidatePackage() got errors = %v", result.Errors)
			}
		})
	}
}

func TestValidateProject(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test project structure
	dirs := []string{
		"pkg1",
		"pkg2",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		"pkg1/main.go": `package pkg1

import "fmt"

func main() {
	fmt.Println("Hello from pkg1!")
}`,
		"pkg2/types.go": `package pkg2

type User struct {
	Name string
}`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	validator := NewValidator(tmpDir)

	result, err := validator.ValidateProject(context.Background())
	if err != nil {
		t.Errorf("ValidateProject() error = %v", err)
		return
	}

	if len(result.Errors) > 0 {
		t.Errorf("ValidateProject() got errors = %v", result.Errors)
	}
}
