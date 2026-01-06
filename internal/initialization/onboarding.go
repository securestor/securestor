package initialization

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin123"
	defaultAdminEmail    = "admin@securestor.io"
	defaultTenantName    = "Admin Organization"
	defaultTenantSlug    = "admin"
)

// RunOnboarding performs first-time setup: creates admin tenant, user, roles, and OAuth2 scopes
func RunOnboarding(db *sql.DB) error {
	log.Println("🚀 Checking if onboarding is required...")

	// Check if admin tenant already exists
	var tenantExists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)", defaultTenantSlug).Scan(&tenantExists)
	if err != nil {
		return fmt.Errorf("failed to check tenant existence: %v", err)
	}

	if tenantExists {
		log.Println("✅ System already initialized, skipping onboarding")
		return nil
	}

	log.Println("🎯 First-time setup detected, running automatic onboarding...")

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// 1. Create admin tenant
	tenantID, err := createAdminTenant(tx)
	if err != nil {
		return fmt.Errorf("failed to create admin tenant: %v", err)
	}
	log.Printf("✅ Created admin tenant: %s", tenantID)

	// 2. Create admin user
	userID, err := createAdminUser(tx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %v", err)
	}
	log.Printf("✅ Created admin user: %s", userID)

	// 3. Create default roles
	roleIDs, err := createDefaultRoles(tx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to create roles: %v", err)
	}
	log.Printf("✅ Created %d default roles", len(roleIDs))

	// 4. Assign admin role to admin user
	if err := assignAdminRole(tx, userID, roleIDs["admin"], tenantID); err != nil {
		return fmt.Errorf("failed to assign admin role: %v", err)
	}
	log.Println("✅ Assigned admin role to admin user")

	// 5. Create OAuth2 scopes
	if err := createOAuth2Scopes(tx, tenantID); err != nil {
		return fmt.Errorf("failed to create OAuth2 scopes: %v", err)
	}
	log.Println("✅ Created OAuth2 scopes")

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Println("🎉 Onboarding completed successfully!")
	log.Printf("📝 Default credentials - Username: %s, Password: %s, Tenant: %s",
		defaultAdminUsername, defaultAdminPassword, defaultTenantSlug)
	log.Println("⚠️  IMPORTANT: Please change the default password immediately after first login!")

	return nil
}

func createAdminTenant(tx *sql.Tx) (string, error) {
	var tenantID string
	err := tx.QueryRow(`
		INSERT INTO tenants (name, slug, description, contact_email, is_active, plan, max_users, max_repositories, features)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING tenant_id
	`, defaultTenantName, defaultTenantSlug, "Default admin organization", defaultAdminEmail, true, "enterprise", 100, 1000, `{"all"}`).Scan(&tenantID)

	if err != nil {
		return "", err
	}
	return tenantID, nil
}

func createAdminUser(tx *sql.Tx, tenantID string) (string, error) {
	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %v", err)
	}

	var userID string
	err = tx.QueryRow(`
		INSERT INTO users (tenant_id, username, email, password_hash, first_name, last_name, is_active, email_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING user_id
	`, tenantID, defaultAdminUsername, defaultAdminEmail, string(hashedPassword), "Admin", "User", true, true).Scan(&userID)

	if err != nil {
		return "", err
	}
	return userID, nil
}

func createDefaultRoles(tx *sql.Tx, tenantID string) (map[string]string, error) {
	roles := []struct {
		name        string
		description string
		permissions []string
	}{
		{
			name:        "admin",
			description: "Full system administrator with all permissions",
			permissions: []string{"*"},
		},
		{
			name:        "developer",
			description: "Can manage repositories and artifacts",
			permissions: []string{"repositories:*", "artifacts:*", "security:read"},
		},
		{
			name:        "viewer",
			description: "Read-only access to repositories and artifacts",
			permissions: []string{"repositories:read", "artifacts:read"},
		},
		{
			name:        "scanner",
			description: "Can trigger and view security scans",
			permissions: []string{"security:*", "artifacts:read"},
		},
		{
			name:        "auditor",
			description: "Can view audit logs and compliance reports",
			permissions: []string{"logs:read", "compliance:read"},
		},
		{
			name:        "deployer",
			description: "Can deploy artifacts and manage API keys",
			permissions: []string{"artifacts:create", "artifacts:read", "apikeys:*"},
		},
	}

	roleIDs := make(map[string]string)

	for _, role := range roles {
		var roleID string
		err := tx.QueryRow(`
			INSERT INTO roles_uuid (tenant_id, name, description, created_at)
			VALUES ($1, $2, $3, $4)
			RETURNING role_id
		`, tenantID, role.name, role.description, time.Now()).Scan(&roleID)

		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %v", role.name, err)
		}

		roleIDs[role.name] = roleID

		// Create permissions for this role
		for _, permission := range role.permissions {
			_, err = tx.Exec(`
				INSERT INTO permissions (tenant_id, role_id, resource, action, created_at)
				VALUES ($1, $2, $3, $4, $5)
			`, tenantID, roleID, permission, "all", time.Now())

			if err != nil {
				return nil, fmt.Errorf("failed to create permission %s for role %s: %v", permission, role.name, err)
			}
		}
	}

	return roleIDs, nil
}

func assignAdminRole(tx *sql.Tx, userID, roleID, tenantID string) error {
	_, err := tx.Exec(`
		INSERT INTO user_roles_uuid (tenant_id, user_id, role_id, assigned_at)
		VALUES ($1, $2, $3, $4)
	`, tenantID, userID, roleID, time.Now())

	return err
}

func createOAuth2Scopes(tx *sql.Tx, tenantID string) error {
	scopes := []struct {
		name        string
		resource    string
		description string
		isDefault   bool
	}{
		{"artifacts:read", "artifacts", "Read artifacts", true},
		{"artifacts:write", "artifacts", "Upload and modify artifacts", false},
		{"artifacts:delete", "artifacts", "Delete artifacts", false},
		{"repositories:read", "repositories", "Read repository information", true},
		{"repositories:write", "repositories", "Create and modify repositories", false},
		{"repositories:delete", "repositories", "Delete repositories", false},
		{"security:read", "security", "View security scans", false},
		{"security:write", "security", "Trigger security scans", false},
		{"logs:read", "logs", "View audit logs", false},
		{"apikeys:read", "apikeys", "List API keys", false},
		{"apikeys:write", "apikeys", "Create and modify API keys", false},
		{"apikeys:delete", "apikeys", "Delete API keys", false},
		{"admin:all", "admin", "Full administrative access", false},
	}

	for _, scope := range scopes {
		_, err := tx.Exec(`
			INSERT INTO oauth2_scopes (tenant_id, name, resource, description, is_default)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tenant_id, name) DO NOTHING
		`, tenantID, scope.name, scope.resource, scope.description, scope.isDefault)

		if err != nil {
			return fmt.Errorf("failed to create scope %s: %v", scope.name, err)
		}
	}

	return nil
}

// CheckDefaultPasswordInUse checks if the admin user is still using the default password
func CheckDefaultPasswordInUse(db *sql.DB, username string) (bool, error) {
	var passwordHash string
	err := db.QueryRow(`
		SELECT password_hash FROM users WHERE username = $1
	`, username).Scan(&passwordHash)

	if err != nil {
		return false, err
	}

	// Check if the current password hash matches default password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(defaultAdminPassword))
	return err == nil, nil
}
