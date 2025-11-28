#!/bin/bash

# Script to populate default OAuth2 scopes into the database
# This script inserts the default scopes required for API key management

set -e

echo "🔧 Populating Default OAuth2 Scopes..."
echo "========================================"

# Database credentials
DB_PASSWORD="${DB_PASSWORD:-securestor123}"

# Get the tenant ID dynamically (Admin Organization)
if command -v docker &> /dev/null && docker ps --format "{{.Names}}" | grep -q "postgres-primary"; then
    POSTGRES_CONTAINER=$(docker ps --format "{{.Names}}" | grep "postgres-primary" | head -1)
    TENANT_ID=$(docker exec -i $POSTGRES_CONTAINER env PGPASSWORD="$DB_PASSWORD" psql -U securestor -d securestor -t -c "SELECT tenant_id FROM tenants LIMIT 1;" | tr -d ' ')
else
    TENANT_ID=$(PGPASSWORD="$DB_PASSWORD" psql -U securestor -d securestor -t -c "SELECT tenant_id FROM tenants LIMIT 1;" | tr -d ' ')
fi

if [[ -z "$TENANT_ID" ]]; then
    echo "❌ Error: No tenant found in database. Please ensure tenants are created first."
    exit 1
fi

echo "Using tenant ID: $TENANT_ID"

# Check if running in Docker or local environment
if command -v docker &> /dev/null && docker ps --format "{{.Names}}" | grep -q "postgres-primary"; then
    echo "📦 Using docker to connect to database..."
    POSTGRES_CONTAINER=$(docker ps --format "{{.Names}}" | grep "postgres-primary" | head -1)
    DB_CMD="docker exec -i $POSTGRES_CONTAINER env PGPASSWORD=$DB_PASSWORD psql -U securestor -d securestor"
else
    echo "📦 Using local psql connection..."
    DB_CMD="PGPASSWORD=$DB_PASSWORD psql -U securestor -d securestor"
fi

# Check current scope count
COUNT_RESULT=$($DB_CMD -c "SELECT COUNT(*) FROM oauth2_scopes WHERE tenant_id = '$TENANT_ID';" 2>/dev/null)
CURRENT_COUNT=$(echo "$COUNT_RESULT" | grep -E '^ *[0-9]+$' | tr -d ' ')
CURRENT_COUNT=${CURRENT_COUNT:-0}
echo "Current scopes in database: $CURRENT_COUNT"

if [ "$CURRENT_COUNT" -gt 0 ] 2>/dev/null; then
    echo "ℹ️  Scopes already exist in database. Skipping insertion."
    $DB_CMD -c "SELECT name, resource, is_default FROM oauth2_scopes WHERE tenant_id = '$TENANT_ID' ORDER BY name;" 2>/dev/null || true
    exit 0
fi

echo "✏️  Inserting default scopes..."

# Insert default scopes
if [[ $DB_CMD == *"docker exec"* ]]; then
    # For Docker exec, use here-doc differently
    docker exec -i $POSTGRES_CONTAINER env PGPASSWORD="$DB_PASSWORD" psql -U securestor -d securestor << EOF
INSERT INTO oauth2_scopes (scope_id, tenant_id, name, description, resource, actions, is_default) 
VALUES 
  (gen_random_uuid(), '$TENANT_ID', 'read', 'Read access to resources', '*', ARRAY['read'], true),
  (gen_random_uuid(), '$TENANT_ID', 'write', 'Write access to resources', '*', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'admin', 'Administrative access', '*', ARRAY['read', 'write', 'delete', 'admin'], false),
  (gen_random_uuid(), '$TENANT_ID', 'artifacts:read', 'Read access to artifacts', 'artifacts', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'artifacts:write', 'Write access to artifacts', 'artifacts', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'artifacts:delete', 'Delete access to artifacts', 'artifacts', ARRAY['read', 'write', 'delete'], false),
  (gen_random_uuid(), '$TENANT_ID', 'scans:read', 'Read access to scans', 'scans', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'scans:write', 'Write access to scans', 'scans', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'compliance:read', 'Read access to compliance', 'compliance', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'compliance:write', 'Write access to compliance', 'compliance', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'repositories:read', 'Read access to repositories', 'repositories', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'repositories:write', 'Write access to repositories', 'repositories', ARRAY['read', 'write'], false);
EOF
else
    # For local psql
    $DB_CMD << EOF
INSERT INTO oauth2_scopes (scope_id, tenant_id, name, description, resource, actions, is_default) 
VALUES 
  (gen_random_uuid(), '$TENANT_ID', 'read', 'Read access to resources', '*', ARRAY['read'], true),
  (gen_random_uuid(), '$TENANT_ID', 'write', 'Write access to resources', '*', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'admin', 'Administrative access', '*', ARRAY['read', 'write', 'delete', 'admin'], false),
  (gen_random_uuid(), '$TENANT_ID', 'artifacts:read', 'Read access to artifacts', 'artifacts', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'artifacts:write', 'Write access to artifacts', 'artifacts', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'artifacts:delete', 'Delete access to artifacts', 'artifacts', ARRAY['read', 'write', 'delete'], false),
  (gen_random_uuid(), '$TENANT_ID', 'scans:read', 'Read access to scans', 'scans', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'scans:write', 'Write access to scans', 'scans', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'compliance:read', 'Read access to compliance', 'compliance', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'compliance:write', 'Write access to compliance', 'compliance', ARRAY['read', 'write'], false),
  (gen_random_uuid(), '$TENANT_ID', 'repositories:read', 'Read access to repositories', 'repositories', ARRAY['read'], false),
  (gen_random_uuid(), '$TENANT_ID', 'repositories:write', 'Write access to repositories', 'repositories', ARRAY['read', 'write'], false);
EOF
fi

# Verify insertion
COUNT_RESULT=$($DB_CMD -c "SELECT COUNT(*) FROM oauth2_scopes;" 2>/dev/null)
NEW_COUNT=$(echo "$COUNT_RESULT" | grep -E '^ *[0-9]+$' | tr -d ' ')
echo ""
echo "✅ Scopes populated successfully!"
echo "Total scopes in database: $NEW_COUNT"
echo ""

# Display the scopes
echo "📋 Scopes available for API keys:"
echo "=================================="
$DB_CMD -c "SELECT name, resource, description, is_default FROM oauth2_scopes ORDER BY resource, name;" 2>/dev/null || true

echo ""
echo "✅ Done! Scopes are now available in the API Key Management page."