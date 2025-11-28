# SecureStore Deployment Guide

This guide explains how to deploy SecureStore in different configurations using the unified `docker-compose.yml` file.

## Deployment Modes

The `docker-compose.yml` file now supports two deployment modes:

### 1. Simple Deployment (Default)
Single instance of each service - suitable for development and testing.

### 2. High Availability Deployment
Multiple instances with replication, load balancing, and failover - suitable for production.

---

## Simple Deployment

### Start Services
```bash
docker-compose up -d
```

This starts:
- 1x PostgreSQL database
- 1x Redis cache
- 1x API server
- 1x OPA policy engine
- 1x Keycloak (with dedicated PostgreSQL)

### Access Points
- **API**: http://localhost:8080
- **Keycloak**: http://localhost:8090
- **OPA**: http://localhost:8181
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379

### Stop Services
```bash
docker-compose down
```

### Remove Volumes (Clean Reset)
```bash
docker-compose down -v
```

---

## High Availability Deployment

### Start Services
```bash
docker-compose --profile ha up -d
```

This starts:
- 3x Storage nodes (replicated)
- 1x PostgreSQL Primary + 1x PostgreSQL Standby
- 1x Redis Master + 2x Redis Replicas
- 3x Redis Sentinels (for automatic failover)
- 3x API instances (load balanced)
- 1x Nginx load balancer
- 1x OPA policy engine
- 1x Keycloak (with dedicated PostgreSQL)

### Architecture

```
┌─────────────┐
│   Nginx LB  │ :80, :443
└──────┬──────┘
       │
  ┌────┴────┬────────┐
  │         │        │
┌─▼──┐  ┌──▼─┐  ┌──▼─┐
│API1│  │API2│  │API3│
└────┘  └────┘  └────┘
  │      │       │
  └──────┼───────┘
         │
    ┌────┴─────────────┬──────────────┐
    │                  │              │
┌───▼────┐      ┌──────▼───┐   ┌─────▼────┐
│Postgres│      │  Redis   │   │ Storage  │
│ Primary│◄────►│  Master  │   │ Nodes    │
└────┬───┘      └────┬─────┘   │(3-way    │
     │               │          │replica)  │
┌────▼───┐      ┌────▼─────┐   └──────────┘
│Postgres│      │  Redis   │
│Standby │      │ Replicas │
└────────┘      │& Sentinels│
                └───────────┘
```

### Access Points
- **Load Balancer**: http://localhost (port 80)
- **Load Balancer HTTPS**: https://localhost (port 443)
- **API Instance 1**: Internal only
- **API Instance 2**: Internal only
- **API Instance 3**: Internal only
- **PostgreSQL Primary**: localhost:5432
- **PostgreSQL Standby**: localhost:5433
- **Redis Master**: localhost:6379
- **Redis Replica 1**: localhost:6380
- **Redis Replica 2**: localhost:6381
- **Redis Sentinel 1**: localhost:26379
- **Redis Sentinel 2**: localhost:26380
- **Redis Sentinel 3**: localhost:26381
- **Keycloak**: http://localhost:8090
- **OPA**: http://localhost:8181

### Stop Services
```bash
docker-compose --profile ha down
```

### Remove Volumes (Clean Reset)
```bash
docker-compose --profile ha down -v
```

---

## Configuration Files Required for HA

Before starting HA deployment, ensure these files exist:

### 1. Nginx Configuration
Create `configs/nginx-ha.conf`:
```nginx
events {
    worker_connections 1024;
}

http {
    upstream api_backend {
        least_conn;
        server api-1:8080 max_fails=3 fail_timeout=30s;
        server api-2:8080 max_fails=3 fail_timeout=30s;
        server api-3:8080 max_fails=3 fail_timeout=30s;
    }

    server {
        listen 80;
        
        location / {
            proxy_pass http://api_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
        
        location /health {
            access_log off;
            return 200 "healthy\n";
            add_header Content-Type text/plain;
        }
    }
}
```

### 2. Redis Sentinel Configurations

Create `configs/sentinel-1.conf`:
```conf
port 26379
sentinel monitor mymaster redis-master 6379 2
sentinel auth-pass mymaster redis123
sentinel down-after-milliseconds mymaster 5000
sentinel parallel-syncs mymaster 1
sentinel failover-timeout mymaster 10000
```

Create `configs/sentinel-2.conf` and `configs/sentinel-3.conf` with the same content.

### 3. PostgreSQL Replication Scripts

Create `scripts/postgres_setup.sh`:
```bash
#!/bin/bash
# Add replication user and configuration for primary
```

Create `scripts/postgres_standby_setup.sh`:
```bash
#!/bin/bash
# Configure standby to replicate from primary
```

---

## Switching Between Modes

### From Simple to HA
```bash
# Stop simple deployment
docker-compose down

# Start HA deployment
docker-compose --profile ha up -d
```

### From HA to Simple
```bash
# Stop HA deployment
docker-compose --profile ha down

# Start simple deployment
docker-compose up -d
```

---

## Environment Variables

Both modes support the same environment variables. Create a `.env` file:

```env
# Database
DATABASE_URL=postgresql://securestor:securestor123@postgres:5432/securestor?sslmode=disable

# Redis
REDIS_URL=redis://:redis123@redis:6379/0

# Security
JWT_SECRET=your-secret-key-change-in-production

# Storage
STORAGE_PATH=/app/storage
MAX_FILE_SIZE=524288000

# Scanning
ENABLE_SECURITY_SCANNING=true
SCANNER_TIMEOUT=600

# OPA
OPA_URL=http://opa:8181
OPA_ENABLED=true
```

---

## Monitoring

### View Logs

**Simple deployment:**
```bash
docker-compose logs -f api
```

**HA deployment:**
```bash
# All API instances
docker-compose --profile ha logs -f api-1 api-2 api-3

# Specific instance
docker-compose --profile ha logs -f api-1

# Nginx load balancer
docker-compose --profile ha logs -f nginx-lb
```

### Health Checks

**Simple deployment:**
```bash
curl http://localhost:8080/api/v1/health/live
```

**HA deployment:**
```bash
# Via load balancer
curl http://localhost/api/v1/health/live

# Check load balancer
curl http://localhost/health
```

---

## Troubleshooting

### Check Service Status
```bash
# Simple
docker-compose ps

# HA
docker-compose --profile ha ps
```

### Check Specific Service Health
```bash
# Simple
docker-compose exec api curl -f http://localhost:8080/api/v1/health/live

# HA
docker-compose --profile ha exec api-1 curl -f http://localhost:8080/api/v1/health/live
```

### Restart a Service
```bash
# Simple
docker-compose restart api

# HA
docker-compose --profile ha restart api-1
```

### View Resource Usage
```bash
docker stats
```

---

## Production Recommendations

For production HA deployment:

1. **Use external secrets management** (not hardcoded passwords)
2. **Enable SSL/TLS** on Nginx load balancer
3. **Set up monitoring** (Prometheus/Grafana)
4. **Configure backup strategies** for PostgreSQL and Redis
5. **Use persistent volumes** with proper backup policies
6. **Set resource limits** in docker-compose.yml
7. **Enable log aggregation** (ELK/Loki)
8. **Configure alerts** for service failures
9. **Test failover scenarios** regularly
10. **Document recovery procedures**

---

## Migration Path

### Development → Staging → Production

1. **Development**: Use simple deployment
2. **Staging**: Use HA deployment to test failover
3. **Production**: Use HA deployment with production-grade configurations

---


## Support

For issues or questions, please check the main README.md or open an issue in the repository.
