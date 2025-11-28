-- Migration: 023_artifact_signing_integrity.sql
-- Description: Add artifact signing and integrity verification support
-- Features:
--   - SHA-256 and SHA-512 hash storage for artifacts
--   - Artifact signatures table (Cosign, PGP, Sigstore)
--   - Repository signature policy enforcement
--   - Signature verification metadata

-- =====================================================
-- 1. Extend artifacts table with integrity hashes
-- =====================================================

-- Add SHA-256 and SHA-512 hash columns to artifacts table
ALTER TABLE artifacts
ADD COLUMN IF NOT EXISTS sha256_hash VARCHAR(64),
ADD COLUMN IF NOT EXISTS sha512_hash VARCHAR(128),
ADD COLUMN IF NOT EXISTS hash_algorithm VARCHAR(20) DEFAULT 'sha256',
ADD COLUMN IF NOT EXISTS signature_required BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS signature_verified BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS signature_verified_at TIMESTAMP;

-- Create index for hash lookups (content addressable storage)
CREATE INDEX IF NOT EXISTS idx_artifacts_sha256_hash ON artifacts(sha256_hash);
CREATE INDEX IF NOT EXISTS idx_artifacts_sha512_hash ON artifacts(sha512_hash);
CREATE INDEX IF NOT EXISTS idx_artifacts_signature_verified ON artifacts(signature_verified);

COMMENT ON COLUMN artifacts.sha256_hash IS 'SHA-256 hash of artifact content (canonical)';
COMMENT ON COLUMN artifacts.sha512_hash IS 'SHA-512 hash of artifact content (optional)';
COMMENT ON COLUMN artifacts.hash_algorithm IS 'Primary hash algorithm used for integrity verification';
COMMENT ON COLUMN artifacts.signature_required IS 'Whether this artifact requires a valid signature';
COMMENT ON COLUMN artifacts.signature_verified IS 'Whether artifact signature has been verified';
COMMENT ON COLUMN artifacts.signature_verified_at IS 'Timestamp when signature was last verified';

-- =====================================================
-- 2. Add signature policy to repositories
-- =====================================================

ALTER TABLE repositories
ADD COLUMN IF NOT EXISTS signature_policy VARCHAR(20) DEFAULT 'optional' CHECK (signature_policy IN ('disabled', 'optional', 'required', 'strict')),
ADD COLUMN IF NOT EXISTS allowed_signers TEXT[], -- Array of allowed signer identities/public keys
ADD COLUMN IF NOT EXISTS signature_verification_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS cosign_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS pgp_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS sigstore_enabled BOOLEAN DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_repositories_signature_policy ON repositories(signature_policy);

COMMENT ON COLUMN repositories.signature_policy IS 'Signature requirement policy: disabled, optional, required, strict';
COMMENT ON COLUMN repositories.allowed_signers IS 'List of allowed signer identities (email, key fingerprint, or public key URL)';
COMMENT ON COLUMN repositories.signature_verification_enabled IS 'Enable automatic signature verification on upload';
COMMENT ON COLUMN repositories.cosign_enabled IS 'Enable Cosign signature support for container images';
COMMENT ON COLUMN repositories.pgp_enabled IS 'Enable PGP signature support for packages';
COMMENT ON COLUMN repositories.sigstore_enabled IS 'Enable Sigstore attestation support';

-- =====================================================
-- 3. Create artifact_signatures table
-- =====================================================

CREATE TABLE IF NOT EXISTS artifact_signatures (
    signature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(repository_id) ON DELETE CASCADE,
    
    -- Signature metadata
    signature_type VARCHAR(20) NOT NULL CHECK (signature_type IN ('cosign', 'pgp', 'sigstore', 'x509', 'ssh')),
    signature_format VARCHAR(20) NOT NULL CHECK (signature_format IN ('binary', 'ascii-armor', 'json', 'pem', 'der')),
    signature_data BYTEA NOT NULL, -- The actual signature bytes
    signature_algorithm VARCHAR(50), -- e.g., 'RSA-SHA256', 'Ed25519', 'ECDSA-P256'
    
    -- Signer information
    signer_identity VARCHAR(500), -- Email, key ID, or certificate subject
    signer_fingerprint VARCHAR(128), -- Key or certificate fingerprint
    public_key TEXT, -- Public key or certificate (PEM format)
    public_key_url TEXT, -- URL to public key (e.g., cosign transparency log)
    
    -- Verification metadata
    verified BOOLEAN DEFAULT false,
    verification_method VARCHAR(50), -- 'online', 'offline', 'transparency-log'
    verification_status VARCHAR(20) DEFAULT 'pending' CHECK (verification_status IN ('pending', 'valid', 'invalid', 'expired', 'revoked', 'untrusted')),
    verification_error TEXT,
    verified_at TIMESTAMP,
    verified_by UUID REFERENCES users(id), -- User or system that verified
    
    -- Cosign-specific fields
    cosign_bundle JSONB, -- Cosign bundle with transparency log entry
    cosign_certificate TEXT, -- X.509 certificate used for signing
    cosign_signature_digest VARCHAR(64), -- SHA-256 of signature payload
    rekor_log_index BIGINT, -- Rekor transparency log index
    rekor_uuid VARCHAR(100), -- Rekor entry UUID
    
    -- PGP-specific fields
    pgp_key_id VARCHAR(40), -- PGP key ID
    pgp_key_fingerprint VARCHAR(40), -- PGP key fingerprint
    pgp_signature_version INTEGER, -- PGP signature version
    
    -- Sigstore attestation fields
    sigstore_bundle JSONB, -- Sigstore bundle with DSSE envelope
    sigstore_predicate_type VARCHAR(200), -- SLSA provenance, vulnerability scan, etc.
    attestation_payload JSONB, -- Decoded attestation payload
    
    -- Blob storage
    signature_storage_path TEXT, -- Path to signature blob in storage
    certificate_storage_path TEXT, -- Path to certificate blob
    
    -- Timestamps
    signed_at TIMESTAMP, -- When artifact was originally signed
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP, -- Signature expiration (if applicable)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for artifact_signatures
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_tenant_id ON artifact_signatures(tenant_id);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_artifact_id ON artifact_signatures(artifact_id);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_repository_id ON artifact_signatures(repository_id);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_type ON artifact_signatures(signature_type);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_verified ON artifact_signatures(verified);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_verification_status ON artifact_signatures(verification_status);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_signer_identity ON artifact_signatures(signer_identity);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_signer_fingerprint ON artifact_signatures(signer_fingerprint);
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_expires_at ON artifact_signatures(expires_at);

-- Composite index for common queries
CREATE INDEX IF NOT EXISTS idx_artifact_signatures_lookup ON artifact_signatures(tenant_id, artifact_id, signature_type);

-- =====================================================
-- 4. Create public_keys table for trusted signers
-- =====================================================

CREATE TABLE IF NOT EXISTS public_keys (
    key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    repository_id UUID REFERENCES repositories(repository_id) ON DELETE CASCADE, -- NULL = tenant-wide
    
    -- Key metadata
    key_name VARCHAR(255) NOT NULL,
    key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('pgp', 'x509', 'cosign', 'ssh', 'jwk')),
    key_format VARCHAR(20) NOT NULL CHECK (key_format IN ('pem', 'der', 'ascii-armor', 'jwk', 'openssh')),
    public_key TEXT NOT NULL, -- The actual public key data
    
    -- Key identification
    key_fingerprint VARCHAR(128) NOT NULL,
    key_id_short VARCHAR(40), -- Short key ID (PGP, etc.)
    key_algorithm VARCHAR(50), -- RSA, Ed25519, ECDSA, etc.
    key_size INTEGER, -- Key size in bits
    
    -- Signer identity
    owner_email VARCHAR(255),
    owner_name VARCHAR(255),
    organization VARCHAR(255),
    
    -- Trust and status
    trusted BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMP,
    revocation_reason TEXT,
    
    -- Validity period
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    
    -- Metadata
    description TEXT,
    key_source VARCHAR(100), -- 'manual', 'keyserver', 'transparency-log', 'ca'
    key_source_url TEXT,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id),
    
    UNIQUE(tenant_id, key_fingerprint)
);

-- Indexes for public_keys
CREATE INDEX IF NOT EXISTS idx_public_keys_tenant_id ON public_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_public_keys_repository_id ON public_keys(repository_id);
CREATE INDEX IF NOT EXISTS idx_public_keys_fingerprint ON public_keys(key_fingerprint);
CREATE INDEX IF NOT EXISTS idx_public_keys_trusted ON public_keys(trusted);
CREATE INDEX IF NOT EXISTS idx_public_keys_enabled ON public_keys(enabled);
CREATE INDEX IF NOT EXISTS idx_public_keys_owner_email ON public_keys(owner_email);

-- =====================================================
-- 5. Create signature_verification_logs table
-- =====================================================

CREATE TABLE IF NOT EXISTS signature_verification_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    signature_id UUID REFERENCES artifact_signatures(signature_id) ON DELETE CASCADE,
    
    -- Verification details
    verification_type VARCHAR(50) NOT NULL, -- 'upload', 'download', 'periodic', 'manual'
    verification_result VARCHAR(20) NOT NULL CHECK (verification_result IN ('success', 'failure', 'error')),
    verification_status VARCHAR(20) NOT NULL,
    verification_method VARCHAR(50),
    
    -- Error details
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Context
    verified_by UUID REFERENCES users(id),
    client_ip VARCHAR(45),
    user_agent VARCHAR(500),
    
    -- Timestamps
    verified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for signature_verification_logs
CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_tenant_id ON signature_verification_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_artifact_id ON signature_verification_logs(artifact_id);
CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_signature_id ON signature_verification_logs(signature_id);
CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_result ON signature_verification_logs(verification_result);
CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_verified_at ON signature_verification_logs(verified_at);

-- =====================================================
-- 6. Create views for signature status
-- =====================================================

-- View: Artifacts with signature status
CREATE OR REPLACE VIEW artifacts_with_signatures AS
SELECT 
    a.id AS artifact_id,
    a.tenant_id,
    a.name,
    a.version,
    a.type,
    a.repository_id,
    a.sha256_hash,
    a.sha512_hash,
    a.signature_required,
    a.signature_verified,
    a.signature_verified_at,
    COUNT(DISTINCT asig.signature_id) AS signature_count,
    COUNT(DISTINCT CASE WHEN asig.verified = true THEN asig.signature_id END) AS verified_signature_count,
    COUNT(DISTINCT CASE WHEN asig.verification_status = 'valid' THEN asig.signature_id END) AS valid_signature_count,
    MAX(asig.verified_at) AS last_verification_at,
    BOOL_OR(asig.verified) AS has_verified_signature,
    ARRAY_AGG(DISTINCT asig.signature_type) FILTER (WHERE asig.signature_type IS NOT NULL) AS signature_types
FROM artifacts a
LEFT JOIN artifact_signatures asig ON a.id = asig.artifact_id
GROUP BY a.id, a.tenant_id, a.name, a.version, a.type, a.repository_id, 
         a.sha256_hash, a.sha512_hash, a.signature_required, 
         a.signature_verified, a.signature_verified_at;

-- View: Repository signature policies
CREATE OR REPLACE VIEW repository_signature_policies AS
SELECT 
    r.repository_id,
    r.tenant_id,
    r.name AS repository_name,
    r.type AS repository_type,
    r.signature_policy,
    r.signature_verification_enabled,
    r.cosign_enabled,
    r.pgp_enabled,
    r.sigstore_enabled,
    r.allowed_signers,
    COUNT(DISTINCT a.id) AS artifact_count,
    COUNT(DISTINCT CASE WHEN a.signature_verified = true THEN a.id END) AS verified_artifact_count,
    COUNT(DISTINCT asig.signature_id) AS total_signatures
FROM repositories r
LEFT JOIN artifacts a ON r.repository_id = a.repository_id
LEFT JOIN artifact_signatures asig ON a.id = asig.artifact_id
GROUP BY r.repository_id, r.tenant_id, r.name, r.type, r.signature_policy,
         r.signature_verification_enabled, r.cosign_enabled, r.pgp_enabled,
         r.sigstore_enabled, r.allowed_signers;

-- =====================================================
-- 7. Create triggers for signature management
-- =====================================================

-- Trigger: Update artifact signature status on signature insert/update
CREATE OR REPLACE FUNCTION update_artifact_signature_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Update artifact's signature_verified flag based on signatures
    UPDATE artifacts
    SET 
        signature_verified = EXISTS (
            SELECT 1 FROM artifact_signatures
            WHERE artifact_id = NEW.artifact_id
            AND verified = true
            AND verification_status = 'valid'
        ),
        signature_verified_at = CASE
            WHEN NEW.verified = true AND NEW.verification_status = 'valid' THEN NEW.verified_at
            ELSE signature_verified_at
        END,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.artifact_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_artifact_signature_status ON artifact_signatures;
CREATE TRIGGER trigger_update_artifact_signature_status
    AFTER INSERT OR UPDATE OF verified, verification_status ON artifact_signatures
    FOR EACH ROW
    EXECUTE FUNCTION update_artifact_signature_status();

-- Trigger: Update updated_at timestamp on artifact_signatures
CREATE OR REPLACE FUNCTION update_signature_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_artifact_signatures_updated_at ON artifact_signatures;
CREATE TRIGGER trigger_artifact_signatures_updated_at
    BEFORE UPDATE ON artifact_signatures
    FOR EACH ROW
    EXECUTE FUNCTION update_signature_timestamp();

-- Trigger: Update updated_at timestamp on public_keys
DROP TRIGGER IF EXISTS trigger_public_keys_updated_at ON public_keys;
CREATE TRIGGER trigger_public_keys_updated_at
    BEFORE UPDATE ON public_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_signature_timestamp();

-- =====================================================
-- 8. Add comments for documentation
-- =====================================================

COMMENT ON TABLE artifact_signatures IS 'Stores cryptographic signatures for artifacts (Cosign, PGP, Sigstore)';
COMMENT ON TABLE public_keys IS 'Trusted public keys for signature verification';
COMMENT ON TABLE signature_verification_logs IS 'Audit log of signature verification attempts';

COMMENT ON COLUMN artifact_signatures.signature_type IS 'Type of signature: cosign (containers), pgp (packages), sigstore (attestations)';
COMMENT ON COLUMN artifact_signatures.cosign_bundle IS 'Cosign transparency log bundle for keyless signing';
COMMENT ON COLUMN artifact_signatures.sigstore_bundle IS 'Sigstore DSSE envelope with attestation';
COMMENT ON COLUMN artifact_signatures.rekor_log_index IS 'Rekor transparency log entry index for verification';

-- =====================================================
-- 9. Grant permissions
-- =====================================================

-- Grant SELECT on views to application role (if exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'securestor_app') THEN
        GRANT SELECT ON artifacts_with_signatures TO securestor_app;
        GRANT SELECT ON repository_signature_policies TO securestor_app;
    END IF;
END $$;
