-- Migration 017: S3/MinIO Replication Node Support
-- Adds support for S3-compatible remote storage as replication targets

-- Add columns for S3/remote storage support
ALTER TABLE replication_nodes 
    ADD COLUMN IF NOT EXISTS node_type VARCHAR(20) DEFAULT 'local' CHECK (node_type IN ('local', 's3', 'minio', 'gcs', 'azure')),
    ADD COLUMN IF NOT EXISTS s3_endpoint VARCHAR(500),
    ADD COLUMN IF NOT EXISTS s3_region VARCHAR(100),
    ADD COLUMN IF NOT EXISTS s3_bucket VARCHAR(255),
    ADD COLUMN IF NOT EXISTS s3_access_key TEXT,
    ADD COLUMN IF NOT EXISTS s3_secret_key TEXT,
    ADD COLUMN IF NOT EXISTS s3_use_ssl BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS s3_path_prefix VARCHAR(500);

-- Create index for node type filtering
CREATE INDEX IF NOT EXISTS idx_replication_nodes_type ON replication_nodes(tenant_id, node_type, is_active);

-- Update existing nodes to be 'local' type
UPDATE replication_nodes SET node_type = 'local' WHERE node_type IS NULL;

-- Add comment to document the schema
COMMENT ON COLUMN replication_nodes.node_type IS 'Type of storage node: local (filesystem), s3 (AWS S3), minio (MinIO), gcs (Google Cloud Storage), azure (Azure Blob)';
COMMENT ON COLUMN replication_nodes.s3_endpoint IS 'S3-compatible endpoint URL (e.g., http://127.0.0.1:9000 for MinIO)';
COMMENT ON COLUMN replication_nodes.s3_bucket IS 'S3 bucket name for remote storage';
COMMENT ON COLUMN replication_nodes.s3_path_prefix IS 'Optional path prefix within the bucket';
