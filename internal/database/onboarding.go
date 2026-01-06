package database

import (
	"database/sql"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// InitializeDefaultData creates the default admin tenant, user, roles, and OAuth2 scopes
// This runs automatically on first startup if no tenants exist
func InitializeDefaultData(db *sql.DB) error {
	log.Println("🔍 Checking if initial setup is required...")

	// Check if any tenants exist
	var tenantCount int
	err := db.QueryRow("SELECT COUNT(*) FROM tenants").Scan(&tenantCount)
	if err != nil {
		return fmt.Errorf("failed to check tenant count: %w", err)
	}

	if tenantCount > 0 {
		log.Println("✓ Tenants already exist, skipping initial setup")
		return nil
	}

	log.Println("🚀 No tenants found. Running first-time setup...")
	log.Println("   Creating default admin tenant and user...")

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create default admin tenant
	var tenantID string
	err = tx.QueryRow(`
		INSERT INTO tenants (name, slug, description, contact_email, is_active, plan, max_users, max_repositories)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING tenant_id
	`, "Admin Organization", "admin", "Default admin organization", "admin@securestor.io", true, "enterprise", 100, 1000).Scan(&tenantID)

	if err != nil {
		return fmt.Errorf("failed to create admin tenant: %w", err)
	}
	log.Printf("✓ Created admin tenant: %s", tenantID)

	// Generate bcrypt hash for default password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	// Create default admin user
	var userID string
	err = tx.QueryRow(`
		INSERT INTO users (tenant_id, username, email, first_name, last_name, password_hash, is_active, is_email_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING user_id
	`, tenantID, "admin", "admin@securestor.io", "Admin", "User", string(passwordHash), true, true).Scan(&userID)

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}
	log.Printf("✓ Created admin user: %s", userID)

	// Create default roles
	roles := []struct {
		name        string
		description string
	}{
		{"admin", "Full system administrator with all permissions"},
		{"developer", "Developer with artifact management permissions"},
		{"viewer", "Read-only access to repositories and artifacts"},
		{"scanner", "Security scanning and vulnerability management"},
		{"auditor", "Audit log and compliance review access"},
		{"deployer", "Deployment and artifact publication permissions"},
	}

	roleIDs := make(map[string]string)
	for _, role := range roles {
		var roleID string
		err = tx.QueryRow(`
			INSERT INTO roles_uuid (tenant_id, role_name, description)
			VALUES ($1, $2, $3)
			RETURNING role_id
		`, tenantID, role.name, role.description).Scan(&roleID)

		if err != nil {
			return fmt.Errorf("failed to create role %s: %w", role.name, err)
		}
		roleIDs[role.name] = roleID
		log.Printf("✓ Created role: %s", role.name)
	}

	// Assign admin role to admin user
	_, err = tx.Exec(`
		INSERT INTO user_roles_uuid (tenant_id, user_id, role_id)
		VALUES ($1, $2, $3)
	`, tenantID, userID, roleIDs["admin"])

	if err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}
	log.Println("✓ Assigned admin role to admin user")

	// Create default permissions for admin role
	permissions := []struct {
		resource string
		action   string
	}{
		// Repository permissions
		{"repositories", "create"},
		{"repositories", "read"},
		{"repositories", "update"},
		{"repositories", "delete"},

		// Artifact permissions
		{"artifacts", "upload"},
		{"artifacts", "download"},
		{"artifacts", "delete"},
		{"artifacts", "read"},

		// Security permissions
		{"security", "scan"},
		{"security", "view_results"},
		{"security", "manage_policies"},

		// User management
		{"users", "create"},
		{"users", "read"},
		{"users", "update"},
		{"users", "delete"},

		// Role management
		{"roles", "create"},
		{"roles", "read"},
		{"roles", "update"},
		{"roles", "delete"},

		// Tenant management
		{"tenants", "read"},
		{"tenants", "update"},
		{"tenants", "manage_settings"},

		// Audit and compliance
		{"audit_logs", "read"},
		{"compliance", "read"},
		{"compliance", "manage_policies"},

		// API keys
		{"api_keys", "create"},
		{"api_keys", "read"},
		{"api_keys", "revoke"},
	}

	for _, perm := range permissions {
		var permID string
		permName := fmt.Sprintf("%s:%s", perm.resource, perm.action)
		// First create the permission in permissions_uuid table
		err = tx.QueryRow(`
			INSERT INTO permissions_uuid (permission_name, resource, action, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (permission_name) DO UPDATE SET description = $4
			RETURNING permission_id
		`, permName, perm.resource, perm.action, fmt.Sprintf("Permission to %s %s", perm.action, perm.resource)).Scan(&permID)

		if err != nil {
			log.Printf("⚠️  Warning: Could not create/get permission %s:%s: %v", perm.resource, perm.action, err)
			continue
		}

		// Assign permission to admin role
		_, err = tx.Exec(`
			INSERT INTO role_permissions_uuid (tenant_id, role_id, permission_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, tenantID, roleIDs["admin"], permID)

		if err != nil {
			log.Printf("⚠️  Warning: Could not assign permission to admin role: %v", err)
		}
	}
	log.Printf("✓ Created and assigned %d permissions to admin role", len(permissions))

	// Create default OAuth2 scopes
	scopes := []struct {
		name        string
		resource    string
		isDefault   bool
		description string
	}{
		{"repo:read", "repositories", true, "Read access to repositories"},
		{"repo:write", "repositories", true, "Write access to repositories"},
		{"repo:delete", "repositories", false, "Delete repositories"},
		{"artifact:read", "artifacts", true, "Read artifacts"},
		{"artifact:write", "artifacts", true, "Upload artifacts"},
		{"artifact:delete", "artifacts", false, "Delete artifacts"},
		{"security:scan", "security", false, "Trigger security scans"},
		{"security:read", "security", true, "View security scan results"},
		{"user:read", "users", false, "Read user information"},
		{"user:write", "users", false, "Manage users"},
		{"admin:all", "admin", false, "Full administrative access"},
	}

	for _, scope := range scopes {
		_, err = tx.Exec(`
			INSERT INTO oauth2_scopes (tenant_id, name, resource, is_default, description)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, name) DO NOTHING
		`, tenantID, scope.name, scope.resource, scope.isDefault, scope.description)

		if err != nil {
			log.Printf("⚠️  Warning: Could not create OAuth2 scope %s: %v", scope.name, err)
		}
	}
	log.Printf("✓ Created %d OAuth2 scopes", len(scopes))

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("✅ First-time setup completed successfully!")
	log.Println("")
	log.Println("================================================")
	log.Println("  DEFAULT CREDENTIALS (Change immediately!)    ")
	log.Println("================================================")
	log.Println("  Username: admin")
	log.Println("  Password: admin123")
	log.Println("  Tenant:   admin")
	log.Println("================================================")
	log.Println("")

	return nil
}
