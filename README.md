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

- [API Documentation](docs/api.md) - REST API endpoints and examples
- [Database Schema](docs/database.md) - Data model and relationships
- [Algorithm Guide](docs/algorithm.md) - How Balanced Majority Judgment works
- [Development Setup](docs/development.md) - Local development guide
- [Deployment Guide](docs/deployment.md) - Production deployment instructions

## Project Structure

```
├── server/                 # Go backend service
│   ├── auth/              # Authentication utilities
│   ├── cliparse/          # CLI argument parsing
│   ├── db/                # Database schema and operations
│   ├── router/            # HTTP routing and handlers
│   └── main.go            # Application entry point
├── docs/                  # Documentation
└── MVP.md                 # Product specification
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and how to contribute to this project.

## License

[MIT License](LICENSE)