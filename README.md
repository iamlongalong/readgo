# ReadGo

[English](README.md) | [中文](README_zh.md)

ReadGo is a Go code analysis tool that helps developers understand and navigate Go codebases. It provides functionality for analyzing Go source code, including type information, function signatures, and package dependencies.

## Features

- Project-wide code analysis
- Type information extraction
- Function signature analysis
- Package dependency tracking
- Interface implementation detection
- Code structure visualization
- Cache support for better performance

## Installation

### Prerequisites

- Go 1.16 or later
- Make (optional, for using Makefile commands)
- golangci-lint (for code linting)

### Installing the Tool

1. Clone the repository:
```bash
git clone https://github.com/iamlongalong/readgo.git
cd readgo
```

2. Install development tools:
```bash
make install-tools
```

3. Build the project:
```bash
make build
```

## Usage

### Basic Commands

```go
// Initialize an analyzer
analyzer := readgo.NewAnalyzer()

// Analyze a file
result, err := analyzer.AnalyzeFile(context.Background(), "main.go")

// Analyze a package
result, err := analyzer.AnalyzePackage(context.Background(), "mypackage")

// Analyze an entire project
result, err := analyzer.AnalyzeProject(context.Background(), ".")

// Find a specific type
typeInfo, err := analyzer.FindType(context.Background(), "mypackage", "MyType")

// Find an interface
interfaceInfo, err := analyzer.FindInterface(context.Background(), "mypackage", "MyInterface")
```

### Development Commands

The project includes a Makefile with common development commands:

```bash
# Show all available commands
make help

# Build the project
make build

# Run tests
make test

# Run code checks (format, vet, lint, test)
make check

# Run pre-commit checks
make pre-commit

# Format code
make fmt

# Clean build artifacts
make clean
```

## Configuration

### Analyzer Options

```go
analyzer := readgo.NewAnalyzer(
    readgo.WithWorkDir("path/to/workspace"),
    readgo.WithCacheTTL(time.Minute * 5),
)
```

### Cache Configuration

The analyzer includes a caching system to improve performance:

- Default TTL: 5 minutes
- Cache can be disabled by setting TTL to 0
- Cache statistics available via `GetCacheStats()`

## Project Structure

```
.
├── analyzer.go       # Main analyzer implementation
├── cache.go         # Caching system
├── common.go        # Common utilities
├── errors.go        # Error definitions
├── options.go       # Configuration options
├── reader.go        # Source code reader
├── types.go         # Type definitions
└── validator.go     # Code validation
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and checks (`make pre-commit`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## Development Workflow

1. Make your changes
2. Run `make fmt` to format code
3. Run `make check` to verify changes
4. Run `make pre-commit` before committing
5. Create pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- The Go team for the excellent `go/ast` and `go/types` packages
- The community for feedback and contributions
