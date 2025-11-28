-- UUID-based RBAC Migration
-- This creates the complete RBAC system with UUID primary keys
-- Run this after the base UUID tables (tenants_uuid, users_uuid) are created

BEGIN;

-- Roles table with UUID support
CREATE TABLE IF NOT EXISTS roles_uuid (
    role_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    role_name           VARCHAR(50) NOT NULL,
    display_name        VARCHAR(100),
    description         TEXT,
    is_system_role      BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, role_name)
);

-- Permissions table with UUID support  
CREATE TABLE IF NOT EXISTS permissions_uuid (
    permission_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    permission_name     VARCHAR(100) NOT NULL,
    resource            VARCHAR(50) NOT NULL,
    action              VARCHAR(50) NOT NULL,
    description         TEXT,
    created_at          TIMESTAMP DEFAULT NOW(),
    UNIQUE(permission_name)
);

-- User roles junction table
CREATE TABLE IF NOT EXISTS user_roles_uuid (
    user_id             UUID NOT NULL REFERENCES users_uuid(user_id) ON DELETE CASCADE,
    role_id             UUID NOT NULL REFERENCES roles_uuid(role_id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    assigned_by         UUID REFERENCES users_uuid(user_id),
    assigned_at         TIMESTAMP DEFAULT NOW(),
    expires_at          TIMESTAMP,
    PRIMARY KEY(user_id, role_id, tenant_id)
);

-- Role permissions junction table
CREATE TABLE IF NOT EXISTS role_permissions_uuid (
    role_id             UUID NOT NULL REFERENCES roles_uuid(role_id) ON DELETE CASCADE,
    permission_id       UUID NOT NULL REFERENCES permissions_uuid(permission_id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    granted_at          TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY(role_id, permission_id, tenant_id)
);

-- User invites table with UUID support
CREATE TABLE IF NOT EXISTS user_invites_uuid (
    invite_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    email               VARCHAR(255) NOT NULL,
    first_name          VARCHAR(100),
    last_name           VARCHAR(100),
    invited_by          UUID NOT NULL REFERENCES users_uuid(user_id),
    invite_token        VARCHAR(255) UNIQUE NOT NULL,
    expires_at          TIMESTAMP NOT NULL,
    accepted_at         TIMESTAMP,
    created_at          TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- User invite roles junction table
CREATE TABLE IF NOT EXISTS user_invite_roles_uuid (
    invite_id           UUID NOT NULL REFERENCES user_invites_uuid(invite_id) ON DELETE CASCADE,
    role_id             UUID NOT NULL REFERENCES roles_uuid(role_id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    PRIMARY KEY(invite_id, role_id, tenant_id)
);

-- User sessions table with UUID support
CREATE TABLE IF NOT EXISTS user_sessions_uuid (
    session_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users_uuid(user_id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    access_token_hash   VARCHAR(255),
    refresh_token_hash  VARCHAR(255),
    id_token_hash       VARCHAR(255),
    expires_at          TIMESTAMP NOT NULL,
    created_at          TIMESTAMP DEFAULT NOW(),
    last_accessed_at    TIMESTAMP DEFAULT NOW(),
    ip_address          INET,
    user_agent          TEXT
);

-- API keys table with UUID support
CREATE TABLE IF NOT EXISTS api_keys_uuid (
    key_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users_uuid(user_id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants_uuid(tenant_id) ON DELETE CASCADE,
    key_name            VARCHAR(100) NOT NULL,
    key_hash            VARCHAR(255) NOT NULL,
    permissions         JSONB,
    expires_at          TIMESTAMP,
    last_used_at        TIMESTAMP,
    created_at          TIMESTAMP DEFAULT NOW(),
    is_active           BOOLEAN DEFAULT TRUE,
    UNIQUE(tenant_id, key_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_roles_uuid_tenant ON roles_uuid(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_uuid_user ON user_roles_uuid(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_uuid_role ON user_roles_uuid(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_uuid_tenant ON user_roles_uuid(tenant_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_uuid_role ON role_permissions_uuid(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_uuid_permission ON role_permissions_uuid(permission_id);
CREATE INDEX IF NOT EXISTS idx_user_invites_uuid_email ON user_invites_uuid(email);
CREATE INDEX IF NOT EXISTS idx_user_invites_uuid_token ON user_invites_uuid(invite_token);
CREATE INDEX IF NOT EXISTS idx_user_sessions_uuid_user ON user_sessions_uuid(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_uuid_user ON api_keys_uuid(user_id);

-- Insert default permissions
INSERT INTO permissions_uuid (permission_name, resource, action, description) VALUES
    ('artifacts.read', 'artifacts', 'read', 'Read access to artifacts'),
    ('artifacts.write', 'artifacts', 'write', 'Write access to artifacts'),
    ('artifacts.delete', 'artifacts', 'delete', 'Delete access to artifacts'),
    ('repositories.read', 'repositories', 'read', 'Read access to repositories'),
    ('repositories.write', 'repositories', 'write', 'Write access to repositories'),
    ('repositories.delete', 'repositories', 'delete', 'Delete access to repositories'),
    ('users.read', 'users', 'read', 'Read access to users'),
    ('users.write', 'users', 'write', 'Write access to users'),
    ('users.delete', 'users', 'delete', 'Delete access to users'),
    ('system.admin', 'system', 'admin', 'Full system administration access')
ON CONFLICT (permission_name) DO NOTHING;

COMMIT;