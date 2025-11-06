# Deployment Guide

This guide covers deploying Quickly Pick to production environments.

## Production Requirements

- **Go 1.25.3+** runtime
- **PostgreSQL 12+** database
- **Reverse proxy** (nginx, Caddy, or cloud load balancer)
- **SSL certificate** for HTTPS
- **Process manager** (systemd, Docker, or cloud service)

## Environment Variables

Set these environment variables in production:

```bash
# Required
DATABASE_URL=postgres://user:password@db-host:5432/quicklypick?sslmode=require
PORT=3318

# Optional
LOG_LEVEL=info
RATE_LIMIT_ENABLED=true
CORS_ORIGINS=https://yourdomain.com
```

## Deployment Options

### Option 1: Docker (Recommended)

#### Build Production Image

```dockerfile
# Dockerfile
FROM golang:1.25.3-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
EXPOSE 3318
CMD ["./main"]
```

#### Build and Deploy

```bash
# Build image
docker build -t quickly-pick:latest .

# Run container
docker run -d \
  --name quickly-pick \
  --restart unless-stopped \
  -p 3318:3318 \
  -e DATABASE_URL="postgres://..." \
  quickly-pick:latest
```

#### Docker Compose Production

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  app:
    build: .
    restart: unless-stopped
    ports:
      - "3318:3318"
    environment:
      - DATABASE_URL=postgres://user:password@db:5432/quicklypick
      - LOG_LEVEL=info
    depends_on:
      - db
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3318/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  db:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_DB=quicklypick
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  postgres_data:
```

Deploy with:
```bash
docker-compose -f docker-compose.prod.yml up -d
```

### Option 2: Systemd Service

#### Build Binary

```bash
# On deployment server
git clone https://github.com/danielhkuo/quickly-pick.git
cd quickly-pick/server
go build -o /usr/local/bin/quickly-pick main.go
```

#### Create Service File

```ini
# /etc/systemd/system/quickly-pick.service
[Unit]
Description=Quickly Pick Polling Service
After=network.target postgresql.service

[Service]
Type=simple
User=quickly-pick
Group=quickly-pick
WorkingDirectory=/opt/quickly-pick
ExecStart=/usr/local/bin/quickly-pick
Restart=always
RestartSec=5

# Environment
Environment=DATABASE_URL=postgres://user:password@localhost:5432/quicklypick
Environment=PORT=3318
Environment=LOG_LEVEL=info

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/quickly-pick

[Install]
WantedBy=multi-user.target
```

#### Deploy Service

```bash
# Create user
sudo useradd --system --shell /bin/false quickly-pick
sudo mkdir -p /opt/quickly-pick
sudo chown quickly-pick:quickly-pick /opt/quickly-pick

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable quickly-pick
sudo systemctl start quickly-pick

# Check status
sudo systemctl status quickly-pick
```

### Option 3: Cloud Platforms

#### Heroku

```bash
# Install Heroku CLI and login
heroku create your-app-name

# Add PostgreSQL addon
heroku addons:create heroku-postgresql:mini

# Deploy
git push heroku main

# Set environment variables
heroku config:set LOG_LEVEL=info
```

#### Railway

```bash
# Install Railway CLI
railway login
railway init
railway add postgresql
railway deploy
```

#### Google Cloud Run

```bash
# Build and push image
gcloud builds submit --tag gcr.io/PROJECT-ID/quickly-pick

# Deploy
gcloud run deploy quickly-pick \
  --image gcr.io/PROJECT-ID/quickly-pick \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars DATABASE_URL="postgres://..."
```

## Database Setup

### PostgreSQL Configuration

#### Production Settings

```sql
-- postgresql.conf
shared_preload_libraries = 'pg_stat_statements'
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
```

#### Create Database and User

```sql
-- As postgres superuser
CREATE USER quicklypick WITH PASSWORD 'secure_password';
CREATE DATABASE quicklypick OWNER quicklypick;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE quicklypick TO quicklypick;
```

#### Connection Pooling (Optional)

Use PgBouncer for connection pooling:

```ini
# pgbouncer.ini
[databases]
quicklypick = host=localhost port=5432 dbname=quicklypick

[pgbouncer]
listen_port = 6432
listen_addr = 127.0.0.1
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 100
default_pool_size = 20
```

Update DATABASE_URL to use port 6432.

## Reverse Proxy Setup

### Nginx

```nginx
# /etc/nginx/sites-available/quickly-pick
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:3318;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /health {
        proxy_pass http://127.0.0.1:3318/health;
        access_log off;
    }
}
```

Enable the site:
```bash
sudo ln -s /etc/nginx/sites-available/quickly-pick /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Caddy

```caddyfile
# Caddyfile
yourdomain.com {
    reverse_proxy localhost:3318
    
    # Health check endpoint
    handle /health {
        reverse_proxy localhost:3318
        log {
            output discard
        }
    }
}
```

## SSL/TLS Setup

### Let's Encrypt with Certbot

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d yourdomain.com

# Auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

### Caddy (Automatic HTTPS)

Caddy automatically handles SSL certificates:

```bash
sudo caddy start --config /etc/caddy/Caddyfile
```

## Monitoring and Logging

### Health Checks

Set up monitoring for the `/health` endpoint:

```bash
# Simple health check script
#!/bin/bash
if curl -f http://localhost:3318/health > /dev/null 2>&1; then
    echo "Service is healthy"
    exit 0
else
    echo "Service is unhealthy"
    exit 1
fi
```

### Log Management

#### Structured Logging

The application uses structured JSON logging. Configure log rotation:

```bash
# /etc/logrotate.d/quickly-pick
/var/log/quickly-pick/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 quickly-pick quickly-pick
    postrotate
        systemctl reload quickly-pick
    endscript
}
```

#### Centralized Logging

For production, consider:
- **ELK Stack** (Elasticsearch, Logstash, Kibana)
- **Grafana Loki**
- **Cloud logging** (AWS CloudWatch, GCP Logging)

### Metrics and Alerting

#### Prometheus Metrics

Add metrics endpoint to your application:

```go
// Add to router
http.Handle("/metrics", promhttp.Handler())
```

#### Basic Alerts

Set up alerts for:
- Service downtime (health check failures)
- High response times
- Database connection errors
- High memory/CPU usage

## Security Considerations

### Application Security

```bash
# Run as non-root user
USER=quickly-pick

# Limit file permissions
chmod 600 .env
chown quickly-pick:quickly-pick .env

# Use secrets management
# - AWS Secrets Manager
# - HashiCorp Vault
# - Kubernetes Secrets
```

### Database Security

```sql
-- Limit database permissions
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO quicklypick;
GRANT CREATE ON SCHEMA public TO quicklypick;

-- Enable SSL
ALTER SYSTEM SET ssl = on;
SELECT pg_reload_conf();
```

### Network Security

- Use firewall rules to limit access
- Enable fail2ban for SSH protection
- Use VPC/private networks when possible
- Implement rate limiting at proxy level

## Backup Strategy

### Database Backups

```bash
#!/bin/bash
# backup-db.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/quickly-pick"
mkdir -p $BACKUP_DIR

pg_dump $DATABASE_URL | gzip > $BACKUP_DIR/backup_$DATE.sql.gz

# Keep only last 30 days
find $BACKUP_DIR -name "backup_*.sql.gz" -mtime +30 -delete
```

### Automated Backups

```bash
# Add to crontab
0 2 * * * /usr/local/bin/backup-db.sh
```

## Performance Optimization

### Database Optimization

```sql
-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM poll WHERE share_slug = 'example';

-- Update statistics
ANALYZE;

-- Monitor slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;
```

### Application Optimization

- Enable HTTP/2 in reverse proxy
- Use connection pooling
- Implement caching for read-heavy endpoints
- Optimize database queries
- Use CDN for static assets

## Troubleshooting

### Common Issues

#### High Memory Usage
```bash
# Check memory usage
free -h
ps aux --sort=-%mem | head

# Adjust Go garbage collector
export GOGC=100  # Default is 100
```

#### Database Connection Issues
```bash
# Check connection limits
SELECT count(*) FROM pg_stat_activity;
SHOW max_connections;

# Monitor connections
SELECT state, count(*) FROM pg_stat_activity GROUP BY state;
```

#### Slow Queries
```sql
-- Enable slow query logging
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- 1 second
SELECT pg_reload_conf();

-- Check slow queries
SELECT query, mean_time, calls FROM pg_stat_statements 
WHERE mean_time > 1000 ORDER BY mean_time DESC;
```

### Emergency Procedures

#### Service Recovery
```bash
# Restart service
sudo systemctl restart quickly-pick

# Check logs
sudo journalctl -u quickly-pick -f

# Rollback deployment
git checkout previous-version
sudo systemctl restart quickly-pick
```

#### Database Recovery
```bash
# Restore from backup
gunzip -c backup_20241030_020000.sql.gz | psql $DATABASE_URL
```

## Scaling Considerations

### Horizontal Scaling

- Use load balancer (nginx, HAProxy, cloud LB)
- Ensure stateless application design
- Consider database read replicas
- Implement distributed caching (Redis)

### Vertical Scaling

- Monitor CPU, memory, and I/O usage
- Scale database resources first
- Add more application instances as needed

This deployment guide provides a solid foundation for running Quickly Pick in production. Adjust configurations based on your specific requirements and infrastructure.