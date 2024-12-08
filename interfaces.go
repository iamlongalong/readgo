package readgo

import "context"

// Validator defines the interface for validating Go code
type Validator interface {
	// ValidateFile validates a specific Go source file
	ValidateFile(ctx context.Context, filePath string) (*ValidationResult, error)

	// ValidatePackage validates a Go package
	ValidatePackage(ctx context.Context, pkgPath string) (*ValidationResult, error)

	// ValidateProject validates the entire project
	ValidateProject(ctx context.Context) (*ValidationResult, error)
}

// SourceReader defines the interface for reading Go source code
type SourceReader interface {
	// GetFileTree returns the file tree starting from the given root
	GetFileTree(ctx context.Context, root string, opts TreeOptions) (*FileTreeNode, error)

	// ReadSourceFile reads a source file with the specified options
	ReadSourceFile(ctx context.Context, path string, opts ReadOptions) ([]byte, error)

	// GetPackageFiles returns all files in a package
	GetPackageFiles(ctx context.Context, pkgPath string, opts TreeOptions) ([]*FileTreeNode, error)

	// SearchFiles searches for files matching the given pattern
	SearchFiles(ctx context.Context, pattern string, opts TreeOptions) ([]*FileTreeNode, error)
}

// CodeAnalyzer defines the interface for analyzing Go code
type CodeAnalyzer interface {
	// FindType finds a specific type in the given package
	FindType(ctx context.Context, pkgPath, typeName string) (*TypeInfo, error)

	// FindInterface finds a specific interface in the given package
	FindInterface(ctx context.Context, pkgPath, interfaceName string) (*TypeInfo, error)

	// FindFunction finds a specific function in the given package
	FindFunction(ctx context.Context, pkgPath, funcName string) (*TypeInfo, error)

	// AnalyzeFile analyzes a specific Go source file
	AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error)

	// AnalyzePackage analyzes a Go package
	AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error)

	// AnalyzeProject analyzes a Go project at the specified path
	AnalyzeProject(ctx context.Context, projectPath string) (*AnalysisResult, error)
}
