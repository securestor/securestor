-- DEPRECATED: This migration file has been superseded by internal/database/migrations.go
-- All tables from this file are now automatically created by the Go migration system
-- This file is kept for reference only - DO NOT EXECUTE MANUALLY
--
-- Migration: Add comprehensive compliance policy tables
-- Add this to your database migration

-- Compliance Policies table
CREATE TABLE IF NOT EXISTS compliance_policies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- data_retention, gdpr, dpdp, legal_hold, audit_logging, data_locality, access_control, encryption, data_protection, incident_response, data_transfer, children_protection
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- active, inactive, draft
    rules TEXT, -- JSON string of policy rules
    region VARCHAR(10) DEFAULT 'GLOBAL', -- EU, US, IN, GLOBAL
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    enforced_at TIMESTAMP,
    description TEXT
);

-- Legal Holds table
CREATE TABLE IF NOT EXISTS legal_holds (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    case_number VARCHAR(255) NOT NULL,
    reason TEXT NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, released, expired
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit Logs table (immutable)
CREATE TABLE IF NOT EXISTS audit_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_msg TEXT,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

-- Data Locality table
CREATE TABLE IF NOT EXISTS data_locality (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    required_region VARCHAR(10) NOT NULL,
    current_region VARCHAR(10) NOT NULL,
    compliant BOOLEAN NOT NULL DEFAULT TRUE,
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Access Control Policies table
CREATE TABLE IF NOT EXISTS access_control_policies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    permissions JSONB, -- Array of permissions
    roles JSONB, -- Array of roles
    conditions TEXT, -- JSON conditions
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Encryption Policies table
CREATE TABLE IF NOT EXISTS encryption_policies (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    encryption_at_rest BOOLEAN NOT NULL DEFAULT TRUE,
    encryption_in_transit BOOLEAN NOT NULL DEFAULT TRUE,
    key_management VARCHAR(100) DEFAULT 'local', -- local, aws_kms, azure_kv, gcp_kms
    algorithm VARCHAR(100) DEFAULT 'AES-256',
    key_rotation_days INTEGER DEFAULT 90,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Deletion Queue table (for GDPR and retention)
CREATE TABLE IF NOT EXISTS deletion_queue (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    scheduled_by VARCHAR(255) NOT NULL,
    reason VARCHAR(255) NOT NULL,
    scheduled_at TIMESTAMP NOT NULL,
    executed_at TIMESTAMP,
    status VARCHAR(50) DEFAULT 'pending' -- pending, executing, completed, failed
);

-- Compliance Reports table
CREATE TABLE IF NOT EXISTS compliance_reports (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    policy_id BIGINT REFERENCES compliance_policies(id),
    status VARCHAR(50) NOT NULL, -- compliant, non_compliant, warning
    details TEXT,
    recommendations JSONB,
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    next_check TIMESTAMP
);

-- India DPDP-specific tables
-- Consent Management for DPDP compliance
CREATE TABLE IF NOT EXISTS dpdp_consent_records (
    id SERIAL PRIMARY KEY,
    data_principal_id VARCHAR(255) NOT NULL, -- Individual whose data is being processed
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    consent_type VARCHAR(50) NOT NULL, -- explicit, deemed, guardian
    purpose TEXT NOT NULL,
    processing_activities JSONB, -- List of processing activities consented to
    consent_given_at TIMESTAMP NOT NULL,
    consent_withdrawn_at TIMESTAMP,
    consent_status VARCHAR(50) DEFAULT 'active', -- active, withdrawn, expired
    withdrawal_reason TEXT,
    guardian_consent BOOLEAN DEFAULT FALSE,
    guardian_details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Data Principal Rights Requests (DPDP Chapter V)
CREATE TABLE IF NOT EXISTS dpdp_rights_requests (
    id SERIAL PRIMARY KEY,
    data_principal_id VARCHAR(255) NOT NULL,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    request_type VARCHAR(50) NOT NULL, -- erasure, correction, grievance, portability, information
    request_details TEXT NOT NULL,
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_due_date TIMESTAMP NOT NULL, -- Must respond within prescribed time
    status VARCHAR(50) DEFAULT 'pending', -- pending, in_progress, completed, rejected
    response_text TEXT,
    responded_at TIMESTAMP,
    escalated_to_board BOOLEAN DEFAULT FALSE,
    escalation_date TIMESTAMP,
    resolution_details TEXT
);

-- Cross-Border Data Transfer Log (DPDP Section 16)
CREATE TABLE IF NOT EXISTS dpdp_transfer_log (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    transfer_type VARCHAR(50) NOT NULL, -- cross_border, domestic
    source_country VARCHAR(10) NOT NULL,
    destination_country VARCHAR(10) NOT NULL,
    transfer_mechanism VARCHAR(100), -- adequacy_decision, contractual_safeguards, government_approval
    purpose TEXT NOT NULL,
    data_categories JSONB, -- Types of personal data being transferred
    safeguards_applied JSONB, -- Security and privacy safeguards
    transfer_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    approval_reference VARCHAR(255), -- Government approval reference if required
    impact_assessment_done BOOLEAN DEFAULT FALSE,
    notification_sent_to_board BOOLEAN DEFAULT FALSE
);

-- Data Breach Incident Register (DPDP Section 25)
CREATE TABLE IF NOT EXISTS dpdp_breach_register (
    id SERIAL PRIMARY KEY,
    incident_reference VARCHAR(255) UNIQUE NOT NULL,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE SET NULL,
    breach_type VARCHAR(100) NOT NULL, -- unauthorized_access, data_loss, system_compromise, etc.
    severity_level VARCHAR(50) NOT NULL, -- low, medium, high, critical
    affected_individuals_count INTEGER,
    data_categories_affected JSONB,
    breach_discovered_at TIMESTAMP NOT NULL,
    breach_reported_to_board_at TIMESTAMP,
    individuals_notified_at TIMESTAMP,
    cause_of_breach TEXT NOT NULL,
    immediate_actions_taken TEXT,
    remedial_measures JSONB,
    likely_consequences TEXT,
    status VARCHAR(50) DEFAULT 'investigating', -- investigating, contained, resolved, ongoing
    investigation_findings TEXT,
    lessons_learned TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Data Protection Impact Assessment (DPIA) Records
CREATE TABLE IF NOT EXISTS dpdp_impact_assessments (
    id SERIAL PRIMARY KEY,
    assessment_name VARCHAR(255) NOT NULL,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    processing_purpose TEXT NOT NULL,
    data_types JSONB, -- Categories of personal data
    processing_activities JSONB,
    necessity_justification TEXT, -- Why processing is necessary
    proportionality_assessment TEXT, -- Whether processing is proportionate
    risks_identified JSONB, -- Privacy and security risks
    mitigation_measures JSONB, -- Risk mitigation strategies
    residual_risks JSONB, -- Risks remaining after mitigation
    assessment_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP,
    next_review_date TIMESTAMP,
    assessor_name VARCHAR(255),
    approved_by VARCHAR(255),
    status VARCHAR(50) DEFAULT 'draft' -- draft, approved, under_review, rejected
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_compliance_policies_type ON compliance_policies(type);
CREATE INDEX IF NOT EXISTS idx_compliance_policies_status ON compliance_policies(status);
CREATE INDEX IF NOT EXISTS idx_legal_holds_artifact_id ON legal_holds(artifact_id);
CREATE INDEX IF NOT EXISTS idx_legal_holds_status ON legal_holds(status);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_data_locality_artifact_id ON data_locality(artifact_id);
CREATE INDEX IF NOT EXISTS idx_deletion_queue_status ON deletion_queue(status);
CREATE INDEX IF NOT EXISTS idx_deletion_queue_scheduled_at ON deletion_queue(scheduled_at);

-- DPDP-specific indexes
CREATE INDEX IF NOT EXISTS idx_dpdp_consent_records_data_principal ON dpdp_consent_records(data_principal_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_consent_records_artifact_id ON dpdp_consent_records(artifact_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_consent_records_status ON dpdp_consent_records(consent_status);
CREATE INDEX IF NOT EXISTS idx_dpdp_consent_records_given_at ON dpdp_consent_records(consent_given_at);
CREATE INDEX IF NOT EXISTS idx_dpdp_rights_requests_data_principal ON dpdp_rights_requests(data_principal_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_rights_requests_artifact_id ON dpdp_rights_requests(artifact_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_rights_requests_type ON dpdp_rights_requests(request_type);
CREATE INDEX IF NOT EXISTS idx_dpdp_rights_requests_status ON dpdp_rights_requests(status);
CREATE INDEX IF NOT EXISTS idx_dpdp_rights_requests_due_date ON dpdp_rights_requests(response_due_date);
CREATE INDEX IF NOT EXISTS idx_dpdp_transfer_log_artifact_id ON dpdp_transfer_log(artifact_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_transfer_log_countries ON dpdp_transfer_log(source_country, destination_country);
CREATE INDEX IF NOT EXISTS idx_dpdp_transfer_log_date ON dpdp_transfer_log(transfer_date);
CREATE INDEX IF NOT EXISTS idx_dpdp_breach_register_reference ON dpdp_breach_register(incident_reference);
CREATE INDEX IF NOT EXISTS idx_dpdp_breach_register_artifact_id ON dpdp_breach_register(artifact_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_breach_register_discovered_at ON dpdp_breach_register(breach_discovered_at);
CREATE INDEX IF NOT EXISTS idx_dpdp_breach_register_severity ON dpdp_breach_register(severity_level);
CREATE INDEX IF NOT EXISTS idx_dpdp_breach_register_status ON dpdp_breach_register(status);
CREATE INDEX IF NOT EXISTS idx_dpdp_impact_assessments_artifact_id ON dpdp_impact_assessments(artifact_id);
CREATE INDEX IF NOT EXISTS idx_dpdp_impact_assessments_status ON dpdp_impact_assessments(status);
CREATE INDEX IF NOT EXISTS idx_dpdp_impact_assessments_next_review ON dpdp_impact_assessments(next_review_date);

-- Triggers to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_compliance_policies_updated_at BEFORE UPDATE ON compliance_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_legal_holds_updated_at BEFORE UPDATE ON legal_holds
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_access_control_policies_updated_at BEFORE UPDATE ON access_control_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_encryption_policies_updated_at BEFORE UPDATE ON encryption_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- DPDP table triggers
CREATE TRIGGER update_dpdp_consent_records_updated_at BEFORE UPDATE ON dpdp_consent_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_dpdp_breach_register_updated_at BEFORE UPDATE ON dpdp_breach_register
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default policies
INSERT INTO compliance_policies (name, type, status, rules, region, created_by, description) VALUES
('GDPR Data Protection', 'gdpr', 'active', 
 '{"right_to_erasure": true, "data_portability": true, "consent_required": true, "processing_purpose": ["security_analysis", "compliance_audit"], "data_categories": ["personal_data"], "retention_period": 2555}', 
 'EU', 'system', 'GDPR compliance policy for EU data protection'),
 
('7-Year Data Retention', 'data_retention', 'active',
 '{"artifact_types": ["docker", "maven", "npm"], "retention_days": 2555, "delete_after_days": 2585, "grace_period_days": 30, "notify_before_days": 7}',
 'GLOBAL', 'system', 'Standard 7-year data retention policy'),

('US Data Locality', 'data_locality', 'active',
 '{"allowed_regions": ["US"], "enforce_region": true}',
 'US', 'system', 'US data locality requirement'),

('Standard Encryption Policy', 'encryption', 'active',
 '{"encryption_at_rest": true, "encryption_in_transit": true, "key_management": "local", "algorithm": "AES-256", "key_rotation_days": 90}',
 'GLOBAL', 'system', 'Standard encryption requirements'),

-- India DPDP Act 2023 Compliance Policies
('India DPDP Data Protection', 'dpdp', 'active',
 '{"consent_required": true, "consent_type": "explicit", "purpose_limitation": true, "data_minimization": true, "right_to_erasure": true, "right_to_correction": true, "right_to_grievance": true, "processing_purpose": ["legitimate_business", "security_analysis", "compliance_audit"], "data_categories": ["personal_data", "sensitive_personal_data"], "retention_period": 1095, "breach_notification_hours": 72, "data_fiduciary_obligations": true, "cross_border_transfer_restrictions": true}',
 'IN', 'system', 'India DPDP Act 2023 compliance policy for personal data protection'),

('India Data Locality', 'data_locality', 'active',
 '{"allowed_regions": ["IN"], "enforce_region": true, "cross_border_transfer": "restricted", "government_data_localization": true, "sensitive_data_localization": true, "mirror_copy_required": false}',
 'IN', 'system', 'India data locality requirements under DPDP Act'),

('India Sensitive Data Protection', 'data_protection', 'active',
 '{"sensitive_data_categories": ["financial", "health", "biometric", "sexual_orientation", "transgender_status", "caste", "religious_beliefs"], "enhanced_security": true, "explicit_consent": true, "purpose_bound": true, "processing_restrictions": ["profiling", "tracking", "targeted_advertising"], "children_data_protection": true, "guardian_consent_required": true}',
 'IN', 'system', 'Enhanced protection for sensitive personal data under DPDP Act'),

('India Breach Notification', 'incident_response', 'active',
 '{"notification_timeline_hours": 72, "notify_data_protection_board": true, "notify_affected_individuals": true, "breach_severity_threshold": "likely_harm", "assessment_criteria": ["volume", "nature", "cause", "identity_of_individuals"], "remedial_measures_required": true, "documentation_retention_days": 1095}',
 'IN', 'system', 'India DPDP Act breach notification requirements'),

('India Cross-Border Transfer', 'data_transfer', 'active',
 '{"restricted_countries": [], "adequacy_decision_required": false, "contractual_safeguards": true, "government_approval": false, "notification_to_board": true, "transfer_impact_assessment": true, "data_subject_rights_preservation": true, "transfer_purpose_limitation": true}',
 'IN', 'system', 'India DPDP Act cross-border data transfer restrictions'),

('India Children Data Protection', 'children_protection', 'active',
 '{"age_threshold": 18, "guardian_consent_required": true, "verifiable_parental_consent": true, "no_behavioral_monitoring": true, "no_targeted_advertising": true, "enhanced_data_security": true, "limited_data_collection": true, "shorter_retention_period": 365, "special_deletion_rights": true}',
 'IN', 'system', 'Special protection for children data under DPDP Act');