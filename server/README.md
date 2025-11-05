# Quickly Pick Server

Go backend service for the Quickly Pick polling application using Balanced Majority Judgment (BMJ) voting.

## Overview

This is the core backend service that provides a REST API for creating polls, voting with sliders, and computing BMJ results. The server is designed to be stateless, secure, and scalable.

## Quick Start

### Prerequisites

- Go 1.25.3+
- PostgreSQL 12+

### Development Setup

```bash
# Clone repository
git clone https://github.com/danielhkuo/quickly-pick.git
cd quickly-pick/server

# Start PostgreSQL (using Docker)
docker-compose -f ../docker-postgres-compose.yml up -d

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgres://user:password@localhost:5432/quicklypick?sslmode=disable"
export PORT=8080

# Run server
go run main.go
```

The server will start on `http://localhost:8080` and automatically create the database schema.

### Production Deployment

```bash
# Build binary
go build -o quickly-pick main.go

# Run with production settings
./quickly-pick \
  --port 8080 \
  --database-url "postgres://user:pass@host:5432/quicklypick?sslmode=require" \
  --admin-key-salt "$(openssl rand -hex 32)" \
  --poll-slug-salt "$(openssl rand -hex 32)"
```

## Architecture

### Package Structure

```
server/
├── auth/              # Authentication and token generation
├── cliparse/          # CLI argument parsing and configuration
├── db/                # Database schema and operations
├── handlers/          # HTTP request handlers
├── middleware/        # HTTP middleware and utilities
├── models/            # Data types and structures
├── router/            # HTTP routing and endpoint setup
├── main.go            # Application entry point
├── go.mod             # Go module definition
└── .env               # Environment variables (development)
```

### Key Components

- **Authentication** (`auth/`) - HMAC-based admin keys, secure voter tokens, share slugs
- **Database** (`db/`) - PostgreSQL schema with proper constraints and indexes
- **Handlers** (`handlers/`) - REST API endpoint implementations
- **Models** (`models/`) - Request/response types and domain objects
- **Middleware** (`middleware/`) - Logging, JSON responses, error handling

## API Endpoints

### Poll Management (Admin)

```bash
# Create poll
POST /polls
{"title": "Best Pizza", "creator_name": "Alice"}
→ {"poll_id": "abc123", "admin_key": "xyz789"}

# Add options
POST /polls/abc123/options
X-Admin-Key: xyz789
{"label": "Pepperoni"}
→ {"option_id": "opt456"}

# Publish poll
POST /polls/abc123/publish
X-Admin-Key: xyz789
→ {"share_slug": "pizza-poll", "share_url": "https://..."}

# Close poll
POST /polls/abc123/close
X-Admin-Key: xyz789
→ {"closed_at": "2024-10-30T...", "snapshot": {...}}
```

### Voting (Public)

```bash
# Claim username
POST /polls/pizza-poll/claim-username
{"username": "bob_voter"}
→ {"voter_token": "voter123"}

# Submit ballot
POST /polls/pizza-poll/ballots
X-Voter-Token: voter123
{
  "scores": {
    "opt456": 0.8,  # Love pepperoni
    "opt789": 0.2   # Dislike pineapple
  }
}
→ {"ballot_id": "ballot456", "message": "Ballot submitted successfully"}
```

### Data Access (Public)

```bash
# Get poll info
GET /polls/pizza-poll
→ {"poll": {...}, "options": [...]}

# Get results (only after close)
GET /polls/pizza-poll/results
→ {"rankings": [...], "method": "bmj", ...}

# Health check
GET /health
→ {"status": "ok"}
```

## Configuration

### Command Line Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--port` | `PORT` | `8080` | HTTP server port |
| `--database-url` | `DATABASE_URL` | *required* | PostgreSQL connection string |
| `--admin-key-salt` | `ADMIN_KEY_SALT` | `"default-admin-salt"` | Salt for admin key generation |
| `--poll-slug-salt` | `POLL_SLUG_SALT` | `"default-slug-salt"` | Salt for share slug generation |

### Environment File (.env)

```env
# Development configuration
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/quicklypick_dev?sslmode=disable
ADMIN_KEY_SALT=dev-admin-salt-not-for-production
POLL_SLUG_SALT=dev-slug-salt-not-for-production
```

### Production Security

```bash
# Generate secure salts
export ADMIN_KEY_SALT=$(openssl rand -hex 32)
export POLL_SLUG_SALT=$(openssl rand -hex 32)

# Use secure database connection
export DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=require"
```

## Database Schema

### Core Tables

- **poll** - Poll metadata and lifecycle state
- **option** - Poll choices/options
- **username_claim** - Username to voter token mapping
- **ballot** - Submitted votes with metadata
- **score** - Individual option scores (0.0 to 1.0)
- **result_snapshot** - Computed BMJ results

### Key Features

- **Referential integrity** - Foreign key constraints with cascade delete
- **Unique constraints** - One username per poll, one ballot per voter
- **Check constraints** - Score values in [0,1], valid poll status
- **Indexes** - Optimized for common query patterns

## BMJ Algorithm

### Slider Semantics

```
Hate ←──────── Meh ──────────→ Love
0.0           0.5            1.0
```

Converted to signed scores: `s = 2 × value01 - 1`

### Ranking Criteria (Lexicographic Order)

1. **Veto status** (non-vetoed first)
2. **Median** (higher wins)
3. **10th percentile** (least-misery tiebreaker)
4. **90th percentile** (upside tiebreaker)
5. **Mean** (average tiebreaker)
6. **Option ID** (stable final tiebreaker)

### Soft Veto Rule

Options are vetoed if:
- **High negativity**: ≥33% negative scores
- **Non-positive median**: median ≤ 0

## Security Features

### Authentication

- **Admin Keys** - HMAC-SHA256 based, deterministic, verifiable
- **Voter Tokens** - Cryptographically random, 192-bit entropy
- **Share Slugs** - Short, deterministic, HMAC-based

### Privacy Protection

- **IP Hashing** - One-way HMAC hash for abuse detection
- **Token Security** - Voter tokens never logged or exposed
- **Results Sealing** - Results hidden until poll closes (403 Forbidden)

### Input Validation

- **SQL Injection** - Parameterized queries only
- **Range Validation** - Score values in [0,1]
- **Length Limits** - String field length validation
- **Status Validation** - Proper poll state transitions

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./auth
go test ./handlers
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter (install golangci-lint first)
golangci-lint run

# Check for vulnerabilities
go list -json -m all | nancy sleuth
```

### Database Operations

```bash
# Connect to database
psql $DATABASE_URL

# Reset database (development only)
dropdb quicklypick && createdb quicklypick

# View schema
\d+ poll
\d+ option
```

## Monitoring

### Health Check

```bash
curl http://localhost:8080/health
```

### Structured Logging

```json
{
  "time": "2024-10-30T14:30:00Z",
  "level": "INFO",
  "msg": "poll created",
  "poll_id": "abc123",
  "creator": "Alice"
}
```

### Key Metrics to Monitor

- Request duration and throughput
- Database connection pool usage
- Poll creation and voting rates
- Error rates by endpoint
- Memory and CPU usage

## Deployment

### Docker

```dockerfile
FROM golang:1.25.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

### Systemd Service

```ini
[Unit]
Description=Quickly Pick Polling Service
After=network.target postgresql.service

[Service]
Type=simple
User=quickly-pick
ExecStart=/usr/local/bin/quickly-pick
Restart=always
Environment=DATABASE_URL=postgres://...
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

### Reverse Proxy (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Performance

### Database Optimization

- Connection pooling (25 max connections)
- Prepared statements for repeated queries
- Proper indexing on query patterns
- Query optimization with EXPLAIN ANALYZE

### Application Optimization

- Stateless design for horizontal scaling
- Efficient JSON encoding/decoding
- Minimal memory allocations
- Proper resource cleanup

### Scaling Considerations

- Load balancer for multiple instances
- Database read replicas for heavy read workloads
- Redis for session storage (if needed)
- CDN for static assets

## Troubleshooting

### Common Issues

**"connection refused"**
- Check if PostgreSQL is running
- Verify DATABASE_URL is correct
- Check firewall settings

**"database does not exist"**
- Create database: `createdb quicklypick`
- Check database name in connection string

**"port already in use"**
- Change port: `--port 8081`
- Kill process: `lsof -ti:8080 | xargs kill`

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug
go run main.go

# Database query logging
export DATABASE_URL="postgres://...?log_statement=all"
```

### Performance Issues

```sql
-- Check slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC LIMIT 10;

-- Check connection usage
SELECT count(*) FROM pg_stat_activity;
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `go fmt ./...` and `golangci-lint run`
5. Submit a pull request

### Code Standards

- Use `gofmt` for formatting
- Follow Go naming conventions
- Write tests for new functionality
- Update documentation for API changes
- Use structured logging with context

## License

Copyright (c) 2025 Daniel Kuo.

Source-available; no permission granted to use, copy, modify, or distribute. See [LICENSE](../LICENSE).

## Documentation

- [API Documentation](../docs/api.md) - REST API reference
- [Database Schema](../docs/database.md) - Database design and queries
- [BMJ Algorithm](../docs/algorithm.md) - Voting algorithm explanation
- [Development Setup](../docs/development.md) - Local development guide
- [Deployment Guide](../docs/deployment.md) - Production deployment

## Package Documentation

- [auth/](auth/README.md) - Authentication and token generation
- [cliparse/](cliparse/README.md) - Configuration and CLI parsing
- [db/](db/README.md) - Database schema and operations
- [handlers/](handlers/README.md) - HTTP request handlers
- [middleware/](middleware/README.md) - HTTP middleware and utilities
- [models/](models/README.md) - Data types and structures
- [router/](router/README.md) - HTTP routing and endpoints