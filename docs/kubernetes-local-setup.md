# Kubernetes Local Development Setup Guide

This guide covers setting up and deploying SecureStor on a local Kubernetes cluster using minikube for development and testing purposes.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Building Docker Images](#building-docker-images)
- [Deploying to Kubernetes](#deploying-to-kubernetes)
- [Accessing the Application](#accessing-the-application)
- [Common Issues and Solutions](#common-issues-and-solutions)
- [Useful Commands](#useful-commands)

## Prerequisites

### System Requirements

- **OS**: Linux, macOS, or Windows with WSL2
- **CPU**: 4+ cores (6+ recommended)
- **RAM**: 8 GB minimum (16 GB recommended)
- **Disk**: 50 GB free space
- **Virtualization**: Enabled in BIOS (VT-x/AMD-v)

### Required Tools

#### 1. Install Docker

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y docker.io docker-compose
sudo usermod -aG docker $USER
newgrp docker

# Verify installation
docker --version
```

#### 2. Install kubectl

```bash
# Download latest stable version
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# Install kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Verify installation
kubectl version --client
```

#### 3. Install minikube

```bash
# Download minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64

# Install minikube
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Verify installation
minikube version
```

## Environment Setup

### Step 1: Start Minikube Cluster

```bash
# Start minikube with recommended resources
minikube start --cpus=4 --memory=8192 --disk-size=40g --driver=docker

# Verify cluster is running
minikube status

# Expected output:
# minikube
# type: Control Plane
# host: Running
# kubelet: Running
# apiserver: Running
# kubeconfig: Configured
```

### Step 2: Enable Required Addons

```bash
# Enable ingress controller
minikube addons enable ingress

# Enable metrics server (optional, for monitoring)
minikube addons enable metrics-server

# Verify addons
minikube addons list
```

### Step 3: Configure Docker Environment

```bash
# Point Docker CLI to minikube's Docker daemon
# This allows you to build images directly in minikube
eval $(minikube docker-env)

# Verify you're using minikube's Docker
docker ps
```

### Step 4: Create Kubernetes Namespace

```bash
# Create securestor namespace
kubectl create namespace securestor

# Set as default namespace for convenience
kubectl config set-context --current --namespace=securestor

# Verify namespace
kubectl get namespaces
```

## Building Docker Images

### Step 1: Build Backend API Image

```bash
cd /path/to/securestor

# Build API image (ensure you're using minikube's Docker)
docker build -t securestor/api:latest -f Dockerfile .

# Verify image
docker images | grep securestor
```

### Step 2: Build Frontend Image

```bash
cd /path/to/securestor/frontend

# Build frontend with local development configuration
docker build \
  --build-arg REACT_APP_BACKEND_PORT=8080 \
  --build-arg REACT_APP_API_URL=http://localhost:8080 \
  --build-arg REACT_APP_DEFAULT_TENANT=admin \
  --build-arg REACT_APP_DEV_MODE=true \
  --build-arg REACT_APP_BASE_DOMAIN=localhost \
  -t securestor/frontend:latest \
  .

# Verify image
docker images | grep securestor/frontend
```

**Important Build Args Explained:**
- `REACT_APP_BACKEND_PORT=8080`: API server port
- `REACT_APP_API_URL=http://localhost:8080`: Full API URL for local access
- `REACT_APP_DEFAULT_TENANT=admin`: Default tenant for development
- `REACT_APP_DEV_MODE=true`: Enables development features
- `REACT_APP_BASE_DOMAIN=localhost`: Base domain for local testing

### Step 3: Load Images into Minikube (if needed)

```bash
# Only needed if you built images outside minikube's Docker
minikube image load securestor/api:latest
minikube image load securestor/frontend:latest

# Verify images are loaded
minikube image ls | grep securestor
```

## Deploying to Kubernetes

### Step 1: Update Kubernetes Manifests for Local Development

The manifests in `k8s/` need adjustments for local development:

#### Update Storage Class

Edit `k8s/storage.yaml`:
```yaml
# Change from nfs-storage to standard
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: api-storage
spec:
  accessModes:
    - ReadWriteOnce  # Changed from ReadWriteMany
  storageClassName: standard  # Changed from nfs-storage
  resources:
    requests:
      storage: 10Gi
```

#### Update Image Pull Policy

Edit `k8s/api.yaml` and `k8s/services.yaml`:
```yaml
spec:
  containers:
  - name: securestor-api
    image: securestor/api:latest
    imagePullPolicy: Never  # Add this line to use local images
```

#### Update Ingress Hostname

Edit `k8s/ingress.yaml`:
```yaml
spec:
  rules:
  - host: securestor.local  # Use .local for development
```

### Step 2: Create ConfigMaps

```bash
# Create OPA policies ConfigMap
kubectl create configmap opa-policies \
  --from-file=policies/ \
  -n securestor

# Verify ConfigMap
kubectl get configmap opa-policies -n securestor
```

### Step 3: Deploy All Components

```bash
cd /path/to/securestor

# Apply all Kubernetes manifests
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/storage.yaml
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/services.yaml
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/ingress.yaml

# Watch pods come up
kubectl get pods -n securestor -w
```

### Step 4: Wait for All Pods to be Ready

```bash
# Check pod status
kubectl get pods -n securestor

# All pods should show Running status:
# NAME                        READY   STATUS    RESTARTS   AGE
# api-xxxxx                   1/1     Running   0          2m
# frontend-xxxxx              1/1     Running   0          2m
# postgres-primary-0          1/1     Running   0          3m
# postgres-standby-0          1/1     Running   0          3m
# redis-master-0              1/1     Running   0          3m
# redis-replica-0             1/1     Running   0          3m
# redis-replica-1             1/1     Running   0          3m
# redis-sentinel-0            1/1     Running   0          3m
# redis-sentinel-1            1/1     Running   0          3m
# redis-sentinel-2            1/1     Running   0          3m
# keycloak-xxxxx              1/1     Running   0          2m
# opa-xxxxx                   1/1     Running   0          2m
# nginx-lb-xxxxx              1/1     Running   0          2m
```

## Accessing the Application

### Method 1: Port Forwarding (Recommended)

Use the provided port-forward management script:

```bash
# Start the port-forward manager
cd /path/to/securestor
./scripts/start-local-access.sh

# The script monitors and auto-restarts port forwards every 5 seconds
# Logs: /tmp/securestor-access.log

# Access the application:
# Frontend: http://localhost:3000
# API:      http://localhost:8080
```

**Manual Port Forwarding:**
```bash
# Terminal 1: Forward API service
kubectl port-forward -n securestor service/api 8080:8080

# Terminal 2: Forward Frontend service
kubectl port-forward -n securestor service/frontend 3000:3000
```

### Method 2: Ingress with Hosts File

```bash
# Get minikube IP
minikube ip
# Example output: 192.168.49.2

# Add to /etc/hosts
echo "192.168.49.2 securestor.local" | sudo tee -a /etc/hosts

# Access application
# http://securestor.local
```

### Step 5: Setup Initial Admin User

```bash
# Create admin user
kubectl exec -it -n securestor deployment/api -- \
  /app/api setup-admin \
  --username admin \
  --password admin123 \
  --email admin@securestor.io \
  --tenant admin

# Login credentials:
# Username: admin
# Password: admin123
# Tenant: admin
```

**⚠️ Security Warning:** Change the default password immediately after first login!

## Common Issues and Solutions

### Issue 1: Pods Stuck in Pending State

**Symptom:** Pods show `Pending` status
```bash
kubectl get pods -n securestor
# NAME                    READY   STATUS    RESTARTS   AGE
# api-xxxxx               0/1     Pending   0          5m
```

**Solution:**
```bash
# Check PVC status
kubectl get pvc -n securestor

# If PVC is unbound, check storage class
kubectl get sc

# Use 'standard' storage class for local development
# Edit k8s/storage.yaml and change storageClassName to 'standard'
```

### Issue 2: ImagePullBackOff Errors

**Symptom:** Pods fail with `ImagePullBackOff`
```bash
kubectl describe pod <pod-name> -n securestor
# Events:
#   Failed to pull image "securestor/api:latest": image not found
```

**Solution:**
```bash
# Ensure images are built in minikube's Docker environment
eval $(minikube docker-env)

# Rebuild images
docker build -t securestor/api:latest .

# OR load images into minikube
minikube image load securestor/api:latest

# Update deployment to use imagePullPolicy: Never
kubectl set image deployment/api api=securestor/api:latest -n securestor
kubectl patch deployment api -n securestor -p '{"spec":{"template":{"spec":{"containers":[{"name":"api","imagePullPolicy":"Never"}]}}}}'
```

### Issue 3: Redis Sentinel CrashLoopBackOff

**Symptom:** Redis Sentinel pods keep restarting
```bash
kubectl logs -n securestor redis-sentinel-0
# Could not resolve master hostname
```

**Solution:** The issue is DNS resolution timing. The fix is already included in `k8s/redis.yaml`:
```yaml
# Wait for master to be resolvable before starting sentinel
command: ["/bin/sh", "-c"]
args:
  - |
    until redis-cli -h redis-master-0.redis-master.securestor.svc.cluster.local -p 6379 ping; do
      echo "Waiting for redis-master to be ready..."
      sleep 2
    done
    redis-sentinel /etc/redis/sentinel.conf
```

### Issue 4: Frontend Shows "Tenant Not Found"

**Symptom:** Browser shows tenant validation error

**Solution:**
```bash
# Verify admin tenant exists in database
kubectl exec -it -n securestor postgres-primary-0 -- \
  psql -U postgres -d securestor -c "SELECT id, slug, name FROM tenants;"

# If admin tenant doesn't exist, create it
kubectl exec -it -n securestor deployment/api -- \
  /app/api setup-admin --username admin --password admin123 --email admin@securestor.io --tenant admin
```

### Issue 5: API Calls Return 502 Bad Gateway

**Symptom:** Browser console shows API errors
```
GET http://localhost:3000/api/v1/... 502 (Bad Gateway)
```

**Root Cause:** Frontend is calling APIs on wrong port (3000 instead of 8080)

**Solution:**
```bash
# Rebuild frontend with correct environment variables
cd frontend
docker build \
  --build-arg REACT_APP_BACKEND_PORT=8080 \
  --build-arg REACT_APP_API_URL=http://localhost:8080 \
  --build-arg REACT_APP_DEFAULT_TENANT=admin \
  --no-cache \
  -t securestor/frontend:v2 \
  .

# Update deployment
kubectl set image deployment/frontend frontend=securestor/frontend:v2 -n securestor

# Hard refresh browser (Ctrl+Shift+R)
```

### Issue 6: Port Forward Keeps Dying

**Symptom:** `kubectl port-forward` terminates frequently

**Solution:** Use the monitoring script:
```bash
# The script auto-restarts port forwards
./scripts/start-local-access.sh

# Check logs
tail -f /tmp/securestor-access.log
```

### Issue 7: Browser Cache Issues

**Symptom:** Changes don't appear after redeployment

**Solution:**
```bash
# Use versioned image tags
docker build -t securestor/frontend:v3 .
kubectl set image deployment/frontend frontend=securestor/frontend:v3 -n securestor

# Hard refresh browser: Ctrl+Shift+R (Linux/Windows) or Cmd+Shift+R (Mac)
# Or clear browser cache completely
```

## Useful Commands

### Cluster Management

```bash
# Check cluster status
minikube status

# Access minikube dashboard
minikube dashboard

# SSH into minikube node
minikube ssh

# Stop minikube
minikube stop

# Delete minikube cluster
minikube delete

# View cluster info
kubectl cluster-info
```

### Pod Operations

```bash
# List all pods
kubectl get pods -n securestor

# Watch pods in real-time
kubectl get pods -n securestor -w

# Describe pod (shows events and errors)
kubectl describe pod <pod-name> -n securestor

# View pod logs
kubectl logs <pod-name> -n securestor

# Follow logs in real-time
kubectl logs -f <pod-name> -n securestor

# Execute command in pod
kubectl exec -it <pod-name> -n securestor -- /bin/sh

# Copy files from/to pod
kubectl cp <pod-name>:/path/to/file ./local-file -n securestor
kubectl cp ./local-file <pod-name>:/path/to/file -n securestor
```

### Deployment Operations

```bash
# Restart deployment
kubectl rollout restart deployment/api -n securestor

# Check rollout status
kubectl rollout status deployment/api -n securestor

# View rollout history
kubectl rollout history deployment/api -n securestor

# Rollback to previous version
kubectl rollout undo deployment/api -n securestor

# Scale deployment
kubectl scale deployment/api --replicas=3 -n securestor

# Update image
kubectl set image deployment/api api=securestor/api:v2 -n securestor
```

### Service & Networking

```bash
# List services
kubectl get services -n securestor

# List ingress
kubectl get ingress -n securestor

# Test service connectivity
kubectl run -it --rm debug --image=busybox --restart=Never -n securestor -- sh
# Inside pod:
wget -O- http://api:8080/health

# Port forward service
kubectl port-forward service/api 8080:8080 -n securestor
```

### Storage

```bash
# List PersistentVolumes
kubectl get pv

# List PersistentVolumeClaims
kubectl get pvc -n securestor

# Describe PVC
kubectl describe pvc <pvc-name> -n securestor

# List StorageClasses
kubectl get storageclass
```

### Debugging

```bash
# Check events
kubectl get events -n securestor --sort-by='.lastTimestamp'

# Check resource usage
kubectl top nodes
kubectl top pods -n securestor

# View all resources
kubectl get all -n securestor

# Export resource YAML
kubectl get deployment api -n securestor -o yaml

# Edit resource in-place
kubectl edit deployment api -n securestor
```

### Database Operations

```bash
# Connect to PostgreSQL primary
kubectl exec -it -n securestor postgres-primary-0 -- psql -U postgres -d securestor

# Run SQL query
kubectl exec -it -n securestor postgres-primary-0 -- \
  psql -U postgres -d securestor -c "SELECT * FROM tenants;"

# Check replication status
kubectl exec -it -n securestor postgres-primary-0 -- \
  psql -U postgres -c "SELECT * FROM pg_stat_replication;"

# Connect to Redis
kubectl exec -it -n securestor redis-master-0 -- redis-cli

# Check Redis Sentinel status
kubectl exec -it -n securestor redis-sentinel-0 -- \
  redis-cli -p 26379 SENTINEL masters
```

### Cleanup

```bash
# Delete all resources in namespace
kubectl delete all --all -n securestor

# Delete namespace (deletes everything)
kubectl delete namespace securestor

# Delete specific resource
kubectl delete deployment api -n securestor

# Force delete stuck pod
kubectl delete pod <pod-name> -n securestor --force --grace-period=0
```

## Development Workflow

### Making Code Changes

```bash
# 1. Make code changes in your editor

# 2. Ensure you're using minikube's Docker
eval $(minikube docker-env)

# 3. Build new image with version tag
docker build -t securestor/api:v2 .

# 4. Update deployment
kubectl set image deployment/api api=securestor/api:v2 -n securestor

# 5. Watch rollout
kubectl rollout status deployment/api -n securestor

# 6. Check logs
kubectl logs -f deployment/api -n securestor
```

### Rapid Development Cycle

For faster iteration, you can use:
1. **Hot reload** by mounting source code as volume
2. **Skaffold** for automated build/deploy cycle
3. **Tilt** for smart rebuilds

Example Skaffold configuration:
```yaml
apiVersion: skaffold/v2beta29
kind: Config
build:
  local:
    push: false
  artifacts:
  - image: securestor/api
    docker:
      dockerfile: Dockerfile
deploy:
  kubectl:
    manifests:
    - k8s/*.yaml
```

## Performance Tuning

### Adjust Minikube Resources

```bash
# Stop minikube
minikube stop

# Start with more resources
minikube start --cpus=6 --memory=16384 --disk-size=80g
```

### Resource Limits

Edit deployments to set appropriate limits:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

## Next Steps

- Review [Production Kubernetes Deployment](kubernetes-deployment.md) for production setup
- Check [Production Hardening Guide](production-hardening.md) for security best practices
- Explore [Docker Deployment](docker-deployment.md) for simpler container-only deployment

## Support

For issues and questions:
- GitHub Issues: [securestor/issues](https://github.com/yourusername/securestor/issues)
- Documentation: [docs/](../)
- Troubleshooting: See "Common Issues" section above

---

**Last Updated:** January 8, 2026
