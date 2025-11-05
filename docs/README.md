# Quickly Pick Documentation

Comprehensive documentation for the Quickly Pick polling service.

## Getting Started

New to Quickly Pick? Start here:

1. **[Development Setup](development.md)** - Set up your local development environment
2. **[API Documentation](api.md)** - Learn the REST API endpoints
3. **[Algorithm Guide](algorithm.md)** - Understand how BMJ voting works

## User Guides

### For Developers

- **[Development Setup](development.md)** - Local development environment setup
- **[API Documentation](api.md)** - Complete REST API reference with examples
- **[Database Schema](database.md)** - Database design and relationships

### For DevOps/Deployment

- **[Deployment Guide](deployment.md)** - Production deployment instructions
- **[Database Schema](database.md)** - Database setup and migration strategies

### For Product/Business

- **[Algorithm Guide](algorithm.md)** - How Balanced Majority Judgment works
- **[MVP Specification](../MVP.md)** - Product requirements and scope

## Technical Reference

### Architecture

- **[Server Overview](../server/README.md)** - Backend service architecture
- **[Database Schema](database.md)** - Data model and query patterns
- **[API Documentation](api.md)** - HTTP endpoints and authentication

### Algorithm

- **[BMJ Algorithm](algorithm.md)** - Voting algorithm explanation
- **[Slider Semantics](algorithm.md#slider-to-score-conversion)** - How slider values map to scores
- **[Ranking Rules](algorithm.md#ranking-algorithm)** - How winners are determined

## Package Documentation

The server is organized into focused packages, each with detailed documentation:

### Core Packages

- **[auth/](../server/auth/README.md)** - Authentication and token generation
  - Admin key generation and validation
  - Voter token creation
  - Share slug generation
  - IP hashing for privacy

- **[handlers/](../server/handlers/README.md)** - HTTP request handlers
  - Poll management (create, publish, close)
  - Voting operations (claim username, submit ballot)
  - Results access (get poll info, get results)

- **[models/](../server/models/README.md)** - Data types and structures
  - Request/response types
  - Domain objects (Poll, Option, Ballot)
  - BMJ result types

### Infrastructure Packages

- **[db/](../server/db/README.md)** - Database schema and operations
  - Schema creation and validation
  - Table relationships and constraints
  - Performance indexes

- **[middleware/](../server/middleware/README.md)** - HTTP middleware and utilities
  - Request logging
  - JSON response formatting
  - Error handling
  - Client IP extraction

- **[router/](../server/router/README.md)** - HTTP routing and endpoints
  - Route definitions
  - Middleware integration
  - Authentication patterns

- **[cliparse/](../server/cliparse/README.md)** - Configuration management
  - CLI argument parsing
  - Environment variable handling
  - Configuration validation

## API Quick Reference

### Poll Lifecycle

```bash
# 1. Create poll (draft)
POST /polls → {poll_id, admin_key}

# 2. Add options
POST /polls/:id/options → {option_id}

# 3. Publish (open for voting)
POST /polls/:id/publish → {share_slug, share_url}

# 4. Close poll (compute results)
POST /polls/:id/close → {closed_at, snapshot}
```

### Voting Flow

```bash
# 1. Claim username
POST /polls/:slug/claim-username → {voter_token}

# 2. Submit ballot (can be updated)
POST /polls/:slug/ballots → {ballot_id, message}
```

### Data Access

```bash
# Get poll info (always available)
GET /polls/:slug → {poll, options}

# Get results (only after close)
GET /polls/:slug/results → {rankings, method, ...}

# Get participation count
GET /polls/:slug/ballot-count → {ballot_count}
```

## Key Concepts

### Poll States

- **draft** - Being created, options can be added
- **open** - Published, accepting votes
- **closed** - Finished, results available

### Authentication

- **Admin Key** - HMAC-based key for poll management
- **Voter Token** - Random token for ballot submission
- **Share Slug** - Public identifier for poll access

### BMJ Voting

- **Slider Range** - 0.0 (hate) to 1.0 (love)
- **Signed Scores** - Converted to -1.0 to +1.0 range
- **Soft Veto** - Options with ≥33% negative votes and non-positive median
- **Ranking** - Lexicographic order by veto, median, p10, p90, mean

### Security Features

- **Results Sealing** - Results hidden until poll closes (403 Forbidden)
- **Privacy Protection** - IP hashing, no token logging
- **Input Validation** - Range checks, SQL injection prevention
- **Authentication** - HMAC validation for admin operations

## Development Workflow

### Local Development

1. **Setup** - Follow [Development Setup](development.md)
2. **Code** - Make changes with proper testing
3. **Test** - Run `go test ./...` for all tests
4. **Format** - Use `go fmt ./...` for consistent formatting
5. **Lint** - Run `golangci-lint run` for code quality

### Testing Strategy

- **Unit Tests** - Test individual functions and methods
- **Integration Tests** - Test complete API workflows
- **Database Tests** - Test schema and query behavior
- **Error Tests** - Test all error conditions and edge cases

### Documentation Updates

When making changes:

1. **Update package READMEs** - If changing package behavior
2. **Update API docs** - If changing endpoints or responses
3. **Update algorithm docs** - If changing BMJ implementation
4. **Update deployment docs** - If changing configuration or requirements

## Troubleshooting

### Common Issues

- **Database Connection** - Check DATABASE_URL and PostgreSQL status
- **Port Conflicts** - Use different port or kill existing process
- **Authentication Errors** - Verify admin keys and voter tokens
- **Results Access** - Ensure poll is closed before accessing results

### Debug Resources

- **[Development Guide](development.md#troubleshooting)** - Local development issues
- **[Deployment Guide](deployment.md#troubleshooting)** - Production issues
- **[Server README](../server/README.md#troubleshooting)** - General server issues

## Contributing

### Documentation

- Keep READMEs up to date with code changes
- Use clear examples and code snippets
- Include both happy path and error scenarios
- Follow consistent formatting and structure

### Code

- Write tests for new functionality
- Update documentation for API changes
- Follow Go conventions and best practices
- Use structured logging with appropriate context

## External Resources

### Go Development

- [Go Documentation](https://golang.org/doc/)
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [PostgreSQL Go Driver](https://github.com/lib/pq)

### PostgreSQL

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [SQL Performance Tuning](https://www.postgresql.org/docs/current/performance-tips.html)

### HTTP APIs

- [REST API Design](https://restfulapi.net/)
- [HTTP Status Codes](https://httpstatuses.com/)
- [JSON API Specification](https://jsonapi.org/)

### Deployment

- [Docker Documentation](https://docs.docker.com/)
- [Nginx Configuration](https://nginx.org/en/docs/)
- [Systemd Service Files](https://www.freedesktop.org/software/systemd/man/systemd.service.html)

---

**Need help?** Check the specific package documentation or create an issue in the repository.