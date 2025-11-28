#!/bin/sh

# Redis Sentinel startup script with connectivity check
echo "Starting Redis Sentinel with connectivity check..."

# Wait for Redis master connectivity
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    echo "Checking connectivity to redis-master... (attempt $((RETRY_COUNT + 1)))"
    
    # Try to connect to redis-master port 6379
    if nc -z redis-master 6379 2>/dev/null; then
        echo "✅ Connection successful to redis-master:6379"
        break
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "⏳ Connection failed, retrying in 2 seconds..."
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "❌ Failed to connect to redis-master after $MAX_RETRIES attempts"
    exit 1
fi

# Create necessary directories
mkdir -p /var/lib/sentinel /var/log/sentinel

# Start Redis Sentinel
echo "🚀 Starting Redis Sentinel..."
exec redis-sentinel /etc/redis/sentinel.conf --bind 0.0.0.0 --port 26379