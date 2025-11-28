-- Migration: 020_monitoring_metrics.sql
-- Description: Create tables for system monitoring metrics and alerts
-- Date: 2025-11-08

-- Cache Statistics Table
CREATE TABLE IF NOT EXISTS cache_statistics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    l1_hits BIGINT DEFAULT 0,
    l1_misses BIGINT DEFAULT 0,
    l2_hits BIGINT DEFAULT 0,
    l2_misses BIGINT DEFAULT 0,
    l3_hits BIGINT DEFAULT 0,
    l3_misses BIGINT DEFAULT 0,
    l1_size_mb DECIMAL(10,2) DEFAULT 0,
    l2_size_gb DECIMAL(10,2) DEFAULT 0,
    l3_size_gb DECIMAL(10,2) DEFAULT 0,
    l1_latency_ms DECIMAL(10,2) DEFAULT 0,
    l2_latency_ms DECIMAL(10,2) DEFAULT 0,
    l3_latency_ms DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for time-based queries
CREATE INDEX IF NOT EXISTS idx_cache_statistics_timestamp ON cache_statistics(timestamp DESC);

-- Performance Metrics Table
CREATE TABLE IF NOT EXISTS performance_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    request_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    response_time_ms DECIMAL(10,2) DEFAULT 0,
    bandwidth_mb DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for time-based queries
CREATE INDEX IF NOT EXISTS idx_performance_metrics_timestamp ON performance_metrics(timestamp DESC);

-- Remote Repository Health Table
CREATE TABLE IF NOT EXISTS remote_repository_health (
    id SERIAL PRIMARY KEY,
    repository_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    last_check_at TIMESTAMP DEFAULT NOW(),
    response_time_ms INTEGER DEFAULT 0,
    error_count_24h INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for repository lookups
CREATE INDEX IF NOT EXISTS idx_remote_repository_health_repo_id ON remote_repository_health(repository_id);

-- System Alerts Table
CREATE TABLE IF NOT EXISTS system_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    message TEXT NOT NULL,
    repository_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP,
    metadata JSONB
);

-- Indexes for alerts
CREATE INDEX IF NOT EXISTS idx_system_alerts_status ON system_alerts(status);
CREATE INDEX IF NOT EXISTS idx_system_alerts_severity ON system_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_system_alerts_created_at ON system_alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_alerts_repository_id ON system_alerts(repository_id);

-- Insert sample data for testing
INSERT INTO cache_statistics (timestamp, l1_hits, l1_misses, l2_hits, l2_misses, l3_hits, l3_misses, 
                              l1_size_mb, l2_size_gb, l3_size_gb, l1_latency_ms, l2_latency_ms, l3_latency_ms)
VALUES 
    (NOW() - INTERVAL '1 hour', 9500, 500, 7500, 2500, 3600, 1400, 512.5, 8.3, 45.2, 2.5, 8.3, 45.7),
    (NOW() - INTERVAL '30 minutes', 10200, 300, 8100, 1900, 3900, 1100, 524.8, 8.5, 46.1, 2.3, 7.9, 44.2),
    (NOW() - INTERVAL '15 minutes', 11000, 200, 8800, 1200, 4200, 800, 535.2, 8.7, 47.3, 2.1, 7.5, 43.5),
    (NOW(), 12000, 150, 9200, 800, 4500, 500, 548.6, 9.0, 48.5, 1.9, 7.2, 42.8);

INSERT INTO performance_metrics (timestamp, request_count, error_count, response_time_ms, bandwidth_mb)
VALUES 
    (NOW() - INTERVAL '1 hour', 15000, 150, 120.5, 2500.0),
    (NOW() - INTERVAL '30 minutes', 16200, 120, 110.3, 2650.0),
    (NOW() - INTERVAL '15 minutes', 17500, 100, 105.8, 2800.0),
    (NOW(), 18000, 80, 98.2, 2950.0);

INSERT INTO system_alerts (alert_type, severity, message, status, created_at)
VALUES 
    ('high_error_rate', 'high', 'Error rate exceeded 5% threshold in the last hour', 'active', NOW() - INTERVAL '2 hours'),
    ('cache_miss_rate', 'medium', 'L1 cache miss rate is higher than normal (15%)', 'active', NOW() - INTERVAL '1 hour'),
    ('slow_response', 'medium', 'Average response time increased to 120ms', 'acknowledged', NOW() - INTERVAL '30 minutes');

COMMENT ON TABLE cache_statistics IS 'Stores cache performance metrics for L1/L2/L3 caches';
COMMENT ON TABLE performance_metrics IS 'Stores system performance metrics including response times and throughput';
COMMENT ON TABLE remote_repository_health IS 'Tracks health status of remote repositories';
COMMENT ON TABLE system_alerts IS 'Stores system alerts and notifications';
