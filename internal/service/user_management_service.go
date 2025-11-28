package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// UserManagementService handles user CRUD operations and management
type UserManagementService struct {
	db *sql.DB
}

// NewUserManagementService creates a new user management service
func NewUserManagementService(db *sql.DB) *UserManagementService {
	return &UserManagementService{
		db: db,
	}
}

// UserFilter represents filtering options for user queries
type UserFilter struct {
	Search    string
	IsActive  *bool
	RoleID    *int64
	TenantID  *int64
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// UserInvite represents a user invitation
type UserInvite struct {
	ID          uuid.UUID   `json:"id"`
	Email       string      `json:"email"`
	FirstName   *string     `json:"first_name,omitempty"`
	LastName    *string     `json:"last_name,omitempty"`
	RoleIDs     []uuid.UUID `json:"role_ids"`
	InvitedBy   uuid.UUID   `json:"invited_by"`
	InviteToken string      `json:"invite_token"`
	ExpiresAt   time.Time   `json:"expires_at"`
	AcceptedAt  *time.Time  `json:"accepted_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	IsExpired   bool        `json:"is_expired"`
	TenantID    uuid.UUID   `json:"tenant_id,omitempty"`
}

// GetUsers retrieves users with filtering and pagination
func (s *UserManagementService) GetUsers(ctx context.Context, filter *UserFilter) ([]*models.User, int, error) {
	// Build the WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 0

	if filter.Search != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)",
			argCount, argCount, argCount, argCount)
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm)
	}

	if filter.IsActive != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, *filter.IsActive)
	}

	if filter.TenantID != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
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

	// Main query
	query := fmt.Sprintf(`
		SELECT user_id, sub, username, email, first_name, last_name, display_name,
		       is_active, is_email_verified, last_login_at, created_at, updated_at, tenant_id
		FROM users %s %s %s %s
	`, whereClause, orderBy, limitClause, offsetClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID, &user.Sub, &user.Username, &user.Email,
			&user.FirstName, &user.LastName, &user.DisplayName,
			&user.IsActive, &user.IsEmailVerified, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt, &user.TenantID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		// Load user roles
		roles, err := s.getUserRoles(ctx, user.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get user roles: %w", err)
		}
		user.Roles = roles

		users = append(users, user)
	}

	return users, totalCount, nil
}

// GetUserByUsername retrieves a user by username
func (s *UserManagementService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT user_id, sub, username, email, password_hash, first_name, last_name, display_name,
		       is_active, is_email_verified, last_login_at, created_at, updated_at, tenant_id
		FROM users 
		WHERE username = $1 AND is_active = true
	`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Sub, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.DisplayName,
		&user.IsActive, &user.IsEmailVerified, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.TenantID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Load roles and permissions
	roles, err := s.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	user.Roles = roles

	permissions, err := s.getUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	user.Permissions = permissions

	return user, nil
}

// GetUserByID retrieves a user by ID with roles and permissions
func (s *UserManagementService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT user_id, sub, username, email, password_hash, first_name, last_name, display_name,
		       is_active, is_email_verified, last_login_at, created_at, updated_at, tenant_id
		FROM users WHERE user_id = $1
	`

	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Sub, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.DisplayName,
		&user.IsActive, &user.IsEmailVerified, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.TenantID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Load user roles
	roles, err := s.getUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	user.Roles = roles

	// Load user permissions
	permissions, err := s.getUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}
	user.Permissions = permissions

	return user, nil
}

// CreateUser creates a new user
func (s *UserManagementService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
		INSERT INTO users (tenant_id, sub, username, email, password_hash, first_name, last_name, display_name, is_active, is_email_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING user_id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		user.TenantID, user.Sub, user.Username, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.DisplayName, user.IsActive, user.IsEmailVerified,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateUser updates an existing user
func (s *UserManagementService) UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) (*models.User, error) {
	if len(updates) == 0 {
		return s.GetUserByID(ctx, userID)
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argCount := 0

	for field, value := range updates {
		argCount++
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argCount))
		args = append(args, value)
	}

	argCount++
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())

	argCount++
	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE users SET %s WHERE user_id = $%d
	`, strings.Join(setParts, ", "), argCount)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.GetUserByID(ctx, userID)
}

// DeleteUser soft deletes a user
func (s *UserManagementService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET is_active = false, updated_at = NOW() WHERE user_id = $1`

	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// InviteUser creates a user invitation
func (s *UserManagementService) InviteUser(ctx context.Context, email string, firstName, lastName *string, roleIDs []uuid.UUID, invitedBy uuid.UUID, tenantID uuid.UUID) (*UserInvite, error) {
	// Generate invite token
	token, err := s.generateInviteToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}

	// Set expiration (7 days from now)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	query := `
		INSERT INTO user_invites_uuid (email, first_name, last_name, invited_by, invite_token, expires_at, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING invite_id, created_at
	`

	invite := &UserInvite{
		Email:       email,
		FirstName:   firstName,
		LastName:    lastName,
		RoleIDs:     roleIDs,
		InvitedBy:   invitedBy,
		InviteToken: token,
		ExpiresAt:   expiresAt,
		TenantID:    tenantID,
	}

	err = s.db.QueryRowContext(ctx, query,
		invite.Email, invite.FirstName, invite.LastName,
		invite.InvitedBy, invite.InviteToken, invite.ExpiresAt, invite.TenantID,
	).Scan(&invite.ID, &invite.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user invite: %w", err)
	}

	// Associate roles with the invite
	if len(roleIDs) > 0 {
		err = s.associateInviteRoles(ctx, invite.ID, roleIDs, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to associate roles with invite: %w", err)
		}
	}

	return invite, nil
}

// GetUserInvites retrieves pending user invitations
func (s *UserManagementService) GetUserInvites(ctx context.Context, includeExpired bool) ([]*UserInvite, error) {
	whereClause := "WHERE accepted_at IS NULL"
	if !includeExpired {
		whereClause += " AND expires_at > NOW()"
	}

	query := fmt.Sprintf(`
		SELECT invite_id, email, first_name, last_name, invited_by, invite_token,
		       expires_at, accepted_at, created_at, expires_at < NOW() as is_expired, tenant_id
		FROM user_invites_uuid %s
		ORDER BY created_at DESC
	`, whereClause)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query user invites: %w", err)
	}
	defer rows.Close()

	var invites []*UserInvite
	for rows.Next() {
		invite := &UserInvite{}
		err := rows.Scan(
			&invite.ID, &invite.Email, &invite.FirstName, &invite.LastName,
			&invite.InvitedBy, &invite.InviteToken, &invite.ExpiresAt,
			&invite.AcceptedAt, &invite.CreatedAt, &invite.IsExpired, &invite.TenantID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user invite: %w", err)
		}

		// Load associated roles
		roleIDs, err := s.getInviteRoles(ctx, invite.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get invite roles: %w", err)
		}
		invite.RoleIDs = roleIDs

		invites = append(invites, invite)
	}

	return invites, nil
}

// AcceptInvite accepts a user invitation and creates the user
func (s *UserManagementService) AcceptInvite(ctx context.Context, token, username, password string) (*models.User, error) {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get invite details
	var invite UserInvite
	query := `
		SELECT invite_id, email, first_name, last_name, expires_at, accepted_at, tenant_id
		FROM user_invites_uuid 
		WHERE invite_token = $1 AND accepted_at IS NULL AND expires_at > NOW()
	`

	err = tx.QueryRowContext(ctx, query, token).Scan(
		&invite.ID, &invite.Email, &invite.FirstName, &invite.LastName,
		&invite.ExpiresAt, &invite.AcceptedAt, &invite.TenantID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired invite token")
		}
		return nil, fmt.Errorf("failed to get invite: %w", err)
	}

	// Create the user
	user := &models.User{
		Username:        username,
		Email:           invite.Email,
		FirstName:       invite.FirstName,
		LastName:        invite.LastName,
		IsActive:        true,
		IsEmailVerified: true,
		TenantID:        invite.TenantID,
	}

	query = `
		INSERT INTO users (username, email, first_name, last_name, is_active, is_email_verified, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING user_id, created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, query,
		user.Username, user.Email, user.FirstName, user.LastName,
		user.IsActive, user.IsEmailVerified, user.TenantID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Get invite roles and assign to user
	roleIDs, err := s.getInviteRolesInTx(ctx, tx, invite.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invite roles: %w", err)
	}

	for _, roleID := range roleIDs {
		err = s.assignUserRoleInTx(ctx, tx, user.ID, roleID, user.TenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to assign user role: %w", err)
		}
	}

	// Mark invite as accepted
	_, err = tx.ExecContext(ctx,
		`UPDATE user_invites_uuid SET accepted_at = NOW() WHERE invite_id = $1`,
		invite.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark invite as accepted: %w", err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.GetUserByID(ctx, user.ID)
}

// AssignUserRole assigns a role to a user
func (s *UserManagementService) AssignUserRole(ctx context.Context, userID, roleID, tenantID uuid.UUID) error {
	return s.assignUserRoleInTx(ctx, s.db, userID, roleID, tenantID)
}

// RemoveUserRole removes a role from a user
func (s *UserManagementService) RemoveUserRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`

	result, err := s.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove user role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user role assignment not found")
	}

	return nil
}

// Helper methods
func (s *UserManagementService) getUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	query := `
		SELECT r.role_id, r.name, r.display_name, r.description, r.is_system_role, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON r.role_id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.IsSystemRole, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (s *UserManagementService) getUserPermissions(ctx context.Context, userID uuid.UUID) ([]models.Permission, error) {
	query := `
		SELECT DISTINCT p.permission_id, p.name, p.resource, p.action, p.description, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.permission_id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.resource, p.action
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
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

func (s *UserManagementService) generateInviteToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *UserManagementService) associateInviteRoles(ctx context.Context, inviteID uuid.UUID, roleIDs []uuid.UUID, tenantID uuid.UUID) error {
	for _, roleID := range roleIDs {
		query := `INSERT INTO user_invite_roles_uuid (invite_id, role_id, tenant_id) VALUES ($1, $2, $3)`
		_, err := s.db.ExecContext(ctx, query, inviteID, roleID, tenantID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *UserManagementService) getInviteRoles(ctx context.Context, inviteID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT role_id FROM user_invite_roles_uuid WHERE invite_id = $1`

	rows, err := s.db.QueryContext(ctx, query, inviteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleIDs []uuid.UUID
	for rows.Next() {
		var roleID uuid.UUID
		err := rows.Scan(&roleID)
		if err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}

	return roleIDs, nil
}

func (s *UserManagementService) getInviteRolesInTx(ctx context.Context, tx *sql.Tx, inviteID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT role_id FROM user_invite_roles_uuid WHERE invite_id = $1`

	rows, err := tx.QueryContext(ctx, query, inviteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roleIDs []uuid.UUID
	for rows.Next() {
		var roleID uuid.UUID
		err := rows.Scan(&roleID)
		if err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, roleID)
	}

	return roleIDs, nil
}

func (s *UserManagementService) assignUserRoleInTx(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}, userID, roleID, tenantID uuid.UUID) error {
	query := `
		INSERT INTO user_roles (tenant_id, user_id, role_id, assigned_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING
	`

	_, err := tx.ExecContext(ctx, query, tenantID, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to assign user role: %w", err)
	}

	return nil
}

// GetRoles returns a list of all roles for user management purposes
func (s *UserManagementService) GetRoles(ctx context.Context) ([]models.Role, error) {
	query := `
		SELECT id, name, display_name, description, is_system_role, created_at, updated_at
		FROM roles 
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.IsSystemRole, &role.CreatedAt, &role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return roles, nil
}
