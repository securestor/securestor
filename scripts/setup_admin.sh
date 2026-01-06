#!/bin/bash

# SecureStore Admin User Setup Script
# This script creates the default admin user in the database with enterprise features:
# - Creates dedicated admin organization tenant
# - Maps admin user to admin tenant for proper multi-tenancy
# - Generates secure bcrypt password hash
# - Sets up comprehensive RBAC system with 6 roles and 30+ permissions
# - Validates tenant mapping and authentication setup

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-securestor}"
DB_USER="${DB_USER:-securestor}"
DB_PASSWORD="${DB_PASSWORD:-securestor123}"
DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-docker-compose.yml}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-securestor_postgres-primary_1}"

ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@securestor.io}"
ADMIN_FIRST_NAME="${ADMIN_FIRST_NAME:-Admin}"
ADMIN_LAST_NAME="${ADMIN_LAST_NAME:-User}"

# Function to generate bcrypt hash
generate_bcrypt_hash() {
    local password="$1"
    docker run --rm python:3.9-alpine sh -c "pip install bcrypt >/dev/null 2>&1 && python -c 'import bcrypt; print(bcrypt.hashpw(b\"${password}\", bcrypt.gensalt(rounds=12)).decode())'" 2>/dev/null
}

# Generate bcrypt hash for the password
echo "Generating bcrypt hash for password..."
ADMIN_PASSWORD_HASH=$(generate_bcrypt_hash "$ADMIN_PASSWORD")

if [[ -z "$ADMIN_PASSWORD_HASH" || ${#ADMIN_PASSWORD_HASH} -ne 60 ]]; then
    echo "❌ Error: Failed to generate valid bcrypt hash"
    exit 1
fi

echo "Generated hash (first 20 chars): ${ADMIN_PASSWORD_HASH:0:20}..."
echo ""

echo "🚀 Setting up admin user in SecureStore..."
echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo "Admin Username: $ADMIN_USERNAME"
echo "Admin Email: $ADMIN_EMAIL"
echo ""

# Function to check if Docker and PostgreSQL container are available
check_postgres() {
    echo "Checking Docker and PostgreSQL container..."
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        echo "❌ Error: Docker is required but not found"
        echo "   Please install Docker and ensure it's running"
        exit 1
    fi
    
    # Check if PostgreSQL container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "$POSTGRES_CONTAINER"; then
        echo "❌ Error: PostgreSQL container '$POSTGRES_CONTAINER' is not running"
        echo "   Please start the containers with: docker compose up -d"
        echo "   Or verify the container name with: docker ps"
        exit 1
    fi
    
    # Test PostgreSQL connection via Docker
    if ! docker exec -i "$POSTGRES_CONTAINER" env PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; then
        echo "❌ Error: Cannot connect to PostgreSQL database in container"
        echo "   Container: $POSTGRES_CONTAINER"
        echo "   Database: $DB_USER@$DB_NAME"
        echo "   Password: ${DB_PASSWORD:0:3}***"
        exit 1
    fi
    
    echo "✅ Docker and PostgreSQL container connection successful"
    echo ""
}

# Function to create admin user and roles
setup_admin_user() {
    echo "Creating admin user and roles..."
    
    docker exec -i "$POSTGRES_CONTAINER" env PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" << EOF

-- Create tenant first (OAuth2 scopes require tenant_id)
-- This will be used in subsequent inserts

-- Create tenant and admin user with proper mapping (UUID-based schema)
DO \$\$
DECLARE
    admin_tenant_id UUID;
    admin_user_id UUID;
    existing_user_count INTEGER;
BEGIN
    -- First, check if we have any existing admin tenant
    SELECT tenant_id INTO admin_tenant_id 
    FROM tenants 
    WHERE slug = 'admin' OR name = 'Admin Organization'
    LIMIT 1;
    
    -- If no admin tenant exists, create one
    IF admin_tenant_id IS NULL THEN
        admin_tenant_id := gen_random_uuid();
        INSERT INTO tenants (tenant_id, name, slug, description, is_active, created_at, updated_at)
        VALUES (
            admin_tenant_id,
            'Admin Organization', 
            'admin', 
            'System administration tenant for SecureStore admin users', 
            true,
            NOW(),
            NOW()
        );
        
        RAISE NOTICE 'Created admin tenant with ID: %', admin_tenant_id;
    ELSE
        RAISE NOTICE 'Using existing admin tenant with ID: %', admin_tenant_id;
    END IF;
    
    -- Check if admin user already exists in this tenant
    SELECT user_id INTO admin_user_id 
    FROM users 
    WHERE username = '$ADMIN_USERNAME' AND tenant_id = admin_tenant_id;
    
    IF admin_user_id IS NULL THEN
        -- Create new admin user
        admin_user_id := gen_random_uuid();
        INSERT INTO users (
            user_id,
            tenant_id,
            username, 
            email, 
            first_name, 
            last_name, 
            password_hash, 
            is_active, 
            is_email_verified, 
            created_at, 
            updated_at
        ) VALUES (
            admin_user_id,
            admin_tenant_id,
            '$ADMIN_USERNAME',
            '$ADMIN_EMAIL',
            '$ADMIN_FIRST_NAME',
            '$ADMIN_LAST_NAME',
            '$ADMIN_PASSWORD_HASH',
            true,
            true,
            NOW(),
            NOW()
        );
        
        RAISE NOTICE 'Created admin user with ID: % in tenant: %', admin_user_id, admin_tenant_id;
    ELSE
        -- Update existing admin user with new credentials and ensure proper tenant mapping
        UPDATE users SET
            tenant_id = admin_tenant_id,
            email = '$ADMIN_EMAIL',
            first_name = '$ADMIN_FIRST_NAME',
            last_name = '$ADMIN_LAST_NAME',
            password_hash = '$ADMIN_PASSWORD_HASH',
            is_active = true,
            is_email_verified = true,
            updated_at = NOW()
        WHERE user_id = admin_user_id;
        
        RAISE NOTICE 'Updated existing admin user ID: % with tenant mapping: %', admin_user_id, admin_tenant_id;
    END IF;
    
    -- Verify the password hash was set correctly
    SELECT COUNT(*) INTO existing_user_count
    FROM users 
    WHERE username = '$ADMIN_USERNAME' 
      AND tenant_id = admin_tenant_id 
      AND password_hash IS NOT NULL 
      AND LENGTH(password_hash) = 60;
    
    IF existing_user_count = 0 THEN
        RAISE EXCEPTION 'Failed to set password hash correctly for admin user';
    END IF;
    
    RAISE NOTICE 'Admin user setup completed successfully with tenant mapping';
END \$\$;

-- Create admin role (UUID-based)
INSERT INTO roles (role_id, tenant_id, name, display_name, description, created_at, updated_at) 
SELECT gen_random_uuid(), tenant_id, 'admin', 'System Administrator', 'Administrator role with full system access', NOW(), NOW()
FROM tenants 
WHERE slug = 'admin'
ON CONFLICT DO NOTHING;

-- Create additional roles (UUID-based)
INSERT INTO roles (role_id, tenant_id, name, display_name, description, created_at, updated_at)
SELECT gen_random_uuid(), t.tenant_id, role.name, role.display_name, role.description, NOW(), NOW()
FROM tenants t, 
     (VALUES 
        ('developer', 'Developer', 'Developer role with artifact management access'),
        ('auditor', 'Auditor', 'Auditor role with read-only compliance access'),
        ('user', 'Basic User', 'Basic user role with limited access'),
        ('viewer', 'Viewer', 'Read-only access to artifacts and reports'),
        ('compliance_manager', 'Compliance Manager', 'Compliance management and reporting access')
     ) as role(name, display_name, description)
WHERE t.slug = 'admin'
ON CONFLICT DO NOTHING;

-- Assign admin role to admin user (UUID-based)
-- Must include tenant_id since it's part of the unique constraint
INSERT INTO user_roles (user_role_id, tenant_id, user_id, role_id, assigned_by, assigned_at)
SELECT gen_random_uuid(), t.tenant_id, u.user_id, r.role_id, u.user_id, CURRENT_TIMESTAMP
FROM users u, roles r, tenants t
WHERE u.username = '$ADMIN_USERNAME' 
  AND u.tenant_id = t.tenant_id
  AND r.tenant_id = t.tenant_id
  AND r.name = 'admin'
  AND t.slug = 'admin'
ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING;

-- Create comprehensive permissions for enterprise features (UUID-based)
INSERT INTO permissions (permission_id, tenant_id, name, resource, action, description, created_at) 
SELECT gen_random_uuid(), t.tenant_id, perm.name, perm.resource, perm.action, perm.description, NOW()
FROM tenants t,
     (VALUES
        -- Artifact permissions
        ('artifacts.read', 'artifacts', 'read', 'Read artifact information'),
        ('artifacts.create', 'artifacts', 'create', 'Upload and create artifacts'),
        ('artifacts.update', 'artifacts', 'update', 'Update artifact metadata'),
        ('artifacts.delete', 'artifacts', 'delete', 'Delete artifacts'),
        ('artifacts.download', 'artifacts', 'download', 'Download artifacts'),
        -- Repository permissions
        ('repositories.read', 'repositories', 'read', 'Read repository information'),
        ('repositories.create', 'repositories', 'create', 'Create new repositories'),
        ('repositories.update', 'repositories', 'update', 'Update repository settings'),
        ('repositories.delete', 'repositories', 'delete', 'Delete repositories'),
        -- Security scan permissions
        ('scans.read', 'scans', 'read', 'Read scan results'),
        ('scans.create', 'scans', 'create', 'Initiate security scans'),
        ('scans.manage', 'scans', 'manage', 'Manage scan configurations'),
        -- Compliance permissions
        ('compliance.read', 'compliance', 'read', 'Read compliance reports and audits'),
        ('compliance.write', 'compliance', 'write', 'Create and update compliance data'),
        ('compliance.manage', 'compliance', 'manage', 'Manage compliance policies'),
        -- User management permissions
        ('users.create', 'users', 'create', 'Create new users'),
        ('users.read', 'users', 'read', 'Read user information'),
        ('users.update', 'users', 'update', 'Update user profiles'),
        ('users.delete', 'users', 'delete', 'Delete users'),
        -- Role and permission management
        ('roles.create', 'roles', 'create', 'Create new roles'),
        ('roles.read', 'roles', 'read', 'Read role information'),
        ('roles.update', 'roles', 'update', 'Update role definitions'),
        ('roles.delete', 'roles', 'delete', 'Delete roles'),
        ('permissions.manage', 'permissions', 'manage', 'Manage role permissions'),
        -- Tenant management
        ('tenants.read', 'tenants', 'read', 'Read tenant information'),
        ('tenants.update', 'tenants', 'update', 'Update tenant settings'),
        ('tenants.manage', 'tenants', 'manage', 'Full tenant administration'),
        -- System administration
        ('system.admin', 'system', 'admin', 'Full system administration access'),
        ('system.config', 'system', 'config', 'Manage system configuration'),
        ('system.monitoring', 'system', 'monitoring', 'Access system monitoring and logs'),
        -- API and integration permissions
        ('api.keys', 'api', 'keys', 'Manage API keys'),
        ('api.admin', 'api', 'admin', 'Administrative API access')
     ) as perm(name, resource, action, description)
WHERE t.slug = 'admin'
ON CONFLICT DO NOTHING;

-- Assign all permissions to admin role (UUID-based)
-- Must include tenant_id since it's part of the unique constraint
INSERT INTO role_permissions (role_perm_id, tenant_id, role_id, permission_id, granted_by, granted_at)
SELECT gen_random_uuid(), t.tenant_id, r.role_id, p.permission_id, u.user_id, CURRENT_TIMESTAMP
FROM roles r, permissions p, tenants t, users u
WHERE r.tenant_id = t.tenant_id
  AND p.tenant_id = t.tenant_id
  AND t.slug = 'admin'
  AND r.name = 'admin'
  AND u.username = '$ADMIN_USERNAME'
  AND u.tenant_id = t.tenant_id
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- Assign specific permissions to developer role
INSERT INTO role_permissions (role_perm_id, tenant_id, role_id, permission_id, granted_by, granted_at)
SELECT gen_random_uuid(), t.tenant_id, r.role_id, p.permission_id, u.user_id, CURRENT_TIMESTAMP
FROM roles r, permissions p, tenants t, users u
WHERE r.tenant_id = t.tenant_id
  AND p.tenant_id = t.tenant_id
  AND t.slug = 'admin'
  AND r.name = 'developer' 
  AND p.name IN (
    'artifacts.read', 'artifacts.create', 'artifacts.update', 'artifacts.download',
    'repositories.read', 'repositories.create', 'repositories.update',
    'scans.read', 'scans.create', 'api.keys'
  )
  AND u.username = '$ADMIN_USERNAME'
  AND u.tenant_id = t.tenant_id
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- Assign specific permissions to auditor role
INSERT INTO role_permissions (role_perm_id, tenant_id, role_id, permission_id, granted_by, granted_at)
SELECT gen_random_uuid(), t.tenant_id, r.role_id, p.permission_id, u.user_id, CURRENT_TIMESTAMP
FROM roles r, permissions p, tenants t, users u
WHERE r.tenant_id = t.tenant_id
  AND p.tenant_id = t.tenant_id
  AND t.slug = 'admin'
  AND r.name = 'auditor' 
  AND p.name IN (
    'artifacts.read', 'repositories.read', 'scans.read',
    'compliance.read', 'users.read', 'system.monitoring'
  )
  AND u.username = '$ADMIN_USERNAME'
  AND u.tenant_id = t.tenant_id
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- Assign specific permissions to basic user role
INSERT INTO role_permissions (role_perm_id, tenant_id, role_id, permission_id, granted_by, granted_at)
SELECT gen_random_uuid(), t.tenant_id, r.role_id, p.permission_id, u.user_id, CURRENT_TIMESTAMP
FROM roles r, permissions p, tenants t, users u
WHERE r.tenant_id = t.tenant_id
  AND p.tenant_id = t.tenant_id
  AND t.slug = 'admin'
  AND r.name = 'user' 
  AND p.name IN (
    'artifacts.read', 'artifacts.download', 'repositories.read'
  )
  AND u.username = '$ADMIN_USERNAME'
  AND u.tenant_id = t.tenant_id
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- Assign specific permissions to viewer role
INSERT INTO role_permissions (role_perm_id, tenant_id, role_id, permission_id, granted_by, granted_at)
SELECT gen_random_uuid(), t.tenant_id, r.role_id, p.permission_id, u.user_id, CURRENT_TIMESTAMP
FROM roles r, permissions p, tenants t, users u
WHERE r.tenant_id = t.tenant_id
  AND p.tenant_id = t.tenant_id
  AND t.slug = 'admin'  
  AND r.name = 'viewer' 
  AND p.name IN (
    'artifacts.read', 'repositories.read', 'scans.read', 'compliance.read'
  )
  AND u.username = '$ADMIN_USERNAME'
  AND u.tenant_id = t.tenant_id
ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING;

-- Output confirmation
SELECT 'Admin user created successfully with UUID-based schema' as status;
EOF
}

# Function to verify setup
verify_setup() {
    echo "Verifying admin user setup..."
    
    # Verify tenant creation
    TENANT_INFO=$(docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT t.tenant_id, t.name, t.slug, t.is_active 
        FROM tenants t 
        WHERE t.slug = 'admin' OR t.name = 'Admin Organization'
        ORDER BY t.created_at DESC LIMIT 1;
    " 2>/dev/null)
    
    if [[ -n "$TENANT_INFO" ]]; then
        echo "✅ Admin tenant verification successful"
        echo "   Tenant details: $(echo "$TENANT_INFO" | xargs)"
    else
        echo "❌ Error: Could not verify admin tenant setup"
        return 1
    fi
    
    # Verify admin user creation with tenant mapping
    ADMIN_USER_INFO=$(docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT u.user_id, u.username, u.email, u.tenant_id, t.name as tenant_name,
               CASE WHEN u.password_hash IS NOT NULL THEN 'SET' ELSE 'NULL' END as password_status,
               LENGTH(u.password_hash) as hash_length,
               SUBSTRING(u.password_hash, 1, 7) as hash_prefix
        FROM users u 
        LEFT JOIN tenants t ON u.tenant_id = t.tenant_id
        WHERE u.username = '$ADMIN_USERNAME' 
        ORDER BY u.created_at DESC LIMIT 1;
    " 2>/dev/null)
    
    if [[ -n "$ADMIN_USER_INFO" ]]; then
        echo "✅ Admin user verification successful"
        echo "   User details: $(echo "$ADMIN_USER_INFO" | xargs)"
    else
        echo "❌ Error: Could not verify admin user setup"
        return 1
    fi
    
    # Verify admin role assignment
    ROLE_ASSIGNMENT=$(docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT u.username, r.name as role, ur.assigned_at
        FROM users u 
        JOIN user_roles ur ON u.user_id = ur.user_id 
        JOIN roles r ON ur.role_id = r.role_id 
        WHERE u.username = '$ADMIN_USERNAME' AND r.name = 'admin';
    " 2>/dev/null)
    
    if [[ -n "$ROLE_ASSIGNMENT" ]]; then
        echo "✅ Admin role assignment verification successful"
        echo "   Role assignment: $(echo "$ROLE_ASSIGNMENT" | xargs)"
    else
        echo "⚠️  Warning: Could not verify admin role assignment"
    fi
    
    # Verify role and permission setup
    PERMISSION_COUNT=$(docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT COUNT(*) 
        FROM role_permissions rp 
        JOIN roles r ON rp.role_id = r.role_id 
        WHERE r.name = 'admin';
    " 2>/dev/null | xargs)
    
    echo "✅ Admin role has $PERMISSION_COUNT permissions assigned"
    
    # Verify password hash format
    HASH_VALIDATION=$(docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            CASE WHEN password_hash LIKE '\$2b\$%' OR password_hash LIKE '\$2a\$%' THEN 'VALID_BCRYPT' 
                 ELSE 'INVALID_FORMAT' END as hash_format,
            LENGTH(password_hash) as hash_length
        FROM users 
        WHERE username = '$ADMIN_USERNAME';
    " 2>/dev/null | xargs)
    
    echo "✅ Password hash validation: $HASH_VALIDATION"
    
    # Show summary of created roles
    echo "📋 Created roles:"
    docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT r.name, r.display_name, COUNT(rp.role_perm_id) as permission_count
        FROM roles r 
        LEFT JOIN role_permissions rp ON r.role_id = rp.role_id 
        GROUP BY r.role_id, r.name, r.display_name 
        ORDER BY r.name;
    " 2>/dev/null
    
    # Show tenant-user mapping summary
    echo "🏢 Tenant-User mapping:"
    docker exec -i "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT t.name as tenant_name, t.slug, 
               COUNT(u.user_id) as user_count,
               STRING_AGG(u.username, ', ') as usernames
        FROM tenants t
        LEFT JOIN users u ON t.tenant_id = u.tenant_id
        GROUP BY t.tenant_id, t.name, t.slug
        ORDER BY t.created_at;
    " 2>/dev/null
}

# Function to test API login
test_login() {
    echo "Testing API login..."
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        echo "⚠️  curl not found, skipping API test"
        return
    fi
    
    # Test login endpoint (assuming API is running on port 8080)
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"username\": \"$ADMIN_USERNAME\", \"password\": \"$ADMIN_PASSWORD\"}" \
        2>/dev/null || echo "")
    
    if [[ $LOGIN_RESPONSE == *"token"* ]]; then
        echo "✅ API login test successful"
    else
        echo "⚠️  API login test failed or API not running"
        echo "   Make sure the API server is running on http://localhost:8080"
    fi
}

# Main execution
main() {
    echo "===========================================" 
    echo "   SecureStore Admin User Setup Script"
    echo "==========================================="
    echo ""
    
    # Check dependencies
    if ! command -v docker &> /dev/null; then
        echo "❌ Error: Docker is required but not found"
        echo "   Please install Docker and ensure it's running"
        exit 1
    fi
    
    # Run setup steps
    check_postgres
    setup_admin_user
    verify_setup
    test_login
    
    echo ""
    echo "🎉 Admin user setup completed successfully!"
    echo ""
    echo "Login Credentials:"
    echo "  👤 Username: $ADMIN_USERNAME"
    echo "  🔑 Password: $ADMIN_PASSWORD"
    echo "  📧 Email: $ADMIN_EMAIL"
    echo ""
    echo "Access Points:"
    echo "  🌐 Web Interface: http://localhost:3000"
    echo "  🔌 API Endpoint: http://localhost:8080/api"
    echo ""
    echo "Enterprise Features Setup:"
    echo "  ✅ Admin user with full system permissions"
    echo "  ✅ RBAC system with 6 predefined roles"
    echo "  ✅ Comprehensive permission model (30+ permissions)"
    echo "  ✅ Tenant-aware user management with dedicated admin tenant"
    echo "  ✅ Bcrypt password hash with proper validation"
    echo "  ✅ Admin Organization tenant (slug: admin) created"
    echo ""
    echo "Available Roles:"
    echo "  🛡️  admin - Full system administration"
    echo "  🔧 developer - Artifact management access"
    echo "  📊 auditor - Read-only compliance access"
    echo "  👁️  viewer - Read-only access to artifacts"
    echo "  👤 user - Basic artifact access"
    echo "  📋 compliance_manager - Compliance management"
    echo ""
    echo "Next Steps:"
    echo "  1. Start the application: docker-compose up -d"
    echo "  2. Access the web interface and verify login works"
    echo "  3. Test artifact upload/download functionality"
    echo "  4. Set up additional users with appropriate roles"
    echo "  5. Configure compliance policies as needed"
    echo "  6. Change the default password for production use"
    echo ""
    echo "📖 For more information, see the application documentation"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "SecureStore Admin User Setup Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Environment Variables:"
        echo "  POSTGRES_CONTAINER  PostgreSQL container name (default: securestor-postgres-1)"
        echo "  DB_NAME            Database name (default: securestor)"
        echo "  DB_USER            Database user (default: securestor)"
        echo "  ADMIN_USERNAME     Admin username (default: admin)"
        echo "  ADMIN_PASSWORD     Admin password (default: admin123)"
        echo "  ADMIN_EMAIL        Admin email (default: admin@localhost)"
        echo ""
        echo "Examples:"
        echo "  $0                                    # Use defaults"
        echo "  ADMIN_PASSWORD=mypass $0              # Custom password"
        echo "  DB_HOST=myhost DB_PORT=5433 $0        # Custom database"
        exit 0
        ;;
    --version|-v)
        echo "SecureStore Admin Setup Script v1.0.0"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac