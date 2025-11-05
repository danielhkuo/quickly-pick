# Quickly Pick - Balanced Polling Service

A backend-first, temporary group polling service with a **bipolar slider** (hate ↔ meh ↔ love) and **Balanced Majority Judgment (BMJ)** aggregation.

## Quick Start

```bash
# Start PostgreSQL (using Docker)
docker-compose -f docker-postgres-compose.yml up -d

# Run the server
cd server
go run main.go --port 8080 --database-url "postgres://user:password@localhost:5432/quicklypick?sslmode=disable"
```

## What is Quickly Pick?

Create a poll, share a link, everyone votes with a slider per option. Results are hidden during voting and revealed once the poll is closed. The winner is chosen by **BMJ**, which respects intensity and avoids options that many people dislike.

### Key Features

- **No accounts required** - identity is a unique username per poll
- **One ballot per voter** (editable while poll is open)
- **Results sealed until close** (no peek-then-tweak)
- **Balanced Majority Judgment** algorithm for fair results
- **Simple stack** - single API process + PostgreSQL database

## Documentation

### User Guides
- [API Documentation](docs/api.md) - REST API endpoints and examples
- [Development Setup](docs/development.md) - Local development guide
- [Deployment Guide](docs/deployment.md) - Production deployment instructions

### Technical Reference
- [Database Schema](docs/database.md) - Data model and relationships
- [Algorithm Guide](docs/algorithm.md) - How Balanced Majority Judgment works
- [Server Documentation](server/README.md) - Backend service overview

### Package Documentation
- [Authentication](server/auth/README.md) - Token generation and validation
- [Configuration](server/cliparse/README.md) - CLI parsing and config management
- [Database](server/db/README.md) - Schema creation and operations
- [Handlers](server/handlers/README.md) - HTTP request handlers
- [Middleware](server/middleware/README.md) - HTTP utilities and logging
- [Models](server/models/README.md) - Data types and structures
- [Router](server/router/README.md) - HTTP routing and endpoints

## Project Structure

```
├── server/                 # Go backend service
│   ├── auth/              # Authentication utilities
│   ├── cliparse/          # CLI argument parsing  
│   ├── db/                # Database schema and operations
│   ├── handlers/          # HTTP request handlers
│   ├── middleware/        # HTTP middleware and utilities
│   ├── models/            # Data types and structures
│   ├── router/            # HTTP routing and endpoints
│   └── main.go            # Application entry point
├── docs/                  # Documentation
└── MVP.md                 # Product specification
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and how to contribute to this project.

## License

Copyright (c) 2025 Daniel Kuo.

Source-available; no permission granted to use, copy, modify, or distribute. See [LICENSE](LICENSE).