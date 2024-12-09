package readgo

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

// DefaultAnalyzer implements the CodeAnalyzer interface
type DefaultAnalyzer struct {
	workDir string
	cache   *Cache
	reader  SourceReader
	opts    *AnalyzerOptions
}

// NewAnalyzer creates a new DefaultAnalyzer with the given options
func NewAnalyzer(opts ...Option) *DefaultAnalyzer {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	reader := NewDefaultReader().WithWorkDir(options.WorkDir)

	return &DefaultAnalyzer{
		workDir: options.WorkDir,
		cache:   NewCache(options.CacheTTL),
		reader:  reader,
		opts:    options,
	}
}

// validatePath checks if the path is safe to access
func (a *DefaultAnalyzer) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Convert to absolute path
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(a.opts.WorkDir, path)
	}

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Check if the path is within workDir
	workDirAbs, err := filepath.Abs(a.opts.WorkDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if !strings.HasPrefix(absPath, workDirAbs) {
		return fmt.Errorf("path is outside of working directory")
	}

	return nil
}

// safeReadFile reads a file with security checks
func (a *DefaultAnalyzer) safeReadFile(path string) ([]byte, error) {
	if err := a.validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Get absolute path
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(a.opts.WorkDir, path)
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

// loadGoMod loads and parses the go.mod file
func (a *DefaultAnalyzer) loadGoMod() (*modfile.File, error) {
	goModPath := filepath.Join(a.opts.WorkDir, "go.mod")

	content, err := a.safeReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("read go.mod: %w", err)
	}

	modFile, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.mod: %w", err)
	}

	return modFile, nil
}

// loadPackage loads a package with basic configuration
// It supports both local and third-party packages
func (a *DefaultAnalyzer) loadPackage(pkgPath string) (*packages.Package, error) {
	if pkgPath == "" {
		return nil, &AnalysisError{
			Op:      "load package",
			Wrapped: ErrInvalidInput,
		}
	}

	// Handle relative paths
	if strings.HasPrefix(pkgPath, "./") || strings.HasPrefix(pkgPath, "../") {
		absPath := filepath.Clean(filepath.Join(a.workDir, pkgPath))
		cfg := &packages.Config{
			Mode: packages.NeedName |
				packages.NeedFiles |
				packages.NeedCompiledGoFiles |
				packages.NeedImports |
				packages.NeedTypes |
				packages.NeedTypesSizes |
				packages.NeedSyntax |
				packages.NeedTypesInfo |
				packages.NeedDeps,
			Dir: absPath,
			Env: append(os.Environ(), "GO111MODULE=on"),
		}

		// Load the package
		pkgs, err := packages.Load(cfg, ".")
		if err != nil {
			return nil, &PackageError{
				Package: pkgPath,
				Op:      "load",
				Wrapped: fmt.Errorf("load error: %w", err),
			}
		}

		if len(pkgs) == 0 {
			return nil, &PackageError{
				Package: pkgPath,
				Op:      "load",
				Wrapped: fmt.Errorf("no packages found: %w", ErrNotFound),
			}
		}

		// Print debug information
		fmt.Printf("Loaded package: %s\n", pkgs[0].PkgPath)
		fmt.Printf("Package name: %s\n", pkgs[0].Name)
		fmt.Printf("Package files: %v\n", pkgs[0].GoFiles)
		if len(pkgs[0].Errors) > 0 {
			fmt.Printf("Package errors: %v\n", pkgs[0].Errors)
		}

		// Check for package errors
		if len(pkgs[0].Errors) > 0 {
			errors := make([]string, len(pkgs[0].Errors))
			for i, err := range pkgs[0].Errors {
				errors[i] = err.Error()
			}
			return nil, &PackageError{
				Package: pkgPath,
				Op:      "load",
				Errors:  errors,
			}
		}

		return pkgs[0], nil
	}

	// For non-relative paths, use packages.Load
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesSizes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedDeps,
		Dir: a.workDir,
		Env: append(os.Environ(), "GO111MODULE=on"),
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, &PackageError{
			Package: pkgPath,
			Op:      "load",
			Wrapped: fmt.Errorf("load error: %w", err),
		}
	}

	if len(pkgs) == 0 {
		return nil, &PackageError{
			Package: pkgPath,
			Op:      "load",
			Wrapped: fmt.Errorf("no packages found: %w", ErrNotFound),
		}
	}

	if len(pkgs[0].Errors) > 0 {
		fmt.Printf("Package errors: %v\n", pkgs[0].Errors)
	}

	// Check for package errors
	if len(pkgs[0].Errors) > 0 {
		errors := make([]string, len(pkgs[0].Errors))
		for i, err := range pkgs[0].Errors {
			errors[i] = err.Error()
		}
		return nil, &PackageError{
			Package: pkgPath,
			Op:      "load",
			Errors:  errors,
		}
	}

	return pkgs[0], nil
}

// FindType finds a type in the given package
func (a *DefaultAnalyzer) FindType(ctx context.Context, pkgPath, typeName string) (result *TypeInfo, err error) {
	if a.cache != nil {
		key := TypeCacheKey{
			Package:  pkgPath,
			TypeName: typeName,
		}
		if cached, ok := a.cache.GetType(key); ok {
			return cached, nil
		}
		defer func() {
			if err == nil && result != nil {
				a.cache.SetType(key, result)
			}
		}()
	}

	if typeName == "" {
		return nil, &TypeLookupError{
			Package: pkgPath,
			Wrapped: ErrInvalidInput,
		}
	}

	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, &TypeLookupError{
			TypeName: typeName,
			Package:  pkgPath,
			Wrapped:  err,
		}
	}

	// First try to find in the package's scope
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj != nil {
		typeObj, ok := obj.(*types.TypeName)
		if !ok {
			return nil, &TypeLookupError{
				TypeName: typeName,
				Package:  pkgPath,
				Wrapped:  fmt.Errorf("symbol is not a type"),
			}
		}
		result = &TypeInfo{
			Name:       typeObj.Name(),
			Package:    pkgPath,
			IsExported: typeObj.Exported(),
			Type:       typeObj.Type().Underlying().String(),
		}
		return result, nil
	}

	// If not found, try to find in imported packages
	for importPath, imp := range pkg.Imports {
		if obj := imp.Types.Scope().Lookup(typeName); obj != nil {
			typeObj, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			result = &TypeInfo{
				Name:       typeObj.Name(),
				Package:    importPath,
				IsExported: typeObj.Exported(),
				Type:       typeObj.Type().Underlying().String(),
			}
			return result, nil
		}
	}

	return nil, &TypeLookupError{
		TypeName: typeName,
		Package:  pkgPath,
		Wrapped:  ErrNotFound,
	}
}

// FindInterface finds an interface in the given package
func (a *DefaultAnalyzer) FindInterface(ctx context.Context, pkgPath, interfaceName string) (result *TypeInfo, err error) {
	if a.cache != nil {
		key := TypeCacheKey{
			Package:  pkgPath,
			TypeName: interfaceName,
			Kind:     "interface",
		}
		if cached, ok := a.cache.GetType(key); ok {
			return cached, nil
		}
		defer func() {
			if err == nil && result != nil {
				a.cache.SetType(key, result)
			}
		}()
	}

	if interfaceName == "" {
		return nil, &TypeLookupError{
			Package: pkgPath,
			Kind:    "interface",
			Wrapped: ErrInvalidInput,
		}
	}

	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, &TypeLookupError{
			TypeName: interfaceName,
			Package:  pkgPath,
			Kind:     "interface",
			Wrapped:  err,
		}
	}

	// First try to find in the package's scope
	obj := pkg.Types.Scope().Lookup(interfaceName)
	if obj != nil {
		typeObj, ok := obj.(*types.TypeName)
		if !ok {
			return nil, &TypeLookupError{
				TypeName: interfaceName,
				Package:  pkgPath,
				Kind:     "interface",
				Wrapped:  fmt.Errorf("symbol is not a type"),
			}
		}
		if _, ok := typeObj.Type().Underlying().(*types.Interface); !ok {
			return nil, &TypeLookupError{
				TypeName: interfaceName,
				Package:  pkgPath,
				Kind:     "interface",
				Wrapped:  fmt.Errorf("type is not an interface"),
			}
		}
		result = &TypeInfo{
			Name:       typeObj.Name(),
			Package:    pkgPath,
			IsExported: typeObj.Exported(),
			Type:       typeObj.Type().Underlying().String(),
		}
		return result, nil
	}

	// If not found, try to find in imported packages
	for importPath, imp := range pkg.Imports {
		if obj := imp.Types.Scope().Lookup(interfaceName); obj != nil {
			typeObj, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			if _, ok := typeObj.Type().Underlying().(*types.Interface); !ok {
				continue
			}
			result = &TypeInfo{
				Name:       typeObj.Name(),
				Package:    importPath,
				IsExported: typeObj.Exported(),
				Type:       typeObj.Type().Underlying().String(),
			}
			return result, nil
		}
	}

	return nil, &TypeLookupError{
		TypeName: interfaceName,
		Package:  pkgPath,
		Kind:     "interface",
		Wrapped:  ErrNotFound,
	}
}

// FindFunction finds a function in the given package
// It supports both local and third-party packages
func (a *DefaultAnalyzer) FindFunction(ctx context.Context, pkgPath, funcName string) (*TypeInfo, error) {
	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, err
	}

	// First try to find in the package's scope
	obj := pkg.Types.Scope().Lookup(funcName)
	if obj == nil {
		// If not found, try to find in imported packages
		for _, imp := range pkg.Imports {
			if obj = imp.Types.Scope().Lookup(funcName); obj != nil {
				break
			}
		}
	}

	if obj == nil {
		return nil, fmt.Errorf("function not found: %s", funcName)
	}

	fun, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("not a function: %s", funcName)
	}

	return &TypeInfo{
		Name:       fun.Name(),
		Package:    pkg.PkgPath,
		IsExported: fun.Exported(),
		Type:       fun.Type().String(),
	}, nil
}

// AnalyzeFile analyzes a specific Go source file
func (a *DefaultAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error) {
	if err := a.validatePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Read file content
	content, err := a.reader.ReadSourceFile(ctx, filePath, ReadOptions{
		IncludeComments: true,
		StripSpaces:     false,
	})
	if err != nil {
		return nil, err
	}

	// Parse file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, &AnalysisError{Op: "parse file", Path: filePath, Wrapped: err}
	}

	result := &AnalysisResult{
		Name:       filepath.Base(filePath),
		Path:       filePath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
		Types:      make([]TypeInfo, 0),
		Functions:  make([]FunctionInfo, 0),
		Imports:    make([]string, 0),
	}

	// Collect imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		result.Imports = append(result.Imports, path)
	}

	// Analyze declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						info := TypeInfo{
							Name:       typeSpec.Name.Name,
							Package:    file.Name.Name,
							IsExported: typeSpec.Name.IsExported(),
						}

						switch t := typeSpec.Type.(type) {
						case *ast.InterfaceType:
							methods := make([]string, 0)
							for _, method := range t.Methods.List {
								if len(method.Names) > 0 {
									methodType := types.ExprString(method.Type)
									for _, name := range method.Names {
										methods = append(methods, fmt.Sprintf("%s%s", name.Name, methodType))
									}
								}
							}
							info.Type = fmt.Sprintf("interface{%s}", strings.Join(methods, "; "))
						case *ast.StructType:
							fields := make([]string, 0)
							for _, field := range t.Fields.List {
								if len(field.Names) > 0 {
									fieldType := types.ExprString(field.Type)
									for _, name := range field.Names {
										fields = append(fields, fmt.Sprintf("%s %s", name.Name, fieldType))
									}
								}
							}
							info.Type = fmt.Sprintf("struct{%s}", strings.Join(fields, "; "))
						default:
							info.Type = fmt.Sprintf("%T", t)
						}

						result.Types = append(result.Types, info)
					}
				}
			}
		case *ast.FuncDecl:
			if d.Name != nil {
				result.Functions = append(result.Functions, FunctionInfo{
					Name:       d.Name.Name,
					Package:    file.Name.Name,
					IsExported: d.Name.IsExported(),
				})
			}
		}
	}

	return result, nil
}

// AnalyzePackage analyzes a Go package
func (a *DefaultAnalyzer) AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error) {
	// Load the package
	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, &AnalysisError{
			Op:      "analyze package",
			Path:    pkgPath,
			Wrapped: fmt.Errorf("failed to load package: %w", err),
		}
	}

	// Create result
	result := &AnalysisResult{
		Name:       pkg.Name,
		Path:       pkg.PkgPath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Extract types
	for _, obj := range pkg.TypesInfo.Defs {
		if obj == nil {
			continue
		}

		if named, ok := obj.Type().(*types.Named); ok {
			result.Types = append(result.Types, TypeInfo{
				Name:       obj.Name(),
				Package:    pkg.PkgPath,
				Type:       named.String(),
				IsExported: obj.Exported(),
			})
		}
	}

	// Extract functions
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok {
				result.Functions = append(result.Functions, FunctionInfo{
					Name:       funcDecl.Name.Name,
					Package:    pkg.PkgPath,
					IsExported: funcDecl.Name.IsExported(),
				})
			}
			return true
		})
	}

	// Extract imports
	for _, imp := range pkg.Imports {
		result.Imports = append(result.Imports, imp.PkgPath)
	}

	return result, nil
}

// AnalyzeProject analyzes a Go project at the specified path
func (a *DefaultAnalyzer) AnalyzeProject(ctx context.Context, projectPath string) (*AnalysisResult, error) {
	if projectPath == "" {
		projectPath = "."
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, &AnalysisError{
			Op:      "analyze project",
			Path:    projectPath,
			Wrapped: fmt.Errorf("failed to get absolute path: %w", err),
		}
	}

	// Create result
	result := &AnalysisResult{
		Name:       filepath.Base(absPath),
		Path:       absPath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Load the package
	pkg, err := a.loadPackage(".")
	if err != nil {
		return nil, &AnalysisError{
			Op:      "analyze project",
			Path:    projectPath,
			Wrapped: fmt.Errorf("failed to load package: %w", err),
		}
	}

	// Extract types
	for _, obj := range pkg.TypesInfo.Defs {
		if obj == nil {
			continue
		}

		if named, ok := obj.Type().(*types.Named); ok {
			result.Types = append(result.Types, TypeInfo{
				Name:       obj.Name(),
				Package:    pkg.PkgPath,
				Type:       named.String(),
				IsExported: obj.Exported(),
			})
		}
	}

	// Extract functions
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok {
				result.Functions = append(result.Functions, FunctionInfo{
					Name:       funcDecl.Name.Name,
					Package:    pkg.PkgPath,
					IsExported: funcDecl.Name.IsExported(),
				})
			}
			return true
		})
	}

	// Extract imports
	for _, imp := range pkg.Imports {
		result.Imports = append(result.Imports, imp.PkgPath)
	}

	return result, nil
}

// GetCacheStats returns cache statistics if caching is enabled
func (a *DefaultAnalyzer) GetCacheStats() map[string]interface{} {
	if a.cache == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	stats := a.cache.Stats()
	stats["enabled"] = true
	return stats
}
