// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

/*
Package cliparse handles command-line argument parsing and configuration.

# Configuration

ParseFlags returns a Config struct with all settings:

	cfg, err := cliparse.ParseFlags(os.Args[1:])

# Config Fields

  - Port: Server listen port (default: 3318)
  - DatabaseURL: PostgreSQL connection string (required)
  - AdminKeySalt: Secret for admin key HMAC (required)
  - PollSlugSalt: Secret for share slug generation (required)

# CLI Flags

	-p, --port        Server port
	-d, --database-url Database URL
	--admin-salt      Admin key salt
	--slug-salt       Poll slug salt

# Environment Variables

Flags fall back to environment variables:

	PORT          → -p
	DATABASE_URL  → -d
	ADMIN_KEY_SALT → --admin-salt
	POLL_SLUG_SALT → --slug-salt

CLI flags take precedence over environment variables.

# Validation

ParseFlags returns an error if required values are missing:

  - DATABASE_URL must be provided
  - ADMIN_KEY_SALT must be provided
  - POLL_SLUG_SALT must be provided

# Example

	// In main.go
	cfg, err := cliparse.ParseFlags(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	// ...
	mux := router.NewRouter(db, cfg)
*/
package cliparse
