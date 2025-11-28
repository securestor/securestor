-- DEPRECATED: This migration file should be integrated into internal/database/migrations.go
-- Manual execution may cause schema conflicts - use the Go migration system instead
-- This file is kept for reference only
--
-- Migration: Add password_hash column to users table
-- This enables local authentication alongside OIDC authentication

-- Add password_hash column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);

-- Add username column to be nullable (for OIDC-only users)
-- But first check if we need to make sub nullable for local auth users
ALTER TABLE users ALTER COLUMN sub DROP NOT NULL;

-- Create an index for username lookups (for login)
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Add a comment to document the authentication modes
COMMENT ON COLUMN users.sub IS 'OIDC subject identifier - nullable for local auth users';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt password hash - nullable for OIDC-only users';