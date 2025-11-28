-- Migration: Add tenant_id to cached_artifacts for multi-tenancy
-- Description: Adds tenant_id column to cached_artifacts and related tables

-- Add tenant_id to cached_artifacts
ALTER TABLE cached_artifacts 
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE;

-- Create index for tenant filtering
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_tenant ON cached_artifacts(tenant_id);

-- Update existing records to default tenant (if any)
-- Find the first tenant and assign to existing records
DO $$
DECLARE
    default_tenant_id UUID;
BEGIN
    SELECT tenant_id INTO default_tenant_id FROM tenants ORDER BY created_at ASC LIMIT 1;
    
    IF default_tenant_id IS NOT NULL THEN
        UPDATE cached_artifacts 
        SET tenant_id = default_tenant_id 
        WHERE tenant_id IS NULL;
    END IF;
END $$;

-- Make tenant_id NOT NULL after backfilling
ALTER TABLE cached_artifacts 
ALTER COLUMN tenant_id SET NOT NULL;

-- Update unique constraint to include tenant_id
ALTER TABLE cached_artifacts 
DROP CONSTRAINT IF EXISTS unique_artifact_path;

ALTER TABLE cached_artifacts 
DROP CONSTRAINT IF EXISTS unique_artifact_path_tenant;

ALTER TABLE cached_artifacts 
ADD CONSTRAINT unique_artifact_path_tenant UNIQUE (artifact_path, cache_level, tenant_id);

-- Add tenant_id to scan_queue
ALTER TABLE scan_queue 
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE;

-- Update existing scan_queue records
UPDATE scan_queue sq
SET tenant_id = ca.tenant_id
FROM cached_artifacts ca
WHERE sq.cached_artifact_id = ca.id AND sq.tenant_id IS NULL;

-- Create index for tenant filtering on scan_queue
CREATE INDEX IF NOT EXISTS idx_scan_queue_tenant ON scan_queue(tenant_id);

-- Add tenant_id to scan_history
ALTER TABLE scan_history 
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE;

-- Update existing scan_history records
UPDATE scan_history sh
SET tenant_id = ca.tenant_id
FROM cached_artifacts ca
WHERE sh.cached_artifact_id = ca.id AND sh.tenant_id IS NULL;

-- Create index for tenant filtering on scan_history
CREATE INDEX IF NOT EXISTS idx_scan_history_tenant ON scan_history(tenant_id);

-- Update the track_cache_access function to accept tenant_id
CREATE OR REPLACE FUNCTION track_cache_access(
    p_artifact_path TEXT,
    p_cache_level VARCHAR(10),
    p_hit BOOLEAN,
    p_response_time_ms INTEGER,
    p_cache_source VARCHAR(10),
    p_client_ip INET,
    p_user_agent TEXT,
    p_tenant_id UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_cached_artifact_id UUID;
BEGIN
    -- Find or create cached artifact record
    INSERT INTO cached_artifacts (artifact_path, artifact_type, cache_level, size_bytes, last_accessed, tenant_id)
    VALUES (
        p_artifact_path, 
        split_part(p_artifact_path, '/', 1),  -- Extract type from path
        p_cache_level,
        0,  -- Size will be updated separately
        NOW(),
        p_tenant_id
    )
    ON CONFLICT (artifact_path, cache_level, tenant_id) 
    DO UPDATE SET 
        last_accessed = NOW(),
        hit_count = cached_artifacts.hit_count + CASE WHEN p_hit THEN 1 ELSE 0 END,
        miss_count = cached_artifacts.miss_count + CASE WHEN p_hit THEN 0 ELSE 1 END
    RETURNING id INTO v_cached_artifact_id;
    
    -- Log the access
    INSERT INTO cache_access_logs (
        cached_artifact_id,
        accessed_at,
        hit,
        response_time_ms,
        cache_source,
        client_ip,
        user_agent,
        tenant_id
    ) VALUES (
        v_cached_artifact_id,
        NOW(),
        p_hit,
        p_response_time_ms,
        p_cache_source,
        p_client_ip,
        p_user_agent,
        p_tenant_id
    );
    
    RETURN v_cached_artifact_id;
END;
$$ LANGUAGE plpgsql;

-- Update queue_cache_scan function to accept tenant_id
CREATE OR REPLACE FUNCTION queue_cache_scan(
    p_cached_artifact_id UUID,
    p_priority INTEGER DEFAULT 50,
    p_tenant_id UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_scan_id UUID;
    v_artifact_path TEXT;
    v_artifact_type VARCHAR(50);
    v_tenant_id UUID;
BEGIN
    -- Get artifact info including tenant_id
    SELECT artifact_path, artifact_type, tenant_id
    INTO v_artifact_path, v_artifact_type, v_tenant_id
    FROM cached_artifacts 
    WHERE id = p_cached_artifact_id;
    
    -- Create scan job
    INSERT INTO scan_queue (
        cached_artifact_id,
        artifact_path,
        artifact_type,
        priority,
        status,
        tenant_id
    ) VALUES (
        p_cached_artifact_id,
        v_artifact_path,
        v_artifact_type,
        p_priority,
        'queued',
        COALESCE(p_tenant_id, v_tenant_id)
    )
    RETURNING id INTO v_scan_id;
    
    -- Update artifact scan status
    UPDATE cached_artifacts
    SET scan_status = 'pending'
    WHERE id = p_cached_artifact_id;
    
    RETURN v_scan_id;
END;
$$ LANGUAGE plpgsql;

-- Note: Skipping cache_statistics updates as it has a different schema

COMMENT ON COLUMN cached_artifacts.tenant_id IS 'Tenant isolation for cached artifacts';
COMMENT ON COLUMN scan_queue.tenant_id IS 'Tenant isolation for scan queue';
COMMENT ON COLUMN scan_history.tenant_id IS 'Tenant isolation for scan history';
