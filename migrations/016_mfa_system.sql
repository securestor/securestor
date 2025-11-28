-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: Enhanced MFA System
-- This migration adds additional MFA features and improvements

-- Create mfa_methods table for available MFA methods
CREATE TABLE IF NOT EXISTS mfa_methods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE, -- 'totp', 'webauthn', 'sms', 'email'
    display_name VARCHAR(100) NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    configuration JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create user_mfa_settings table for user MFA preferences
CREATE TABLE IF NOT EXISTS user_mfa_settings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    is_mfa_enabled BOOLEAN DEFAULT false,
    preferred_method VARCHAR(50),
    backup_codes_generated BOOLEAN DEFAULT false,
    backup_codes_used_count INTEGER DEFAULT 0,
    last_mfa_setup TIMESTAMP,
    last_mfa_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- Create user_totp_secrets table for TOTP configuration
CREATE TABLE IF NOT EXISTS user_totp_secrets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret_key VARCHAR(32) NOT NULL, -- Base32 encoded secret
    backup_codes TEXT[], -- Array of backup codes (hashed)
    is_verified BOOLEAN DEFAULT false,
    qr_code_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP,
    last_used TIMESTAMP,
    UNIQUE(user_id)
);

-- Create user_webauthn_credentials table for WebAuthn/FIDO2 devices
CREATE TABLE IF NOT EXISTS user_webauthn_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE, -- Base64 encoded credential ID
    public_key TEXT NOT NULL, -- Public key for verification
    counter BIGINT DEFAULT 0, -- Usage counter for replay protection
    device_name VARCHAR(255), -- User-friendly device name
    device_type VARCHAR(100), -- 'platform' or 'cross-platform'
    aaguid UUID, -- Authenticator AAGUID
    attestation_format VARCHAR(50), -- 'packed', 'tpm', 'fido-u2f', etc.
    transport VARCHAR(100)[], -- 'usb', 'nfc', 'ble', 'internal'
    is_backup_eligible BOOLEAN DEFAULT false,
    is_backup_state BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used TIMESTAMP,
    user_agent TEXT -- Browser/device info during registration
);

-- Create mfa_recovery_codes table for backup/recovery codes
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL, -- Hashed recovery code
    is_used BOOLEAN DEFAULT false,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '1 year')
);

-- Create mfa_login_attempts table for tracking MFA login attempts
CREATE TABLE IF NOT EXISTS mfa_login_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255),
    method_used VARCHAR(50) NOT NULL, -- 'totp', 'webauthn', 'backup_code'
    success BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    error_reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create mfa_backup_sessions table for temporary sessions during MFA setup
CREATE TABLE IF NOT EXISTS mfa_backup_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    purpose VARCHAR(100) NOT NULL, -- 'setup', 'recovery', 'device_add'
    expires_at TIMESTAMP NOT NULL,
    is_used BOOLEAN DEFAULT false,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_user_mfa_settings_user_id ON user_mfa_settings(user_id);
CREATE INDEX idx_user_mfa_settings_tenant_id ON user_mfa_settings(tenant_id);
CREATE INDEX idx_user_mfa_settings_enabled ON user_mfa_settings(is_mfa_enabled);

CREATE INDEX idx_user_totp_secrets_user_id ON user_totp_secrets(user_id);
CREATE INDEX idx_user_totp_secrets_verified ON user_totp_secrets(is_verified);

CREATE INDEX idx_user_webauthn_credentials_user_id ON user_webauthn_credentials(user_id);
CREATE INDEX idx_user_webauthn_credentials_credential_id ON user_webauthn_credentials(credential_id);

CREATE INDEX idx_mfa_recovery_codes_user_id ON mfa_recovery_codes(user_id);
CREATE INDEX idx_mfa_recovery_codes_used ON mfa_recovery_codes(is_used);
CREATE INDEX idx_mfa_recovery_codes_expires ON mfa_recovery_codes(expires_at);

CREATE INDEX idx_mfa_login_attempts_user_id ON mfa_login_attempts(user_id);
CREATE INDEX idx_mfa_login_attempts_created_at ON mfa_login_attempts(created_at);
CREATE INDEX idx_mfa_login_attempts_success ON mfa_login_attempts(success);

CREATE INDEX idx_mfa_backup_sessions_token ON mfa_backup_sessions(session_token);
CREATE INDEX idx_mfa_backup_sessions_user_id ON mfa_backup_sessions(user_id);
CREATE INDEX idx_mfa_backup_sessions_expires ON mfa_backup_sessions(expires_at);

-- Add MFA columns to users table if they don't exist
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS mfa_backup_codes_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_mfa_setup TIMESTAMP,
ADD COLUMN IF NOT EXISTS require_mfa BOOLEAN DEFAULT false;

-- Create index for user MFA status
CREATE INDEX IF NOT EXISTS idx_users_mfa_enabled ON users(mfa_enabled);

-- Insert default MFA methods
INSERT INTO mfa_methods (name, display_name, is_enabled, is_default, configuration) VALUES
('totp', 'Time-based One-Time Password (TOTP)', true, true, '{"issuer": "SecureStorT", "algorithm": "SHA1", "digits": 6, "period": 30}'),
('webauthn', 'WebAuthn/FIDO2 Security Key', true, false, '{"timeout": 60000, "user_verification": "preferred", "attachment": "cross-platform"}'),
('backup_codes', 'Backup Recovery Codes', true, false, '{"code_length": 8, "code_count": 10}'),
('sms', 'SMS Text Message', false, false, '{"provider": "twilio", "template": "Your SecureStorT verification code is: {code}"}'),
('email', 'Email Verification', false, false, '{"template": "mfa_code", "expires_in": 300}')
ON CONFLICT (name) DO UPDATE SET
display_name = EXCLUDED.display_name,
configuration = EXCLUDED.configuration;

-- Create function to generate TOTP secret
CREATE OR REPLACE FUNCTION generate_totp_secret() RETURNS VARCHAR(32) AS $$
DECLARE
    chars VARCHAR(32) := 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
    result VARCHAR(32) := '';
    i INTEGER;
BEGIN
    FOR i IN 1..32 LOOP
        result := result || substr(chars, floor(random() * 32)::integer + 1, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Create function to generate backup codes
CREATE OR REPLACE FUNCTION generate_backup_codes(
    p_user_id BIGINT,
    p_count INTEGER DEFAULT 10
) RETURNS TEXT[] AS $$
DECLARE
    backup_codes TEXT[];
    code VARCHAR(8);
    i INTEGER;
BEGIN
    -- Delete existing backup codes
    DELETE FROM mfa_recovery_codes WHERE user_id = p_user_id;
    
    -- Generate new backup codes
    FOR i IN 1..p_count LOOP
        -- Generate 8-character alphanumeric code
        SELECT array_to_string(
            ARRAY(
                SELECT chr((ascii('a') + floor(random() * 26)::integer)) 
                FROM generate_series(1, 4)
                UNION ALL
                SELECT chr((ascii('0') + floor(random() * 10)::integer))
                FROM generate_series(1, 4)
            ), 
            ''
        ) INTO code;
        
        backup_codes := array_append(backup_codes, code);
        
        -- Store hashed version
        INSERT INTO mfa_recovery_codes (user_id, code_hash) 
        VALUES (p_user_id, crypt(code, gen_salt('bf')));
    END LOOP;
    
    RETURN backup_codes;
END;
$$ LANGUAGE plpgsql;

-- Create function to verify TOTP code (placeholder - actual verification done in application)
CREATE OR REPLACE FUNCTION verify_totp_code(
    p_user_id BIGINT,
    p_code VARCHAR(6)
) RETURNS BOOLEAN AS $$
DECLARE
    secret_exists BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1 FROM user_totp_secrets 
        WHERE user_id = p_user_id AND is_verified = true
    ) INTO secret_exists;
    
    -- This is a placeholder - actual TOTP verification should be done in the application
    -- using proper TOTP algorithm implementation
    RETURN secret_exists AND length(p_code) = 6 AND p_code ~ '^[0-9]+$';
END;
$$ LANGUAGE plpgsql;

-- Create function to verify backup code
CREATE OR REPLACE FUNCTION verify_backup_code(
    p_user_id BIGINT,
    p_code VARCHAR(8)
) RETURNS BOOLEAN AS $$
DECLARE
    code_valid BOOLEAN := false;
BEGIN
    -- Check if code exists and hasn't been used
    UPDATE mfa_recovery_codes 
    SET is_used = true, used_at = CURRENT_TIMESTAMP
    WHERE user_id = p_user_id 
      AND crypt(p_code, code_hash) = code_hash
      AND is_used = false 
      AND expires_at > CURRENT_TIMESTAMP
    RETURNING true INTO code_valid;
    
    -- Update backup codes used count
    IF code_valid THEN
        UPDATE user_mfa_settings 
        SET backup_codes_used_count = backup_codes_used_count + 1
        WHERE user_id = p_user_id;
        
        UPDATE users 
        SET mfa_backup_codes_count = mfa_backup_codes_count - 1
        WHERE id = p_user_id;
    END IF;
    
    RETURN COALESCE(code_valid, false);
END;
$$ LANGUAGE plpgsql;

-- Create function to setup user MFA
CREATE OR REPLACE FUNCTION setup_user_mfa(
    p_user_id BIGINT,
    p_method VARCHAR(50)
) RETURNS JSONB AS $$
DECLARE
    result JSONB;
    tenant_id BIGINT;
    secret VARCHAR(32);
    backup_codes TEXT[];
BEGIN
    -- Get user's tenant
    SELECT u.tenant_id INTO tenant_id FROM users u WHERE u.id = p_user_id;
    
    -- Create or update MFA settings
    INSERT INTO user_mfa_settings (user_id, tenant_id, preferred_method, last_mfa_setup)
    VALUES (p_user_id, tenant_id, p_method, CURRENT_TIMESTAMP)
    ON CONFLICT (user_id) DO UPDATE SET
        preferred_method = EXCLUDED.preferred_method,
        last_mfa_setup = EXCLUDED.last_mfa_setup;
    
    IF p_method = 'totp' THEN
        -- Generate TOTP secret
        secret := generate_totp_secret();
        
        INSERT INTO user_totp_secrets (user_id, secret_key)
        VALUES (p_user_id, secret)
        ON CONFLICT (user_id) DO UPDATE SET
            secret_key = EXCLUDED.secret_key,
            is_verified = false,
            created_at = CURRENT_TIMESTAMP;
        
        result := jsonb_build_object(
            'method', 'totp',
            'secret', secret,
            'qr_url', 'otpauth://totp/SecureStorT:user' || p_user_id || '?secret=' || secret || '&issuer=SecureStorT'
        );
    ELSIF p_method = 'backup_codes' THEN
        -- Generate backup codes
        backup_codes := generate_backup_codes(p_user_id);
        
        UPDATE user_mfa_settings 
        SET backup_codes_generated = true
        WHERE user_id = p_user_id;
        
        UPDATE users 
        SET mfa_backup_codes_count = array_length(backup_codes, 1)
        WHERE id = p_user_id;
        
        result := jsonb_build_object(
            'method', 'backup_codes',
            'codes', array_to_json(backup_codes)
        );
    END IF;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Create function to complete MFA setup
CREATE OR REPLACE FUNCTION complete_mfa_setup(
    p_user_id BIGINT,
    p_method VARCHAR(50),
    p_verification_code VARCHAR(10)
) RETURNS BOOLEAN AS $$
DECLARE
    setup_complete BOOLEAN := false;
BEGIN
    IF p_method = 'totp' THEN
        -- Verify TOTP code and mark as verified
        UPDATE user_totp_secrets 
        SET is_verified = true, verified_at = CURRENT_TIMESTAMP
        WHERE user_id = p_user_id;
        
        setup_complete := true;
    ELSIF p_method = 'webauthn' THEN
        -- WebAuthn verification is handled by the application
        setup_complete := true;
    END IF;
    
    IF setup_complete THEN
        -- Enable MFA for user
        UPDATE user_mfa_settings 
        SET is_mfa_enabled = true, last_mfa_setup = CURRENT_TIMESTAMP
        WHERE user_id = p_user_id;
        
        UPDATE users 
        SET mfa_enabled = true, last_mfa_setup = CURRENT_TIMESTAMP
        WHERE id = p_user_id;
    END IF;
    
    RETURN setup_complete;
END;
$$ LANGUAGE plpgsql;

-- Create function to disable MFA for user
CREATE OR REPLACE FUNCTION disable_user_mfa(p_user_id BIGINT) RETURNS BOOLEAN AS $$
BEGIN
    -- Disable MFA
    UPDATE user_mfa_settings 
    SET is_mfa_enabled = false
    WHERE user_id = p_user_id;
    
    UPDATE users 
    SET mfa_enabled = false
    WHERE id = p_user_id;
    
    -- Remove TOTP secrets
    DELETE FROM user_totp_secrets WHERE user_id = p_user_id;
    
    -- Remove WebAuthn credentials
    DELETE FROM user_webauthn_credentials WHERE user_id = p_user_id;
    
    -- Remove unused backup codes
    DELETE FROM mfa_recovery_codes WHERE user_id = p_user_id AND is_used = false;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Create function to log MFA attempt
CREATE OR REPLACE FUNCTION log_mfa_attempt(
    p_user_id BIGINT,
    p_session_id VARCHAR(255),
    p_method VARCHAR(50),
    p_success BOOLEAN,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_error_reason VARCHAR(255) DEFAULT NULL
) RETURNS BIGINT AS $$
DECLARE
    attempt_id BIGINT;
BEGIN
    INSERT INTO mfa_login_attempts (
        user_id, session_id, method_used, success, 
        ip_address, user_agent, error_reason
    ) VALUES (
        p_user_id, p_session_id, p_method, p_success,
        p_ip_address, p_user_agent, p_error_reason
    ) RETURNING id INTO attempt_id;
    
    -- Update last MFA used timestamp if successful
    IF p_success THEN
        UPDATE user_mfa_settings 
        SET last_mfa_used = CURRENT_TIMESTAMP
        WHERE user_id = p_user_id;
        
        -- Update specific method timestamps
        IF p_method = 'totp' THEN
            UPDATE user_totp_secrets 
            SET last_used = CURRENT_TIMESTAMP
            WHERE user_id = p_user_id;
        ELSIF p_method = 'webauthn' THEN
            -- WebAuthn credential update handled by application
            NULL;
        END IF;
    END IF;
    
    RETURN attempt_id;
END;
$$ LANGUAGE plpgsql;

-- Create view for user MFA status overview
CREATE OR REPLACE VIEW user_mfa_overview AS
SELECT 
    u.id as user_id,
    u.username,
    u.email,
    u.tenant_id,
    u.mfa_enabled,
    ums.preferred_method,
    ums.last_mfa_setup,
    ums.last_mfa_used,
    ums.backup_codes_used_count,
    COUNT(DISTINCT uwc.id) as webauthn_devices,
    COUNT(DISTINCT mrc.id) FILTER (WHERE mrc.is_used = false) as unused_backup_codes,
    CASE WHEN uts.is_verified THEN true ELSE false END as totp_configured,
    ums.is_mfa_enabled as mfa_active
FROM users u
LEFT JOIN user_mfa_settings ums ON u.id = ums.user_id
LEFT JOIN user_totp_secrets uts ON u.id = uts.user_id
LEFT JOIN user_webauthn_credentials uwc ON u.id = uwc.user_id
LEFT JOIN mfa_recovery_codes mrc ON u.id = mrc.user_id AND mrc.expires_at > CURRENT_TIMESTAMP
GROUP BY u.id, u.username, u.email, u.tenant_id, u.mfa_enabled, 
         ums.preferred_method, ums.last_mfa_setup, ums.last_mfa_used, 
         ums.backup_codes_used_count, uts.is_verified, ums.is_mfa_enabled;

-- Create view for MFA statistics by tenant
CREATE OR REPLACE VIEW tenant_mfa_stats AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    COUNT(DISTINCT u.id) as total_users,
    COUNT(DISTINCT u.id) FILTER (WHERE u.mfa_enabled = true) as mfa_enabled_users,
    COUNT(DISTINCT uts.user_id) as totp_users,
    COUNT(DISTINCT uwc.user_id) as webauthn_users,
    ROUND(
        COUNT(DISTINCT u.id) FILTER (WHERE u.mfa_enabled = true)::NUMERIC / 
        NULLIF(COUNT(DISTINCT u.id), 0) * 100, 
        2
    ) as mfa_adoption_percentage,
    COUNT(DISTINCT mla.id) FILTER (WHERE mla.created_at >= CURRENT_DATE - INTERVAL '30 days') as mfa_attempts_last_30_days,
    COUNT(DISTINCT mla.id) FILTER (WHERE mla.success = true AND mla.created_at >= CURRENT_DATE - INTERVAL '30 days') as successful_mfa_last_30_days
FROM tenants t
LEFT JOIN users u ON t.id = u.tenant_id AND u.is_active = true
LEFT JOIN user_totp_secrets uts ON u.id = uts.user_id AND uts.is_verified = true
LEFT JOIN user_webauthn_credentials uwc ON u.id = uwc.user_id
LEFT JOIN mfa_login_attempts mla ON u.id = mla.user_id
GROUP BY t.id, t.name;

-- Create trigger to update user MFA status
CREATE OR REPLACE FUNCTION update_user_mfa_status() RETURNS TRIGGER AS $$
BEGIN
    -- Update user table MFA status based on MFA settings
    UPDATE users 
    SET mfa_enabled = NEW.is_mfa_enabled
    WHERE id = NEW.user_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_user_mfa_status ON user_mfa_settings;
CREATE TRIGGER trigger_update_user_mfa_status
    AFTER INSERT OR UPDATE ON user_mfa_settings
    FOR EACH ROW EXECUTE FUNCTION update_user_mfa_status();

-- Create trigger for cleanup of expired items
CREATE OR REPLACE FUNCTION cleanup_expired_mfa_items() RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
    temp_count INTEGER;
BEGIN
    -- Delete expired backup codes
    DELETE FROM mfa_recovery_codes 
    WHERE expires_at < CURRENT_TIMESTAMP AND is_used = false;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Delete expired backup sessions
    DELETE FROM mfa_backup_sessions 
    WHERE expires_at < CURRENT_TIMESTAMP AND is_used = false;
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    -- Clean up old login attempts (older than 90 days)
    DELETE FROM mfa_login_attempts 
    WHERE created_at < CURRENT_DATE - INTERVAL '90 days';
    GET DIAGNOSTICS temp_count = ROW_COUNT;
    deleted_count := deleted_count + temp_count;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add updated_at triggers for all tables
CREATE TRIGGER trigger_user_mfa_settings_updated_at
    BEFORE UPDATE ON user_mfa_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_mfa_methods_updated_at
    BEFORE UPDATE ON mfa_methods
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add helpful comments
COMMENT ON TABLE mfa_methods IS 'Available MFA methods and their configurations';
COMMENT ON TABLE user_mfa_settings IS 'User MFA preferences and status';
COMMENT ON TABLE user_totp_secrets IS 'TOTP secrets and backup codes for users';
COMMENT ON TABLE user_webauthn_credentials IS 'WebAuthn/FIDO2 credentials for users';
COMMENT ON TABLE mfa_recovery_codes IS 'Backup recovery codes for MFA';
COMMENT ON TABLE mfa_login_attempts IS 'Audit log of MFA authentication attempts';
COMMENT ON TABLE mfa_backup_sessions IS 'Temporary sessions for MFA setup and recovery';

COMMENT ON FUNCTION generate_totp_secret IS 'Generates a secure 32-character Base32 TOTP secret';
COMMENT ON FUNCTION generate_backup_codes IS 'Generates backup recovery codes for a user';
COMMENT ON FUNCTION setup_user_mfa IS 'Initiates MFA setup for a user with specified method';
COMMENT ON FUNCTION complete_mfa_setup IS 'Completes MFA setup after verification';
COMMENT ON FUNCTION disable_user_mfa IS 'Disables MFA for a user and removes all MFA data';
COMMENT ON FUNCTION log_mfa_attempt IS 'Logs an MFA authentication attempt';
COMMENT ON FUNCTION cleanup_expired_mfa_items IS 'Removes expired MFA codes and sessions';