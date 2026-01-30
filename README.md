# Quickly Pick - Balanced Polling Service

A full-stack group polling service with a **bipolar slider** (hate ↔ meh ↔ love) and **Balanced Majority Judgment (BMJ)** aggregation.

## Quick Start

### Backend Setup

```bash
# Start PostgreSQL (using Docker)
docker-compose -f docker-postgres-compose.yml up -d

# Run the server
cd server
go run main.go --port 3318 --database-url "postgres://user:password@localhost:5432/quicklypick?sslmode=disable"
```

### Frontend Setup

```bash
# In a new terminal, navigate to frontend
cd client/web-old

# Install dependencies
npm install

# Configure environment
cp .env.example .env
# Edit .env to set VITE_API_BASE_URL=http://localhost:3318

# Start development server
npm run dev
```

The frontend will be available at `http://localhost:5173`

## What is Quickly Pick?

Create a poll, share a link, everyone votes with a slider per option. Results are hidden during voting and revealed once the poll is closed. The winner is chosen by **BMJ**, which respects intensity and avoids options that many people dislike.

### Key Features

- **No accounts required** - identity is a unique username per poll
- **One ballot per voter** (editable while poll is open)
- **Results sealed until close** (no peek-then-tweak)
- **Balanced Majority Judgment** algorithm for fair results
- **Modern React frontend** with mobile-first design
- **Simple stack** - React + Go API + PostgreSQL

## Documentation

### Getting Started
- [Development Setup](docs/development.md) - Local setup and testing
- [API Documentation](docs/api.md) - REST API endpoints and examples
- [Deployment Guide](docs/deployment.md) - Production deployment instructions

### Technical Reference
- [Database Schema](docs/database.md) - Data model and relationships
- [Algorithm Guide](docs/algorithm.md) - How Balanced Majority Judgment works

### Package Documentation

Run `go doc` for any package:

```bash
go doc ./server/auth
go doc ./server/handlers
go doc ./server/models
```

## Project Structure

```
├── client/
│   └── web-old/              # React frontend application
│       ├── src/
│       │   ├── api/          # API client and utilities
│       │   ├── components/   # Reusable UI components
│       │   ├── pages/        # Route components
│       │   └── types/        # TypeScript definitions
│       └── public/           # Static assets
├── server/                   # Go backend service
│   ├── auth/                # Authentication utilities
│   ├── cliparse/            # CLI argument parsing
│   ├── db/                  # Database schema and operations
│   ├── handlers/            # HTTP request handlers
│   ├── middleware/          # HTTP middleware and utilities
│   ├── models/              # Data types and structures
│   ├── router/              # HTTP routing and endpoints
│   └── main.go              # Application entry point
└── docs/                    # Documentation
```

## Technology Stack

### Frontend
- **React 19** with new compiler
- **TypeScript** for type safety
- **React Router 6** for routing
- **Vite** for fast development and optimized builds
- **Native Fetch API** for HTTP requests
- **CSS** for styling (no frameworks)

### Backend
- **Go 1.25.3+** for high performance
- **PostgreSQL 12+** for reliable data storage
- **Standard library** HTTP server (no frameworks)

## Development Workflow

### Running Both Services

```bash
# Terminal 1: Backend
cd server
go run main.go

# Terminal 2: Frontend
cd client/web-old
npm run dev
```

### Building for Production

```bash
# Build frontend
cd client/web-old
npm run build
# Output in client/web-old/dist/

# Build backend
cd server
go build -o quickly-pick main.go
```

### Running Tests

```bash
cd server
go test ./...
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and how to contribute to this project.

## License

Copyright (c) 2025 Daniel Kuo.

Source-available; no permission granted to use, copy, modify, or distribute. See [LICENSE](LICENSE).
