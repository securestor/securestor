package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/securestor/securestor/internal/models"
)

// AuditRepository handles audit log operations
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// CreateLog creates a new audit log entry
func (r *AuditRepository) CreateLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			event_type, resource_id, resource_type, user_id, action, 
			old_value, new_value, ip_address, user_agent, success, 
			error_msg, timestamp, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query,
		log.EventType,
		log.ResourceID,
		log.ResourceType,
		log.UserID,
		log.Action,
		log.OldValue,
		log.NewValue,
		log.IPAddress,
		log.UserAgent,
		log.Success,
		log.ErrorMsg,
		log.Timestamp,
		log.Metadata,
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetLogs retrieves audit logs with filtering
func (r *AuditRepository) GetLogs(ctx context.Context, filters AuditFilters) ([]*models.AuditLog, int, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

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

	if !filters.StartDate.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filters.StartDate)
		argIndex++
	}

	if !filters.EndDate.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filters.EndDate)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get logs with pagination
	query := fmt.Sprintf(`
		SELECT id, event_type, resource_id, resource_type, user_id, action,
			   old_value, new_value, ip_address, user_agent, success,
			   error_msg, timestamp, metadata
		FROM audit_logs 
		%s 
		ORDER BY timestamp DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		err := rows.Scan(
			&log.ID,
			&log.EventType,
			&log.ResourceID,
			&log.ResourceType,
			&log.UserID,
			&log.Action,
			&log.OldValue,
			&log.NewValue,
			&log.IPAddress,
			&log.UserAgent,
			&log.Success,
			&log.ErrorMsg,
			&log.Timestamp,
			&log.Metadata,
		)
		if err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

// AuditFilters defines filters for audit log queries
type AuditFilters struct {
	EventType    string
	ResourceType string
	ResourceID   string
	UserID       string
	StartDate    time.Time
	EndDate      time.Time
	Limit        int
	Offset       int
}
