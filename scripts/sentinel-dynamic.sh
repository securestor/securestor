#!/bin/sh

# Dynamic Redis Sentinel startup script
echo "Starting Redis Sentinel with dynamic Redis master discovery..."

REDIS_AUTH_PASSWORD="${REDIS_PASSWORD:-}"
if [ -n "${REDIS_PASSWORD_FILE:-}" ] && [ -f "${REDIS_PASSWORD_FILE}" ]; then
    REDIS_AUTH_PASSWORD=$(cat "${REDIS_PASSWORD_FILE}")
fi

if [ -z "$REDIS_AUTH_PASSWORD" ]; then
    REDIS_AUTH_PASSWORD="redis123"
fi

# Wait for Redis master connectivity and get its IP
MAX_RETRIES=30
RETRY_COUNT=0
REDIS_MASTER_IP=""

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    echo "Discovering Redis master IP... (attempt $((RETRY_COUNT + 1)))"
    
    # Try to get Redis master IP from Docker DNS
    REDIS_MASTER_IP=$(getent hosts redis-master | awk '{print $1}' | head -n1)
    
    if [ -n "$REDIS_MASTER_IP" ] && [ "$REDIS_MASTER_IP" != "127.0.0.1" ]; then
        echo "✅ Redis master IP discovered: $REDIS_MASTER_IP"
        
        # Test connectivity
        if nc -z "$REDIS_MASTER_IP" 6379 2>/dev/null; then
            echo "✅ Connection verified to $REDIS_MASTER_IP:6379"
            break
        fi
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "⏳ Discovery failed, retrying in 2 seconds..."
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ Failed to discover Redis master after $MAX_RETRIES attempts"
    exit 1
fi

# Create necessary directories
mkdir -p /var/lib/sentinel /var/log/sentinel

# Generate dynamic sentinel configuration
echo "📝 Generating dynamic sentinel configuration..."
cat > /tmp/sentinel-dynamic.conf << EOF
port 26379
bind 0.0.0.0

# Monitor Redis master instance (dynamically discovered)
sentinel monitor mymaster $REDIS_MASTER_IP 6379 2

# Authentication for connecting to Redis master and replicas
sentinel auth-pass mymaster $REDIS_AUTH_PASSWORD

# Down after this many milliseconds without a response
sentinel down-after-milliseconds mymaster 5000

# Failover timeout in milliseconds
sentinel failover-timeout mymaster 30000

# Number of replicas to reconfigure for the new master after failover
sentinel parallel-syncs mymaster 1

# Sentinel log file
logfile /var/log/sentinel/sentinel.log

# Working directory
dir /var/lib/sentinel

# Configuration persistence
save ""
EOF

echo "🔧 Dynamic configuration created with master IP: $REDIS_MASTER_IP"

# Start Redis Sentinel with dynamic configuration
echo "🚀 Starting Redis Sentinel with dynamic configuration..."
exec redis-sentinel /tmp/sentinel-dynamic.conf --bind 0.0.0.0 --port 26379