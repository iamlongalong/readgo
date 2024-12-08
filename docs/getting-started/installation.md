# Installation

ReadGo is a Go package that can be installed using the standard Go toolchain.

## Requirements

- Go 1.22 or later
- Git (for installation from source)

## Installation Methods

### Using go get

The simplest way to install ReadGo is using `go get`:

```bash
go get github.com/iamlongalong/readgo@v0.2.0
```

### From Source

To install from source:

```bash
git clone https://github.com/iamlongalong/readgo.git
cd readgo
go install
```

## Verifying Installation

To verify that ReadGo is installed correctly, create a simple test program:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/iamlongalong/readgo"
)

func main() {
    analyzer := readgo.NewAnalyzer()
    result, err := analyzer.AnalyzeProject(context.Background(), ".")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Successfully analyzed project: %s\n", result.Name)
}
```

Save this as `test.go` and run:

```bash
go mod init test
go mod tidy
go run test.go
```

If you see output about the project analysis, ReadGo is installed correctly.

## Next Steps

- Read the [Quick Start Guide](quick-start.md) to learn basic usage
- Check out the [Examples](../user-guide/examples.md) for more complex use cases
- Learn about [Configuration Options](../user-guide/configuration.md) 