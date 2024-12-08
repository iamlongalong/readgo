# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.0] - 2024-03-14

### Added
- Core code analysis functionality
  - Type and interface discovery
  - Package structure analysis
  - File-level analysis
  - Support for third-party package analysis
- Caching system
  - Thread-safe implementation with sync.Map
  - TTL-based cache invalidation
  - Separate caches for types, packages, and files
  - Cache statistics tracking
- Configuration system
  - Flexible options for analyzer behavior
  - Chainable option functions
  - Default configurations
- Error handling
  - Custom error types for different scenarios
  - Detailed error messages with context
  - Error wrapping support
- CI/CD Integration
  - GitHub Actions workflows
  - Automated testing and linting
  - Release automation with GoReleaser
- Documentation
  - Comprehensive README
  - Architecture documentation
  - API documentation with examples
- Example projects
  - Basic usage examples
  - Third-party package analysis
  - Standard library analysis
  - Self-analysis capabilities
- Testing infrastructure
  - Comprehensive test suite
  - Test helpers and utilities
  - Mock test data

### Changed
- Improved package path handling
- Enhanced error messages and debugging
- Optimized cache performance
- Refined API interfaces

### Security
- Safe file system operations
- Proper error handling
- Thread-safe implementations

## Notes
This is the initial release of the ReadGo project, providing a solid foundation for Go code analysis.
The project aims to help developers understand and navigate Go codebases more effectively. 