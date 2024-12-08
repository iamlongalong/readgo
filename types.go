package readgo2

import (
	"time"
)

// FileType 文件类型
type FileType string

const (
	FileTypeGo        FileType = "go"        // Go源文件
	FileTypeTest      FileType = "test"      // 测试文件
	FileTypeGenerated FileType = "generated" // 生成的文件
	FileTypeAll       FileType = "all"       // 所有文件
)

// ValidationLevel 验证级别
type ValidationLevel string

const (
	ValidationLevelBasic    ValidationLevel = "basic"    // 基础验证：语法检查
	ValidationLevelStandard ValidationLevel = "standard" // 标准验证：语法+类型检查
	ValidationLevelStrict   ValidationLevel = "strict"   // 严格验证：语法+类型+依赖+lint
)

// ValidationResult 验证结果
type ValidationResult struct {
	Valid      bool      `json:"valid"`
	StartTime  string    `json:"start_time"`
	Duration   string    `json:"duration"`
	AnalyzedAt time.Time `json:"analyzed_at"`

	// 错误信息
	Errors []ValidationError `json:"errors,omitempty"`

	// 警告信息
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// 验证统计
	Stats ValidationStats `json:"stats"`

	// 外部依赖
	HasExternalDeps bool     `json:"has_external_deps"`
	ExternalDeps    []string `json:"external_deps,omitempty"`

	// 循环依赖
	HasCircularDeps bool     `json:"has_circular_deps"`
	CircularDeps    []string `json:"circular_deps,omitempty"`

	// 导入信息
	Imports []string `json:"imports,omitempty"`
}

// HasErrors 检查是否有错误
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ValidationError 验证错误
type ValidationError struct {
	Level   string   `json:"level"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	File    string   `json:"file,omitempty"`
	Line    int      `json:"line,omitempty"`
	Column  int      `json:"column,omitempty"`
	Details []string `json:"details,omitempty"`
}

// ValidationWarning 验证警告
type ValidationWarning struct {
	Level   string   `json:"level"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	File    string   `json:"file,omitempty"`
	Line    int      `json:"line,omitempty"`
	Column  int      `json:"column,omitempty"`
	Details []string `json:"details,omitempty"`
}

// ValidationStats 验证统计
type ValidationStats struct {
	FilesChecked      int `json:"files_checked"`
	Files             int `json:"files"`
	Functions         int `json:"functions"`
	Types             int `json:"types"`
	ErrorCount        int `json:"error_count"`
	WarningCount      int `json:"warning_count"`
	SyntaxErrors      int `json:"syntax_errors"`
	TypeErrors        int `json:"type_errors"`
	DependencyErrors  int `json:"dependency_errors"`
	CircularDeps      int `json:"circular_deps"`
	UnusedImports     int `json:"unused_imports"`
	UnusedVariables   int `json:"unused_variables"`
	UnusedFunctions   int `json:"unused_functions"`
	UnusedTypes       int `json:"unused_types"`
	UnusedInterfaces  int `json:"unused_interfaces"`
	UnusedStructs     int `json:"unused_structs"`
	UnusedConstants   int `json:"unused_constants"`
	UnusedPackages    int `json:"unused_packages"`
	UnusedFiles       int `json:"unused_files"`
	UnusedDirectories int `json:"unused_directories"`
}

// TreeOptions 文件树选项
type TreeOptions struct {
	MaxDepth        int      `json:"max_depth"`        // 最大深度，0 表示不限制
	MaxFiles        int      `json:"max_files"`        // 最大文件数，0 表示不限制
	FileTypes       FileType `json:"file_types"`       // 文件类型过滤
	ExcludePatterns []string `json:"exclude_patterns"` // 排除的文件/目录模式
	IncludePatterns []string `json:"include_patterns"` // 包含的文件/目录模式
	FollowSymlinks  bool     `json:"follow_symlinks"`  // 是否跟随符号链接
}

// AnalysisDepth 分析深度
type AnalysisDepth string

const (
	AnalysisDepthBasic    AnalysisDepth = "basic"    // 基础分析
	AnalysisDepthStandard AnalysisDepth = "standard" // 标准分析
	AnalysisDepthDeep     AnalysisDepth = "deep"     // 深度分析
)

// AnalyzeOptions 分析选项
type AnalyzeOptions struct {
	// 基础控制
	Depth          AnalysisDepth `json:"depth"`
	MaxConcurrency int           `json:"max_concurrency"` // 最大并发数
	Timeout        time.Duration `json:"timeout"`         // 分析超时时间

	// 范围控制
	Scope struct {
		IncludeTests    bool     `json:"include_tests"`
		IncludeInternal bool     `json:"include_internal"`
		IncludeVendor   bool     `json:"include_vendor"`
		MaxDepth        int      `json:"max_depth"`
		FileTypes       FileType `json:"file_types"`
		ExcludePatterns []string `json:"exclude_patterns"`
		IncludePatterns []string `json:"include_patterns"`
	} `json:"scope"`

	// 内容控制
	Content struct {
		IncludeComments bool `json:"include_comments"`
		IncludeExamples bool `json:"include_examples"`
		IncludeMethods  bool `json:"include_methods"`
		IncludeFields   bool `json:"include_fields"`
		IncludePosition bool `json:"include_position"`
	} `json:"content"`

	// 依赖分析
	Dependencies struct {
		AnalyzeImports  bool `json:"analyze_imports"`
		AnalyzeExports  bool `json:"analyze_exports"`
		MaxDepth        int  `json:"max_depth"`
		IncludeIndirect bool `json:"include_indirect"`
	} `json:"dependencies"`
}

// ReadOptions 读取选项
type ReadOptions struct {
	IncludeComments bool `json:"include_comments"`
	StripSpaces     bool `json:"strip_spaces"`
}

// FileTreeNode 文件树节点
type FileTreeNode struct {
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Type     string          `json:"type"` // file/directory
	Size     int64           `json:"size,omitempty"`
	ModTime  time.Time       `json:"mod_time"`
	Children []*FileTreeNode `json:"children,omitempty"`
}

// TypeInfo 类型信息
type TypeInfo struct {
	Name    string      `json:"name"`
	Package string      `json:"package"`
	Type    string      `json:"type"` // type, interface, function
	Fields  []FieldInfo `json:"fields,omitempty"`
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Tag  string `json:"tag,omitempty"`
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	StartTime  string    `json:"start_time"`
	Duration   string    `json:"duration"`
	AnalyzedAt time.Time `json:"analyzed_at"`
	Valid      bool      `json:"valid"`

	// 错误信息
	Errors []string `json:"errors,omitempty"`

	// 警告信息
	Warnings []ValidationWarning `json:"warnings,omitempty"`

	// 验证统计
	Stats ValidationStats `json:"stats"`

	// 外部依赖
	HasExternalDeps bool     `json:"has_external_deps"`
	ExternalDeps    []string `json:"external_deps,omitempty"`

	// 循环依赖
	HasCircularDeps bool     `json:"has_circular_deps"`
	CircularDeps    []string `json:"circular_deps,omitempty"`

	// 导入信息
	Imports []string `json:"imports,omitempty"`

	// 类型信息
	Types      []TypeInfo      `json:"types,omitempty"`
	Interfaces []InterfaceInfo `json:"interfaces,omitempty"`
	Functions  []FunctionInfo  `json:"functions,omitempty"`
}

// HasErrors 检查是否有错误
func (r *AnalysisResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name    string         `json:"name"`
	Package string         `json:"package"`
	Methods []MethodInfo   `json:"methods,omitempty"`
	Embeds  []InterfaceRef `json:"embeds,omitempty"`
}

// MethodInfo 方法信息
type MethodInfo struct {
	Name       string   `json:"name"`
	Parameters []string `json:"parameters,omitempty"`
	Results    []string `json:"results,omitempty"`
}

// InterfaceRef 接口引用
type InterfaceRef struct {
	Name    string `json:"name"`
	Package string `json:"package"`
}

// FunctionInfo 函数信息
type FunctionInfo struct {
	Name       string   `json:"name"`
	Package    string   `json:"package"`
	Parameters []string `json:"parameters,omitempty"`
	Results    []string `json:"results,omitempty"`
	Receiver   string   `json:"receiver,omitempty"`
}

// Summary 代码摘要信息
type Summary struct {
	// 基本信息
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Type      string    `json:"type"` // file or package
	CreatedAt time.Time `json:"created_at"`

	// 统计信息
	Stats struct {
		Functions  int `json:"functions,omitempty"`
		Types      int `json:"types,omitempty"`
		Interfaces int `json:"interfaces,omitempty"`
		Lines      int `json:"lines,omitempty"`
	} `json:"stats"`

	// 核心组件列表
	Components []string `json:"components,omitempty"`

	// 依赖信息
	Dependencies []string `json:"dependencies,omitempty"`

	// 文本摘要
	Description string `json:"description"`
}
