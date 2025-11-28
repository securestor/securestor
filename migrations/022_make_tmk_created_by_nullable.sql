-- Migration to make created_by nullable in tenant_master_keys
-- This allows system-created TMKs (e.g., during automatic initialization)

ALTER TABLE tenant_master_keys 
ALTER COLUMN created_by DROP NOT NULL;

-- Add comment to document why this is nullable
COMMENT ON COLUMN tenant_master_keys.created_by IS 'User who created the key. NULL for system-created keys.';
