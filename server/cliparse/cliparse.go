package cliparse

import (
	"errors"
	"flag"
	"os"
	"strconv"
)

type Config struct {
	Port         int
	DatabaseURL  string
	DatabaseType string
	AdminKeySalt string
	PollSlugSalt string
}

// ParseFlags validates flags and sets port number
func ParseFlags(args []string) (Config, error) {
	var cfg Config

	fs := flag.NewFlagSet("quickly-pick", flag.ContinueOnError)

	// Network config (can be CLI args or env)
	fs.IntVar(&cfg.Port, "p", 0, "Server port")
	fs.StringVar(&cfg.DatabaseURL, "d", "", "Database URL")
	fs.StringVar(&cfg.DatabaseType, "t", "", "Database type (sqlite or postgres)")

	// Secrets (prefer env variables, but allow CLI for dev)
	fs.StringVar(&cfg.AdminKeySalt, "admin-salt", "", "Admin key salt (prefer env)")
	fs.StringVar(&cfg.PollSlugSalt, "slug-salt", "", "Poll slug salt (prefer env)")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// Fall back to environment variables
	if cfg.Port == 0 {
		if portStr := os.Getenv("PORT"); portStr != "" {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return Config{}, errors.New("invalid PORT env variable")
			}
			cfg.Port = port
		} else {
			cfg.Port = 3318 // default
		}
	}
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("database URL required (use -d or DATABASE_URL env)")
	}

	if cfg.DatabaseType == "" {
		cfg.DatabaseType = os.Getenv("DATABASE_TYPE")
		if cfg.DatabaseType == "" {
			cfg.DatabaseType = "sqlite"
		}
	}

	// Secrets - MUST be provided
	if cfg.AdminKeySalt == "" {
		cfg.AdminKeySalt = os.Getenv("ADMIN_KEY_SALT")
	}
	if cfg.AdminKeySalt == "" {
		return Config{}, errors.New("ADMIN_KEY_SALT required")
	}

	if cfg.PollSlugSalt == "" {
		cfg.PollSlugSalt = os.Getenv("POLL_SLUG_SALT")
	}
	if cfg.PollSlugSalt == "" {
		return Config{}, errors.New("POLL_SLUG_SALT required")
	}

	return cfg, nil
}
