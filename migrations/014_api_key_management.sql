-- DEPRECATED: This migration file has been superseded by internal/database/migrations.go
-- All API key management tables from this file are now automatically created by the Go migration system
-- This file is kept for reference only - DO NOT EXECUTE MANUALLY
--
-- API Key Management Migration
-- This migration creates tables for API key management with scopes and analytics

-- API keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    key_id VARCHAR(64) NOT NULL UNIQUE,
    key_hash VARCHAR(128) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id BIGINT REFERENCES tenants(id) ON DELETE CASCADE,
    scopes TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip INET,
    usage_count BIGINT DEFAULT 0,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    rate_limit_per_day INTEGER DEFAULT 10000,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for API keys
CREATE INDEX IF NOT EXISTS idx_api_keys_key_id ON api_keys(key_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_scopes ON api_keys USING gin(scopes);

-- API key usage logs table for analytics
CREATE TABLE IF NOT EXISTS api_key_usage_logs (
    id BIGSERIAL PRIMARY KEY,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    request_size_bytes INTEGER,
    response_size_bytes INTEGER,
    ip_address INET,
    user_agent TEXT,
    error_message TEXT,
    request_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for usage logs (optimized for analytics queries)
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_api_key_id ON api_key_usage_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_timestamp ON api_key_usage_logs(request_timestamp);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_endpoint ON api_key_usage_logs(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_status_code ON api_key_usage_logs(status_code);

-- API key rate limiting table
CREATE TABLE IF NOT EXISTS api_key_rate_limits (
    id BIGSERIAL PRIMARY KEY,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    time_window VARCHAR(20) NOT NULL, -- 'hour', 'day', 'month'
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    request_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(api_key_id, time_window, window_start)
);

-- Index for rate limiting lookups
CREATE INDEX IF NOT EXISTS idx_api_key_rate_limits_lookup ON api_key_rate_limits(api_key_id, time_window, window_start);

-- API scopes table for predefined scopes
CREATE TABLE IF NOT EXISTS api_scopes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(200) NOT NULL,
    description TEXT,
    resource VARCHAR(100) NOT NULL,
    actions TEXT[] DEFAULT '{}',
    is_default BOOLEAN DEFAULT false,
    is_sensitive BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API key scope assignments (many-to-many relationship)
CREATE TABLE IF NOT EXISTS api_key_scopes (
    id BIGSERIAL PRIMARY KEY,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    scope_id BIGINT NOT NULL REFERENCES api_scopes(id) ON DELETE CASCADE,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(api_key_id, scope_id)
);

-- API key revocation table for audit trail
CREATE TABLE IF NOT EXISTS api_key_revocations (
    id BIGSERIAL PRIMARY KEY,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    revoked_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    revocation_reason VARCHAR(255),
    revoked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

-- Insert default API scopes
INSERT INTO api_scopes (name, display_name, description, resource, actions, is_default) VALUES
-- Artifact management scopes
('artifacts:read', 'Read Artifacts', 'View and download artifacts', 'artifacts', ARRAY['read', 'download'], true),
('artifacts:write', 'Write Artifacts', 'Upload and update artifacts', 'artifacts', ARRAY['create', 'update'], false),
('artifacts:delete', 'Delete Artifacts', 'Delete artifacts', 'artifacts', ARRAY['delete'], false),

-- Vulnerability scanning scopes
('scan:read', 'Read Scan Results', 'View vulnerability scan results', 'scans', ARRAY['read'], true),
('scan:execute', 'Execute Scans', 'Trigger vulnerability scans', 'scans', ARRAY['create', 'execute'], false),

-- User management scopes
('users:read', 'Read Users', 'View user information', 'users', ARRAY['read'], false),
('users:write', 'Manage Users', 'Create and update users', 'users', ARRAY['create', 'update'], false),

-- Repository management scopes
('repositories:read', 'Read Repositories', 'View repository information', 'repositories', ARRAY['read'], true),
('repositories:write', 'Manage Repositories', 'Create and update repositories', 'repositories', ARRAY['create', 'update'], false),

-- Compliance and audit scopes
('compliance:read', 'Read Compliance', 'View compliance reports', 'compliance', ARRAY['read'], false),
('audit:read', 'Read Audit Logs', 'View audit logs', 'audit', ARRAY['read'], false),

-- Admin scopes (sensitive)
('admin:system', 'System Administration', 'Full system administration access', 'system', ARRAY['*'], false),
('admin:users', 'User Administration', 'Full user management access', 'users', ARRAY['*'], false)

ON CONFLICT (name) DO NOTHING;

-- Update scopes marked as sensitive
UPDATE api_scopes SET is_sensitive = true 
WHERE name IN ('admin:system', 'admin:users', 'users:write', 'artifacts:delete');

-- Create views for analytics

-- Daily API usage summary
CREATE OR REPLACE VIEW daily_api_usage AS
SELECT 
    ak.id as api_key_id,
    ak.name as api_key_name,
    ak.user_id,
    u.username,
    DATE(aul.request_timestamp) as usage_date,
    COUNT(*) as total_requests,
    COUNT(CASE WHEN aul.status_code >= 200 AND aul.status_code < 300 THEN 1 END) as successful_requests,
    COUNT(CASE WHEN aul.status_code >= 400 THEN 1 END) as error_requests,
    AVG(aul.response_time_ms) as avg_response_time,
    SUM(aul.request_size_bytes) as total_request_bytes,
    SUM(aul.response_size_bytes) as total_response_bytes
FROM api_keys ak
JOIN api_key_usage_logs aul ON ak.id = aul.api_key_id
LEFT JOIN users u ON ak.user_id = u.id
GROUP BY ak.id, ak.name, ak.user_id, u.username, DATE(aul.request_timestamp);

-- API endpoint usage summary
CREATE OR REPLACE VIEW api_endpoint_usage AS
SELECT 
    aul.endpoint,
    aul.method,
    COUNT(*) as total_requests,
    COUNT(DISTINCT aul.api_key_id) as unique_api_keys,
    AVG(aul.response_time_ms) as avg_response_time,
    COUNT(CASE WHEN aul.status_code >= 200 AND aul.status_code < 300 THEN 1 END) as successful_requests,
    COUNT(CASE WHEN aul.status_code >= 400 AND aul.status_code < 500 THEN 1 END) as client_errors,
    COUNT(CASE WHEN aul.status_code >= 500 THEN 1 END) as server_errors,
    DATE_TRUNC('hour', aul.request_timestamp) as hour_bucket
FROM api_key_usage_logs aul
WHERE aul.request_timestamp >= NOW() - INTERVAL '7 days'
GROUP BY aul.endpoint, aul.method, DATE_TRUNC('hour', aul.request_timestamp);

-- Functions for API key management

-- Function to generate API key ID
CREATE OR REPLACE FUNCTION generate_api_key_id()
RETURNS VARCHAR(64) AS $$
DECLARE
    chars TEXT := 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    result TEXT := '';
    i INTEGER;
BEGIN
    FOR i IN 1..64 LOOP
        result := result || substr(chars, floor(random() * length(chars) + 1)::integer, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup old usage logs (for maintenance)
CREATE OR REPLACE FUNCTION cleanup_old_api_usage_logs(days_to_keep INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM api_key_usage_logs 
    WHERE request_timestamp < NOW() - (days_to_keep || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to reset rate limits (called by scheduler)
CREATE OR REPLACE FUNCTION reset_rate_limits(window_type VARCHAR(20))
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
    cutoff_time TIMESTAMP WITH TIME ZONE;
BEGIN
    CASE window_type
        WHEN 'hour' THEN
            cutoff_time := DATE_TRUNC('hour', NOW()) - INTERVAL '1 hour';
        WHEN 'day' THEN
            cutoff_time := DATE_TRUNC('day', NOW()) - INTERVAL '1 day';
        WHEN 'month' THEN
            cutoff_time := DATE_TRUNC('month', NOW()) - INTERVAL '1 month';
        ELSE
            RAISE EXCEPTION 'Invalid window_type: %', window_type;
    END CASE;
    
    DELETE FROM api_key_rate_limits 
    WHERE time_window = window_type AND window_start < cutoff_time;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to check rate limit
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_api_key_id BIGINT,
    p_window_type VARCHAR(20),
    p_limit INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    current_count INTEGER;
    window_start_time TIMESTAMP WITH TIME ZONE;
BEGIN
    CASE p_window_type
        WHEN 'hour' THEN
            window_start_time := DATE_TRUNC('hour', NOW());
        WHEN 'day' THEN
            window_start_time := DATE_TRUNC('day', NOW());
        ELSE
            RAISE EXCEPTION 'Invalid window_type: %', p_window_type;
    END CASE;
    
    -- Get or create rate limit record
    INSERT INTO api_key_rate_limits (api_key_id, time_window, window_start, request_count)
    VALUES (p_api_key_id, p_window_type, window_start_time, 1)
    ON CONFLICT (api_key_id, time_window, window_start)
    DO UPDATE SET 
        request_count = api_key_rate_limits.request_count + 1,
        updated_at = NOW()
    RETURNING request_count INTO current_count;
    
    RETURN current_count <= p_limit;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic updates

-- Update updated_at on api_keys
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;
CREATE TRIGGER update_api_keys_updated_at
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Update updated_at on api_scopes
DROP TRIGGER IF EXISTS update_api_scopes_updated_at ON api_scopes;
CREATE TRIGGER update_api_scopes_updated_at
    BEFORE UPDATE ON api_scopes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Update updated_at on api_key_rate_limits
DROP TRIGGER IF EXISTS update_api_key_rate_limits_updated_at ON api_key_rate_limits;
CREATE TRIGGER update_api_key_rate_limits_updated_at
    BEFORE UPDATE ON api_key_rate_limits
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_hourly ON api_key_usage_logs(DATE_TRUNC('hour', request_timestamp), api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_daily ON api_key_usage_logs(DATE_TRUNC('day', request_timestamp), api_key_id);

-- Partial indexes for active API keys
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(user_id, tenant_id) WHERE is_active = true AND (expires_at IS NULL OR expires_at > NOW());