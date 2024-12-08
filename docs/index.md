# ReadGo Documentation

A powerful Go code analysis tool that helps developers explore and understand Go codebases with ease.

## Features

- **Type Analysis**: Discover and analyze types, interfaces, and structs in Go code
- **Package Analysis**: Analyze package structure and dependencies
- **File Analysis**: Analyze individual Go source files
- **Third-party Support**: Analyze third-party packages and dependencies
- **Caching**: Efficient caching system for improved performance
- **Flexible Configuration**: Customizable options for different analysis needs

## Quick Start

```bash
go get github.com/iamlongalong/readgo@v0.2.0
```

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
    analyzer := readgo.NewAnalyzer(
        readgo.WithWorkDir("."),
        readgo.WithCacheTTL(5*time.Minute),
    )

    // Analyze the current project
    result, err := analyzer.AnalyzeProject(context.Background(), ".")
    if err != nil {
        log.Fatal(err)
    }

    // Print analysis results
    fmt.Printf("Project: %s\n", result.Name)
    fmt.Printf("Types found: %d\n", len(result.Types))
    fmt.Printf("Functions found: %d\n", len(result.Functions))
}
```

## Documentation

- [Installation Guide](getting-started/installation.md)
- [Quick Start Guide](getting-started/quick-start.md)
- [Basic Usage](user-guide/basic-usage.md)
- [Configuration](user-guide/configuration.md)
- [Examples](user-guide/examples.md)
- [Architecture](ARCHITECTURE.md)
- [API Reference](api/interfaces.md)

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details. 