# ReadGo2

ReadGo2 是一个强大的 Go 项目代码阅读和分析工具。它提供了一套完整的接口来帮助开发者更好地理解和分析 Go 项目的代码结构。

## 主要功能

- 源码读取：支持文件树遍历、源文件读取、包文件获取和文件搜索
- 代码验证：支持多个验证级别，包括语法检查、类型检查和依赖检查
- 代码分析：支持项目级、包级、结构体级、接口级和函数级的深度分析

## 核心接口

- `SourceReader`: 源码读取接口
- `Validator`: 代码验证接口
- `CodeAnalyzer`: 代码分析接口

## 使用示例

```go
package main

import (
    "context"
    "github.com/bytedance/readgo2"
)

func main() {
    // 创建源码读取器
    reader := readgo2.NewSourceReader("/path/to/project")
    
    // 获取文件树
    tree, err := reader.GetFileTree(context.Background(), ".", readgo2.TreeOptions{
        MaxDepth:  3,
        FileTypes: readgo2.FileTypeGo,
    })
    if err != nil {
        panic(err)
    }
    
    // TODO: 处理文件树
}
```

## 待实现功能

- [ ] Validator 接口实现
- [ ] CodeAnalyzer 接口实现
- [ ] 更多的文件类型支持
- [ ] 更好的性能优化
- [ ] 更多的分析特性

## 贡献

欢迎提交 Pull Request 或提出 Issue。

## 许可证

MIT License
