-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: Add OPA Policy Management and Audit Logging
-- This migration adds support for Open Policy Agent (OPA) integration and policy audit logging

-- Policy Documents (OPA Rego policies)
CREATE TABLE IF NOT EXISTS opa_policies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    policy_type VARCHAR(50) NOT NULL, -- 'rbac', 'abac', 'custom'
    rego_policy TEXT NOT NULL, -- The actual Rego policy document
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    activated_at TIMESTAMP
);

-- Policy Data (data that policies use for decisions)
CREATE TABLE IF NOT EXISTS opa_policy_data (
    id SERIAL PRIMARY KEY,
    data_key VARCHAR(100) UNIQUE NOT NULL, -- e.g., 'user_attributes', 'resource_metadata'
    data_value JSONB NOT NULL, -- JSON data used by policies
    description TEXT,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Policy Audit Log (records all policy decisions)
CREATE TABLE IF NOT EXISTS policy_audit_log (
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(100), -- Unique request identifier
    user_id BIGINT REFERENCES users(id),
    resource VARCHAR(200), -- e.g., '/api/artifacts/123'
    action VARCHAR(100), -- e.g., 'read', 'write', 'delete'
    decision VARCHAR(20) NOT NULL, -- 'allow', 'deny'
    policy_name VARCHAR(100) REFERENCES opa_policies(name),
    input_data JSONB, -- The input sent to OPA
    policy_output JSONB, -- The output from OPA
    execution_time_ms INTEGER, -- Time taken for policy evaluation
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Policy Violations (significant security events)
CREATE TABLE IF NOT EXISTS policy_violations (
    id SERIAL PRIMARY KEY,
    audit_log_id BIGINT REFERENCES policy_audit_log(id),
    violation_type VARCHAR(100) NOT NULL, -- e.g., 'unauthorized_access', 'privilege_escalation'
    severity VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'critical'
    description TEXT,
    is_investigated BOOLEAN DEFAULT false,
    investigated_by BIGINT REFERENCES users(id),
    investigated_at TIMESTAMP,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Token Introspection Cache (for OAuth2 token introspection)
CREATE TABLE IF NOT EXISTS token_introspection_cache (
    id SERIAL PRIMARY KEY,
    token_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256 hash of the token
    is_active BOOLEAN NOT NULL,
    token_type VARCHAR(50), -- 'access_token', 'refresh_token'
    client_id VARCHAR(100),
    username VARCHAR(100),
    scope TEXT, -- Space-separated scopes
    expires_at TIMESTAMP,
    issued_at TIMESTAMP,
    cached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Compliance Reports
CREATE TABLE IF NOT EXISTS compliance_reports (
    id SERIAL PRIMARY KEY,
    report_type VARCHAR(100) NOT NULL, -- e.g., 'access_audit', 'policy_violations', 'user_activity'
    report_format VARCHAR(20) NOT NULL, -- 'json', 'csv', 'pdf'
    parameters JSONB, -- Report parameters
    file_path TEXT, -- Path to generated report file
    file_size BIGINT, -- File size in bytes
    generated_by BIGINT REFERENCES users(id),
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP, -- When the report file expires
    download_count INTEGER DEFAULT 0,
    last_downloaded TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_opa_policies_name_active ON opa_policies(name, is_active);
CREATE INDEX IF NOT EXISTS idx_opa_policies_type_active ON opa_policies(policy_type, is_active);
CREATE INDEX IF NOT EXISTS idx_opa_policy_data_key ON opa_policy_data(data_key);
CREATE INDEX IF NOT EXISTS idx_policy_audit_log_user_id ON policy_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_policy_audit_log_resource ON policy_audit_log(resource);
CREATE INDEX IF NOT EXISTS idx_policy_audit_log_created_at ON policy_audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_policy_audit_log_decision ON policy_audit_log(decision);
CREATE INDEX IF NOT EXISTS idx_policy_audit_log_request_id ON policy_audit_log(request_id);
CREATE INDEX IF NOT EXISTS idx_policy_violations_severity ON policy_violations(severity);
CREATE INDEX IF NOT EXISTS idx_policy_violations_investigated ON policy_violations(is_investigated);
CREATE INDEX IF NOT EXISTS idx_token_introspection_cache_hash ON token_introspection_cache(token_hash);
CREATE INDEX IF NOT EXISTS idx_token_introspection_cache_expires ON token_introspection_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_type ON compliance_reports(report_type);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_generated_at ON compliance_reports(generated_at);

-- Update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_opa_policies_updated_at 
    BEFORE UPDATE ON opa_policies 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_opa_policy_data_updated_at 
    BEFORE UPDATE ON opa_policy_data 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create view for policy statistics
CREATE OR REPLACE VIEW policy_statistics AS
SELECT 
    COUNT(*) as total_policies,
    COUNT(*) FILTER (WHERE is_active = true) as active_policies,
    COUNT(*) FILTER (WHERE policy_type = 'rbac') as rbac_policies,
    COUNT(*) FILTER (WHERE policy_type = 'abac') as abac_policies,
    COUNT(*) FILTER (WHERE policy_type = 'custom') as custom_policies
FROM opa_policies;

-- Create view for recent policy decisions
CREATE OR REPLACE VIEW recent_policy_decisions AS
SELECT 
    pal.id,
    pal.user_id,
    u.username,
    pal.resource,
    pal.action,
    pal.decision,
    pal.policy_name,
    pal.execution_time_ms,
    pal.created_at
FROM policy_audit_log pal
LEFT JOIN users u ON pal.user_id = u.id
WHERE pal.created_at >= NOW() - INTERVAL '24 hours'
ORDER BY pal.created_at DESC;

-- Create view for policy violations summary
CREATE OR REPLACE VIEW policy_violations_summary AS
SELECT 
    severity,
    COUNT(*) as violation_count,
    COUNT(*) FILTER (WHERE is_investigated = false) as uninvestigated_count,
    COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') as recent_count
FROM policy_violations
GROUP BY severity;

-- Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON opa_policies TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON opa_policy_data TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON policy_audit_log TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON policy_violations TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON token_introspection_cache TO securestor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON compliance_reports TO securestor_app;
GRANT SELECT ON policy_statistics TO securestor_app;
GRANT SELECT ON recent_policy_decisions TO securestor_app;
GRANT SELECT ON policy_violations_summary TO securestor_app;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO securestor_app;

-- Insert default policies
INSERT INTO opa_policies (name, description, policy_type, rego_policy, created_by) VALUES
('default_rbac', 'Default role-based access control policy', 'rbac', 
'package securestor.rbac

import rego.v1

default allow := false

# Allow if user has required role
allow if {
    input.user.roles[_] == input.required_role
}

# Allow admins everything
allow if {
    input.user.roles[_] == "admin"
}

# Deny if explicitly forbidden
allow if {
    not forbidden
}

forbidden if {
    input.action == "delete"
    input.resource.type == "critical"
    not input.user.roles[_] == "admin"
}', 1),

('resource_abac', 'Attribute-based access control for resources', 'abac',
'package securestor.abac

import rego.v1

default allow := false

# Allow access based on resource attributes and user attributes
allow if {
    # User has required clearance level
    user_clearance := input.user.attributes.clearance_level
    resource_classification := input.resource.attributes.classification
    clearance_hierarchy[user_clearance] >= clearance_hierarchy[resource_classification]
    
    # User belongs to required department
    input.user.attributes.department == input.resource.attributes.department
}

# Clearance level hierarchy
clearance_hierarchy := {
    "public": 0,
    "internal": 1,
    "confidential": 2,
    "secret": 3,
    "top_secret": 4
}

# Time-based access control
allow if {
    # Allow access during business hours
    business_hours
    input.action == "read"
}

business_hours if {
    now := time.now_ns()
    day := time.weekday(now)
    hour := time.hour(now)
    
    # Monday to Friday
    day >= 1
    day <= 5
    
    # 9 AM to 5 PM
    hour >= 9
    hour < 17
}', 1),

('audit_policy', 'Comprehensive audit and logging policy', 'custom',
'package securestor.audit

import rego.v1

# Always audit these actions
audit_required if {
    input.action in ["create", "update", "delete"]
}

# Always audit admin actions
audit_required if {
    input.user.roles[_] == "admin"
}

# Always audit access to sensitive resources
audit_required if {
    input.resource.attributes.sensitive == true
}

# Determine log level based on action and resource
log_level := "info" if {
    input.action == "read"
}

log_level := "warn" if {
    input.action in ["create", "update"]
}

log_level := "error" if {
    input.action == "delete"
}

log_level := "critical" if {
    input.resource.attributes.classification == "top_secret"
}', 1)

ON CONFLICT (name) DO NOTHING;

-- Insert sample policy data
INSERT INTO opa_policy_data (data_key, data_value, description) VALUES
('user_attributes', '{}', 'User attribute mappings for ABAC'),
('resource_metadata', '{}', 'Resource metadata for policy decisions'),
('business_rules', '{
    "allowed_file_types": ["pdf", "docx", "xlsx", "pptx"],
    "max_file_size_mb": 100,
    "restricted_domains": ["competitor.com", "blocked.org"],
    "maintenance_windows": [
        {"start": "02:00", "end": "04:00", "timezone": "UTC"}
    ]
}', 'Business rules configuration')
ON CONFLICT (data_key) DO NOTHING;

COMMIT;