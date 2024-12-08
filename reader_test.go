package readgo2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestGetFileTree(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建测试目录结构
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

	// 创建目录结构
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}

	// 创建测试文件
	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	reader := NewSourceReader(tmpDir)

	tests := []struct {
		name     string
		opts     TreeOptions
		wantLen  int // 期望的文件数量
		validate func(t *testing.T, files []*FileTreeNode)
	}{
		{
			name: "All files",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantLen: 5, // 所有文件
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
			wantLen: 3, // 只有.go文件
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

			// 收集所有文件
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

			// 过滤掉隐藏文件和临时文件
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

func TestReadSourceFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	reader := NewSourceReader(tmpDir)

	// 创建测试文件
	validContent := `package test
type User struct {
	Name string
}`
	validFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(validFile, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建空文件
	emptyFile := filepath.Join(tmpDir, "empty.go")
	if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		file     string
		content  string
		opts     ReadOptions
		wantErr  bool
		validate func(t *testing.T, content []byte)
	}{
		{
			name:    "Valid Go file",
			file:    "test.go",
			content: validContent,
			opts: ReadOptions{
				IncludeComments: true,
			},
			wantErr: false,
			validate: func(t *testing.T, content []byte) {
				if !strings.Contains(string(content), "type User struct") {
					t.Error("Expected file content to contain User struct")
				}
			},
		},
		{
			name:    "Non-existent file",
			file:    "nonexistent.go",
			content: "",
			opts: ReadOptions{
				IncludeComments: false,
			},
			wantErr: true,
			validate: func(t *testing.T, content []byte) {
				if len(content) > 0 {
					t.Error("Expected empty content for non-existent file")
				}
			},
		},
		{
			name:    "Empty file",
			file:    "empty.go",
			content: "",
			opts: ReadOptions{
				StripSpaces: true,
			},
			wantErr: false,
			validate: func(t *testing.T, content []byte) {
				if len(content) != 0 {
					t.Error("Expected empty file content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := reader.ReadSourceFile(context.Background(), tt.file, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadSourceFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, content)
			}
		})
	}
}

func TestGetPackageFiles(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建包结构
	pkgPath := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		t.Fatalf("Failed to create test package: %v", err)
	}

	// 创建测试文件
	files := map[string]string{
		"main.go":      mockFileContent,
		"types.go":     "package testpkg\n\ntype Config struct{}\n",
		"main_test.go": "package testpkg_test\n\nfunc TestMain(t *testing.T) {}\n",
		"README.md":    "# Test Package",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(pkgPath, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	reader := NewSourceReader(tmpDir)

	tests := []struct {
		name    string
		opts    TreeOptions
		wantLen int
	}{
		{
			name: "All Go files",
			opts: TreeOptions{
				FileTypes: FileTypeGo,
			},
			wantLen: 3,
		},
		{
			name: "No test files",
			opts: TreeOptions{
				FileTypes:       FileTypeGo,
				ExcludePatterns: []string{"*_test.go"},
			},
			wantLen: 2,
		},
		{
			name: "All files",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := reader.GetPackageFiles(context.Background(), "testpkg", tt.opts)
			if err != nil {
				t.Fatalf("GetPackageFiles failed: %v", err)
			}

			if len(files) != tt.wantLen {
				t.Errorf("GetPackageFiles() got %d files, want %d", len(files), tt.wantLen)
			}
		})
	}
}

func TestSearchFiles(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建测试文件
	files := map[string]string{
		"pkg1/user.go":      "package pkg1\ntype User struct{}\n",
		"pkg2/customer.go":  "package pkg2\ntype Customer struct{}\n",
		"pkg1/user_test.go": "package pkg1_test\nfunc TestUser(t *testing.T) {}\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	reader := NewSourceReader(tmpDir)

	tests := []struct {
		name      string
		pattern   string
		opts      TreeOptions
		wantFiles int
		wantErr   bool
		validate  func(t *testing.T, files []*FileTreeNode)
	}{
		{
			name:    "Find user files",
			pattern: "user",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantFiles: 2,
			validate: func(t *testing.T, files []*FileTreeNode) {
				for _, file := range files {
					if !strings.Contains(strings.ToLower(file.Name), "user") {
						t.Errorf("File %s does not match pattern 'user'", file.Name)
					}
				}
			},
		},
		{
			name:    "Find only source files",
			pattern: "user",
			opts: TreeOptions{
				FileTypes:       FileTypeGo,
				ExcludePatterns: []string{"*_test.go"},
			},
			wantFiles: 1,
			validate: func(t *testing.T, files []*FileTreeNode) {
				if len(files) != 1 || !strings.HasSuffix(files[0].Name, "user.go") {
					t.Error("Expected only user.go file")
				}
			},
		},
		{
			name:    "No matches",
			pattern: "nonexistent",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantFiles: 0,
		},
		{
			name:    "Invalid pattern",
			pattern: "[invalid",
			opts: TreeOptions{
				FileTypes: FileTypeAll,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := reader.SearchFiles(context.Background(), tt.pattern, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 过滤掉隐藏文件和临时文件
				var filteredFiles []*FileTreeNode
				for _, file := range files {
					if !strings.HasPrefix(file.Name, ".") && !strings.HasSuffix(file.Name, "~") {
						filteredFiles = append(filteredFiles, file)
					}
				}

				if len(filteredFiles) != tt.wantFiles {
					t.Errorf("SearchFiles() got %d files, want %d", len(filteredFiles), tt.wantFiles)
					for _, file := range filteredFiles {
						t.Logf("Found file: %s", file.Name)
					}
				}

				if tt.validate != nil {
					tt.validate(t, filteredFiles)
				}
			}
		})
	}
}

// 添加第三方包依赖测试
func TestReadSourceFileWithExternalDeps(t *testing.T) {
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

	testFile := filepath.Join(tmpDir, "external_deps.go")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	reader := NewSourceReader(tmpDir)

	tests := []struct {
		name     string
		opts     ReadOptions
		validate func(t *testing.T, content []byte)
	}{
		{
			name: "Check external imports",
			opts: ReadOptions{
				IncludeComments: true,
				StripSpaces:     false,
			},
			validate: func(t *testing.T, content []byte) {
				if !strings.Contains(string(content), "golang.org/x/tools/go/analysis") {
					t.Error("Expected file to contain external package import")
				}
				if !strings.Contains(string(content), "analysis.Analyzer") {
					t.Error("Expected file to contain usage of external package")
				}
			},
		},
		{
			name: "Check with stripped spaces",
			opts: ReadOptions{
				StripSpaces: true,
			},
			validate: func(t *testing.T, content []byte) {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, " ") || strings.HasSuffix(line, " ") {
						t.Error("Expected no leading/trailing spaces after stripping")
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := reader.ReadSourceFile(context.Background(), "external_deps.go", tt.opts)
			if err != nil {
				t.Fatalf("ReadSourceFile() error = %v", err)
			}
			tt.validate(t, content)
		})
	}
}

// 测试大文件读取
func TestReadLargeSourceFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	reader := NewSourceReader(tmpDir)

	// 创建一个大文件
	var largeContent strings.Builder
	largeContent.WriteString("package large\n\n")
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&largeContent, "func Function%d() int { return %d }\n", i, i)
	}

	largeFile := filepath.Join(tmpDir, "large.go")
	if err := os.WriteFile(largeFile, []byte(largeContent.String()), 0644); err != nil {
		t.Fatal(err)
	}

	// 记录初始内存使用
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	content, err := reader.ReadSourceFile(context.Background(), "large.go", ReadOptions{})
	if err != nil {
		t.Fatalf("ReadSourceFile() error = %v", err)
	}

	// 记录最终内存使用
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// 验证文件内容
	lines := strings.Split(string(content), "\n")
	if len(lines) < 1000 {
		t.Errorf("Expected at least 1000 lines, got %d", len(lines))
	}

	// 验证内存使用增长
	memGrowth := m2.Alloc - m1.Alloc
	if memGrowth > 5*1024*1024 { // 5MB
		t.Errorf("Memory growth too high: %d bytes", memGrowth)
	}

	// 验证每个函数
	for i := 0; i < 1000; i++ {
		expectedFunc := fmt.Sprintf("func Function%d() int { return %d }", i, i)
		found := false
		for _, line := range lines {
			if strings.TrimSpace(line) == expectedFunc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Function%d not found in content", i)
			break // 只报告第一个错误
		}
	}
}

// 测试并发读取
func TestConcurrentFileReading(t *testing.T) {
	tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	reader := NewSourceReader(tmpDir)

	// 创建多个测试文件
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf(`package pkg%d
func Func%d() {}`, i, i)

		filename := fmt.Sprintf("file%d.go", i)
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, 10)

	// 并发读取文件
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			filename := fmt.Sprintf("file%d.go", i)
			_, err := reader.ReadSourceFile(context.Background(), filename, ReadOptions{})
			if err != nil {
				errs <- fmt.Errorf("failed to read file%d: %v", i, err)
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errs)

	// 检查错误
	for err := range errs {
		t.Error(err)
	}
}
