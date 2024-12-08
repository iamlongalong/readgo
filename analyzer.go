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
	"sort"
	"strings"
	"time"

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

// loadPackage loads a package with basic configuration
// It supports both local and third-party packages
func (a *DefaultAnalyzer) loadPackage(pkgPath string) (*packages.Package, error) {
	if pkgPath == "" {
		return nil, &AnalysisError{
			Op:      "load package",
			Wrapped: ErrInvalidInput,
		}
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Dir: a.workDir,
	}

	// Handle different types of package paths
	if pkgPath == "." {
		pkgPath = "./..."
	} else if !filepath.IsAbs(pkgPath) && !strings.Contains(pkgPath, "/") {
		pkgPath = "./" + pkgPath
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, &PackageError{
			Package: pkgPath,
			Op:      "load",
			Wrapped: err,
		}
	}

	if len(pkgs) == 0 {
		return nil, &PackageError{
			Package: pkgPath,
			Op:      "load",
			Wrapped: ErrNotFound,
		}
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

// resolvePath resolves a file or directory path against the working directory
// DEPRECATED: This function is no longer used and will be removed
func (a *DefaultAnalyzer) resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(a.opts.WorkDir, path), nil
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
	if filePath == "" {
		return nil, ErrInvalidInput
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
// It supports analyzing both local and third-party packages
func (a *DefaultAnalyzer) AnalyzePackage(ctx context.Context, pkgPath string) (result *AnalysisResult, err error) {
	if a.cache != nil {
		key := PackageCacheKey{
			Path: pkgPath,
			Mode: "full",
		}
		if cached, ok := a.cache.GetPackage(key); ok {
			return cached, nil
		}
		defer func() {
			if err == nil && result != nil {
				a.cache.SetPackage(key, result)
			}
		}()
	}

	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, err
	}

	result = &AnalysisResult{
		Name:       pkg.Name,
		Path:       pkgPath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// First analyze the package's own scope
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		switch obj := obj.(type) {
		case *types.TypeName:
			result.Types = append(result.Types, TypeInfo{
				Name:       obj.Name(),
				Package:    pkgPath,
				Type:       obj.Type().Underlying().String(),
				IsExported: obj.Exported(),
			})
		case *types.Func:
			result.Functions = append(result.Functions, FunctionInfo{
				Name:       obj.Name(),
				Package:    pkgPath,
				IsExported: obj.Exported(),
			})
		}
	}

	// Then analyze imports and their types
	for importPath, imp := range pkg.Imports {
		// Add import path
		result.Imports = append(result.Imports, importPath)

		// Add imported types and functions
		scope := imp.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			switch obj := obj.(type) {
			case *types.TypeName:
				result.Types = append(result.Types, TypeInfo{
					Name:       obj.Name(),
					Package:    importPath,
					Type:       obj.Type().Underlying().String(),
					IsExported: obj.Exported(),
				})
			case *types.Func:
				result.Functions = append(result.Functions, FunctionInfo{
					Name:       obj.Name(),
					Package:    importPath,
					IsExported: obj.Exported(),
				})
			}
		}
	}

	return result, nil
}

// AnalyzeProject analyzes a Go project at the specified path
func (a *DefaultAnalyzer) AnalyzeProject(ctx context.Context, projectPath string) (*AnalysisResult, error) {
	if projectPath == "" {
		return nil, ErrInvalidInput
	}

	// Read go.mod to get project name
	goModPath := filepath.Join(a.workDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, &AnalysisError{Op: "read go.mod", Path: goModPath, Wrapped: err}
	}

	// Extract module name from go.mod
	var moduleName string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "module ") {
			moduleName = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "module "))
			break
		}
	}

	if moduleName == "" {
		return nil, &AnalysisError{Op: "parse go.mod", Path: goModPath, Wrapped: fmt.Errorf("module name not found")}
	}

	// Get all Go files in the project
	files, err := a.reader.GetPackageFiles(ctx, ".", TreeOptions{FileTypes: FileTypeGo})
	if err != nil {
		return nil, err
	}

	// Analyze each file
	var types []TypeInfo
	var functions []FunctionInfo
	imports := make(map[string]struct{})

	for _, file := range files {
		result, err := a.AnalyzeFile(ctx, file.Path)
		if err != nil {
			return nil, err
		}

		types = append(types, result.Types...)
		functions = append(functions, result.Functions...)
		for _, imp := range result.Imports {
			imports[imp] = struct{}{}
		}
	}

	// Convert imports map to slice
	importsList := make([]string, 0, len(imports))
	for imp := range imports {
		importsList = append(importsList, imp)
	}
	sort.Strings(importsList)

	return &AnalysisResult{
		Name:       moduleName,
		Path:       a.workDir,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
		Types:      types,
		Functions:  functions,
		Imports:    importsList,
	}, nil
}

// contains checks if a string slice contains a specific string
// DEPRECATED: This function is no longer used and will be removed
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
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
