package readgo

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultValidator implements the code validator
type DefaultValidator struct {
	baseDir string
}

// NewValidator creates a new validator
func NewValidator(baseDir string) *DefaultValidator {
	return &DefaultValidator{baseDir: baseDir}
}

// ValidateFile validates a Go source file
func (v *DefaultValidator) ValidateFile(ctx context.Context, filePath string) (*ValidationResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("empty file path")
	}

	absPath := filepath.Join(v.baseDir, filePath)
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("file access error: %w", err)
	}

	result := &ValidationResult{
		Name:       filepath.Base(filePath),
		Path:       filePath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
	}

	// Check for basic syntax issues
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		// Check for unused imports
		if imp, ok := n.(*ast.ImportSpec); ok {
			if imp.Name != nil && imp.Name.Name == "_" {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Type:    "unused_import",
					Message: fmt.Sprintf("unused import: %s", imp.Path.Value),
					File:    filePath,
					Line:    fset.Position(imp.Pos()).Line,
					Column:  fset.Position(imp.Pos()).Column,
				})
			}
		}

		// Check for syntax errors
		if _, ok := n.(*ast.BadExpr); ok {
			result.Errors = append(result.Errors, fmt.Sprintf("syntax error at %v", fset.Position(n.Pos())))
		}
		if _, ok := n.(*ast.BadStmt); ok {
			result.Errors = append(result.Errors, fmt.Sprintf("syntax error at %v", fset.Position(n.Pos())))
		}
		if _, ok := n.(*ast.BadDecl); ok {
			result.Errors = append(result.Errors, fmt.Sprintf("syntax error at %v", fset.Position(n.Pos())))
		}

		return true
	})

	return result, nil
}

// ValidatePackage validates a Go package
func (v *DefaultValidator) ValidatePackage(ctx context.Context, pkgPath string) (*ValidationResult, error) {
	if pkgPath == "" {
		return nil, fmt.Errorf("empty package path")
	}

	absPath := filepath.Join(v.baseDir, pkgPath)
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("package access error: %w", err)
	}

	result := &ValidationResult{
		Name:       filepath.Base(pkgPath),
		Path:       pkgPath,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Parse package files
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
	}

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			// Check for basic syntax issues
			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					return true
				}

				// Check for unused imports
				if imp, ok := n.(*ast.ImportSpec); ok {
					if imp.Name != nil && imp.Name.Name == "_" {
						result.Warnings = append(result.Warnings, ValidationWarning{
							Type:    "unused_import",
							Message: fmt.Sprintf("unused import: %s", imp.Path.Value),
							File:    fileName,
							Line:    fset.Position(imp.Pos()).Line,
							Column:  fset.Position(imp.Pos()).Column,
						})
					}
				}

				// Check for syntax errors
				if _, ok := n.(*ast.BadExpr); ok {
					result.Errors = append(result.Errors, fmt.Sprintf("syntax error at %v", fset.Position(n.Pos())))
				}
				if _, ok := n.(*ast.BadStmt); ok {
					result.Errors = append(result.Errors, fmt.Sprintf("syntax error at %v", fset.Position(n.Pos())))
				}
				if _, ok := n.(*ast.BadDecl); ok {
					result.Errors = append(result.Errors, fmt.Sprintf("syntax error at %v", fset.Position(n.Pos())))
				}

				return true
			})
		}
	}

	return result, nil
}

// ValidateProject validates the entire project
func (v *DefaultValidator) ValidateProject(ctx context.Context) (*ValidationResult, error) {
	result := &ValidationResult{
		Name:       filepath.Base(v.baseDir),
		Path:       v.baseDir,
		StartTime:  time.Now().Format(time.RFC3339),
		AnalyzedAt: time.Now(),
	}

	// Walk through all Go files in the project
	err := filepath.Walk(v.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			relPath, err := filepath.Rel(v.baseDir, path)
			if err != nil {
				return err
			}

			fileResult, err := v.ValidateFile(ctx, relPath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("error validating %s: %v", relPath, err))
				return nil
			}

			result.Errors = append(result.Errors, fileResult.Errors...)
			result.Warnings = append(result.Warnings, fileResult.Warnings...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("project validation error: %w", err)
	}

	return result, nil
}
