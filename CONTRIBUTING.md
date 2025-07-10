# Contributing to Harbinger

Thank you for your interest in contributing to Harbinger! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to abide by our code of conduct: be respectful, inclusive, and constructive in all interactions.

## How to Contribute

### Reporting Issues

- Check if the issue already exists in the [issue tracker](https://github.com/javanhut/harbinger/issues)
- Provide a clear description of the problem
- Include steps to reproduce the issue
- Share your environment details (OS, Go version, etc.)

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and ensure all pass (`make test`)
5. Run code quality checks (`make check`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to your branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/harbinger.git
cd harbinger

# Add upstream remote
git remote add upstream https://github.com/javanhut/harbinger.git

# Install dependencies
make mod

# Run tests
make test

# Run the tool in development mode
make dev
```

### Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Ensure `make lint` passes
- Write clear commit messages

### Testing

- Add tests for new features
- Ensure all tests pass with `make test`
- Run race condition tests with `make test-race`
- Aim for good test coverage

## Project Structure

```
harbinger/
├── cmd/           # CLI commands
├── internal/      # Internal packages
├── pkg/           # Public packages
└── docs/          # Documentation
```

## Questions?

Feel free to open an issue for any questions about contributing.