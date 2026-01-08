# Kubernetes Deployment Guide

This guide covers deploying SecureStor on Kubernetes for production-grade, scalable deployments.

> **📝 Note:** For local development and testing with minikube, see the [Kubernetes Local Setup Guide](kubernetes-local-setup.md).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Architecture Overview](#architecture-overview)
- [Quick Start](#quick-start)
- [Production Deployment](#production-deployment)
- [Scaling](#scaling)
- [Monitoring](#monitoring)
- [Backup & Disaster Recovery](#backup--disaster-recovery)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Cluster Requirements

**Minimum Cluster:**
- Kubernetes 1.24+
- 3 worker nodes
- 4 CPU cores per node
- 8 GB RAM per node
- 100 GB storage per node
- LoadBalancer service support (or Ingress controller)

**Recommended Production Cluster:**
- Kubernetes 1.26+
- 5+ worker nodes
- 8 CPU cores per node
- 16 GB RAM per node
- 500 GB SSD storage per node
- LoadBalancer or Ingress controller
- StorageClass with dynamic provisioning
- Monitoring stack (Prometheus/Grafana)

### Tools Required

```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# helm (optional but recommended)
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# kustomize (optional)
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
```

## Architecture Overview

```
┌─────────────────────────────────────────────┐
│           Ingress Controller                │
│  (nginx-ingress / traefik / istio)         │
└──────────────────┬──────────────────────────┘
                   │
        ┌──────────┴──────────┐
        ▼                     ▼
┌───────────────┐    ┌───────────────┐
│   Frontend    │    │   API Server  │
│  (3 replicas) │    │  (3 replicas) │
└───────┬───────┘    └───────┬───────┘
        │                    │
        └────────┬───────────┘
                 │
        ┌────────┴────────┐
        ▼                 ▼
┌──────────────┐   ┌──────────────┐
│  PostgreSQL  │   │    Redis     │
│ StatefulSet  │   │ StatefulSet  │
│ (Primary +   │   │  (Sentinel   │
│  Replicas)   │   │   Cluster)   │
└──────────────┘   └──────────────┘
        │                 │
        ▼                 ▼
┌──────────────┐   ┌──────────────┐
│ PersistentVC │   │ PersistentVC │
└──────────────┘   └──────────────┘
```

## Quick Start

### Step 1: Create Namespace

```bash
kubectl create namespace securestor

# Set as default namespace
kubectl config set-context --current --namespace=securestor
```

### Step 2: Create Secrets

```bash
# Generate secure passwords
export POSTGRES_PASSWORD=$(openssl rand -base64 32)
export REDIS_PASSWORD=$(openssl rand -base64 32)
export JWT_SECRET=$(openssl rand -base64 64)
export ENCRYPTION_KEY=$(openssl rand -hex 32)

# Create database secret
kubectl create secret generic postgres-credentials \
  --from-literal=username=securestor \
  --from-literal=password=$POSTGRES_PASSWORD \
  --from-literal=database=securestor \
  -n securestor

# Create Redis secret
kubectl create secret generic redis-credentials \
  --from-literal=password=$REDIS_PASSWORD \
  -n securestor

# Create application secrets
kubectl create secret generic securestor-secrets \
  --from-literal=jwt-secret=$JWT_SECRET \
  --from-literal=encryption-key=$ENCRYPTION_KEY \
  -n securestor
```

### Step 3: Apply Kubernetes Manifests

```bash
# Clone repository
git clone https://github.com/securestor/securestor.git
cd securestor/k8s

# Apply all manifests
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f storage.yaml
kubectl apply -f postgres.yaml
kubectl apply -f redis.yaml
kubectl apply -f api.yaml
kubectl apply -f services.yaml
kubectl apply -f ingress.yaml
```

### Step 4: Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n securestor

# Check services
kubectl get svc -n securestor

# Check ingress
kubectl get ingress -n securestor

# View logs
kubectl logs -f deployment/securestor-api -n securestor
```

### Step 5: Access Application

```bash
# Get LoadBalancer IP (if using LoadBalancer service)
kubectl get svc securestor-frontend -n securestor

# Or get Ingress address
kubectl get ingress securestor-ingress -n securestor

# Port forward for local access (testing)
kubectl port-forward svc/securestor-frontend 3000:80 -n securestor
```

## Production Deployment

### PostgreSQL StatefulSet

Create `k8s/postgres.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: securestor
spec:
  ports:
  - port: 5432
    name: postgres
  clusterIP: None
  selector:
    app: postgres
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: securestor
spec:
  serviceName: postgres
  replicas: 3
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:14
        ports:
        - containerPort: 5432
          name: postgres
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: password
        - name: POSTGRES_DB
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: database
        - name: PGDATA
          value: /var/lib/postgresql/data/pgdata
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - securestor
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - securestor
          initialDelaySeconds: 5
          periodSeconds: 5
  volumeClaimTemplates:
  - metadata:
      name: postgres-storage
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: fast-ssd
      resources:
        requests:
          storage: 100Gi
```

### Redis StatefulSet with Sentinel

Create `k8s/redis.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: securestor
data:
  master.conf: |
    bind 0.0.0.0
    protected-mode no
    port 6379
    tcp-backlog 511
    timeout 0
    tcp-keepalive 300
    daemonize no
    supervised no
    pidfile /var/run/redis_6379.pid
    loglevel notice
    logfile ""
  sentinel.conf: |
    bind 0.0.0.0
    port 26379
    sentinel monitor mymaster redis-0.redis.securestor.svc.cluster.local 6379 2
    sentinel down-after-milliseconds mymaster 5000
    sentinel parallel-syncs mymaster 1
    sentinel failover-timeout mymaster 10000
---
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: securestor
spec:
  ports:
  - port: 6379
    name: redis
  - port: 26379
    name: sentinel
  clusterIP: None
  selector:
    app: redis
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: securestor
spec:
  serviceName: redis
  replicas: 3
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
          name: redis
        command:
        - redis-server
        - /conf/master.conf
        - --requirepass
        - $(REDIS_PASSWORD)
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-credentials
              key: password
        volumeMounts:
        - name: redis-config
          mountPath: /conf
        - name: redis-storage
          mountPath: /data
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
      - name: sentinel
        image: redis:7-alpine
        ports:
        - containerPort: 26379
          name: sentinel
        command:
        - redis-sentinel
        - /conf/sentinel.conf
        volumeMounts:
        - name: redis-config
          mountPath: /conf
      volumes:
      - name: redis-config
        configMap:
          name: redis-config
  volumeClaimTemplates:
  - metadata:
      name: redis-storage
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: fast-ssd
      resources:
        requests:
          storage: 50Gi
```

### API Deployment

Create `k8s/api.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: securestor-api
  namespace: securestor
spec:
  replicas: 3
  selector:
    matchLabels:
      app: securestor-api
  template:
    metadata:
      labels:
        app: securestor-api
    spec:
      containers:
      - name: api
        image: securestor/api:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: DATABASE_URL
          value: "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable"
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: password
        - name: POSTGRES_DB
          valueFrom:
            secretKeyRef:
              name: postgres-credentials
              key: database
        - name: REDIS_URL
          value: "redis://:$(REDIS_PASSWORD)@redis:6379/0"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-credentials
              key: password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: securestor-secrets
              key: jwt-secret
        - name: ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: securestor-secrets
              key: encryption-key
        - name: ENVIRONMENT
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        - name: SCANNER_ENABLED
          value: "true"
        - name: SCANNER_ON_UPLOAD
          value: "true"
        volumeMounts:
        - name: artifacts-storage
          mountPath: /data/artifacts
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /api/v1/health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/v1/health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: artifacts-storage
        persistentVolumeClaim:
          claimName: artifacts-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: securestor-api
  namespace: securestor
spec:
  selector:
    app: securestor-api
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

### Frontend Deployment

Create `k8s/frontend.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: securestor-frontend
  namespace: securestor
spec:
  replicas: 3
  selector:
    matchLabels:
      app: securestor-frontend
  template:
    metadata:
      labels:
        app: securestor-frontend
    spec:
      containers:
      - name: frontend
        image: securestor/frontend:latest
        ports:
        - containerPort: 80
          name: http
        env:
        - name: API_BASE_URL
          value: "http://securestor-api:8080"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: securestor-frontend
  namespace: securestor
spec:
  selector:
    app: securestor-frontend
  ports:
  - port: 80
    targetPort: 80
  type: LoadBalancer
```

### Ingress Configuration

Create `k8s/ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: securestor-ingress
  namespace: securestor
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/proxy-body-size: "5000m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "600"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - registry.yourcompany.com
    secretName: securestor-tls
  rules:
  - host: registry.yourcompany.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: securestor-api
            port:
              number: 8080
      - path: /
        pathType: Prefix
        backend:
          service:
            name: securestor-frontend
            port:
              number: 80
```

### Storage Configuration

Create `k8s/storage.yaml`:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/aws-ebs  # or gce-pd, azure-disk
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
allowVolumeExpansion: true
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: artifacts-pvc
  namespace: securestor
spec:
  accessModes:
  - ReadWriteMany  # For shared access across API pods
  storageClassName: fast-ssd
  resources:
    requests:
      storage: 500Gi
```

## Scaling

### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: securestor-api-hpa
  namespace: securestor
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: securestor-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Manual Scaling

```bash
# Scale API deployment
kubectl scale deployment securestor-api --replicas=5 -n securestor

# Scale frontend
kubectl scale deployment securestor-frontend --replicas=5 -n securestor

# Scale PostgreSQL (requires careful planning)
kubectl scale statefulset postgres --replicas=5 -n securestor
```

## Monitoring

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: securestor-api
  namespace: securestor
spec:
  selector:
    matchLabels:
      app: securestor-api
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

### Key Metrics to Monitor

```bash
# API metrics
securestor_api_requests_total
securestor_api_request_duration_seconds
securestor_api_errors_total

# Storage metrics
securestor_storage_used_bytes
securestor_storage_available_bytes
securestor_artifact_count

# Scanner metrics
securestor_scans_total
securestor_vulnerabilities_found
securestor_scan_duration_seconds
```

## Backup & Disaster Recovery

### Velero Backup

```bash
# Install Velero
velero install \
  --provider aws \
  --plugins velero/velero-plugin-for-aws:v1.7.0 \
  --bucket securestor-backups \
  --backup-location-config region=us-east-1

# Create backup schedule
velero schedule create securestor-daily \
  --schedule="0 2 * * *" \
  --include-namespaces securestor \
  --ttl 720h0m0s

# Manual backup
velero backup create securestor-backup-$(date +%Y%m%d) \
  --include-namespaces securestor

# Restore from backup
velero restore create --from-backup securestor-backup-20260107
```

## Troubleshooting

### Common Issues

```bash
# Check pod status
kubectl get pods -n securestor

# View pod logs
kubectl logs -f deployment/securestor-api -n securestor

# Describe pod for events
kubectl describe pod <pod-name> -n securestor

# Check resource usage
kubectl top pods -n securestor
kubectl top nodes

# Debug pod
kubectl exec -it <pod-name> -n securestor -- /bin/sh
```

### Useful Commands

```bash
# Restart deployment
kubectl rollout restart deployment/securestor-api -n securestor

# Check rollout status
kubectl rollout status deployment/securestor-api -n securestor

# View deployment history
kubectl rollout history deployment/securestor-api -n securestor

# Rollback deployment
kubectl rollout undo deployment/securestor-api -n securestor
```

---

**Next Steps:**
- [Kubernetes Local Setup](kubernetes-local-setup.md) - Local development with minikube
- [Production Hardening Guide](production-hardening.md)
- [Monitoring & Observability](monitoring.md)
- [Security Best Practices](security.md)