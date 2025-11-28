-- Migration 012: Remote Proxy Schema
-- Creates tables for remote proxy functionality

-- Remote repositories configuration
CREATE TABLE IF NOT EXISTS remote_repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT,
    name VARCHAR(255) NOT NULL,
    protocol VARCHAR(50) NOT NULL, -- maven, npm, docker, pypi, helm
    base_url VARCHAR(500) NOT NULL,
    description TEXT,
    
    -- Authentication
    auth_type VARCHAR(50), -- none, basic, bearer, oauth, api_key
    auth_token TEXT,
    auth_username VARCHAR(255),
    auth_password VARCHAR(255),
    
    -- Configuration
    enabled BOOLEAN DEFAULT true,
    cache_enabled BOOLEAN DEFAULT true,
    cache_ttl_minutes INTEGER DEFAULT 1440, -- 24 hours
    scan_enabled BOOLEAN DEFAULT true,
    max_concurrent_requests INTEGER DEFAULT 10,
    request_timeout_seconds INTEGER DEFAULT 30,
    
    -- Performance
    bandwidth_limit_mbps INTEGER, -- NULL = unlimited
    connection_pool_size INTEGER DEFAULT 5,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_health_check TIMESTAMP,
    
    CONSTRAINT unique_remote_repo_name UNIQUE(name)
);

CREATE INDEX IF NOT EXISTS idx_remote_repo_tenant ON remote_repositories(tenant_id);
CREATE INDEX IF NOT EXISTS idx_remote_repo_protocol ON remote_repositories(protocol);
CREATE INDEX IF NOT EXISTS idx_remote_repo_enabled ON remote_repositories(enabled);

-- Proxied artifacts cache metadata
CREATE TABLE IF NOT EXISTS proxied_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT,
    remote_repository_id UUID NOT NULL,
    artifact_id VARCHAR(255) NOT NULL, -- maven: groupId:artifactId:version, npm: @org/package@version, etc.
    artifact_type VARCHAR(50) NOT NULL, -- jar, npm, docker-image, wheel, helm-chart
    
    -- Cache tiers
    l1_cached BOOLEAN DEFAULT false, -- Redis
    l1_expires_at TIMESTAMP,
    l2_cached BOOLEAN DEFAULT false, -- Local disk
    l2_expires_at TIMESTAMP,
    l3_cached BOOLEAN DEFAULT false, -- Cloud S3
    l3_expires_at TIMESTAMP,
    
    -- Content metadata
    content_hash VARCHAR(128), -- SHA-256
    content_size_bytes BIGINT,
    content_type VARCHAR(100),
    
    -- Origin metadata
    origin_url VARCHAR(500),
    origin_last_modified TIMESTAMP,
    origin_etag VARCHAR(255),
    
    -- Retrieval tracking
    fetch_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP,
    first_cached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Security
    scan_status VARCHAR(50), -- pending, scanning, passed, failed, blocked
    scan_result_id UUID,
    vulnerability_count INTEGER DEFAULT 0,
    blocked BOOLEAN DEFAULT false,
    block_reason TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_proxied_artifact_repo FOREIGN KEY (remote_repository_id) REFERENCES remote_repositories(id) ON DELETE CASCADE,
    CONSTRAINT unique_proxied_artifact UNIQUE(remote_repository_id, artifact_id)
);

CREATE INDEX IF NOT EXISTS idx_proxied_artifact_tenant ON proxied_artifacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_proxied_artifact_repo ON proxied_artifacts(remote_repository_id);
CREATE INDEX IF NOT EXISTS idx_proxied_artifact_blocked ON proxied_artifacts(blocked);
CREATE INDEX IF NOT EXISTS idx_proxied_artifact_scan_status ON proxied_artifacts(scan_status);
CREATE INDEX IF NOT EXISTS idx_proxied_artifact_expires_at ON proxied_artifacts(l1_expires_at, l2_expires_at, l3_expires_at);
);

-- Remote repository health monitoring
CREATE TABLE IF NOT EXISTS remote_repository_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    remote_repository_id UUID NOT NULL,
    
    -- Health status
    status VARCHAR(50) NOT NULL, -- healthy, degraded, unhealthy, unknown
    last_check_at TIMESTAMP,
    last_successful_check_at TIMESTAMP,
    
    -- Metrics
    response_time_ms INTEGER,
    error_count INTEGER DEFAULT 0,
    error_count_24h INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    success_rate FLOAT DEFAULT 100.0,
    
    -- Details
    last_error_message TEXT,
    last_error_code VARCHAR(10),
    check_details TEXT, -- JSON with detailed diagnostics
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_health_repo FOREIGN KEY (remote_repository_id) REFERENCES remote_repositories(id) ON DELETE CASCADE,
    CONSTRAINT unique_health_per_repo UNIQUE(remote_repository_id)
);

CREATE INDEX IF NOT EXISTS idx_health_status ON remote_repository_health(status);
CREATE INDEX IF NOT EXISTS idx_health_last_check ON remote_repository_health(last_check_at);
);

-- Proxy request metrics and analytics
CREATE TABLE IF NOT EXISTS proxy_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT,
    remote_repository_id UUID NOT NULL,
    
    -- Request information
    artifact_id VARCHAR(255) NOT NULL,
    request_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Cache hit/miss analysis
    cache_tier_hit VARCHAR(50), -- l1, l2, l3, origin, none
    cache_hit BOOLEAN,
    
    -- Performance
    total_time_ms INTEGER,
    network_time_ms INTEGER,
    cache_lookup_time_ms INTEGER,
    
    -- Request details
    request_method VARCHAR(10), -- GET, HEAD, etc.
    response_code INTEGER,
    response_size_bytes BIGINT,
    
    -- Retry information
    retry_count INTEGER DEFAULT 0,
    fallback_used BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_metrics_repo FOREIGN KEY (remote_repository_id) REFERENCES remote_repositories(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_metrics_tenant ON proxy_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_metrics_repo ON proxy_metrics(remote_repository_id);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON proxy_metrics(request_timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_cache_hit ON proxy_metrics(cache_hit);
CREATE INDEX IF NOT EXISTS idx_metrics_cache_tier ON proxy_metrics(cache_tier_hit);
);

-- Protocol adapter configurations
CREATE TABLE IF NOT EXISTS protocol_adapter_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    remote_repository_id UUID NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    
    -- Protocol-specific settings
    config_json TEXT NOT NULL, -- JSON with protocol-specific configuration
    
    -- Features
    list_enabled BOOLEAN DEFAULT false,
    search_enabled BOOLEAN DEFAULT false,
    version_browsing_enabled BOOLEAN DEFAULT true,
    
    -- Defaults
    default_artifact_retention_days INTEGER DEFAULT 30,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_adapter_config_repo FOREIGN KEY (remote_repository_id) REFERENCES remote_repositories(id) ON DELETE CASCADE,
    CONSTRAINT unique_adapter_per_repo UNIQUE(remote_repository_id, protocol)
);

CREATE INDEX IF NOT EXISTS idx_adapter_config_protocol ON protocol_adapter_configs(protocol);
);

-- Security scan results for proxied artifacts
CREATE TABLE IF NOT EXISTS proxy_security_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    proxied_artifact_id UUID NOT NULL,
    remote_repository_id UUID NOT NULL,
    
    -- Scan information
    scan_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    scan_engine VARCHAR(100), -- trivy, grype, snyk, custom
    scan_duration_seconds INTEGER,
    
    -- Results
    vulnerability_count INTEGER DEFAULT 0,
    critical_count INTEGER DEFAULT 0,
    high_count INTEGER DEFAULT 0,
    medium_count INTEGER DEFAULT 0,
    low_count INTEGER DEFAULT 0,
    
    -- Details
    scan_result_json TEXT, -- Full scan results
    remediation_available BOOLEAN DEFAULT false,
    
    -- Actions
    scan_status VARCHAR(50), -- passed, failed, needs_review
    action_taken VARCHAR(100), -- allowed, blocked, quarantined
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_scan_artifact FOREIGN KEY (proxied_artifact_id) REFERENCES proxied_artifacts(id) ON DELETE CASCADE,
    CONSTRAINT fk_scan_repo FOREIGN KEY (remote_repository_id) REFERENCES remote_repositories(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scan_artifact ON proxy_security_scans(proxied_artifact_id);
CREATE INDEX IF NOT EXISTS idx_scan_timestamp ON proxy_security_scans(scan_timestamp);
CREATE INDEX IF NOT EXISTS idx_scan_status ON proxy_security_scans(scan_status);
);

-- Cache eviction events (for audit trail)
CREATE TABLE IF NOT EXISTS cache_eviction_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT,
    artifact_id VARCHAR(255) NOT NULL,
    
    -- Eviction reason
    eviction_reason VARCHAR(100), -- ttl_expired, lru_evicted, manual_delete, space_pressure
    eviction_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Cache tiers affected
    l1_evicted BOOLEAN DEFAULT false,
    l2_evicted BOOLEAN DEFAULT false,
    l3_evicted BOOLEAN DEFAULT false,
    
    -- Impact
    freed_space_bytes BIGINT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_eviction_tenant ON cache_eviction_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_eviction_reason ON cache_eviction_events(eviction_reason);
CREATE INDEX IF NOT EXISTS idx_eviction_timestamp ON cache_eviction_events(eviction_timestamp);
);

-- Proxy service configuration per tenant
CREATE TABLE IF NOT EXISTS proxy_service_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id BIGINT,
    
    -- Global settings
    proxy_enabled BOOLEAN DEFAULT true,
    enable_cache BOOLEAN DEFAULT true,
    enable_security_scanning BOOLEAN DEFAULT true,
    enable_health_monitoring BOOLEAN DEFAULT true,
    
    -- Cache configuration
    l1_max_size_gb FLOAT DEFAULT 16.0,
    l1_ttl_hours INTEGER DEFAULT 24,
    l2_max_size_gb FLOAT DEFAULT 500.0,
    l2_ttl_days INTEGER DEFAULT 7,
    l3_max_size_gb FLOAT DEFAULT 5000.0,
    l3_ttl_days INTEGER DEFAULT 30,
    
    -- L3 Cloud configuration
    l3_provider VARCHAR(50), -- s3, gcs, azure
    l3_provider_config TEXT, -- JSON with provider-specific config
    
    -- Performance
    max_concurrent_requests INTEGER DEFAULT 100,
    request_timeout_seconds INTEGER DEFAULT 30,
    retry_attempts INTEGER DEFAULT 3,
    enable_compression BOOLEAN DEFAULT true,
    
    -- Security
    require_authentication BOOLEAN DEFAULT true,
    rate_limit_requests_per_minute INTEGER DEFAULT 60,
    enable_audit_logging BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_proxy_config_per_tenant UNIQUE(tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_proxy_config_tenant ON proxy_service_config(tenant_id);
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_proxied_artifacts_lru ON proxied_artifacts(last_accessed DESC) WHERE l2_cached = true;
CREATE INDEX IF NOT EXISTS idx_remote_repo_health_check ON remote_repository_health(last_check_at DESC);
CREATE INDEX IF NOT EXISTS idx_proxy_metrics_analysis ON proxy_metrics(tenant_id, cache_hit, request_timestamp DESC);
