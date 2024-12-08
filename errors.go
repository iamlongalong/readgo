package readgo

import (
	"fmt"
)

// Error types for better error handling and context
var (
	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = fmt.Errorf("not found")

	// ErrInvalidInput indicates invalid input parameters
	ErrInvalidInput = fmt.Errorf("invalid input")

	// ErrPermission indicates permission related errors
	ErrPermission = fmt.Errorf("permission denied")
)

// AnalysisError represents an error that occurred during code analysis
type AnalysisError struct {
	Op      string // Operation that failed (e.g., "load package", "parse file")
	Path    string // File or package path where the error occurred
	Wrapped error  // The underlying error
}

func (e *AnalysisError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("analysis error: %s: %v", e.Op, e.Wrapped)
	}
	return fmt.Sprintf("analysis error: %s: %s: %v", e.Op, e.Path, e.Wrapped)
}

func (e *AnalysisError) Unwrap() error {
	return e.Wrapped
}

// ValidationError represents an error that occurred during code validation
type ValidationError struct {
	File    string // File where the error occurred
	Line    int    // Line number where the error occurred
	Column  int    // Column number where the error occurred
	Message string // Error message
	Wrapped error  // The underlying error
}

func (e *ValidationError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("validation error: %s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
	}
	if e.File != "" {
		return fmt.Sprintf("validation error: %s: %s", e.File, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Wrapped
}

// TypeLookupError represents an error that occurred during type lookup
type TypeLookupError struct {
	TypeName string // Name of the type being looked up
	Package  string // Package where the lookup was performed
	Kind     string // Kind of type (e.g., "interface", "struct")
	Wrapped  error  // The underlying error
}

func (e *TypeLookupError) Error() string {
	if e.Kind != "" {
		return fmt.Sprintf("type lookup error: %s %s not found in package %s", e.Kind, e.TypeName, e.Package)
	}
	return fmt.Sprintf("type lookup error: %s not found in package %s", e.TypeName, e.Package)
}

func (e *TypeLookupError) Unwrap() error {
	return e.Wrapped
}

// PackageError represents an error that occurred during package operations
type PackageError struct {
	Package string   // Package path
	Op      string   // Operation that failed
	Errors  []string // List of errors
	Wrapped error    // The underlying error
}

func (e *PackageError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("package error: %s: %s: %v (and %d more errors)",
			e.Package, e.Op, e.Errors[0], len(e.Errors)-1)
	}
	return fmt.Sprintf("package error: %s: %s: %v", e.Package, e.Op, e.Wrapped)
}

func (e *PackageError) Unwrap() error {
	return e.Wrapped
}
