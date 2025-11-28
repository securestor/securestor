-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: 010_oauth2_m2m_support.sql
-- Description: Add OAuth2 client credentials, API keys, and refresh token rotation support

-- OAuth2 clients for machine-to-machine authentication
CREATE TABLE oauth2_clients (
    id BIGSERIAL PRIMARY KEY,
    client_id VARCHAR(255) UNIQUE NOT NULL,
    client_secret_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    grant_types TEXT[] NOT NULL DEFAULT ARRAY['client_credentials'],
    scopes TEXT[] NOT NULL DEFAULT ARRAY['read'],
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- Indexes for OAuth2 clients
CREATE INDEX idx_oauth2_clients_client_id ON oauth2_clients(client_id);
CREATE INDEX idx_oauth2_clients_created_by ON oauth2_clients(created_by);
CREATE INDEX idx_oauth2_clients_active ON oauth2_clients(is_active);
CREATE INDEX idx_oauth2_clients_expires_at ON oauth2_clients(expires_at);

-- API keys for programmatic access
CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    key_id VARCHAR(255) UNIQUE NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    scopes TEXT[] NOT NULL DEFAULT ARRAY['read'],
    user_id BIGINT NOT NULL REFERENCES users(id),
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for API keys
CREATE INDEX idx_api_keys_key_id ON api_keys(key_id);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_active ON api_keys(is_active);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at);

-- Refresh token store with rotation support
CREATE TABLE refresh_token_store (
    id BIGSERIAL PRIMARY KEY,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id),
    session_id VARCHAR(255),
    client_id VARCHAR(255) REFERENCES oauth2_clients(client_id),
    expires_at TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMPTZ,
    parent_token_id BIGINT REFERENCES refresh_token_store(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for refresh token store
CREATE INDEX idx_refresh_token_store_token_hash ON refresh_token_store(token_hash);
CREATE INDEX idx_refresh_token_store_user_id ON refresh_token_store(user_id);
CREATE INDEX idx_refresh_token_store_session_id ON refresh_token_store(session_id);
CREATE INDEX idx_refresh_token_store_client_id ON refresh_token_store(client_id);
CREATE INDEX idx_refresh_token_store_expires_at ON refresh_token_store(expires_at);
CREATE INDEX idx_refresh_token_store_revoked ON refresh_token_store(is_revoked);

-- SCIM groups for IdP integration
CREATE TABLE scim_groups (
    id BIGSERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    members_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_sync_at TIMESTAMPTZ
);

-- SCIM group memberships
CREATE TABLE scim_group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES scim_groups(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_user_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

-- Indexes for SCIM tables
CREATE INDEX idx_scim_groups_external_id ON scim_groups(external_id);
CREATE INDEX idx_scim_group_members_group_id ON scim_group_members(group_id);
CREATE INDEX idx_scim_group_members_user_id ON scim_group_members(user_id);
CREATE INDEX idx_scim_group_members_external_user_id ON scim_group_members(external_user_id);

-- OAuth2 scopes table for dynamic scope management
CREATE TABLE oauth2_scopes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    resource VARCHAR(255) NOT NULL,
    actions TEXT[] NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default OAuth2 scopes
INSERT INTO oauth2_scopes (name, description, resource, actions, is_default) VALUES
('read', 'Read access to resources', '*', ARRAY['read'], true),
('write', 'Write access to resources', '*', ARRAY['read', 'write'], false),
('admin', 'Administrative access', '*', ARRAY['read', 'write', 'delete', 'admin'], false),
('artifacts:read', 'Read access to artifacts', 'artifacts', ARRAY['read'], false),
('artifacts:write', 'Write access to artifacts', 'artifacts', ARRAY['read', 'write'], false),
('artifacts:delete', 'Delete access to artifacts', 'artifacts', ARRAY['read', 'write', 'delete'], false),
('scans:read', 'Read access to scans', 'scans', ARRAY['read'], false),
('scans:write', 'Write access to scans', 'scans', ARRAY['read', 'write'], false),
('compliance:read', 'Read access to compliance', 'compliance', ARRAY['read'], false),
('compliance:write', 'Write access to compliance', 'compliance', ARRAY['read', 'write'], false);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_oauth2_clients_updated_at BEFORE UPDATE ON oauth2_clients FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_scim_groups_updated_at BEFORE UPDATE ON scim_groups FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Function to cleanup expired tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired refresh tokens
    DELETE FROM refresh_token_store 
    WHERE expires_at < NOW() - INTERVAL '1 day';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Update last used timestamps to help with cleanup
    UPDATE oauth2_clients 
    SET last_used_at = NOW() 
    WHERE client_id IN (
        SELECT DISTINCT client_id 
        FROM refresh_token_store 
        WHERE client_id IS NOT NULL 
        AND created_at > NOW() - INTERVAL '1 hour'
    );
    
    UPDATE api_keys 
    SET last_used_at = NOW() 
    WHERE key_id IN (
        SELECT key_id FROM api_keys 
        WHERE updated_at > NOW() - INTERVAL '1 hour'
    );
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- View for OAuth2 client statistics
CREATE VIEW oauth2_client_stats AS
SELECT 
    c.id,
    c.client_id,
    c.name,
    c.is_active,
    c.created_at,
    c.last_used_at,
    COUNT(rt.id) as active_tokens,
    u.username as created_by_username
FROM oauth2_clients c
LEFT JOIN refresh_token_store rt ON c.client_id = rt.client_id AND rt.is_revoked = false
LEFT JOIN users u ON c.created_by = u.id
GROUP BY c.id, c.client_id, c.name, c.is_active, c.created_at, c.last_used_at, u.username;

-- View for API key statistics
CREATE VIEW api_key_stats AS
SELECT 
    ak.id,
    ak.key_id,
    ak.name,
    ak.scopes,
    ak.is_active,
    ak.created_at,
    ak.last_used_at,
    ak.expires_at,
    u.username,
    u.email
FROM api_keys ak
JOIN users u ON ak.user_id = u.id;

-- View for active sessions with refresh tokens
CREATE VIEW active_session_tokens AS
SELECT 
    rt.id,
    rt.session_id,
    rt.user_id,
    rt.client_id,
    rt.expires_at,
    rt.created_at,
    u.username,
    u.email,
    c.name as client_name
FROM refresh_token_store rt
JOIN users u ON rt.user_id = u.id
LEFT JOIN oauth2_clients c ON rt.client_id = c.client_id
WHERE rt.is_revoked = false 
AND rt.expires_at > NOW()
ORDER BY rt.created_at DESC;

-- Create initial test OAuth2 client for development
INSERT INTO oauth2_clients (
    client_id, 
    client_secret_hash, 
    name, 
    description, 
    grant_types, 
    scopes, 
    created_by
) VALUES (
    'securestor-m2m-test',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewgFgfCdtY6NlMK.', -- bcrypt hash of 'test-secret-key-2024'
    'Test M2M Client',
    'Test OAuth2 client for machine-to-machine authentication',
    ARRAY['client_credentials'],
    ARRAY['read', 'write', 'artifacts:read', 'artifacts:write', 'scans:read'],
    (SELECT id FROM users WHERE username = 'admin@securestor.local' LIMIT 1)
) ON CONFLICT (client_id) DO NOTHING;

COMMENT ON TABLE oauth2_clients IS 'OAuth2 clients for machine-to-machine authentication';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access with scope-based permissions';
COMMENT ON TABLE refresh_token_store IS 'Secure storage for refresh tokens with rotation support';
COMMENT ON TABLE scim_groups IS 'SCIM groups synchronized from identity provider';
COMMENT ON TABLE scim_group_members IS 'SCIM group membership mappings';
COMMENT ON TABLE oauth2_scopes IS 'Available OAuth2 scopes for access control';

-- Grant permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON oauth2_clients TO securestor;
GRANT SELECT, INSERT, UPDATE, DELETE ON api_keys TO securestor;
GRANT SELECT, INSERT, UPDATE, DELETE ON refresh_token_store TO securestor;
GRANT SELECT, INSERT, UPDATE, DELETE ON scim_groups TO securestor;
GRANT SELECT, INSERT, UPDATE, DELETE ON scim_group_members TO securestor;
GRANT SELECT, INSERT, UPDATE, DELETE ON oauth2_scopes TO securestor;

GRANT USAGE, SELECT ON SEQUENCE oauth2_clients_id_seq TO securestor;
GRANT USAGE, SELECT ON SEQUENCE api_keys_id_seq TO securestor;
GRANT USAGE, SELECT ON SEQUENCE refresh_token_store_id_seq TO securestor;
GRANT USAGE, SELECT ON SEQUENCE scim_groups_id_seq TO securestor;
GRANT USAGE, SELECT ON SEQUENCE scim_group_members_id_seq TO securestor;
GRANT USAGE, SELECT ON SEQUENCE oauth2_scopes_id_seq TO securestor;