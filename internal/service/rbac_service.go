package service

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/securestor/securestor/internal/models"
)

// RBACService handles role-based access control operations
type RBACService struct {
	db     *sql.DB
	logger *log.Logger
}

// NewRBACService creates a new RBAC service
func NewRBACService(db *sql.DB, logger *log.Logger) *RBACService {
	return &RBACService{
		db:     db,
		logger: logger,
	}
}

// PermissionChecker provides methods for checking user permissions
type PermissionChecker struct {
	service *RBACService
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(service *RBACService) *PermissionChecker {
	return &PermissionChecker{
		service: service,
	}
}

// GetUserPermissions retrieves all permissions for a user
func (s *RBACService) GetUserPermissions(userID int64) (map[string][]string, error) {
	query := `
		SELECT p.resource, p.action
		FROM user_permissions_view upv
		JOIN permissions p ON upv.permission_id = p.id
		WHERE upv.user_id = $1
		ORDER BY p.resource, p.action`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user permissions: %w", err)
	}
	defer rows.Close()

	permissions := make(map[string][]string)

	for rows.Next() {
		var resource, action string
		if err := rows.Scan(&resource, &action); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}

		permissions[resource] = append(permissions[resource], action)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permissions: %w", err)
	}

	return permissions, nil
}

// GetUserRoles retrieves all roles for a user
func (s *RBACService) GetUserRoles(userID int64) ([]string, error) {
	query := `
		SELECT r.name
		FROM user_role_view urv
		JOIN roles r ON urv.role_id = r.id
		WHERE urv.user_id = $1
		ORDER BY r.name`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, roleName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return roles, nil
}

// HasPermission checks if a user has a specific permission
func (pc *PermissionChecker) HasPermission(userID int64, resource, action string) (bool, error) {
	query := `
		SELECT 1
		FROM user_permissions_view upv
		JOIN permissions p ON upv.permission_id = p.id
		WHERE upv.user_id = $1 AND p.resource = $2 AND (p.action = $3 OR p.action = 'admin')
		LIMIT 1`

	var exists int
	err := pc.service.db.QueryRow(query, userID, resource, action).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return true, nil
}

// HasRole checks if a user has a specific role
func (pc *PermissionChecker) HasRole(userID int64, roleName string) (bool, error) {
	query := `
		SELECT 1
		FROM user_role_view urv
		JOIN roles r ON urv.role_id = r.id
		WHERE urv.user_id = $1 AND r.name = $2
		LIMIT 1`

	var exists int
	err := pc.service.db.QueryRow(query, userID, roleName).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check role: %w", err)
	}

	return true, nil
}

// HasAnyRole checks if a user has any of the specified roles
func (pc *PermissionChecker) HasAnyRole(userID int64, roleNames []string) (bool, error) {
	if len(roleNames) == 0 {
		return false, nil
	}

	// Build query with IN clause
	query := `
		SELECT 1
		FROM user_role_view urv
		JOIN roles r ON urv.role_id = r.id
		WHERE urv.user_id = $1 AND r.name = ANY($2)
		LIMIT 1`

	var exists int
	err := pc.service.db.QueryRow(query, userID, roleNames).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check roles: %w", err)
	}

	return true, nil
}

// IsAdmin checks if a user has admin role
func (pc *PermissionChecker) IsAdmin(userID int64) (bool, error) {
	return pc.HasRole(userID, "admin")
}

// AssignRoleToUser assigns a role to a user
func (s *RBACService) AssignRoleToUser(userID, roleID int64, assignedBy *int64) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by, assigned_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, role_id) DO NOTHING`

	_, err := s.db.Exec(query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (s *RBACService) RemoveRoleFromUser(userID, roleID int64) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`

	_, err := s.db.Exec(query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	return nil
}

// GetRoleByName retrieves a role by name
func (s *RBACService) GetRoleByName(name string) (*models.Role, error) {
	role := &models.Role{}
	query := `
		SELECT id, name, display_name, description, is_system_role, created_at, updated_at
		FROM roles WHERE name = $1`

	err := s.db.QueryRow(query, name).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSystemRole, &role.CreatedAt, &role.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

// CreateRole creates a new role
func (s *RBACService) CreateRole(role *models.Role) error {
	query := `
		INSERT INTO roles (name, display_name, description, is_system_role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	err := s.db.QueryRow(query,
		role.Name, role.DisplayName, role.Description, role.IsSystemRole,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetPermissionByName retrieves a permission by name
func (s *RBACService) GetPermissionByName(name string) (*models.Permission, error) {
	permission := &models.Permission{}
	query := `
		SELECT id, name, resource, action, description, created_at
		FROM permissions WHERE name = $1`

	err := s.db.QueryRow(query, name).Scan(
		&permission.ID, &permission.Name, &permission.Resource,
		&permission.Action, &permission.Description, &permission.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return permission, nil
}

// CreatePermission creates a new permission
func (s *RBACService) CreatePermission(permission *models.Permission) error {
	query := `
		INSERT INTO permissions (name, resource, action, description, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at`

	err := s.db.QueryRow(query,
		permission.Name, permission.Resource, permission.Action, permission.Description,
	).Scan(&permission.ID, &permission.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	return nil
}

// AssignPermissionToRole assigns a permission to a role
func (s *RBACService) AssignPermissionToRole(roleID, permissionID int64) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (role_id, permission_id) DO NOTHING`

	_, err := s.db.Exec(query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}

	return nil
}

// RemovePermissionFromRole removes a permission from a role
func (s *RBACService) RemovePermissionFromRole(roleID, permissionID int64) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`

	_, err := s.db.Exec(query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	return nil
}

// SyncUserRolesFromClaims synchronizes user roles from OIDC token claims
func (s *RBACService) SyncUserRolesFromClaims(userID int64, claimRoles []string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current user roles
	currentRoles, err := s.GetUserRoles(userID)
	if err != nil {
		return fmt.Errorf("failed to get current user roles: %w", err)
	}

	// Convert to maps for easier comparison
	currentRoleMap := make(map[string]bool)
	for _, role := range currentRoles {
		currentRoleMap[role] = true
	}

	claimRoleMap := make(map[string]bool)
	for _, role := range claimRoles {
		claimRoleMap[role] = true
	}

	// Add new roles
	for _, role := range claimRoles {
		if !currentRoleMap[role] {
			// Get role ID
			roleRecord, err := s.GetRoleByName(role)
			if err != nil {
				// Skip if role doesn't exist in our system
				s.logger.Printf("Warning: Role '%s' from claims not found in system", role)
				continue
			}

			// Assign role to user
			_, err = tx.Exec(`
				INSERT INTO user_roles (user_id, role_id, assigned_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT (user_id, role_id) DO NOTHING`,
				userID, roleRecord.ID)
			if err != nil {
				return fmt.Errorf("failed to assign role %s to user: %w", role, err)
			}
		}
	}

	// Remove roles that are no longer in claims (only for non-system roles)
	for _, role := range currentRoles {
		if !claimRoleMap[role] {
			// Check if it's a system role before removing
			roleRecord, err := s.GetRoleByName(role)
			if err != nil {
				continue
			}

			if !roleRecord.IsSystemRole {
				_, err = tx.Exec(`
					DELETE FROM user_roles 
					WHERE user_id = $1 AND role_id = $2`,
					userID, roleRecord.ID)
				if err != nil {
					return fmt.Errorf("failed to remove role %s from user: %w", role, err)
				}
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit role sync: %w", err)
	}

	return nil
}

// EnhanceAuthContext enriches the auth context with database-stored permissions
func (s *RBACService) EnhanceAuthContext(authCtx *models.AuthContext, userID int64) error {
	// Get user permissions from database
	permissions, err := s.GetUserPermissions(userID)
	if err != nil {
		return fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Get user roles from database
	roles, err := s.GetUserRoles(userID)
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	// Set permissions and roles
	authCtx.Permissions = permissions
	authCtx.Roles = roles
	authCtx.IsAdmin = false

	// Check if user is admin
	for _, role := range roles {
		if role == "admin" {
			authCtx.IsAdmin = true
			break
		}
	}

	return nil
}
