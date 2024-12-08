# Quick Start Guide

This guide will help you get started with ReadGo by walking through some common use cases.

## Basic Project Analysis

Here's a simple example that analyzes a Go project:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/iamlongalong/readgo"
)

func main() {
    // Create a new analyzer with default options
    analyzer := readgo.NewAnalyzer(
        readgo.WithWorkDir("."),
        readgo.WithCacheTTL(5*time.Minute),
    )

    // Analyze the current project
    result, err := analyzer.AnalyzeProject(context.Background(), ".")
    if err != nil {
        log.Fatal(err)
    }

    // Print project information
    fmt.Printf("Project: %s\n", result.Name)
    fmt.Printf("Path: %s\n", result.Path)
    fmt.Printf("Analyzed at: %s\n\n", result.AnalyzedAt)

    // Print types
    fmt.Println("Types:")
    for _, t := range result.Types {
        fmt.Printf("  - %s.%s: %s\n", t.Package, t.Name, t.Type)
    }

    // Print functions
    fmt.Println("\nFunctions:")
    for _, f := range result.Functions {
        fmt.Printf("  - %s.%s\n", f.Package, f.Name)
    }
}
```

## Finding Specific Types

To find a specific type in a package:

```go
typeInfo, err := analyzer.FindType(context.Background(), "path/to/package", "TypeName")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found type: %s.%s (%s)\n", typeInfo.Package, typeInfo.Name, typeInfo.Type)
```

## Analyzing Third-party Packages

ReadGo can analyze third-party packages:

```go
result, err := analyzer.AnalyzePackage(context.Background(), "github.com/stretchr/testify/assert")
if err != nil {
    log.Fatal(err)
}

// Print exported types
for _, t := range result.Types {
    if t.IsExported {
        fmt.Printf("Type: %s.%s\n", t.Package, t.Name)
    }
}
```

## Configuration Options

ReadGo provides several configuration options:

```go
analyzer := readgo.NewAnalyzer(
    // Set working directory
    readgo.WithWorkDir("."),

    // Enable caching with TTL
    readgo.WithCacheTTL(5*time.Minute),

    // Enable concurrent analysis
    readgo.WithConcurrentAnalysis(true),

    // Set analysis timeout
    readgo.WithAnalysisTimeout(30*time.Second),
)
```

## Next Steps

- Check out the [Basic Usage Guide](../user-guide/basic-usage.md) for more details
- Learn about [Configuration Options](../user-guide/configuration.md)
- See [Examples](../user-guide/examples.md) for more use cases
- Read the [Architecture Documentation](../ARCHITECTURE.md) to understand the internals 