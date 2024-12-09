package readgo

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// DefaultReader implements the SourceReader interface
type DefaultReader struct {
	workDir string
}

// NewDefaultReader creates a new DefaultReader instance
func NewDefaultReader() *DefaultReader {
	return &DefaultReader{
		workDir: ".",
	}
}

// WithWorkDir sets the working directory for the reader
func (r *DefaultReader) WithWorkDir(dir string) *DefaultReader {
	r.workDir = dir
	return r
}

// validatePath checks if the path is safe to access
func (r *DefaultReader) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Convert to absolute path
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(r.workDir, path)
	}

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Check if the path is within workDir
	workDirAbs, err := filepath.Abs(r.workDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if !strings.HasPrefix(absPath, workDirAbs) {
		return fmt.Errorf("path is outside of working directory")
	}

	return nil
}

// safeReadFile reads a file with security checks
func (r *DefaultReader) safeReadFile(path string) ([]byte, error) {
	if err := r.validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Get absolute path
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(r.workDir, path)
	}

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Verify file exists and get info
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}

	// Check file size
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large: %s", path)
	}

	// Check file extension for allowed types
	ext := strings.ToLower(filepath.Ext(path))
	if !isAllowedExtension(ext) {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Read file with limited size
	return os.ReadFile(absPath)
}

// GetFileTree returns the file tree starting from the given root
func (r *DefaultReader) GetFileTree(ctx context.Context, root string, opts TreeOptions) (*FileTreeNode, error) {
	if err := r.validatePath(root); err != nil {
		return nil, fmt.Errorf("invalid root path: %w", err)
	}

	if root == "" {
		root = "."
	}

	absRoot := filepath.Join(r.workDir, root)
	absRoot, err := filepath.Abs(absRoot)
	if err != nil {
		return nil, err
	}

	tree := &FileTreeNode{
		Name: filepath.Base(absRoot),
		Path: root,
		Type: "directory",
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if path matches exclude patterns
		for _, pattern := range opts.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, info.Name()); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip if path doesn't match include patterns
		if len(opts.IncludePatterns) > 0 {
			matched := false
			for _, pattern := range opts.IncludePatterns {
				if m, _ := filepath.Match(pattern, info.Name()); m {
					matched = true
					break
				}
			}
			if !matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip if file type doesn't match
		if !info.IsDir() && opts.FileTypes != FileTypeAll {
			switch opts.FileTypes {
			case FileTypeGo:
				if !strings.HasSuffix(info.Name(), ".go") {
					return nil
				}
			case FileTypeTest:
				if !strings.HasSuffix(info.Name(), "_test.go") {
					return nil
				}
			case FileTypeGenerated:
				content, err := r.safeReadFile(path)
				if err != nil {
					return err
				}
				if !isGeneratedFile(content) {
					return nil
				}
			}
		}

		// Convert absolute path to relative path
		relPath, err := filepath.Rel(r.workDir, path)
		if err != nil {
			return err
		}

		node := &FileTreeNode{
			Name:    info.Name(),
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		if info.IsDir() {
			node.Type = "directory"
		} else {
			node.Type = "file"
		}

		// Find parent node
		if path != absRoot {
			parentPath := filepath.Dir(relPath)
			parentNode := findNode(tree, parentPath)
			if parentNode != nil {
				parentNode.Children = append(parentNode.Children, node)
				// Sort children by name
				sort.Slice(parentNode.Children, func(i, j int) bool {
					return parentNode.Children[i].Name < parentNode.Children[j].Name
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tree, nil
}

// ReadFile reads a source file
func (r *DefaultReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	absPath := filepath.Join(r.workDir, filePath)
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, err
	}

	if info.Size() > 1*1024*1024 { // 1MB
		return nil, fmt.Errorf("file too large: %s", filePath)
	}

	if !strings.HasPrefix(absPath, r.workDir) {
		return nil, fmt.Errorf("path is outside of working directory: %s", absPath)
	}

	file, err := os.OpenFile(absPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if !utf8.Valid(content) {
		return nil, fmt.Errorf("file contains invalid UTF-8: %s", filePath)
	}

	return content, nil
}

// GetPackageFiles returns all files in a package
func (r *DefaultReader) GetPackageFiles(ctx context.Context, pkgPath string, opts TreeOptions) ([]*FileTreeNode, error) {
	tree, err := r.GetFileTree(ctx, pkgPath, opts)
	if err != nil {
		return nil, err
	}

	var files []*FileTreeNode
	var collect func(*FileTreeNode)
	collect = func(node *FileTreeNode) {
		if node.Type == "file" {
			files = append(files, node)
		}
		for _, child := range node.Children {
			collect(child)
		}
	}
	collect(tree)

	return files, nil
}

// SearchFiles searches for files matching the given pattern
func (r *DefaultReader) SearchFiles(ctx context.Context, pattern string, opts TreeOptions) ([]*FileTreeNode, error) {
	if pattern == "" {
		return nil, ErrInvalidInput
	}

	tree, err := r.GetFileTree(ctx, ".", opts)
	if err != nil {
		return nil, err
	}

	var matches []*FileTreeNode
	var search func(*FileTreeNode)
	search = func(node *FileTreeNode) {
		if node.Type == "file" && strings.Contains(node.Name, pattern) {
			matches = append(matches, node)
		}
		for _, child := range node.Children {
			search(child)
		}
	}
	search(tree)

	return matches, nil
}

// ReadSourceFile reads a source file with the specified options
func (r *DefaultReader) ReadSourceFile(ctx context.Context, path string, opts ReadOptions) ([]byte, error) {
	content, err := r.safeReadFile(path)
	if err != nil {
		return nil, err
	}

	// Process content based on options
	if opts.StripSpaces {
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}
		content = []byte(strings.Join(lines, "\n"))
	}

	// Comment stripping is not implemented yet
	// if !opts.IncludeComments {
	// 	// TODO: Implement comment stripping
	// }

	return content, nil
}

// isGeneratedFile checks if a file is generated based on its content
func isGeneratedFile(content []byte) bool {
	contentStr := string(content)
	markers := []string{
		"Code generated", "DO NOT EDIT",
		"@generated",
		"// Generated by",
		"/* Generated by",
	}

	for _, marker := range markers {
		if strings.Contains(contentStr, marker) {
			return true
		}
	}
	return false
}

// findNode finds a node in the tree by its path
func findNode(root *FileTreeNode, path string) *FileTreeNode {
	if root.Path == path {
		return root
	}

	for _, child := range root.Children {
		if child.Type == "directory" {
			if node := findNode(child, path); node != nil {
				return node
			}
		}
	}

	return nil
}

// ReadFileWithFunctions reads a source file and returns its content along with function positions
func (r *DefaultReader) ReadFileWithFunctions(ctx context.Context, path string) (*FileContent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	// Read file content
	content, err := r.safeReadFile(path)
	if err != nil {
		return nil, err
	}

	// Get absolute path for parsing
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(r.workDir, path)
	}

	// Parse the file to get function positions
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	// Extract function positions
	var functions []FunctionPosition
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			pos := fset.Position(fn.Pos())
			end := fset.Position(fn.End())
			functions = append(functions, FunctionPosition{
				Name:      fn.Name.Name,
				StartLine: pos.Line,
				EndLine:   end.Line,
			})
		}
		return true
	})

	// Sort functions by start line
	sort.Slice(functions, func(i, j int) bool {
		return functions[i].StartLine < functions[j].StartLine
	})

	return &FileContent{
		Content:   content,
		Functions: functions,
	}, nil
}
