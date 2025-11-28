package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// RoleManagementService handles role and permission management
type RoleManagementService struct {
	db *sql.DB
}

// NewRoleManagementService creates a new role management service
func NewRoleManagementService(db *sql.DB) *RoleManagementService {
	return &RoleManagementService{
		db: db,
	}
}

// RoleFilter represents filtering options for role queries
type RoleFilter struct {
	Search        string
	IsSystemRole  *bool
	IncludeSystem bool
	Limit         int
	Offset        int
	SortBy        string
	SortOrder     string
}

// PermissionFilter represents filtering options for permission queries
type PermissionFilter struct {
	Search    string
	Resource  string
	Action    string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// RoleWithPermissions represents a role with its associated permissions
type RoleWithPermissions struct {
	models.Role
	Permissions []models.Permission `json:"permissions"`
	UserCount   int                 `json:"user_count"`
}

// GetRoles retrieves roles with filtering and pagination
func (s *RoleManagementService) GetRoles(ctx context.Context, filter *RoleFilter) ([]*RoleWithPermissions, int, error) {
	// Build the WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if filter.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR display_name ILIKE $%d OR description ILIKE $%d)",
			argCount, argCount, argCount)
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm)
	}

	if filter.IsSystemRole != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND is_system_role = $%d", argCount)
		args = append(args, *filter.IsSystemRole)
	}

	if !filter.IncludeSystem {
		whereClause += " AND is_system_role = false"
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM roles %s", whereClause)
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get role count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if filter.SortBy != "" {
		sortOrder := "ASC"
		if strings.ToUpper(filter.SortOrder) == "DESC" {
			sortOrder = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", filter.SortBy, sortOrder)
	}

	// Build LIMIT and OFFSET
	limit := 50
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset >= 0 {
		offset = filter.Offset
	}

	argCount++
	limitClause := fmt.Sprintf("LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	offsetClause := fmt.Sprintf("OFFSET $%d", argCount)
	args = append(args, offset)

	// Main query with user count
	query := fmt.Sprintf(`
		SELECT r.role_id, r.name, r.display_name, r.description, r.is_system_role, 
		       r.created_at, r.updated_at, COALESCE(user_counts.user_count, 0) as user_count
		FROM roles r
		LEFT JOIN (
			SELECT role_id, COUNT(*) as user_count
			FROM user_roles ur
			JOIN users u ON ur.user_id = u.user_id AND u.is_active = true
			GROUP BY role_id
		) user_counts ON r.role_id = user_counts.role_id
		%s %s %s %s
	`, whereClause, orderBy, limitClause, offsetClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []*RoleWithPermissions
	for rows.Next() {
		role := &RoleWithPermissions{}
		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.IsSystemRole, &role.CreatedAt, &role.UpdatedAt, &role.UserCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan role: %w", err)
		}

		// Load role permissions
		permissions, err := s.getRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get role permissions: %w", err)
		}
		role.Permissions = permissions

		roles = append(roles, role)
	}

	return roles, totalCount, nil
}

// GetRoleByID retrieves a role by ID with permissions
func (s *RoleManagementService) GetRoleByID(ctx context.Context, roleID uuid.UUID) (*RoleWithPermissions, error) {
	query := `
		SELECT r.role_id, r.name, r.display_name, r.description, r.is_system_role, 
		       r.created_at, r.updated_at, COALESCE(user_counts.user_count, 0) as user_count
		FROM roles r
		LEFT JOIN (
			SELECT role_id, COUNT(*) as user_count
			FROM user_roles ur
			JOIN users u ON ur.user_id = u.user_id AND u.is_active = true
			GROUP BY role_id
		) user_counts ON r.role_id = user_counts.role_id
		WHERE r.role_id = $1
	`

	role := &RoleWithPermissions{}
	err := s.db.QueryRowContext(ctx, query, roleID).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSystemRole, &role.CreatedAt, &role.UpdatedAt, &role.UserCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Load role permissions
	permissions, err := s.getRolePermissions(ctx, role.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	role.Permissions = permissions

	return role, nil
}

// CreateRole creates a new role
func (s *RoleManagementService) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	query := `
		INSERT INTO roles (name, display_name, description, is_system_role, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING role_id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		role.Name, role.DisplayName, role.Description, role.IsSystemRole, role.TenantID,
	).Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// UpdateRole updates an existing role
func (s *RoleManagementService) UpdateRole(ctx context.Context, roleID uuid.UUID, updates map[string]interface{}) (*models.Role, error) {
	if len(updates) == 0 {
		return s.getRoleByIDSimple(ctx, roleID)
	}

	// Prevent updating system roles
	isSystemRole, err := s.isSystemRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if isSystemRole {
		return nil, fmt.Errorf("cannot update system role")
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 0

	for field, value := range updates {
		if field == "is_system_role" {
			continue // Don't allow updating system role flag
		}
		argCount++
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argCount))
		args = append(args, value)
	}

	if len(setParts) == 0 {
		return s.getRoleByIDSimple(ctx, roleID)
	}

	argCount++
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	argCount++
	args = append(args, roleID)

	query := fmt.Sprintf(`
		UPDATE roles SET %s WHERE role_id = $%d
	`, strings.Join(setParts, ", "), argCount)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return s.getRoleByIDSimple(ctx, roleID)
}

// DeleteRole deletes a role
func (s *RoleManagementService) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	// Prevent deleting system roles
	isSystemRole, err := s.isSystemRole(ctx, roleID)
	if err != nil {
		return err
	}
	if isSystemRole {
		return fmt.Errorf("cannot delete system role")
	}

	// Check if role has users
	userCount, err := s.getRoleUserCount(ctx, roleID)
	if err != nil {
		return err
	}
	if userCount > 0 {
		return fmt.Errorf("cannot delete role with assigned users")
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete role permissions
	_, err = tx.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role permissions: %w", err)
	}

	// Delete role
	result, err := tx.ExecContext(ctx, "DELETE FROM roles WHERE role_id = $1", roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role not found")
	}

	return tx.Commit()
}

// GetPermissions retrieves permissions with filtering
func (s *RoleManagementService) GetPermissions(ctx context.Context, filter *PermissionFilter) ([]models.Permission, int, error) {
	// Build the WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if filter.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm)
	}

	if filter.Resource != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND resource = $%d", argCount)
		args = append(args, filter.Resource)
	}

	if filter.Action != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, filter.Action)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM permissions %s", whereClause)
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get permission count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY resource, action"
	if filter.SortBy != "" {
		sortOrder := "ASC"
		if strings.ToUpper(filter.SortOrder) == "DESC" {
			sortOrder = "DESC"
		}
		orderBy = fmt.Sprintf("ORDER BY %s %s", filter.SortBy, sortOrder)
	}

	// Build LIMIT and OFFSET
	limit := 100
	if filter.Limit > 0 && filter.Limit <= 500 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Offset >= 0 {
		offset = filter.Offset
	}

	argCount++
	limitClause := fmt.Sprintf("LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	offsetClause := fmt.Sprintf("OFFSET $%d", argCount)
	args = append(args, offset)

	// Main query
	query := fmt.Sprintf(`
		SELECT permission_id, name, resource, action, description, created_at
		FROM permissions %s %s %s %s
	`, whereClause, orderBy, limitClause, offsetClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, totalCount, nil
}

// CreatePermission creates a new permission
func (s *RoleManagementService) CreatePermission(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	query := `
		INSERT INTO permissions (name, resource, action, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := s.db.QueryRowContext(ctx, query,
		permission.Name, permission.Resource, permission.Action, permission.Description,
	).Scan(&permission.ID, &permission.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return permission, nil
}

// AssignRolePermissions assigns permissions to a role (replaces existing)
func (s *RoleManagementService) AssignRolePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	// Prevent modifying system roles
	isSystemRole, err := s.isSystemRole(ctx, roleID)
	if err != nil {
		return err
	}
	if isSystemRole {
		return fmt.Errorf("cannot modify system role permissions")
	}

	// Get role's tenant_id
	var tenantID uuid.UUID
	err = s.db.QueryRowContext(ctx, "SELECT tenant_id FROM roles WHERE role_id = $1", roleID).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("failed to get role tenant: %w", err)
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove existing permissions
	_, err = tx.ExecContext(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return fmt.Errorf("failed to remove existing permissions: %w", err)
	}

	// Add new permissions
	for _, permissionID := range permissionIDs {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO role_permissions (tenant_id, role_id, permission_id) VALUES ($1, $2, $3)",
			tenantID, roleID, permissionID)
		if err != nil {
			return fmt.Errorf("failed to assign permission: %w", err)
		}
	}

	return tx.Commit()
}

// AddRolePermission adds a single permission to a role
func (s *RoleManagementService) AddRolePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	// Prevent modifying system roles
	isSystemRole, err := s.isSystemRole(ctx, roleID)
	if err != nil {
		return err
	}
	if isSystemRole {
		return fmt.Errorf("cannot modify system role permissions")
	}

	// Get role's tenant_id
	var tenantID uuid.UUID
	err = s.db.QueryRowContext(ctx, "SELECT tenant_id FROM roles WHERE role_id = $1", roleID).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("failed to get role tenant: %w", err)
	}

	query := `
		INSERT INTO role_permissions (tenant_id, role_id, permission_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, role_id, permission_id) DO NOTHING
	`

	_, err = s.db.ExecContext(ctx, query, tenantID, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to add role permission: %w", err)
	}

	return nil
}

// RemoveRolePermission removes a permission from a role
func (s *RoleManagementService) RemoveRolePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	// Prevent modifying system roles
	isSystemRole, err := s.isSystemRole(ctx, roleID)
	if err != nil {
		return err
	}
	if isSystemRole {
		return fmt.Errorf("cannot modify system role permissions")
	}

	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`

	result, err := s.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove role permission: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role permission not found")
	}

	return nil
}

// GetRoleUsers retrieves users assigned to a role
func (s *RoleManagementService) GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]models.User, error) {
	query := `
		SELECT u.user_id, u.sub, u.username, u.email, u.first_name, u.last_name, 
		       u.display_name, u.is_active, u.is_email_verified, u.last_login_at,
		       u.created_at, u.updated_at
		FROM users u
		JOIN user_roles ur ON u.user_id = ur.user_id
		WHERE ur.role_id = $1 AND u.is_active = true
		ORDER BY u.username
	`

	rows, err := s.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query role users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Sub, &user.Username, &user.Email,
			&user.FirstName, &user.LastName, &user.DisplayName,
			&user.IsActive, &user.IsEmailVerified, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetResourcesAndActions returns available resources and actions for permission management
func (s *RoleManagementService) GetResourcesAndActions(ctx context.Context) (map[string][]string, error) {
	query := `SELECT DISTINCT resource, action FROM permissions ORDER BY resource, action`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query resources and actions: %w", err)
	}
	defer rows.Close()

	resourceActions := make(map[string][]string)
	for rows.Next() {
		var resource, action string
		err := rows.Scan(&resource, &action)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resource and action: %w", err)
		}

		if _, exists := resourceActions[resource]; !exists {
			resourceActions[resource] = []string{}
		}
		resourceActions[resource] = append(resourceActions[resource], action)
	}

	return resourceActions, nil
}

// Helper methods

func (s *RoleManagementService) getRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	query := `
		SELECT p.permission_id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.permission_id = rp.permission_id
		WHERE rp.role_id = $1
		ORDER BY p.resource, p.action
	`

	rows, err := s.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description, &permission.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func (s *RoleManagementService) getRoleByIDSimple(ctx context.Context, roleID uuid.UUID) (*models.Role, error) {
	query := `
		SELECT role_id, name, display_name, description, is_system_role, created_at, updated_at
		FROM roles WHERE role_id = $1
	`

	role := &models.Role{}
	err := s.db.QueryRowContext(ctx, query, roleID).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSystemRole, &role.CreatedAt, &role.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

func (s *RoleManagementService) isSystemRole(ctx context.Context, roleID uuid.UUID) (bool, error) {
	var isSystemRole bool
	err := s.db.QueryRowContext(ctx,
		"SELECT is_system_role FROM roles WHERE role_id = $1", roleID).Scan(&isSystemRole)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("role not found")
		}
		return false, fmt.Errorf("failed to check if system role: %w", err)
	}

	return isSystemRole, nil
}

func (s *RoleManagementService) getRoleUserCount(ctx context.Context, roleID uuid.UUID) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_roles ur 
		JOIN users u ON ur.user_id = u.user_id 
		WHERE ur.role_id = $1 AND u.is_active = true
	`, roleID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get role user count: %w", err)
	}

	return count, nil
}
