package readgo2

import (
	"context"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
)

// DefaultValidator 默认的代码验证器实现
type DefaultValidator struct {
	baseDir string
	fset    *token.FileSet
}

// NewValidator 创建新的代码验证器
func NewValidator(baseDir string) Validator {
	return &DefaultValidator{
		baseDir: baseDir,
		fset:    token.NewFileSet(),
	}
}

// ValidateFile 验证特定文件
func (v *DefaultValidator) ValidateFile(ctx context.Context, filePath string, level ValidationLevel) (*ValidationResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	result := &ValidationResult{
		Valid:      true,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	absPath := filepath.Join(v.baseDir, filePath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Level:   "error",
			Code:    "FILE_NOT_FOUND",
			Message: fmt.Sprintf("file not found: %s", filePath),
			File:    filePath,
		})
		return result, nil
	}

	// 基础语法检查
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Level:   "error",
			Code:    "SYNTAX_ERROR",
			Message: err.Error(),
			File:    filePath,
		})
		result.Stats.SyntaxErrors++
		return result, nil
	}

	// 标准验证：类型检查
	if level >= ValidationLevelStandard {
		conf := types.Config{
			Error: func(err error) {
				if tErr, ok := err.(types.Error); ok {
					pos := fset.Position(tErr.Pos)
					result.Errors = append(result.Errors, ValidationError{
						Level:   "error",
						Code:    "TYPE_ERROR",
						Message: tErr.Msg,
						File:    filePath,
						Line:    pos.Line,
						Column:  pos.Column,
					})
					result.Stats.TypeErrors++
					result.Valid = false
				}
			},
			Importer: importer.Default(),
		}

		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}

		_, err := conf.Check("", fset, []*ast.File{file}, info)
		if err != nil {
			// 类型错误已经通过 Error 回调处理
		}

		// 检查未使用的导入
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			used := false
			for _, use := range info.Uses {
				if use.Pkg() != nil && use.Pkg().Path() == path {
					used = true
					break
				}
			}
			if !used {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Level:   "warning",
					Code:    "UNUSED_IMPORT",
					Message: fmt.Sprintf("unused import: %s", path),
					File:    filePath,
				})
				result.Stats.UnusedImports++
			}
		}

		// 检查未使用的变量
		for ident, obj := range info.Defs {
			if obj == nil {
				continue
			}
			if v, ok := obj.(*types.Var); ok && !v.IsField() {
				used := false
				for _, useObj := range info.Uses {
					if useObj == obj {
						used = true
						break
					}
				}
				if !used {
					pos := fset.Position(ident.Pos())
					result.Warnings = append(result.Warnings, ValidationWarning{
						Level:   "warning",
						Code:    "UNUSED_VAR",
						Message: fmt.Sprintf("unused variable: %s", ident.Name),
						File:    filePath,
						Line:    pos.Line,
						Column:  pos.Column,
					})
					result.Stats.UnusedVariables++
				}
			}
		}
	}

	// 严格验证：依赖检查 + lint
	if level == ValidationLevelStrict {
		// 使用 packages 包进行依赖分析
		cfg := &packages.Config{
			Mode: packages.NeedImports | packages.NeedDeps | packages.NeedTypes,
			Dir:  filepath.Dir(absPath),
		}
		pkgs, err := packages.Load(cfg, "file="+absPath)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Level:   "error",
				Code:    "DEP_ERROR",
				Message: err.Error(),
				File:    filePath,
			})
			result.Stats.DependencyErrors++
			result.Valid = false
		} else if len(pkgs) > 0 {
			for _, err := range pkgs[0].Errors {
				result.Errors = append(result.Errors, ValidationError{
					Level:   "error",
					Code:    "DEP_ERROR",
					Message: err.Error(),
					File:    filePath,
				})
				result.Stats.DependencyErrors++
				result.Valid = false
			}

			// 检查外部依赖
			for _, imp := range pkgs[0].Imports {
				if !strings.HasPrefix(imp.PkgPath, ".") && !strings.HasPrefix(imp.PkgPath, "internal/") {
					if strings.Contains(imp.PkgPath, "golang.org/x/") || strings.Contains(imp.PkgPath, "github.com/") {
						result.HasExternalDeps = true
						result.ExternalDeps = append(result.ExternalDeps, imp.PkgPath)
					}
				}
			}
		}
	}

	result.Stats.FilesChecked++
	result.Duration = time.Since(result.AnalyzedAt).String()

	return result, nil
}

// ValidatePackage 验证特定包
func (v *DefaultValidator) ValidatePackage(ctx context.Context, pkgPath string, level ValidationLevel) (*ValidationResult, error) {
	if level == "" {
		level = ValidationLevelStandard
	}

	result := &ValidationResult{
		Valid: true,
		Stats: ValidationStats{},
	}

	// 获取包下的所有Go文件
	absPkgPath := filepath.Join(v.baseDir, pkgPath)
	entries, err := os.ReadDir(absPkgPath)
	if err != nil {
		return nil, fmt.Errorf("read package directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !isGoFile(entry.Name()) {
			continue
		}

		fileResult, err := v.ValidateFile(ctx, filepath.Join(pkgPath, entry.Name()), level)
		if err != nil {
			return nil, err
		}

		result.Stats.FilesChecked++
		result.Stats.ErrorCount += len(fileResult.Errors)
		result.Stats.WarningCount += len(fileResult.Warnings)
		result.Stats.SyntaxErrors += fileResult.Stats.SyntaxErrors
		result.Stats.TypeErrors += fileResult.Stats.TypeErrors
		result.Stats.DependencyErrors += fileResult.Stats.DependencyErrors

		result.Errors = append(result.Errors, fileResult.Errors...)
		result.Warnings = append(result.Warnings, fileResult.Warnings...)

		if !fileResult.Valid {
			result.Valid = false
		}
	}

	return result, nil
}

// ValidateProject 验证整个项目
func (v *DefaultValidator) ValidateProject(ctx context.Context, level ValidationLevel) (*ValidationResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	result := &ValidationResult{
		Valid:      true,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// 配置包加载器
	cfg := &packages.Config{
		Mode: packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedModule |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles,
		Context: ctx,
		Dir:     v.baseDir,
		Tests:   true,
	}

	// 加载所有包
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	// 使用 sync.WaitGroup 和 sync.Mutex 来安全地收集信息
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 收集所有包的信息
	for _, pkg := range pkgs {
		wg.Add(1)
		go func(pkg *packages.Package) {
			defer wg.Done()

			// 检查包错误
			if len(pkg.Errors) > 0 {
				mu.Lock()
				for _, err := range pkg.Errors {
					result.Errors = append(result.Errors, ValidationError{
						Level:   "error",
						Code:    "PKG_ERROR",
						Message: err.Error(),
						File:    pkg.PkgPath,
					})
				}
				result.Valid = false
				mu.Unlock()
				return
			}

			// 收集导入信息
			mu.Lock()
			for _, imp := range pkg.Imports {
				if !strings.HasPrefix(imp.PkgPath, ".") && !strings.HasPrefix(imp.PkgPath, "internal/") {
					if strings.Contains(imp.PkgPath, "golang.org/x/") || strings.Contains(imp.PkgPath, "github.com/") {
						result.HasExternalDeps = true
						result.ExternalDeps = append(result.ExternalDeps, imp.PkgPath)
					}
				}
				result.Imports = append(result.Imports, imp.PkgPath)
			}
			mu.Unlock()

			// 标准验证：类型检查
			if level >= ValidationLevelStandard {
				for _, file := range pkg.Syntax {
					ast.Inspect(file, func(n ast.Node) bool {
						switch v := n.(type) {
						case *ast.ImportSpec:
							if v.Name != nil && v.Name.Name == "_" {
								pos := pkg.Fset.Position(v.Pos())
								mu.Lock()
								result.Warnings = append(result.Warnings, ValidationWarning{
									Level:   "warning",
									Code:    "BLANK_IMPORT",
									Message: fmt.Sprintf("blank import: %s", v.Path.Value),
									File:    pos.Filename,
									Line:    pos.Line,
									Column:  pos.Column,
								})
								mu.Unlock()
							}
						case *ast.FuncDecl:
							if v.Body == nil {
								pos := pkg.Fset.Position(v.Pos())
								mu.Lock()
								result.Warnings = append(result.Warnings, ValidationWarning{
									Level:   "warning",
									Code:    "EMPTY_FUNC",
									Message: fmt.Sprintf("empty function: %s", v.Name.Name),
									File:    pos.Filename,
									Line:    pos.Line,
									Column:  pos.Column,
								})
								mu.Unlock()
							}
						}
						return true
					})
				}
			}

			// 严格验证：依赖检查 + lint
			if level == ValidationLevelStrict {
				// 检查未使用的导入
				for _, file := range pkg.Syntax {
					for _, imp := range file.Imports {
						path := strings.Trim(imp.Path.Value, `"`)
						used := false
						for _, use := range pkg.TypesInfo.Uses {
							if use.Pkg() != nil && use.Pkg().Path() == path {
								used = true
								break
							}
						}
						if !used {
							pos := pkg.Fset.Position(imp.Pos())
							mu.Lock()
							result.Warnings = append(result.Warnings, ValidationWarning{
								Level:   "warning",
								Code:    "UNUSED_IMPORT",
								Message: fmt.Sprintf("unused import: %s", path),
								File:    pos.Filename,
								Line:    pos.Line,
								Column:  pos.Column,
							})
							result.Stats.UnusedImports++
							mu.Unlock()
						}
					}
				}

				// 检查未使用的变量
				for ident, obj := range pkg.TypesInfo.Defs {
					if obj == nil {
						continue
					}
					if v, ok := obj.(*types.Var); ok && !v.IsField() {
						used := false
						for _, useObj := range pkg.TypesInfo.Uses {
							if useObj == obj {
								used = true
								break
							}
						}
						if !used {
							pos := pkg.Fset.Position(ident.Pos())
							mu.Lock()
							result.Warnings = append(result.Warnings, ValidationWarning{
								Level:   "warning",
								Code:    "UNUSED_VAR",
								Message: fmt.Sprintf("unused variable: %s", ident.Name),
								File:    pos.Filename,
								Line:    pos.Line,
								Column:  pos.Column,
							})
							result.Stats.UnusedVariables++
							mu.Unlock()
						}
					}
				}

				// 检查类型错误
				for _, err := range pkg.TypeErrors {
					pos := pkg.Fset.Position(err.Pos)
					mu.Lock()
					result.Errors = append(result.Errors, ValidationError{
						Level:   "error",
						Code:    "TYPE_ERROR",
						Message: err.Msg,
						File:    pos.Filename,
						Line:    pos.Line,
						Column:  pos.Column,
					})
					result.Stats.TypeErrors++
					result.Valid = false
					mu.Unlock()
				}

				// 检查语法错误
				for _, err := range pkg.Errors {
					if strings.Contains(err.Error(), "syntax error") {
						mu.Lock()
						result.Errors = append(result.Errors, ValidationError{
							Level:   "error",
							Code:    "SYNTAX_ERROR",
							Message: err.Error(),
							File:    pkg.PkgPath,
						})
						result.Stats.SyntaxErrors++
						result.Valid = false
						mu.Unlock()
					}
				}
			}

			mu.Lock()
			result.Stats.FilesChecked += len(pkg.CompiledGoFiles)
			mu.Unlock()
		}(pkg)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 更新统计信息
	mu.Lock()
	result.Stats.ErrorCount = len(result.Errors)
	result.Stats.WarningCount = len(result.Warnings)
	mu.Unlock()

	result.Duration = time.Since(result.AnalyzedAt).String()
	return result, nil
}

// ValidateDependencies 验证依赖关系
func (v *DefaultValidator) ValidateDependencies(ctx context.Context, pkgPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid: true,
		Stats: ValidationStats{},
	}

	cfg := &packages.Config{
		Mode: packages.NeedImports | packages.NeedDeps,
		Dir:  filepath.Join(v.baseDir, pkgPath),
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	for _, pkg := range pkgs {
		result.Stats.FilesChecked += len(pkg.GoFiles)

		for _, err := range pkg.Errors {
			result.Errors = append(result.Errors, ValidationError{
				Level:   "error",
				Code:    "DEP_ERROR",
				Message: err.Error(),
				File:    pkgPath,
			})
			result.Stats.DependencyErrors++
			result.Valid = false
		}
	}

	return result, nil
}

// CheckCircularDependencies 检查循环依赖
func (v *DefaultValidator) CheckCircularDependencies(ctx context.Context, pkgPath string) (*ValidationResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	result := &ValidationResult{
		Valid:      true,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// 配置包加载器
	cfg := &packages.Config{
		Mode: packages.NeedImports |
			packages.NeedDeps |
			packages.NeedName |
			packages.NeedModule |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Context: ctx,
		Dir:     v.baseDir,
		Tests:   true,
	}

	// 加载包
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found at %s", pkgPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		result.Valid = false
		for _, err := range pkg.Errors {
			result.Errors = append(result.Errors, ValidationError{
				Level:   "error",
				Code:    "PKG_ERROR",
				Message: err.Error(),
				File:    pkgPath,
			})
		}
		return result, nil
	}

	// 使用 sync.Mutex 来保护结果
	var mu sync.Mutex

	// 创建依赖图
	graph := make(map[string][]string)
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	// 递归构建依赖图
	var buildGraph func(p *packages.Package)
	buildGraph = func(p *packages.Package) {
		if p == nil {
			return
		}

		pkgID := p.PkgPath
		if visited[pkgID] {
			return
		}
		visited[pkgID] = true

		deps := make([]string, 0)
		for _, imp := range p.Imports {
			// 只考虑项目内的包
			if strings.HasPrefix(imp.PkgPath, pkg.Module.Path) {
				deps = append(deps, imp.PkgPath)
				buildGraph(imp)
			}
		}

		// 检查源文件中的导入
		for _, f := range p.Syntax {
			for _, imp := range f.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if strings.HasPrefix(path, pkg.Module.Path) {
					found := false
					for _, dep := range deps {
						if dep == path {
							found = true
							break
						}
					}
					if !found {
						deps = append(deps, path)
					}
				}
			}
		}

		graph[pkgID] = deps
	}

	buildGraph(pkg)

	// 重置访问标记
	visited = make(map[string]bool)
	stack = make(map[string]bool)

	// 检测循环依赖
	var detectCycle func(pkg string, path []string) bool
	detectCycle = func(pkg string, path []string) bool {
		if stack[pkg] {
			// 找到循环依赖
			cycle := append(path, pkg)
			mu.Lock()
			for i := 0; i < len(cycle)-1; i++ {
				result.CircularDeps = append(result.CircularDeps,
					fmt.Sprintf("%s -> %s", cycle[i], cycle[i+1]))
			}
			mu.Unlock()
			return true
		}

		if visited[pkg] {
			return false
		}

		visited[pkg] = true
		stack[pkg] = true

		for _, dep := range graph[pkg] {
			if detectCycle(dep, append(path, pkg)) {
				return true
			}
		}

		stack[pkg] = false
		return false
	}

	// 从当前包开始检测循环依赖
	if detectCycle(pkg.PkgPath, nil) {
		mu.Lock()
		result.HasCircularDeps = true
		result.Valid = false
		result.Stats.CircularDeps++
		result.Errors = append(result.Errors, ValidationError{
			Level:   "error",
			Code:    "CIRCULAR_DEP",
			Message: "Circular dependency detected",
			File:    pkgPath,
			Details: result.CircularDeps,
		})
		mu.Unlock()
	}

	// 收集导入信息
	for _, imp := range pkg.Imports {
		mu.Lock()
		result.Imports = append(result.Imports, imp.PkgPath)
		if !strings.HasPrefix(imp.PkgPath, ".") && !strings.HasPrefix(imp.PkgPath, "internal/") {
			if strings.Contains(imp.PkgPath, "golang.org/x/") || strings.Contains(imp.PkgPath, "github.com/") {
				result.HasExternalDeps = true
				result.ExternalDeps = append(result.ExternalDeps, imp.PkgPath)
			}
		}
		mu.Unlock()
	}

	// 检查源文件中的导入
	for _, f := range pkg.Syntax {
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			mu.Lock()
			result.Imports = append(result.Imports, path)
			if !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "internal/") {
				if strings.Contains(path, "golang.org/x/") || strings.Contains(path, "github.com/") {
					result.HasExternalDeps = true
					result.ExternalDeps = append(result.ExternalDeps, path)
				}
			}
			mu.Unlock()
		}
	}

	// 去重导入
	if len(result.Imports) > 0 {
		mu.Lock()
		seen := make(map[string]bool)
		unique := make([]string, 0, len(result.Imports))
		for _, imp := range result.Imports {
			if !seen[imp] {
				seen[imp] = true
				unique = append(unique, imp)
			}
		}
		result.Imports = unique
		mu.Unlock()
	}

	// 去重外部依赖
	if len(result.ExternalDeps) > 0 {
		mu.Lock()
		seen := make(map[string]bool)
		unique := make([]string, 0, len(result.ExternalDeps))
		for _, dep := range result.ExternalDeps {
			if !seen[dep] {
				seen[dep] = true
				unique = append(unique, dep)
			}
		}
		result.ExternalDeps = unique
		mu.Unlock()
	}

	// 更新统计信息
	mu.Lock()
	result.Stats.FilesChecked = len(pkg.CompiledGoFiles)
	result.Stats.ErrorCount = len(result.Errors)
	result.Stats.WarningCount = len(result.Warnings)
	mu.Unlock()

	result.Duration = time.Since(result.AnalyzedAt).String()
	return result, nil
}

// ValidateExternalDependencies 验证外部依赖
func (v *DefaultValidator) ValidateExternalDependencies(ctx context.Context, pkgPath string) (*ValidationResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	result := &ValidationResult{
		Valid:      true,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// 配置包加载器
	cfg := &packages.Config{
		Mode: packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Context: ctx,
		Dir:     v.baseDir,
		Tests:   true,
	}

	// 加载包
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found at %s", pkgPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		result.Valid = false
		for _, err := range pkg.Errors {
			result.Errors = append(result.Errors, ValidationError{
				Level:   "error",
				Code:    "PKG_ERROR",
				Message: err.Error(),
				File:    pkgPath,
			})
		}
		return result, nil
	}

	// 使用 sync.WaitGroup 和 sync.Mutex 来安全地收集信息
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 收集导入信息
	for _, imp := range pkg.Imports {
		result.Imports = append(result.Imports, imp.PkgPath)
		if !strings.HasPrefix(imp.PkgPath, ".") && !strings.HasPrefix(imp.PkgPath, "internal/") {
			if strings.Contains(imp.PkgPath, "golang.org/x/") || strings.Contains(imp.PkgPath, "github.com/") {
				result.HasExternalDeps = true
				result.ExternalDeps = append(result.ExternalDeps, imp.PkgPath)
			}
		}
	}

	// 递归检查依赖
	visited := make(map[string]bool)
	var checkDeps func(p *packages.Package)
	checkDeps = func(p *packages.Package) {
		if p == nil || visited[p.PkgPath] {
			return
		}
		visited[p.PkgPath] = true

		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp.PkgPath, ".") && !strings.HasPrefix(imp.PkgPath, "internal/") {
				if strings.Contains(imp.PkgPath, "golang.org/x/") || strings.Contains(imp.PkgPath, "github.com/") {
					mu.Lock()
					result.HasExternalDeps = true
					result.ExternalDeps = append(result.ExternalDeps, imp.PkgPath)
					mu.Unlock()
				}
			}
			wg.Add(1)
			go func(imp *packages.Package) {
				defer wg.Done()
				checkDeps(imp)
			}(imp)
		}

		// 检查源文件中的导入
		for _, f := range p.Syntax {
			for _, imp := range f.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "internal/") {
					if strings.Contains(path, "golang.org/x/") || strings.Contains(path, "github.com/") {
						mu.Lock()
						result.HasExternalDeps = true
						result.ExternalDeps = append(result.ExternalDeps, path)
						result.Imports = append(result.Imports, path)
						mu.Unlock()
					}
				}
			}
		}
	}

	checkDeps(pkg)

	// 等待所有 goroutine 完成
	wg.Wait()

	// 去重外部依赖
	if len(result.ExternalDeps) > 0 {
		mu.Lock()
		seen := make(map[string]bool)
		unique := make([]string, 0, len(result.ExternalDeps))
		for _, dep := range result.ExternalDeps {
			if !seen[dep] {
				seen[dep] = true
				unique = append(unique, dep)
			}
		}
		result.ExternalDeps = unique
		mu.Unlock()
	}

	// 去重导入
	if len(result.Imports) > 0 {
		mu.Lock()
		seen := make(map[string]bool)
		unique := make([]string, 0, len(result.Imports))
		for _, imp := range result.Imports {
			if !seen[imp] {
				seen[imp] = true
				unique = append(unique, imp)
			}
		}
		result.Imports = unique
		mu.Unlock()
	}

	// 更新统计信息
	mu.Lock()
	result.Stats.FilesChecked = len(pkg.CompiledGoFiles)
	result.Stats.ErrorCount = len(result.Errors)
	result.Stats.WarningCount = len(result.Warnings)
	mu.Unlock()

	result.Duration = time.Since(result.AnalyzedAt).String()
	return result, nil
}

// 辅助函数

func isGoFile(name string) bool {
	return filepath.Ext(name) == ".go"
}

func isHiddenDir(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
