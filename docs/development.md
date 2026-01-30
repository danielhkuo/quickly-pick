# Development Guide

This guide covers local development setup, testing, and project structure for Quickly Pick.

## Prerequisites

- **Go 1.25.3+** - [Install Go](https://go.dev/dl/)
- **PostgreSQL 12+** - [Install PostgreSQL](https://www.postgresql.org/download/) or use Docker
- **Docker** (optional) - For running PostgreSQL container
- **Node.js 18+** - For frontend development

## Quick Start

### 1. Start the Database

Using Docker (recommended):

```bash
docker-compose -f docker-postgres-compose.yml up -d
```

Or connect to an existing PostgreSQL instance.

### 2. Configure Environment

Create a `.env` file in the `server/` directory:

```bash
cd server
cp .env.example .env  # if example exists, or create manually
```

Required environment variables:

```env
DATABASE_URL=postgres://user:password@localhost:5432/quicklypick?sslmode=disable
PORT=3318
ADMIN_KEY_SALT=your-secret-admin-salt-min-32-chars
POLL_SLUG_SALT=your-secret-slug-salt-min-32-chars
```

**Security note:** Use strong, unique salts in production. Generate with:

```bash
openssl rand -base64 32
```

### 3. Run the Server

```bash
cd server
go run main.go
```

Or with explicit flags:

```bash
go run main.go -p 3318 -d "postgres://user:pass@localhost:5432/quicklypick?sslmode=disable"
```

The server will:
1. Load `.env` file (if present)
2. Connect to PostgreSQL
3. Create tables (if not exist)
4. Start listening on the configured port

### 4. Run the Frontend

```bash
cd client/web-old
npm install
npm run dev
```

Frontend runs at `http://localhost:5173` by default.

## Environment Variables

### Server Configuration

| Variable | CLI Flag | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | `-d` | (required) | PostgreSQL connection string |
| `PORT` | `-p` | `3318` | Server port |
| `ADMIN_KEY_SALT` | `--admin-salt` | (required) | Secret for admin key generation |
| `POLL_SLUG_SALT` | `--slug-salt` | (required) | Secret for share slug generation |

### Frontend Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | `http://localhost:3318` | Backend API URL |

## Testing

### Run All Backend Tests

```bash
cd server
go test ./...
```

### Run Tests for a Specific Package

```bash
cd server
go test ./handlers
go test ./auth
go test ./middleware
go test ./cliparse
```

### Run a Single Test

```bash
cd server
go test ./handlers -run TestBMJ_BasicRanking
go test ./handlers -run TestVoting_SubmitBallot
```

### Run Tests with Verbose Output

```bash
cd server
go test -v ./...
```

### Run Tests with Coverage

```bash
cd server
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend Tests

```bash
cd client/web-old
npm run lint  # ESLint
npm run build  # Type checking via build
```

## Project Structure

```
quickly-pick/
├── server/                    # Go backend
│   ├── main.go               # Entry point
│   ├── auth/                 # Authentication utilities
│   │   └── auth.go           # HMAC keys, voter tokens
│   ├── cliparse/             # Configuration
│   │   └── cliparse.go       # CLI flags, env vars
│   ├── db/                   # Database
│   │   └── schema.go         # Table creation
│   ├── handlers/             # HTTP handlers
│   │   ├── polls.go          # Poll CRUD
│   │   ├── voting.go         # Ballot submission
│   │   ├── results.go        # Results retrieval
│   │   ├── devices.go        # Device management
│   │   └── bmj.go            # BMJ algorithm
│   ├── middleware/           # HTTP middleware
│   │   └── middleware.go     # CORS, logging, JSON helpers
│   ├── models/               # Data types
│   │   └── types.go          # Request/response structs
│   └── router/               # Route definitions
│       └── router.go         # Endpoint registration
│
├── client/
│   └── web-old/              # React frontend
│       ├── src/
│       │   ├── api/          # API client
│       │   ├── components/   # React components
│       │   ├── pages/        # Route pages
│       │   ├── hooks/        # Custom hooks
│       │   └── utils/        # Utilities
│       └── public/           # Static assets
│
└── docs/                     # Documentation
    ├── api.md               # API reference
    ├── algorithm.md         # BMJ explanation
    ├── database.md          # Schema docs
    ├── development.md       # This file
    └── deployment.md        # Production setup
```

## Common Tasks

### Add a New Endpoint

1. Define request/response types in `server/models/types.go`
2. Create handler function in appropriate file under `server/handlers/`
3. Register route in `server/router/router.go`
4. Write tests in `*_test.go` file

### Modify Database Schema

1. Add new tables/columns to `server/db/schema.go`
2. Use `IF NOT EXISTS` for safe migrations
3. Update relevant model types
4. Document changes in `docs/database.md`

### Debug Database Queries

Connect directly to PostgreSQL:

```bash
docker exec -it quickly-pick-postgres psql -U user -d quicklypick
```

Useful queries:

```sql
-- List polls
SELECT id, title, status, share_slug FROM poll;

-- Count ballots per poll
SELECT poll_id, COUNT(*) FROM ballot GROUP BY poll_id;

-- View scores for a poll
SELECT o.label, s.value01
FROM score s
JOIN option o ON s.option_id = o.id
JOIN ballot b ON s.ballot_id = b.id
WHERE b.poll_id = 'your-poll-id';
```

## Code Style

### Go

- Standard Go formatting (`go fmt`)
- Package names match directory names
- Exported functions have doc comments
- Error handling with wrapped errors (`fmt.Errorf("...: %w", err)`)

### Frontend

- ESLint configuration in project
- TypeScript strict mode
- Functional components with hooks

## Troubleshooting

### "DATABASE_URL required" error

Ensure `.env` file exists in `server/` directory or set environment variable:

```bash
export DATABASE_URL="postgres://..."
```

### "ADMIN_KEY_SALT required" error

Both salt values are required for security. Add to `.env`:

```env
ADMIN_KEY_SALT=your-32-char-secret
POLL_SLUG_SALT=another-32-char-secret
```

### Database connection refused

1. Check PostgreSQL is running: `docker ps`
2. Verify connection string format
3. Check firewall/network settings

### Port already in use

```bash
# Find process using port 3318
lsof -i :3318

# Kill it or use different port
go run main.go -p 3319
```
