package readgo

import (
	"context"
	"strings"
	"testing"
)

func TestGetFileTree(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		opts    TreeOptions
		wantErr bool
		check   func(*testing.T, *FileTreeNode)
	}{
		{
			name: "Only Go files",
			root: ".",
			opts: TreeOptions{
				IncludePatterns: []string{"*.go"},
			},
			wantErr: false,
			check: func(t *testing.T, tree *FileTreeNode) {
				var files []string
				var collect func(*FileTreeNode)
				collect = func(node *FileTreeNode) {
					if node.Type == "file" {
						files = append(files, node.Name)
					}
					for _, child := range node.Children {
						collect(child)
					}
				}
				collect(tree)

				for _, file := range files {
					if !strings.HasSuffix(file, ".go") {
						t.Errorf("Found non-Go file: %s", file)
					}
				}
			},
		},
		{
			name:    "Non-existent directory",
			root:    "nonexistent",
			opts:    TreeOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewDefaultReader().WithWorkDir("testdata/basic")
			tree, err := reader.GetFileTree(context.Background(), tt.root, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFileTree() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, tree)
			}
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

func TestReadFileWithFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestFiles(t, tmpDir)

	tests := []struct {
		name          string
		path          string
		wantFuncCount int
		wantErr       bool
	}{
		{
			name:          "Valid Go file",
			path:          "testdata/basic/main.go",
			wantFuncCount: 7, // String() method and 6 other functions
			wantErr:       false,
		},
		{
			name:          "Non-existent file",
			path:          "nonexistent.go",
			wantFuncCount: 0,
			wantErr:       true,
		},
		{
			name:          "Empty path",
			path:          "",
			wantFuncCount: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewDefaultReader().WithWorkDir(tmpDir)
			result, err := reader.ReadFileWithFunctions(context.Background(), tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFileWithFunctions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(result.Functions) != tt.wantFuncCount {
					t.Errorf("ReadFileWithFunctions() got %d functions:", len(result.Functions))
					for _, fn := range result.Functions {
						t.Errorf("  - %s (lines %d-%d)", fn.Name, fn.StartLine, fn.EndLine)
					}
					t.Errorf("want %d functions", tt.wantFuncCount)
				}
				// Verify function positions are in order
				for i := 1; i < len(result.Functions); i++ {
					if result.Functions[i].StartLine <= result.Functions[i-1].StartLine {
						t.Errorf("Functions not properly ordered: %v after %v", result.Functions[i], result.Functions[i-1])
					}
				}
				// Verify each function has valid line numbers
				for _, fn := range result.Functions {
					if fn.StartLine <= 0 || fn.EndLine <= 0 || fn.EndLine < fn.StartLine {
						t.Errorf("Invalid line numbers for function %s: start=%d, end=%d", fn.Name, fn.StartLine, fn.EndLine)
					}
				}
			}
		})
	}
}
