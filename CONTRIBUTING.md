# Contributing to Parallax

Thank you for considering contributing to Parallax! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by the [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/parallax.git
   cd parallax
   ```
3. Create a new branch for your feature:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Environment

1. Install prerequisites:
   - Go 1.20+
   - Docker
   - kubectl
   - kind (for local testing)

2. Set up your development environment:
   ```bash
   # Install dependencies
   go mod tidy
   
   # Run the operator locally
   make run
   ```

## Making Changes

1. Make your changes following the project's coding standards
2. Add tests for your changes
3. Update documentation as needed
4. Run tests:
   ```bash
   make test
   make test-e2e
   ```

## Pull Request Process

1. Update the README.md and documentation with details of changes if applicable
2. Update the CHANGELOG.md with details of changes
3. The PR will be merged once you have:
   - Two approvals from maintainers
   - All CI checks passing
   - No merge conflicts

## Coding Standards

- Follow the Go code review comments: https://github.com/golang/go/wiki/CodeReviewComments
- Use meaningful commit messages
- Keep PRs focused on a single feature or bug fix
- Include tests for new features
- Update documentation for any changes

## Testing

- Write unit tests for new features
- Add integration tests where appropriate
- Run all tests before submitting a PR:
  ```bash
  make test
  make test-e2e
  ```

## Documentation

- Update README.md for significant changes
- Add comments to complex code sections
- Document any new configuration options
- Update examples if necessary

## Release Process

1. Create a release branch from main
2. Update version numbers and changelog
3. Create a PR for the release
4. After approval, tag the release
5. Create a GitHub release

## Questions or Issues?

If you have questions or encounter issues:
1. Check the existing issues
2. Search the documentation
3. Open a new issue if needed

## Thank You!

Your contributions are greatly appreciated. Thank you for helping make Parallax better! 