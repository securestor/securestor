# Docker Compose Deployment Guide

This guide covers deploying SecureStor using Docker Compose for development, testing, and small-scale production environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Basic Deployment](#basic-deployment)
- [High Availability Setup](#high-availability-setup)
- [Configuration](#configuration)
- [SSL/TLS Setup](#ssltls-setup)
- [Backup & Restore](#backup--restore)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

**Minimum (Development/Testing):**
- 2 CPU cores
- 4 GB RAM
- 20 GB disk space
- Docker 20.10+
- Docker Compose 2.0+

**Recommended (Production):**
- 4+ CPU cores
- 16 GB RAM
- 100 GB disk space (SSD recommended)
- Docker 20.10+
- Docker Compose 2.0+

### Operating System Support

- Ubuntu 20.04+ (Recommended)
- Debian 11+
- CentOS/RHEL 8+
- macOS 12+ (Development only)
- Windows 10/11 with WSL2 (Development only)

### Network Requirements

- Ports 80 and 443 (HTTP/HTTPS)
- Port 8080 (API - can be internal)
- Port 3000 (Frontend - can be internal)
- Port 5432 (PostgreSQL - internal only)
- Port 6379 (Redis - internal only)

## Basic Deployment

### Step 1: Clone Repository

```bash
git clone https://github.com/securestor/securestor.git
cd securestor
```

### Step 2: Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit configuration
nano .env
```

**Minimum Required Configuration:**

```bash
# Database
POSTGRES_USER=securestor
POSTGRES_PASSWORD=CHANGE_ME_STRONG_PASSWORD
POSTGRES_DB=securestor
DATABASE_URL=postgres://securestor:CHANGE_ME_STRONG_PASSWORD@postgres:5432/securestor?sslmode=disable

# Redis
REDIS_PASSWORD=CHANGE_ME_STRONG_PASSWORD
REDIS_URL=redis://:CHANGE_ME_STRONG_PASSWORD@redis:6379

# JWT Authentication
JWT_SECRET=CHANGE_ME_RANDOM_64_CHAR_STRING

# Encryption
ENCRYPTION_ENABLED=true
ENCRYPTION_KEY=CHANGE_ME_32_BYTE_HEX_KEY

# Environment
ENVIRONMENT=production
LOG_LEVEL=info

# Frontend
FRONTEND_URL=http://your-domain.com
API_BASE_URL=http://your-domain.com
```

### Step 3: Generate Secure Credentials

```bash
# Generate PostgreSQL password
echo "POSTGRES_PASSWORD=$(openssl rand -base64 32)"

# Generate Redis password
echo "REDIS_PASSWORD=$(openssl rand -base64 32)"

# Generate JWT secret
echo "JWT_SECRET=$(openssl rand -base64 64)"

# Generate encryption key
echo "ENCRYPTION_KEY=$(openssl rand -hex 32)"
```

### Step 4: Start Services

```bash
# Pull latest images
docker-compose pull

# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f
```

### Step 5: Verify Deployment

```bash
# Check API health
curl http://localhost:8080/api/v1/health

# Expected response:
# {"status":"healthy","timestamp":"2026-01-07T10:00:00Z"}

# Check database connectivity
curl http://localhost:8080/api/v1/health/db

# Check Redis connectivity
curl http://localhost:8080/api/v1/health/cache
```

### Step 6: Access the Application

1. Open browser: `http://localhost:3000`
2. Login with default credentials:
   - Username: `admin`
   - Password: `admin123`
   - Tenant: `admin`
3. **Change the default password immediately!**

## High Availability Setup

For production environments, use the HA profile with Redis Sentinel and PostgreSQL replication.

### HA Architecture

```
┌──────────────┐
│   Nginx LB   │  (Load Balancer)
└──────┬───────┘
       │
       ├─────────┬─────────┬─────────┐
       ▼         ▼         ▼         ▼
   ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
   │ API-1│ │ API-2│ │ API-3│ │ UI   │
   └───┬──┘ └───┬──┘ └───┬──┘ └──────┘
       │        │        │
       └────────┴────────┘
                │
       ┌────────┴────────┐
       ▼                 ▼
┌─────────────┐   ┌─────────────┐
│ PostgreSQL  │   │   Redis     │
│  Primary +  │   │  Sentinel   │
│  Replicas   │   │   Cluster   │
└─────────────┘   └─────────────┘
```

### Deploy with HA

```bash
# Start with HA profile
docker-compose --profile ha up -d

# This starts:
# - 3 API instances
# - PostgreSQL primary + 2 replicas
# - Redis master + 2 replicas
# - 3 Redis Sentinel instances
# - Nginx load balancer
```

### HA Configuration

Edit `docker-compose.yml` to customize HA settings:

```yaml
# API scaling
services:
  api:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

## Configuration

### Environment Variables Reference

#### Database Configuration

```bash
# Connection
DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=require
DB_MAX_CONNECTIONS=100
DB_MAX_IDLE_CONNECTIONS=25
DB_CONNECTION_LIFETIME=300

# Backup
DB_BACKUP_ENABLED=true
DB_BACKUP_SCHEDULE="0 2 * * *"  # Daily at 2 AM
DB_BACKUP_RETENTION_DAYS=30
```

#### Redis Configuration

```bash
# Connection
REDIS_URL=redis://:password@host:6379/0
REDIS_SENTINEL_ENABLED=true
REDIS_SENTINEL_MASTER=mymaster
REDIS_SENTINEL_ADDRESSES=sentinel1:26379,sentinel2:26379,sentinel3:26379

# Cache TTL
REDIS_CACHE_TTL=3600
REDIS_SESSION_TTL=86400
```

#### Storage Configuration

```bash
# Storage backend
STORAGE_TYPE=filesystem  # or s3, gcs, azure
STORAGE_PATH=/data/artifacts

# Erasure coding
ERASURE_DATA_SHARDS=8
ERASURE_PARITY_SHARDS=4

# Limits
MAX_ARTIFACT_SIZE=5GB
STORAGE_QUOTA_PER_TENANT=1TB
```

#### Security Configuration

```bash
# Authentication
AUTH_ENABLED=true
AUTH_TYPE=jwt  # or oidc, oauth2
API_KEY_ENABLED=true

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=7d

# Encryption
ENCRYPTION_ENABLED=true
ENCRYPTION_KEY=32-byte-hex-key
ENCRYPTION_ALGORITHM=AES-256-GCM

# TLS
TLS_ENABLED=true
TLS_CERT_PATH=/certs/server.crt
TLS_KEY_PATH=/certs/server.key
```

#### Scanner Configuration

```bash
# Scanning
SCANNER_ENABLED=true
SCANNER_ON_UPLOAD=true
SCANNER_CONCURRENT_SCANS=5
SCANNER_TIMEOUT=300

# Policies
SCANNER_BLOCK_CRITICAL=true
SCANNER_BLOCK_HIGH=false
SCANNER_MAX_CVSS_SCORE=7.0

# Cache
SCANNER_CACHE_ENABLED=true
SCANNER_CACHE_TTL=86400
```

## SSL/TLS Setup

### Using Let's Encrypt (Recommended)

```bash
# Install certbot
sudo apt-get install certbot

# Generate certificates
sudo certbot certonly --standalone \
  -d registry.yourcompany.com \
  --email admin@yourcompany.com \
  --agree-tos

# Copy certificates
sudo cp /etc/letsencrypt/live/registry.yourcompany.com/fullchain.pem ./certs/server.crt
sudo cp /etc/letsencrypt/live/registry.yourcompany.com/privkey.pem ./certs/server.key

# Set permissions
sudo chown $USER:$USER ./certs/*
chmod 600 ./certs/server.key
```

### Using Self-Signed Certificates (Testing Only)

```bash
# Create certs directory
mkdir -p certs

# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/server.key \
  -out certs/server.crt \
  -days 365 -nodes \
  -subj "/CN=registry.yourcompany.com"

# Set permissions
chmod 600 certs/server.key
```

### Update Docker Compose for TLS

```yaml
services:
  nginx:
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./certs:/etc/nginx/certs:ro
    environment:
      - TLS_ENABLED=true
```

### Configure Nginx for TLS

Create `configs/nginx-ssl.conf`:

```nginx
server {
    listen 80;
    server_name registry.yourcompany.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name registry.yourcompany.com;

    ssl_certificate /etc/nginx/certs/server.crt;
    ssl_certificate_key /etc/nginx/certs/server.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    client_max_body_size 5G;

    location / {
        proxy_pass http://frontend:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api {
        proxy_pass http://api:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Backup & Restore

### Automated Backup Script

Create `scripts/backup.sh`:

```bash
#!/bin/bash
set -e

BACKUP_DIR="/data/backups"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Create backup directory
mkdir -p $BACKUP_DIR

# Backup PostgreSQL
echo "Backing up database..."
docker exec securestor-postgres-1 pg_dump -U securestor securestor \
  | gzip > $BACKUP_DIR/db_${DATE}.sql.gz

# Backup artifacts
echo "Backing up artifacts..."
tar -czf $BACKUP_DIR/artifacts_${DATE}.tar.gz /data/artifacts

# Backup configuration
echo "Backing up configuration..."
tar -czf $BACKUP_DIR/config_${DATE}.tar.gz .env configs/

# Remove old backups
echo "Cleaning old backups..."
find $BACKUP_DIR -name "*.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $BACKUP_DIR"
```

### Schedule Automated Backups

```bash
# Make script executable
chmod +x scripts/backup.sh

# Add to crontab
crontab -e

# Add this line for daily backups at 2 AM
0 2 * * * /path/to/securestor/scripts/backup.sh >> /var/log/securestor-backup.log 2>&1
```

### Restore from Backup

```bash
#!/bin/bash
# scripts/restore.sh

BACKUP_DATE=$1

if [ -z "$BACKUP_DATE" ]; then
  echo "Usage: $0 <backup_date>"
  echo "Example: $0 20260107_020000"
  exit 1
fi

BACKUP_DIR="/data/backups"

# Stop services
docker-compose down

# Restore database
echo "Restoring database..."
gunzip < $BACKUP_DIR/db_${BACKUP_DATE}.sql.gz | \
  docker exec -i securestor-postgres-1 psql -U securestor securestor

# Restore artifacts
echo "Restoring artifacts..."
tar -xzf $BACKUP_DIR/artifacts_${BACKUP_DATE}.tar.gz -C /

# Restore configuration
echo "Restoring configuration..."
tar -xzf $BACKUP_DIR/config_${BACKUP_DATE}.tar.gz

# Start services
docker-compose up -d

echo "Restore completed!"
```

## Monitoring

### Health Check Endpoints

```bash
# System health
curl http://localhost:8080/api/v1/health

# Component health
curl http://localhost:8080/api/v1/health/db
curl http://localhost:8080/api/v1/health/cache
curl http://localhost:8080/api/v1/health/storage
curl http://localhost:8080/api/v1/scanners/health
```

### Docker Health Checks

```bash
# Check container health
docker-compose ps

# View container logs
docker-compose logs -f api
docker-compose logs -f postgres
docker-compose logs -f redis

# Check resource usage
docker stats
```

### Prometheus Metrics

Add Prometheus to `docker-compose.yml`:

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./configs/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    volumes:
      - grafana_data:/var/lib/grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## Troubleshooting

### Common Issues

#### Issue: Containers won't start

```bash
# Check logs
docker-compose logs

# Check for port conflicts
sudo netstat -tulpn | grep -E ':(80|443|3000|5432|6379|8080)'

# Remove and recreate
docker-compose down -v
docker-compose up -d
```

#### Issue: Database connection failed

```bash
# Check PostgreSQL logs
docker-compose logs postgres

# Test connection
docker exec -it securestor-postgres-1 psql -U securestor -d securestor

# Reset database
docker-compose down postgres
docker volume rm securestor_postgres_data
docker-compose up -d postgres
```

#### Issue: Redis connection failed

```bash
# Check Redis logs
docker-compose logs redis

# Test connection
docker exec -it securestor-redis-1 redis-cli -a YOUR_PASSWORD ping

# Check Sentinel status
docker exec -it securestor-sentinel-1 redis-cli -p 26379 sentinel masters
```

#### Issue: High memory usage

```bash
# Check resource usage
docker stats

# Set memory limits in docker-compose.yml
services:
  api:
    deploy:
      resources:
        limits:
          memory: 4G
```

#### Issue: Slow performance

```bash
# Enable query caching
CACHE_QUERY_RESULTS=true

# Increase database connections
DB_MAX_CONNECTIONS=100

# Add more API instances
docker-compose up -d --scale api=3
```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug

# Restart services
docker-compose restart

# View detailed logs
docker-compose logs -f --tail=100
```

### Getting Help

- **Documentation**: https://docs.securestor.io
- **GitHub Issues**: https://github.com/securestor/securestor/issues
- **Discord**: https://discord.gg/securestor
- **Email**: support@securestor.io

---

**Next Steps:**
- [Kubernetes Deployment Guide](kubernetes-deployment.md)
- [Production Hardening Guide](production-hardening.md)
- [Monitoring & Observability](monitoring.md)