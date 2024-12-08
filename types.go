package readgo

import (
	"time"
)

// FileType represents the type of files to include in the analysis
type FileType string

const (
	// FileTypeAll includes all file types
	FileTypeAll FileType = "all"
	// FileTypeGo includes only Go files
	FileTypeGo FileType = "go"
	// FileTypeTest includes only test files
	FileTypeTest FileType = "test"
	// FileTypeGenerated includes only generated files
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

// FileTreeNode represents a node in the file tree
type FileTreeNode struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Type     string          `json:"type"` // "file" or "directory"
	Size     int64           `json:"size,omitempty"`
	ModTime  time.Time       `json:"mod_time,omitempty"`
	Children []*FileTreeNode `json:"children,omitempty"`
}

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
