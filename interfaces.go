package readgo2

import "context"

// Validator 代码验证接口
type Validator interface {
	// ValidateFile 验证特定文件
	ValidateFile(ctx context.Context, filePath string, level ValidationLevel) (*ValidationResult, error)

	// ValidatePackage 验证特定包
	ValidatePackage(ctx context.Context, pkgPath string, level ValidationLevel) (*ValidationResult, error)

	// ValidateProject 验证整个项目
	ValidateProject(ctx context.Context, level ValidationLevel) (*ValidationResult, error)

	// ValidateDependencies 验证依赖关系
	ValidateDependencies(ctx context.Context, pkgPath string) (*ValidationResult, error)

	// CheckCircularDependencies 检查循环依赖
	CheckCircularDependencies(ctx context.Context, pkgPath string) (*ValidationResult, error)
}

// SourceReader 源码读取接口
type SourceReader interface {
	// GetFileTree 获取文件树
	GetFileTree(ctx context.Context, root string, opts TreeOptions) (*FileTreeNode, error)

	// ReadSourceFile 读取源文件
	ReadSourceFile(ctx context.Context, path string, opts ReadOptions) ([]byte, error)

	// GetPackageFiles 获取包中的文件
	GetPackageFiles(ctx context.Context, pkgPath string, opts TreeOptions) ([]*FileTreeNode, error)

	// SearchFiles 搜索文件
	SearchFiles(ctx context.Context, pattern string, opts TreeOptions) ([]*FileTreeNode, error)
}

// CodeAnalyzer 代码分析器接口
type CodeAnalyzer interface {
	// AnalyzeProject 分析整个项目
	AnalyzeProject(ctx context.Context) (*AnalysisResult, error)

	// AnalyzePackage 分析特定包
	AnalyzePackage(ctx context.Context, pkgPath string) (*AnalysisResult, error)

	// AnalyzeFile 分析特定文件
	AnalyzeFile(ctx context.Context, filePath string) (*AnalysisResult, error)

	// FindType 查找特定类型
	FindType(ctx context.Context, pkgPath, typeName string) (*TypeInfo, error)

	// FindInterface 查找特定接口
	FindInterface(ctx context.Context, pkgPath, interfaceName string) (*TypeInfo, error)

	// FindFunction 查找特定函数
	FindFunction(ctx context.Context, pkgPath, funcName string) (*TypeInfo, error)

	// SummarizeFile 生成文件摘要
	SummarizeFile(ctx context.Context, filePath string) (*Summary, error)

	// SummarizePackage 生成包摘要
	SummarizePackage(ctx context.Context, pkgPath string) (*Summary, error)
}
