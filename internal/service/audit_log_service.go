package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
)

// AuditLogService handles all audit logging functionality
type AuditLogService struct {
	db     *sql.DB
	logger *log.Logger
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID           uuid.UUID              `json:"id"`
	TenantID     string                 `json:"tenant_id"` // UUID as string for multi-tenancy
	EventType    string                 `json:"event_type"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	UserID       *string                `json:"user_id,omitempty"`
	Username     *string                `json:"username,omitempty"`
	Action       string                 `json:"action"`
	OldValue     string                 `json:"old_value,omitempty"`
	NewValue     string                 `json:"new_value,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditLogFilters for querying audit logs
type AuditLogFilters struct {
	TenantID     string    `json:"tenant_id,omitempty"`
	EventType    string    `json:"event_type,omitempty"`
	ResourceType string    `json:"resource_type,omitempty"`
	UserID       string    `json:"user_id,omitempty"`
	Action       string    `json:"action,omitempty"`
	StartTime    time.Time `json:"start_time,omitempty"`
	EndTime      time.Time `json:"end_time,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Offset       int       `json:"offset,omitempty"`
	Success      *bool     `json:"success,omitempty"`
}

// AuditStats represents audit statistics
type AuditStats struct {
	TotalLogs      int64            `json:"total_logs"`
	SuccessfulLogs int64            `json:"successful_logs"`
	FailedLogs     int64            `json:"failed_logs"`
	SuccessRate    float64          `json:"success_rate"`
	EventTypes     map[string]int64 `json:"event_types"`
	ResourceTypes  map[string]int64 `json:"resource_types"`
	Actions        map[string]int64 `json:"actions"`
	TopUsers       []UserActivity   `json:"top_users"`
	Timeline       []TimelineEntry  `json:"timeline"`
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
}

// UserActivity represents user activity statistics
type UserActivity struct {
	UserID   string    `json:"user_id"`
	LogCount int64     `json:"log_count"`
	LastSeen time.Time `json:"last_seen"`
}

// TimelineEntry represents audit activity over time
type TimelineEntry struct {
	Date     string `json:"date"`
	LogCount int64  `json:"log_count"`
	Success  int64  `json:"success"`
	Failed   int64  `json:"failed"`
}

// NewAuditLogService creates a new audit log service
func NewAuditLogService(db *sql.DB, logger *log.Logger) *AuditLogService {
	logger.Printf("DEBUG: NewAuditLogService called - creating service instance")
	return &AuditLogService{
		db:     db,
		logger: logger,
	}
}

// LogEvent logs an audit event
func (s *AuditLogService) LogEvent(ctx context.Context, entry *AuditLogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// Handle user_id: convert to NULL if empty or "anonymous"
	var userID interface{} = nil
	if entry.UserID != nil && *entry.UserID != "" && *entry.UserID != "anonymous" {
		userID = *entry.UserID
	}

	// If tenant_id not set on entry, this is an error - audit logs require tenant_id
	tenantID := entry.TenantID
	if tenantID == "" {
		s.logger.Printf("Warning: audit log entry missing tenant_id - skipping audit log")
		return nil // Skip logging if no tenant context
	}

	query := `
		INSERT INTO audit_logs (
			tenant_id, event_type, resource_id, resource_type, user_id, action,
			old_value, new_value, ip_address, user_agent,
			success, error_msg, metadata, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING log_id, timestamp`

	err = s.db.QueryRowContext(ctx, query,
		tenantID, entry.EventType, entry.ResourceID, entry.ResourceType, userID, entry.Action,
		entry.OldValue, entry.NewValue, entry.IPAddress, entry.UserAgent,
		entry.Success, entry.ErrorMessage, string(metadataJSON), entry.Timestamp,
	).Scan(&entry.ID, &entry.Timestamp)

	if err != nil {
		s.logger.Printf("Failed to log audit event: %v", err)
		return fmt.Errorf("failed to log audit event: %w", err)
	}

	// Set CreatedAt to be the same as Timestamp
	entry.CreatedAt = entry.Timestamp

	return nil
}

// LogDownload logs an artifact download event
func (s *AuditLogService) LogDownload(ctx context.Context, tenantID string, artifactID uuid.UUID, userID, ipAddress, userAgent, sessionID string) error {
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}
	entry := &AuditLogEntry{
		TenantID:     tenantID,
		EventType:    "artifact_access",
		ResourceID:   artifactID.String(),
		ResourceType: "artifact",
		UserID:       userIDPtr,
		Action:       "download",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		SessionID:    sessionID,
		Success:      true, // Downloads are considered successful by default
		Metadata: map[string]interface{}{
			"download_timestamp": time.Now().Unix(),
			"client_type":        "api",
		},
	}
	return s.LogEvent(ctx, entry)
}

// LogUserSession logs user session events (login, logout, etc.)
func (s *AuditLogService) LogUserSession(ctx context.Context, tenantID, userID, action, ipAddress, userAgent, sessionID string, metadata map[string]interface{}) error {
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	// Determine success status from metadata if present, otherwise default to true
	success := true
	if metadata != nil {
		if loginSuccess, ok := metadata["login_successful"].(bool); ok {
			success = loginSuccess
		}
	}

	entry := &AuditLogEntry{
		TenantID:     tenantID,
		EventType:    "user_session",
		ResourceID:   "session",
		ResourceType: "system",
		UserID:       userIDPtr,
		Action:       action,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		SessionID:    sessionID,
		Success:      success,
		Metadata:     metadata,
	}
	return s.LogEvent(ctx, entry)
}

// LogPageView logs page/view access events
func (s *AuditLogService) LogPageView(ctx context.Context, tenantID, userID, page, ipAddress, userAgent, sessionID string) error {
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}
	entry := &AuditLogEntry{
		TenantID:     tenantID,
		EventType:    "page_view",
		ResourceID:   page,
		ResourceType: "system",
		UserID:       userIDPtr,
		Action:       "view",
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		SessionID:    sessionID,
		Success:      true, // Page views are considered successful by default
		Metadata: map[string]interface{}{
			"page":      page,
			"timestamp": time.Now().Unix(),
		},
	}
	return s.LogEvent(ctx, entry)
}

// LogAPIAccess logs API endpoint access
func (s *AuditLogService) LogAPIAccess(ctx context.Context, userID, tenantID, endpoint, method, ipAddress, userAgent, sessionID string, statusCode int) error {
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	// Determine success based on HTTP status code
	success := statusCode >= 200 && statusCode < 400

	entry := &AuditLogEntry{
		TenantID:     tenantID,
		EventType:    "api_access",
		ResourceID:   endpoint,
		ResourceType: "api",
		UserID:       userIDPtr,
		Action:       method,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		SessionID:    sessionID,
		Success:      success,
		Metadata: map[string]interface{}{
			"endpoint":    endpoint,
			"method":      method,
			"status_code": statusCode,
			"timestamp":   time.Now().Unix(),
		},
	}
	return s.LogEvent(ctx, entry)
}

// GetAuditLogs retrieves audit logs with filtering
func (s *AuditLogService) GetAuditLogs(ctx context.Context, filters *AuditLogFilters) ([]*AuditLogEntry, int64, error) {
	log.Printf("CRITICAL DEBUG: GetAuditLogs called in service layer - THIS SHOULD APPEAR IN LOGS")
	// Set default limit
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Limit > 1000 {
		filters.Limit = 1000
	}

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	// Filter by tenant ID (critical for multi-tenancy)
	if filters.TenantID != "" {
		whereClause += fmt.Sprintf(" AND al.tenant_id = $%d", argIndex)
		args = append(args, filters.TenantID)
		argIndex++
	}

	if filters.EventType != "" {
		whereClause += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, filters.EventType)
		argIndex++
	}

	if filters.ResourceType != "" {
		whereClause += fmt.Sprintf(" AND resource_type = $%d", argIndex)
		args = append(args, filters.ResourceType)
		argIndex++
	}

	if filters.UserID != "" {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, filters.UserID)
		argIndex++
	}

	if filters.Action != "" {
		whereClause += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, filters.Action)
		argIndex++
	}

	if !filters.StartTime.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filters.StartTime)
		argIndex++
	}

	if !filters.EndTime.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filters.EndTime)
		argIndex++
	}

	if filters.Success != nil {
		whereClause += fmt.Sprintf(" AND success = $%d", argIndex)
		args = append(args, *filters.Success)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs al %s", whereClause)
	var total int64
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get logs with pagination
	query := fmt.Sprintf(`
		SELECT al.log_id, al.tenant_id, al.event_type, al.resource_type, al.resource_id, al.user_id, 
		       COALESCE(u.username, 'unknown') as username, al.action,
			   COALESCE(al.old_value, '') as old_value, 
			   COALESCE(al.new_value, '') as new_value, 
			   COALESCE(al.ip_address::text, '') as ip_address, 
			   COALESCE(al.user_agent, '') as user_agent,
			   COALESCE(al.success, true) as success,
			   COALESCE(al.error_msg, '') as error_msg,
			   COALESCE(al.metadata::text, '{}') as metadata, 
			   al.timestamp
		FROM audit_logs al
		LEFT JOIN users u ON al.user_id = u.user_id
		%s
		ORDER BY al.timestamp DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	s.logger.Printf("DEBUG: Query = %s", query)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*AuditLogEntry
	for rows.Next() {
		var entry AuditLogEntry
		var metadataJSON string

		s.logger.Printf("DEBUG: About to scan row")

		err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.EventType, &entry.ResourceType,
			&entry.ResourceID, &entry.UserID, &entry.Username, &entry.Action, &entry.OldValue, &entry.NewValue,
			&entry.IPAddress, &entry.UserAgent,
			&entry.Success, &entry.ErrorMessage, &metadataJSON, &entry.Timestamp,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Set CreatedAt to be the same as Timestamp since database only has timestamp
		entry.CreatedAt = entry.Timestamp

		// Parse metadata
		if metadataJSON != "" && metadataJSON != "{}" {
			json.Unmarshal([]byte(metadataJSON), &entry.Metadata)
		}

		logs = append(logs, &entry)
	}

	return logs, total, nil
}

// GetActiveUsersToday returns count of active users today
func (s *AuditLogService) GetActiveUsersToday(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(DISTINCT user_id) 
		FROM audit_logs 
		WHERE timestamp >= CURRENT_DATE
		AND timestamp < CURRENT_DATE + INTERVAL '1 day'
		AND user_id IS NOT NULL`

	var count int64
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// GetDownloadsToday returns count of downloads today
func (s *AuditLogService) GetDownloadsToday(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(*) 
		FROM audit_logs 
		WHERE action = 'download' 
		AND timestamp >= CURRENT_DATE
		AND timestamp < CURRENT_DATE + INTERVAL '1 day'`

	var count int64
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// GetUserActivity returns user activity statistics
func (s *AuditLogService) GetUserActivity(ctx context.Context, userID string, days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}

	query := `
		SELECT 
			COUNT(*) as total_actions,
			COUNT(DISTINCT DATE(timestamp)) as active_days,
			COUNT(CASE WHEN action = 'download' THEN 1 END) as downloads,
			COUNT(CASE WHEN action = 'login' THEN 1 END) as logins,
			COUNT(CASE WHEN event_type = 'page_view' THEN 1 END) as page_views
		FROM audit_logs 
		WHERE user_id = $1 
		AND timestamp >= CURRENT_DATE - INTERVAL '%d days'`

	var totalActions, activeDays, downloads, logins, pageViews int64
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(query, days), userID).Scan(
		&totalActions, &activeDays, &downloads, &logins, &pageViews,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity: %w", err)
	}

	return map[string]interface{}{
		"user_id":       userID,
		"period_days":   days,
		"total_actions": totalActions,
		"active_days":   activeDays,
		"downloads":     downloads,
		"logins":        logins,
		"page_views":    pageViews,
		"last_updated":  time.Now(),
	}, nil
}

// Helper function to extract IP from request
func ExtractIPFromRequest(remoteAddr, xForwardedFor, xRealIP string) string {
	// Check X-Forwarded-For header
	if xForwardedFor != "" {
		ips := []string{}
		for _, ip := range []string{xForwardedFor} {
			if net.ParseIP(ip) != nil {
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	if xRealIP != "" && net.ParseIP(xRealIP) != nil {
		return xRealIP
	}

	// Fall back to RemoteAddr
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}

	return remoteAddr
}

// GetAuditLogByID retrieves a single audit log entry by ID
func (s *AuditLogService) GetAuditLogByID(ctx context.Context, id uuid.UUID) (*AuditLogEntry, error) {
	query := `
		SELECT log_id, tenant_id, event_type, resource_id, resource_type, user_id, action,
			   COALESCE(old_value, '') as old_value, 
			   COALESCE(new_value, '') as new_value, 
			   COALESCE(ip_address::text, '') as ip_address, 
			   COALESCE(user_agent, '') as user_agent,
			   COALESCE(success, true) as success,
			   COALESCE(error_msg, '') as error_msg,
			   COALESCE(metadata::text, '{}') as metadata, 
			   timestamp
		FROM audit_logs 
		WHERE log_id = $1`

	var entry AuditLogEntry
	var metadataJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID, &entry.TenantID, &entry.EventType, &entry.ResourceID, &entry.ResourceType,
		&entry.UserID, &entry.Action, &entry.OldValue, &entry.NewValue,
		&entry.IPAddress, &entry.UserAgent,
		&entry.Success, &entry.ErrorMessage, &metadataJSON, &entry.Timestamp,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	// Set CreatedAt to be the same as Timestamp
	entry.CreatedAt = entry.Timestamp

	// Parse metadata
	if metadataJSON != "" && metadataJSON != "{}" {
		json.Unmarshal([]byte(metadataJSON), &entry.Metadata)
	}

	return &entry, nil
}

// GetAuditStats retrieves audit statistics for a given time period and optional tenant
func (s *AuditLogService) GetAuditStats(ctx context.Context, startTime, endTime time.Time, tenantID ...string) (*AuditStats, error) {
	stats := &AuditStats{
		EventTypes:    make(map[string]int64),
		ResourceTypes: make(map[string]int64),
		Actions:       make(map[string]int64),
		StartTime:     startTime,
		EndTime:       endTime,
	}

	// Build tenant filter
	tenantFilter := ""
	args := []interface{}{startTime, endTime}
	if len(tenantID) > 0 && tenantID[0] != "" {
		tenantFilter = " AND tenant_id = $3"
		args = append(args, tenantID[0])
	}

	// Get total counts
	totalQuery := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN success = true THEN 1 END) as successful,
			COUNT(CASE WHEN success = false THEN 1 END) as failed
		FROM audit_logs 
		WHERE timestamp >= $1 AND timestamp <= $2%s`, tenantFilter)

	err := s.db.QueryRowContext(ctx, totalQuery, args...).Scan(
		&stats.TotalLogs, &stats.SuccessfulLogs, &stats.FailedLogs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get total counts: %w", err)
	}

	// Calculate success rate
	if stats.TotalLogs > 0 {
		stats.SuccessRate = float64(stats.SuccessfulLogs) / float64(stats.TotalLogs) * 100
	}

	// Get event type distribution
	eventTypeQuery := fmt.Sprintf(`
		SELECT event_type, COUNT(*) 
		FROM audit_logs 
		WHERE timestamp >= $1 AND timestamp <= $2%s 
		GROUP BY event_type 
		ORDER BY COUNT(*) DESC`, tenantFilter)

	rows, err := s.db.QueryContext(ctx, eventTypeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get event types: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err == nil {
			stats.EventTypes[eventType] = count
		}
	}

	// Get resource type distribution
	resourceTypeQuery := fmt.Sprintf(`
		SELECT resource_type, COUNT(*) 
		FROM audit_logs 
		WHERE timestamp >= $1 AND timestamp <= $2%s 
		GROUP BY resource_type 
		ORDER BY COUNT(*) DESC`, tenantFilter)

	rows, err = s.db.QueryContext(ctx, resourceTypeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource types: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var resourceType string
		var count int64
		if err := rows.Scan(&resourceType, &count); err == nil {
			stats.ResourceTypes[resourceType] = count
		}
	}

	// Get action distribution
	actionQuery := fmt.Sprintf(`
		SELECT action, COUNT(*) 
		FROM audit_logs 
		WHERE timestamp >= $1 AND timestamp <= $2%s 
		GROUP BY action 
		ORDER BY COUNT(*) DESC`, tenantFilter)

	rows, err = s.db.QueryContext(ctx, actionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var action string
		var count int64
		if err := rows.Scan(&action, &count); err == nil {
			stats.Actions[action] = count
		}
	}

	// Get top users
	userQuery := `
		SELECT user_id, COUNT(*) as log_count, MAX(timestamp) as last_seen
		FROM audit_logs 
		WHERE timestamp >= $1 AND timestamp <= $2 
		GROUP BY user_id 
		ORDER BY COUNT(*) DESC 
		LIMIT 10`

	rows, err = s.db.QueryContext(ctx, userQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get top users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user UserActivity
		if err := rows.Scan(&user.UserID, &user.LogCount, &user.LastSeen); err == nil {
			stats.TopUsers = append(stats.TopUsers, user)
		}
	}

	// Get timeline data (daily aggregation)
	timelineQuery := `
		SELECT 
			DATE(timestamp) as date,
			COUNT(*) as log_count,
			COUNT(CASE WHEN success = true THEN 1 END) as success_count,
			COUNT(CASE WHEN success = false THEN 1 END) as failed_count
		FROM audit_logs 
		WHERE timestamp >= $1 AND timestamp <= $2 
		GROUP BY DATE(timestamp) 
		ORDER BY date ASC`

	rows, err = s.db.QueryContext(ctx, timelineQuery, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry TimelineEntry
		var date time.Time
		if err := rows.Scan(&date, &entry.LogCount, &entry.Success, &entry.Failed); err == nil {
			entry.Date = date.Format("2006-01-02")
			stats.Timeline = append(stats.Timeline, entry)
		}
	}

	return stats, nil
}
