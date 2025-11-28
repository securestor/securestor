#!/bin/bash
# PostgreSQL Standby setup for HA
# This script configures the standby database for streaming replication

set -e

echo "Starting PostgreSQL Standby setup..."

# Wait for primary to be ready
max_attempts=60
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if pg_isready -h postgres-primary -U securestor > /dev/null 2>&1; then
        echo "Primary database is ready"
        break
    fi
    attempt=$((attempt + 1))
    echo "Waiting for primary database... attempt $attempt/$max_attempts"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "ERROR: Primary database did not become ready in time"
    exit 1
fi

# Check if data directory is empty or needs initialization
if [ -z "$(ls -A /var/lib/postgresql/data)" ] || [ ! -f "/var/lib/postgresql/data/PG_VERSION" ]; then
    echo "Data directory is empty or invalid, performing base backup..."
    
    # Remove any existing data
    rm -rf /var/lib/postgresql/data/*
    
    # Set password for pg_basebackup
    export PGPASSWORD='replication123'
    
    # Take base backup from primary (using -R to create replication config)
    echo "Taking base backup from primary..."
    pg_basebackup -h postgres-primary -D /var/lib/postgresql/data -U replication_user -v -P -R --wal-method=stream
    
    if [ $? -ne 0 ]; then
        echo "ERROR: pg_basebackup failed"
        exit 1
    fi
    
    echo "Base backup completed successfully"
    
    # Ensure proper permissions
    chmod 700 /var/lib/postgresql/data
    chown -R postgres:postgres /var/lib/postgresql/data
    
    echo "PostgreSQL Standby setup complete"
else
    echo "Data directory already initialized, skipping base backup"
fi