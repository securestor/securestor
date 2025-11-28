-- Migration: Add adapter-specific tables for unified database and storage operations
-- This extends the existing compliance schema with adapter-specific tables

-- Artifact Compliance Status table (unified compliance tracking)
CREATE TABLE IF NOT EXISTS artifact_compliance_status (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT UNIQUE REFERENCES artifacts(id) ON DELETE CASCADE,
    overall_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    gdpr_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retention_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    encryption_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    data_locality VARCHAR(50) NOT NULL DEFAULT 'unknown',
    legal_hold BOOLEAN NOT NULL DEFAULT FALSE,
    last_compliance_run TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    score INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Retention Records table (detailed retention tracking)
CREATE TABLE IF NOT EXISTS retention_records (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    policy_id BIGINT REFERENCES compliance_policies(id),
    retention_days INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    grace_period_days INTEGER DEFAULT 30,
    status VARCHAR(50) DEFAULT 'active',
    notification_sent BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Storage Metadata table (tracks where and how artifacts are stored)
CREATE TABLE IF NOT EXISTS storage_metadata (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT UNIQUE REFERENCES artifacts(id) ON DELETE CASCADE,
    storage_backend VARCHAR(50) NOT NULL, -- local, s3, gcp, azure, minio
    storage_location JSONB NOT NULL, -- JSON containing location details
    encryption_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    erasure_coding JSONB, -- JSON containing erasure coding config
    checksum VARCHAR(128),
    size BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_verified TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Storage Health Monitoring table
CREATE TABLE IF NOT EXISTS storage_health_logs (
    id SERIAL PRIMARY KEY,
    backend_type VARCHAR(50) NOT NULL,
    backend_name VARCHAR(255),
    status VARCHAR(50) NOT NULL, -- healthy, degraded, unhealthy, offline
    response_time_ms INTEGER,
    available_space_bytes BIGINT,
    used_space_bytes BIGINT,
    total_space_bytes BIGINT,
    healthy_shards INTEGER DEFAULT 0,
    damaged_shards INTEGER DEFAULT 0,
    issues JSONB, -- JSON array of health issues
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Data Integrity Reports table
CREATE TABLE IF NOT EXISTS integrity_reports (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    storage_key VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL, -- healthy, warning, corrupted, unrecoverable
    checksum_valid BOOLEAN DEFAULT TRUE,
    erasure_code_valid BOOLEAN DEFAULT TRUE,
    corrupted_shards JSONB, -- JSON array of corrupted shard indices
    recoverable_shards INTEGER DEFAULT 0,
    required_shards INTEGER DEFAULT 0,
    repair_recommendation VARCHAR(50) DEFAULT 'none',
    last_verified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    repair_attempted BOOLEAN DEFAULT FALSE,
    repair_success BOOLEAN,
    repair_timestamp TIMESTAMP
);

-- Compliance Log Entries table (detailed compliance audit trail)
CREATE TABLE IF NOT EXISTS compliance_log_entries (
    id SERIAL PRIMARY KEY,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE CASCADE,
    policy_id BIGINT REFERENCES compliance_policies(id),
    status VARCHAR(50) NOT NULL, -- compliant, non_compliant, warning, pending
    details TEXT,
    violations JSONB, -- JSON array of violation descriptions
    remediation JSONB, -- JSON object with remediation steps
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    next_check TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system'
);

-- Storage Operations Log table (for audit and monitoring)
CREATE TABLE IF NOT EXISTS storage_operations_log (
    id SERIAL PRIMARY KEY,
    operation_type VARCHAR(50) NOT NULL, -- store, retrieve, delete, verify, repair
    storage_key VARCHAR(500) NOT NULL,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE SET NULL,
    backend_type VARCHAR(50) NOT NULL,
    operation_size_bytes BIGINT,
    duration_ms INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    user_id VARCHAR(255),
    ip_address INET,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Batch Operations table (tracks batch storage operations)
CREATE TABLE IF NOT EXISTS batch_operations (
    id SERIAL PRIMARY KEY,
    operation_type VARCHAR(50) NOT NULL, -- batch_store, batch_delete
    total_items INTEGER NOT NULL,
    successful_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'running', -- running, completed, failed, partial
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_details JSONB,
    initiated_by VARCHAR(255)
);

-- Batch Operation Items table (details of individual items in batch operations)
CREATE TABLE IF NOT EXISTS batch_operation_items (
    id SERIAL PRIMARY KEY,
    batch_id BIGINT REFERENCES batch_operations(id) ON DELETE CASCADE,
    storage_key VARCHAR(500) NOT NULL,
    artifact_id BIGINT REFERENCES artifacts(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, success, failed
    size_bytes BIGINT,
    error_message TEXT,
    processed_at TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_artifact_compliance_status_artifact_id ON artifact_compliance_status(artifact_id);
CREATE INDEX IF NOT EXISTS idx_artifact_compliance_status_overall_status ON artifact_compliance_status(overall_status);

CREATE INDEX IF NOT EXISTS idx_retention_records_artifact_id ON retention_records(artifact_id);
CREATE INDEX IF NOT EXISTS idx_retention_records_expires_at ON retention_records(expires_at);
CREATE INDEX IF NOT EXISTS idx_retention_records_status ON retention_records(status);

CREATE INDEX IF NOT EXISTS idx_storage_metadata_artifact_id ON storage_metadata(artifact_id);
CREATE INDEX IF NOT EXISTS idx_storage_metadata_backend ON storage_metadata(storage_backend);

CREATE INDEX IF NOT EXISTS idx_storage_health_logs_backend_type ON storage_health_logs(backend_type);
CREATE INDEX IF NOT EXISTS idx_storage_health_logs_created_at ON storage_health_logs(created_at);

CREATE INDEX IF NOT EXISTS idx_integrity_reports_artifact_id ON integrity_reports(artifact_id);
CREATE INDEX IF NOT EXISTS idx_integrity_reports_status ON integrity_reports(status);
CREATE INDEX IF NOT EXISTS idx_integrity_reports_last_verified ON integrity_reports(last_verified);

CREATE INDEX IF NOT EXISTS idx_compliance_log_entries_artifact_id ON compliance_log_entries(artifact_id);
CREATE INDEX IF NOT EXISTS idx_compliance_log_entries_policy_id ON compliance_log_entries(policy_id);
CREATE INDEX IF NOT EXISTS idx_compliance_log_entries_status ON compliance_log_entries(status);
CREATE INDEX IF NOT EXISTS idx_compliance_log_entries_checked_at ON compliance_log_entries(checked_at);

CREATE INDEX IF NOT EXISTS idx_storage_operations_log_operation_type ON storage_operations_log(operation_type);
CREATE INDEX IF NOT EXISTS idx_storage_operations_log_backend_type ON storage_operations_log(backend_type);
CREATE INDEX IF NOT EXISTS idx_storage_operations_log_created_at ON storage_operations_log(created_at);
CREATE INDEX IF NOT EXISTS idx_storage_operations_log_artifact_id ON storage_operations_log(artifact_id);

CREATE INDEX IF NOT EXISTS idx_batch_operations_status ON batch_operations(status);
CREATE INDEX IF NOT EXISTS idx_batch_operations_started_at ON batch_operations(started_at);

CREATE INDEX IF NOT EXISTS idx_batch_operation_items_batch_id ON batch_operation_items(batch_id);
CREATE INDEX IF NOT EXISTS idx_batch_operation_items_status ON batch_operation_items(status);

-- Create triggers for updated_at columns
CREATE TRIGGER update_artifact_compliance_status_updated_at BEFORE UPDATE ON artifact_compliance_status
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_retention_records_updated_at BEFORE UPDATE ON retention_records
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_storage_metadata_updated_at BEFORE UPDATE ON storage_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default storage health entry for local backend
INSERT INTO storage_health_logs (
    backend_type, backend_name, status, response_time_ms,
    available_space_bytes, used_space_bytes, total_space_bytes,
    healthy_shards, damaged_shards, issues
) VALUES (
    'local', 'Local Erasure-Coded Storage', 'healthy', 5,
    1073741824000, 107374182400, 1181116006400, -- 1TB total, 100GB used, 900GB available
    0, 0, '[]'::jsonb
) ON CONFLICT DO NOTHING;

-- Create views for easier querying

-- Comprehensive artifact status view
CREATE OR REPLACE VIEW artifact_status_summary AS
SELECT 
    a.id,
    a.name,
    a.type,
    a.size,
    a.created_at,
    acs.overall_status,
    acs.gdpr_status,
    acs.retention_status,
    acs.encryption_status,
    acs.data_locality,
    acs.legal_hold,
    acs.score as compliance_score,
    sm.storage_backend,
    sm.encryption_status as storage_encryption_status,
    rr.expires_at as retention_expires_at,
    rr.status as retention_record_status,
    CASE 
        WHEN EXISTS (SELECT 1 FROM legal_holds lh WHERE lh.artifact_id = a.id AND lh.status = 'active') 
        THEN true 
        ELSE false 
    END as has_active_legal_hold,
    ir.status as integrity_status,
    ir.last_verified as last_integrity_check
FROM artifacts a
LEFT JOIN artifact_compliance_status acs ON a.id = acs.artifact_id
LEFT JOIN storage_metadata sm ON a.id = sm.artifact_id
LEFT JOIN retention_records rr ON a.id = rr.artifact_id AND rr.status = 'active'
LEFT JOIN integrity_reports ir ON a.id = ir.artifact_id 
WHERE ir.id IS NULL OR ir.id = (
    SELECT MAX(id) FROM integrity_reports ir2 WHERE ir2.artifact_id = a.id
);

-- Storage health summary view
CREATE OR REPLACE VIEW storage_health_summary AS
SELECT 
    backend_type,
    backend_name,
    status,
    response_time_ms,
    available_space_bytes,
    used_space_bytes,
    total_space_bytes,
    ROUND(CAST((used_space_bytes::FLOAT / total_space_bytes::FLOAT) * 100 AS NUMERIC), 2) as utilization_percent,
    healthy_shards,
    damaged_shards,
    CASE 
        WHEN damaged_shards > 0 THEN 'degraded'
        WHEN shl.status = 'healthy' AND ROUND(CAST((used_space_bytes::FLOAT / total_space_bytes::FLOAT) * 100 AS NUMERIC), 2) < 90 THEN 'optimal'
        WHEN shl.status = 'healthy' AND ROUND(CAST((used_space_bytes::FLOAT / total_space_bytes::FLOAT) * 100 AS NUMERIC), 2) >= 90 THEN 'high_utilization'
        ELSE shl.status 
    END as health_category,
    created_at as last_check
FROM storage_health_logs shl
WHERE shl.id = (
    SELECT MAX(id) FROM storage_health_logs shl2 
    WHERE shl2.backend_type = shl.backend_type AND shl2.backend_name = shl.backend_name
);

-- Compliance audit summary view
CREATE OR REPLACE VIEW compliance_audit_summary AS
SELECT 
    policy_id,
    cp.name as policy_name,
    cp.type as policy_type,
    COUNT(*) as total_audits,
    COUNT(CASE WHEN cle.status = 'compliant' THEN 1 END) as compliant_count,
    COUNT(CASE WHEN cle.status = 'non_compliant' THEN 1 END) as non_compliant_count,
    COUNT(CASE WHEN cle.status = 'warning' THEN 1 END) as warning_count,
    COUNT(CASE WHEN cle.status = 'pending' THEN 1 END) as pending_count,
    ROUND(CAST((COUNT(CASE WHEN cle.status = 'compliant' THEN 1 END)::FLOAT / COUNT(*)::FLOAT) * 100 AS NUMERIC), 2) as compliance_rate,
    MAX(checked_at) as last_audit_time
FROM compliance_log_entries cle
JOIN compliance_policies cp ON cle.policy_id = cp.id
GROUP BY policy_id, cp.name, cp.type;

-- Comments for documentation
COMMENT ON TABLE artifact_compliance_status IS 'Unified compliance status tracking for all artifacts';
COMMENT ON TABLE retention_records IS 'Detailed retention policy tracking with expiration dates';
COMMENT ON TABLE storage_metadata IS 'Storage location and configuration metadata for artifacts';
COMMENT ON TABLE storage_health_logs IS 'Storage backend health monitoring and metrics';
COMMENT ON TABLE integrity_reports IS 'Data integrity verification results and repair status';
COMMENT ON TABLE compliance_log_entries IS 'Detailed compliance audit trail for policy enforcement';
COMMENT ON TABLE storage_operations_log IS 'Audit log for all storage operations (store/retrieve/delete)';
COMMENT ON TABLE batch_operations IS 'Tracking for batch storage operations';
COMMENT ON TABLE batch_operation_items IS 'Individual items within batch operations';

COMMENT ON VIEW artifact_status_summary IS 'Comprehensive view of artifact status including compliance, storage, and integrity';
COMMENT ON VIEW storage_health_summary IS 'Latest storage backend health status and metrics';
COMMENT ON VIEW compliance_audit_summary IS 'Compliance policy audit statistics and rates';