# SecureStor

<div align="center">

![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)
![Build Status](https://img.shields.io/github/actions/workflow/status/securestor/securestor/ci.yml?branch=main)
![Go Version](https://img.shields.io/github/go-mod/go-version/securestor/securestor)
![GitHub Stars](https://img.shields.io/github/stars/securestor/securestor?style=social)
![Version](https://img.shields.io/github/v/release/securestor/securestor)

**Enterprise-Grade Artifact Repository with Built-in Security & Compliance**

[Features](#-key-features) • [Quick Start](#-quick-start) • [Documentation](https://docs.securestor.io) • [Enterprise](#enterprise-edition)

</div>

---

> ⚠️ **BETA VERSION** - SecureStor is currently in beta. While stable for testing and development environments, we recommend thorough evaluation before production deployment. Community feedback and contributions are welcome!

## 🎯 What is SecureStor?

SecureStor is an **all-in-one artifact repository platform** that combines secure storage, automated vulnerability scanning, and compliance management in a single solution. Built for DevSecOps teams who need to manage artifacts without sacrificing security.

**Perfect for:**
- 🏢 **DevOps Teams** managing multiple artifact types (Docker, npm, Maven, PyPI, Helm)
- 🔒 **Security Teams** requiring automated vulnerability scanning and policy enforcement
- 📋 **Compliance Officers** needing audit trails and compliance reporting
- 🚀 **Startups to Enterprises** seeking cost-effective, self-hosted artifact management

**Why SecureStor?**
- ✅ **One Platform, Multiple Formats** - No need for separate tools for Docker, npm, Maven, etc.
- 🔍 **Security Built-In** - Every artifact is automatically scanned for vulnerabilities
- 📊 **Compliance Ready** - Audit logs, policy enforcement, and reports out-of-the-box
- 💰 **Cost Effective** - Open source with optional enterprise features
- ⚡ **High Performance** - Erasure coding, intelligent caching, and HA support

## 🌟 Editions

SecureStor is available in two editions:

### Community Edition (Open Source)
- ✅ Multi-format artifact management (Docker, npm, Maven, PyPI, Helm)
- ✅ Automated security scanning with vulnerability detection
- ✅ Repository management with proxy caching
- ✅ API key authentication
- ✅ Audit logging and activity tracking
- ✅ User profile
- ✅ RESTful API with comprehensive documentation
- ✅ High-performance storage with erasure coding
- 📖 Open source under AGPL-3.0 license

### Enterprise Edition
All Community features, plus:
- 🏢 **Multi-tenancy** with complete tenant isolation
- 👥 **User & Role Management** with granular RBAC
- ✅ **Compliance Management** with policy enforcement
- 🔄 **Advanced Replication** across multiple regions
- 💾 **Intelligent Cache Management** with optimization
- ⚙️ **Advanced Tenant Settings** and customization
- 🎫 Priority support and SLA guarantees

For Enterprise edition inquiries: sales@securestor.io

## 🚀 Key Features

### Artifact Management
- **Multi-Format Support**: Docker, npm, Maven, PyPI, Helm, Generic artifacts
- **OCI & Registry Compliance**: Full Docker Registry v2 and npm registry compatibility
- **High Availability**: Redis Sentinel clustering with automatic failover
- **Erasure Coding**: Configurable data redundancy (8+4, 16+8 schemes)
- **Metadata Indexing**: Advanced search and filtering capabilities

### Security & Compliance
- **Automated Scanning**: Integrated vulnerability detection using OWASP dep-scan, Blint, Grype
- **Real-time Alerts**: Immediate notification of critical vulnerabilities
- **Compliance Auditing**: Built-in policy enforcement and audit trails
- **License Management**: Automatic license detection and compliance checking
- **Supply Chain Security**: Dependency analysis and risk assessment

### Enterprise Features
- **Multi-tenancy**: Complete tenant isolation with RBAC
- **SSO Integration**: OIDC/OAuth2 support via Keycloak
- **API Key Management**: Scoped access tokens with granular permissions
- **Encryption**: End-to-end encryption with configurable key management
- **Replication**: Multi-region artifact replication with configurable sync
- **Audit Logging**: Comprehensive activity tracking and compliance reporting

### Proxy & Caching
- **Remote Proxies**: Cache artifacts from Docker Hub, npm, Maven Central, PyPI
- **Intelligent Caching**: Automatic background scanning of cached packages
- **Bandwidth Optimization**: Reduce external registry dependencies
- **Offline Mode**: Continue operations during network outages

## � Quick Start

Get SecureStor running in under 5 minutes with Docker Compose.

### Prerequisites
- Docker 20.10+ and Docker Compose 2.0+
- 4GB RAM minimum (8GB recommended)
- 20GB disk space

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/securestor/securestor.git
cd securestor

# 2. Start all services (PostgreSQL, Redis, API, Frontend)
docker-compose up -d

# 3. Verify deployment
curl http://localhost:8080/api/v1/health
# Expected: {"status":"healthy"}

# 4. Access the UI
open http://localhost:3000
```

**Default Login Credentials:**
```
URL:      http://localhost:3000
Username: admin
Password: admin123
Tenant:   admin
```
```bash
# Clone repository
git clone https://github.com/securestor/securestor.git
cd securestor

# Configure production environment
cp .env.example .env
# Edit .env with production settings (see below)

# Start with High Availability
docker-compose --profile ha up -d

# Check logs to see default admin credentials
docker-compose logs api | grep "DEFAULT CREDENTIALS"
```

### 🎉 Automatic First-Time Setup

On first startup, SecureStor automatically creates:
- ✅ Default admin tenant (`admin`)
- ✅ Admin user (username: `admin`, password: `admin123`)
- ✅ 6 default roles (admin, developer, viewer, scanner, auditor, deployer)
- ✅ 28 granular permissions
- ✅ 11 OAuth2 scopes for API key authentication

⚠️ **IMPORTANT**: Change the default password after first login. A warning banner will appear on the dashboard until
# Pull the image
docker pull localhost:8080/myapp:latest
```

#### Using as npm Registry

```bash
# Configure npm
npm config set registry http://localhost:8080/npm

# Login
npm login --registry=http://localhost:8080/npm

# Publish a package
npm publish

# Install packages
npm install express
```

#### Using as Maven Repository

```xml
<!-- Add to pom.xml -->
<repositories>
  <repository>
    <id>securestor</id>
    <url>http://localhost:8080/maven</url>
  </repository>
</repositories>
```

#### Using as PyPI Repository

```bash
# Configure pip
pip config set global.index-url http://localhost:8080/pypi/simple

# Install packages
pip install requests
```

### Next Steps

- 📚 Read the [Full Documentation](https://docs.securestor.io)
- 🔐 Set up [Authentication & Security](#-security-configuration)
- 🏢 Explore [Enterprise Features](#enterprise-edition)
- 🚀 Configure [High Availability](#-architecture)

---

## 📋 Production Deployment

### Production Prerequisites
- Docker & Docker Compose
- PostgreSQL 14+
- Redis 7+ (Sentinel for HA)
- SSL/TLS certificates
- 16GB RAM minimum
- 100GB+ disk space

### Production Installation

### 🎉 Automatic First-Time Setup

On first startup, SecureStor automatically creates:
- ✅ Default admin tenant (`admin`)
- ✅ Admin user with username `admin` and password `admin123`
- ✅ 6 default roles (admin, developer, viewer, scanner, auditor, deployer)
- ✅ 28 granular permissions
- ✅ 11 OAuth2 scopes for API key authentication

**Default Login Credentials:**)

If you need to recreate the setup or run on existing databases:

```bash
# 1. Create admin user and default tenant (optional - runs automatically)
./scripts/setup_admin.sh
```bash
### Manual Setup Scripts (Optional - Legacy)

If you need to recreate the setup or run on existing databases:

```bash
# 1. Create admin user and default tenant (optional - runs automatically)
./scripts/setup_admin.sh
# You'll be prompted for username, email, and password

# 2. Populate OAuth2 scopes (optional - runs automatically)
./scripts/populate_scopes.sh

# 3. Verify admin user
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","tenant":"admin"}'
```

### Production Security Setup

Generate strong credentials before deployment:

```bash
# Generate strong passwords
openssl rand -base64 32  # For PostgreSQL
openssl rand -base64 32  # For Redis

# Generate JWT secret
export JWT_SECRET=$(openssl rand -base64 64)

# Generate encryption keys
export ENCRYPTION_KEY=$(openssl rand -hex 32)

# Update .env file with generated values
```

### Production Checklist

Before deploying to production, ensure:

✅ **Security**
- Strong, randomly generated passwords for all services
- SSL/TLS certificates configured
- Firewall rules configured (allow only 22, 80, 443)
- JWT_SECRET and ENCRYPTION_KEY set to secure random values

✅ **Configuration**
- `.env` file configured with production values
- `LOG_LEVEL=info` (not debug)
- `ENVIRONMENT=production`
- Database backups configured
- Log rotation configured

❌ **Avoid**
- Using default passwords from examples
- Exposing database ports externally (5432, 6379)
- Running without SSL/TLS certificates
- Using `latest` Docker image tags


### Post-Deployment

```bash
# Configure automated database backups
cat > /etc/cron.daily/securestor-backup <<'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker exec securestor-postgres-primary-1 \
  pg_dump -U securestor securestor > /data/backups/securestor_${DATE}.sql
gzip /data/backups/securestor_${DATE}.sql
find /data/backups -name "*.sql.gz" -mtime +30 -delete
EOF
chmod +x /etc/cron.daily/securestor-backup

# Configure firewall
sudo ufw allow 22/tcp   # SSH
sudo ufw allow 80/tcp   # HTTP
sudo ufw allow 443/tcp  # HTTPS
sudo ufw enable
```


### Build Docker Images

```bash
# Build backend image
docker build -t securestor-api:latest -f Dockerfile .

# Build frontend image
docker build -t securestor-frontend:latest -f frontend/Dockerfile ./frontend

# Or build all services with docker-compose
docker-compose build
```

### Docker Registry Configuration

```bash
# Configure Docker to use SecureStor
# Add to /etc/docker/daemon.json:
{
  "insecure-registries": ["registry.yourcompany.com:8080"]
}

sudo systemctl restart docker

# Tag and push images
docker tag myapp:latest registry.yourcompany.com:8080/myapp:latest
docker push registry.yourcompany.com:8080/myapp:latest
```

### npm Registry Configuration

```bash
# Configure npm to use SecureStor
npm config set registry http://registry.yourcompany.com:8080/npm

# Authenticate (if auth enabled)
npm login --registry=http://registry.yourcompany.com:8080/npm

# Publish packages
npm publish
```

## 🔐 Security Configuration

### Authentication Setup

SecureStor supports multiple authentication methods:

```bash
# OIDC/OAuth2 via Keycloak (recommended for enterprise)
KEYCLOAK_ENABLED=true
KEYCLOAK_URL=https://keycloak.yourcompany.com
KEYCLOAK_REALM=securestor
KEYCLOAK_CLIENT_ID=securestor-api

# API Key authentication
API_KEY_ENABLED=true
API_KEY_HEADER=X-API-Key

# JWT authentication
JWT_SECRET=your-secure-secret-key
JWT_EXPIRATION=24h
```

### Encryption Configuration

```bash
# Enable artifact encryption
ENCRYPTION_ENABLED=true
ENCRYPTION_KEY=your-32-byte-encryption-key
ENCRYPTION_ALGORITHM=AES-256-GCM

# Key rotation
ENCRYPTION_KEY_ROTATION_DAYS=90
```

### Scanning Configuration

```bash
# Enable automatic scanning
SCANNER_ENABLED=true
SCANNER_ON_UPLOAD=true
SCANNER_CONCURRENT_SCANS=5

# Scanner thresholds
SCANNER_BLOCK_CRITICAL=true
SCANNER_BLOCK_HIGH=false
SCANNER_MAX_SCORE=7.0
```

## 🏗️ Architecture

### High Availability Setup

```
┌──────────────┐
│ Nginx LB     │
│ (Port 80/443)│
└──────┬───────┘
       │
       ├─────────┬─────────┐
       ▼         ▼         ▼
┌─────────┐ ┌─────────┐ ┌─────────┐
│ API-1   │ │ API-2   │ │ API-3   │
└────┬────┘ └────┬────┘ └────┬────┘
     │           │           │
     └───────────┴───────────┘
                 │
        ┌────────┴────────┐
        ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ PostgreSQL   │  │ Redis        │
│ Primary +    │  │ Sentinel     │
│ Replicas     │  │ Cluster      │
└──────────────┘  └──────────────┘
```

### Storage Architecture

- **Erasure Coding**: Configurable redundancy (8+4, 16+8)
- **Metadata Store**: PostgreSQL with replication
- **Cache Layer**: Redis for session and metadata caching
- **Blob Storage**: File system with optional S3/GCS backend

## 📊 Monitoring & Operations

### Health Checks

```bash
# System health
curl http://localhost:8080/api/v1/health

# Scanner health
curl http://localhost:8080/api/v1/scanners/health

# Database connectivity
curl http://localhost:8080/api/v1/health/db

# Redis connectivity
curl http://localhost:8080/api/v1/health/cache
```

### Metrics & Logging

```bash
# Enable Prometheus metrics
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090

# Configure structured logging
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=/var/log/securestor/app.log
```

### Backup & Recovery

```bash
# Database backup
pg_dump -h localhost -U securestor securestor > backup.sql

# Artifact backup (with metadata)
./bin/securestor backup --output /backup/artifacts --include-metadata

# Restore
./bin/securestor restore --input /backup/artifacts
```

## 🌐 Enterprise Features

### Multi-Tenancy
- **Tenant Isolation**: Complete separation of resources and access
- **Quota Management**: Storage and repository limits per tenant
- **Custom Branding**: Tenant-specific UI customization

### RBAC & Permissions
- **Tenant Admin**: Full control within tenant
- **Repository Manager**: Manage repositories and artifacts
- **Developer**: Push/pull artifacts
- **Auditor**: Read-only access with compliance reports
- **Scanner**: Automated scan operations

### Audit Logging
All operations are logged with user identity, timestamp, IP address, action performed, resource affected, and result status.

## 🚨 Compliance & Policies

### Policy Enforcement
- **Vulnerability Blocking**: Automatically block artifacts with critical vulnerabilities
- **License Compliance**: Enforce approved license policies
- **Retention Policies**: Automated artifact lifecycle management
- **Access Policies**: Fine-grained permission controls

### Compliance Reports
Generate comprehensive compliance reports with filtering by date range, status, and severity. Export to PDF or CSV formats.

## 📦 Repository Types

### Local Repositories
Store and manage artifacts directly in SecureStor with full encryption and replication support.

### Remote Proxies
Cache artifacts from external registries (Docker Hub, npm, Maven Central, PyPI) with automatic security scanning.

### Virtual Repositories
Aggregate multiple repositories (local and remote) into a single unified endpoint.

## 🔍 Advanced Features

### Metadata Search & Indexing
Advanced search with filtering by artifact type, severity, license, date range, tags, and custom metadata.

### Storage Management
- **Erasure Coding**: Configurable redundancy schemes (8+4, 16+8)
- **Garbage Collection**: Automatic cleanup of unused artifacts
- **Storage Statistics**: Real-time monitoring of disk usage
- **Quota Management**: Per-tenant storage limits

### Security Features
- **Automatic Scanning**: Scan artifacts on upload
- **Scanner Health Monitoring**: Track scanner availability and performance
- **Bulk Scanning**: Scan multiple artifacts simultaneously
- **Vulnerability Tracking**: Historical vulnerability records

## 🔧 Troubleshooting

### Common Issues

**Issue**: Slow scanning performance
```bash
# Increase concurrent scans
SCANNER_CONCURRENT_SCANS=10

# Add more scanner workers
SCANNER_WORKER_COUNT=5
```

**Issue**: High storage usage
```bash
# Enable garbage collection
./bin/securestor gc --older-than 90d

# Enable compression
STORAGE_COMPRESSION=true
```

**Issue**: Redis connection errors
```bash
# Check Redis Sentinel status
redis-cli -p 26379 sentinel masters

# Verify failover configuration
REDIS_SENTINEL_ENABLED=true
REDIS_SENTINEL_MASTER=mymaster
```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug

# Enable profiling
PPROF_ENABLED=true
PPROF_PORT=6060

# View profiles
go tool pprof http://localhost:6060/debug/pprof/heap
```

## 📈 Performance Tuning

### Database Optimization

```sql
-- Create indexes for common queries
CREATE INDEX idx_artifacts_name ON artifacts(name);
CREATE INDEX idx_artifacts_type ON artifacts(type);
CREATE INDEX idx_scans_artifact_id ON scans(artifact_id);
CREATE INDEX idx_vulnerabilities_severity ON vulnerabilities(severity);
```

### Caching Strategy

```bash
# Redis cache configuration
REDIS_CACHE_TTL=3600
REDIS_MAX_CONNECTIONS=100
REDIS_IDLE_TIMEOUT=300

# Enable query result caching
CACHE_QUERY_RESULTS=true
CACHE_METADATA=true
```

### Upload Optimization

```bash
# Enable multipart uploads
MULTIPART_UPLOAD_ENABLED=true
MULTIPART_CHUNK_SIZE_MB=10

# Configure concurrent uploads
MAX_CONCURRENT_UPLOADS=10
```

## 📄 License

SecureStor is licensed under the [AGPL-3.0 License](LICENSE).

For commercial licensing and enterprise support, contact: support@securestor.io

## 📖 Documentation

### Deployment Guides
- **[Docker Compose Deployment](docs/docker-deployment.md)** - Deploy with Docker Compose for development and small-scale production
- **[Kubernetes Local Setup](docs/kubernetes-local-setup.md)** - Local development with minikube (step-by-step guide)
- **[Kubernetes Deployment](docs/kubernetes-deployment.md)** - Production-grade Kubernetes deployment with HA
- **[Production Hardening](docs/production-hardening.md)** - Security hardening and best practices

### User Guides
- **[Getting Started](https://docs.securestor.io/getting-started)** - Quick start guide and basic usage
- **[API Documentation](https://docs.securestor.io/api)** - Complete API reference
- **[Security Guide](https://docs.securestor.io/security)** - Security features and configuration

## 🤝 Contributing

We welcome contributions from the community! Please read our guidelines before getting started.

**Quick Links:**
- **[Contributing Guide](CONTRIBUTING.md)** - Development setup, coding standards, and PR process
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community guidelines and expectations
- **[GitHub Issues](https://github.com/securestor/securestor/issues)** - Report bugs or request features
- **[GitHub Discussions](https://github.com/securestor/securestor/discussions)** - Ask questions and share ideas

**How to Contribute:**

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with clear commit messages
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Update documentation as needed
7. Push to your branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request with a clear description

**Good First Issues**: Look for issues labeled `good first issue` to get started!

## 💬 Support

- **Documentation**: https://docs.securestor.io
- **Issues**: https://github.com/securestor/securestor/issues
- **Discussions**: https://github.com/securestor/securestor/discussions
- **Community Chat**: Coming soon
- **Enterprise Support**: support@securestor.io
- **Security Issues**: security@securestor.io (private disclosure)

## 🗓️ Roadmap

- [ ] AI-driven data tiering optimization
- [ ] Immutable storage with WORM compliance
- [ ] Multi-cloud hybrid replication
- [ ] Advanced data provenance tracking
- [ ] Serverless workflow integration
- [ ] Hardware security module (HSM) integration
- [ ] Enhanced ML-based anomaly detection

---

**Built with ❤️ for the DevSecOps community**