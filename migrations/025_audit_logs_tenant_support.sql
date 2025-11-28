-- Migration 025: Add tenant support to audit_logs table
-- STATUS: ✅ INTEGRATED into internal/database/migrations.go
-- This migration adds tenant_id column and converts id to UUID
-- NOTE: This file is kept for reference. The actual schema is defined in migrations.go

-- Step 1: Add new UUID column for id
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS log_id UUID DEFAULT gen_random_uuid();

-- Step 2: Populate log_id with UUIDs for existing rows
UPDATE audit_logs SET log_id = gen_random_uuid() WHERE log_id IS NULL;

-- Step 3: Make log_id NOT NULL
ALTER TABLE audit_logs ALTER COLUMN log_id SET NOT NULL;

-- Step 4: Add tenant_id column
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Step 5: Set default tenant_id for existing rows (use the 'default' tenant)
-- First, get the default tenant ID
DO $$
DECLARE
    default_tenant_id UUID;
BEGIN
    SELECT tenant_id INTO default_tenant_id FROM tenants WHERE slug = 'default' LIMIT 1;
    
    IF default_tenant_id IS NOT NULL THEN
        UPDATE audit_logs SET tenant_id = default_tenant_id WHERE tenant_id IS NULL;
    END IF;
END $$;

-- Step 6: Make tenant_id NOT NULL and add foreign key
ALTER TABLE audit_logs ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE audit_logs ADD CONSTRAINT fk_audit_logs_tenant 
    FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE;

-- Step 7: Drop old serial id column (after ensuring log_id is populated)
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_pkey CASCADE;
ALTER TABLE audit_logs ADD PRIMARY KEY (log_id);
ALTER TABLE audit_logs DROP COLUMN IF EXISTS id;

-- Step 8: Rename log_id to id for consistency
ALTER TABLE audit_logs RENAME COLUMN log_id TO id;

-- Step 9: Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_timestamp ON audit_logs(tenant_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_event_type ON audit_logs(tenant_id, event_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_user_id ON audit_logs(tenant_id, user_id);

-- Step 10: Add comments
COMMENT ON COLUMN audit_logs.id IS 'UUID primary key for audit log entry';
COMMENT ON COLUMN audit_logs.tenant_id IS 'Reference to tenant for multi-tenancy isolation';
