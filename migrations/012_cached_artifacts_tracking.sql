-- Migration: Cache Artifact Tracking and Management
-- Description: Track all cached artifacts across L1/L2/L3 with scan status and metrics

-- Track all cached artifacts across L1/L2/L3
CREATE TABLE IF NOT EXISTS cached_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE,
    artifact_path TEXT NOT NULL,
    artifact_type VARCHAR(50) NOT NULL, -- maven, npm, pypi, helm, docker
    cache_level VARCHAR(10) NOT NULL,   -- L1, L2, L3
    size_bytes BIGINT NOT NULL,
    hit_count INTEGER DEFAULT 0,
    miss_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    expiry_at TIMESTAMP,
    scan_status VARCHAR(50) DEFAULT 'pending', -- pending, scanning, completed, failed
    scan_results_id UUID REFERENCES scan_results(id) ON DELETE SET NULL,
    checksum VARCHAR(128),
    metadata JSONB DEFAULT '{}',        -- package version, dependencies, etc.
    CONSTRAINT unique_artifact_path UNIQUE (artifact_path, cache_level)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_type ON cached_artifacts(artifact_type);
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_cache_level ON cached_artifacts(cache_level);
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_scan_status ON cached_artifacts(scan_status);
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_last_accessed ON cached_artifacts(last_accessed DESC);
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_created_at ON cached_artifacts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cached_artifacts_checksum ON cached_artifacts(checksum);

-- Track cache performance metrics
CREATE TABLE IF NOT EXISTS cache_access_logs (
    id BIGSERIAL PRIMARY KEY,
    cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE CASCADE,
    accessed_at TIMESTAMP DEFAULT NOW(),
    hit BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    cache_source VARCHAR(10),           -- L1, L2, L3, remote
    client_ip INET,
    user_agent TEXT,
    tenant_id UUID,
    user_id UUID
);

-- Index for performance
CREATE INDEX IF NOT EXISTS idx_cache_access_logs_accessed_at ON cache_access_logs(accessed_at DESC);
CREATE INDEX IF NOT EXISTS idx_cache_access_logs_artifact ON cache_access_logs(cached_artifact_id);
CREATE INDEX IF NOT EXISTS idx_cache_access_logs_tenant ON cache_access_logs(tenant_id);

-- Partition by month for better performance
CREATE TABLE IF NOT EXISTS cache_access_logs_partitioned (
    LIKE cache_access_logs INCLUDING ALL
) PARTITION BY RANGE (accessed_at);

-- Create partitions for current and next 6 months
-- Note: In production, automate partition creation

-- Aggregate cache statistics (materialized view for dashboard)
CREATE MATERIALIZED VIEW IF NOT EXISTS cache_statistics AS
SELECT 
    artifact_type,
    cache_level,
    COUNT(*) as total_items,
    SUM(size_bytes) as total_size_bytes,
    SUM(hit_count) as total_hits,
    SUM(miss_count) as total_misses,
    AVG(hit_count::float / NULLIF(hit_count + miss_count, 0)::float) * 100 as avg_hit_rate,
    MAX(last_accessed) as last_activity,
    COUNT(CASE WHEN scan_status = 'completed' THEN 1 END) as scanned_items,
    COUNT(CASE WHEN scan_status = 'pending' THEN 1 END) as pending_scans,
    COUNT(CASE WHEN scan_status = 'scanning' THEN 1 END) as scanning_items,
    COUNT(CASE WHEN scan_status = 'failed' THEN 1 END) as failed_scans,
    NOW() as last_refreshed
FROM cached_artifacts
GROUP BY artifact_type, cache_level;

-- Index for materialized view
CREATE INDEX IF NOT EXISTS idx_cache_statistics_type_level ON cache_statistics (artifact_type, cache_level);

-- Scan queue for background processing
CREATE TABLE IF NOT EXISTS scan_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE CASCADE,
    artifact_path TEXT NOT NULL,
    artifact_type VARCHAR(50) NOT NULL,
    priority INTEGER DEFAULT 50,        -- 0 (lowest) to 100 (highest)
    status VARCHAR(50) DEFAULT 'queued', -- queued, processing, completed, failed
    scan_config JSONB DEFAULT '{}',     -- Scanner configuration
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

-- Indexes for scan queue
CREATE INDEX IF NOT EXISTS idx_scan_queue_status ON scan_queue(status);
CREATE INDEX IF NOT EXISTS idx_scan_queue_priority ON scan_queue(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_scan_queue_artifact ON scan_queue(cached_artifact_id);

-- Scan history for audit and analytics
CREATE TABLE IF NOT EXISTS scan_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE CASCADE,
    artifact_path TEXT NOT NULL,
    checksum VARCHAR(128),
    scan_started_at TIMESTAMP NOT NULL,
    scan_completed_at TIMESTAMP,
    scan_duration_ms INTEGER,
    scanners_used TEXT[],               -- Array of scanner names
    vulnerabilities_found INTEGER DEFAULT 0,
    critical_count INTEGER DEFAULT 0,
    high_count INTEGER DEFAULT 0,
    medium_count INTEGER DEFAULT 0,
    low_count INTEGER DEFAULT 0,
    scan_result JSONB,                  -- Full scan result
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for scan history
CREATE INDEX IF NOT EXISTS idx_scan_history_artifact ON scan_history(cached_artifact_id);
CREATE INDEX IF NOT EXISTS idx_scan_history_checksum ON scan_history(checksum);
CREATE INDEX IF NOT EXISTS idx_scan_history_completed ON scan_history(scan_completed_at DESC);

-- Function to update cache statistics
CREATE OR REPLACE FUNCTION refresh_cache_statistics()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY cache_statistics;
END;
$$ LANGUAGE plpgsql;

-- Function to track cache access
CREATE OR REPLACE FUNCTION track_cache_access(
    p_artifact_path TEXT,
    p_cache_level VARCHAR(10),
    p_hit BOOLEAN,
    p_response_time_ms INTEGER,
    p_cache_source VARCHAR(10),
    p_client_ip INET,
    p_user_agent TEXT
)
RETURNS UUID AS $$
DECLARE
    v_cached_artifact_id UUID;
BEGIN
    -- Find or create cached artifact record
    INSERT INTO cached_artifacts (artifact_path, artifact_type, cache_level, size_bytes, last_accessed)
    VALUES (
        p_artifact_path, 
        split_part(p_artifact_path, '/', 1),  -- Extract type from path
        p_cache_level,
        0,  -- Size will be updated separately
        NOW()
    )
    ON CONFLICT (artifact_path, cache_level) 
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
        user_agent
    ) VALUES (
        v_cached_artifact_id,
        NOW(),
        p_hit,
        p_response_time_ms,
        p_cache_source,
        p_client_ip,
        p_user_agent
    );
    
    RETURN v_cached_artifact_id;
END;
$$ LANGUAGE plpgsql;

-- Function to queue scan for cached artifact
CREATE OR REPLACE FUNCTION queue_cache_scan(
    p_cached_artifact_id UUID,
    p_priority INTEGER DEFAULT 50
)
RETURNS UUID AS $$
DECLARE
    v_scan_id UUID;
    v_artifact_path TEXT;
    v_artifact_type VARCHAR(50);
BEGIN
    -- Get artifact info
    SELECT artifact_path, artifact_type 
    INTO v_artifact_path, v_artifact_type
    FROM cached_artifacts 
    WHERE id = p_cached_artifact_id;
    
    -- Create scan job
    INSERT INTO scan_queue (
        cached_artifact_id,
        artifact_path,
        artifact_type,
        priority,
        status
    ) VALUES (
        p_cached_artifact_id,
        v_artifact_path,
        v_artifact_type,
        p_priority,
        'queued'
    )
    RETURNING id INTO v_scan_id;
    
    -- Update artifact scan status
    UPDATE cached_artifacts
    SET scan_status = 'pending'
    WHERE id = p_cached_artifact_id;
    
    RETURN v_scan_id;
END;
$$ LANGUAGE plpgsql;

-- Scheduled job to refresh cache statistics (every 5 minutes)
-- Note: Requires pg_cron extension
-- SELECT cron.schedule('refresh_cache_stats', '*/5 * * * *', 'SELECT refresh_cache_statistics();');

-- Create view for cache dashboard
CREATE OR REPLACE VIEW cache_dashboard AS
SELECT 
    cs.*,
    (SELECT COUNT(*) FROM scan_queue WHERE status = 'queued') as queued_scans,
    (SELECT COUNT(*) FROM scan_queue WHERE status = 'processing') as processing_scans,
    (SELECT AVG(scan_duration_ms) FROM scan_history WHERE scan_completed_at > NOW() - INTERVAL '24 hours') as avg_scan_time_24h
FROM cache_statistics cs;

-- Grant permissions (adjust based on your user setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON cached_artifacts TO securestor_app;
-- GRANT SELECT, INSERT ON cache_access_logs TO securestor_app;
-- GRANT SELECT ON cache_statistics TO securestor_app;
-- GRANT SELECT, INSERT, UPDATE ON scan_queue TO securestor_app;
-- GRANT SELECT, INSERT ON scan_history TO securestor_app;

-- Add comments for documentation
COMMENT ON TABLE cached_artifacts IS 'Tracks all cached artifacts across L1/L2/L3 cache layers';
COMMENT ON TABLE cache_access_logs IS 'Logs every cache access for analytics and audit';
COMMENT ON TABLE scan_queue IS 'Queue for background scanning of cached artifacts';
COMMENT ON TABLE scan_history IS 'Historical record of all security scans performed';
COMMENT ON MATERIALIZED VIEW cache_statistics IS 'Aggregated cache statistics for dashboard (refreshed every 5 minutes)';
