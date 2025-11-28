-- Migration: 024_artifact_properties
-- Description: Add support for artifact properties (key-value metadata)
-- Features:
--   - Support for custom properties on artifacts
--   - Multi-value properties (JSON arrays)
--   - Sensitive properties (encrypted values)
--   - System properties (read-only, managed by system)
--   - Search and indexing support
--   - Full RBAC integration
--   - Audit logging

-- ============================================
-- artifact_properties table
-- ============================================
CREATE TABLE IF NOT EXISTS artifact_properties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    repository_id VARCHAR(255) NOT NULL,
    artifact_id VARCHAR(255) NOT NULL,
    
    -- Property key-value
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    value_type VARCHAR(50) DEFAULT 'string', -- string, number, boolean, json, array
    
    -- Security and metadata
    is_sensitive BOOLEAN DEFAULT FALSE,
    is_system BOOLEAN DEFAULT FALSE,        -- System-managed properties (read-only)
    is_multi_value BOOLEAN DEFAULT FALSE,   -- Allows multiple values for same key
    
    -- Encryption metadata (for sensitive properties)
    encrypted_value TEXT,                   -- Base64-encoded encrypted value
    encryption_key_id VARCHAR(255),         -- KEK/DEK key ID for decryption
    encryption_algorithm VARCHAR(50),       -- e.g., AES-256-GCM
    nonce VARCHAR(255),                     -- Nonce/IV for decryption
    
    -- Audit fields
    created_by UUID REFERENCES users(user_id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_by UUID REFERENCES users(user_id),
    updated_at TIMESTAMP DEFAULT NOW(),
    version INTEGER DEFAULT 1,              -- Optimistic locking
    
    -- Searchable metadata
    tags TEXT[],                            -- Optional tags for categorization
    description TEXT,                       -- Optional description
    
    -- Constraints
    CONSTRAINT artifact_properties_key_format CHECK (key ~ '^[a-zA-Z0-9._-]+$'),
    CONSTRAINT artifact_properties_value_size CHECK (length(value) <= 65535)
);

-- ============================================
-- Indexes for performance
-- ============================================

-- Primary lookup: by tenant, repo, artifact
CREATE INDEX idx_artifact_properties_lookup 
ON artifact_properties(tenant_id, repository_id, artifact_id);

-- Search by key
CREATE INDEX idx_artifact_properties_key 
ON artifact_properties(tenant_id, key);

-- Search by key-value pair
CREATE INDEX idx_artifact_properties_key_value 
ON artifact_properties(tenant_id, key, value);

-- Search by artifact (for loading all properties)
CREATE INDEX idx_artifact_properties_artifact 
ON artifact_properties(tenant_id, artifact_id);

-- Search system properties
CREATE INDEX idx_artifact_properties_system 
ON artifact_properties(tenant_id, is_system);

-- Full-text search on keys and values (GIN index)
CREATE INDEX idx_artifact_properties_search 
ON artifact_properties USING GIN(to_tsvector('english', key || ' ' || value));

-- Audit tracking
CREATE INDEX idx_artifact_properties_created 
ON artifact_properties(tenant_id, created_at DESC);

-- Updated properties
CREATE INDEX idx_artifact_properties_updated 
ON artifact_properties(tenant_id, updated_at DESC);

-- ============================================
-- property_search_index (optional denormalized view)
-- ============================================
-- Materialized view for fast property searches
CREATE MATERIALIZED VIEW IF NOT EXISTS property_search_index AS
SELECT 
    ap.id,
    ap.tenant_id,
    ap.repository_id,
    ap.artifact_id,
    ap.key,
    ap.value,
    ap.value_type,
    ap.is_sensitive,
    ap.is_system,
    ap.created_at,
    ap.updated_at,
    to_tsvector('english', ap.key || ' ' || ap.value) as search_vector
FROM artifact_properties ap
WHERE ap.is_sensitive = FALSE; -- Don't index sensitive properties

CREATE INDEX idx_property_search_vector ON property_search_index USING GIN(search_vector);
CREATE INDEX idx_property_search_tenant ON property_search_index(tenant_id);

-- ============================================
-- property_audit_log table
-- ============================================
CREATE TABLE IF NOT EXISTS property_audit_log (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    property_id UUID,                      -- Reference to artifact_properties
    artifact_id VARCHAR(255) NOT NULL,
    
    -- Action details
    action VARCHAR(50) NOT NULL,           -- CREATE, UPDATE, DELETE, READ (for sensitive)
    key VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    
    -- Security context
    user_id UUID,
    username VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    
    -- Timestamps
    timestamp TIMESTAMP DEFAULT NOW(),
    correlation_id UUID,                   -- For distributed tracing
    
    -- Additional metadata
    metadata JSONB                         -- Extra context (e.g., API endpoint, reason)
);

-- Indexes for audit queries
CREATE INDEX idx_property_audit_tenant ON property_audit_log(tenant_id, timestamp DESC);
CREATE INDEX idx_property_audit_artifact ON property_audit_log(artifact_id, timestamp DESC);
CREATE INDEX idx_property_audit_user ON property_audit_log(user_id, timestamp DESC);
CREATE INDEX idx_property_audit_action ON property_audit_log(tenant_id, action, timestamp DESC);
CREATE INDEX idx_property_audit_correlation ON property_audit_log(correlation_id);

-- ============================================
-- property_templates table (optional)
-- ============================================
-- Pre-defined property templates for common use cases
CREATE TABLE IF NOT EXISTS property_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),                 -- e.g., 'security', 'compliance', 'custom'
    
    -- Template definition
    properties JSONB NOT NULL,             -- Array of {key, value_type, required, default, description}
    
    -- Metadata
    is_system BOOLEAN DEFAULT FALSE,       -- System templates (read-only)
    is_active BOOLEAN DEFAULT TRUE,
    created_by UUID REFERENCES users(user_id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_property_templates_tenant ON property_templates(tenant_id, is_active);
CREATE INDEX idx_property_templates_category ON property_templates(tenant_id, category);

-- ============================================
-- RBAC Permissions for Properties
-- ============================================

-- Insert new permissions for artifact properties
INSERT INTO permissions (name, description, resource_type, action, created_at)
VALUES 
    ('artifact.properties.read', 'Read artifact properties', 'artifact', 'read', NOW()),
    ('artifact.properties.write', 'Create or update artifact properties', 'artifact', 'write', NOW()),
    ('artifact.properties.delete', 'Delete artifact properties', 'artifact', 'delete', NOW()),
    ('artifact.properties.sensitive.read', 'Read sensitive artifact properties', 'artifact', 'read', NOW()),
    ('artifact.properties.sensitive.write', 'Write sensitive artifact properties', 'artifact', 'write', NOW()),
    ('artifact.properties.search', 'Search artifacts by properties', 'artifact', 'read', NOW()),
    ('artifact.properties.audit', 'View property audit logs', 'artifact', 'read', NOW())
ON CONFLICT (name) DO NOTHING;

-- ============================================
-- System Properties (Pre-populated)
-- ============================================

-- Common system properties that scanners/system will populate
-- These are read-only to users
COMMENT ON TABLE artifact_properties IS 'Stores custom key-value properties for artifacts. Supports multi-value, sensitive (encrypted), and system properties.';
COMMENT ON COLUMN artifact_properties.is_system IS 'System properties are read-only and managed by SecureStor (e.g., scan results, checksums)';
COMMENT ON COLUMN artifact_properties.is_sensitive IS 'Sensitive properties are encrypted at rest and masked in logs/UI';
COMMENT ON COLUMN artifact_properties.is_multi_value IS 'Multi-value properties can have multiple values for the same key';
COMMENT ON COLUMN artifact_properties.encrypted_value IS 'Encrypted value for sensitive properties (stored as base64)';

-- ============================================
-- Functions for Property Management
-- ============================================

-- Function to refresh the materialized view
CREATE OR REPLACE FUNCTION refresh_property_search_index()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY property_search_index;
END;
$$ LANGUAGE plpgsql;

-- Function to log property changes
CREATE OR REPLACE FUNCTION log_property_change()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO property_audit_log (
            tenant_id, property_id, artifact_id, action, key, new_value, user_id
        ) VALUES (
            NEW.tenant_id, NEW.id, NEW.artifact_id, 'CREATE', NEW.key, 
            CASE WHEN NEW.is_sensitive THEN '***REDACTED***' ELSE NEW.value END,
            NEW.created_by
        );
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO property_audit_log (
            tenant_id, property_id, artifact_id, action, key, old_value, new_value, user_id
        ) VALUES (
            NEW.tenant_id, NEW.id, NEW.artifact_id, 'UPDATE', NEW.key,
            CASE WHEN OLD.is_sensitive THEN '***REDACTED***' ELSE OLD.value END,
            CASE WHEN NEW.is_sensitive THEN '***REDACTED***' ELSE NEW.value END,
            NEW.updated_by
        );
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO property_audit_log (
            tenant_id, property_id, artifact_id, action, key, old_value
        ) VALUES (
            OLD.tenant_id, OLD.id, OLD.artifact_id, 'DELETE', OLD.key,
            CASE WHEN OLD.is_sensitive THEN '***REDACTED***' ELSE OLD.value END
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic audit logging
CREATE TRIGGER trigger_log_property_change
AFTER INSERT OR UPDATE OR DELETE ON artifact_properties
FOR EACH ROW EXECUTE FUNCTION log_property_change();

-- ============================================
-- Helper Views
-- ============================================

-- View: All properties with artifact info (non-sensitive only)
CREATE OR REPLACE VIEW v_artifact_properties_public AS
SELECT 
    ap.id,
    ap.tenant_id,
    ap.repository_id,
    ap.artifact_id,
    ap.key,
    ap.value,
    ap.value_type,
    ap.is_system,
    ap.is_multi_value,
    ap.created_by,
    ap.created_at,
    ap.updated_by,
    ap.updated_at,
    ap.description,
    ap.tags
FROM artifact_properties ap
WHERE ap.is_sensitive = FALSE;

-- View: Property statistics per tenant
CREATE OR REPLACE VIEW v_property_statistics AS
SELECT 
    tenant_id,
    COUNT(*) as total_properties,
    COUNT(DISTINCT artifact_id) as artifacts_with_properties,
    COUNT(DISTINCT key) as unique_keys,
    COUNT(*) FILTER (WHERE is_sensitive = TRUE) as sensitive_properties,
    COUNT(*) FILTER (WHERE is_system = TRUE) as system_properties,
    COUNT(*) FILTER (WHERE is_multi_value = TRUE) as multi_value_properties,
    MAX(created_at) as last_property_added
FROM artifact_properties
GROUP BY tenant_id;

-- ============================================
-- Sample System Property Templates
-- ============================================

-- Insert common system property templates
INSERT INTO property_templates (tenant_id, name, description, category, properties, is_system, is_active)
SELECT 
    t.id as tenant_id,
    'security_scan_results',
    'Security scan results and vulnerability information',
    'security',
    '[
        {"key": "scan.trivy.status", "value_type": "string", "required": false, "description": "Trivy scan status"},
        {"key": "scan.trivy.vulnerabilities", "value_type": "number", "required": false, "description": "Number of vulnerabilities found"},
        {"key": "scan.trivy.critical", "value_type": "number", "required": false, "description": "Critical vulnerabilities"},
        {"key": "scan.trivy.high", "value_type": "number", "required": false, "description": "High severity vulnerabilities"}
    ]'::jsonb,
    TRUE,
    TRUE
FROM tenants t
ON CONFLICT DO NOTHING;

INSERT INTO property_templates (tenant_id, name, description, category, properties, is_system, is_active)
SELECT 
    t.id as tenant_id,
    'compliance_metadata',
    'Compliance and regulatory metadata',
    'compliance',
    '[
        {"key": "compliance.license", "value_type": "string", "required": false, "description": "Software license"},
        {"key": "compliance.sbom.id", "value_type": "string", "required": false, "description": "SBOM identifier"},
        {"key": "compliance.signature.verified", "value_type": "boolean", "required": false, "description": "Signature verification status"},
        {"key": "compliance.approved", "value_type": "boolean", "required": false, "description": "Approval status"}
    ]'::jsonb,
    TRUE,
    TRUE
FROM tenants t
ON CONFLICT DO NOTHING;

-- ============================================
-- Performance Tuning
-- ============================================

-- Analyze tables for query planner
ANALYZE artifact_properties;
ANALYZE property_audit_log;
ANALYZE property_templates;

-- Vacuum tables
VACUUM ANALYZE artifact_properties;

COMMENT ON TABLE property_audit_log IS 'Audit log for all property operations (create, update, delete, read sensitive)';
COMMENT ON TABLE property_templates IS 'Pre-defined property templates for common use cases';
COMMENT ON MATERIALIZED VIEW property_search_index IS 'Denormalized index for fast property searches (excludes sensitive properties)';
