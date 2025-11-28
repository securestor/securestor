package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// RunMigrations is the new UUID-based multi-tenant migration system
func RunMigrations(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("🚀 Starting UUID-based database migrations...")

	// Acquire advisory lock to prevent concurrent migrations
	// Lock ID: 123456789 (arbitrary but consistent)
	log.Println("🔒 Acquiring migration lock...")
	lockResult, err := db.ExecContext(ctx, "SELECT pg_advisory_lock(123456789)")
	if err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	log.Println("✓ Migration lock acquired")

	// Ensure lock is released on exit
	defer func() {
		if _, err := db.Exec("SELECT pg_advisory_unlock(123456789)"); err != nil {
			log.Printf("⚠️  Failed to release migration lock: %v", err)
		} else {
			log.Println("✓ Migration lock released")
		}
	}()

	_ = lockResult // Suppress unused variable warning

	// Enable UUID extension
	if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\""); err != nil {
		log.Printf("⚠️  UUID extension already exists or error: %v", err)
	}

	// Drop all existing tables to ensure clean schema
	// This is safe for development and initial deployment
	// dropMigrations := []string{
	// 	"DROP TRIGGER IF EXISTS update_user_activity_summary_updated_at ON user_activity_summary",
	// 	"DROP TRIGGER IF EXISTS update_storage_metadata_updated_at ON storage_metadata",
	// 	"DROP TRIGGER IF EXISTS update_retention_records_updated_at ON retention_records",
	// 	"DROP TRIGGER IF EXISTS update_legal_holds_updated_at ON legal_holds",
	// 	"DROP TRIGGER IF EXISTS update_compliance_policies_updated_at ON compliance_policies",
	// 	"DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys",
	// 	"DROP TRIGGER IF EXISTS update_oauth2_clients_updated_at ON oauth2_clients",
	// 	"DROP TRIGGER IF EXISTS update_user_totp_secrets_updated_at ON user_totp_secrets",
	// 	"DROP TRIGGER IF EXISTS update_user_mfa_settings_updated_at ON user_mfa_settings",
	// 	"DROP TRIGGER IF EXISTS update_roles_updated_at ON roles",
	// 	"DROP TRIGGER IF EXISTS update_scan_results_updated_at ON scan_results",
	// 	"DROP TRIGGER IF EXISTS update_security_scans_updated_at ON security_scans",
	// 	"DROP TRIGGER IF EXISTS update_compliance_audits_updated_at ON compliance_audits",
	// 	"DROP TRIGGER IF EXISTS update_artifact_versions_updated_at ON artifact_versions",
	// 	"DROP TRIGGER IF EXISTS update_artifacts_updated_at ON artifacts",
	// 	"DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories",
	// 	"DROP TRIGGER IF EXISTS update_users_updated_at ON users",
	// 	"DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants",
	// 	"DROP TABLE IF EXISTS user_activity_summary CASCADE",
	// 	"DROP TABLE IF EXISTS storage_health_logs CASCADE",
	// 	"DROP TABLE IF EXISTS integrity_reports CASCADE",
	// 	"DROP TABLE IF EXISTS storage_metadata CASCADE",
	// 	"DROP TABLE IF EXISTS retention_records CASCADE",
	// 	"DROP TABLE IF EXISTS legal_holds CASCADE",
	// 	"DROP TABLE IF EXISTS compliance_policies CASCADE",
	// 	"DROP TABLE IF EXISTS api_key_usage_logs CASCADE",
	// 	"DROP TABLE IF EXISTS api_keys CASCADE",
	// 	"DROP TABLE IF EXISTS mfa_recovery_codes CASCADE",
	// 	"DROP TABLE IF EXISTS webauthn_challenges CASCADE",
	// 	"DROP TABLE IF EXISTS webauthn_credentials CASCADE",
	// 	"DROP TABLE IF EXISTS user_totp_secrets CASCADE",
	// 	"DROP TABLE IF EXISTS mfa_methods CASCADE",
	// 	"DROP TABLE IF EXISTS user_mfa_settings CASCADE",
	// 	"DROP TABLE IF EXISTS oauth2_refresh_tokens CASCADE",
	// 	"DROP TABLE IF EXISTS oauth2_access_tokens CASCADE",
	// 	"DROP TABLE IF EXISTS oauth2_clients CASCADE",
	// 	"DROP TABLE IF EXISTS oauth2_scopes CASCADE",
	// 	"DROP TABLE IF EXISTS role_permissions CASCADE",
	// 	"DROP TABLE IF EXISTS user_roles CASCADE",
	// 	"DROP TABLE IF EXISTS permissions CASCADE",
	// 	"DROP TABLE IF EXISTS roles CASCADE",
	// 	"DROP TABLE IF EXISTS scan_results CASCADE",
	// 	"DROP TABLE IF EXISTS security_scans CASCADE",
	// 	"DROP TABLE IF EXISTS compliance_audits CASCADE",
	// 	"DROP TABLE IF EXISTS artifact_downloads CASCADE",
	// 	"DROP TABLE IF EXISTS audit_logs CASCADE",
	// 	"DROP TABLE IF EXISTS user_sessions CASCADE",
	// 	"DROP TABLE IF EXISTS artifact_tags CASCADE",
	// 	"DROP TABLE IF EXISTS artifact_versions CASCADE",
	// 	"DROP TABLE IF EXISTS artifacts CASCADE",
	// 	"DROP TABLE IF EXISTS repositories CASCADE",
	// 	"DROP TABLE IF EXISTS users CASCADE",
	// 	"DROP TABLE IF EXISTS tenants CASCADE",
	// }

	// // Execute all DROP statements
	// for _, dropMigration := range dropMigrations {
	// 	if _, err := db.ExecContext(ctx, dropMigration); err != nil {
	// 		// Log but don't fail - these tables might not exist
	// 		log.Printf("⚠️  Drop statement note: %v", err)
	// 	}
	// }

	migrations := []string{
		// ============================================
		// CORE FUNCTION AND EXTENSIONS
		// ============================================
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// ============================================
		// MULTI-TENANT CORE TABLES (UUID PKs)
		// ============================================

		// Tenants table - ROOT of all data
		`CREATE TABLE IF NOT EXISTS tenants (
			tenant_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) UNIQUE NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			contact_email VARCHAR(255),
			is_active BOOLEAN DEFAULT true,
			plan VARCHAR(50) DEFAULT 'basic',
			max_users INTEGER DEFAULT 10,
			max_repositories INTEGER DEFAULT 100,
			features TEXT[] DEFAULT '{}',
			settings JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Users table - tenant-scoped
		`CREATE TABLE IF NOT EXISTS users (
			user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			username VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			display_name VARCHAR(255),
			password_hash VARCHAR(255),
			sub VARCHAR(255),
			is_active BOOLEAN DEFAULT true,
			is_email_verified BOOLEAN DEFAULT false,
			mfa_required BOOLEAN DEFAULT false,
			mfa_enforced_at TIMESTAMP,
			last_login_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, username),
			UNIQUE(tenant_id, email)
		)`,

		// User invites table - for managing user invitations
		`CREATE TABLE IF NOT EXISTS user_invites (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			invited_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			invite_token VARCHAR(255) NOT NULL UNIQUE,
			accepted_at TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create indexes for user_invites
		`CREATE INDEX IF NOT EXISTS idx_user_invites_email ON user_invites(email)`,
		`CREATE INDEX IF NOT EXISTS idx_user_invites_token ON user_invites(invite_token)`,
		`CREATE INDEX IF NOT EXISTS idx_user_invites_expires_at ON user_invites(expires_at)`,

		// Repositories table - tenant-scoped
		`CREATE TABLE IF NOT EXISTS repositories (
			repository_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			description TEXT,
			public_access BOOLEAN DEFAULT false,
			enable_indexing BOOLEAN DEFAULT true,
			remote_url TEXT,
			status VARCHAR(20) DEFAULT 'active',
			settings JSONB DEFAULT '{}',
			enable_encryption BOOLEAN DEFAULT false,
			enable_replication BOOLEAN DEFAULT false,
			replication_targets JSONB DEFAULT '[]',
			sync_frequency VARCHAR(50),
			cloud_provider VARCHAR(50),
			cloud_region VARCHAR(100),
			cloud_config JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name)
		)`,

		// Artifacts table - tenant-scoped, repository-scoped
		`CREATE TABLE IF NOT EXISTS artifacts (
			artifact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			repository_id UUID NOT NULL REFERENCES repositories(repository_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			version VARCHAR(100) NOT NULL,
			type VARCHAR(50) NOT NULL,
			size BIGINT NOT NULL,
			checksum VARCHAR(255) NOT NULL,
			uploaded_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			downloads INTEGER DEFAULT 0,
			license VARCHAR(100),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, repository_id, name, version)
		)`,

		// Artifact Versions table
		`CREATE TABLE IF NOT EXISTS artifact_versions (
			version_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			version VARCHAR(100) NOT NULL,
			changes TEXT,
			released_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, artifact_id, version)
		)`,

		// Artifact Tags table
		`CREATE TABLE IF NOT EXISTS artifact_tags (
			tag_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			tag VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, artifact_id, tag)
		)`,

		// Artifact Indexing table
		`CREATE TABLE IF NOT EXISTS artifact_indexing (
			indexing_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			index_status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (index_status IN ('pending', 'indexing', 'completed', 'failed')),
			search_content TEXT,
			keywords TEXT[],
			indexed_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================
		// COMPLIANCE AND SECURITY TABLES
		// ============================================

		// Compliance Audits table
		`CREATE TABLE IF NOT EXISTS compliance_audits (
			audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			status VARCHAR(50) NOT NULL,
			score INTEGER CHECK (score >= 0 AND score <= 100),
			auditor VARCHAR(255) NOT NULL DEFAULT 'system',
			license_compliance VARCHAR(50),
			security_scan VARCHAR(50),
			code_quality VARCHAR(50),
			data_privacy VARCHAR(50),
			audited_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Security Scans table
		`CREATE TABLE IF NOT EXISTS security_scans (
			scan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			status VARCHAR(50) NOT NULL,
			scan_type VARCHAR(50) NOT NULL,
			priority VARCHAR(50) NOT NULL,
			vulnerability_scan BOOLEAN DEFAULT true,
			malware_scan BOOLEAN DEFAULT true,
			license_scan BOOLEAN DEFAULT true,
			dependency_scan BOOLEAN DEFAULT true,
			initiated_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			started_at TIMESTAMP NOT NULL,
			completed_at TIMESTAMP,
			duration INTEGER,
			error_message TEXT,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Scan Results table
		`CREATE TABLE IF NOT EXISTS scan_results (
			result_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			scan_id UUID NOT NULL REFERENCES security_scans(scan_id) ON DELETE CASCADE,
			overall_score INTEGER CHECK (overall_score >= 0 AND overall_score <= 100),
			risk_level VARCHAR(50) NOT NULL,
			summary TEXT,
			recommendations JSONB DEFAULT '{}',
			vulnerability_results JSONB DEFAULT '{}',
			malware_results JSONB DEFAULT '{}',
			license_results JSONB DEFAULT '{}',
			dependency_results JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Vulnerabilities table
		`CREATE TABLE IF NOT EXISTS vulnerabilities (
			vulnerability_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			scan_id UUID REFERENCES security_scans(scan_id) ON DELETE CASCADE,
			cve_id VARCHAR(50),
			severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
			title VARCHAR(255) NOT NULL,
			description TEXT,
			affected_package VARCHAR(255),
			affected_version VARCHAR(100),
			fixed_version VARCHAR(100),
			cvss_score DECIMAL(3,1) CHECK (cvss_score >= 0.0 AND cvss_score <= 10.0),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT vulnerabilities_cve_format CHECK (cve_id IS NULL OR cve_id ~* '^CVE-[0-9]{4}-[0-9]+$')
		)`,

		// ============================================
		// RBAC TABLES (User Roles and Permissions)
		// ============================================

		// Roles table
		`CREATE TABLE IF NOT EXISTS roles (
			role_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(50) NOT NULL,
			display_name VARCHAR(100),
			description TEXT,
			is_system_role BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name)
		)`,

		// Permissions table
		`CREATE TABLE IF NOT EXISTS permissions (
			permission_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(100) NOT NULL,
			resource VARCHAR(50) NOT NULL,
			action VARCHAR(50) NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, resource, action)
		)`,

		// User Roles table (junction)
		`CREATE TABLE IF NOT EXISTS user_roles (
			user_role_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			role_id UUID NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
			assigned_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			UNIQUE(tenant_id, user_id, role_id)
		)`,

		// Role Permissions table (junction)
		`CREATE TABLE IF NOT EXISTS role_permissions (
			role_perm_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			role_id UUID NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
			permission_id UUID NOT NULL REFERENCES permissions(permission_id) ON DELETE CASCADE,
			granted_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, role_id, permission_id)
		)`,

		// ============================================
		// OAUTH2 TABLES
		// ============================================

		// OAuth2 Scopes table
		`CREATE TABLE IF NOT EXISTS oauth2_scopes (
			scope_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			resource VARCHAR(255) NOT NULL,
			actions TEXT[] NOT NULL DEFAULT '{}',
			is_default BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name)
		)`,

		// OAuth2 Clients table
		`CREATE TABLE IF NOT EXISTS oauth2_clients (
			client_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			client_secret TEXT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			scope_restrictions TEXT[] DEFAULT '{}',
			redirect_uris TEXT[] DEFAULT '{}',
			is_active BOOLEAN DEFAULT true,
			is_confidential BOOLEAN DEFAULT true,
			token_endpoint_auth_method VARCHAR(50) DEFAULT 'client_secret_basic',
			grant_types TEXT[] DEFAULT '{client_credentials}',
			access_token_lifetime INTEGER DEFAULT 3600,
			refresh_token_lifetime INTEGER DEFAULT 7776000,
			created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// OAuth2 Access Tokens table
		`CREATE TABLE IF NOT EXISTS oauth2_access_tokens (
			token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES oauth2_clients(client_id) ON DELETE CASCADE,
			token_hash VARCHAR(128) UNIQUE NOT NULL,
			scope TEXT[] DEFAULT '{}',
			expires_at TIMESTAMP NOT NULL,
			is_revoked BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// OAuth2 Refresh Tokens table
		`CREATE TABLE IF NOT EXISTS oauth2_refresh_tokens (
			refresh_token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			client_id UUID NOT NULL REFERENCES oauth2_clients(client_id) ON DELETE CASCADE,
			access_token_id UUID REFERENCES oauth2_access_tokens(token_id) ON DELETE CASCADE,
			token_hash VARCHAR(128) UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			is_revoked BOOLEAN DEFAULT false,
			is_used BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================
		// MFA TABLES
		// ============================================

		// MFA Methods table
		`CREATE TABLE IF NOT EXISTS mfa_methods (
			method_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(50) NOT NULL,
			display_name VARCHAR(100) NOT NULL,
			description TEXT,
			is_enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name)
		)`,

		// User MFA Settings
		`CREATE TABLE IF NOT EXISTS user_mfa_settings (
			mfa_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			is_mfa_enabled BOOLEAN DEFAULT false,
			backup_codes_generated_at TIMESTAMP,
			backup_codes_used_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, user_id)
		)`,

		// User TOTP Secrets
		`CREATE TABLE IF NOT EXISTS user_totp_secrets (
			totp_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			secret_encrypted TEXT NOT NULL,
			is_verified BOOLEAN DEFAULT false,
			backup_codes TEXT[],
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			verified_at TIMESTAMP,
			last_used_at TIMESTAMP,
			UNIQUE(tenant_id, user_id)
		)`,

		// WebAuthn Credentials
		`CREATE TABLE IF NOT EXISTS webauthn_credentials (
			credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			cred_id TEXT NOT NULL,
			public_key TEXT NOT NULL,
			authenticator_data TEXT NOT NULL,
			sign_count BIGINT DEFAULT 0,
			name VARCHAR(100) NOT NULL,
			device_type VARCHAR(50),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP,
			UNIQUE(tenant_id, cred_id)
		)`,

		// MFA Recovery Codes
		`CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
			recovery_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			code_hash TEXT NOT NULL,
			is_used BOOLEAN DEFAULT false,
			used_at TIMESTAMP,
			used_ip INET,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================
		// AUDIT AND LOGGING TABLES
		// ============================================

		// Audit Logs table (Migration 025: UUID-based with tenant support)
		`CREATE TABLE IF NOT EXISTS audit_logs (
			log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			event_type VARCHAR(100) NOT NULL,
			resource_type VARCHAR(100) NOT NULL,
			resource_id TEXT NOT NULL,
			user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
			action VARCHAR(100) NOT NULL,
			old_value TEXT,
			new_value TEXT,
			ip_address INET,
			user_agent TEXT,
			success BOOLEAN DEFAULT true,
			error_msg TEXT,
			metadata JSONB DEFAULT '{}',
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// User Activity Summary
		`CREATE TABLE IF NOT EXISTS user_activity_summary (
			activity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			first_login TIMESTAMP,
			last_login TIMESTAMP,
			total_logins INTEGER DEFAULT 0,
			total_downloads INTEGER DEFAULT 0,
			total_page_views INTEGER DEFAULT 0,
			total_api_calls INTEGER DEFAULT 0,
			last_ip_address INET,
			last_user_agent TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, user_id)
		)`,

		// Artifact Downloads
		`CREATE TABLE IF NOT EXISTS artifact_downloads (
			download_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
			ip_address INET,
			user_agent TEXT,
			download_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			file_size BIGINT,
			download_duration_ms INTEGER,
			success BOOLEAN DEFAULT true,
			error_message TEXT
		)`,

		// ============================================
		// API KEY MANAGEMENT
		// ============================================

		// API Keys
		`CREATE TABLE IF NOT EXISTS api_keys (
			key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			key_hash VARCHAR(128) NOT NULL UNIQUE,
			key_prefix VARCHAR(16) NOT NULL,
			scopes TEXT[] DEFAULT '{}',
			is_active BOOLEAN DEFAULT true,
			last_used_at TIMESTAMP,
			last_used_ip INET,
			usage_count BIGINT DEFAULT 0,
			rate_limit_per_hour INTEGER DEFAULT 1000,
			rate_limit_per_day INTEGER DEFAULT 10000,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// API Key Usage Logs
		`CREATE TABLE IF NOT EXISTS api_key_usage_logs (
			usage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			api_key_id UUID NOT NULL REFERENCES api_keys(key_id) ON DELETE CASCADE,
			endpoint VARCHAR(255) NOT NULL,
			method VARCHAR(10) NOT NULL,
			status_code INTEGER NOT NULL,
			response_time_ms INTEGER,
			request_size_bytes BIGINT,
			response_size_bytes BIGINT,
			ip_address INET,
			user_agent TEXT,
			used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================
		// COMPLIANCE AND GOVERNANCE TABLES
		// ============================================

		// Compliance Policies
		`CREATE TABLE IF NOT EXISTS compliance_policies (
			policy_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(100) NOT NULL,
			status VARCHAR(50) DEFAULT 'draft',
			rules TEXT,
			region VARCHAR(10) DEFAULT 'GLOBAL',
			created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			description TEXT,
			enforced_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tenant_id, name)
		)`,

		// Legal Holds
		`CREATE TABLE IF NOT EXISTS legal_holds (
			hold_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			case_number VARCHAR(255) NOT NULL,
			reason TEXT NOT NULL,
			start_date TIMESTAMP NOT NULL,
			end_date TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active',
			created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Retention Records
		`CREATE TABLE IF NOT EXISTS retention_records (
			retention_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			policy_id UUID REFERENCES compliance_policies(policy_id) ON DELETE SET NULL,
			retention_days INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			grace_period_days INTEGER DEFAULT 30,
			status VARCHAR(50) DEFAULT 'active',
			notification_sent BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================
		// STORAGE TABLES
		// ============================================

		// Storage Metadata
		`CREATE TABLE IF NOT EXISTS storage_metadata (
			storage_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID UNIQUE NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			storage_backend VARCHAR(50) NOT NULL,
			storage_location JSONB NOT NULL,
			encryption_status VARCHAR(50) DEFAULT 'pending',
			erasure_coding JSONB,
			checksum VARCHAR(128),
			size BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_verified TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Storage Health Logs
		`CREATE TABLE IF NOT EXISTS storage_health_logs (
			health_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			backend_type VARCHAR(50) NOT NULL,
			backend_name VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			response_time_ms INTEGER,
			available_space_bytes BIGINT,
			used_space_bytes BIGINT,
			total_space_bytes BIGINT,
			healthy_shards INTEGER DEFAULT 0,
			damaged_shards INTEGER DEFAULT 0,
			issues JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Integrity Reports
		`CREATE TABLE IF NOT EXISTS integrity_reports (
			report_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			storage_key VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL,
			checksum_valid BOOLEAN DEFAULT true,
			erasure_code_valid BOOLEAN DEFAULT true,
			corrupted_shards JSONB,
			recoverable_shards INTEGER DEFAULT 0,
			required_shards INTEGER DEFAULT 0,
			repair_recommendation VARCHAR(50) DEFAULT 'none',
			last_verified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			repair_attempted BOOLEAN DEFAULT false,
			repair_success BOOLEAN,
			repair_timestamp TIMESTAMP
		)`,

		// User Sessions table for session management
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			ip_address VARCHAR(45),
			user_agent TEXT,
			is_active BOOLEAN DEFAULT true,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		// ============================================
		// INDEXES FOR PERFORMANCE
		// ============================================

		// Tenants indexes
		`CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_tenants_is_active ON tenants(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan)`,

		// Users indexes
		`CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_tenant_username ON users(tenant_id, username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active)`,

		// User Sessions indexes
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_is_active ON user_sessions(is_active)`,

		// Repositories indexes
		`CREATE INDEX IF NOT EXISTS idx_repositories_tenant_id ON repositories(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_tenant_name ON repositories(tenant_id, name)`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_type ON repositories(type)`,
		`CREATE INDEX IF NOT EXISTS idx_repositories_status ON repositories(status)`,

		// Artifacts indexes
		`CREATE INDEX IF NOT EXISTS idx_artifacts_tenant_id ON artifacts(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_repository_id ON artifacts(repository_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_tenant_repo_name ON artifacts(tenant_id, repository_id, name, version)`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type)`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_metadata ON artifacts USING GIN (metadata)`,

		// Artifact Indexing indexes
		`CREATE INDEX IF NOT EXISTS idx_artifact_indexing_tenant_id ON artifact_indexing(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_indexing_artifact_id ON artifact_indexing(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_indexing_status ON artifact_indexing(index_status)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_indexing_keywords ON artifact_indexing USING GIN(keywords)`,

		// Compliance Audits indexes
		`CREATE INDEX IF NOT EXISTS idx_compliance_audits_tenant_id ON compliance_audits(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_audits_artifact_id ON compliance_audits(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_audits_status ON compliance_audits(status)`,

		// Security Scans indexes
		`CREATE INDEX IF NOT EXISTS idx_security_scans_tenant_id ON security_scans(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_security_scans_artifact_id ON security_scans(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_security_scans_status ON security_scans(status)`,
		`CREATE INDEX IF NOT EXISTS idx_security_scans_started_at ON security_scans(started_at)`,

		// Scan Results indexes
		`CREATE INDEX IF NOT EXISTS idx_scan_results_tenant_id ON scan_results(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_results_scan_id ON scan_results(scan_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_results_risk_level ON scan_results(risk_level)`,

		// Vulnerabilities indexes
		`CREATE INDEX IF NOT EXISTS idx_vulnerabilities_tenant_id ON vulnerabilities(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vulnerabilities_artifact_id ON vulnerabilities(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vulnerabilities_scan_id ON vulnerabilities(scan_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vulnerabilities_severity ON vulnerabilities(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_vulnerabilities_cve_id ON vulnerabilities(cve_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vulnerabilities_created_at ON vulnerabilities(created_at)`,

		// Roles indexes
		`CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_roles_tenant_name ON roles(tenant_id, name)`,

		// Permissions indexes
		`CREATE INDEX IF NOT EXISTS idx_permissions_tenant_id ON permissions(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON permissions(tenant_id, resource, action)`,

		// User Roles indexes
		`CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_id ON user_roles(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id)`,

		// Role Permissions indexes
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_tenant_id ON role_permissions(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id)`,

		// OAuth2 indexes
		`CREATE INDEX IF NOT EXISTS idx_oauth2_scopes_tenant_id ON oauth2_scopes(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_scopes_name ON oauth2_scopes(tenant_id, name)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_clients_tenant_id ON oauth2_clients(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_clients_active ON oauth2_clients(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_tenant_id ON oauth2_access_tokens(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_client_id ON oauth2_access_tokens(client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_expires ON oauth2_access_tokens(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_revoked ON oauth2_access_tokens(is_revoked)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_refresh_tokens_tenant_id ON oauth2_refresh_tokens(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_refresh_tokens_client_id ON oauth2_refresh_tokens(client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth2_refresh_tokens_expires ON oauth2_refresh_tokens(expires_at)`,

		// MFA indexes
		`CREATE INDEX IF NOT EXISTS idx_mfa_methods_tenant_id ON mfa_methods(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_mfa_settings_tenant_id ON user_mfa_settings(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_mfa_settings_user_id ON user_mfa_settings(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_totp_secrets_tenant_id ON user_totp_secrets(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_totp_secrets_user_id ON user_totp_secrets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_tenant_id ON webauthn_credentials(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mfa_recovery_codes_tenant_id ON mfa_recovery_codes(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mfa_recovery_codes_user_id ON mfa_recovery_codes(user_id)`,

		// Audit Logs indexes (Migration 025: Enhanced for multi-tenant queries)
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id ON audit_logs(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_timestamp ON audit_logs(tenant_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_event_type ON audit_logs(tenant_id, event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_user_id ON audit_logs(tenant_id, user_id)`,

		// User Activity indexes
		`CREATE INDEX IF NOT EXISTS idx_user_activity_tenant_id ON user_activity_summary(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_activity_user_id ON user_activity_summary(user_id)`,

		// Artifact Downloads indexes
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_tenant_id ON artifact_downloads(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_artifact_id ON artifact_downloads(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_user_id ON artifact_downloads(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_timestamp ON artifact_downloads(download_timestamp)`,

		// API Keys indexes
		`CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at)`,

		// API Key Usage indexes
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_tenant_id ON api_key_usage_logs(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_api_key_id ON api_key_usage_logs(api_key_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_usage_logs_used_at ON api_key_usage_logs(used_at)`,

		// Compliance Policies indexes
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_tenant_id ON compliance_policies(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_type ON compliance_policies(type)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_policies_status ON compliance_policies(status)`,

		// Legal Holds indexes
		`CREATE INDEX IF NOT EXISTS idx_legal_holds_tenant_id ON legal_holds(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_legal_holds_artifact_id ON legal_holds(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_legal_holds_status ON legal_holds(status)`,

		// Retention Records indexes
		`CREATE INDEX IF NOT EXISTS idx_retention_records_tenant_id ON retention_records(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_retention_records_artifact_id ON retention_records(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_retention_records_expires_at ON retention_records(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_retention_records_status ON retention_records(status)`,

		// Storage indexes
		`CREATE INDEX IF NOT EXISTS idx_storage_metadata_tenant_id ON storage_metadata(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_storage_metadata_artifact_id ON storage_metadata(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_storage_health_logs_tenant_id ON storage_health_logs(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_storage_health_logs_backend_type ON storage_health_logs(backend_type)`,
		`CREATE INDEX IF NOT EXISTS idx_integrity_reports_tenant_id ON integrity_reports(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_integrity_reports_artifact_id ON integrity_reports(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_integrity_reports_status ON integrity_reports(status)`,

		// ============================================
		// TRIGGERS FOR AUTOMATIC TIMESTAMP UPDATES
		// ============================================

		`DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants`,
		`CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_users_updated_at ON users`,
		`CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories`,
		`CREATE TRIGGER update_repositories_updated_at BEFORE UPDATE ON repositories
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_artifacts_updated_at ON artifacts`,
		`CREATE TRIGGER update_artifacts_updated_at BEFORE UPDATE ON artifacts
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_artifact_versions_updated_at ON artifact_versions`,
		`CREATE TRIGGER update_artifact_versions_updated_at BEFORE UPDATE ON artifact_versions
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_compliance_audits_updated_at ON compliance_audits`,
		`CREATE TRIGGER update_compliance_audits_updated_at BEFORE UPDATE ON compliance_audits
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_security_scans_updated_at ON security_scans`,
		`CREATE TRIGGER update_security_scans_updated_at BEFORE UPDATE ON security_scans
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_scan_results_updated_at ON scan_results`,
		`CREATE TRIGGER update_scan_results_updated_at BEFORE UPDATE ON scan_results
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_roles_updated_at ON roles`,
		`CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_user_mfa_settings_updated_at ON user_mfa_settings`,
		`CREATE TRIGGER update_user_mfa_settings_updated_at BEFORE UPDATE ON user_mfa_settings
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_user_totp_secrets_updated_at ON user_totp_secrets`,
		`CREATE TRIGGER update_user_totp_secrets_updated_at BEFORE UPDATE ON user_totp_secrets
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_oauth2_clients_updated_at ON oauth2_clients`,
		`CREATE TRIGGER update_oauth2_clients_updated_at BEFORE UPDATE ON oauth2_clients
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys`,
		`CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_compliance_policies_updated_at ON compliance_policies`,
		`CREATE TRIGGER update_compliance_policies_updated_at BEFORE UPDATE ON compliance_policies
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_legal_holds_updated_at ON legal_holds`,
		`CREATE TRIGGER update_legal_holds_updated_at BEFORE UPDATE ON legal_holds
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_retention_records_updated_at ON retention_records`,
		`CREATE TRIGGER update_retention_records_updated_at BEFORE UPDATE ON retention_records
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_storage_metadata_updated_at ON storage_metadata`,
		`CREATE TRIGGER update_storage_metadata_updated_at BEFORE UPDATE ON storage_metadata
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_user_activity_summary_updated_at ON user_activity_summary`,
		`CREATE TRIGGER update_user_activity_summary_updated_at BEFORE UPDATE ON user_activity_summary
			FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()`,

		// ============================================
		// REPLICATION SETTINGS (HA - High Availability)
		// ============================================
		// Tenant-level replication configuration defaults
		`CREATE TABLE IF NOT EXISTS tenant_replication_config (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL UNIQUE,
			enable_replication_default BOOLEAN DEFAULT true,
			default_quorum_size INT DEFAULT 2,
			sync_frequency_default VARCHAR(50) DEFAULT 'realtime',
			node_health_check_interval INT DEFAULT 30,
			failover_timeout INT DEFAULT 20,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT tenant_replication_config_fk_tenant 
				FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			CONSTRAINT valid_quorum_size CHECK (default_quorum_size >= 1 AND default_quorum_size <= 5),
			CONSTRAINT valid_health_check_interval CHECK (node_health_check_interval >= 10 AND node_health_check_interval <= 300),
			CONSTRAINT valid_failover_timeout CHECK (failover_timeout >= 5 AND failover_timeout <= 300)
		)`,

		// Global replication nodes registry
		`CREATE TABLE IF NOT EXISTS replication_nodes (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL,
			node_name VARCHAR(255) NOT NULL,
			node_path VARCHAR(1024) NOT NULL,
			priority INT DEFAULT 1,
			is_active BOOLEAN DEFAULT true,
			last_health_check TIMESTAMP,
			is_healthy BOOLEAN DEFAULT true,
			health_status VARCHAR(50) DEFAULT 'unknown',
			storage_available_gb BIGINT,
			storage_total_gb BIGINT,
			error_count INT DEFAULT 0,
			response_time_ms INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT replication_nodes_fk_tenant 
				FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			CONSTRAINT replication_nodes_unique_name 
				UNIQUE(tenant_id, node_name),
			CONSTRAINT valid_priority CHECK (priority >= 1 AND priority <= 100)
		)`,

		// Audit trail for replication configuration changes
		`CREATE TABLE IF NOT EXISTS replication_config_history (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL,
			repository_id UUID,
			entity_type VARCHAR(50) NOT NULL,
			change_type VARCHAR(50) NOT NULL,
			old_value JSONB,
			new_value JSONB,
			changed_by VARCHAR(255),
			ip_address VARCHAR(45),
			changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT replication_config_history_fk_tenant 
				FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			CONSTRAINT replication_config_history_fk_repo 
				FOREIGN KEY (repository_id) REFERENCES repositories(repository_id) ON DELETE SET NULL,
			CONSTRAINT valid_entity_type CHECK (entity_type IN ('GLOBAL_CONFIG', 'REPO_CONFIG', 'NODE_CONFIG')),
			CONSTRAINT valid_change_type CHECK (change_type IN ('CREATE', 'UPDATE', 'DELETE'))
		)`,

		// Update repositories table to add replication settings
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			enable_replication BOOLEAN DEFAULT true`,
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			replication_node_ids UUID[] DEFAULT NULL`,
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			sync_frequency VARCHAR(50) DEFAULT 'realtime'`,
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			override_global_replication BOOLEAN DEFAULT false`,
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			custom_quorum_size INT`,
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			last_replication_sync TIMESTAMP`,
		`ALTER TABLE repositories ADD COLUMN IF NOT EXISTS 
			replication_status VARCHAR(50) DEFAULT 'not_configured'`,

		// Add constraints to repositories if not exists
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'valid_sync_frequency') THEN
				ALTER TABLE repositories ADD CONSTRAINT valid_sync_frequency 
					CHECK (sync_frequency IN ('realtime', 'hourly', 'daily', 'weekly'));
			END IF;
		END $$`,

		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'valid_replication_status') THEN
				ALTER TABLE repositories ADD CONSTRAINT valid_replication_status 
					CHECK (replication_status IN ('not_configured', 'healthy', 'degraded', 'unhealthy', 'syncing'));
			END IF;
		END $$`,

		// Create indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_tenant_replication_config_tenant 
			ON tenant_replication_config(tenant_id)`,

		`CREATE INDEX IF NOT EXISTS idx_replication_nodes_tenant 
			ON replication_nodes(tenant_id)`,

		`CREATE INDEX IF NOT EXISTS idx_replication_nodes_is_active 
			ON replication_nodes(tenant_id, is_active)`,

		`CREATE INDEX IF NOT EXISTS idx_replication_config_history_tenant 
			ON replication_config_history(tenant_id)`,

		`CREATE INDEX IF NOT EXISTS idx_replication_config_history_repo 
			ON replication_config_history(repository_id)`,

		`CREATE INDEX IF NOT EXISTS idx_replication_config_history_changed_at 
			ON replication_config_history(changed_at DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_repositories_enable_replication 
			ON repositories(tenant_id, enable_replication)`,

		// Create function to update updated_at timestamp for replication config
		`CREATE OR REPLACE FUNCTION update_replication_config_updated_at()
			RETURNS TRIGGER AS $$
			BEGIN
				NEW.updated_at = CURRENT_TIMESTAMP;
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql`,

		// Create trigger for tenant_replication_config
		`DROP TRIGGER IF EXISTS trigger_update_tenant_replication_config_updated_at ON tenant_replication_config`,
		`CREATE TRIGGER trigger_update_tenant_replication_config_updated_at
			BEFORE UPDATE ON tenant_replication_config
			FOR EACH ROW
			EXECUTE FUNCTION update_replication_config_updated_at()`,

		// Create trigger for replication_nodes
		`DROP TRIGGER IF EXISTS trigger_update_replication_nodes_updated_at ON replication_nodes`,
		`CREATE TRIGGER trigger_update_replication_nodes_updated_at
			BEFORE UPDATE ON replication_nodes
			FOR EACH ROW
			EXECUTE FUNCTION update_replication_config_updated_at()`,

		// Insert default replication config for existing tenants (if they don't have one)
		`INSERT INTO tenant_replication_config (tenant_id, enable_replication_default, default_quorum_size, sync_frequency_default)
			SELECT tenant_id, true, 2, 'realtime'
			FROM tenants
			WHERE tenant_id NOT IN (SELECT DISTINCT tenant_id FROM tenant_replication_config)
			ON CONFLICT DO NOTHING`,

		// Insert default replication nodes for existing tenants (if they don't have nodes)
		`INSERT INTO replication_nodes (tenant_id, node_name, node_path, priority, is_active, is_healthy)
			SELECT t.tenant_id, 'node1', '/storage/ssd1/securestor', 1, true, true
			FROM tenants t
			WHERE t.tenant_id NOT IN (SELECT DISTINCT tenant_id FROM replication_nodes)
			ON CONFLICT DO NOTHING`,

		`INSERT INTO replication_nodes (tenant_id, node_name, node_path, priority, is_active, is_healthy)
			SELECT t.tenant_id, 'node2', '/storage/ssd2/securestor', 2, true, true
			FROM tenants t
			WHERE t.tenant_id NOT IN (SELECT DISTINCT tenant_id FROM replication_nodes WHERE node_name = 'node2')
			ON CONFLICT DO NOTHING`,

		`INSERT INTO replication_nodes (tenant_id, node_name, node_path, priority, is_active, is_healthy)
			SELECT t.tenant_id, 'node3', '/storage/ssd3/securestor', 3, true, true
			FROM tenants t
			WHERE t.tenant_id NOT IN (SELECT DISTINCT tenant_id FROM replication_nodes WHERE node_name = 'node3')
			ON CONFLICT DO NOTHING`,

		// ============================================
		// UUID-BASED RBAC SYSTEM (Migration 019)
		// ============================================
		// Drop old RBAC tables if they exist and create UUID-based versions

		`DROP TABLE IF EXISTS role_permissions_uuid CASCADE`,
		`DROP TABLE IF EXISTS user_invite_roles_uuid CASCADE`,
		`DROP TABLE IF EXISTS user_invites_uuid CASCADE`,
		`DROP TABLE IF EXISTS user_roles_uuid CASCADE`,
		`DROP TABLE IF EXISTS permissions_uuid CASCADE`,
		`DROP TABLE IF EXISTS roles_uuid CASCADE`,

		// Roles table with UUID support
		`CREATE TABLE roles_uuid (
			role_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id           UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			role_name           VARCHAR(50) NOT NULL,
			display_name        VARCHAR(100),
			description         TEXT,
			is_system_role      BOOLEAN DEFAULT FALSE,
			created_at          TIMESTAMP DEFAULT NOW(),
			updated_at          TIMESTAMP DEFAULT NOW(),
			UNIQUE(tenant_id, role_name)
		)`,

		// Permissions table with UUID support
		`CREATE TABLE permissions_uuid (
			permission_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			permission_name     VARCHAR(100) NOT NULL,
			resource            VARCHAR(50) NOT NULL,
			action              VARCHAR(50) NOT NULL,
			description         TEXT,
			created_at          TIMESTAMP DEFAULT NOW(),
			UNIQUE(permission_name)
		)`,

		// User roles junction table
		`CREATE TABLE user_roles_uuid (
			user_id             UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
			role_id             UUID NOT NULL REFERENCES roles_uuid(role_id) ON DELETE CASCADE,
			tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			assigned_by         UUID REFERENCES users(user_id),
			assigned_at         TIMESTAMP DEFAULT NOW(),
			expires_at          TIMESTAMP,
			PRIMARY KEY(user_id, role_id, tenant_id)
		)`,

		// Role permissions junction table
		`CREATE TABLE role_permissions_uuid (
			role_id             UUID NOT NULL REFERENCES roles_uuid(role_id) ON DELETE CASCADE,
			permission_id       UUID NOT NULL REFERENCES permissions_uuid(permission_id) ON DELETE CASCADE,
			tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			granted_at          TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY(role_id, permission_id, tenant_id)
		)`,

		// User invites table with UUID support
		`CREATE TABLE user_invites_uuid (
			invite_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			email               VARCHAR(255) NOT NULL,
			first_name          VARCHAR(100),
			last_name           VARCHAR(100),
			invited_by          UUID NOT NULL REFERENCES users(user_id),
			invite_token        VARCHAR(255) UNIQUE NOT NULL,
			expires_at          TIMESTAMP NOT NULL,
			accepted_at         TIMESTAMP,
			created_at          TIMESTAMP DEFAULT NOW(),
			UNIQUE(tenant_id, email)
		)`,

		// User invite roles junction table
		`CREATE TABLE user_invite_roles_uuid (
			invite_id           UUID NOT NULL REFERENCES user_invites_uuid(invite_id) ON DELETE CASCADE,
			role_id             UUID NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
			tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			PRIMARY KEY(invite_id, role_id, tenant_id)
		)`,

		// Indexes for performance
		`CREATE INDEX IF NOT EXISTS idx_roles_uuid_tenant ON roles_uuid(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_uuid_user ON user_roles_uuid(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_uuid_role ON user_roles_uuid(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_roles_uuid_tenant ON user_roles_uuid(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_uuid_role ON role_permissions_uuid(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_permissions_uuid_permission ON role_permissions_uuid(permission_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_invites_uuid_email ON user_invites_uuid(email)`,
		`CREATE INDEX IF NOT EXISTS idx_user_invites_uuid_token ON user_invites_uuid(invite_token)`,

		// Insert default permissions
		`INSERT INTO permissions_uuid (permission_name, resource, action, description) VALUES
			('artifacts.read', 'artifacts', 'read', 'Read access to artifacts'),
			('artifacts.write', 'artifacts', 'write', 'Write access to artifacts'),
			('artifacts.delete', 'artifacts', 'delete', 'Delete access to artifacts'),
			('repositories.read', 'repositories', 'read', 'Read access to repositories'),
			('repositories.write', 'repositories', 'write', 'Write access to repositories'),
			('repositories.delete', 'repositories', 'delete', 'Delete access to repositories'),
			('users.read', 'users', 'read', 'Read access to users'),
			('users.write', 'users', 'write', 'Write access to users'),
			('users.delete', 'users', 'delete', 'Delete access to users'),
			('system.admin', 'system', 'admin', 'Full system administration access')
		ON CONFLICT (permission_name) DO NOTHING`,

		// ============================================
		// MIGRATION 013: SECURITY POLICIES & SCANNING
		// ============================================

		// Security scanning policies per tenant
		`CREATE TABLE IF NOT EXISTS proxy_security_policies (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id BIGINT,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			critical_threshold INTEGER DEFAULT 0,
			high_threshold INTEGER DEFAULT 5,
			medium_threshold INTEGER DEFAULT 10,
			auto_block_enabled BOOLEAN DEFAULT true,
			quarantine_enabled BOOLEAN DEFAULT true,
			notify_on_violation BOOLEAN DEFAULT true,
			required_scanners TEXT[] DEFAULT '{"trivy", "grype"}',
			excluded_artifacts TEXT[] DEFAULT '{}',
			compliance_frameworks TEXT[] DEFAULT '{}',
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_policy_name UNIQUE(name)
		)`,

		// Security scan results for proxied artifacts
		`CREATE TABLE IF NOT EXISTS proxy_security_scans (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			proxied_artifact_id UUID NOT NULL,
			remote_repository_id UUID NOT NULL,
			scan_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			scan_engine VARCHAR(100),
			scan_duration_seconds INTEGER,
			vulnerability_count INTEGER DEFAULT 0,
			critical_count INTEGER DEFAULT 0,
			high_count INTEGER DEFAULT 0,
			medium_count INTEGER DEFAULT 0,
			low_count INTEGER DEFAULT 0,
			scan_result_json TEXT,
			remediation_available BOOLEAN DEFAULT false,
			scan_status VARCHAR(50),
			action_taken VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Security scan findings and vulnerabilities
		`CREATE TABLE IF NOT EXISTS proxy_security_scan_findings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scan_id UUID NOT NULL REFERENCES proxy_security_scans(id) ON DELETE CASCADE,
			artifact_id VARCHAR(255) NOT NULL,
			severity VARCHAR(50) NOT NULL CHECK (severity IN ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO')),
			cve_id VARCHAR(100),
			title VARCHAR(500) NOT NULL,
			description TEXT,
			package_name VARCHAR(255),
			installed_version VARCHAR(100),
			fixed_version VARCHAR(100),
			cvss_score FLOAT DEFAULT 0.0,
			epss_score FLOAT DEFAULT 0.0,
			source_url TEXT,
			is_remediatable BOOLEAN DEFAULT false,
			discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			acknowledged_at TIMESTAMP,
			resolved_at TIMESTAMP,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Compliance audit trail
		`CREATE TABLE IF NOT EXISTS proxy_compliance_audits (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id BIGINT,
			artifact_id VARCHAR(255) NOT NULL,
			policy_id UUID NOT NULL REFERENCES proxy_security_policies(id) ON DELETE CASCADE,
			audit_type VARCHAR(100) NOT NULL,
			compliance_status VARCHAR(50) NOT NULL,
			framework VARCHAR(50),
			findings_count INTEGER DEFAULT 0,
			critical_findings INTEGER DEFAULT 0,
			high_findings INTEGER DEFAULT 0,
			medium_findings INTEGER DEFAULT 0,
			passed BOOLEAN,
			audit_details JSONB,
			auditor VARCHAR(255),
			audit_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Vulnerability acknowledgment log
		`CREATE TABLE IF NOT EXISTS proxy_vulnerability_waivers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id BIGINT,
			finding_id UUID NOT NULL REFERENCES proxy_security_scan_findings(id) ON DELETE CASCADE,
			reason TEXT NOT NULL,
			approved_by VARCHAR(255) NOT NULL,
			valid_until TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_waiver_per_finding UNIQUE(finding_id)
		)`,

		// Vulnerability remediation tracking
		`CREATE TABLE IF NOT EXISTS proxy_vulnerability_remediations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id BIGINT,
			finding_id UUID NOT NULL REFERENCES proxy_security_scan_findings(id) ON DELETE CASCADE,
			status VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED')),
			target_version VARCHAR(100),
			remediation_plan TEXT,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			assignee VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_remediation_per_finding UNIQUE(finding_id)
		)`,

		// Create indexes for security policies
		`CREATE INDEX IF NOT EXISTS idx_security_policy_tenant ON proxy_security_policies(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_security_policy_active ON proxy_security_policies(is_active)`,

		// Create indexes for scan findings
		`CREATE INDEX IF NOT EXISTS idx_scan_finding_artifact ON proxy_security_scan_findings(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_finding_severity ON proxy_security_scan_findings(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_finding_cve ON proxy_security_scan_findings(cve_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_finding_resolved ON proxy_security_scan_findings(resolved_at)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_finding_discovered ON proxy_security_scan_findings(discovered_at DESC)`,

		// Create indexes for compliance audits
		`CREATE INDEX IF NOT EXISTS idx_compliance_audit_tenant ON proxy_compliance_audits(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_audit_policy ON proxy_compliance_audits(policy_id)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_audit_framework ON proxy_compliance_audits(framework)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_audit_status ON proxy_compliance_audits(compliance_status)`,
		`CREATE INDEX IF NOT EXISTS idx_compliance_audit_timestamp ON proxy_compliance_audits(audit_timestamp DESC)`,

		// Create indexes for waivers and remediations
		`CREATE INDEX IF NOT EXISTS idx_waiver_tenant ON proxy_vulnerability_waivers(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_waiver_valid_until ON proxy_vulnerability_waivers(valid_until)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_tenant ON proxy_vulnerability_remediations(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_status ON proxy_vulnerability_remediations(status)`,
		`CREATE INDEX IF NOT EXISTS idx_remediation_assigned ON proxy_vulnerability_remediations(assignee)`,

		// ============================================
		// MONITORING & ALERTING TABLES
		// ============================================

		// Alert rules for monitoring
		`CREATE TABLE IF NOT EXISTS proxy_alert_rules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id BIGINT,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			metric VARCHAR(100) NOT NULL,
			operator VARCHAR(10) NOT NULL CHECK (operator IN ('<', '>', '<=', '>=', '==')),
			threshold FLOAT NOT NULL,
			duration_seconds INTEGER NOT NULL DEFAULT 60,
			enabled BOOLEAN DEFAULT true,
			notify_email VARCHAR(255),
			severity VARCHAR(50) NOT NULL CHECK (severity IN ('INFO', 'WARNING', 'CRITICAL')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_alert_rule UNIQUE(tenant_id, name)
		)`,

		// Alert incidents log
		`CREATE TABLE IF NOT EXISTS proxy_alert_incidents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id BIGINT,
			alert_rule_id UUID NOT NULL REFERENCES proxy_alert_rules(id) ON DELETE CASCADE,
			metric_value FLOAT NOT NULL,
			threshold FLOAT NOT NULL,
			status VARCHAR(50) NOT NULL CHECK (status IN ('TRIGGERED', 'RESOLVED', 'ACKNOWLEDGED')),
			triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			resolved_at TIMESTAMP,
			acknowledged_at TIMESTAMP,
			acknowledged_by VARCHAR(255),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create indexes for alert rules and incidents
		`CREATE INDEX IF NOT EXISTS idx_alert_rule_tenant ON proxy_alert_rules(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_rule_enabled ON proxy_alert_rules(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_incident_tenant ON proxy_alert_incidents(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_incident_rule ON proxy_alert_incidents(alert_rule_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_incident_status ON proxy_alert_incidents(status)`,
		`CREATE INDEX IF NOT EXISTS idx_alert_incident_triggered ON proxy_alert_incidents(triggered_at DESC)`,

		// ============================================
		// SYSTEM MONITORING METRICS TABLES
		// ============================================

		// Cache Statistics Table
		`CREATE TABLE IF NOT EXISTS cache_statistics (
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
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cache_statistics_timestamp ON cache_statistics(timestamp DESC)`,

		// Performance Metrics Table
		`CREATE TABLE IF NOT EXISTS performance_metrics (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
			request_count INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			response_time_ms DECIMAL(10,2) DEFAULT 0,
			bandwidth_mb DECIMAL(10,2) DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_performance_metrics_timestamp ON performance_metrics(timestamp DESC)`,

		// System Alerts Table (for monitoring dashboard)
		`CREATE TABLE IF NOT EXISTS system_alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			alert_type VARCHAR(100) NOT NULL,
			severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
			message TEXT NOT NULL,
			repository_id UUID,
			status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
			created_at TIMESTAMP DEFAULT NOW(),
			resolved_at TIMESTAMP,
			metadata JSONB
		)`,

		`CREATE INDEX IF NOT EXISTS idx_system_alerts_status ON system_alerts(status)`,
		`CREATE INDEX IF NOT EXISTS idx_system_alerts_severity ON system_alerts(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_system_alerts_created_at ON system_alerts(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_system_alerts_repository_id ON system_alerts(repository_id)`,

		// ============================================
		// REMOTE REPOSITORY PROXY TABLES
		// ============================================

		// Remote Repository Configuration
		`CREATE TABLE IF NOT EXISTS remote_repositories (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL CHECK (type IN ('maven', 'docker', 'pypi', 'helm', 'npm')),
			remote_url TEXT NOT NULL,
			
			-- Authentication
			auth_type VARCHAR(50) DEFAULT 'none' CHECK (auth_type IN ('none', 'basic', 'bearer', 'apikey')),
			auth_username VARCHAR(255),
			auth_password_encrypted TEXT,
			auth_token_encrypted TEXT,
			
			-- Cache Configuration
			cache_enabled BOOLEAN DEFAULT true,
			cache_ttl_seconds INTEGER DEFAULT 86400,
			cache_max_size_bytes BIGINT DEFAULT 10737418240,
			cache_eviction_policy VARCHAR(20) DEFAULT 'LRU' CHECK (cache_eviction_policy IN ('LRU', 'LFU', 'FIFO')),
			
			-- Health Status
			health_status VARCHAR(20) DEFAULT 'unknown' CHECK (health_status IN ('healthy', 'degraded', 'offline', 'unknown')),
			last_health_check TIMESTAMP,
			last_response_time_ms INTEGER,
			consecutive_failures INTEGER DEFAULT 0,
			
			-- Metadata
			description TEXT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			
			UNIQUE(tenant_id, name)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_remote_repos_tenant ON remote_repositories(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_remote_repos_type ON remote_repositories(type)`,
		`CREATE INDEX IF NOT EXISTS idx_remote_repos_active ON remote_repositories(is_active)`,

		// Cache Statistics for Remote Repositories
		`CREATE TABLE IF NOT EXISTS remote_cache_stats (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			repository_id UUID NOT NULL REFERENCES remote_repositories(id) ON DELETE CASCADE,
			
			-- Time bucket for aggregation
			time_bucket TIMESTAMP NOT NULL,
			
			-- Cache hit/miss metrics
			l1_hits INTEGER DEFAULT 0,
			l1_misses INTEGER DEFAULT 0,
			l2_hits INTEGER DEFAULT 0,
			l2_misses INTEGER DEFAULT 0,
			l3_hits INTEGER DEFAULT 0,
			l3_misses INTEGER DEFAULT 0,
			remote_fetches INTEGER DEFAULT 0,
			
			-- Performance metrics
			avg_response_time_ms INTEGER,
			total_bandwidth_bytes BIGINT DEFAULT 0,
			bandwidth_saved_bytes BIGINT DEFAULT 0,
			
			-- Cache size metrics
			l1_cache_size_bytes BIGINT DEFAULT 0,
			l2_cache_size_bytes BIGINT DEFAULT 0,
			l3_cache_size_bytes BIGINT DEFAULT 0,
			
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			
			UNIQUE(repository_id, time_bucket)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_remote_cache_stats_repo ON remote_cache_stats(repository_id, time_bucket DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_remote_cache_stats_time ON remote_cache_stats(time_bucket DESC)`,

		// Artifact Downloads Log for Remote Repositories
		`CREATE TABLE IF NOT EXISTS remote_artifact_downloads (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			repository_id UUID NOT NULL REFERENCES remote_repositories(id) ON DELETE CASCADE,
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			
			-- Artifact details
			artifact_path TEXT NOT NULL,
			artifact_size_bytes BIGINT,
			artifact_type VARCHAR(50),
			
			-- Cache and performance details
			cache_source VARCHAR(20) CHECK (cache_source IN ('redis', 'disk', 'cloud', 'remote')),
			cache_hit BOOLEAN DEFAULT false,
			response_time_ms INTEGER,
			
			-- Client information
			user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
			client_ip INET,
			user_agent TEXT,
			
			-- Security scan information
			security_scan_queued BOOLEAN DEFAULT false,
			security_scan_completed BOOLEAN DEFAULT false,
			vulnerabilities_found INTEGER DEFAULT 0,
			
			downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_repo ON remote_artifact_downloads(repository_id, downloaded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_tenant ON remote_artifact_downloads(tenant_id, downloaded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_user ON remote_artifact_downloads(user_id, downloaded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_downloads_cache ON remote_artifact_downloads(cache_hit, downloaded_at DESC)`,

		// ============================================
		// CACHE MANAGEMENT & SECURITY SCANNING
		// ============================================

		// Cached Artifacts - Track all cached items across L1/L2/L3
		`CREATE TABLE IF NOT EXISTS cached_artifacts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			
			-- Artifact identification
			artifact_path TEXT NOT NULL,
			artifact_type VARCHAR(50) NOT NULL,
			artifact_name VARCHAR(255),
			artifact_version VARCHAR(100),
			
			-- Cache information
			cache_level VARCHAR(10) NOT NULL CHECK (cache_level IN ('L1', 'L2', 'L3')),
			cache_key VARCHAR(500) NOT NULL,
			size_bytes BIGINT NOT NULL,
			
			-- Performance metrics
			hit_count INTEGER DEFAULT 0,
			miss_count INTEGER DEFAULT 0,
			last_accessed TIMESTAMP,
			
			-- Security and integrity
			checksum VARCHAR(128),
			checksum_algorithm VARCHAR(20) DEFAULT 'sha256',
			encryption_enabled BOOLEAN DEFAULT false,
			encryption_key_id VARCHAR(100),
			
			-- Scan status
			scan_status VARCHAR(50) DEFAULT 'pending' CHECK (scan_status IN ('pending', 'queued', 'scanning', 'completed', 'failed', 'quarantined')),
			scan_results_id UUID,
			last_scan_at TIMESTAMP,
			next_scan_at TIMESTAMP,
			vulnerabilities_count INTEGER DEFAULT 0,
			critical_vulnerabilities INTEGER DEFAULT 0,
			high_vulnerabilities INTEGER DEFAULT 0,
			medium_vulnerabilities INTEGER DEFAULT 0,
			low_vulnerabilities INTEGER DEFAULT 0,
			
			-- Quarantine status
			is_quarantined BOOLEAN DEFAULT false,
			quarantine_reason TEXT,
			quarantined_at TIMESTAMP,
			quarantined_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			
			-- Metadata
			metadata JSONB DEFAULT '{}',
			origin_url TEXT,
			origin_repository VARCHAR(255),
			
			-- Lifecycle
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expiry_at TIMESTAMP,
			
			CONSTRAINT unique_artifact_cache_key UNIQUE (cache_key, cache_level, tenant_id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_tenant ON cached_artifacts(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_type ON cached_artifacts(artifact_type, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_cache_level ON cached_artifacts(cache_level, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_scan_status ON cached_artifacts(scan_status, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_last_accessed ON cached_artifacts(last_accessed DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_created_at ON cached_artifacts(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_checksum ON cached_artifacts(checksum)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_quarantined ON cached_artifacts(is_quarantined, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_next_scan ON cached_artifacts(next_scan_at)`,

		// Cache Access Logs - Detailed access tracking
		`CREATE TABLE IF NOT EXISTS cache_access_logs (
			id BIGSERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE CASCADE,
			
			-- Access details
			accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			access_type VARCHAR(20) DEFAULT 'read' CHECK (access_type IN ('read', 'write', 'delete', 'scan')),
			hit BOOLEAN NOT NULL,
			response_time_ms INTEGER,
			cache_source VARCHAR(10) CHECK (cache_source IN ('L1', 'L2', 'L3', 'remote')),
			
			-- Client information
			user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
			client_ip INET,
			user_agent TEXT,
			
			-- Result
			status_code INTEGER,
			bytes_transferred BIGINT,
			error_message TEXT
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cache_access_logs_accessed_at ON cache_access_logs(accessed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_access_logs_artifact ON cache_access_logs(cached_artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_access_logs_tenant ON cache_access_logs(tenant_id, accessed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_access_logs_user ON cache_access_logs(user_id, accessed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_access_logs_hit ON cache_access_logs(hit, accessed_at DESC)`,

		// Scan Queue - Priority-based scanning
		`CREATE TABLE IF NOT EXISTS scan_queue (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE CASCADE,
			
			-- Artifact details
			artifact_path TEXT NOT NULL,
			artifact_type VARCHAR(50) NOT NULL,
			file_path TEXT,
			
			-- Queue management
			priority INTEGER DEFAULT 50 CHECK (priority >= 0 AND priority <= 100),
			status VARCHAR(50) DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'cancelled')),
			
			-- Scan configuration
			scan_config JSONB DEFAULT '{}',
			scanners_requested TEXT[] DEFAULT '{}',
			scan_type VARCHAR(50) DEFAULT 'full' CHECK (scan_type IN ('quick', 'full', 'deep', 'compliance')),
			
			-- Timing
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			estimated_duration_ms INTEGER,
			actual_duration_ms INTEGER,
			
			-- Retry management
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			last_error TEXT,
			
			-- Worker assignment
			worker_id VARCHAR(100),
			assigned_at TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_scan_queue_status ON scan_queue(status, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_queue_priority ON scan_queue(priority DESC, created_at ASC) WHERE status = 'queued'`,
		`CREATE INDEX IF NOT EXISTS idx_scan_queue_artifact ON scan_queue(cached_artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_queue_tenant ON scan_queue(tenant_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_queue_worker ON scan_queue(worker_id, status)`,

		// Scan History - Audit trail
		`CREATE TABLE IF NOT EXISTS scan_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE CASCADE,
			scan_queue_id UUID REFERENCES scan_queue(id) ON DELETE SET NULL,
			
			-- Artifact identification
			artifact_path TEXT NOT NULL,
			checksum VARCHAR(128),
			
			-- Scan execution
			scan_type VARCHAR(50),
			scanners_used TEXT[],
			scan_started_at TIMESTAMP NOT NULL,
			scan_completed_at TIMESTAMP,
			scan_duration_ms INTEGER,
			
			-- Results summary
			vulnerabilities_found INTEGER DEFAULT 0,
			critical_count INTEGER DEFAULT 0,
			high_count INTEGER DEFAULT 0,
			medium_count INTEGER DEFAULT 0,
			low_count INTEGER DEFAULT 0,
			info_count INTEGER DEFAULT 0,
			
			-- Detailed results
			scan_result JSONB DEFAULT '{}',
			cve_list TEXT[],
			malware_detected BOOLEAN DEFAULT false,
			secrets_detected BOOLEAN DEFAULT false,
			license_issues BOOLEAN DEFAULT false,
			
			-- Policy evaluation
			policy_violations JSONB DEFAULT '[]',
			compliance_status VARCHAR(50),
			risk_score INTEGER,
			
			-- Actions taken
			action_taken VARCHAR(50) CHECK (action_taken IN ('none', 'alert', 'quarantine', 'block', 'delete')),
			action_reason TEXT,
			
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_scan_history_artifact ON scan_history(cached_artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_history_checksum ON scan_history(checksum)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_history_completed ON scan_history(scan_completed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_history_tenant ON scan_history(tenant_id, scan_completed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_scan_history_vulnerabilities ON scan_history(vulnerabilities_found DESC)`,

		// Quarantine Records - Isolated artifacts
		`CREATE TABLE IF NOT EXISTS quarantine_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			cached_artifact_id UUID NOT NULL REFERENCES cached_artifacts(id) ON DELETE CASCADE,
			
			-- Quarantine details
			reason TEXT NOT NULL,
			severity VARCHAR(20) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
			quarantine_type VARCHAR(50) CHECK (quarantine_type IN ('manual', 'policy', 'scan_result', 'malware', 'vulnerability')),
			
			-- Evidence
			scan_history_id UUID REFERENCES scan_history(id) ON DELETE SET NULL,
			cve_ids TEXT[],
			threat_indicators JSONB DEFAULT '{}',
			
			-- Management
			quarantined_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			quarantined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			
			-- Resolution
			status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'released', 'deleted', 'expired')),
			released_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			released_at TIMESTAMP,
			release_reason TEXT,
			
			-- Auto-actions
			auto_delete_at TIMESTAMP,
			notification_sent BOOLEAN DEFAULT false,
			
			notes TEXT
		)`,

		`CREATE INDEX IF NOT EXISTS idx_quarantine_records_tenant ON quarantine_records(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quarantine_records_artifact ON quarantine_records(cached_artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quarantine_records_status ON quarantine_records(status, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quarantine_records_severity ON quarantine_records(severity, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_quarantine_records_quarantined_at ON quarantine_records(quarantined_at DESC)`,

		// Cache Policies - Automated rules
		`CREATE TABLE IF NOT EXISTS cache_policies (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			
			-- Policy identification
			name VARCHAR(255) NOT NULL,
			description TEXT,
			policy_type VARCHAR(50) CHECK (policy_type IN ('scan', 'retention', 'quarantine', 'access', 'compliance')),
			
			-- Rules
			conditions JSONB NOT NULL,
			actions JSONB NOT NULL,
			
			-- Scope
			artifact_types TEXT[],
			cache_levels TEXT[],
			
			-- Status
			enabled BOOLEAN DEFAULT true,
			priority INTEGER DEFAULT 50,
			
			-- Scheduling
			schedule_type VARCHAR(50) CHECK (schedule_type IN ('immediate', 'scheduled', 'periodic')),
			schedule_cron VARCHAR(100),
			
			created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cache_policies_tenant ON cache_policies(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_policies_type ON cache_policies(policy_type, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_policies_enabled ON cache_policies(enabled, tenant_id)`,

		// Cache Audit Logs - Compliance tracking
		`CREATE TABLE IF NOT EXISTS cache_audit_logs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			cached_artifact_id UUID REFERENCES cached_artifacts(id) ON DELETE SET NULL,
			
			-- Event details
			event_type VARCHAR(100) NOT NULL,
			event_category VARCHAR(50) CHECK (event_category IN ('access', 'modification', 'scan', 'quarantine', 'policy', 'system')),
			severity VARCHAR(20) DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
			
			-- Actor
			user_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
			user_email VARCHAR(255),
			user_ip INET,
			
			-- Details
			description TEXT,
			before_state JSONB,
			after_state JSONB,
			metadata JSONB DEFAULT '{}',
			
			-- Context
			policy_id UUID REFERENCES cache_policies(id) ON DELETE SET NULL,
			scan_id UUID REFERENCES scan_history(id) ON DELETE SET NULL,
			
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_cache_audit_logs_tenant ON cache_audit_logs(tenant_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_audit_logs_artifact ON cache_audit_logs(cached_artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_audit_logs_user ON cache_audit_logs(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_audit_logs_event ON cache_audit_logs(event_type, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_audit_logs_severity ON cache_audit_logs(severity, created_at DESC)`,

		// ============================================
		// MATERIALIZED VIEWS FOR PERFORMANCE
		// ============================================

		// Drop and recreate cache_statistics materialized view with tenant_id
		`DROP MATERIALIZED VIEW IF EXISTS cache_stats_by_artifact CASCADE`,

		// Cache Statistics Dashboard
		`CREATE MATERIALIZED VIEW cache_stats_by_artifact AS
		SELECT 
			ca.tenant_id,
			ca.artifact_type,
			ca.cache_level,
			COUNT(*) as total_items,
			SUM(ca.size_bytes) as total_size_bytes,
			SUM(ca.hit_count) as total_hits,
			SUM(ca.miss_count) as total_misses,
			CASE 
				WHEN (SUM(ca.hit_count) + SUM(ca.miss_count)) > 0 
				THEN ROUND((SUM(ca.hit_count)::numeric / (SUM(ca.hit_count) + SUM(ca.miss_count))::numeric) * 100, 2)
				ELSE 0 
			END as hit_rate_percentage,
			MAX(ca.last_accessed) as last_activity,
			COUNT(CASE WHEN ca.scan_status = 'completed' THEN 1 END) as scanned_items,
			COUNT(CASE WHEN ca.scan_status = 'pending' THEN 1 END) as pending_scans,
			COUNT(CASE WHEN ca.scan_status = 'scanning' THEN 1 END) as scanning_items,
			COUNT(CASE WHEN ca.scan_status = 'failed' THEN 1 END) as failed_scans,
			COUNT(CASE WHEN ca.is_quarantined THEN 1 END) as quarantined_items,
			SUM(ca.vulnerabilities_count) as total_vulnerabilities,
			SUM(ca.critical_vulnerabilities) as total_critical,
			SUM(ca.high_vulnerabilities) as total_high,
			NOW() as last_refreshed
		FROM cached_artifacts ca
		GROUP BY ca.tenant_id, ca.artifact_type, ca.cache_level`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cache_stats_by_artifact_unique ON cache_stats_by_artifact (tenant_id, artifact_type, cache_level)`,

		// ============================================
		// HELPER FUNCTIONS
		// ============================================

		// Function to track cache access
		`CREATE OR REPLACE FUNCTION track_cache_access(
			p_tenant_id UUID,
			p_artifact_path TEXT,
			p_cache_level VARCHAR(10),
			p_hit BOOLEAN,
			p_response_time_ms INTEGER,
			p_cache_source VARCHAR(10),
			p_client_ip INET,
			p_user_agent TEXT,
			p_user_id UUID
		)
		RETURNS UUID AS $$
		DECLARE
			v_cached_artifact_id UUID;
		BEGIN
			-- Find or create cached artifact record
			INSERT INTO cached_artifacts (tenant_id, artifact_path, artifact_type, cache_level, size_bytes, last_accessed, cache_key)
			VALUES (
				p_tenant_id,
				p_artifact_path, 
				split_part(p_artifact_path, '/', 1),
				p_cache_level,
				0,
				NOW(),
				p_artifact_path || ':' || p_cache_level
			)
			ON CONFLICT (cache_key, cache_level, tenant_id) 
			DO UPDATE SET 
				last_accessed = NOW(),
				hit_count = cached_artifacts.hit_count + CASE WHEN p_hit THEN 1 ELSE 0 END,
				miss_count = cached_artifacts.miss_count + CASE WHEN p_hit THEN 0 ELSE 1 END
			RETURNING id INTO v_cached_artifact_id;
			
			-- Log the access
			INSERT INTO cache_access_logs (
				tenant_id,
				cached_artifact_id,
				accessed_at,
				hit,
				response_time_ms,
				cache_source,
				client_ip,
				user_agent,
				user_id,
				access_type,
				status_code
			) VALUES (
				p_tenant_id,
				v_cached_artifact_id,
				NOW(),
				p_hit,
				p_response_time_ms,
				p_cache_source,
				p_client_ip,
				p_user_agent,
				p_user_id,
				'read',
				200
			);
			
			RETURN v_cached_artifact_id;
		END;
		$$ LANGUAGE plpgsql`,

		// Function to queue scan
		`CREATE OR REPLACE FUNCTION queue_cache_scan(
			p_tenant_id UUID,
			p_cached_artifact_id UUID,
			p_priority INTEGER DEFAULT 50,
			p_scan_type VARCHAR(50) DEFAULT 'full'
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
				tenant_id,
				cached_artifact_id,
				artifact_path,
				artifact_type,
				priority,
				status,
				scan_type
			) VALUES (
				p_tenant_id,
				p_cached_artifact_id,
				v_artifact_path,
				v_artifact_type,
				p_priority,
				'queued',
				p_scan_type
			)
			RETURNING id INTO v_scan_id;
			
			-- Update artifact scan status
			UPDATE cached_artifacts
			SET 
				scan_status = 'queued',
				updated_at = NOW()
			WHERE id = p_cached_artifact_id;
			
			RETURN v_scan_id;
		END;
		$$ LANGUAGE plpgsql`,

		// Function to quarantine artifact
		`CREATE OR REPLACE FUNCTION quarantine_artifact(
			p_tenant_id UUID,
			p_cached_artifact_id UUID,
			p_reason TEXT,
			p_severity VARCHAR(20),
			p_quarantine_type VARCHAR(50),
			p_user_id UUID,
			p_scan_history_id UUID DEFAULT NULL,
			p_cve_ids TEXT[] DEFAULT NULL
		)
		RETURNS UUID AS $$
		DECLARE
			v_quarantine_id UUID;
		BEGIN
			-- Create quarantine record
			INSERT INTO quarantine_records (
				tenant_id,
				cached_artifact_id,
				reason,
				severity,
				quarantine_type,
				quarantined_by,
				scan_history_id,
				cve_ids,
				status
			) VALUES (
				p_tenant_id,
				p_cached_artifact_id,
				p_reason,
				p_severity,
				p_quarantine_type,
				p_user_id,
				p_scan_history_id,
				p_cve_ids,
				'active'
			)
			RETURNING id INTO v_quarantine_id;
			
			-- Update artifact
			UPDATE cached_artifacts
			SET 
				is_quarantined = true,
				quarantine_reason = p_reason,
				quarantined_at = NOW(),
				quarantined_by = p_user_id,
				scan_status = 'quarantined',
				updated_at = NOW()
			WHERE id = p_cached_artifact_id;
			
			-- Audit log
			INSERT INTO cache_audit_logs (
				tenant_id,
				cached_artifact_id,
				event_type,
				event_category,
				severity,
				user_id,
				description,
				metadata
			) VALUES (
				p_tenant_id,
				p_cached_artifact_id,
				'artifact_quarantined',
				'quarantine',
				p_severity,
				p_user_id,
				p_reason,
				jsonb_build_object(
					'quarantine_id', v_quarantine_id,
					'quarantine_type', p_quarantine_type,
					'scan_history_id', p_scan_history_id
				)
			);
			
			RETURN v_quarantine_id;
		END;
		$$ LANGUAGE plpgsql`,

		// Function to refresh cache statistics
		`CREATE OR REPLACE FUNCTION refresh_cache_statistics()
		RETURNS VOID AS $$
		BEGIN
			REFRESH MATERIALIZED VIEW CONCURRENTLY cache_stats_by_artifact;
		END;
		$$ LANGUAGE plpgsql`,

		// Trigger to update updated_at on cached_artifacts
		`DROP TRIGGER IF EXISTS update_cached_artifacts_updated_at ON cached_artifacts`,
		`CREATE TRIGGER update_cached_artifacts_updated_at
		BEFORE UPDATE ON cached_artifacts
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()`,

		// Trigger to update updated_at on cache_policies
		`DROP TRIGGER IF EXISTS update_cache_policies_updated_at ON cache_policies`,
		`CREATE TRIGGER update_cache_policies_updated_at
		BEFORE UPDATE ON cache_policies
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()`,

		// ============================================
		// MIGRATION 021: ADD TENANT_ID TO CACHED_ARTIFACTS
		// ============================================
		// Note: tenant_id column may already exist from previous runs, using IF NOT EXISTS

		// Add tenant_id to cached_artifacts
		`ALTER TABLE cached_artifacts 
		ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE`,

		// Create index for tenant filtering
		`CREATE INDEX IF NOT EXISTS idx_cached_artifacts_tenant ON cached_artifacts(tenant_id)`,

		// Update existing records to default tenant (if any)
		`DO $$
		DECLARE
			default_tenant_id UUID;
		BEGIN
			SELECT tenant_id INTO default_tenant_id FROM tenants ORDER BY created_at ASC LIMIT 1;
			
			IF default_tenant_id IS NOT NULL THEN
				UPDATE cached_artifacts 
				SET tenant_id = default_tenant_id 
				WHERE tenant_id IS NULL;
			END IF;
		END $$`,

		// Make tenant_id NOT NULL after backfilling
		`DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name = 'cached_artifacts' 
					   AND column_name = 'tenant_id' 
					   AND is_nullable = 'YES') THEN
				ALTER TABLE cached_artifacts ALTER COLUMN tenant_id SET NOT NULL;
			END IF;
		END $$`,

		// Update unique constraint to include tenant_id
		`ALTER TABLE cached_artifacts DROP CONSTRAINT IF EXISTS unique_artifact_cache_key`,
		`ALTER TABLE cached_artifacts DROP CONSTRAINT IF EXISTS unique_artifact_path_tenant`,
		`ALTER TABLE cached_artifacts 
		ADD CONSTRAINT unique_artifact_path_tenant UNIQUE (cache_key, cache_level, tenant_id)`,

		// Add tenant_id to scan_queue
		`ALTER TABLE scan_queue 
		ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE`,

		// Update existing scan_queue records
		`UPDATE scan_queue sq
		SET tenant_id = ca.tenant_id
		FROM cached_artifacts ca
		WHERE sq.cached_artifact_id = ca.id AND sq.tenant_id IS NULL`,

		// Create index for tenant filtering on scan_queue
		`CREATE INDEX IF NOT EXISTS idx_scan_queue_tenant ON scan_queue(tenant_id)`,

		// Add tenant_id to scan_history
		`ALTER TABLE scan_history 
		ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE`,

		// Update existing scan_history records
		`UPDATE scan_history sh
		SET tenant_id = ca.tenant_id
		FROM cached_artifacts ca
		WHERE sh.cached_artifact_id = ca.id AND sh.tenant_id IS NULL`,

		// Create index for tenant filtering on scan_history
		`CREATE INDEX IF NOT EXISTS idx_scan_history_tenant ON scan_history(tenant_id)`,

		// Update the track_cache_access function to accept tenant_id
		`CREATE OR REPLACE FUNCTION track_cache_access(
			p_tenant_id UUID,
			p_artifact_path TEXT,
			p_cache_level VARCHAR(10),
			p_hit BOOLEAN,
			p_response_time_ms INTEGER,
			p_cache_source VARCHAR(10),
			p_client_ip INET,
			p_user_agent TEXT,
			p_user_id UUID
		)
		RETURNS UUID AS $$
		DECLARE
			v_cached_artifact_id UUID;
		BEGIN
			-- Find or create cached artifact record
			INSERT INTO cached_artifacts (tenant_id, artifact_path, artifact_type, cache_level, size_bytes, last_accessed, cache_key)
			VALUES (
				p_tenant_id,
				p_artifact_path, 
				split_part(p_artifact_path, '/', 1),
				p_cache_level,
				0,
				NOW(),
				p_artifact_path || ':' || p_cache_level
			)
			ON CONFLICT (cache_key, cache_level, tenant_id) 
			DO UPDATE SET 
				last_accessed = NOW(),
				hit_count = cached_artifacts.hit_count + CASE WHEN p_hit THEN 1 ELSE 0 END,
				miss_count = cached_artifacts.miss_count + CASE WHEN p_hit THEN 0 ELSE 1 END
			RETURNING id INTO v_cached_artifact_id;
			
			-- Log the access
			INSERT INTO cache_access_logs (
				tenant_id,
				cached_artifact_id,
				accessed_at,
				hit,
				response_time_ms,
				cache_source,
				client_ip,
				user_agent,
				user_id,
				access_type,
				status_code
			) VALUES (
				p_tenant_id,
				v_cached_artifact_id,
				NOW(),
				p_hit,
				p_response_time_ms,
				p_cache_source,
				p_client_ip,
				p_user_agent,
				p_user_id,
				'read',
				200
			);
			
			RETURN v_cached_artifact_id;
		END;
		$$ LANGUAGE plpgsql`,

		// Update queue_cache_scan function to accept tenant_id
		`CREATE OR REPLACE FUNCTION queue_cache_scan(
			p_tenant_id UUID,
			p_cached_artifact_id UUID,
			p_priority INTEGER DEFAULT 50,
			p_scan_type VARCHAR(50) DEFAULT 'full'
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
				tenant_id,
				cached_artifact_id,
				artifact_path,
				artifact_type,
				priority,
				status,
				scan_type
			) VALUES (
				COALESCE(p_tenant_id, v_tenant_id),
				p_cached_artifact_id,
				v_artifact_path,
				v_artifact_type,
				p_priority,
				'queued',
				p_scan_type
			)
			RETURNING id INTO v_scan_id;
			
			-- Update artifact scan status
			UPDATE cached_artifacts
			SET 
				scan_status = 'queued',
				updated_at = NOW()
			WHERE id = p_cached_artifact_id;
			
			RETURN v_scan_id;
		END;
		$$ LANGUAGE plpgsql`,

		// Migration 022: Encryption support - tenant master keys, encryption metadata, audit logging
		`CREATE TABLE IF NOT EXISTS tenant_master_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL UNIQUE,
			encrypted_key BYTEA NOT NULL,
			kms_key_id VARCHAR(255) NOT NULL,
			key_version INTEGER NOT NULL DEFAULT 1,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			rotated_at TIMESTAMP,
			created_by UUID NOT NULL,
			last_accessed_at TIMESTAMP,
			access_count BIGINT DEFAULT 0,
			
			CONSTRAINT fk_tmk_tenant FOREIGN KEY (tenant_id) 
				REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			CONSTRAINT fk_tmk_creator FOREIGN KEY (created_by) 
				REFERENCES users(user_id) ON DELETE SET NULL
		)`,

		`CREATE INDEX IF NOT EXISTS idx_tmk_tenant_active ON tenant_master_keys(tenant_id) 
			WHERE is_active = true`,
		`CREATE INDEX IF NOT EXISTS idx_tmk_created_at ON tenant_master_keys(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tmk_rotation ON tenant_master_keys(rotated_at) 
			WHERE is_active = true AND rotated_at IS NOT NULL`,

		`COMMENT ON TABLE tenant_master_keys IS 'Stores encrypted tenant master keys for envelope encryption'`,
		`COMMENT ON COLUMN tenant_master_keys.encrypted_key IS 'TMK encrypted by KMS root key'`,
		`COMMENT ON COLUMN tenant_master_keys.kms_key_id IS 'KMS key ID or ARN used to encrypt TMK'`,

		// Migration 022: Make created_by nullable for system-created TMKs
		`ALTER TABLE tenant_master_keys ALTER COLUMN created_by DROP NOT NULL`,
		`COMMENT ON COLUMN tenant_master_keys.created_by IS 'User who created the key. NULL for system-created keys (e.g., auto-created during first artifact upload).'`,

		// Add encryption columns to artifacts table
		`ALTER TABLE artifacts 
			ADD COLUMN IF NOT EXISTS encrypted BOOLEAN DEFAULT false,
			ADD COLUMN IF NOT EXISTS encryption_version INTEGER DEFAULT 1,
			ADD COLUMN IF NOT EXISTS encrypted_dek BYTEA,
			ADD COLUMN IF NOT EXISTS encryption_algorithm VARCHAR(50) DEFAULT 'AES-256-GCM',
			ADD COLUMN IF NOT EXISTS encryption_metadata JSONB`,

		`CREATE INDEX IF NOT EXISTS idx_artifacts_encrypted ON artifacts(encrypted) WHERE encrypted = true`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_enc_version ON artifacts(encryption_version)`,

		`COMMENT ON COLUMN artifacts.encrypted IS 'Whether artifact content is encrypted'`,
		`COMMENT ON COLUMN artifacts.encrypted_dek IS 'Data encryption key encrypted by KEK'`,
		`COMMENT ON COLUMN artifacts.encryption_metadata IS 'Nonce, auth tag, and other encryption params'`,

		// Create key_audit_log for compliance and security monitoring
		`CREATE TABLE IF NOT EXISTS key_audit_log (
			id BIGSERIAL PRIMARY KEY,
			event_id UUID NOT NULL DEFAULT gen_random_uuid(),
			timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
			tenant_id UUID NOT NULL,
			user_id UUID,
			key_type VARCHAR(50) NOT NULL,
			key_id VARCHAR(255),
			operation VARCHAR(50) NOT NULL,
			artifact_id UUID,
			repository_id UUID,
			source_ip INET,
			user_agent TEXT,
			success BOOLEAN NOT NULL,
			error_message TEXT,
			duration_ms INTEGER,
			audit_chain_hash BYTEA,
			metadata JSONB,
			
			CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) 
				REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			CONSTRAINT fk_audit_user FOREIGN KEY (user_id) 
				REFERENCES users(user_id) ON DELETE SET NULL
		)`,

		`CREATE INDEX IF NOT EXISTS idx_key_audit_tenant_time ON key_audit_log(tenant_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_key_audit_user_time ON key_audit_log(user_id, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_key_audit_operation ON key_audit_log(operation, timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_key_audit_artifact ON key_audit_log(artifact_id) 
			WHERE artifact_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_key_audit_success ON key_audit_log(success, timestamp DESC) 
			WHERE success = false`,

		`COMMENT ON TABLE key_audit_log IS 'Immutable audit log for all encryption key operations'`,
		`COMMENT ON COLUMN key_audit_log.audit_chain_hash IS 'SHA-256 hash linking to previous event for tamper detection'`,

		// Create function to update last_accessed_at for TMKs
		`CREATE OR REPLACE FUNCTION update_tmk_access()
		RETURNS TRIGGER AS $$
		BEGIN
			UPDATE tenant_master_keys 
			SET 
				last_accessed_at = NOW(),
				access_count = access_count + 1
			WHERE tenant_id = NEW.tenant_id 
			  AND NEW.key_type = 'TMK' 
			  AND is_active = true;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Trigger to track TMK access
		`DROP TRIGGER IF EXISTS trigger_tmk_access ON key_audit_log`,
		`CREATE TRIGGER trigger_tmk_access
			AFTER INSERT ON key_audit_log
			FOR EACH ROW
			WHEN (NEW.key_type = 'TMK' AND NEW.operation IN ('access', 'decrypt'))
			EXECUTE FUNCTION update_tmk_access()`,

		// Create encryption_settings table for tenant-specific encryption config
		`CREATE TABLE IF NOT EXISTS encryption_settings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL UNIQUE,
			encryption_enabled BOOLEAN DEFAULT true,
			enforce_encryption BOOLEAN DEFAULT false,
			allowed_algorithms TEXT[] DEFAULT ARRAY['AES-256-GCM'],
			key_rotation_days INTEGER DEFAULT 90,
			auto_rotate BOOLEAN DEFAULT true,
			kms_provider VARCHAR(50) DEFAULT 'mock',
			kms_config JSONB,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_by UUID,
			
			CONSTRAINT fk_enc_settings_tenant FOREIGN KEY (tenant_id) 
				REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			CONSTRAINT fk_enc_settings_updater FOREIGN KEY (updated_by) 
				REFERENCES users(user_id) ON DELETE SET NULL,
			CONSTRAINT check_rotation_days CHECK (key_rotation_days >= 30 AND key_rotation_days <= 365)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_enc_settings_tenant ON encryption_settings(tenant_id)`,

		`COMMENT ON TABLE encryption_settings IS 'Tenant-specific encryption configuration and policies'`,
		`COMMENT ON COLUMN encryption_settings.enforce_encryption IS 'Reject unencrypted artifact uploads if true'`,

		// Create view for encryption status dashboard
		`CREATE OR REPLACE VIEW encryption_status_view AS
		SELECT 
			t.tenant_id,
			t.name as tenant_name,
			es.encryption_enabled,
			es.enforce_encryption,
			es.key_rotation_days,
			tmk.key_version as current_tmk_version,
			tmk.rotated_at as last_rotation,
			EXTRACT(DAY FROM NOW() - COALESCE(tmk.rotated_at, tmk.created_at)) as days_since_rotation,
			tmk.access_count as tmk_access_count,
			COUNT(DISTINCT a.artifact_id) FILTER (WHERE a.encrypted = true) as encrypted_artifacts,
			COUNT(DISTINCT a.artifact_id) FILTER (WHERE a.encrypted = false) as unencrypted_artifacts,
			COUNT(DISTINCT kal.id) FILTER (WHERE kal.success = false AND kal.timestamp > NOW() - INTERVAL '24 hours') as failed_ops_24h
		FROM tenants t
		LEFT JOIN encryption_settings es ON es.tenant_id = t.tenant_id
		LEFT JOIN tenant_master_keys tmk ON tmk.tenant_id = t.tenant_id AND tmk.is_active = true
		LEFT JOIN artifacts a ON a.tenant_id = t.tenant_id
		LEFT JOIN key_audit_log kal ON kal.tenant_id = t.tenant_id
		GROUP BY t.tenant_id, t.name, es.encryption_enabled, es.enforce_encryption, 
				 es.key_rotation_days, tmk.key_version, tmk.rotated_at, 
				 tmk.created_at, tmk.access_count`,

		`COMMENT ON VIEW encryption_status_view IS 'Dashboard view for encryption status per tenant'`,

		// Insert default encryption settings for existing tenants
		`INSERT INTO encryption_settings (tenant_id, encryption_enabled, enforce_encryption, kms_provider)
		SELECT tenant_id, false, false, 'mock'
		FROM tenants
		WHERE tenant_id NOT IN (SELECT tenant_id FROM encryption_settings)`,

		// =====================================================
		// Migration 023: Artifact signing and integrity verification
		// =====================================================

		// Extend artifacts table with integrity hashes
		`ALTER TABLE artifacts
		ADD COLUMN IF NOT EXISTS sha256_hash VARCHAR(64),
		ADD COLUMN IF NOT EXISTS sha512_hash VARCHAR(128),
		ADD COLUMN IF NOT EXISTS hash_algorithm VARCHAR(20) DEFAULT 'sha256',
		ADD COLUMN IF NOT EXISTS signature_required BOOLEAN DEFAULT false,
		ADD COLUMN IF NOT EXISTS signature_verified BOOLEAN DEFAULT false,
		ADD COLUMN IF NOT EXISTS signature_verified_at TIMESTAMP`,

		`CREATE INDEX IF NOT EXISTS idx_artifacts_sha256_hash ON artifacts(sha256_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_sha512_hash ON artifacts(sha512_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_artifacts_signature_verified ON artifacts(signature_verified)`,

		// Add signature policy to repositories
		`ALTER TABLE repositories
		ADD COLUMN IF NOT EXISTS signature_policy VARCHAR(20) DEFAULT 'optional' CHECK (signature_policy IN ('disabled', 'optional', 'required', 'strict')),
		ADD COLUMN IF NOT EXISTS allowed_signers TEXT[],
		ADD COLUMN IF NOT EXISTS signature_verification_enabled BOOLEAN DEFAULT false,
		ADD COLUMN IF NOT EXISTS cosign_enabled BOOLEAN DEFAULT false,
		ADD COLUMN IF NOT EXISTS pgp_enabled BOOLEAN DEFAULT false,
		ADD COLUMN IF NOT EXISTS sigstore_enabled BOOLEAN DEFAULT false`,

		`CREATE INDEX IF NOT EXISTS idx_repositories_signature_policy ON repositories(signature_policy)`,

		// Create artifact_signatures table
		`CREATE TABLE IF NOT EXISTS artifact_signatures (
			signature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			repository_id UUID NOT NULL REFERENCES repositories(repository_id) ON DELETE CASCADE,
			signature_type VARCHAR(20) NOT NULL CHECK (signature_type IN ('cosign', 'pgp', 'sigstore', 'x509', 'ssh')),
			signature_format VARCHAR(20) NOT NULL CHECK (signature_format IN ('binary', 'ascii-armor', 'json', 'pem', 'der')),
			signature_data BYTEA NOT NULL,
			signature_algorithm VARCHAR(50),
			signer_identity VARCHAR(500),
			signer_fingerprint VARCHAR(128),
			public_key TEXT,
			public_key_url TEXT,
			verified BOOLEAN DEFAULT false,
			verification_method VARCHAR(50),
			verification_status VARCHAR(20) DEFAULT 'pending' CHECK (verification_status IN ('pending', 'valid', 'invalid', 'expired', 'revoked', 'untrusted')),
			verification_error TEXT,
			verified_at TIMESTAMP,
			verified_by UUID REFERENCES users(user_id),
			cosign_bundle JSONB,
			cosign_certificate TEXT,
			cosign_signature_digest VARCHAR(64),
			rekor_log_index BIGINT,
			rekor_uuid VARCHAR(100),
			pgp_key_id VARCHAR(40),
			pgp_key_fingerprint VARCHAR(40),
			pgp_signature_version INTEGER,
			sigstore_bundle JSONB,
			sigstore_predicate_type VARCHAR(200),
			attestation_payload JSONB,
			signature_storage_path TEXT,
			certificate_storage_path TEXT,
			signed_at TIMESTAMP,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for artifact_signatures
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_tenant_id ON artifact_signatures(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_artifact_id ON artifact_signatures(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_repository_id ON artifact_signatures(repository_id)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_type ON artifact_signatures(signature_type)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_verified ON artifact_signatures(verified)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_verification_status ON artifact_signatures(verification_status)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_signer_identity ON artifact_signatures(signer_identity)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_signer_fingerprint ON artifact_signatures(signer_fingerprint)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_expires_at ON artifact_signatures(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_artifact_signatures_lookup ON artifact_signatures(tenant_id, artifact_id, signature_type)`,

		// Create public_keys table
		`CREATE TABLE IF NOT EXISTS public_keys (
			key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			repository_id UUID REFERENCES repositories(repository_id) ON DELETE CASCADE,
			key_name VARCHAR(255) NOT NULL,
			key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('pgp', 'x509', 'cosign', 'ssh', 'jwk')),
			key_format VARCHAR(20) NOT NULL CHECK (key_format IN ('pem', 'der', 'ascii-armor', 'jwk', 'openssh')),
			public_key TEXT NOT NULL,
			key_fingerprint VARCHAR(128) NOT NULL,
			key_id_short VARCHAR(40),
			key_algorithm VARCHAR(50),
			key_size INTEGER,
			owner_email VARCHAR(255),
			owner_name VARCHAR(255),
			organization VARCHAR(255),
			trusted BOOLEAN DEFAULT false,
			enabled BOOLEAN DEFAULT true,
			revoked BOOLEAN DEFAULT false,
			revoked_at TIMESTAMP,
			revocation_reason TEXT,
			valid_from TIMESTAMP,
			valid_until TIMESTAMP,
			description TEXT,
			key_source VARCHAR(100),
			key_source_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by UUID REFERENCES users(user_id),
			UNIQUE(tenant_id, key_fingerprint)
		)`,

		// Indexes for public_keys
		`CREATE INDEX IF NOT EXISTS idx_public_keys_tenant_id ON public_keys(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_public_keys_repository_id ON public_keys(repository_id)`,
		`CREATE INDEX IF NOT EXISTS idx_public_keys_fingerprint ON public_keys(key_fingerprint)`,
		`CREATE INDEX IF NOT EXISTS idx_public_keys_trusted ON public_keys(trusted)`,
		`CREATE INDEX IF NOT EXISTS idx_public_keys_enabled ON public_keys(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_public_keys_owner_email ON public_keys(owner_email)`,

		// Create signature_verification_logs table
		`CREATE TABLE IF NOT EXISTS signature_verification_logs (
			log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			artifact_id UUID NOT NULL REFERENCES artifacts(artifact_id) ON DELETE CASCADE,
			signature_id UUID REFERENCES artifact_signatures(signature_id) ON DELETE CASCADE,
			verification_type VARCHAR(50) NOT NULL,
			verification_result VARCHAR(20) NOT NULL CHECK (verification_result IN ('success', 'failure', 'error')),
			verification_status VARCHAR(20) NOT NULL,
			verification_method VARCHAR(50),
			error_message TEXT,
			error_code VARCHAR(50),
			verified_by UUID REFERENCES users(user_id),
			client_ip VARCHAR(45),
			user_agent VARCHAR(500),
			verified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes for signature_verification_logs
		`CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_tenant_id ON signature_verification_logs(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_artifact_id ON signature_verification_logs(artifact_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_signature_id ON signature_verification_logs(signature_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_result ON signature_verification_logs(verification_result)`,
		`CREATE INDEX IF NOT EXISTS idx_sig_verification_logs_verified_at ON signature_verification_logs(verified_at)`,

		// Create views for signature status
		`CREATE OR REPLACE VIEW artifacts_with_signatures AS
		SELECT 
			a.artifact_id,
			a.tenant_id,
			a.name,
			a.version,
			a.type,
			a.repository_id,
			a.sha256_hash,
			a.sha512_hash,
			a.signature_required,
			a.signature_verified,
			a.signature_verified_at,
			COUNT(DISTINCT asig.signature_id) AS signature_count,
			COUNT(DISTINCT CASE WHEN asig.verified = true THEN asig.signature_id END) AS verified_signature_count,
			COUNT(DISTINCT CASE WHEN asig.verification_status = 'valid' THEN asig.signature_id END) AS valid_signature_count,
			MAX(asig.verified_at) AS last_verification_at,
			BOOL_OR(asig.verified) AS has_verified_signature,
			ARRAY_AGG(DISTINCT asig.signature_type) FILTER (WHERE asig.signature_type IS NOT NULL) AS signature_types
		FROM artifacts a
		LEFT JOIN artifact_signatures asig ON a.artifact_id = asig.artifact_id
		GROUP BY a.artifact_id, a.tenant_id, a.name, a.version, a.type, a.repository_id, 
				 a.sha256_hash, a.sha512_hash, a.signature_required, 
				 a.signature_verified, a.signature_verified_at`,

		`CREATE OR REPLACE VIEW repository_signature_policies AS
		SELECT 
			r.repository_id,
			r.tenant_id,
			r.name AS repository_name,
			r.type AS repository_type,
			r.signature_policy,
			r.signature_verification_enabled,
			r.cosign_enabled,
			r.pgp_enabled,
			r.sigstore_enabled,
			r.allowed_signers,
			COUNT(DISTINCT a.artifact_id) AS artifact_count,
			COUNT(DISTINCT CASE WHEN a.signature_verified = true THEN a.artifact_id END) AS verified_artifact_count,
			COUNT(DISTINCT asig.signature_id) AS total_signatures
		FROM repositories r
		LEFT JOIN artifacts a ON r.repository_id = a.repository_id
		LEFT JOIN artifact_signatures asig ON a.artifact_id = asig.artifact_id
		GROUP BY r.repository_id, r.tenant_id, r.name, r.type, r.signature_policy,
				 r.signature_verification_enabled, r.cosign_enabled, r.pgp_enabled,
				 r.sigstore_enabled, r.allowed_signers`,

		// Triggers for signature management
		`CREATE OR REPLACE FUNCTION update_artifact_signature_status()
		RETURNS TRIGGER AS $$
		BEGIN
			UPDATE artifacts
			SET 
				signature_verified = EXISTS (
					SELECT 1 FROM artifact_signatures
					WHERE artifact_id = NEW.artifact_id
					AND verified = true
					AND verification_status = 'valid'
				),
				signature_verified_at = CASE
					WHEN NEW.verified = true AND NEW.verification_status = 'valid' THEN NEW.verified_at
					ELSE signature_verified_at
				END,
				updated_at = CURRENT_TIMESTAMP
			WHERE artifact_id = NEW.artifact_id;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS trigger_update_artifact_signature_status ON artifact_signatures`,

		`CREATE TRIGGER trigger_update_artifact_signature_status
			AFTER INSERT OR UPDATE OF verified, verification_status ON artifact_signatures
			FOR EACH ROW
			EXECUTE FUNCTION update_artifact_signature_status()`,

		`CREATE OR REPLACE FUNCTION update_signature_timestamp()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS trigger_artifact_signatures_updated_at ON artifact_signatures`,

		`CREATE TRIGGER trigger_artifact_signatures_updated_at
			BEFORE UPDATE ON artifact_signatures
			FOR EACH ROW
			EXECUTE FUNCTION update_signature_timestamp()`,

		`DROP TRIGGER IF EXISTS trigger_public_keys_updated_at ON public_keys`,

		`CREATE TRIGGER trigger_public_keys_updated_at
			BEFORE UPDATE ON public_keys
			FOR EACH ROW
			EXECUTE FUNCTION update_signature_timestamp()`,

		// ============================================
		// Migration 024: Artifact Properties System
		// ============================================

		`CREATE TABLE IF NOT EXISTS artifact_properties (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			repository_id VARCHAR(255) NOT NULL,
			artifact_id VARCHAR(255) NOT NULL,
			key VARCHAR(255) NOT NULL,
			value TEXT NOT NULL,
			value_type VARCHAR(50) DEFAULT 'string',
			is_sensitive BOOLEAN DEFAULT FALSE,
			is_system BOOLEAN DEFAULT FALSE,
			is_multi_value BOOLEAN DEFAULT FALSE,
			encrypted_value TEXT,
			encryption_key_id VARCHAR(255),
			encryption_algorithm VARCHAR(50),
			nonce VARCHAR(255),
			created_by UUID REFERENCES users(user_id),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_by UUID REFERENCES users(user_id),
			updated_at TIMESTAMP DEFAULT NOW(),
			version INTEGER DEFAULT 1,
			tags TEXT[],
			description TEXT,
			CONSTRAINT artifact_properties_key_format CHECK (key ~ '^[a-zA-Z0-9._-]+$'),
			CONSTRAINT artifact_properties_value_size CHECK (length(value) <= 65535)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_lookup 
		 ON artifact_properties(tenant_id, repository_id, artifact_id)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_key 
		 ON artifact_properties(tenant_id, key)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_key_value 
		 ON artifact_properties(tenant_id, key, value)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_artifact 
		 ON artifact_properties(tenant_id, artifact_id)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_system 
		 ON artifact_properties(tenant_id, is_system)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_search 
		 ON artifact_properties USING GIN(to_tsvector('english', key || ' ' || value))`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_created 
		 ON artifact_properties(tenant_id, created_at DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_artifact_properties_updated 
		 ON artifact_properties(tenant_id, updated_at DESC)`,

		`CREATE MATERIALIZED VIEW IF NOT EXISTS property_search_index AS
		 SELECT 
			ap.id,
			ap.tenant_id,
			ap.repository_id,
			ap.artifact_id,
			ap.key,
			ap.value,
			ap.value_type,
			ap.is_sensitive,
			ap.is_system,
			ap.created_at,
			ap.updated_at,
			to_tsvector('english', ap.key || ' ' || ap.value) as search_vector
		 FROM artifact_properties ap
		 WHERE ap.is_sensitive = FALSE`,

		`CREATE INDEX IF NOT EXISTS idx_property_search_vector 
		 ON property_search_index USING GIN(search_vector)`,

		`CREATE INDEX IF NOT EXISTS idx_property_search_tenant 
		 ON property_search_index(tenant_id)`,

		`CREATE TABLE IF NOT EXISTS property_audit_log (
			id BIGSERIAL PRIMARY KEY,
			tenant_id UUID NOT NULL,
			property_id UUID,
			artifact_id VARCHAR(255) NOT NULL,
			action VARCHAR(50) NOT NULL,
			key VARCHAR(255) NOT NULL,
			old_value TEXT,
			new_value TEXT,
			user_id UUID,
			username VARCHAR(255),
			ip_address INET,
			user_agent TEXT,
			timestamp TIMESTAMP DEFAULT NOW(),
			correlation_id UUID,
			metadata JSONB
		)`,

		`CREATE INDEX IF NOT EXISTS idx_property_audit_tenant 
		 ON property_audit_log(tenant_id, timestamp DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_property_audit_artifact 
		 ON property_audit_log(artifact_id, timestamp DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_property_audit_user 
		 ON property_audit_log(user_id, timestamp DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_property_audit_action 
		 ON property_audit_log(tenant_id, action, timestamp DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_property_audit_correlation 
		 ON property_audit_log(correlation_id)`,

		`CREATE TABLE IF NOT EXISTS property_templates (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			category VARCHAR(100),
			properties JSONB NOT NULL,
			is_system BOOLEAN DEFAULT FALSE,
			is_active BOOLEAN DEFAULT TRUE,
			created_by UUID REFERENCES users(user_id),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(tenant_id, name)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_property_templates_tenant 
		 ON property_templates(tenant_id, is_active)`,

		`CREATE INDEX IF NOT EXISTS idx_property_templates_category 
		 ON property_templates(tenant_id, category)`,

		// Note: Property permissions need to be added per-tenant through the application
		// or manually as they require a tenant_id

		`CREATE OR REPLACE FUNCTION refresh_property_search_index()
		 RETURNS void AS $$
		 BEGIN
			REFRESH MATERIALIZED VIEW CONCURRENTLY property_search_index;
		 END;
		 $$ LANGUAGE plpgsql`,

		`CREATE OR REPLACE FUNCTION log_property_change()
		 RETURNS TRIGGER AS $$
		 BEGIN
			IF TG_OP = 'INSERT' THEN
				INSERT INTO property_audit_log (
					tenant_id, property_id, artifact_id, action, key, new_value, user_id
				) VALUES (
					NEW.tenant_id, NEW.id, NEW.artifact_id, 'CREATE', NEW.key, 
					CASE WHEN NEW.is_sensitive THEN '***REDACTED***' ELSE NEW.value END,
					NEW.created_by
				);
			ELSIF TG_OP = 'UPDATE' THEN
				INSERT INTO property_audit_log (
					tenant_id, property_id, artifact_id, action, key, old_value, new_value, user_id
				) VALUES (
					NEW.tenant_id, NEW.id, NEW.artifact_id, 'UPDATE', NEW.key,
					CASE WHEN OLD.is_sensitive THEN '***REDACTED***' ELSE OLD.value END,
					CASE WHEN NEW.is_sensitive THEN '***REDACTED***' ELSE NEW.value END,
					NEW.updated_by
				);
			ELSIF TG_OP = 'DELETE' THEN
				INSERT INTO property_audit_log (
					tenant_id, property_id, artifact_id, action, key, old_value
				) VALUES (
					OLD.tenant_id, OLD.id, OLD.artifact_id, 'DELETE', OLD.key,
					CASE WHEN OLD.is_sensitive THEN '***REDACTED***' ELSE OLD.value END
				);
			END IF;
			RETURN NEW;
		 END;
		 $$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS trigger_log_property_change ON artifact_properties`,

		`CREATE TRIGGER trigger_log_property_change
		 AFTER INSERT OR UPDATE OR DELETE ON artifact_properties
		 FOR EACH ROW EXECUTE FUNCTION log_property_change()`,

		`CREATE OR REPLACE VIEW v_artifact_properties_public AS
		 SELECT 
			ap.id,
			ap.tenant_id,
			ap.repository_id,
			ap.artifact_id,
			ap.key,
			ap.value,
			ap.value_type,
			ap.is_system,
			ap.is_multi_value,
			ap.created_by,
			ap.created_at,
			ap.updated_by,
			ap.updated_at,
			ap.description,
			ap.tags
		 FROM artifact_properties ap
		 WHERE ap.is_sensitive = FALSE`,

		`CREATE OR REPLACE VIEW v_property_statistics AS
		 SELECT 
			tenant_id,
			COUNT(*) as total_properties,
			COUNT(DISTINCT artifact_id) as artifacts_with_properties,
			COUNT(DISTINCT key) as unique_keys,
			COUNT(*) FILTER (WHERE is_sensitive = TRUE) as sensitive_properties,
			COUNT(*) FILTER (WHERE is_system = TRUE) as system_properties,
			COUNT(*) FILTER (WHERE is_multi_value = TRUE) as multi_value_properties,
			MAX(created_at) as last_property_added
		 FROM artifact_properties
		 GROUP BY tenant_id`,

		// Migration 017: S3/MinIO Replication Node Support
		`ALTER TABLE replication_nodes 
			ADD COLUMN IF NOT EXISTS node_type VARCHAR(20) DEFAULT 'local' CHECK (node_type IN ('local', 's3', 'minio', 'gcs', 'azure')),
			ADD COLUMN IF NOT EXISTS s3_endpoint VARCHAR(500),
			ADD COLUMN IF NOT EXISTS s3_region VARCHAR(100),
			ADD COLUMN IF NOT EXISTS s3_bucket VARCHAR(255),
			ADD COLUMN IF NOT EXISTS s3_access_key TEXT,
			ADD COLUMN IF NOT EXISTS s3_secret_key TEXT,
			ADD COLUMN IF NOT EXISTS s3_use_ssl BOOLEAN DEFAULT true,
			ADD COLUMN IF NOT EXISTS s3_path_prefix VARCHAR(500)`,

		`CREATE INDEX IF NOT EXISTS idx_replication_nodes_type 
			ON replication_nodes(tenant_id, node_type, is_active)`,

		`UPDATE replication_nodes SET node_type = 'local' WHERE node_type IS NULL`,

		`COMMENT ON COLUMN replication_nodes.node_type IS 'Type of storage node: local (filesystem), s3 (AWS S3), minio (MinIO), gcs (Google Cloud Storage), azure (Azure Blob)'`,

		`COMMENT ON COLUMN replication_nodes.s3_endpoint IS 'S3-compatible endpoint URL (e.g., http://127.0.0.1:9000 for MinIO)'`,

		`COMMENT ON COLUMN replication_nodes.s3_bucket IS 'S3 bucket name for remote storage'`,

		`COMMENT ON COLUMN replication_nodes.s3_path_prefix IS 'Optional path prefix within the bucket'`,
	}

	// Execute all migrations
	for i, migration := range migrations {
		if _, err := db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migration %d failed: %w\nSQL: %s", i, err, migration)
		}
	}

	log.Println("✅ UUID-based database migrations completed successfully!")
	return nil
}
