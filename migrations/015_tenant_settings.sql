-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: Add Tenant Settings and Configuration Management
-- This migration adds tenant-specific settings and configuration options

-- Update tenants table with additional settings columns
ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS plan VARCHAR(50) DEFAULT 'basic',
ADD COLUMN IF NOT EXISTS max_users INTEGER DEFAULT 100,
ADD COLUMN IF NOT EXISTS features TEXT[] DEFAULT '{}',
ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';

-- Create indexes for tenant queries
CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan);
CREATE INDEX IF NOT EXISTS idx_tenants_active ON tenants(is_active);
CREATE INDEX IF NOT EXISTS idx_tenants_subdomain ON tenants(subdomain) WHERE subdomain IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_domain ON tenants(domain) WHERE domain IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_settings_gin ON tenants USING GIN (settings);

-- Create tenant_quotas table for tracking resource usage
CREATE TABLE IF NOT EXISTS tenant_quotas (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource_type VARCHAR(100) NOT NULL, -- 'users', 'storage', 'api_calls', 'artifacts'
    quota_limit BIGINT NOT NULL,
    current_usage BIGINT DEFAULT 0,
    reset_period VARCHAR(20) DEFAULT 'monthly', -- 'daily', 'weekly', 'monthly', 'never'
    last_reset TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, resource_type)
);

-- Create indexes for quota queries
CREATE INDEX idx_tenant_quotas_tenant_id ON tenant_quotas(tenant_id);
CREATE INDEX idx_tenant_quotas_resource_type ON tenant_quotas(resource_type);
CREATE INDEX idx_tenant_quotas_usage ON tenant_quotas(current_usage, quota_limit);

-- Create tenant_feature_flags table for feature toggles
CREATE TABLE IF NOT EXISTS tenant_feature_flags (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    feature_name VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN DEFAULT false,
    configuration JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, feature_name)
);

-- Create indexes for feature flags
CREATE INDEX idx_tenant_feature_flags_tenant_id ON tenant_feature_flags(tenant_id);
CREATE INDEX idx_tenant_feature_flags_feature ON tenant_feature_flags(feature_name);
CREATE INDEX idx_tenant_feature_flags_enabled ON tenant_feature_flags(is_enabled);

-- Create tenant_domains table for custom domain management
CREATE TABLE IF NOT EXISTS tenant_domains (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL,
    is_primary BOOLEAN DEFAULT false,
    ssl_certificate TEXT,
    ssl_private_key TEXT,
    verification_status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'verified', 'failed'
    verification_token VARCHAR(255),
    verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain)
);

-- Create indexes for domain management
CREATE INDEX idx_tenant_domains_tenant_id ON tenant_domains(tenant_id);
CREATE INDEX idx_tenant_domains_domain ON tenant_domains(domain);
CREATE INDEX idx_tenant_domains_primary ON tenant_domains(is_primary) WHERE is_primary = true;
CREATE INDEX idx_tenant_domains_verification ON tenant_domains(verification_status);

-- Create tenant_billing_info table for billing and subscription management
CREATE TABLE IF NOT EXISTS tenant_billing_info (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_name VARCHAR(100) NOT NULL,
    billing_cycle VARCHAR(20) DEFAULT 'monthly', -- 'monthly', 'yearly'
    price_per_cycle DECIMAL(10,2),
    currency VARCHAR(3) DEFAULT 'USD',
    billing_email VARCHAR(255),
    payment_method_id VARCHAR(255),
    subscription_id VARCHAR(255),
    subscription_status VARCHAR(50) DEFAULT 'active', -- 'active', 'canceled', 'past_due', 'unpaid'
    current_period_start TIMESTAMP,
    current_period_end TIMESTAMP,
    trial_end TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id)
);

-- Create indexes for billing queries
CREATE INDEX idx_tenant_billing_tenant_id ON tenant_billing_info(tenant_id);
CREATE INDEX idx_tenant_billing_status ON tenant_billing_info(subscription_status);
CREATE INDEX idx_tenant_billing_period_end ON tenant_billing_info(current_period_end);

-- Create tenant_audit_logs table for tenant-specific audit logging
CREATE TABLE IF NOT EXISTS tenant_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for audit log queries
CREATE INDEX idx_tenant_audit_logs_tenant_id ON tenant_audit_logs(tenant_id);
CREATE INDEX idx_tenant_audit_logs_user_id ON tenant_audit_logs(user_id);
CREATE INDEX idx_tenant_audit_logs_action ON tenant_audit_logs(action);
CREATE INDEX idx_tenant_audit_logs_created_at ON tenant_audit_logs(created_at);
CREATE INDEX idx_tenant_audit_logs_resource ON tenant_audit_logs(resource_type, resource_id);

-- Create tenant_settings_history table for settings change tracking
CREATE TABLE IF NOT EXISTS tenant_settings_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    changed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    setting_category VARCHAR(100) NOT NULL, -- 'security', 'user_management', etc.
    old_settings JSONB,
    new_settings JSONB,
    change_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for settings history
CREATE INDEX idx_tenant_settings_history_tenant_id ON tenant_settings_history(tenant_id);
CREATE INDEX idx_tenant_settings_history_category ON tenant_settings_history(setting_category);
CREATE INDEX idx_tenant_settings_history_created_at ON tenant_settings_history(created_at);

-- Create function to update quota usage
CREATE OR REPLACE FUNCTION update_tenant_quota_usage(
    p_tenant_id BIGINT,
    p_resource_type VARCHAR,
    p_delta BIGINT
) RETURNS BOOLEAN AS $$
DECLARE
    current_limit BIGINT;
    new_usage BIGINT;
BEGIN
    -- Get current quota and update usage
    UPDATE tenant_quotas 
    SET current_usage = current_usage + p_delta,
        updated_at = CURRENT_TIMESTAMP
    WHERE tenant_id = p_tenant_id AND resource_type = p_resource_type
    RETURNING current_usage, quota_limit INTO new_usage, current_limit;
    
    -- Check if quota is exceeded
    IF new_usage > current_limit THEN
        -- Rollback the usage update
        UPDATE tenant_quotas 
        SET current_usage = current_usage - p_delta
        WHERE tenant_id = p_tenant_id AND resource_type = p_resource_type;
        
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Create function to reset quotas based on reset period
CREATE OR REPLACE FUNCTION reset_tenant_quotas() RETURNS INTEGER AS $$
DECLARE
    reset_count INTEGER := 0;
    quota_record RECORD;
BEGIN
    FOR quota_record IN 
        SELECT id, tenant_id, resource_type, reset_period, last_reset
        FROM tenant_quotas 
        WHERE reset_period != 'never'
    LOOP
        -- Check if reset is needed based on period
        IF (quota_record.reset_period = 'daily' AND quota_record.last_reset < CURRENT_DATE) OR
           (quota_record.reset_period = 'weekly' AND quota_record.last_reset < DATE_TRUNC('week', CURRENT_DATE)) OR
           (quota_record.reset_period = 'monthly' AND quota_record.last_reset < DATE_TRUNC('month', CURRENT_DATE)) THEN
            
            UPDATE tenant_quotas 
            SET current_usage = 0,
                last_reset = CURRENT_TIMESTAMP,
                updated_at = CURRENT_TIMESTAMP
            WHERE id = quota_record.id;
            
            reset_count := reset_count + 1;
        END IF;
    END LOOP;
    
    RETURN reset_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to get tenant feature flags
CREATE OR REPLACE FUNCTION get_tenant_features(p_tenant_id BIGINT) 
RETURNS TABLE(feature_name VARCHAR, is_enabled BOOLEAN, configuration JSONB) AS $$
BEGIN
    RETURN QUERY
    SELECT tff.feature_name, tff.is_enabled, tff.configuration
    FROM tenant_feature_flags tff
    WHERE tff.tenant_id = p_tenant_id
    ORDER BY tff.feature_name;
END;
$$ LANGUAGE plpgsql;

-- Create function to check if tenant feature is enabled
CREATE OR REPLACE FUNCTION is_tenant_feature_enabled(
    p_tenant_id BIGINT,
    p_feature_name VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    feature_enabled BOOLEAN := false;
BEGIN
    SELECT is_enabled INTO feature_enabled
    FROM tenant_feature_flags
    WHERE tenant_id = p_tenant_id AND feature_name = p_feature_name;
    
    RETURN COALESCE(feature_enabled, false);
END;
$$ LANGUAGE plpgsql;

-- Create function to log tenant audit events
CREATE OR REPLACE FUNCTION log_tenant_audit(
    p_tenant_id BIGINT,
    p_user_id BIGINT,
    p_action VARCHAR,
    p_resource_type VARCHAR DEFAULT NULL,
    p_resource_id VARCHAR DEFAULT NULL,
    p_old_values JSONB DEFAULT NULL,
    p_new_values JSONB DEFAULT NULL,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_request_id VARCHAR DEFAULT NULL
) RETURNS BIGINT AS $$
DECLARE
    audit_id BIGINT;
BEGIN
    INSERT INTO tenant_audit_logs (
        tenant_id, user_id, action, resource_type, resource_id,
        old_values, new_values, ip_address, user_agent, request_id
    ) VALUES (
        p_tenant_id, p_user_id, p_action, p_resource_type, p_resource_id,
        p_old_values, p_new_values, p_ip_address, p_user_agent, p_request_id
    ) RETURNING id INTO audit_id;
    
    RETURN audit_id;
END;
$$ LANGUAGE plpgsql;

-- Create view for tenant usage summary
CREATE OR REPLACE VIEW tenant_usage_summary AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    t.plan,
    t.max_users,
    COUNT(DISTINCT u.id) as current_users,
    COUNT(DISTINCT ak.id) as api_keys,
    COALESCE(SUM(tq.current_usage) FILTER (WHERE tq.resource_type = 'storage'), 0) as storage_used,
    COALESCE(SUM(tq.quota_limit) FILTER (WHERE tq.resource_type = 'storage'), 0) as storage_limit,
    CASE 
        WHEN t.max_users > 0 THEN (COUNT(DISTINCT u.id)::FLOAT / t.max_users * 100)
        ELSE 0 
    END as user_usage_percent,
    t.is_active,
    t.created_at
FROM tenants t
LEFT JOIN users u ON t.id = u.tenant_id AND u.is_active = true
LEFT JOIN api_keys ak ON t.id = ak.tenant_id AND ak.is_active = true
LEFT JOIN tenant_quotas tq ON t.id = tq.tenant_id
GROUP BY t.id, t.name, t.plan, t.max_users, t.is_active, t.created_at;

-- Create view for tenant security overview
CREATE OR REPLACE VIEW tenant_security_overview AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    (t.settings->>'security'->>'mfa_required')::BOOLEAN as mfa_required,
    (t.settings->>'security'->>'require_sso')::BOOLEAN as sso_required,
    COUNT(DISTINCT u.id) FILTER (WHERE u.mfa_enabled = true) as users_with_mfa,
    COUNT(DISTINCT u.id) as total_users,
    COUNT(DISTINCT ak.id) as active_api_keys,
    COUNT(DISTINCT tal.id) FILTER (WHERE tal.created_at >= CURRENT_DATE - INTERVAL '30 days') as recent_audit_events,
    COUNT(DISTINCT CASE WHEN tal.action LIKE '%_failed' THEN tal.id END) FILTER (WHERE tal.created_at >= CURRENT_DATE - INTERVAL '7 days') as recent_security_events
FROM tenants t
LEFT JOIN users u ON t.id = u.tenant_id AND u.is_active = true
LEFT JOIN api_keys ak ON t.id = ak.tenant_id AND ak.is_active = true
LEFT JOIN tenant_audit_logs tal ON t.id = tal.tenant_id
GROUP BY t.id, t.name, t.settings;

-- Insert default quota records for existing tenants
INSERT INTO tenant_quotas (tenant_id, resource_type, quota_limit)
SELECT id, 'users', COALESCE(max_users, 100)
FROM tenants
WHERE id NOT IN (SELECT tenant_id FROM tenant_quotas WHERE resource_type = 'users')
ON CONFLICT (tenant_id, resource_type) DO NOTHING;

INSERT INTO tenant_quotas (tenant_id, resource_type, quota_limit)
SELECT id, 'storage', 10737418240 -- 10GB in bytes
FROM tenants
WHERE id NOT IN (SELECT tenant_id FROM tenant_quotas WHERE resource_type = 'storage')
ON CONFLICT (tenant_id, resource_type) DO NOTHING;

INSERT INTO tenant_quotas (tenant_id, resource_type, quota_limit)
SELECT id, 'api_calls', 100000 -- 100k API calls per month
FROM tenants
WHERE id NOT IN (SELECT tenant_id FROM tenant_quotas WHERE resource_type = 'api_calls')
ON CONFLICT (tenant_id, resource_type) DO NOTHING;

-- Insert default feature flags for existing tenants
INSERT INTO tenant_feature_flags (tenant_id, feature_name, is_enabled)
SELECT t.id, 'api_access', true
FROM tenants t
WHERE t.id NOT IN (SELECT tenant_id FROM tenant_feature_flags WHERE feature_name = 'api_access')
ON CONFLICT (tenant_id, feature_name) DO NOTHING;

INSERT INTO tenant_feature_flags (tenant_id, feature_name, is_enabled)
SELECT t.id, 'vulnerability_scanning', true
FROM tenants t
WHERE t.id NOT IN (SELECT tenant_id FROM tenant_feature_flags WHERE feature_name = 'vulnerability_scanning')
ON CONFLICT (tenant_id, feature_name) DO NOTHING;

INSERT INTO tenant_feature_flags (tenant_id, feature_name, is_enabled)
SELECT t.id, 'compliance_reports', false
FROM tenants t
WHERE t.id NOT IN (SELECT tenant_id FROM tenant_feature_flags WHERE feature_name = 'compliance_reports')
ON CONFLICT (tenant_id, feature_name) DO NOTHING;

-- Create trigger to update tenant quotas when users are created/deleted
CREATE OR REPLACE FUNCTION update_user_quota_trigger() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.is_active = true THEN
        PERFORM update_tenant_quota_usage(NEW.tenant_id, 'users', 1);
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.is_active = false AND NEW.is_active = true THEN
            PERFORM update_tenant_quota_usage(NEW.tenant_id, 'users', 1);
        ELSIF OLD.is_active = true AND NEW.is_active = false THEN
            PERFORM update_tenant_quota_usage(NEW.tenant_id, 'users', -1);
        END IF;
    ELSIF TG_OP = 'DELETE' AND OLD.is_active = true THEN
        PERFORM update_tenant_quota_usage(OLD.tenant_id, 'users', -1);
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create triggers for quota management
DROP TRIGGER IF EXISTS trigger_update_user_quota ON users;
CREATE TRIGGER trigger_update_user_quota
    AFTER INSERT OR UPDATE OR DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION update_user_quota_trigger();

-- Create trigger to log tenant settings changes
CREATE OR REPLACE FUNCTION log_tenant_settings_changes() RETURNS TRIGGER AS $$
BEGIN
    -- Log settings changes
    IF OLD.settings IS DISTINCT FROM NEW.settings THEN
        INSERT INTO tenant_settings_history (
            tenant_id, setting_category, old_settings, new_settings, change_reason
        ) VALUES (
            NEW.id, 'general', OLD.settings, NEW.settings, 'Settings updated'
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_log_tenant_settings_changes ON tenants;
CREATE TRIGGER trigger_log_tenant_settings_changes
    AFTER UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION log_tenant_settings_changes();

-- Add updated_at trigger for all new tables
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create updated_at triggers for all tables
CREATE TRIGGER trigger_tenant_quotas_updated_at
    BEFORE UPDATE ON tenant_quotas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_tenant_feature_flags_updated_at
    BEFORE UPDATE ON tenant_feature_flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_tenant_domains_updated_at
    BEFORE UPDATE ON tenant_domains
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_tenant_billing_info_updated_at
    BEFORE UPDATE ON tenant_billing_info
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create cleanup job for old audit logs (function for scheduled execution)
CREATE OR REPLACE FUNCTION cleanup_old_tenant_audit_logs(retention_days INTEGER DEFAULT 90) RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM tenant_audit_logs 
    WHERE created_at < CURRENT_DATE - INTERVAL '1 day' * retention_days;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add helpful comments
COMMENT ON TABLE tenant_quotas IS 'Tracks resource quotas and usage for each tenant';
COMMENT ON TABLE tenant_feature_flags IS 'Manages feature toggles and configurations per tenant';
COMMENT ON TABLE tenant_domains IS 'Custom domain management for tenant branding';
COMMENT ON TABLE tenant_billing_info IS 'Billing and subscription information for tenants';
COMMENT ON TABLE tenant_audit_logs IS 'Audit trail for tenant-specific actions and changes';
COMMENT ON TABLE tenant_settings_history IS 'Historical tracking of tenant settings changes';

COMMENT ON FUNCTION update_tenant_quota_usage IS 'Updates tenant resource usage and enforces quotas';
COMMENT ON FUNCTION reset_tenant_quotas IS 'Resets quotas based on their reset period (daily/weekly/monthly)';
COMMENT ON FUNCTION get_tenant_features IS 'Returns all feature flags for a tenant';
COMMENT ON FUNCTION is_tenant_feature_enabled IS 'Checks if a specific feature is enabled for a tenant';
COMMENT ON FUNCTION log_tenant_audit IS 'Creates audit log entries for tenant actions';
COMMENT ON FUNCTION cleanup_old_tenant_audit_logs IS 'Removes old audit log entries based on retention policy';