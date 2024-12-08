package readgo

import (
	"context"
	"os"
	"strings"
	"testing"
)

// setupTestEnvironment creates a temporary test environment
// DEPRECATED: This function is no longer used and will be removed
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
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name     string
		opts     TreeOptions
		wantType FileType
	}{
		{
			name: "All files",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantType: FileTypeAll,
		},
		{
			name: "Only Go files",
			opts: TreeOptions{
				FileTypes: FileTypeGo,
			},
			wantType: FileTypeGo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewDefaultReader().WithWorkDir(tmpDir)
			tree, err := reader.GetFileTree(context.Background(), ".", tt.opts)
			if err != nil {
				t.Fatalf("GetFileTree() error = %v", err)
			}

			// Verify tree structure
			if tree == nil {
				t.Fatal("GetFileTree() returned nil tree")
			}
			if tree.Type != "directory" {
				t.Errorf("Root node type = %v, want directory", tree.Type)
			}

			// Check file types based on options
			var checkFiles func(*FileTreeNode)
			checkFiles = func(node *FileTreeNode) {
				for _, child := range node.Children {
					if child.Type == "file" {
						switch tt.wantType {
						case FileTypeGo:
							if !strings.HasSuffix(child.Name, ".go") {
								t.Errorf("Found non-Go file: %s", child.Path)
							}
						}
					} else {
						checkFiles(child)
					}
				}
			}
			checkFiles(tree)
		})
	}
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name     string
		path     string
		opts     ReadOptions
		wantErr  bool
		wantText string
	}{
		{
			name: "Valid file",
			path: "testdata/basic/main.go",
			opts: ReadOptions{
				IncludeComments: true,
				StripSpaces:     false,
			},
			wantErr:  false,
			wantText: "package basic",
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
			reader := NewDefaultReader().WithWorkDir(tmpDir)
			content, err := reader.ReadSourceFile(context.Background(), tt.path, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadSourceFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(string(content), tt.wantText) {
				t.Errorf("ReadSourceFile() content = %v, want %v", string(content), tt.wantText)
			}
		})
	}
}
