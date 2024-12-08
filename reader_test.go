package readgo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestEnvironment creates a temporary test environment
func setupTestEnvironment(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "readgo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestGetFileTree(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create test directory structure
	dirs := []string{
		"pkg1",
		"pkg1/subpkg",
		"pkg2",
	}

	files := map[string]string{
		"pkg1/file1.go":         "package pkg1",
		"pkg1/subpkg/file2.go":  "package subpkg",
		"pkg2/file3.go":         "package pkg2",
		"pkg2/not_go_file.txt":  "some text",
		"pkg1/subpkg/README.md": "# Test Package",
	}

	// Create directories
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}

	// Create test files
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	reader := NewReader(tmpDir)

	tests := []struct {
		name     string
		opts     TreeOptions
		wantLen  int
		validate func(t *testing.T, files []*FileTreeNode)
	}{
		{
			name: "All files",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantLen: 5,
			validate: func(t *testing.T, files []*FileTreeNode) {
				var hasGoFile, hasTextFile, hasMarkdown bool
				for _, file := range files {
					switch {
					case strings.HasSuffix(file.Name, ".go"):
						hasGoFile = true
					case strings.HasSuffix(file.Name, ".txt"):
						hasTextFile = true
					case strings.HasSuffix(file.Name, ".md"):
						hasMarkdown = true
					}
				}
				if !hasGoFile || !hasTextFile || !hasMarkdown {
					t.Error("Missing expected file types")
				}
			},
		},
		{
			name: "Only Go files",
			opts: TreeOptions{
				FileTypes:       FileTypeGo,
				ExcludePatterns: []string{"*_test.go"},
			},
			wantLen: 3,
			validate: func(t *testing.T, files []*FileTreeNode) {
				for _, file := range files {
					if !strings.HasSuffix(file.Name, ".go") {
						t.Errorf("Found non-Go file: %s", file.Name)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := reader.GetFileTree(context.Background(), ".", tt.opts)
			if err != nil {
				t.Fatalf("GetFileTree failed: %v", err)
			}

			// Collect all files
			var allFiles []*FileTreeNode
			var collectFiles func(*FileTreeNode)
			collectFiles = func(node *FileTreeNode) {
				if node.Type == "file" {
					allFiles = append(allFiles, node)
				}
				for _, child := range node.Children {
					collectFiles(child)
				}
			}
			collectFiles(tree)

			// Filter hidden and temporary files
			var filteredFiles []*FileTreeNode
			for _, file := range allFiles {
				if !strings.HasPrefix(file.Name, ".") && !strings.HasSuffix(file.Name, "~") {
					filteredFiles = append(filteredFiles, file)
				}
			}

			if len(filteredFiles) != tt.wantLen {
				t.Errorf("GetFileTree() got %d files, want %d", len(filteredFiles), tt.wantLen)
				for _, file := range filteredFiles {
					t.Logf("Found file: %s", file.Name)
				}
			}

			if tt.validate != nil {
				tt.validate(t, filteredFiles)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

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

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid file",
			path:    "main.go",
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
			reader := NewReader(tmpDir)
			result, err := reader.ReadFile(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("ReadFile() returned nil result")
			}
		})
	}
}
