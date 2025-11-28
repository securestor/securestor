-- DEPRECATED: This migration file has been superseded by internal/database/migrations.go
-- All RBAC tables from this file are now automatically created by the Go migration system
-- This file is kept for reference only - DO NOT EXECUTE MANUALLY
--
-- Migration 009: RBAC (Role-Based Access Control) Tables
-- Add comprehensive RBAC system with users, roles, permissions

-- Users table (OIDC user information)
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    sub VARCHAR(255) UNIQUE NOT NULL,  -- OIDC subject identifier
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    display_name VARCHAR(200),
    is_active BOOLEAN DEFAULT true,
    is_email_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Roles table (application roles)
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100),
    description TEXT,
    is_system_role BOOLEAN DEFAULT false,  -- System roles cannot be deleted
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Permissions table (granular permissions)
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    resource VARCHAR(50) NOT NULL,  -- e.g., 'artifacts', 'repositories', 'users'
    action VARCHAR(50) NOT NULL,    -- e.g., 'read', 'write', 'delete', 'admin'
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User roles junction table
CREATE TABLE IF NOT EXISTS user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by INTEGER REFERENCES users(id),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(user_id, role_id)
);

-- Role permissions junction table
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(role_id, permission_id)
);

-- User sessions table (for OIDC session management)
CREATE TABLE IF NOT EXISTS user_sessions (
    id VARCHAR(128) PRIMARY KEY,  -- Session ID
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token_hash VARCHAR(255),
    refresh_token_hash VARCHAR(255),
    id_token_hash VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_sub ON users(sub);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);

CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON permissions(resource, action);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Insert default system roles
INSERT INTO roles (name, display_name, description, is_system_role) VALUES
    ('admin', 'Administrator', 'Full system access and administration', true),
    ('developer', 'Developer', 'Read/write access to artifacts and repositories', true),
    ('auditor', 'Auditor', 'Read-only access for compliance and auditing', true),
    ('user', 'User', 'Basic user access', true)
ON CONFLICT (name) DO NOTHING;

-- Insert granular permissions
INSERT INTO permissions (name, resource, action, description) VALUES
    -- Artifact permissions
    ('artifacts:read', 'artifacts', 'read', 'View artifacts and their details'),
    ('artifacts:write', 'artifacts', 'write', 'Upload and modify artifacts'),
    ('artifacts:delete', 'artifacts', 'delete', 'Delete artifacts'),
    ('artifacts:admin', 'artifacts', 'admin', 'Full artifact administration'),
    
    -- Repository permissions
    ('repositories:read', 'repositories', 'read', 'View repositories'),
    ('repositories:write', 'repositories', 'write', 'Create and modify repositories'),
    ('repositories:delete', 'repositories', 'delete', 'Delete repositories'),
    ('repositories:admin', 'repositories', 'admin', 'Full repository administration'),
    
    -- User management permissions
    ('users:read', 'users', 'read', 'View user information'),
    ('users:write', 'users', 'write', 'Modify user information'),
    ('users:delete', 'users', 'delete', 'Delete users'),
    ('users:admin', 'users', 'admin', 'Full user administration'),
    
    -- Role management permissions
    ('roles:read', 'roles', 'read', 'View roles and permissions'),
    ('roles:write', 'roles', 'write', 'Assign and modify roles'),
    ('roles:admin', 'roles', 'admin', 'Full role administration'),
    
    -- Compliance permissions
    ('compliance:read', 'compliance', 'read', 'View compliance reports and audits'),
    ('compliance:write', 'compliance', 'write', 'Create and modify compliance policies'),
    ('compliance:admin', 'compliance', 'admin', 'Full compliance administration'),
    
    -- Security scanning permissions
    ('scans:read', 'scans', 'read', 'View security scan results'),
    ('scans:write', 'scans', 'write', 'Initiate and modify security scans'),
    ('scans:admin', 'scans', 'admin', 'Full security scan administration'),
    
    -- Policy permissions
    ('policies:read', 'policies', 'read', 'View OPA policies'),
    ('policies:write', 'policies', 'write', 'Create and modify OPA policies'),
    ('policies:admin', 'policies', 'admin', 'Full policy administration'),
    
    -- System permissions
    ('system:admin', 'system', 'admin', 'Full system administration access'),
    ('system:monitoring', 'system', 'monitoring', 'Access to system monitoring and health checks')
ON CONFLICT (name) DO NOTHING;

-- Assign permissions to roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE 
    -- Admin role gets all permissions
    (r.name = 'admin') OR
    
    -- Developer role permissions
    (r.name = 'developer' AND p.name IN (
        'artifacts:read', 'artifacts:write', 'artifacts:delete',
        'repositories:read', 'repositories:write',
        'scans:read', 'scans:write',
        'compliance:read',
        'policies:read'
    )) OR
    
    -- Auditor role permissions
    (r.name = 'auditor' AND p.name IN (
        'artifacts:read',
        'repositories:read', 
        'compliance:read', 'compliance:write',
        'scans:read',
        'policies:read',
        'users:read'
    )) OR
    
    -- User role permissions
    (r.name = 'user' AND p.name IN (
        'artifacts:read',
        'repositories:read',
        'scans:read'
    ))
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Add update timestamp trigger for users table
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language plpgsql;

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_roles_updated_at 
    BEFORE UPDATE ON roles 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Session cleanup function (remove expired sessions)
CREATE OR REPLACE FUNCTION cleanup_expired_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions 
    WHERE expires_at < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create a view for user permissions (for easy permission checking)
CREATE OR REPLACE VIEW user_permissions AS
SELECT DISTINCT
    u.id as user_id,
    u.username,
    u.email,
    p.name as permission_name,
    p.resource,
    p.action,
    r.name as role_name
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
JOIN role_permissions rp ON r.id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id
WHERE u.is_active = true
  AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP);

-- Create a view for user roles (for easy role checking)
CREATE OR REPLACE VIEW user_role_view AS
SELECT
    u.id as user_id,
    u.username,
    u.email,
    u.sub,
    r.name as role_name,
    r.display_name as role_display_name,
    ur.assigned_at,
    ur.expires_at
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
WHERE u.is_active = true
  AND (ur.expires_at IS NULL OR ur.expires_at > CURRENT_TIMESTAMP);