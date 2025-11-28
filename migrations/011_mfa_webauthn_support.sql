-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: Add MFA and WebAuthn Support
-- This migration adds support for Multi-Factor Authentication (TOTP) and WebAuthn

-- MFA Methods table to track supported MFA types
CREATE TABLE IF NOT EXISTS mfa_methods (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default MFA methods
INSERT INTO mfa_methods (name, display_name, description) VALUES
('totp', 'TOTP Authenticator', 'Time-based One-Time Password using authenticator apps'),
('webauthn', 'WebAuthn/FIDO2', 'Hardware security keys and biometric authentication'),
('backup_codes', 'Backup Codes', 'Single-use recovery codes for account recovery')
ON CONFLICT (name) DO NOTHING;

-- User MFA Settings
CREATE TABLE IF NOT EXISTS user_mfa_settings (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_mfa_enabled BOOLEAN DEFAULT false,
    backup_codes_generated_at TIMESTAMP,
    backup_codes_used_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

-- TOTP Secrets for users
CREATE TABLE IF NOT EXISTS user_totp_secrets (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret_encrypted TEXT NOT NULL, -- AES encrypted TOTP secret
    is_verified BOOLEAN DEFAULT false,
    backup_codes TEXT[], -- Encrypted backup codes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMP,
    last_used_at TIMESTAMP,
    UNIQUE(user_id)
);

-- WebAuthn Credentials
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL, -- Base64 encoded credential ID
    public_key TEXT NOT NULL, -- Base64 encoded public key
    authenticator_data TEXT NOT NULL, -- Base64 encoded authenticator data
    sign_count BIGINT DEFAULT 0,
    name VARCHAR(100) NOT NULL, -- User-friendly name for the device
    device_type VARCHAR(50), -- 'platform' or 'cross-platform'
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    UNIQUE(credential_id)
);

-- MFA Authentication Attempts (for rate limiting and auditing)
CREATE TABLE IF NOT EXISTS mfa_attempts (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mfa_method VARCHAR(50) NOT NULL,
    attempt_type VARCHAR(20) NOT NULL, -- 'verify', 'setup', 'recovery'
    success BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- MFA Recovery Codes
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL, -- bcrypt hash of the recovery code
    is_used BOOLEAN DEFAULT false,
    used_at TIMESTAMP,
    used_ip INET,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- WebAuthn Challenges (temporary storage for challenge verification)
CREATE TABLE IF NOT EXISTS webauthn_challenges (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge TEXT NOT NULL,
    challenge_type VARCHAR(20) NOT NULL, -- 'registration' or 'authentication'
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, challenge_type)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_mfa_settings_user_id ON user_mfa_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_user_totp_secrets_user_id ON user_totp_secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_credential_id ON webauthn_credentials(credential_id);
CREATE INDEX IF NOT EXISTS idx_mfa_attempts_user_id_created_at ON mfa_attempts(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_mfa_recovery_codes_user_id ON mfa_recovery_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_mfa_recovery_codes_code_hash ON mfa_recovery_codes(code_hash);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_user_id ON webauthn_challenges(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_challenges_expires_at ON webauthn_challenges(expires_at);

-- Update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_mfa_settings_updated_at 
    BEFORE UPDATE ON user_mfa_settings 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add MFA requirement to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS mfa_required BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS mfa_enforced_at TIMESTAMP;

-- Create view for user MFA status
CREATE OR REPLACE VIEW user_mfa_status AS
SELECT 
    u.id as user_id,
    u.email,
    u.username,
    u.mfa_required,
    u.mfa_enforced_at,
    COALESCE(ums.is_mfa_enabled, false) as is_mfa_enabled,
    uts.is_verified as totp_verified,
    (SELECT COUNT(*) FROM webauthn_credentials wc WHERE wc.user_id = u.id AND wc.is_active = true) as webauthn_count,
    (SELECT COUNT(*) FROM mfa_recovery_codes mrc WHERE mrc.user_id = u.id AND mrc.is_used = false) as recovery_codes_count,
    ums.created_at as mfa_setup_at,
    ums.updated_at as mfa_updated_at
FROM users u
LEFT JOIN user_mfa_settings ums ON u.id = ums.user_id
LEFT JOIN user_totp_secrets uts ON u.id = uts.user_id;

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON mfa_methods TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_mfa_settings TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON user_totp_secrets TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON webauthn_credentials TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON mfa_attempts TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON mfa_recovery_codes TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON webauthn_challenges TO securestor_app;
GRANT SELECT ON user_mfa_status TO securestor_app;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO securestor_app;

-- Insert test data for development
INSERT INTO user_mfa_settings (user_id, is_mfa_enabled) 
SELECT id, false FROM users LIMIT 5
ON CONFLICT (user_id) DO NOTHING;

COMMIT;