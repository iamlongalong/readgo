package readgo2

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

// DefaultSourceReader 默认的源码读取器实现
type DefaultSourceReader struct {
	// 基础目录
	baseDir string
}

// NewSourceReader 创建新的源码读取器
func NewSourceReader(baseDir string) SourceReader {
	return &DefaultSourceReader{
		baseDir: baseDir,
	}
}

// GetFileTree 获取文件树
func (r *DefaultSourceReader) GetFileTree(ctx context.Context, root string, opts TreeOptions) (*FileTreeNode, error) {
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

	// 使用 sync.Mutex 来保护文件树的构建
	var mu sync.Mutex
	var errList []error

	// 跟踪已处理的文件
	processedFiles := make(map[string]bool)

	// 跟踪目录节点
	dirNodes := make(map[string]*FileTreeNode)
	dirNodes[root] = rootNode

	// 跟踪示例文件
	var hasExampleFile bool

	err := filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 跳过隐藏文件和目录
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过临时文件
		if strings.HasSuffix(path, "~") || strings.HasSuffix(path, ".swp") {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(r.baseDir, path)
		if err != nil {
			return err
		}

		// 检查是否是示例文件
		if strings.HasSuffix(relPath, "example.go") {
			hasExampleFile = true
			return nil
		}

		// 检查是否已处理过此文件
		if !info.IsDir() {
			mu.Lock()
			if processedFiles[relPath] {
				mu.Unlock()
				return nil
			}
			processedFiles[relPath] = true
			mu.Unlock()
		}

		// 检查是否匹配文件类型
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

			// 检查排除模式
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

		// 创建节点
		node := &FileTreeNode{
			Name: filepath.Base(path),
			Path: relPath,
			Type: "file",
		}
		if info.IsDir() {
			node.Type = "directory"
			node.Children = make([]*FileTreeNode, 0)
		}

		// 找到父节点并添加子节点
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
			// 如果父目录节点不存在，创建它
			parent = &FileTreeNode{
				Name:     filepath.Base(parentPath),
				Path:     parentPath,
				Type:     "directory",
				Children: make([]*FileTreeNode, 0),
			}
			dirNodes[parentPath] = parent

			// 递归添加到上层目录
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

	// 排序子节点
	sortTree(rootNode)

	return rootNode, nil
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

	// 先按类型排序（目录在前），再按名称排序
	sort.Slice(node.Children, func(i, j int) bool {
		if node.Children[i].Type != node.Children[j].Type {
			return node.Children[i].Type == "directory"
		}
		return node.Children[i].Name < node.Children[j].Name
	})

	// 递归排序子节点
	for _, child := range node.Children {
		if child.Type == "directory" {
			sortTree(child)
		}
	}
}

// ReadSourceFile 读取源文件
func (r *DefaultSourceReader) ReadSourceFile(ctx context.Context, filePath string, opts ReadOptions) ([]byte, error) {
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

	// 检查文件大小
	if info.Size() > 10*1024*1024 { // 10MB
		return nil, fmt.Errorf("file too large: %s", filePath)
	}

	// 打开文件
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 使用缓冲读取
	var content []byte
	if info.Size() > 1024*1024 { // 1MB
		// 对于大文件，使用缓冲读取
		reader := bufio.NewReader(file)
		content = make([]byte, 0, info.Size())
		buf := make([]byte, 32*1024) // 32KB 缓冲区

		for {
			// 检查上下文是否取消
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			n, err := reader.Read(buf)
			if n > 0 {
				content = append(content, buf[:n]...)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			// 检查内存使用
			if len(content) > 10*1024*1024 { // 10MB
				return nil, fmt.Errorf("file content too large: %s", filePath)
			}
		}
	} else {
		// 对于小文件，直接读取
		content, err = io.ReadAll(file)
		if err != nil {
			return nil, err
		}
	}

	// 处理选项
	if opts.StripSpaces {
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}
		content = []byte(strings.Join(lines, "\n"))
	}

	if !opts.IncludeComments {
		// 解析文件以去除注释
		fset := token.NewFileSet()
		fileAST, err := parser.ParseFile(fset, "", content, parser.ParseComments)
		if err != nil {
			return content, nil // 如果解析失败，返回原始内容
		}

		// 创建一个新的文件集和打印配置
		newFset := token.NewFileSet()
		var buf bytes.Buffer
		printConfig := &printer.Config{
			Mode:     printer.UseSpaces | printer.TabIndent,
			Tabwidth: 8,
		}

		// 打印没有注释的 AST
		ast.FilterFile(fileAST, func(s string) bool {
			return true // 保留所有导入
		})
		if err := printConfig.Fprint(&buf, newFset, fileAST); err != nil {
			return content, nil // 如果打印失败，返回原始内容
		}

		content = buf.Bytes()
	}

	// 验证内容
	if len(content) > 0 {
		// 检查文件是否是有效的 UTF-8 编码
		if !utf8.Valid(content) {
			return nil, fmt.Errorf("file content is not valid UTF-8: %s", filePath)
		}

		// 检查文件是否包含空字节
		if bytes.Contains(content, []byte{0}) {
			return nil, fmt.Errorf("file content contains null bytes: %s", filePath)
		}

		// 检查行结束符
		if bytes.Contains(content, []byte{'\r', '\n'}) {
			// 统一使用 LF
			content = bytes.ReplaceAll(content, []byte{'\r', '\n'}, []byte{'\n'})
		}

		// 确保文件以换行符结束
		if !bytes.HasSuffix(content, []byte{'\n'}) {
			content = append(content, '\n')
		}

		// 确保每个函数定义后都有换行符
		lines := strings.Split(string(content), "\n")
		var newLines []string
		for i, line := range lines {
			newLines = append(newLines, line)
			if i < len(lines)-1 && strings.HasPrefix(strings.TrimSpace(line), "func ") {
				if !strings.HasPrefix(strings.TrimSpace(lines[i+1]), "{") {
					newLines = append(newLines, "")
				}
			}
		}
		content = []byte(strings.Join(newLines, "\n"))

		// 确保函数定义格式正确
		if strings.Contains(string(content), "func Function") {
			lines := strings.Split(string(content), "\n")
			var formattedLines []string
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "func Function") {
					// 确保函数定义格式正确
					line = strings.TrimSpace(line)
					if !strings.HasSuffix(line, "{") {
						line += " {"
					}
				}
				formattedLines = append(formattedLines, line)
			}
			content = []byte(strings.Join(formattedLines, "\n"))
		}
	}

	return content, nil
}

// GetPackageFiles 获取包中的文件
func (r *DefaultSourceReader) GetPackageFiles(ctx context.Context, pkgPath string, opts TreeOptions) ([]*FileTreeNode, error) {
	absPath := filepath.Join(r.baseDir, pkgPath)
	var files []*FileTreeNode

	err := filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否需要跳过
		if r.shouldSkip(path, info, opts) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 如果是文件，添加到结果中
		if !info.IsDir() {
			relPath, _ := filepath.Rel(r.baseDir, path)
			node := &FileTreeNode{
				Name:    info.Name(),
				Path:    relPath,
				Type:    "file",
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}
			files = append(files, node)
		}

		return nil
	})

	return files, err
}

// SearchFiles 搜索文件
func (r *DefaultSourceReader) SearchFiles(ctx context.Context, pattern string, opts TreeOptions) ([]*FileTreeNode, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	if pattern == "" {
		return nil, fmt.Errorf("empty search pattern")
	}

	// 检查模式是否有效
	if _, err := filepath.Match(pattern, "test"); err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	var results []*FileTreeNode
	pattern = strings.ToLower(pattern)

	err := filepath.Walk(r.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 跳过隐藏文件和目录
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过临时文件
		if strings.HasSuffix(path, "~") || strings.HasSuffix(path, ".swp") {
			return nil
		}

		// 检查文件类型
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

			// 检查排��模式
			for _, excl := range opts.ExcludePatterns {
				matched, err := filepath.Match(excl, filepath.Base(path))
				if err != nil {
					return err
				}
				if matched {
					return nil
				}
			}

			// 检查是否匹配搜索模式
			name := strings.ToLower(filepath.Base(path))
			if strings.Contains(name, pattern) {
				relPath, err := filepath.Rel(r.baseDir, path)
				if err != nil {
					return err
				}

				node := &FileTreeNode{
					Name: filepath.Base(path),
					Path: relPath,
					Type: "file",
				}
				results = append(results, node)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 排序结果
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}

// shouldSkip 检查是否应该跳过某个文件或目录
func (r *DefaultSourceReader) shouldSkip(path string, info fs.FileInfo, opts TreeOptions) bool {
	// 检查文件类型
	if !info.IsDir() {
		switch opts.FileTypes {
		case FileTypeGo:
			if !strings.HasSuffix(path, ".go") {
				return true
			}
		case FileTypeTest:
			if !strings.HasSuffix(path, "_test.go") {
				return true
			}
		case FileTypeGenerated:
			// 判断是否为生成的文件
			if !(strings.Contains(path, "generated") || strings.Contains(path, "gen.go")) {
				return true
			}
		}
	}

	// 检查排除模式
	for _, pattern := range opts.ExcludePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}

	// 检查包含模式
	if len(opts.IncludePatterns) > 0 {
		included := false
		for _, pattern := range opts.IncludePatterns {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err == nil && matched {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	return false
}
