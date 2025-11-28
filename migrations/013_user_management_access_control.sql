-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: Enhanced User Management and Access Control
-- Add comprehensive user management features with fine-grained access control

-- User invitations table
CREATE TABLE IF NOT EXISTS user_invites (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    invited_by BIGINT NOT NULL REFERENCES users(id),
    invite_token VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    accepted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for quick invite token lookups
CREATE INDEX IF NOT EXISTS idx_user_invites_token ON user_invites(invite_token);
CREATE INDEX IF NOT EXISTS idx_user_invites_email ON user_invites(email);
CREATE INDEX IF NOT EXISTS idx_user_invites_expires_at ON user_invites(expires_at);

-- User invite roles association table
CREATE TABLE IF NOT EXISTS user_invite_roles (
    id BIGSERIAL PRIMARY KEY,
    invite_id BIGINT NOT NULL REFERENCES user_invites(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(invite_id, role_id)
);

-- User sessions table for session management
CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(128) NOT NULL UNIQUE,
    refresh_token VARCHAR(128),
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for session management
CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- User activity log table
CREATE TABLE IF NOT EXISTS user_activity_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    session_id BIGINT REFERENCES user_sessions(id) ON DELETE SET NULL,
    activity_type VARCHAR(50) NOT NULL,
    resource VARCHAR(100),
    action VARCHAR(50),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for activity logging
CREATE INDEX IF NOT EXISTS idx_user_activity_log_user_id ON user_activity_log(user_id);
CREATE INDEX IF NOT EXISTS idx_user_activity_log_created_at ON user_activity_log(created_at);
CREATE INDEX IF NOT EXISTS idx_user_activity_log_activity_type ON user_activity_log(activity_type);

-- Password reset tokens table
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for password reset tokens
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);

-- Enhance users table with additional fields
ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id BIGINT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS department VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS job_title VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
ALTER TABLE users ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'UTC';
ALTER TABLE users ADD COLUMN IF NOT EXISTS language VARCHAR(10) DEFAULT 'en';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS force_password_change BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS login_attempts INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP WITH TIME ZONE;

-- Add user preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    preference_key VARCHAR(100) NOT NULL,
    preference_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, preference_key)
);

-- Index for user preferences
CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);

-- Tenant management table
CREATE TABLE IF NOT EXISTS tenants (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    subdomain VARCHAR(100) UNIQUE,
    domain VARCHAR(255),
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    plan VARCHAR(50) DEFAULT 'basic',
    max_users INTEGER DEFAULT 100,
    features JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add foreign key constraint for tenant_id in users table
ALTER TABLE users ADD CONSTRAINT fk_users_tenant_id 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE SET NULL;

-- Tenant users count view for quota management
CREATE OR REPLACE VIEW tenant_user_counts AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    t.max_users,
    COUNT(u.id) as current_users,
    (t.max_users - COUNT(u.id)) as available_slots
FROM tenants t
LEFT JOIN users u ON t.id = u.tenant_id AND u.is_active = true
GROUP BY t.id, t.name, t.max_users;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email_verified ON users(is_email_verified);
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at);

-- Function to cleanup expired invites (can be called by cron job)
CREATE OR REPLACE FUNCTION cleanup_expired_invites()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM user_invites 
    WHERE expires_at < NOW() - INTERVAL '30 days' 
    AND accepted_at IS NULL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup old user sessions
CREATE OR REPLACE FUNCTION cleanup_old_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM user_sessions 
    WHERE expires_at < NOW() - INTERVAL '7 days';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to unlock user accounts after lockout period
CREATE OR REPLACE FUNCTION unlock_user_accounts()
RETURNS INTEGER AS $$
DECLARE
    unlocked_count INTEGER;
BEGIN
    UPDATE users 
    SET locked_until = NULL, login_attempts = 0
    WHERE locked_until IS NOT NULL AND locked_until < NOW();
    
    GET DIAGNOSTICS unlocked_count = ROW_COUNT;
    RETURN unlocked_count;
END;
$$ LANGUAGE plpgsql;

-- Create some default system roles if they don't exist
INSERT INTO roles (name, display_name, description, is_system_role) VALUES
('super_admin', 'Super Administrator', 'Full system access with all permissions', true),
('tenant_admin', 'Tenant Administrator', 'Full access within tenant scope', true),
('user_manager', 'User Manager', 'Can manage users and roles within tenant', true),
('viewer', 'Viewer', 'Read-only access to resources', true)
ON CONFLICT (name) DO NOTHING;

-- Create default permissions if they don't exist
INSERT INTO permissions (name, resource, action, description) VALUES
-- User management permissions
('users.create', 'users', 'create', 'Create new users'),
('users.read', 'users', 'read', 'View user information'),
('users.update', 'users', 'update', 'Update user information'),
('users.delete', 'users', 'delete', 'Delete users'),
('users.invite', 'users', 'invite', 'Invite new users'),
('users.manage_roles', 'users', 'manage_roles', 'Assign and remove user roles'),

-- Role management permissions
('roles.create', 'roles', 'create', 'Create new roles'),
('roles.read', 'roles', 'read', 'View role information'),
('roles.update', 'roles', 'update', 'Update role information'),
('roles.delete', 'roles', 'delete', 'Delete roles'),
('roles.assign_permissions', 'roles', 'assign_permissions', 'Assign permissions to roles'),

-- Tenant management permissions
('tenants.create', 'tenants', 'create', 'Create new tenants'),
('tenants.read', 'tenants', 'read', 'View tenant information'),
('tenants.update', 'tenants', 'update', 'Update tenant settings'),
('tenants.delete', 'tenants', 'delete', 'Delete tenants'),

-- System administration permissions
('system.admin', 'system', 'admin', 'Full system administration access'),
('system.audit', 'system', 'audit', 'Access audit logs and compliance reports'),
('system.settings', 'system', 'settings', 'Manage system settings')

ON CONFLICT (name) DO NOTHING;

-- Assign permissions to default roles
DO $$
DECLARE
    super_admin_role_id BIGINT;
    tenant_admin_role_id BIGINT;
    user_manager_role_id BIGINT;
    viewer_role_id BIGINT;
    perm_id BIGINT;
BEGIN
    -- Get role IDs
    SELECT id INTO super_admin_role_id FROM roles WHERE name = 'super_admin';
    SELECT id INTO tenant_admin_role_id FROM roles WHERE name = 'tenant_admin';
    SELECT id INTO user_manager_role_id FROM roles WHERE name = 'user_manager';
    SELECT id INTO viewer_role_id FROM roles WHERE name = 'viewer';

    -- Assign all permissions to super_admin
    FOR perm_id IN SELECT id FROM permissions LOOP
        INSERT INTO role_permissions (role_id, permission_id) 
        VALUES (super_admin_role_id, perm_id)
        ON CONFLICT DO NOTHING;
    END LOOP;

    -- Assign tenant-scoped permissions to tenant_admin
    FOR perm_id IN SELECT id FROM permissions WHERE resource IN ('users', 'roles', 'tenants') AND action != 'delete' LOOP
        INSERT INTO role_permissions (role_id, permission_id) 
        VALUES (tenant_admin_role_id, perm_id)
        ON CONFLICT DO NOTHING;
    END LOOP;

    -- Assign user management permissions to user_manager
    FOR perm_id IN SELECT id FROM permissions WHERE resource = 'users' LOOP
        INSERT INTO role_permissions (role_id, permission_id) 
        VALUES (user_manager_role_id, perm_id)
        ON CONFLICT DO NOTHING;
    END LOOP;

    -- Assign read permissions to viewer
    FOR perm_id IN SELECT id FROM permissions WHERE action = 'read' LOOP
        INSERT INTO role_permissions (role_id, permission_id) 
        VALUES (viewer_role_id, perm_id)
        ON CONFLICT DO NOTHING;
    END LOOP;
END
$$;

-- Create updated_at trigger for relevant tables
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply the trigger to tables that need it
DROP TRIGGER IF EXISTS update_user_invites_updated_at ON user_invites;
CREATE TRIGGER update_user_invites_updated_at
    BEFORE UPDATE ON user_invites
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_preferences_updated_at ON user_preferences;
CREATE TRIGGER update_user_preferences_updated_at
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();