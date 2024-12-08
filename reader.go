package readgo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

// DefaultReader implements the source code reader
type DefaultReader struct {
	baseDir string
}

// NewReader creates a new source code reader
func NewReader(baseDir string) *DefaultReader {
	return &DefaultReader{
		baseDir: baseDir,
	}
}

// GetFileTree gets the file tree starting from the given root
func (r *DefaultReader) GetFileTree(ctx context.Context, root string, opts TreeOptions) (*FileTreeNode, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	absRoot := filepath.Join(r.baseDir, root)
	rootNode := &FileTreeNode{
		Name:     filepath.Base(absRoot),
		Path:     root,
		Type:     "directory",
		Children: make([]*FileTreeNode, 0),
	}

	var mu sync.Mutex
	var errList []error

	processedFiles := make(map[string]bool)
	dirNodes := make(map[string]*FileTreeNode)
	dirNodes[root] = rootNode

	err := filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, "~") || strings.HasSuffix(path, ".swp") {
			return nil
		}

		relPath, err := filepath.Rel(r.baseDir, path)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			mu.Lock()
			if processedFiles[relPath] {
				mu.Unlock()
				return nil
			}
			processedFiles[relPath] = true
			mu.Unlock()
		}

		if !info.IsDir() {
			if opts.FileTypes != FileTypeAll {
				switch opts.FileTypes {
				case FileTypeGo:
					if !strings.HasSuffix(path, ".go") {
						return nil
					}
				case FileTypeTest:
					if !strings.HasSuffix(path, "_test.go") {
						return nil
					}
				case FileTypeGenerated:
					if !strings.Contains(path, "generated") && !strings.Contains(path, "gen.go") {
						return nil
					}
				}
			}

			for _, pattern := range opts.ExcludePatterns {
				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return err
				}
				if matched {
					return nil
				}
			}
		}

		node := &FileTreeNode{
			Name: filepath.Base(path),
			Path: relPath,
			Type: "file",
		}
		if info.IsDir() {
			node.Type = "directory"
			node.Children = make([]*FileTreeNode, 0)
		}

		if relPath == root {
			return nil
		}

		parentPath := filepath.Dir(relPath)
		mu.Lock()
		if info.IsDir() {
			dirNodes[relPath] = node
		}

		parent, ok := dirNodes[parentPath]
		if !ok {
			parent = &FileTreeNode{
				Name:     filepath.Base(parentPath),
				Path:     parentPath,
				Type:     "directory",
				Children: make([]*FileTreeNode, 0),
			}
			dirNodes[parentPath] = parent

			grandParentPath := filepath.Dir(parentPath)
			if grandParent, ok := dirNodes[grandParentPath]; ok {
				grandParent.Children = append(grandParent.Children, parent)
			}
		}

		parent.Children = append(parent.Children, node)
		mu.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(errList) > 0 {
		return nil, fmt.Errorf("multiple errors: %v", errList)
	}

	sortTree(rootNode)

	return rootNode, nil
}

// ReadFile reads a source file
func (r *DefaultReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	absPath := filepath.Join(r.baseDir, filePath)
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, err
	}

	if info.Size() > 10*1024*1024 { // 10MB
		return nil, fmt.Errorf("file too large: %s", filePath)
	}

	file, err := os.Open(absPath)
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

func sortTree(node *FileTreeNode) {
	if node.Children == nil {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].Type != node.Children[j].Type {
			return node.Children[i].Type == "directory"
		}
		return node.Children[i].Name < node.Children[j].Name
	})

	for _, child := range node.Children {
		if child.Type == "directory" {
			sortTree(child)
		}
	}
}
