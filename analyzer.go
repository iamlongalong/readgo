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

	"golang.org/x/tools/go/packages"
)

// DefaultAnalyzer implements the CodeAnalyzer interface
type DefaultAnalyzer struct {
	workDir string
	cache   *Cache
	reader  SourceReader
}

// NewAnalyzer creates a new DefaultAnalyzer instance
func NewAnalyzer(opts ...Option) *DefaultAnalyzer {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	return &DefaultAnalyzer{
		workDir: options.WorkDir,
		cache:   NewCache(options.CacheTTL),
		reader:  NewSourceReader(options.WorkDir),
	}
}

// AnalyzeFile analyzes a specific Go source file
func (a *DefaultAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error) {
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

	// Load the package
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
		return nil, &TypeLookupError{
			TypeName: typeName,
			Package:  pkgPath,
			Wrapped:  err,
		}
	}

	if len(pkgs) == 0 {
		return nil, &TypeLookupError{
			TypeName: typeName,
			Package:  pkgPath,
			Wrapped:  fmt.Errorf("no packages found"),
		}
	}

	pkg := pkgs[0]

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

	// Load the package
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
		return nil, &TypeLookupError{
			TypeName: interfaceName,
			Package:  pkgPath,
			Kind:     "interface",
			Wrapped:  err,
		}
	}

	if len(pkgs) == 0 {
		return nil, &TypeLookupError{
			TypeName: interfaceName,
			Package:  pkgPath,
			Kind:     "interface",
			Wrapped:  fmt.Errorf("no packages found"),
		}
	}

	pkg := pkgs[0]

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

// AnalyzeProject analyzes a Go project at the specified path
func (a *DefaultAnalyzer) AnalyzeProject(ctx context.Context, projectPath string) (*AnalysisResult, error) {
	if projectPath == "" {
		projectPath = "."
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filepath.Join(a.workDir, projectPath))
	if err != nil {
		return nil, &AnalysisError{
			Op:      "analyze project",
			Path:    projectPath,
			Wrapped: fmt.Errorf("failed to get absolute path: %w", err),
		}
	}

	// Create result
	result := &AnalysisResult{
		Name:       "main", // Use package name from the first package
		Path:       absPath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Load the package
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

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, &AnalysisError{
			Op:      "analyze project",
			Path:    projectPath,
			Wrapped: fmt.Errorf("failed to load packages: %w", err),
		}
	}

	for _, pkg := range pkgs {
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
	}

	return result, nil
}

// AnalyzePackage analyzes a Go package
func (a *DefaultAnalyzer) AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error) {
	// Load the package
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
		return nil, &AnalysisError{
			Op:      "analyze package",
			Path:    pkgPath,
			Wrapped: fmt.Errorf("failed to load package: %w", err),
		}
	}

	if len(pkgs) == 0 {
		return nil, &AnalysisError{
			Op:      "analyze package",
			Path:    pkgPath,
			Wrapped: fmt.Errorf("no packages found"),
		}
	}

	pkg := pkgs[0]

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

// GetCacheStats returns cache statistics
func (a *DefaultAnalyzer) GetCacheStats() map[string]interface{} {
	stats := a.cache.Stats()
	stats["enabled"] = a.cache.ttl > 0
	return stats
}
