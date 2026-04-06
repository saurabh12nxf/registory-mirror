# Contributing to Registry Mirror

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/registry-mirror`
3. Create a branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test ./...`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to the branch: `git push origin feature/your-feature-name`
8. Submit a pull request

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Keep functions small and focused
- Add comments for complex logic
- Use meaningful variable names

### Testing

- Write tests for new features
- Maintain test coverage above 70%
- Use table-driven tests where appropriate
- Include both positive and negative test cases

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb (Add, Fix, Update, etc.)
- Keep the first line under 50 characters
- Add detailed description if needed

Example:
```
Add signature verification for images

- Implement Cosign integration
- Add policy-based verification
- Include batch verification support
```

### Pull Request Process

1. Update documentation if needed
2. Add tests for new functionality
3. Ensure all tests pass
4. Update FEATURES.md if adding a new feature
5. Request review from maintainers

## Testing

### Run All Tests
```bash
go test ./...
```

### Run with Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Package
```bash
go test ./internal/storage/...
```

### Run Benchmarks
```bash
go test -bench=. -benchmem ./...
```

## Project Structure

```
registry-mirror/
├── cmd/              # CLI commands
├── internal/
│   ├── analytics/    # Usage statistics
│   ├── cache/        # Cache management
│   ├── git/          # Repository scanning
│   ├── mirror/       # Image syncing
│   ├── registry/     # Docker registry client
│   ├── security/     # Signature verification
│   ├── storage/      # Database operations
│   └── tui/          # Terminal UI
├── .github/
│   └── workflows/    # CI/CD pipelines
└── docs/             # Documentation
```

## Areas for Contribution

### High Priority
- Additional test coverage
- Performance optimizations
- Bug fixes
- Documentation improvements

### Medium Priority
- New registry support (Harbor, Artifactory, etc.)
- Additional TUI views
- Metrics export (Prometheus)
- Webhook support

### Low Priority
- Web dashboard
- Multi-registry support
- Image vulnerability scanning
- Custom plugins

## Code Review Process

1. Maintainers will review your PR within 48 hours
2. Address any feedback or requested changes
3. Once approved, maintainers will merge your PR
4. Your contribution will be included in the next release

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for questions or ideas
- Check existing issues before creating new ones

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Help others learn and grow

Thank you for contributing to Registry Mirror! 🎉
