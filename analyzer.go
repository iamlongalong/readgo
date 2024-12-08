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

// DefaultAnalyzer implements the code analyzer interface
type DefaultAnalyzer struct {
	// workDir is the working directory for the analyzer
	// It's used as the base directory for resolving relative paths
	// and as the root directory for project analysis
	workDir string
}

// NewAnalyzer creates a new analyzer instance
// workDir specifies the working directory for the analyzer
// If workDir is empty, the current directory will be used
func NewAnalyzer(workDir string) *DefaultAnalyzer {
	if workDir == "" {
		workDir = "."
	}
	return &DefaultAnalyzer{workDir: workDir}
}

// loadPackage loads a package with basic configuration
// It supports both local and third-party packages
func (a *DefaultAnalyzer) loadPackage(pkgPath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Dir: a.workDir,
	}

	// Handle different types of package paths:
	// 1. Empty or "." -> analyze current directory
	// 2. Relative path -> resolve against workDir
	// 3. Absolute path -> use as is
	// 4. Import path (e.g. "github.com/user/repo") -> use as is
	if pkgPath == "" || pkgPath == "." {
		pkgPath = "./..."
	} else if !filepath.IsAbs(pkgPath) && !strings.Contains(pkgPath, "/") {
		// For local packages without path separator
		pkgPath = "./" + pkgPath
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found at %s", pkgPath)
	}

	// Check for package errors
	if len(pkgs[0].Errors) > 0 {
		return nil, fmt.Errorf("package has errors: %v", pkgs[0].Errors[0])
	}

	return pkgs[0], nil
}

// resolvePath resolves a file or directory path against the working directory
// It returns an absolute path and any error encountered
func (a *DefaultAnalyzer) resolvePath(path string) (string, error) {
	if path == "" {
		path = "."
	}

	// If the path is already absolute, use it as is
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Resolve relative path against workDir
	absPath, err := filepath.Abs(filepath.Join(a.workDir, path))
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	return absPath, nil
}

// FindType finds a type in the given package
// It supports both local and third-party packages
func (a *DefaultAnalyzer) FindType(ctx context.Context, pkgPath, typeName string) (*TypeInfo, error) {
	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s: %w", pkgPath, err)
	}

	// First try to find in the package's scope
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj != nil {
		typeObj, ok := obj.(*types.TypeName)
		if !ok {
			return nil, fmt.Errorf("symbol %q in package %s is not a type", typeName, pkgPath)
		}
		return &TypeInfo{
			Name:       typeObj.Name(),
			Package:    pkgPath,
			IsExported: typeObj.Exported(),
			Type:       typeObj.Type().Underlying().String(),
		}, nil
	}

	// If not found, try to find in imported packages
	for importPath, imp := range pkg.Imports {
		if obj := imp.Types.Scope().Lookup(typeName); obj != nil {
			typeObj, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			return &TypeInfo{
				Name:       typeObj.Name(),
				Package:    importPath,
				IsExported: typeObj.Exported(),
				Type:       typeObj.Type().Underlying().String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("type %q not found in package %s or its imports", typeName, pkgPath)
}

// FindInterface finds an interface in the given package
// It supports both local and third-party packages
func (a *DefaultAnalyzer) FindInterface(ctx context.Context, pkgPath, interfaceName string) (*TypeInfo, error) {
	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s: %w", pkgPath, err)
	}

	// First try to find in the package's scope
	obj := pkg.Types.Scope().Lookup(interfaceName)
	if obj != nil {
		typeObj, ok := obj.(*types.TypeName)
		if !ok {
			return nil, fmt.Errorf("symbol %q in package %s is not a type", interfaceName, pkgPath)
		}
		if _, ok := typeObj.Type().Underlying().(*types.Interface); !ok {
			return nil, fmt.Errorf("type %q in package %s is not an interface", interfaceName, pkgPath)
		}
		return &TypeInfo{
			Name:       typeObj.Name(),
			Package:    pkgPath,
			IsExported: typeObj.Exported(),
			Type:       typeObj.Type().Underlying().String(),
		}, nil
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
			return &TypeInfo{
				Name:       typeObj.Name(),
				Package:    importPath,
				IsExported: typeObj.Exported(),
				Type:       typeObj.Type().Underlying().String(),
			}, nil
		}
	}

	return nil, fmt.Errorf("interface %q not found in package %s or its imports", interfaceName, pkgPath)
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

// AnalyzeFile analyzes a Go source file
// filePath can be:
// - Absolute path to the file
// - Path relative to the working directory
func (a *DefaultAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("empty file path")
	}

	absPath, err := a.resolvePath(filePath)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file access error: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Get the directory containing the file
	dir := filepath.Dir(absPath)
	cfg := &packages.Config{
		Mode: packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found")
	}

	pkg := pkgs[0]
	result := &AnalysisResult{
		Name:       filepath.Base(filePath),
		Path:       filePath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Collect imports and analyze imported packages
	for _, imp := range file.Imports {
		if imp.Path != nil {
			importPath := strings.Trim(imp.Path.Value, "\"")
			result.Imports = append(result.Imports, importPath)

			// Try to analyze imported package
			for _, pkg := range pkg.Imports {
				if pkg.PkgPath == importPath {
					scope := pkg.Types.Scope()
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
					break
				}
			}
		}
	}

	// Analyze AST
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			result.Functions = append(result.Functions, FunctionInfo{
				Name:       node.Name.Name,
				Package:    pkg.PkgPath,
				IsExported: node.Name.IsExported(),
			})
		case *ast.TypeSpec:
			if typeInfo, ok := pkg.TypesInfo.Types[node.Type]; ok {
				result.Types = append(result.Types, TypeInfo{
					Name:       node.Name.Name,
					Package:    pkg.PkgPath,
					Type:       typeInfo.Type.String(),
					IsExported: node.Name.IsExported(),
				})
			}
		}
		return true
	})

	return result, nil
}

// AnalyzePackage analyzes a Go package
// It supports analyzing both local and third-party packages
func (a *DefaultAnalyzer) AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error) {
	pkg, err := a.loadPackage(pkgPath)
	if err != nil {
		return nil, err
	}

	result := &AnalysisResult{
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

// AnalyzeProject analyzes a Go project
// projectPath can be:
// - Empty or "." to analyze the working directory
// - Absolute path to the project
// - Path relative to the working directory
func (a *DefaultAnalyzer) AnalyzeProject(ctx context.Context, projectPath string) (*AnalysisResult, error) {
	absPath, err := a.resolvePath(projectPath)
	if err != nil {
		return nil, err
	}

	cfg := &packages.Config{
		Mode: packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Dir: absPath,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	result := &AnalysisResult{
		Name:       filepath.Base(absPath),
		Path:       absPath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	for _, pkg := range pkgs {
		// Skip packages with errors
		if len(pkg.Errors) > 0 {
			continue
		}

		// Add imports
		for _, imp := range pkg.Imports {
			if !contains(result.Imports, imp.PkgPath) {
				result.Imports = append(result.Imports, imp.PkgPath)
			}
		}

		// Analyze package scope
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			switch obj := obj.(type) {
			case *types.TypeName:
				result.Types = append(result.Types, TypeInfo{
					Name:       obj.Name(),
					Package:    pkg.PkgPath,
					Type:       obj.Type().Underlying().String(),
					IsExported: obj.Exported(),
				})
			case *types.Func:
				result.Functions = append(result.Functions, FunctionInfo{
					Name:       obj.Name(),
					Package:    pkg.PkgPath,
					IsExported: obj.Exported(),
				})
			}
		}
	}

	return result, nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
