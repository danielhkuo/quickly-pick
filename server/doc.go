// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package main provides the entry point for the Quickly Pick API server.

Quickly Pick is a group polling service with a bipolar slider interface
(hate ↔ meh ↔ love) and Balanced Majority Judgment (BMJ) voting algorithm.

# Starting the Server

The server requires environment variables or CLI flags for configuration:

	DATABASE_URL=postgres://... go run main.go

Or with flags:

	go run main.go -p 3318 -d "postgres://..."

# Configuration

Required settings:

  - DATABASE_URL (-d): PostgreSQL connection string
  - ADMIN_KEY_SALT (--admin-salt): Secret for admin key HMAC
  - POLL_SLUG_SALT (--slug-salt): Secret for share slug generation

Optional settings:

  - PORT (-p): Server port (default: 3318)

# Architecture

The server uses a handler-based architecture with dependency injection:

  - handlers: HTTP request handlers (polls, voting, results, devices)
  - router: Route definitions using Go 1.22+ routing
  - middleware: CORS, logging, JSON helpers
  - models: Request/response types
  - auth: Token generation and validation
  - db: Schema creation
  - cliparse: Configuration parsing

See package documentation for each component.
*/
package main
