package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// UserSessionService handles user session tracking
type UserSessionService struct {
	db           *sql.DB
	logger       *log.Logger
	auditService *AuditLogService
}

// UserSession represents an active user session
type UserSession struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	ExpiresAt    time.Time              `json:"expires_at"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SessionStats represents session statistics
type SessionStats struct {
	ActiveSessions   int64   `json:"active_sessions"`
	TotalSessions    int64   `json:"total_sessions"`
	UniqueUsers      int64   `json:"unique_users"`
	AverageSession   float64 `json:"average_session_minutes"`
	SessionsToday    int64   `json:"sessions_today"`
	ActiveUsersToday int64   `json:"active_users_today"`
}

// NewUserSessionService creates a new user session service
func NewUserSessionService(db *sql.DB, logger *log.Logger, auditService *AuditLogService) *UserSessionService {
	service := &UserSessionService{
		db:           db,
		logger:       logger,
		auditService: auditService,
	}

	// Start cleanup routine
	go service.startCleanupRoutine()

	return service
}

// CreateSession creates a new user session
func (s *UserSessionService) CreateSession(ctx context.Context, tenantID, userID, ipAddress, userAgent string) (*UserSession, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	session := &UserSession{
		ID:           sessionID,
		UserID:       userID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour), // 24 hour expiry
		IsActive:     true,
	}

	// Insert into database
	query := `
		INSERT INTO user_sessions (
			id, user_id, ip_address, user_agent, created_at, 
			last_activity, expires_at, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = s.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.IPAddress, session.UserAgent,
		session.CreatedAt, session.LastActivity, session.ExpiresAt, session.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Log session creation
	if s.auditService != nil {
		s.auditService.LogUserSession(ctx, tenantID, userID, "login", ipAddress, userAgent, sessionID, map[string]interface{}{
			"session_created": true,
			"login_time":      time.Now().Unix(),
		})
	}

	s.logger.Printf("Created session %s for user %s", sessionID, userID)
	return session, nil
}

// GetSession retrieves a session by ID
func (s *UserSessionService) GetSession(ctx context.Context, sessionID string) (*UserSession, error) {
	query := `
		SELECT id, user_id, ip_address, user_agent, created_at, 
			   last_activity, expires_at, is_active
		FROM user_sessions 
		WHERE id = $1`

	var session UserSession
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.IPAddress, &session.UserAgent,
		&session.CreatedAt, &session.LastActivity, &session.ExpiresAt, &session.IsActive,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) || !session.IsActive {
		return nil, nil
	}

	return &session, nil
}

// UpdateActivity updates the last activity time for a session
func (s *UserSessionService) UpdateActivity(ctx context.Context, sessionID, userID, ipAddress, userAgent string) error {
	query := `
		UPDATE user_sessions 
		SET last_activity = $1 
		WHERE id = $2 AND is_active = true AND expires_at > $1`

	result, err := s.db.ExecContext(ctx, query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		// Session might be expired or doesn't exist, create a new one
		// Query user to get tenant_id
		var tenantID string
		userQuery := `SELECT tenant_id FROM users WHERE id = $1`
		err = s.db.QueryRowContext(ctx, userQuery, userID).Scan(&tenantID)
		if err != nil && err != sql.ErrNoRows {
			s.logger.Printf("Warning: failed to get tenant_id for user %s: %v", userID, err)
			tenantID = "" // Use empty tenant_id if we can't query it
		}
		_, err = s.CreateSession(ctx, tenantID, userID, ipAddress, userAgent)
		return err
	}

	return err
}

// EndSession ends a user session
func (s *UserSessionService) EndSession(ctx context.Context, sessionID, userID, ipAddress, userAgent string) error {
	query := `
		UPDATE user_sessions 
		SET is_active = false, last_activity = $1 
		WHERE id = $2`

	_, err := s.db.ExecContext(ctx, query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Log session end - query user to get tenant_id
	if s.auditService != nil {
		var tenantID string
		userQuery := `SELECT tenant_id FROM users WHERE id = $1`
		qErr := s.db.QueryRowContext(ctx, userQuery, userID).Scan(&tenantID)
		if qErr != nil && qErr != sql.ErrNoRows {
			s.logger.Printf("Warning: failed to get tenant_id for user %s: %v", userID, qErr)
			tenantID = "" // Use empty tenant_id if we can't query it
		}
		s.auditService.LogUserSession(ctx, tenantID, userID, "logout", ipAddress, userAgent, sessionID, map[string]interface{}{
			"session_ended": true,
			"logout_time":   time.Now().Unix(),
		})
	}

	s.logger.Printf("Ended session %s for user %s", sessionID, userID)
	return nil
}

// GetActiveSessions returns all active sessions for a user
func (s *UserSessionService) GetActiveSessions(ctx context.Context, userID string) ([]*UserSession, error) {
	query := `
		SELECT id, user_id, ip_address, user_agent, created_at, 
			   last_activity, expires_at, is_active
		FROM user_sessions 
		WHERE user_id = $1 AND is_active = true AND expires_at > $2
		ORDER BY last_activity DESC`

	rows, err := s.db.QueryContext(ctx, query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*UserSession
	for rows.Next() {
		var session UserSession
		err := rows.Scan(
			&session.ID, &session.UserID, &session.IPAddress, &session.UserAgent,
			&session.CreatedAt, &session.LastActivity, &session.ExpiresAt, &session.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// GetSessionStats returns session statistics
func (s *UserSessionService) GetSessionStats(ctx context.Context) (*SessionStats, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	query := `
		SELECT 
			COUNT(CASE WHEN is_active = true AND expires_at > $1 THEN 1 END) as active_sessions,
			COUNT(*) as total_sessions,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(AVG(EXTRACT(EPOCH FROM (COALESCE(last_activity, created_at) - created_at))/60), 0) as avg_session_minutes,
			COUNT(CASE WHEN created_at >= $2 THEN 1 END) as sessions_today,
			COUNT(DISTINCT CASE WHEN created_at >= $2 THEN user_id END) as active_users_today
		FROM user_sessions`

	var stats SessionStats
	err := s.db.QueryRowContext(ctx, query, now, today).Scan(
		&stats.ActiveSessions, &stats.TotalSessions, &stats.UniqueUsers,
		&stats.AverageSession, &stats.SessionsToday, &stats.ActiveUsersToday,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}

	return &stats, nil
}

// CleanupExpiredSessions removes expired sessions
func (s *UserSessionService) CleanupExpiredSessions(ctx context.Context) error {
	query := `
		UPDATE user_sessions 
		SET is_active = false 
		WHERE expires_at <= $1 AND is_active = true`

	result, err := s.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		s.logger.Printf("Cleaned up %d expired sessions", rowsAffected)
	}

	return nil
}

// startCleanupRoutine starts a background routine to cleanup expired sessions
func (s *UserSessionService) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := s.CleanupExpiredSessions(ctx)
			if err != nil {
				s.logger.Printf("Failed to cleanup expired sessions: %v", err)
			}
			cancel()
		}
	}
}

// generateSessionID generates a random UUID session ID
func generateSessionID() (string, error) {
	id := uuid.New()
	return id.String(), nil
}

// GetOrCreateSession gets an existing session or creates a new one
func (s *UserSessionService) GetOrCreateSession(ctx context.Context, sessionID, userID, ipAddress, userAgent string) (*UserSession, error) {
	if sessionID != "" {
		// Try to get existing session
		session, err := s.GetSession(ctx, sessionID)
		if err == nil && session != nil {
			// Update activity
			s.UpdateActivity(ctx, sessionID, userID, ipAddress, userAgent)
			return session, nil
		}
	}

	// Create new session - query user to get tenant_id
	var tenantID string
	userQuery := `SELECT tenant_id FROM users WHERE id = $1`
	err := s.db.QueryRowContext(ctx, userQuery, userID).Scan(&tenantID)
	if err != nil && err != sql.ErrNoRows {
		s.logger.Printf("Warning: failed to get tenant_id for user %s: %v", userID, err)
		tenantID = "" // Use empty tenant_id if we can't query it
	}

	// Create new session
	return s.CreateSession(ctx, tenantID, userID, ipAddress, userAgent)
}
