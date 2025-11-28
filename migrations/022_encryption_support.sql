-- Migration: Add encryption support
-- Description: Adds tenant master keys table, encryption metadata to artifacts, and audit logging
-- Version: 022
-- Date: 2025-11-13

BEGIN;

-- Create tenant_master_keys table for storing encrypted TMKs
CREATE TABLE IF NOT EXISTS tenant_master_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,
    encrypted_key BYTEA NOT NULL,
    kms_key_id VARCHAR(255) NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMP,
    created_by UUID NOT NULL,
    last_accessed_at TIMESTAMP,
    access_count BIGINT DEFAULT 0,
    
    CONSTRAINT fk_tmk_tenant FOREIGN KEY (tenant_id) 
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_tmk_creator FOREIGN KEY (created_by) 
        REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE INDEX idx_tmk_tenant_active ON tenant_master_keys(tenant_id) 
    WHERE is_active = true;
CREATE INDEX idx_tmk_created_at ON tenant_master_keys(created_at DESC);
CREATE INDEX idx_tmk_rotation ON tenant_master_keys(rotated_at) 
    WHERE is_active = true AND rotated_at IS NOT NULL;

COMMENT ON TABLE tenant_master_keys IS 'Stores encrypted tenant master keys for envelope encryption';
COMMENT ON COLUMN tenant_master_keys.encrypted_key IS 'TMK encrypted by KMS root key';
COMMENT ON COLUMN tenant_master_keys.kms_key_id IS 'KMS key ID or ARN used to encrypt TMK';

-- Add encryption columns to artifacts table
ALTER TABLE artifacts 
    ADD COLUMN IF NOT EXISTS encrypted BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS encryption_version INTEGER DEFAULT 1,
    ADD COLUMN IF NOT EXISTS encrypted_dek BYTEA,
    ADD COLUMN IF NOT EXISTS encryption_algorithm VARCHAR(50) DEFAULT 'AES-256-GCM',
    ADD COLUMN IF NOT EXISTS encryption_metadata JSONB;

CREATE INDEX idx_artifacts_encrypted ON artifacts(encrypted) WHERE encrypted = true;
CREATE INDEX idx_artifacts_enc_version ON artifacts(encryption_version);

COMMENT ON COLUMN artifacts.encrypted IS 'Whether artifact content is encrypted';
COMMENT ON COLUMN artifacts.encrypted_dek IS 'Data encryption key encrypted by KEK';
COMMENT ON COLUMN artifacts.encryption_metadata IS 'Nonce, auth tag, and other encryption params';

-- Create key_audit_log for compliance and security monitoring
CREATE TABLE IF NOT EXISTS key_audit_log (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    key_type VARCHAR(50) NOT NULL,  -- TMK, KEK, DEK
    key_id VARCHAR(255),
    operation VARCHAR(50) NOT NULL, -- encrypt, decrypt, rotate, access, generate
    artifact_id UUID,
    repository_id UUID,
    source_ip INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    duration_ms INTEGER,
    audit_chain_hash BYTEA,  -- Hash of previous event for tamper detection
    metadata JSONB,
    
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) 
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_audit_user FOREIGN KEY (user_id) 
        REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE INDEX idx_key_audit_tenant_time ON key_audit_log(tenant_id, timestamp DESC);
CREATE INDEX idx_key_audit_user_time ON key_audit_log(user_id, timestamp DESC);
CREATE INDEX idx_key_audit_operation ON key_audit_log(operation, timestamp DESC);
CREATE INDEX idx_key_audit_artifact ON key_audit_log(artifact_id) 
    WHERE artifact_id IS NOT NULL;
CREATE INDEX idx_key_audit_success ON key_audit_log(success, timestamp DESC) 
    WHERE success = false;

COMMENT ON TABLE key_audit_log IS 'Immutable audit log for all encryption key operations';
COMMENT ON COLUMN key_audit_log.audit_chain_hash IS 'SHA-256 hash linking to previous event for tamper detection';

-- Create function to update last_accessed_at for TMKs
CREATE OR REPLACE FUNCTION update_tmk_access()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE tenant_master_keys 
    SET 
        last_accessed_at = NOW(),
        access_count = access_count + 1
    WHERE tenant_id = NEW.tenant_id 
      AND key_type = 'TMK' 
      AND is_active = true;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to track TMK access
CREATE TRIGGER trigger_tmk_access
    AFTER INSERT ON key_audit_log
    FOR EACH ROW
    WHEN (NEW.key_type = 'TMK' AND NEW.operation IN ('access', 'decrypt'))
    EXECUTE FUNCTION update_tmk_access();

-- Create encryption_settings table for tenant-specific encryption config
CREATE TABLE IF NOT EXISTS encryption_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE,
    encryption_enabled BOOLEAN DEFAULT true,
    enforce_encryption BOOLEAN DEFAULT false,
    allowed_algorithms TEXT[] DEFAULT ARRAY['AES-256-GCM'],
    key_rotation_days INTEGER DEFAULT 90,
    auto_rotate BOOLEAN DEFAULT true,
    kms_provider VARCHAR(50) DEFAULT 'mock', -- mock, aws-kms, azure-keyvault, vault
    kms_config JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by UUID,
    
    CONSTRAINT fk_enc_settings_tenant FOREIGN KEY (tenant_id) 
        REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_enc_settings_updater FOREIGN KEY (updated_by) 
        REFERENCES users(user_id) ON DELETE SET NULL,
    CONSTRAINT check_rotation_days CHECK (key_rotation_days >= 30 AND key_rotation_days <= 365)
);

CREATE INDEX idx_enc_settings_tenant ON encryption_settings(tenant_id);

COMMENT ON TABLE encryption_settings IS 'Tenant-specific encryption configuration and policies';
COMMENT ON COLUMN encryption_settings.enforce_encryption IS 'Reject unencrypted artifact uploads if true';

-- Create view for encryption status dashboard
CREATE OR REPLACE VIEW encryption_status_view AS
SELECT 
    t.tenant_id,
    t.name as tenant_name,
    es.encryption_enabled,
    es.enforce_encryption,
    es.key_rotation_days,
    tmk.key_version as current_tmk_version,
    tmk.rotated_at as last_rotation,
    EXTRACT(DAY FROM NOW() - COALESCE(tmk.rotated_at, tmk.created_at)) as days_since_rotation,
    tmk.access_count as tmk_access_count,
    COUNT(DISTINCT a.artifact_id) FILTER (WHERE a.encrypted = true) as encrypted_artifacts,
    COUNT(DISTINCT a.artifact_id) FILTER (WHERE a.encrypted = false) as unencrypted_artifacts,
    COUNT(DISTINCT kal.id) FILTER (WHERE kal.success = false AND kal.timestamp > NOW() - INTERVAL '24 hours') as failed_ops_24h
FROM tenants t
LEFT JOIN encryption_settings es ON es.tenant_id = t.tenant_id
LEFT JOIN tenant_master_keys tmk ON tmk.tenant_id = t.tenant_id AND tmk.is_active = true
LEFT JOIN artifacts a ON a.tenant_id = t.tenant_id
LEFT JOIN key_audit_log kal ON kal.tenant_id = t.tenant_id
GROUP BY t.tenant_id, t.name, es.encryption_enabled, es.enforce_encryption, 
         es.key_rotation_days, tmk.key_version, tmk.rotated_at, 
         tmk.created_at, tmk.access_count;

COMMENT ON VIEW encryption_status_view IS 'Dashboard view for encryption status per tenant';

-- Insert default encryption settings for existing tenants
INSERT INTO encryption_settings (tenant_id, encryption_enabled, enforce_encryption, kms_provider)
SELECT tenant_id, false, false, 'mock'
FROM tenants
WHERE tenant_id NOT IN (SELECT tenant_id FROM encryption_settings);

COMMIT;
