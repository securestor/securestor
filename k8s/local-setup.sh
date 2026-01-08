#!/bin/bash
set -e

echo "=== SecureStor Local Kubernetes Setup ==="

# Build and load Docker images
echo "Building API image..."
docker build -t securestor/api:latest .

echo "Building Frontend image..."
docker build -t securestor/frontend:latest -f frontend/Dockerfile ./frontend

echo "Loading images into minikube..."
minikube image load securestor/api:latest
minikube image load securestor/frontend:latest

# Create PersistentVolumes for local development
echo "Creating PersistentVolumes..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: api-storage-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteMany
  hostPath:
    path: /data/api-storage
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: redis-master-pv
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /data/redis-master
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: redis-replica-0-pv
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /data/redis-replica-0
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: redis-replica-1-pv
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /data/redis-replica-1
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: postgres-primary-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /data/postgres-primary
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: postgres-standby-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /data/postgres-standby
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: keycloak-db-pv
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /data/keycloak-db
  storageClassName: standard
EOF

echo "Restarting failed pods..."
kubectl delete pods -n securestor --field-selector status.phase=Failed --ignore-not-found=true
kubectl delete pods -n securestor -l app=frontend --ignore-not-found=true
kubectl delete pods -n securestor -l app=api --ignore-not-found=true
kubectl delete pods -n securestor -l app=redis-sentinel --ignore-not-found=true

echo ""
echo "=== Setup Complete ==="
echo "Run: kubectl get pods -n securestor --watch"
echo "to monitor pod status"
