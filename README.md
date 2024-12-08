# readgo

A powerful Go code analysis and validation library that helps you understand and validate Go code.

## Features

- Analyze Go source code structure (packages, types, functions)
- Support for analyzing third-party packages
- Code validation with detailed error reporting
- File tree traversal and search
- Rich API for code introspection
- Comprehensive error handling and validation
- Support for custom analysis options

## Installation

```bash
go get github.com/iamlongalong/readgo
```

## Quick Start

Here's a simple example of analyzing a Go project:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/iamlongalong/readgo"
)

func main() {
    // Create a new analyzer
    analyzer := readgo.NewAnalyzer(".")

    // Analyze the current project
    result, err := analyzer.AnalyzeProject(context.Background(), ".")
    if err != nil {
        log.Fatalf("Failed to analyze project: %v", err)
    }

    // Print project information
    fmt.Printf("Project: %s\n", result.Name)
    fmt.Printf("Path: %s\n", result.Path)
    fmt.Printf("Analyzed at: %s\n", result.AnalyzedAt)

    // Print types
    fmt.Println("\nTypes:")
    for _, t := range result.Types {
        fmt.Printf("  - %s.%s: %s\n", t.Package, t.Name, t.Type)
    }

    // Print functions
    fmt.Println("\nFunctions:")
    for _, f := range result.Functions {
        fmt.Printf("  - %s.%s\n", f.Package, f.Name)
    }

    // Print imports
    fmt.Println("\nImports:")
    for _, imp := range result.Imports {
        fmt.Printf("  - %s\n", imp)
    }
}
```

## Usage

### Analyzing Code

The library provides several ways to analyze Go code:

1. Analyze a single file:
```go
result, err := analyzer.AnalyzeFile(ctx, "main.go")
if err != nil {
    log.Fatalf("Failed to analyze file: %v", err)
}
```

2. Analyze a package:
```go
result, err := analyzer.AnalyzePackage(ctx, "mypackage")
if err != nil {
    log.Fatalf("Failed to analyze package: %v", err)
}
```

3. Analyze an entire project:
```go
result, err := analyzer.AnalyzeProject(ctx, ".")
if err != nil {
    log.Fatalf("Failed to analyze project: %v", err)
}
```

4. Find specific types or interfaces:
```go
// Find a type
typeInfo, err := analyzer.FindType(ctx, "mypackage", "MyType")
if err != nil {
    log.Fatalf("Failed to find type: %v", err)
}

// Find an interface
interfaceInfo, err := analyzer.FindInterface(ctx, "io", "Reader")
if err != nil {
    log.Fatalf("Failed to find interface: %v", err)
}
```

### Reading Source Code

The library provides a powerful source code reader:

1. Get file tree:
```go
opts := readgo.TreeOptions{
    FileTypes: readgo.FileTypeGo,
    ExcludePatterns: []string{"vendor/*", "*.test.go"},
}
tree, err := reader.GetFileTree(ctx, ".", opts)
```

2. Read source file:
```go
opts := readgo.ReadOptions{
    IncludeComments: true,
    StripSpaces: false,
}
content, err := reader.ReadSourceFile(ctx, "main.go", opts)
```

3. Get package files:
```go
files, err := reader.GetPackageFiles(ctx, "mypackage", opts)
```

4. Search files:
```go
files, err := reader.SearchFiles(ctx, "*.go", opts)
```

### Validating Code

The library provides code validation capabilities:

1. Validate a single file:
```go
result, err := validator.ValidateFile(ctx, "main.go")
if err != nil {
    log.Fatalf("Failed to validate file: %v", err)
}
// Check validation results
if len(result.Errors) > 0 {
    fmt.Println("Validation errors:")
    for _, err := range result.Errors {
        fmt.Printf("  - %s\n", err)
    }
}
```

2. Validate a package:
```go
result, err := validator.ValidatePackage(ctx, "mypackage")
```

3. Validate an entire project:
```go
result, err := validator.ValidateProject(ctx)
```

### Working with Third-Party Packages

The library supports analyzing third-party packages:

```go
// Analyze a standard library package
result, err := analyzer.AnalyzePackage(ctx, "net/http")

// Find an interface in a third-party package
interfaceInfo, err := analyzer.FindInterface(ctx, "github.com/pkg/errors", "Wrapper")
```

## Examples

Check out the `examples` directory for more detailed examples:

- `examples/basic`: Basic usage of the analyzer
- `examples/analyze_stdlib`: Analyzing standard library packages
- `examples/validator`: Using the code validator

## API Documentation

### Core Interfaces

#### CodeAnalyzer Interface

The main interface for code analysis:

```go
type CodeAnalyzer interface {
    FindType(ctx context.Context, pkgPath, typeName string) (*TypeInfo, error)
    FindInterface(ctx context.Context, pkgPath, interfaceName string) (*TypeInfo, error)
    FindFunction(ctx context.Context, pkgPath, funcName string) (*TypeInfo, error)
    AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error)
    AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error)
    AnalyzeProject(ctx context.Context, projectPath string) (*AnalysisResult, error)
}
```

#### SourceReader Interface

Interface for reading and traversing source code:

```go
type SourceReader interface {
    GetFileTree(ctx context.Context, root string, opts TreeOptions) (*FileTreeNode, error)
    ReadSourceFile(ctx context.Context, path string, opts ReadOptions) ([]byte, error)
    GetPackageFiles(ctx context.Context, pkgPath string, opts TreeOptions) ([]*FileTreeNode, error)
    SearchFiles(ctx context.Context, pattern string, opts TreeOptions) ([]*FileTreeNode, error)
}
```

#### Validator Interface

Interface for code validation:

```go
type Validator interface {
    ValidateFile(ctx context.Context, filePath string) (*ValidationResult, error)
    ValidatePackage(ctx context.Context, pkgPath string) (*ValidationResult, error)
    ValidateProject(ctx context.Context) (*ValidationResult, error)
}
```

### Data Types

#### Analysis Types

```go
// TypeInfo represents information about a Go type
type TypeInfo struct {
    Name       string `json:"name"`
    Package    string `json:"package"`
    Type       string `json:"type"`
    IsExported bool   `json:"is_exported"`
}

// FunctionInfo represents information about a Go function
type FunctionInfo struct {
    Name       string `json:"name"`
    Package    string `json:"package"`
    IsExported bool   `json:"is_exported"`
}

// AnalysisResult represents the result of code analysis
type AnalysisResult struct {
    Name       string         `json:"name"`
    Path       string         `json:"path"`
    StartTime  string         `json:"start_time"`
    AnalyzedAt time.Time      `json:"analyzed_at"`
    Types      []TypeInfo     `json:"types,omitempty"`
    Functions  []FunctionInfo `json:"functions,omitempty"`
    Imports    []string       `json:"imports,omitempty"`
}
```

#### Validation Types

```go
// ValidationWarning represents a warning during validation
type ValidationWarning struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    File    string `json:"file,omitempty"`
    Line    int    `json:"line,omitempty"`
    Column  int    `json:"column,omitempty"`
}

// ValidationResult represents the result of code validation
type ValidationResult struct {
    Name       string              `json:"name"`
    Path       string              `json:"path"`
    StartTime  string              `json:"start_time"`
    AnalyzedAt time.Time           `json:"analyzed_at"`
    Errors     []string            `json:"errors,omitempty"`
    Warnings   []ValidationWarning `json:"warnings,omitempty"`
}
```

#### Configuration Types

```go
// FileType represents the type of files to include in the analysis
type FileType string

const (
    FileTypeAll       FileType = "all"
    FileTypeGo       FileType = "go"
    FileTypeTest     FileType = "test"
    FileTypeGenerated FileType = "generated"
)

// TreeOptions represents options for file tree operations
type TreeOptions struct {
    FileTypes       FileType `json:"file_types"`
    ExcludePatterns []string `json:"exclude_patterns,omitempty"`
    IncludePatterns []string `json:"include_patterns,omitempty"`
}

// ReadOptions represents options for reading source files
type ReadOptions struct {
    IncludeComments bool `json:"include_comments"`
    StripSpaces     bool `json:"strip_spaces"`
}
```

## Error Handling

The library uses standard Go error handling patterns. All errors are wrapped with meaningful context using `fmt.Errorf` and the `%w` verb. This allows you to use `errors.Is` and `errors.As` for error checking:

```go
result, err := analyzer.AnalyzeFile(ctx, "main.go")
if err != nil {
    if os.IsNotExist(err) {
        log.Fatal("File does not exist")
    }
    if errors.Is(err, ErrInvalidSyntax) {
        log.Fatal("Invalid Go syntax")
    }
    log.Fatalf("Unknown error: %v", err)
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

Before submitting a pull request:
1. Make sure all tests pass: `go test ./...`
2. Run `go vet` and `golint`
3. Format your code: `go fmt ./...`
4. Add tests for new functionality
5. Update documentation as needed

## License

This project is licensed under the MIT License - see the LICENSE file for details.
