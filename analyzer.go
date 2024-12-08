package readgo2

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
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
)

// DefaultAnalyzer 默认的代码分析器实现
type DefaultAnalyzer struct {
	baseDir string
}

// NewAnalyzer 创建新的代码分析器
func NewAnalyzer(baseDir string) CodeAnalyzer {
	return &DefaultAnalyzer{
		baseDir: baseDir,
	}
}

// AnalyzeProject 分析整个项目
func (a *DefaultAnalyzer) AnalyzeProject(ctx context.Context) (*AnalysisResult, error) {
	startTime := time.Now()
	result := &AnalysisResult{
		Name:       filepath.Base(a.baseDir),
		Path:       a.baseDir,
		Type:       "project",
		StartTime:  startTime.Format(time.RFC3339),
		AnalyzedAt: startTime,
	}

	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:     a.baseDir,
		Context: ctx,
		Tests:   true,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	// ���息
	for _, pkg := range pkgs {
		result.Stats.Files += len(pkg.GoFiles)

		for _, syntax := range pkg.Syntax {
			ast.Inspect(syntax, func(n ast.Node) bool {
				if n == nil {
					return true
				}
				switch n.(type) {
				case *ast.FuncDecl:
					result.Stats.Functions++
				case *ast.TypeSpec:
					result.Stats.Types++
				}
				return true
			})
		}
	}

	result.Duration = time.Since(startTime).String()
	return result, nil
}

// AnalyzePackage 分析特定包
func (a *DefaultAnalyzer) AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}
	if pkgPath == "" {
		return nil, fmt.Errorf("empty package path")
	}

	// 验证包路径
	if strings.HasPrefix(pkgPath, "/") || strings.HasPrefix(pkgPath, "..") {
		return nil, fmt.Errorf("invalid package path: %s", pkgPath)
	}

	result := &AnalysisResult{
		Name:       filepath.Base(pkgPath),
		Path:       pkgPath,
		Type:       "package",
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
		Valid:      true,
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
		Dir:     filepath.Join(a.baseDir, pkgPath),
		Tests:   true,
	}

	// 加载包
	pkgs, err := packages.Load(cfg, ".")
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
			result.Errors = append(result.Errors, fmt.Sprintf("package error: %v", err))
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

	// 收集类型信息
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			obj := scope.Lookup(name)
			switch v := obj.(type) {
			case *types.TypeName:
				typeInfo := TypeInfo{
					Name:    v.Name(),
					Package: pkg.PkgPath,
				}

				if named, ok := v.Type().(*types.Named); ok {
					if _, ok := named.Underlying().(*types.Struct); ok {
						typeInfo.Type = "struct"
						structType := named.Underlying().(*types.Struct)
						for i := 0; i < structType.NumFields(); i++ {
							field := structType.Field(i)
							fieldInfo := FieldInfo{
								Name: field.Name(),
								Type: field.Type().String(),
							}
							if tag := structType.Tag(i); tag != "" {
								fieldInfo.Tag = tag
							}
							typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
						}
					} else if _, ok := named.Underlying().(*types.Interface); ok {
						typeInfo.Type = "interface"
					}
				}

				mu.Lock()
				result.Types = append(result.Types, typeInfo)
				result.Stats.Types++
				mu.Unlock()
			}
		}(name)
	}

	// 收集���数信息
	for _, f := range pkg.Syntax {
		wg.Add(1)
		go func(f *ast.File) {
			defer wg.Done()
			ast.Inspect(f, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.FuncDecl:
					funcInfo := FunctionInfo{
						Name:    v.Name.Name,
						Package: pkg.PkgPath,
					}

					if v.Recv != nil && len(v.Recv.List) > 0 {
						if expr, ok := v.Recv.List[0].Type.(ast.Expr); ok {
							if tv, ok := pkg.TypesInfo.Types[expr]; ok {
								funcInfo.Receiver = tv.Type.String()
							}
						}
					}

					if v.Type.Params != nil {
						for _, param := range v.Type.Params.List {
							if expr, ok := param.Type.(ast.Expr); ok {
								if tv, ok := pkg.TypesInfo.Types[expr]; ok {
									funcInfo.Parameters = append(funcInfo.Parameters, tv.Type.String())
								}
							}
						}
					}

					if v.Type.Results != nil {
						for _, result := range v.Type.Results.List {
							if expr, ok := result.Type.(ast.Expr); ok {
								if tv, ok := pkg.TypesInfo.Types[expr]; ok {
									funcInfo.Results = append(funcInfo.Results, tv.Type.String())
								}
							}
						}
					}

					mu.Lock()
					result.Functions = append(result.Functions, funcInfo)
					result.Stats.Functions++
					mu.Unlock()
				}
				return true
			})
		}(f)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 更新文件计数
	mu.Lock()
	result.Stats.Files = len(pkg.CompiledGoFiles)
	mu.Unlock()

	result.Duration = time.Since(result.AnalyzedAt).String()

	return result, nil
}

// AnalyzeFile 分析特定文件
func (a *DefaultAnalyzer) AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}
	if filePath == "" {
		return nil, fmt.Errorf("empty file path")
	}

	result := &AnalysisResult{
		Name:       filepath.Base(filePath),
		Path:       filePath,
		Type:       "file",
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
		Valid:      true,
	}

	absPath := filepath.Join(a.baseDir, filePath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// 解析文件
	fset := token.NewFileSet()
	fileAST, err := parser.ParseFile(fset, absPath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
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
		Dir:     filepath.Dir(absPath),
		Tests:   true,
	}

	// 加载包
	pkgs, err := packages.Load(cfg, "file="+absPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("load package error: %v", err))
		return result, nil
	}

	if len(pkgs) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "no package found")
		return result, nil
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		result.Valid = false
		for _, err := range pkg.Errors {
			result.Errors = append(result.Errors, err.Error())
		}
		return result, nil
	}

	// 收集导入信息
	for _, imp := range fileAST.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		result.Imports = append(result.Imports, path)
		if !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "internal/") {
			if strings.Contains(path, "golang.org/x/") || strings.Contains(path, "github.com/") {
				result.HasExternalDeps = true
				result.ExternalDeps = append(result.ExternalDeps, path)
			}
		}
	}

	// 收集类型信息
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		switch v := obj.(type) {
		case *types.TypeName:
			typeInfo := TypeInfo{
				Name:    v.Name(),
				Package: pkg.PkgPath,
			}

			if named, ok := v.Type().(*types.Named); ok {
				if _, ok := named.Underlying().(*types.Struct); ok {
					typeInfo.Type = "struct"
					structType := named.Underlying().(*types.Struct)
					for i := 0; i < structType.NumFields(); i++ {
						field := structType.Field(i)
						fieldInfo := FieldInfo{
							Name: field.Name(),
							Type: field.Type().String(),
						}
						if tag := structType.Tag(i); tag != "" {
							fieldInfo.Tag = tag
						}
						typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
					}
				} else if _, ok := named.Underlying().(*types.Interface); ok {
					typeInfo.Type = "interface"
				}
			}

			result.Types = append(result.Types, typeInfo)
			result.Stats.Types++
		}
	}

	// 收集函数信息
	ast.Inspect(fileAST, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncDecl:
			funcInfo := FunctionInfo{
				Name:    v.Name.Name,
				Package: pkg.PkgPath,
			}

			if v.Recv != nil && len(v.Recv.List) > 0 {
				if expr, ok := v.Recv.List[0].Type.(ast.Expr); ok {
					if tv, ok := pkg.TypesInfo.Types[expr]; ok {
						funcInfo.Receiver = tv.Type.String()
					}
				}
			}

			if v.Type.Params != nil {
				for _, param := range v.Type.Params.List {
					if expr, ok := param.Type.(ast.Expr); ok {
						if tv, ok := pkg.TypesInfo.Types[expr]; ok {
							funcInfo.Parameters = append(funcInfo.Parameters, tv.Type.String())
						}
					}
				}
			}

			if v.Type.Results != nil {
				for _, result := range v.Type.Results.List {
					if expr, ok := result.Type.(ast.Expr); ok {
						if tv, ok := pkg.TypesInfo.Types[expr]; ok {
							funcInfo.Results = append(funcInfo.Results, tv.Type.String())
						}
					}
				}
			}

			result.Functions = append(result.Functions, funcInfo)
			result.Stats.Functions++
		}
		return true
	})

	// 收集变量声明
	ast.Inspect(fileAST, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.ValueSpec:
			for _, name := range v.Names {
				if obj := pkg.TypesInfo.ObjectOf(name); obj != nil {
					switch obj.(type) {
					case *types.Var:
						if v.Values != nil {
							for _, value := range v.Values {
								if tv, ok := pkg.TypesInfo.Types[value]; ok {
									if tv.IsValue() && tv.Value != nil {
										// 这是一个常量或变量声明
										if obj.(*types.Var).IsField() {
											continue // 跳过结构体字段
										}
										if tv.Value.String() != "<nil>" {
											// 这是一个有值的变量声明
											result.Stats.Types++
										}
									}
								}
							}
						}
					case *types.Const:
						result.Stats.Types++
					}
				}
			}
		}
		return true
	})

	// 收集全局变量和常量
	for _, decl := range fileAST.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						if obj := pkg.TypesInfo.ObjectOf(name); obj != nil {
							switch obj.(type) {
							case *types.Var:
								result.Stats.Types++
							case *types.Const:
								result.Stats.Types++
							}
						}
					}
				}
			}
		}
	}

	// 收集接口信息
	for _, decl := range fileAST.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if ifaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
						ifaceInfo := InterfaceInfo{
							Name:    typeSpec.Name.Name,
							Package: pkg.PkgPath,
						}

						for _, method := range ifaceType.Methods.List {
							if funcType, ok := method.Type.(*ast.FuncType); ok {
								methodInfo := MethodInfo{
									Name: method.Names[0].Name,
								}

								if funcType.Params != nil {
									for _, param := range funcType.Params.List {
										if expr, ok := param.Type.(ast.Expr); ok {
											if tv, ok := pkg.TypesInfo.Types[expr]; ok {
												methodInfo.Parameters = append(methodInfo.Parameters, tv.Type.String())
											}
										}
									}
								}

								if funcType.Results != nil {
									for _, result := range funcType.Results.List {
										if expr, ok := result.Type.(ast.Expr); ok {
											if tv, ok := pkg.TypesInfo.Types[expr]; ok {
												methodInfo.Results = append(methodInfo.Results, tv.Type.String())
											}
										}
									}
								}

								ifaceInfo.Methods = append(ifaceInfo.Methods, methodInfo)
							}
						}

						result.Interfaces = append(result.Interfaces, ifaceInfo)
					}
				}
			}
		}
	}

	result.Stats.Files++
	result.Duration = time.Since(result.AnalyzedAt).String()

	return result, nil
}

// FindType finds a specific type in the given package
func (a *DefaultAnalyzer) FindType(ctx context.Context, pkgPath, typeName string) (*TypeInfo, error) {
	pkg, err := a.loadPackage(ctx, pkgPath)
	if err != nil {
		return nil, err
	}

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil || obj.Type() == nil {
			continue
		}

		if named, ok := obj.Type().(*types.Named); ok {
			if named.Obj().Name() == typeName {
				return &TypeInfo{
					Name:    named.Obj().Name(),
					Package: named.Obj().Pkg().Path(),
					Type:    "type",
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("type %s not found in package %s", typeName, pkgPath)
}

// FindInterface finds a specific interface in the given package
func (a *DefaultAnalyzer) FindInterface(ctx context.Context, pkgPath, interfaceName string) (*TypeInfo, error) {
	pkg, err := a.loadPackage(ctx, pkgPath)
	if err != nil {
		return nil, err
	}

	for _, obj := range pkg.TypesInfo.Defs {
		if obj == nil {
			continue
		}

		if named, ok := obj.Type().(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Interface); ok && named.Obj().Name() == interfaceName {
				return &TypeInfo{
					Name:    named.Obj().Name(),
					Package: named.Obj().Pkg().Path(),
					Type:    "interface",
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("interface %s not found in package %s", interfaceName, pkgPath)
}

// FindFunction finds a specific function in the given package
func (a *DefaultAnalyzer) FindFunction(ctx context.Context, pkgPath, funcName string) (*TypeInfo, error) {
	pkg, err := a.loadPackage(ctx, pkgPath)
	if err != nil {
		return nil, err
	}

	for _, obj := range pkg.TypesInfo.Defs {
		if obj == nil {
			continue
		}

		if fn, ok := obj.(*types.Func); ok && fn.Name() == funcName {
			return &TypeInfo{
				Name:    fn.Name(),
				Package: fn.Pkg().Path(),
				Type:    "function",
			}, nil
		}
	}

	return nil, fmt.Errorf("function %s not found in package %s", funcName, pkgPath)
}

func (a *DefaultAnalyzer) loadPackage(ctx context.Context, pkgPath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:     a.baseDir,
		Context: ctx,
		Tests:   true,
	}

	pkgs, err := packages.Load(cfg, fmt.Sprintf("pattern=%s", filepath.Join(a.baseDir, pkgPath)))
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found at %s", pkgPath)
	}

	return pkgs[0], nil
}

// SummarizeFile 生成文件摘要
func (a *DefaultAnalyzer) SummarizeFile(ctx context.Context, filePath string) (*Summary, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(a.baseDir, filePath)
	}

	summary := &Summary{
		Name:      filepath.Base(absPath),
		Path:      absPath,
		Type:      "file",
		CreatedAt: time.Now(),
	}

	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:     filepath.Dir(absPath),
		Context: ctx,
		Tests:   true,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found for file %s", absPath)
	}

	pkg := pkgs[0]
	var fileAST *ast.File
	for _, f := range pkg.Syntax {
		pos := pkg.Fset.Position(f.Pos())
		if filepath.Clean(pos.Filename) == filepath.Clean(absPath) {
			fileAST = f
			break
		}
	}

	if fileAST == nil {
		return nil, fmt.Errorf("file AST not found: %s", absPath)
	}

	// 收集组件信息
	var components []string
	ast.Inspect(fileAST, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		switch node := n.(type) {
		case *ast.FuncDecl:
			summary.Stats.Functions++
			components = append(components, fmt.Sprintf("func %s", node.Name.Name))
		case *ast.TypeSpec:
			if _, ok := node.Type.(*ast.InterfaceType); ok {
				summary.Stats.Interfaces++
				components = append(components, fmt.Sprintf("interface %s", node.Name.Name))
			} else {
				summary.Stats.Types++
				components = append(components, fmt.Sprintf("type %s", node.Name.Name))
			}
		}
		return true
	})

	// 统计代码行数
	fileContent, err := os.ReadFile(absPath)
	if err == nil {
		summary.Stats.Lines = len(strings.Split(string(fileContent), "\n"))
	}

	// 收集入依赖
	for _, imp := range fileAST.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if path != "C" && !strings.HasPrefix(path, "builtin") {
			summary.Dependencies = append(summary.Dependencies, path)
		}
	}

	summary.Components = components

	// 生成描述性摘要
	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("This file contains %d functions, %d types and %d interfaces.\n",
		summary.Stats.Functions, summary.Stats.Types, summary.Stats.Interfaces))

	if len(components) > 0 {
		desc.WriteString("\nKey components:\n")
		for _, comp := range components {
			desc.WriteString("- " + comp + "\n")
		}
	}

	if len(summary.Dependencies) > 0 {
		desc.WriteString("\nMain dependencies:\n")
		for _, dep := range summary.Dependencies {
			desc.WriteString("- " + dep + "\n")
		}
	}

	summary.Description = desc.String()
	return summary, nil
}

// SummarizePackage 生成包摘要
func (a *DefaultAnalyzer) SummarizePackage(ctx context.Context, pkgPath string) (*Summary, error) {
	absPath := pkgPath
	if !filepath.IsAbs(pkgPath) {
		absPath = filepath.Join(a.baseDir, pkgPath)
	}

	summary := &Summary{
		Name:      filepath.Base(absPath),
		Path:      absPath,
		Type:      "package",
		CreatedAt: time.Now(),
	}

	cfg := &packages.Config{
		Mode:    packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
		Dir:     absPath,
		Context: ctx,
		Tests:   false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no package found at %s", absPath)
	}

	pkg := pkgs[0]
	var components []string

	// 统计总代码行数
	for _, goFile := range pkg.GoFiles {
		content, err := os.ReadFile(goFile)
		if err == nil {
			summary.Stats.Lines += len(strings.Split(string(content), "\n"))
		}
	}

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return true
			}

			switch node := n.(type) {
			case *ast.FuncDecl:
				summary.Stats.Functions++
				if node.Recv == nil { // 只收集包级函数
					components = append(components, fmt.Sprintf("func %s", node.Name.Name))
				}
			case *ast.TypeSpec:
				if _, ok := node.Type.(*ast.InterfaceType); ok {
					summary.Stats.Interfaces++
					components = append(components, fmt.Sprintf("interface %s", node.Name.Name))
				} else {
					summary.Stats.Types++
					components = append(components, fmt.Sprintf("type %s", node.Name.Name))
				}
			}
			return true
		})

		// 收集导入依赖
		fileAST := file // 创建一个新变量以避免循环变量问题
		for _, imp := range fileAST.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if path != "C" && !strings.HasPrefix(path, "builtin") {
				found := false
				for _, existing := range summary.Dependencies {
					if existing == path {
						found = true
						break
					}
				}
				if !found {
					summary.Dependencies = append(summary.Dependencies, path)
				}
			}
		}
	}

	summary.Components = components

	// 生成描述性摘要
	var desc strings.Builder
	desc.WriteString(fmt.Sprintf("Package %s contains %d files with total %d lines of code.\n",
		summary.Name, len(pkg.GoFiles), summary.Stats.Lines))
	desc.WriteString(fmt.Sprintf("It defines %d functions, %d types and %d interfaces.\n",
		summary.Stats.Functions, summary.Stats.Types, summary.Stats.Interfaces))

	if len(components) > 0 {
		desc.WriteString("\nKey components:\n")
		for _, comp := range components {
			desc.WriteString("- " + comp + "\n")
		}
	}

	if len(summary.Dependencies) > 0 {
		desc.WriteString("\nMain dependencies:\n")
		for _, dep := range summary.Dependencies {
			desc.WriteString("- " + dep + "\n")
		}
	}

	summary.Description = desc.String()
	return summary, nil
}
