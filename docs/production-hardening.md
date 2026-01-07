# Production Hardening Guide

This guide covers security hardening and production best practices for SecureStor deployments.

## Table of Contents

- [Security Baseline](#security-baseline)
- [Authentication & Authorization](#authentication--authorization)
- [Network Security](#network-security)
- [Data Protection](#data-protection)
- [System Hardening](#system-hardening)
- [Monitoring & Auditing](#monitoring--auditing)
- [Compliance](#compliance)
- [Incident Response](#incident-response)

## Security Baseline

### Pre-Deployment Security Checklist

- [ ] Change all default passwords
- [ ] Generate strong random secrets
- [ ] Enable TLS/SSL for all services
- [ ] Configure firewall rules
- [ ] Set up network segmentation
- [ ] Enable audit logging
- [ ] Configure backup strategy
- [ ] Set up monitoring and alerts
- [ ] Review and harden container images
- [ ] Implement secrets management
- [ ] Configure rate limiting
- [ ] Enable RBAC
- [ ] Set up vulnerability scanning
- [ ] Document security procedures

## Authentication & Authorization

### Multi-Factor Authentication (MFA)

Enable MFA for all administrative accounts:

```bash
# In .env or environment configuration
MFA_ENABLED=true
MFA_ISSUER=SecureStor
MFA_REQUIRED_FOR_ADMIN=true

# Supported MFA methods
MFA_METHODS=totp,webauthn
```

### OAuth2/OIDC Integration

Configure Keycloak or external identity provider:

```bash
# Keycloak configuration
KEYCLOAK_ENABLED=true
KEYCLOAK_URL=https://keycloak.yourcompany.com
KEYCLOAK_REALM=securestor
KEYCLOAK_CLIENT_ID=securestor-api
KEYCLOAK_CLIENT_SECRET=your-client-secret

# OIDC configuration
OIDC_ENABLED=true
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
OIDC_REDIRECT_URL=https://registry.yourcompany.com/auth/callback
```

### API Key Security

Best practices for API keys:

```bash
# API key configuration
API_KEY_ENABLED=true
API_KEY_EXPIRATION=90d
API_KEY_MAX_PER_USER=5
API_KEY_ROTATION_REQUIRED=true
API_KEY_ROTATION_DAYS=90

# Scope-based access control
API_KEY_SCOPES=read:artifacts,write:artifacts,delete:artifacts,scan:artifacts
```

### Role-Based Access Control (RBAC)

Configure granular permissions:

```yaml
# Example RBAC configuration
roles:
  admin:
    permissions:
      - "*:*"
  
  developer:
    permissions:
      - "read:artifacts"
      - "write:artifacts"
      - "read:repositories"
  
  auditor:
    permissions:
      - "read:artifacts"
      - "read:scans"
      - "read:audit-logs"
  
  scanner:
    permissions:
      - "read:artifacts"
      - "write:scans"
```

## Network Security

### Firewall Configuration

**Ubuntu/Debian (UFW):**

```bash
# Allow SSH (be careful!)
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Deny all other incoming
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status verbose
```

**RHEL/CentOS (firewalld):**

```bash
# Allow HTTP/HTTPS
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https

# Deny PostgreSQL/Redis from external
sudo firewall-cmd --permanent --remove-port=5432/tcp
sudo firewall-cmd --permanent --remove-port=6379/tcp

# Reload firewall
sudo firewall-cmd --reload
```

### TLS/SSL Configuration

**Strong TLS Configuration:**

```nginx
# Nginx TLS hardening
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
ssl_prefer_server_ciphers on;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 10m;
ssl_stapling on;
ssl_stapling_verify on;

# HSTS
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

# Security headers
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
```

### Rate Limiting

Protect against DDoS and brute force:

```nginx
# Nginx rate limiting
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;

server {
    location /api/v1/auth/login {
        limit_req zone=login burst=5 nodelay;
    }
    
    location /api {
        limit_req zone=api burst=200 nodelay;
    }
}
```

**Application-level rate limiting:**

```bash
# Environment configuration
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST=200
RATE_LIMIT_LOGIN_ATTEMPTS=5
RATE_LIMIT_LOGIN_WINDOW=300  # 5 minutes
```

### Network Segmentation

Use Docker networks or Kubernetes NetworkPolicies:

```yaml
# Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: securestor-network-policy
  namespace: securestor
spec:
  podSelector:
    matchLabels:
      app: securestor-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: securestor-frontend
    - podSelector:
        matchLabels:
          app: nginx-ingress
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

## Data Protection

### Encryption at Rest

**Database encryption:**

```sql
-- PostgreSQL transparent data encryption (TDE)
-- Requires PostgreSQL with encryption support

-- Enable encryption for tablespace
CREATE TABLESPACE encrypted_space
  LOCATION '/var/lib/postgresql/encrypted'
  WITH (encryption = true);

-- Use encrypted tablespace
ALTER TABLE artifacts SET TABLESPACE encrypted_space;
```

**Application-level encryption:**

```bash
# Environment configuration
ENCRYPTION_ENABLED=true
ENCRYPTION_ALGORITHM=AES-256-GCM
ENCRYPTION_KEY=<32-byte-hex-key>

# Key rotation
ENCRYPTION_KEY_ROTATION_ENABLED=true
ENCRYPTION_KEY_ROTATION_DAYS=90
```

### Encryption in Transit

Force TLS for all connections:

```bash
# PostgreSQL - require SSL
ssl = on
ssl_cert_file = '/etc/ssl/certs/server.crt'
ssl_key_file = '/etc/ssl/private/server.key'
ssl_ca_file = '/etc/ssl/certs/ca.crt'

# Require SSL for all connections
hostssl all all 0.0.0.0/0 md5

# Redis - require TLS
tls-port 6379
port 0
tls-cert-file /etc/redis/certs/redis.crt
tls-key-file /etc/redis/certs/redis.key
tls-ca-cert-file /etc/redis/certs/ca.crt
```

### Secrets Management

**Using HashiCorp Vault:**

```bash
# Enable Vault integration
VAULT_ENABLED=true
VAULT_ADDR=https://vault.yourcompany.com:8200
VAULT_TOKEN=<vault-token>
VAULT_MOUNT_PATH=secret/securestor

# Retrieve secrets from Vault
vault kv get secret/securestor/database
vault kv get secret/securestor/jwt
vault kv get secret/securestor/encryption
```

**Using Kubernetes Secrets with encryption:**

```bash
# Enable encryption at rest for Kubernetes secrets
# Add to kube-apiserver configuration:
--encryption-provider-config=/etc/kubernetes/encryption-config.yaml

# encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: <base64-encoded-32-byte-key>
    - identity: {}
```

### Backup Encryption

Encrypt all backups:

```bash
#!/bin/bash
# Encrypted backup script

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/secure/backups"
ENCRYPTION_KEY="/secure/backup.key"

# Database backup with encryption
pg_dump -U securestor securestor | \
  gzip | \
  openssl enc -aes-256-cbc -salt -pbkdf2 -pass file:$ENCRYPTION_KEY \
  > $BACKUP_DIR/db_${DATE}.sql.gz.enc

# Artifacts backup with encryption
tar -czf - /data/artifacts | \
  openssl enc -aes-256-cbc -salt -pbkdf2 -pass file:$ENCRYPTION_KEY \
  > $BACKUP_DIR/artifacts_${DATE}.tar.gz.enc

# Upload to S3 with server-side encryption
aws s3 cp $BACKUP_DIR/db_${DATE}.sql.gz.enc \
  s3://securestor-backups/ \
  --server-side-encryption AES256
```

## System Hardening

### Container Security

**Use minimal base images:**

```dockerfile
# Use distroless or alpine base
FROM gcr.io/distroless/static-debian11

# Or use Alpine with security updates
FROM alpine:3.18
RUN apk update && apk upgrade
```

**Run as non-root user:**

```dockerfile
# Create non-root user
RUN addgroup -g 1000 securestor && \
    adduser -D -u 1000 -G securestor securestor

# Switch to non-root user
USER securestor

# Set working directory
WORKDIR /app
```

**Scan images for vulnerabilities:**

```bash
# Using Trivy
trivy image securestor/api:latest

# Using Grype
grype securestor/api:latest

# Using Snyk
snyk container test securestor/api:latest
```

### Resource Limits

**Docker Compose:**

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
          pids: 100
        reservations:
          cpus: '1'
          memory: 2G
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
```

**Kubernetes:**

```yaml
containers:
- name: api
  resources:
    limits:
      cpu: 2000m
      memory: 4Gi
    requests:
      cpu: 1000m
      memory: 2Gi
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
      - ALL
```

### Database Hardening

**PostgreSQL:**

```sql
-- Disable unnecessary extensions
DROP EXTENSION IF EXISTS plpythonu;

-- Restrict network access
-- In pg_hba.conf:
hostssl all all 10.0.0.0/8 md5
host all all 127.0.0.1/32 md5
local all all md5

-- Set connection limits
ALTER ROLE securestor CONNECTION LIMIT 100;

-- Enable query logging
log_statement = 'ddl'
log_duration = on
log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '

-- Enable audit logging
pgaudit.log = 'all'
pgaudit.log_level = 'log'
```

**Redis:**

```bash
# Disable dangerous commands
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command CONFIG ""
rename-command SHUTDOWN ""

# Set maxmemory policy
maxmemory 2gb
maxmemory-policy allkeys-lru

# Enable AOF for persistence
appendonly yes
appendfsync everysec
```

## Monitoring & Auditing

### Audit Logging

Enable comprehensive audit logging:

```bash
# Application audit logging
AUDIT_ENABLED=true
AUDIT_LOG_PATH=/var/log/securestor/audit.log
AUDIT_LOG_LEVEL=info
AUDIT_LOG_FORMAT=json

# Log all API requests
AUDIT_LOG_REQUESTS=true
AUDIT_LOG_RESPONSES=false  # Don't log sensitive data

# Log authentication events
AUDIT_LOG_AUTH=true
AUDIT_LOG_FAILED_AUTH=true
```

### Security Monitoring

**Fail2Ban for brute force protection:**

```bash
# Install fail2ban
sudo apt-get install fail2ban

# Create SecureStor filter
cat > /etc/fail2ban/filter.d/securestor.conf <<'EOF'
[Definition]
failregex = ^.*"msg":"Failed login attempt".*"ip":"<HOST>".*$
ignoreregex =
EOF

# Configure jail
cat > /etc/fail2ban/jail.d/securestor.conf <<'EOF'
[securestor]
enabled = true
port = 80,443
filter = securestor
logpath = /var/log/securestor/audit.log
maxretry = 5
bantime = 3600
findtime = 600
EOF

# Restart fail2ban
sudo systemctl restart fail2ban
```

### Security Alerts

Configure alerts for security events:

```yaml
# Prometheus alerting rules
groups:
- name: security
  rules:
  - alert: HighFailedLoginRate
    expr: rate(securestor_failed_logins_total[5m]) > 10
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High rate of failed logins detected"
      
  - alert: UnauthorizedAccessAttempt
    expr: securestor_unauthorized_access_total > 0
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "Unauthorized access attempt detected"
      
  - alert: CriticalVulnerabilityDetected
    expr: securestor_critical_vulnerabilities_total > 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Critical vulnerability detected in artifact"
```

## Compliance

### GDPR Compliance

Implement data protection requirements:

```bash
# Enable data retention policies
DATA_RETENTION_ENABLED=true
DATA_RETENTION_DAYS=365

# Enable right to deletion
GDPR_ENABLED=true
GDPR_DATA_EXPORT_ENABLED=true
GDPR_DATA_DELETION_ENABLED=true

# Enable consent management
CONSENT_REQUIRED=true
CONSENT_TYPES=analytics,marketing
```

### SOC 2 Compliance

Implement SOC 2 controls:

- **Access Control**: RBAC with MFA
- **Audit Logging**: Comprehensive audit trails
- **Encryption**: At rest and in transit
- **Change Management**: Version control and approvals
- **Incident Response**: Documented procedures
- **Business Continuity**: Backup and disaster recovery

### HIPAA Compliance

Additional requirements for healthcare:

```bash
# Enable additional logging
HIPAA_ENABLED=true
HIPAA_AUDIT_LOG_RETENTION_YEARS=6

# Enable encryption
ENCRYPTION_REQUIRED=true
ENCRYPTION_ALGORITHM=AES-256-GCM

# Access controls
HIPAA_MFA_REQUIRED=true
HIPAA_PASSWORD_POLICY=strict
```

## Incident Response

### Security Incident Playbook

1. **Detection**: Identify security incident
2. **Containment**: Isolate affected systems
3. **Eradication**: Remove threat
4. **Recovery**: Restore normal operations
5. **Lessons Learned**: Document and improve

### Emergency Response Commands

```bash
# Immediately disable user account
curl -X POST http://localhost:8080/api/v1/admin/users/{userId}/disable \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Revoke all API keys for user
curl -X DELETE http://localhost:8080/api/v1/admin/users/{userId}/api-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Enable maintenance mode
curl -X POST http://localhost:8080/api/v1/admin/maintenance \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"enabled": true, "message": "Security maintenance"}'

# Rotate secrets
./scripts/rotate-secrets.sh

# Force password reset for all users
curl -X POST http://localhost:8080/api/v1/admin/force-password-reset \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Incident Response Contacts

```yaml
# contacts.yaml
security_team:
  email: security@securestor.io
  phone: +1-555-SECURITY
  on_call: security-oncall@securestor.io

incident_response:
  email: incident@securestor.io
  slack: #incident-response
  pagerduty: https://securestor.pagerduty.com
```

---

**Security is an ongoing process. Regularly review and update your security measures.**

**Next Steps:**
- [Monitoring & Observability](monitoring.md)
- [Disaster Recovery](disaster-recovery.md)
- [Security Auditing](security-auditing.md)