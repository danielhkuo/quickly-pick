# Development Setup

This guide will help you set up a local development environment for Quickly Pick.

## Prerequisites

- **Go 1.25.3+** - [Install Go](https://golang.org/doc/install)
- **PostgreSQL 12+** - [Install PostgreSQL](https://www.postgresql.org/download/) or use Docker
- **Git** - For version control

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/danielhkuo/quickly-pick.git
cd quickly-pick
```

### 2. Start PostgreSQL

#### Option A: Using Docker (Recommended)

```bash
# Start PostgreSQL container
docker-compose -f docker-postgres-compose.yml up -d

# Verify it's running
docker-compose -f docker-postgres-compose.yml ps
```

#### Option B: Local PostgreSQL

```bash
# Create database
createdb quicklypick

# Or using psql
psql -c "CREATE DATABASE quicklypick;"
```

### 3. Configure Environment

```bash
cd server
cp .env.example .env
```

Edit `.env` with your database connection:

```env
DATABASE_URL=postgres://user:password@localhost:5432/quicklypick?sslmode=disable
PORT=8080
```

### 4. Install Dependencies

```bash
cd server
go mod download
```

### 5. Run the Server

```bash
go run main.go
```

The server will start on `http://localhost:8080` and automatically create the database schema.

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./auth
go test ./cliparse
```

### Code Formatting

```bash
# Format all Go files
go fmt ./...

# Run linter (install golangci-lint first)
golangci-lint run
```

### Database Operations

#### Reset Database

```bash
# Drop and recreate database
dropdb quicklypick && createdb quicklypick

# Or restart Docker container
docker-compose -f docker-postgres-compose.yml down
docker-compose -f docker-postgres-compose.yml up -d
```

#### Connect to Database

```bash
# Using psql
psql postgres://user:password@localhost:5432/quicklypick

# Or with Docker
docker-compose -f docker-postgres-compose.yml exec postgres psql -U user -d quicklypick
```

## Project Structure

```
server/
├── auth/              # Authentication utilities
│   ├── auth.go        # Token generation and validation
│   ├── auth_test.go   # Auth tests
│   └── README.md      # Auth documentation
├── cliparse/          # CLI argument parsing
│   ├── cliparse.go    # Configuration parsing
│   ├── cliparse_test.go
│   └── README.md      # CLI documentation
├── db/                # Database operations
│   ├── schema.go      # Schema creation
│   └── README.md      # Database documentation
├── handlers/          # HTTP request handlers
│   ├── polls.go       # Poll management handlers
│   ├── voting.go      # Voting handlers
│   ├── results.go     # Results handlers
│   ├── *_test.go      # Handler tests
│   └── README.md      # Handler documentation
├── middleware/        # HTTP middleware
│   ├── middleware.go  # Logging, JSON responses, utilities
│   └── README.md      # Middleware documentation
├── models/            # Data types and structures
│   ├── types.go       # Request/response types, domain models
│   └── README.md      # Models documentation
├── router/            # HTTP routing
│   ├── router.go      # Route definitions and setup
│   └── README.md      # Router documentation
├── .env               # Environment variables
├── go.mod             # Go module definition
├── go.sum             # Go module checksums
├── README.md          # Server documentation
└── main.go            # Application entry point
```

## Configuration Options

The server accepts these command-line flags:

```bash
go run main.go --help
```

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--port` | `PORT` | `8080` | HTTP server port |
| `--database-url` | `DATABASE_URL` | Required | PostgreSQL connection string |

## API Testing

### Using curl

```bash
# Health check
curl http://localhost:8080/health

# Create a poll
curl -X POST http://localhost:8080/polls \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Poll","creator_name":"Dev"}'
```

### Using HTTPie

```bash
# Install HTTPie
pip install httpie

# Create poll
http POST localhost:8080/polls title="Test Poll" creator_name="Dev"
```

## Debugging

### Enable Debug Logging

Set environment variable:
```bash
export LOG_LEVEL=debug
go run main.go
```

### Database Query Logging

Add to your PostgreSQL configuration or Docker environment:
```
log_statement = 'all'
log_min_duration_statement = 0
```

### Common Issues

#### "connection refused" Error
- Check if PostgreSQL is running
- Verify DATABASE_URL is correct
- Check firewall settings

#### "database does not exist" Error
- Create the database: `createdb quicklypick`
- Or check database name in DATABASE_URL

#### Port Already in Use
- Change port: `go run main.go --port 8081`
- Or kill process using port: `lsof -ti:8080 | xargs kill`

## IDE Setup

### VS Code

Recommended extensions:
- [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go)
- [PostgreSQL](https://marketplace.visualstudio.com/items?itemName=ms-ossdata.vscode-postgresql)

### GoLand

GoLand has built-in Go support. Configure database connection in the Database tool window.

## Contributing

### Before Submitting PRs

1. Run tests: `go test ./...`
2. Format code: `go fmt ./...`
3. Run linter: `golangci-lint run`
4. Update documentation if needed

### Commit Message Format

```
type(scope): description

Examples:
feat(api): add poll creation endpoint
fix(auth): handle expired tokens correctly
docs(readme): update installation instructions
```

## Performance Testing

### Load Testing with hey

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test health endpoint
hey -n 1000 -c 10 http://localhost:8080/health

# Test poll creation
hey -n 100 -c 5 -m POST -H "Content-Type: application/json" \
  -d '{"title":"Load Test","creator_name":"Bot"}' \
  http://localhost:8080/polls
```

## Docker Development

### Build Docker Image

```bash
# From project root
docker build -t quickly-pick ./server

# Run container
docker run -p 8080:8080 --env-file server/.env quickly-pick
```

### Docker Compose Full Stack

```bash
# Start both database and app
docker-compose up -d
```

## Troubleshooting

### Reset Everything

```bash
# Stop all containers
docker-compose down -v

# Remove Go module cache
go clean -modcache

# Restart fresh
docker-compose up -d
cd server && go run main.go
```

### Check Dependencies

```bash
# Verify Go installation
go version

# Check module status
go mod verify

# Update dependencies
go mod tidy
```