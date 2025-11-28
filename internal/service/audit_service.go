package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/securestor/securestor/internal/models"
)

// AuditService handles audit logging and compliance reporting
type AuditService struct {
	db *sql.DB
}

// NewAuditService creates a new audit service instance
func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{
		db: db,
	}
}

// LogPolicyDecision logs a policy decision from OPA
func (s *AuditService) LogPolicyDecision(ctx context.Context, decision *models.PolicyAuditLog) error {
	query := `
		INSERT INTO policy_audit_logs (
			id, request_id, user_id, resource, action, decision, 
			policy_name, input_data, policy_output, execution_time_ms,
			ip_address, user_agent, session_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	inputJSON, err := json.Marshal(decision.InputData)
	if err != nil {
		return fmt.Errorf("failed to marshal input data: %w", err)
	}

	outputJSON, err := json.Marshal(decision.PolicyOutput)
	if err != nil {
		return fmt.Errorf("failed to marshal policy output: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		decision.ID, decision.RequestID, decision.UserID, decision.Resource,
		decision.Action, decision.Decision, decision.PolicyName,
		inputJSON, outputJSON, decision.ExecutionTimeMs,
		decision.IPAddress, decision.UserAgent, decision.SessionID, decision.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to log policy decision: %w", err)
	}

	return nil
}

// LogSecurityViolation logs a security policy violation
func (s *AuditService) LogSecurityViolation(ctx context.Context, violation *models.SecurityPolicyViolation) error {
	query := `
		INSERT INTO security_policy_violations (
			id, audit_log_id, violation_type, severity, description, 
			is_investigated, investigated_by, investigated_at, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
		violation.ID, violation.AuditLogID, violation.ViolationType,
		violation.Severity, violation.Description, violation.IsInvestigated,
		violation.InvestigatedBy, violation.InvestigatedAt, violation.Notes, violation.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to log security violation: %w", err)
	}

	return nil
}

// GetPolicyAuditLogs retrieves policy audit logs with filtering
func (s *AuditService) GetPolicyAuditLogs(ctx context.Context, filter *PolicyAuditFilter) ([]*models.PolicyAuditLog, error) {
	query := `
		SELECT id, request_id, user_id, resource, action, decision, 
		       policy_name, input_data, policy_output, execution_time_ms,
		       ip_address, user_agent, session_id, created_at
		FROM policy_audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if filter.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
	}

	if filter.Resource != "" {
		argCount++
		query += fmt.Sprintf(" AND resource = $%d", argCount)
		args = append(args, filter.Resource)
	}

	if filter.Action != "" {
		argCount++
		query += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, filter.Action)
	}

	if filter.PolicyName != "" {
		argCount++
		query += fmt.Sprintf(" AND policy_name = $%d", argCount)
		args = append(args, filter.PolicyName)
	}

	if filter.Decision != "" {
		argCount++
		query += fmt.Sprintf(" AND decision = $%d", argCount)
		args = append(args, filter.Decision)
	}

	if !filter.StartDate.IsZero() {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query policy audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.PolicyAuditLog
	for rows.Next() {
		log := &models.PolicyAuditLog{}
		var inputData, policyOutput []byte

		err := rows.Scan(
			&log.ID, &log.RequestID, &log.UserID, &log.Resource, &log.Action,
			&log.Decision, &log.PolicyName, &inputData, &policyOutput,
			&log.ExecutionTimeMs, &log.IPAddress, &log.UserAgent, &log.SessionID, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy audit log: %w", err)
		}

		if len(inputData) > 0 {
			if err := json.Unmarshal(inputData, &log.InputData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal input data: %w", err)
			}
		}

		if len(policyOutput) > 0 {
			if err := json.Unmarshal(policyOutput, &log.PolicyOutput); err != nil {
				return nil, fmt.Errorf("failed to unmarshal policy output: %w", err)
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetSecurityViolations retrieves security violations with filtering
func (s *AuditService) GetSecurityViolations(ctx context.Context, filter *SecurityViolationFilter) ([]*models.SecurityPolicyViolation, error) {
	query := `
		SELECT id, audit_log_id, violation_type, severity, description,
		       is_investigated, investigated_by, investigated_at, notes, created_at
		FROM security_policy_violations
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	if filter.ViolationType != "" {
		argCount++
		query += fmt.Sprintf(" AND violation_type = $%d", argCount)
		args = append(args, filter.ViolationType)
	}

	if filter.Severity != "" {
		argCount++
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, filter.Severity)
	}

	if !filter.StartDate.IsZero() {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, filter.EndDate)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query security violations: %w", err)
	}
	defer rows.Close()

	var violations []*models.SecurityPolicyViolation
	for rows.Next() {
		violation := &models.SecurityPolicyViolation{}

		err := rows.Scan(
			&violation.ID, &violation.AuditLogID, &violation.ViolationType,
			&violation.Severity, &violation.Description, &violation.IsInvestigated,
			&violation.InvestigatedBy, &violation.InvestigatedAt, &violation.Notes, &violation.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan security violation: %w", err)
		}

		violations = append(violations, violation)
	}

	return violations, nil
}

// GenerateComplianceReport generates a compliance report for a given time period
func (s *AuditService) GenerateComplianceReport(ctx context.Context, startDate, endDate time.Time, generatedBy int64) (*models.AuditComplianceReport, error) {
	report := &models.AuditComplianceReport{
		ReportType:   "compliance_summary",
		ReportFormat: "json",
		GeneratedBy:  generatedBy,
		GeneratedAt:  time.Now(),
	}

	// Get policy decision statistics
	policyStats, err := s.getPolicyDecisionStats(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy statistics: %w", err)
	}

	// Get security violation statistics
	violationStats, err := s.getViolationStats(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get violation statistics: %w", err)
	}

	// Build report parameters
	parameters := map[string]interface{}{
		"start_date":          startDate,
		"end_date":            endDate,
		"policy_decisions":    policyStats,
		"security_violations": violationStats,
		"compliance_score":    s.calculateComplianceScore(policyStats, violationStats),
	}

	parametersJSON, err := json.Marshal(parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Store the report
	query := `
		INSERT INTO audit_compliance_reports (
			report_type, report_format, parameters, 
			generated_by, generated_at
		) VALUES ($1, $2, $3, $4, $5) RETURNING id
	`

	err = s.db.QueryRowContext(ctx, query,
		report.ReportType, report.ReportFormat, parametersJSON,
		report.GeneratedBy, report.GeneratedAt,
	).Scan(&report.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to store compliance report: %w", err)
	}

	report.Parameters = parameters

	return report, nil
}

// getPolicyDecisionStats gets policy decision statistics
func (s *AuditService) getPolicyDecisionStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	query := `
		SELECT 
			decision,
			COUNT(*) as count
		FROM policy_audit_logs 
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY decision
	`

	rows, err := s.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]interface{}{
		"total":   0,
		"allowed": 0,
		"denied":  0,
	}

	for rows.Next() {
		var decision string
		var count int
		if err := rows.Scan(&decision, &count); err != nil {
			return nil, err
		}

		stats[decision] = count
		stats["total"] = stats["total"].(int) + count
	}

	return stats, nil
}

// getViolationStats gets security violation statistics
func (s *AuditService) getViolationStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	query := `
		SELECT 
			severity,
			COUNT(*) as count
		FROM security_policy_violations 
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY severity
	`

	rows, err := s.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]interface{}{
		"total":    0,
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, err
		}

		stats[severity] = count
		stats["total"] = stats["total"].(int) + count
	}

	return stats, nil
}

// calculateComplianceScore calculates a compliance score based on violations
func (s *AuditService) calculateComplianceScore(policyStats, violationStats map[string]interface{}) float64 {
	totalDecisions := policyStats["total"].(int)
	totalViolations := violationStats["total"].(int)

	if totalDecisions == 0 {
		return 100.0
	}

	// Simple compliance score calculation
	violationRate := float64(totalViolations) / float64(totalDecisions)
	return (1.0 - violationRate) * 100.0
}

// Filter types for audit queries
type PolicyAuditFilter struct {
	UserID     *int64
	Resource   string
	Action     string
	PolicyName string
	Decision   string
	StartDate  time.Time
	EndDate    time.Time
	Limit      int
}

type SecurityViolationFilter struct {
	ViolationType string
	Severity      string
	StartDate     time.Time
	EndDate       time.Time
	Limit         int
}
