# ReadGo

[English](README.md) | [中文](README_zh.md)

ReadGo 是一个 Go 代码分析工具，帮助开发者理解和导航 Go 代码库。它提供了分析 Go 源代码的功能，包括类型信息、函数签名和包依赖关系的分析。

## 特性

- 项目级代码分析
- 类型信息提取
- 函数签名分析
- 包依赖跟踪
- 接口实现检测
- 代码结构可视化
- 缓存支持以提升性能

## 安装

### 前置条件

- Go 1.16 或更高版本
- Make（可选，用于使用 Makefile 命令）
- golangci-lint（用于代码检查）

### 工具安装

1. 克隆仓库：
```bash
git clone https://github.com/iamlongalong/readgo.git
cd readgo
```

2. 安装开发工具：
```bash
make install-tools
```

3. 构建项目：
```bash
make build
```

## 使用方法

### 基本命令

```go
// 初始化分析器
analyzer := readgo.NewAnalyzer()

// 分析文件
result, err := analyzer.AnalyzeFile(context.Background(), "main.go")

// 分析包
result, err := analyzer.AnalyzePackage(context.Background(), "mypackage")

// 分析整个项目
result, err := analyzer.AnalyzeProject(context.Background(), ".")

// 查找特定类型
typeInfo, err := analyzer.FindType(context.Background(), "mypackage", "MyType")

// 查找接口
interfaceInfo, err := analyzer.FindInterface(context.Background(), "mypackage", "MyInterface")
```

### 开发命令

项目包含了常用的开发命令（通过 Makefile）：

```bash
# 显示所有可用命令
make help

# 构建项目
make build

# 运行测试
make test

# 运行代码检查（格式化、vet、lint、test）
make check

# 运行提交前检查
make pre-commit

# 格式化代码
make fmt

# 清理构建产物
make clean
```

## 配置

### 分析器选项

```go
analyzer := readgo.NewAnalyzer(
    readgo.WithWorkDir("path/to/workspace"),
    readgo.WithCacheTTL(time.Minute * 5),
)
```

### 缓存配置

分析器包含缓存系统以提升性能：

- 默认 TTL：5 分钟
- 可以通过设置 TTL 为 0 来禁用缓存
- 可以通过 `GetCacheStats()` 获取缓存统计信息

## 项目结构

```
.
├── analyzer.go       # 主分析器实现
├── cache.go         # 缓存系统
├── common.go        # 通用工具
├── errors.go        # 错误定义
├── options.go       # 配置选项
├── reader.go        # 源码读取器
├── types.go         # 类型定义
└── validator.go     # 代码验证
```

## 贡献指南

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 运行测试和检查 (`make pre-commit`)
4. 提交更改 (`git commit -m '添加精彩特性'`)
5. 推送到分支 (`git push origin feature/amazing-feature`)
6. 创建 Pull Request

## 开发工作流程

1. 进行代码修改
2. 运行 `make fmt` 格式化代码
3. 运行 `make check` 验证更改
4. 运行 `make pre-commit` 进行提交前检查
5. 创建 pull request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 致谢

- Go 团队提供的优秀 `go/ast` 和 `go/types` 包
- 社区的反馈和贡献 
