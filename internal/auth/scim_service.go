package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
)

// SCIMService handles SCIM group mapping and user provisioning
type SCIMService struct {
	db       *sql.DB
	logger   *logger.Logger
	settings *SCIMSettings
}

// SCIMSettings contains SCIM configuration
type SCIMSettings struct {
	// External IdP groups to internal roles mapping
	GroupRoleMapping map[string][]string `json:"group_role_mapping"`

	// Enabled SCIM features
	EnableUserProvisioning bool `json:"enable_user_provisioning"`
	EnableGroupMapping     bool `json:"enable_group_mapping"`

	// External group attributes
	GroupClaimName string `json:"group_claim_name"` // e.g., "groups", "memberOf"
	GroupFormat    string `json:"group_format"`     // e.g., "dn", "name"
}

// DefaultSCIMSettings provides default SCIM configuration
func DefaultSCIMSettings() *SCIMSettings {
	return &SCIMSettings{
		GroupRoleMapping: map[string][]string{
			"SecureStore-Admins":     {"admin", "artifact_admin"},
			"SecureStore-Auditors":   {"auditor"},
			"SecureStore-Developers": {"developer", "artifact_read"},
			"SecureStore-QA":         {"tester", "artifact_read"},
			"CN=SecureStore-Admins,OU=Groups,DC=company,DC=com":     {"admin", "artifact_admin"},
			"CN=SecureStore-Developers,OU=Groups,DC=company,DC=com": {"developer", "artifact_read"},
		},
		EnableUserProvisioning: true,
		EnableGroupMapping:     true,
		GroupClaimName:         "groups",
		GroupFormat:            "name",
	}
}

// NewSCIMService creates a new SCIM service
func NewSCIMService(db *sql.DB, logger *logger.Logger) *SCIMService {
	return &SCIMService{
		db:       db,
		logger:   logger,
		settings: DefaultSCIMSettings(),
	}
}

// GetSettings returns the current SCIM settings
func (s *SCIMService) GetSettings() *SCIMSettings {
	return s.settings
}

// LoadSettings loads SCIM settings from database
func (s *SCIMService) LoadSettings() error {
	query := `
		SELECT settings_value 
		FROM scim_settings 
		WHERE settings_key = 'group_mapping' 
		AND is_active = true
		ORDER BY updated_at DESC 
		LIMIT 1
	`

	var settingsJSON string
	err := s.db.QueryRow(query).Scan(&settingsJSON)
	if err != nil {
		// Use defaults if no settings found
		s.logger.Info("Using default SCIM settings")
		return nil
	}

	return json.Unmarshal([]byte(settingsJSON), s.settings)
}

// SaveSettings saves SCIM settings to database
func (s *SCIMService) SaveSettings(settings *SCIMSettings) error {
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO scim_settings (settings_key, settings_value, updated_by)
		VALUES ('group_mapping', $1, 'system')
		ON CONFLICT (settings_key) 
		DO UPDATE SET 
			settings_value = EXCLUDED.settings_value,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = EXCLUDED.updated_by
	`

	_, err = s.db.Exec(query, string(settingsJSON))
	if err != nil {
		return fmt.Errorf("failed to save SCIM settings: %w", err)
	}

	s.settings = settings
	return nil
}

// ProcessUserClaims processes user claims from IdP and maps groups to roles
func (s *SCIMService) ProcessUserClaims(claims map[string]interface{}, userID uuid.UUID) (*models.UserGroupInfo, error) {
	if !s.settings.EnableGroupMapping {
		return &models.UserGroupInfo{
			UserID: userID,
			Roles:  []string{"user"}, // Default role
		}, nil
	}

	// Extract groups from claims
	groups, err := s.extractGroupsFromClaims(claims)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to extract groups from claims for user %s", userID.String()), err)
		groups = []string{} // Continue with empty groups
	}

	// Map groups to roles
	roles := s.mapGroupsToRoles(groups)

	// Ensure at least 'user' role is assigned
	if len(roles) == 0 {
		roles = []string{"user"}
	}

	userGroupInfo := &models.UserGroupInfo{
		UserID: userID,
		Groups: groups,
		Roles:  roles,
	}

	// Save group mapping to database
	err = s.saveUserGroupMapping(userGroupInfo)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to save user group mapping for user %s", userID.String()), err)
		// Continue without saving - don't fail authentication
	}

	return userGroupInfo, nil
}

// extractGroupsFromClaims extracts group information from OIDC claims
func (s *SCIMService) extractGroupsFromClaims(claims map[string]interface{}) ([]string, error) {
	groupClaim, exists := claims[s.settings.GroupClaimName]
	if !exists {
		return []string{}, nil
	}

	switch groups := groupClaim.(type) {
	case []interface{}:
		result := make([]string, 0, len(groups))
		for _, group := range groups {
			if groupStr, ok := group.(string); ok {
				result = append(result, s.normalizeGroupName(groupStr))
			}
		}
		return result, nil

	case []string:
		result := make([]string, 0, len(groups))
		for _, group := range groups {
			result = append(result, s.normalizeGroupName(group))
		}
		return result, nil

	case string:
		// Single group or comma-separated groups
		if strings.Contains(groups, ",") {
			groupList := strings.Split(groups, ",")
			result := make([]string, 0, len(groupList))
			for _, group := range groupList {
				result = append(result, s.normalizeGroupName(strings.TrimSpace(group)))
			}
			return result, nil
		}
		return []string{s.normalizeGroupName(groups)}, nil

	default:
		return nil, fmt.Errorf("unsupported group claim format: %T", groupClaim)
	}
}

// normalizeGroupName normalizes group names based on format settings
func (s *SCIMService) normalizeGroupName(groupName string) string {
	switch s.settings.GroupFormat {
	case "dn":
		// Extract CN from Distinguished Name
		if strings.HasPrefix(groupName, "CN=") {
			parts := strings.Split(groupName, ",")
			if len(parts) > 0 {
				cnPart := parts[0]
				if strings.HasPrefix(cnPart, "CN=") {
					return cnPart[3:] // Remove "CN=" prefix
				}
			}
		}
		return groupName

	case "name":
		fallthrough
	default:
		return groupName
	}
}

// mapGroupsToRoles maps external groups to internal roles
func (s *SCIMService) mapGroupsToRoles(groups []string) []string {
	roleSet := make(map[string]bool)

	for _, group := range groups {
		if roles, exists := s.settings.GroupRoleMapping[group]; exists {
			for _, role := range roles {
				roleSet[role] = true
			}
		}
	}

	roles := make([]string, 0, len(roleSet))
	for role := range roleSet {
		roles = append(roles, role)
	}

	return roles
}

// saveUserGroupMapping saves user group mapping to database
func (s *SCIMService) saveUserGroupMapping(userGroupInfo *models.UserGroupInfo) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing mappings
	_, err = tx.Exec("DELETE FROM scim_user_groups WHERE user_id = $1", userGroupInfo.UserID)
	if err != nil {
		return fmt.Errorf("failed to delete existing group mappings: %w", err)
	}

	// Insert new group mappings
	for _, group := range userGroupInfo.Groups {
		_, err = tx.Exec(`
			INSERT INTO scim_user_groups (user_id, group_name, mapped_at)
			VALUES ($1, $2, CURRENT_TIMESTAMP)
		`, userGroupInfo.UserID, group)
		if err != nil {
			return fmt.Errorf("failed to insert group mapping: %w", err)
		}
	}

	// Update user roles in user_roles table
	_, err = tx.Exec("DELETE FROM user_roles WHERE user_id = $1", userGroupInfo.UserID)
	if err != nil {
		return fmt.Errorf("failed to delete existing user roles: %w", err)
	}

	for _, role := range userGroupInfo.Roles {
		_, err = tx.Exec(`
			INSERT INTO user_roles (user_id, role_name, assigned_at, assigned_by)
			VALUES ($1, $2, CURRENT_TIMESTAMP, 'scim_auto')
		`, userGroupInfo.UserID, role)
		if err != nil {
			return fmt.Errorf("failed to insert user role: %w", err)
		}
	}

	return tx.Commit()
}

// GetUserGroups retrieves current group mappings for a user
func (s *SCIMService) GetUserGroups(userID int64) ([]string, error) {
	query := `
		SELECT group_name 
		FROM scim_user_groups 
		WHERE user_id = $1
		ORDER BY group_name
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user groups: %w", err)
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var group string
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// GetGroupStatistics returns SCIM group mapping statistics
func (s *SCIMService) GetGroupStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total users with group mappings
	var totalUsers int64
	err := s.db.QueryRow("SELECT COUNT(DISTINCT user_id) FROM scim_user_groups").Scan(&totalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get total users: %w", err)
	}
	stats["total_users_with_groups"] = totalUsers

	// Group distribution
	groupQuery := `
		SELECT group_name, COUNT(*) as user_count
		FROM scim_user_groups
		GROUP BY group_name
		ORDER BY user_count DESC
	`

	rows, err := s.db.Query(groupQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query group distribution: %w", err)
	}
	defer rows.Close()

	groupDistribution := make(map[string]int64)
	for rows.Next() {
		var groupName string
		var userCount int64
		if err := rows.Scan(&groupName, &userCount); err != nil {
			return nil, fmt.Errorf("failed to scan group distribution: %w", err)
		}
		groupDistribution[groupName] = userCount
	}
	stats["group_distribution"] = groupDistribution

	// Role distribution
	roleQuery := `
		SELECT role_name, COUNT(*) as user_count
		FROM user_roles
		WHERE assigned_by = 'scim_auto'
		GROUP BY role_name
		ORDER BY user_count DESC
	`

	rows, err = s.db.Query(roleQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query role distribution: %w", err)
	}
	defer rows.Close()

	roleDistribution := make(map[string]int64)
	for rows.Next() {
		var roleName string
		var userCount int64
		if err := rows.Scan(&roleName, &userCount); err != nil {
			return nil, fmt.Errorf("failed to scan role distribution: %w", err)
		}
		roleDistribution[roleName] = userCount
	}
	stats["role_distribution"] = roleDistribution

	return stats, nil
}

// ValidateGroupMapping validates the group to role mapping configuration
func (s *SCIMService) ValidateGroupMapping(mapping map[string][]string) error {
	validRoles := []string{"admin", "auditor", "developer", "tester", "user", "artifact_admin", "artifact_read"}
	roleSet := make(map[string]bool)
	for _, role := range validRoles {
		roleSet[role] = true
	}

	for group, roles := range mapping {
		if group == "" {
			return fmt.Errorf("empty group name not allowed")
		}

		if len(roles) == 0 {
			return fmt.Errorf("group %s has no roles assigned", group)
		}

		for _, role := range roles {
			if !roleSet[role] {
				return fmt.Errorf("invalid role %s for group %s", role, group)
			}
		}
	}

	return nil
}
