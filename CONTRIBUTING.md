# Contributing to Quickly Pick

Thank you for your interest in contributing to Quickly Pick! This document provides guidelines and information for contributors.

## Code of Conduct

This project follows a simple code of conduct: be respectful, constructive, and collaborative. We welcome contributions from developers of all experience levels.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
3. **Set up the development environment** following [docs/development.md](docs/development.md)
4. **Create a feature branch** from `main`
5. **Make your changes** with tests
6. **Submit a pull request**

## Development Workflow

### Branch Naming

Use descriptive branch names with prefixes:

```
feat/add-poll-expiration
fix/handle-empty-ballots
docs/update-api-examples
refactor/simplify-auth-flow
```

### Commit Messages

Follow the conventional commit format:

```
type(scope): description

feat(api): add poll expiration endpoint
fix(auth): handle expired tokens correctly
docs(readme): update installation instructions
test(bmj): add edge case tests for veto logic
refactor(db): simplify schema creation
```

Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

### Code Style

#### Go Code Standards

- Follow standard Go formatting: `go fmt ./...`
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and small
- Handle errors explicitly

```go
// Good
func CreatePoll(title, creatorName string) (*Poll, error) {
    if title == "" {
        return nil, errors.New("title cannot be empty")
    }

    poll := &Poll{
        ID:          generateID(),
        Title:       title,
        CreatorName: creatorName,
        Status:      "draft",
        CreatedAt:   time.Now(),
    }

    return poll, nil
}

// Avoid
func cp(t, c string) *Poll {
    return &Poll{t, c, "draft"}
}
```

#### Database Guidelines

- Use descriptive table and column names
- Add appropriate indexes for query patterns
- Include foreign key constraints
- Use transactions for multi-table operations

#### API Design

- Follow RESTful conventions
- Use consistent error response format
- Include proper HTTP status codes
- Validate input parameters

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package
go test ./auth -v
```

### Writing Tests

- Write tests for new features and bug fixes
- Include edge cases and error conditions
- Use table-driven tests for multiple scenarios
- Mock external dependencies

```go
func TestCreatePoll(t *testing.T) {
    tests := []struct {
        name        string
        title       string
        creatorName string
        wantErr     bool
    }{
        {
            name:        "valid poll",
            title:       "Test Poll",
            creatorName: "Alice",
            wantErr:     false,
        },
        {
            name:        "empty title",
            title:       "",
            creatorName: "Alice",
            wantErr:     true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            poll, err := CreatePoll(tt.title, tt.creatorName)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreatePoll() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && poll.Title != tt.title {
                t.Errorf("CreatePoll() title = %v, want %v", poll.Title, tt.title)
            }
        })
    }
}
```

## Documentation

### Code Documentation

- Document all exported functions and types
- Include usage examples for complex functions
- Explain algorithm choices and trade-offs

```go
// BMJRanking computes Balanced Majority Judgment ranking for poll options.
// It converts slider values to signed scores, computes statistical aggregates,
// applies soft veto rules, and returns options ranked by lexicographic ordering.
//
// The algorithm prioritizes options with higher medians while penalizing
// options that receive significant negative feedback (soft veto).
//
// Returns options sorted by rank (best first) with detailed statistics.
func BMJRanking(scores map[string][]float64) ([]RankedOption, error) {
    // Implementation...
}
```

### API Documentation

- Update [docs/api.md](docs/api.md) for endpoint changes
- Include request/response examples
- Document error conditions and status codes

### README Updates

- Update feature lists for new functionality
- Add new dependencies to installation instructions
- Update examples if API changes

## Pull Request Process

### Before Submitting

1. **Run tests**: `go test ./...`
2. **Format code**: `go fmt ./...`
3. **Run linter**: `golangci-lint run`
4. **Update documentation** if needed
5. **Test manually** with development server

### PR Description Template

```markdown
## Description

Brief description of changes and motivation.

## Type of Change

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing

- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist

- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings or errors
```

### Review Process

1. **Automated checks** must pass (tests, linting)
2. **Code review** by maintainer
3. **Manual testing** if needed
4. **Merge** after approval

## Issue Reporting

### Bug Reports

Use the bug report template:

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:

1. Create poll with '...'
2. Submit ballot with '...'
3. See error

**Expected behavior**
What you expected to happen.

**Environment**

- OS: [e.g. macOS, Ubuntu]
- Go version: [e.g. 1.25.3]
- Database: [e.g. PostgreSQL 15]

**Additional context**
Any other context about the problem.
```

### Feature Requests

Use the feature request template:

```markdown
**Is your feature request related to a problem?**
A clear description of what the problem is.

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Other solutions you've considered.

**Additional context**
Any other context about the feature request.
```

## Architecture Decisions

### Adding New Features

Consider these questions:

1. **Does it fit the MVP scope?** Check against [MVP.md](MVP.md)
2. **Is it backward compatible?** Avoid breaking existing APIs
3. **Does it require database changes?** Plan migration strategy
4. **How does it affect performance?** Consider scalability impact
5. **Is it testable?** Ensure good test coverage

### Database Changes

For schema changes:

1. **Create migration scripts** (future enhancement)
2. **Test with existing data**
3. **Document breaking changes**
4. **Consider rollback strategy**

### API Changes

For API modifications:

1. **Maintain backward compatibility** when possible
2. **Version new endpoints** if breaking changes needed
3. **Update documentation** and examples
4. **Consider client impact**

## Security Guidelines

### Input Validation

- Validate all user inputs
- Use parameterized queries for database operations
- Sanitize data for logging
- Implement rate limiting

### Authentication

- Use secure token generation
- Implement proper session management
- Hash sensitive data
- Follow principle of least privilege

### Error Handling

- Don't expose internal details in error messages
- Log security events appropriately
- Implement proper error recovery

## Performance Considerations

### Database Queries

- Use appropriate indexes
- Avoid N+1 query problems
- Consider query complexity
- Monitor slow queries

### Memory Usage

- Avoid memory leaks
- Use appropriate data structures
- Consider garbage collection impact
- Profile memory usage for large datasets

### Concurrency

- Handle concurrent access properly
- Use appropriate locking mechanisms
- Consider race conditions
- Test under load

## Release Process

### Version Numbering

We use semantic versioning (SemVer):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. **Update version** in relevant files
2. **Update CHANGELOG.md**
3. **Run full test suite**
4. **Create release tag**
5. **Deploy to staging**
6. **Test deployment**
7. **Deploy to production**

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Pull Request Comments**: Code-specific discussions

### Resources

- [Development Setup](docs/development.md)
- [API Documentation](docs/api.md)
- [Database Schema](docs/database.md)
- [Algorithm Guide](docs/algorithm.md)

## Recognition

Contributors will be recognized in:

- **README.md** contributors section
- **Release notes** for significant contributions
- **GitHub contributors** page

Thank you for contributing to Quickly Pick! Your efforts help make this project better for everyone.
