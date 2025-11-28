#!/bin/bash
# PostgreSQL Primary setup for HA
# This script configures the primary database for replication

set -e

echo "Setting up PostgreSQL primary for replication..."

# Update pg_hba.conf BEFORE PostgreSQL starts (in initdb phase)
# This ensures replication connections are allowed from the start
PGDATA=${PGDATA:-/var/lib/postgresql/data}

# Add replication entries to pg_hba.conf if they don't exist
if [ -f "$PGDATA/pg_hba.conf" ]; then
    if ! grep -q "replication replication_user" "$PGDATA/pg_hba.conf"; then
        echo "" >> "$PGDATA/pg_hba.conf"
        echo "# Replication connections from standby servers" >> "$PGDATA/pg_hba.conf"
        echo "host    replication     replication_user    0.0.0.0/0               md5" >> "$PGDATA/pg_hba.conf"
        echo "host    replication     replication_user    ::/0                    md5" >> "$PGDATA/pg_hba.conf"
        echo "# Allow all connections for application" >> "$PGDATA/pg_hba.conf"
        echo "host    all             all                 0.0.0.0/0               md5" >> "$PGDATA/pg_hba.conf"
        echo "host    all             all                 ::/0                    md5" >> "$PGDATA/pg_hba.conf"
        echo "Added replication entries to pg_hba.conf"
    fi
fi

# Create replication user
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'replication_user') THEN
            CREATE USER replication_user WITH ENCRYPTED PASSWORD 'replication123' REPLICATION;
            RAISE NOTICE 'Created replication_user';
        ELSE
            RAISE NOTICE 'replication_user already exists';
        END IF;
    END
    \$\$;
    
    GRANT CONNECT ON DATABASE securestor TO replication_user;
EOSQL

# Reload PostgreSQL configuration to apply pg_hba.conf changes
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -c "SELECT pg_reload_conf();"

echo "PostgreSQL Primary setup complete:"
echo "  - Replication user created"
echo "  - pg_hba.conf configured for replication"
echo "  - Configuration reloaded"