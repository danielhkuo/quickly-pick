# Production Deployment Guide

This guide covers deploying Quickly Pick to production environments.

## Prerequisites

- PostgreSQL 12+ database
- Server with Go runtime or Docker
- Domain name (optional, for HTTPS)
- Reverse proxy (nginx, Caddy, or similar)

## Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db?sslmode=require` |
| `PORT` | Server listen port | `3318` |
| `ADMIN_KEY_SALT` | Secret for admin key HMAC | 32+ random characters |
| `POLL_SLUG_SALT` | Secret for share slug HMAC | 32+ random characters |

### Generating Secure Salts

```bash
# Generate two unique salts
openssl rand -base64 32  # For ADMIN_KEY_SALT
openssl rand -base64 32  # For POLL_SLUG_SALT
```

**Security notes:**
- Use different salts for each variable
- Never commit salts to version control
- Rotate salts only if all existing admin keys should be invalidated

## Deployment Options

### Option 1: Docker Deployment

#### Dockerfile

Create `Dockerfile` in the `server/` directory:

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /quickly-pick main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /quickly-pick .

EXPOSE 3318
CMD ["./quickly-pick"]
```

#### Build and Run

```bash
# Build image
docker build -t quickly-pick:latest ./server

# Run container
docker run -d \
  --name quickly-pick \
  -p 3318:3318 \
  -e DATABASE_URL="postgres://..." \
  -e ADMIN_KEY_SALT="your-admin-salt" \
  -e POLL_SLUG_SALT="your-slug-salt" \
  quickly-pick:latest
```

#### Docker Compose (Full Stack)

```yaml
version: '3.8'

services:
  api:
    build: ./server
    ports:
      - "3318:3318"
    environment:
      - DATABASE_URL=postgres://quicklypick:password@db:5432/quicklypick?sslmode=disable
      - ADMIN_KEY_SALT=${ADMIN_KEY_SALT}
      - POLL_SLUG_SALT=${POLL_SLUG_SALT}
    depends_on:
      - db
    restart: unless-stopped

  db:
    image: postgres:15-alpine
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=quicklypick
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=quicklypick
    restart: unless-stopped

volumes:
  pgdata:
```

### Option 2: Direct Binary Deployment

#### Build

```bash
cd server
go build -o quickly-pick main.go
```

#### Systemd Service

Create `/etc/systemd/system/quickly-pick.service`:

```ini
[Unit]
Description=Quickly Pick API Server
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/quickly-pick
ExecStart=/opt/quickly-pick/quickly-pick
Restart=always
RestartSec=5

# Environment (or use EnvironmentFile)
Environment=PORT=3318
Environment=DATABASE_URL=postgres://user:pass@localhost:5432/quicklypick?sslmode=disable
Environment=ADMIN_KEY_SALT=your-admin-salt
Environment=POLL_SLUG_SALT=your-slug-salt

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable quickly-pick
sudo systemctl start quickly-pick
sudo systemctl status quickly-pick
```

## Database Setup

### Production PostgreSQL Configuration

```sql
-- Create user and database
CREATE USER quicklypick WITH PASSWORD 'strong-password-here';
CREATE DATABASE quicklypick OWNER quicklypick;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE quicklypick TO quicklypick;
```

### Connection Pooling

For high traffic, use PgBouncer or similar:

```ini
[databases]
quicklypick = host=localhost dbname=quicklypick

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
```

### Backups

```bash
# Daily backup script
pg_dump -h localhost -U quicklypick quicklypick | gzip > /backups/quicklypick-$(date +%Y%m%d).sql.gz
```

## Reverse Proxy Configuration

### Nginx

```nginx
upstream quickly_pick {
    server 127.0.0.1:3318;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name api.quickly-pick.com;

    ssl_certificate /etc/letsencrypt/live/api.quickly-pick.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.quickly-pick.com/privkey.pem;

    location / {
        proxy_pass http://quickly_pick;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}

server {
    listen 80;
    server_name api.quickly-pick.com;
    return 301 https://$server_name$request_uri;
}
```

### Caddy

```
api.quickly-pick.com {
    reverse_proxy localhost:3318
}
```

## Health Monitoring

### Health Endpoint

The `/health` endpoint returns `OK` when the server is running:

```bash
curl https://api.quickly-pick.com/health
```

### Monitoring Script

```bash
#!/bin/bash
HEALTH_URL="https://api.quickly-pick.com/health"
EXPECTED="OK"

response=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ "$response" != "200" ]; then
    echo "Health check failed: HTTP $response"
    # Send alert (email, Slack, PagerDuty, etc.)
    exit 1
fi
```

### Recommended Monitoring

- **Uptime monitoring**: Pingdom, UptimeRobot, or similar
- **Metrics**: Prometheus + Grafana
- **Logs**: Structured JSON logs to ELK/Loki
- **Alerts**: Set up for:
  - Health check failures
  - High error rates
  - Database connection issues
  - High latency

## Security Checklist

- [ ] Use HTTPS everywhere (TLS 1.2+)
- [ ] Set strong, unique salts for ADMIN_KEY_SALT and POLL_SLUG_SALT
- [ ] Use SSL mode for database connections (`sslmode=require`)
- [ ] Firewall database port (only allow app server)
- [ ] Regular security updates for OS and dependencies
- [ ] Rate limiting at reverse proxy level
- [ ] Log aggregation for audit trail
- [ ] Regular database backups with tested restore procedure

## Scaling Considerations

### Horizontal Scaling

The API is stateless - run multiple instances behind a load balancer:

```
                    ┌──────────────────┐
                    │  Load Balancer   │
                    └────────┬─────────┘
             ┌───────────────┼───────────────┐
             ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │  API (1)    │ │  API (2)    │ │  API (3)    │
    └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
           └───────────────┼───────────────┘
                           ▼
                    ┌──────────────┐
                    │  PostgreSQL  │
                    └──────────────┘
```

### Database Scaling

- Read replicas for results queries
- Connection pooling (PgBouncer)
- Table partitioning for old polls (archive)

### Caching (Future)

For high-read scenarios:
- Cache poll metadata (Redis)
- Cache closed results (immutable)
- Cache ballot counts

## Troubleshooting

### Application won't start

Check logs:
```bash
journalctl -u quickly-pick -f
```

Common issues:
- Missing environment variables
- Database connection refused
- Port already in use

### Database connection errors

```bash
# Test connection
psql "postgres://user:pass@host:5432/db?sslmode=require"

# Check max connections
SELECT count(*) FROM pg_stat_activity;
```

### High memory usage

The Go server is lightweight. If memory grows:
- Check for connection leaks
- Review database query patterns
- Consider connection pooling
