package service

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// AuditExportService handles exporting audit data in various formats
type AuditExportService struct {
	db       *sql.DB
	auditSvc *AuditService
}

// NewAuditExportService creates a new audit export service
func NewAuditExportService(db *sql.DB, auditSvc *AuditService) *AuditExportService {
	return &AuditExportService{
		db:       db,
		auditSvc: auditSvc,
	}
}

// ExportFormat represents the supported export formats
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatSIEM ExportFormat = "siem"
	FormatXML  ExportFormat = "xml"
)

// ExportRequest represents an audit export request
type ExportRequest struct {
	Format          ExportFormat           `json:"format"`
	StartDate       time.Time              `json:"start_date"`
	EndDate         time.Time              `json:"end_date"`
	DataTypes       []string               `json:"data_types"` // "policy_decisions", "violations", "mfa_attempts", etc.
	Filters         map[string]interface{} `json:"filters"`
	Compression     bool                   `json:"compression"`
	MaxRecords      int                    `json:"max_records"`
	IncludeMetadata bool                   `json:"include_metadata"`
}

// ExportResult represents the result of an export operation
type ExportResult struct {
	ID           string     `json:"id"`
	Format       string     `json:"format"`
	Status       string     `json:"status"`
	RecordCount  int        `json:"record_count"`
	FilePath     string     `json:"file_path,omitempty"`
	FileSize     int64      `json:"file_size,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// ExportAuditData exports audit data according to the specified request
func (s *AuditExportService) ExportAuditData(ctx context.Context, request *ExportRequest, writer io.Writer) (*ExportResult, error) {
	result := &ExportResult{
		ID:        fmt.Sprintf("export_%d", time.Now().UnixNano()),
		Format:    string(request.Format),
		Status:    "in_progress",
		CreatedAt: time.Now(),
	}

	// Validate request
	if err := s.validateExportRequest(request); err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
		return result, err
	}

	// Export data based on format
	var recordCount int
	var err error

	switch request.Format {
	case FormatJSON:
		recordCount, err = s.exportToJSON(ctx, request, writer)
	case FormatCSV:
		recordCount, err = s.exportToCSV(ctx, request, writer)
	case FormatSIEM:
		recordCount, err = s.exportToSIEM(ctx, request, writer)
	case FormatXML:
		recordCount, err = s.exportToXML(ctx, request, writer)
	default:
		err = fmt.Errorf("unsupported export format: %s", request.Format)
	}

	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = err.Error()
		return result, err
	}

	// Complete the result
	now := time.Now()
	result.Status = "completed"
	result.RecordCount = recordCount
	result.CompletedAt = &now

	return result, nil
}

// exportToJSON exports audit data in JSON format
func (s *AuditExportService) exportToJSON(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	// Start JSON object
	writer.Write([]byte("{\n"))

	if request.IncludeMetadata {
		metadata := map[string]interface{}{
			"export_id":   fmt.Sprintf("export_%d", time.Now().UnixNano()),
			"export_time": time.Now().UTC(),
			"start_date":  request.StartDate,
			"end_date":    request.EndDate,
			"format":      request.Format,
			"data_types":  request.DataTypes,
			"filters":     request.Filters,
		}

		writer.Write([]byte("  \"metadata\": "))
		encoder.Encode(metadata)
		writer.Write([]byte(",\n"))
	}

	writer.Write([]byte("  \"data\": {\n"))

	totalRecords := 0
	isFirst := true

	// Export policy decisions
	if s.shouldIncludeDataType(request.DataTypes, "policy_decisions") {
		if !isFirst {
			writer.Write([]byte(",\n"))
		}
		writer.Write([]byte("    \"policy_decisions\": "))

		count, err := s.exportPolicyDecisionsJSON(ctx, request, encoder)
		if err != nil {
			return 0, err
		}
		totalRecords += count
		isFirst = false
	}

	// Export security violations
	if s.shouldIncludeDataType(request.DataTypes, "violations") {
		if !isFirst {
			writer.Write([]byte(",\n"))
		}
		writer.Write([]byte("    \"security_violations\": "))

		count, err := s.exportSecurityViolationsJSON(ctx, request, encoder)
		if err != nil {
			return 0, err
		}
		totalRecords += count
		isFirst = false
	}

	// Export MFA attempts
	if s.shouldIncludeDataType(request.DataTypes, "mfa_attempts") {
		if !isFirst {
			writer.Write([]byte(",\n"))
		}
		writer.Write([]byte("    \"mfa_attempts\": "))

		count, err := s.exportMFAAttemptsJSON(ctx, request, encoder)
		if err != nil {
			return 0, err
		}
		totalRecords += count
		isFirst = false
	}

	writer.Write([]byte("\n  }\n"))
	writer.Write([]byte("}\n"))

	return totalRecords, nil
}

// exportToCSV exports audit data in CSV format
func (s *AuditExportService) exportToCSV(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	totalRecords := 0

	// Export policy decisions
	if s.shouldIncludeDataType(request.DataTypes, "policy_decisions") {
		count, err := s.exportPolicyDecisionsCSV(ctx, request, csvWriter)
		if err != nil {
			return 0, err
		}
		totalRecords += count
	}

	// Export security violations
	if s.shouldIncludeDataType(request.DataTypes, "violations") {
		count, err := s.exportSecurityViolationsCSV(ctx, request, csvWriter)
		if err != nil {
			return 0, err
		}
		totalRecords += count
	}

	// Export MFA attempts
	if s.shouldIncludeDataType(request.DataTypes, "mfa_attempts") {
		count, err := s.exportMFAAttemptsCSV(ctx, request, csvWriter)
		if err != nil {
			return 0, err
		}
		totalRecords += count
	}

	return totalRecords, nil
}

// exportToSIEM exports audit data in SIEM-compatible format (JSON lines)
func (s *AuditExportService) exportToSIEM(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	totalRecords := 0

	// Export policy decisions
	if s.shouldIncludeDataType(request.DataTypes, "policy_decisions") {
		count, err := s.exportPolicyDecisionsSIEM(ctx, request, writer)
		if err != nil {
			return 0, err
		}
		totalRecords += count
	}

	// Export security violations
	if s.shouldIncludeDataType(request.DataTypes, "violations") {
		count, err := s.exportSecurityViolationsSIEM(ctx, request, writer)
		if err != nil {
			return 0, err
		}
		totalRecords += count
	}

	// Export MFA attempts
	if s.shouldIncludeDataType(request.DataTypes, "mfa_attempts") {
		count, err := s.exportMFAAttemptsSIEM(ctx, request, writer)
		if err != nil {
			return 0, err
		}
		totalRecords += count
	}

	return totalRecords, nil
}

// exportToXML exports audit data in XML format
func (s *AuditExportService) exportToXML(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	writer.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))
	writer.Write([]byte("<audit_export>\n"))

	if request.IncludeMetadata {
		writer.Write([]byte("  <metadata>\n"))
		writer.Write([]byte(fmt.Sprintf("    <export_id>export_%d</export_id>\n", time.Now().UnixNano())))
		writer.Write([]byte(fmt.Sprintf("    <export_time>%s</export_time>\n", time.Now().UTC().Format(time.RFC3339))))
		writer.Write([]byte(fmt.Sprintf("    <start_date>%s</start_date>\n", request.StartDate.Format(time.RFC3339))))
		writer.Write([]byte(fmt.Sprintf("    <end_date>%s</end_date>\n", request.EndDate.Format(time.RFC3339))))
		writer.Write([]byte(fmt.Sprintf("    <format>%s</format>\n", request.Format)))
		writer.Write([]byte("  </metadata>\n"))
	}

	totalRecords := 0

	// Export policy decisions
	if s.shouldIncludeDataType(request.DataTypes, "policy_decisions") {
		writer.Write([]byte("  <policy_decisions>\n"))
		count, err := s.exportPolicyDecisionsXML(ctx, request, writer)
		if err != nil {
			return 0, err
		}
		writer.Write([]byte("  </policy_decisions>\n"))
		totalRecords += count
	}

	// Export security violations
	if s.shouldIncludeDataType(request.DataTypes, "violations") {
		writer.Write([]byte("  <security_violations>\n"))
		count, err := s.exportSecurityViolationsXML(ctx, request, writer)
		if err != nil {
			return 0, err
		}
		writer.Write([]byte("  </security_violations>\n"))
		totalRecords += count
	}

	writer.Write([]byte("</audit_export>\n"))

	return totalRecords, nil
}

// Helper methods for specific data type exports

func (s *AuditExportService) exportPolicyDecisionsJSON(ctx context.Context, request *ExportRequest, encoder *json.Encoder) (int, error) {
	filter := &PolicyAuditFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	// Apply additional filters from request
	if userID, ok := request.Filters["user_id"].(int64); ok {
		filter.UserID = &userID
	}
	if resource, ok := request.Filters["resource"].(string); ok {
		filter.Resource = resource
	}
	if action, ok := request.Filters["action"].(string); ok {
		filter.Action = action
	}
	if decision, ok := request.Filters["decision"].(string); ok {
		filter.Decision = decision
	}

	logs, err := s.auditSvc.GetPolicyAuditLogs(ctx, filter)
	if err != nil {
		return 0, err
	}

	err = encoder.Encode(logs)
	return len(logs), err
}

func (s *AuditExportService) exportPolicyDecisionsCSV(ctx context.Context, request *ExportRequest, csvWriter *csv.Writer) (int, error) {
	// Write CSV header
	header := []string{
		"id", "request_id", "user_id", "resource", "action", "decision",
		"policy_name", "execution_time_ms", "ip_address", "user_agent",
		"session_id", "created_at",
	}
	csvWriter.Write(header)

	filter := &PolicyAuditFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	logs, err := s.auditSvc.GetPolicyAuditLogs(ctx, filter)
	if err != nil {
		return 0, err
	}

	for _, log := range logs {
		record := []string{
			fmt.Sprintf("%d", log.ID),
			log.RequestID,
			fmt.Sprintf("%d", *log.UserID),
			log.Resource,
			log.Action,
			log.Decision,
			log.PolicyName,
			fmt.Sprintf("%d", log.ExecutionTimeMs),
			stringPtrToString(log.IPAddress),
			stringPtrToString(log.UserAgent),
			stringPtrToString(log.SessionID),
			log.CreatedAt.Format(time.RFC3339),
		}
		csvWriter.Write(record)
	}

	return len(logs), nil
}

func (s *AuditExportService) exportPolicyDecisionsSIEM(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	filter := &PolicyAuditFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	logs, err := s.auditSvc.GetPolicyAuditLogs(ctx, filter)
	if err != nil {
		return 0, err
	}

	encoder := json.NewEncoder(writer)
	for _, log := range logs {
		// Convert to SIEM format with additional metadata
		siemEvent := map[string]interface{}{
			"timestamp":      log.CreatedAt.UTC(),
			"event_type":     "policy_decision",
			"source":         "securestor",
			"severity":       getSeverityForDecision(log.Decision),
			"user_id":        log.UserID,
			"resource":       log.Resource,
			"action":         log.Action,
			"decision":       log.Decision,
			"policy_name":    log.PolicyName,
			"execution_time": log.ExecutionTimeMs,
			"source_ip":      log.IPAddress,
			"user_agent":     log.UserAgent,
			"session_id":     log.SessionID,
			"request_id":     log.RequestID,
		}

		encoder.Encode(siemEvent)
	}

	return len(logs), nil
}

func (s *AuditExportService) exportPolicyDecisionsXML(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	filter := &PolicyAuditFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	logs, err := s.auditSvc.GetPolicyAuditLogs(ctx, filter)
	if err != nil {
		return 0, err
	}

	for _, log := range logs {
		writer.Write([]byte("    <policy_decision>\n"))
		writer.Write([]byte(fmt.Sprintf("      <id>%d</id>\n", log.ID)))
		writer.Write([]byte(fmt.Sprintf("      <request_id>%s</request_id>\n", log.RequestID)))
		writer.Write([]byte(fmt.Sprintf("      <user_id>%d</user_id>\n", *log.UserID)))
		writer.Write([]byte(fmt.Sprintf("      <resource>%s</resource>\n", log.Resource)))
		writer.Write([]byte(fmt.Sprintf("      <action>%s</action>\n", log.Action)))
		writer.Write([]byte(fmt.Sprintf("      <decision>%s</decision>\n", log.Decision)))
		writer.Write([]byte(fmt.Sprintf("      <policy_name>%s</policy_name>\n", log.PolicyName)))
		writer.Write([]byte(fmt.Sprintf("      <created_at>%s</created_at>\n", log.CreatedAt.Format(time.RFC3339))))
		writer.Write([]byte("    </policy_decision>\n"))
	}

	return len(logs), nil
}

func (s *AuditExportService) exportSecurityViolationsJSON(ctx context.Context, request *ExportRequest, encoder *json.Encoder) (int, error) {
	filter := &SecurityViolationFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	violations, err := s.auditSvc.GetSecurityViolations(ctx, filter)
	if err != nil {
		return 0, err
	}

	err = encoder.Encode(violations)
	return len(violations), err
}

func (s *AuditExportService) exportSecurityViolationsCSV(ctx context.Context, request *ExportRequest, csvWriter *csv.Writer) (int, error) {
	// Write CSV header
	header := []string{
		"id", "audit_log_id", "violation_type", "severity", "description",
		"is_investigated", "investigated_by", "investigated_at", "notes", "created_at",
	}
	csvWriter.Write(header)

	filter := &SecurityViolationFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	violations, err := s.auditSvc.GetSecurityViolations(ctx, filter)
	if err != nil {
		return 0, err
	}

	for _, violation := range violations {
		record := []string{
			fmt.Sprintf("%d", violation.ID),
			fmt.Sprintf("%d", violation.AuditLogID),
			violation.ViolationType,
			violation.Severity,
			violation.Description,
			fmt.Sprintf("%t", violation.IsInvestigated),
			int64PtrToString(violation.InvestigatedBy),
			timePtrToString(violation.InvestigatedAt),
			stringPtrToString(violation.Notes),
			violation.CreatedAt.Format(time.RFC3339),
		}
		csvWriter.Write(record)
	}

	return len(violations), nil
}

func (s *AuditExportService) exportSecurityViolationsSIEM(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	filter := &SecurityViolationFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	violations, err := s.auditSvc.GetSecurityViolations(ctx, filter)
	if err != nil {
		return 0, err
	}

	encoder := json.NewEncoder(writer)
	for _, violation := range violations {
		siemEvent := map[string]interface{}{
			"timestamp":       violation.CreatedAt.UTC(),
			"event_type":      "security_violation",
			"source":          "securestor",
			"severity":        violation.Severity,
			"violation_type":  violation.ViolationType,
			"description":     violation.Description,
			"audit_log_id":    violation.AuditLogID,
			"is_investigated": violation.IsInvestigated,
			"investigated_by": violation.InvestigatedBy,
			"investigated_at": violation.InvestigatedAt,
			"notes":           violation.Notes,
		}

		encoder.Encode(siemEvent)
	}

	return len(violations), nil
}

func (s *AuditExportService) exportSecurityViolationsXML(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	filter := &SecurityViolationFilter{
		StartDate: request.StartDate,
		EndDate:   request.EndDate,
		Limit:     request.MaxRecords,
	}

	violations, err := s.auditSvc.GetSecurityViolations(ctx, filter)
	if err != nil {
		return 0, err
	}

	for _, violation := range violations {
		writer.Write([]byte("    <security_violation>\n"))
		writer.Write([]byte(fmt.Sprintf("      <id>%d</id>\n", violation.ID)))
		writer.Write([]byte(fmt.Sprintf("      <audit_log_id>%d</audit_log_id>\n", violation.AuditLogID)))
		writer.Write([]byte(fmt.Sprintf("      <violation_type>%s</violation_type>\n", violation.ViolationType)))
		writer.Write([]byte(fmt.Sprintf("      <severity>%s</severity>\n", violation.Severity)))
		writer.Write([]byte(fmt.Sprintf("      <description>%s</description>\n", violation.Description)))
		writer.Write([]byte(fmt.Sprintf("      <created_at>%s</created_at>\n", violation.CreatedAt.Format(time.RFC3339))))
		writer.Write([]byte("    </security_violation>\n"))
	}

	return len(violations), nil
}

func (s *AuditExportService) exportMFAAttemptsJSON(ctx context.Context, request *ExportRequest, encoder *json.Encoder) (int, error) {
	// This would fetch MFA attempts from the database
	// For now, return empty array as placeholder
	mfaAttempts := []interface{}{}
	err := encoder.Encode(mfaAttempts)
	return len(mfaAttempts), err
}

func (s *AuditExportService) exportMFAAttemptsCSV(ctx context.Context, request *ExportRequest, csvWriter *csv.Writer) (int, error) {
	// Write CSV header for MFA attempts
	header := []string{
		"id", "user_id", "attempt_type", "success", "ip_address", "user_agent", "created_at",
	}
	csvWriter.Write(header)

	// This would fetch and write MFA attempts
	// For now, return 0 as placeholder
	return 0, nil
}

func (s *AuditExportService) exportMFAAttemptsSIEM(ctx context.Context, request *ExportRequest, writer io.Writer) (int, error) {
	// This would fetch MFA attempts and export in SIEM format
	// For now, return 0 as placeholder
	return 0, nil
}

// Helper functions

func (s *AuditExportService) validateExportRequest(request *ExportRequest) error {
	if request.StartDate.IsZero() || request.EndDate.IsZero() {
		return fmt.Errorf("start_date and end_date are required")
	}

	if request.EndDate.Before(request.StartDate) {
		return fmt.Errorf("end_date must be after start_date")
	}

	if len(request.DataTypes) == 0 {
		return fmt.Errorf("at least one data type must be specified")
	}

	if request.MaxRecords > 100000 {
		return fmt.Errorf("max_records cannot exceed 100,000")
	}

	return nil
}

func (s *AuditExportService) shouldIncludeDataType(dataTypes []string, dataType string) bool {
	if len(dataTypes) == 0 {
		return true // Include all if none specified
	}

	for _, dt := range dataTypes {
		if dt == dataType {
			return true
		}
	}
	return false
}

func getSeverityForDecision(decision string) string {
	switch decision {
	case "deny":
		return "high"
	case "allow":
		return "low"
	default:
		return "medium"
	}
}

func stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func int64PtrToString(ptr *int64) string {
	if ptr == nil {
		return ""
	}
	return fmt.Sprintf("%d", *ptr)
}

func timePtrToString(ptr *time.Time) string {
	if ptr == nil {
		return ""
	}
	return ptr.Format(time.RFC3339)
}
