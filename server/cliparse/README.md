# cliparse

Command-line and environment configuration parsing for the Quickly Pick server.

This package provides a single entry point—`ParseFlags(args)`—that reads config from CLI flags with fallbacks to environment variables, applies a default port when unset, and enforces required settings for the database URL and salts.

## Configuration

```go
type Config struct {
    Port         int
    DatabaseURL  string
    AdminKeySalt string
    PollSlugSalt string
}
````

### Semantics

* **CLI flags win** over environment variables.
* **Port**

  * If `-p` is provided: use it.
  * Else if `PORT` is set: parse it as an integer.
  * Else: default to **3318**.
* **DATABASE_URL** is **required** (via `-d` or `DATABASE_URL`).
* **ADMIN_KEY_SALT** is **required** (via `-admin-salt` or `ADMIN_KEY_SALT`).
* **POLL_SLUG_SALT** is **required** (via `-slug-salt` or `POLL_SLUG_SALT`).
* This package validates presence of required values and that `PORT` (when read from env) is a valid integer. It does not enforce a port range (e.g., 1–65535).

## API

### `ParseFlags(args []string) (Config, error)`

Parses flags and environment variables into a `Config`.

**Parameters**

* `args []string`: typically `os.Args[1:]`

**Returns**

* `Config`: parsed configuration (returned by value)
* `error`: flag parsing error or missing/invalid required settings

**Example**

```go
cfg, err := cliparse.ParseFlags(os.Args[1:])
if err != nil {
    log.Fatal(err)
}
log.Printf("Listening on :%d", cfg.Port)
```

## Supported flags and env vars

| Setting        | Flag          | Env var          | Default      |
| -------------- | ------------- | ---------------- | ------------ |
| Port           | `-p`          | `PORT`           | `3318`       |
| Database URL   | `-d`          | `DATABASE_URL`   | *(required)* |
| Admin key salt | `-admin-salt` | `ADMIN_KEY_SALT` | *(required)* |
| Poll slug salt | `-slug-salt`  | `POLL_SLUG_SALT` | *(required)* |

> Note: These are the exact flag names defined in code. There are no `--port` / `--database-url` long aliases.

## Usage examples

### CLI-only

```bash
./server \
  -p 3318 \
  -d "postgres://user:pass@localhost:5432/quickly_pick_dev?sslmode=disable" \
  -admin-salt "dev-admin-salt" \
  -slug-salt "dev-slug-salt"
```

### Env-only

```bash
export PORT=3318
export DATABASE_URL="postgres://user:pass@localhost:5432/quickly_pick_dev?sslmode=disable"
export ADMIN_KEY_SALT="dev-admin-salt"
export POLL_SLUG_SALT="dev-slug-salt"

./server
```

### Mixed (flags override env)

```bash
export PORT=9000
export DATABASE_URL="postgres://user:pass@localhost:5432/quickly_pick_dev?sslmode=disable"
export ADMIN_KEY_SALT="dev-admin-salt"
export POLL_SLUG_SALT="dev-slug-salt"

# Overrides PORT from env
./server -p 8080
```

## Common errors

* `DATABASE_URL required`
* `ADMIN_KEY_SALT required`
* `POLL_SLUG_SALT required`
* `invalid PORT env variable`

## How this fits with `main.go`

The server’s `main.go` loads a `.env` file (if present) and then calls:

```go
cfg, err := cliparse.ParseFlags(os.Args[1:])
```

So you can use either:

* a `.env` file (loaded by `godotenv`), or
* normal environment variables, or
* CLI flags (highest priority).

## Testing

Run unit tests:

```bash
go test ./cliparse
```

The tests cover:

* environment variable parsing (`PORT`, `DATABASE_URL`, salts)
* CLI overriding environment variables