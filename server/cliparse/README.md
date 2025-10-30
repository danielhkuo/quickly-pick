# CLI Parse Package

Command-line argument parsing and configuration management for the Quickly Pick server.

## Overview

This package handles parsing command-line flags and environment variables to configure the server. It provides a clean interface for managing application configuration with sensible defaults and validation.

## Configuration Structure

```go
type Config struct {
    Port        int    // HTTP server port
    DatabaseURL string // PostgreSQL connection string
    AdminKeySalt string // Salt for generating admin keys
    PollSlugSalt string // Salt for generating share slugs
}
```

## Functions

### `ParseFlags(args []string) (*Config, error)`

Parses command-line arguments and environment variables into a configuration struct.

**Parameters:**
- `args []string` - Command-line arguments (typically `os.Args[1:]`)

**Returns:**
- `*Config` - Parsed configuration
- `error` - Parsing or validation error

**Usage:**
```go
cfg, err := cliparse.ParseFlags(os.Args[1:])
if err != nil {
    log.Fatal("Configuration error:", err)
}

fmt.Printf("Starting server on port %d\n", cfg.Port)
```

## Supported Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--port` | `PORT` | `8080` | HTTP server port |
| `--database-url` | `DATABASE_URL` | *required* | PostgreSQL connection string |
| `--admin-key-salt` | `ADMIN_KEY_SALT` | `"default-admin-salt"` | Salt for admin key generation |
| `--poll-slug-salt` | `POLL_SLUG_SALT` | `"default-slug-salt"` | Salt for share slug generation |

## Examples

### Command Line Usage

```bash
# Basic usage with required database URL
./server --database-url "postgres://user:pass@localhost/quicklypick"

# Custom port
./server --port 3000 --database-url "postgres://..."

# All options
./server \
  --port 8080 \
  --database-url "postgres://user:pass@localhost/quicklypick" \
  --admin-key-salt "my-secret-admin-salt" \
  --poll-slug-salt "my-secret-slug-salt"
```

### Environment Variables

```bash
# Set environment variables
export PORT=8080
export DATABASE_URL="postgres://user:pass@localhost/quicklypick"
export ADMIN_KEY_SALT="production-admin-salt"
export POLL_SLUG_SALT="production-slug-salt"

# Run server (will use environment variables)
./server
```

### Mixed Usage

Environment variables are used as defaults, command-line flags override them:

```bash
# DATABASE_URL from environment, port from command line
export DATABASE_URL="postgres://user:pass@localhost/quicklypick"
./server --port 3000
```

## Configuration Validation

The parser validates configuration values:

### Port Validation
- Must be between 1 and 65535
- Common ports (80, 443, 8080, 3000) are recommended

### Database URL Validation
- Must be a valid PostgreSQL connection string
- Should include database name
- SSL mode is recommended for production

### Salt Validation
- Salts should be non-empty strings
- Production salts should be cryptographically random
- Default salts are only for development

## Integration Example

```go
package main

import (
    "log"
    "os"
    
    "github.com/danielhkuo/quickly-pick/cliparse"
)

func main() {
    // Parse configuration
    cfg, err := cliparse.ParseFlags(os.Args[1:])
    if err != nil {
        log.Fatal("Configuration error:", err)
    }
    
    // Validate production settings
    if cfg.AdminKeySalt == "default-admin-salt" {
        log.Warn("Using default admin key salt - not secure for production")
    }
    
    // Use configuration
    server := &http.Server{
        Addr: fmt.Sprintf(":%d", cfg.Port),
        Handler: createHandler(cfg),
    }
    
    log.Printf("Starting server on port %d", cfg.Port)
    log.Fatal(server.ListenAndServe())
}
```

## Production Configuration

### Security Considerations

1. **Never use default salts in production**
   ```bash
   # Generate secure salts
   export ADMIN_KEY_SALT=$(openssl rand -hex 32)
   export POLL_SLUG_SALT=$(openssl rand -hex 32)
   ```

2. **Use secure database connections**
   ```bash
   export DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=require"
   ```

3. **Restrict port access**
   - Use reverse proxy (nginx, Caddy) for public access
   - Bind application to localhost if behind proxy

### Environment File (.env)

For development, create a `.env` file:

```env
# Development configuration
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/quicklypick_dev?sslmode=disable
ADMIN_KEY_SALT=dev-admin-salt-not-for-production
POLL_SLUG_SALT=dev-slug-salt-not-for-production
```

Load with:
```go
import "github.com/joho/godotenv"

func main() {
    // Load .env file (optional)
    godotenv.Load()
    
    cfg, err := cliparse.ParseFlags(os.Args[1:])
    // ...
}
```

## Error Handling

The parser returns specific errors for different validation failures:

```go
cfg, err := cliparse.ParseFlags(args)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "database-url"):
        log.Fatal("Database configuration error:", err)
    case strings.Contains(err.Error(), "port"):
        log.Fatal("Port configuration error:", err)
    default:
        log.Fatal("Configuration error:", err)
    }
}
```

## Testing

### Unit Tests

```go
func TestParseFlags(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        want    *Config
        wantErr bool
    }{
        {
            name: "valid config",
            args: []string{
                "--port", "8080",
                "--database-url", "postgres://localhost/test",
            },
            want: &Config{
                Port:        8080,
                DatabaseURL: "postgres://localhost/test",
                AdminKeySalt: "default-admin-salt",
                PollSlugSalt: "default-slug-salt",
            },
            wantErr: false,
        },
        {
            name: "missing database url",
            args: []string{"--port", "8080"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseFlags(tt.args)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseFlags() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

```bash
# Test with environment variables
export DATABASE_URL="postgres://localhost/test"
go test -v ./cliparse

# Test with command line args
go test -v ./cliparse -args --port 3000 --database-url "postgres://localhost/test"
```

## Dependencies

- `flag` - Standard library flag parsing
- `os` - Environment variable access
- `strconv` - String to integer conversion
- `fmt` - Error formatting

## Best Practices

1. **Always validate configuration** before starting the server
2. **Use environment variables** for sensitive values (passwords, salts)
3. **Provide helpful error messages** for configuration issues
4. **Document all configuration options** in deployment guides
5. **Use different salts** for different environments (dev, staging, prod)

## Troubleshooting

### Common Issues

**"database-url is required"**
- Set `DATABASE_URL` environment variable or use `--database-url` flag

**"invalid port"**
- Port must be a number between 1-65535
- Check for typos in port number

**"connection refused"**
- Database URL might be incorrect
- Database server might not be running
- Check firewall settings

### Debug Configuration

Add debug logging to see parsed configuration:

```go
cfg, err := cliparse.ParseFlags(os.Args[1:])
if err != nil {
    log.Fatal(err)
}

log.Printf("Configuration: Port=%d, DB=%s", cfg.Port, cfg.DatabaseURL)
```