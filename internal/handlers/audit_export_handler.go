package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/securestor/securestor/internal/service"
)

// AuditExportHandler handles audit data export requests
type AuditExportHandler struct {
	exportSvc *service.AuditExportService
}

// NewAuditExportHandler creates a new audit export handler
func NewAuditExportHandler(exportSvc *service.AuditExportService) *AuditExportHandler {
	return &AuditExportHandler{
		exportSvc: exportSvc,
	}
}

// ExportAuditData handles POST /api/admin/audit/export
func (h *AuditExportHandler) ExportAuditData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// TODO: Add admin permission check

	// Parse request body
	var exportReq service.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&exportReq); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body")
		return
	}

	// Set default values
	if exportReq.MaxRecords == 0 {
		exportReq.MaxRecords = 10000
	}

	if len(exportReq.DataTypes) == 0 {
		exportReq.DataTypes = []string{"policy_decisions", "violations"}
	}

	// Set response headers based on format
	filename := fmt.Sprintf("audit_export_%s", time.Now().Format("20060102_150405"))

	switch exportReq.Format {
	case service.FormatJSON:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))
	case service.FormatCSV:
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))
	case service.FormatSIEM:
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.jsonl", filename))
	case service.FormatXML:
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xml", filename))
	default:
		writeJSONError(w, http.StatusBadRequest, "invalid_format", "Unsupported export format")
		return
	}

	// Perform export
	result, err := h.exportSvc.ExportAuditData(r.Context(), &exportReq, w)
	if err != nil {
		// Reset headers and write error response
		w.Header().Del("Content-Disposition")
		w.Header().Set("Content-Type", "application/json")
		writeJSONError(w, http.StatusInternalServerError, "export_failed", err.Error())
		return
	}

	// If we get here, the export was successful and data was written to the response
	// The result contains metadata about the export, but since we're streaming
	// the data directly to the response, we can't return the metadata
	_ = result
}

// GetExportFormats handles GET /api/admin/audit/export/formats
func (h *AuditExportHandler) GetExportFormats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	formats := map[string]interface{}{
		"formats": []map[string]interface{}{
			{
				"format":      "json",
				"name":        "JSON",
				"description": "JavaScript Object Notation format",
				"mime_type":   "application/json",
				"extension":   ".json",
			},
			{
				"format":      "csv",
				"name":        "CSV",
				"description": "Comma-Separated Values format",
				"mime_type":   "text/csv",
				"extension":   ".csv",
			},
			{
				"format":      "siem",
				"name":        "SIEM (JSON Lines)",
				"description": "SIEM-compatible JSON Lines format",
				"mime_type":   "application/x-ndjson",
				"extension":   ".jsonl",
			},
			{
				"format":      "xml",
				"name":        "XML",
				"description": "Extensible Markup Language format",
				"mime_type":   "application/xml",
				"extension":   ".xml",
			},
		},
		"data_types": []map[string]interface{}{
			{
				"type":        "policy_decisions",
				"name":        "Policy Decisions",
				"description": "OPA policy authorization decisions",
			},
			{
				"type":        "violations",
				"name":        "Security Violations",
				"description": "Security policy violations and incidents",
			},
			{
				"type":        "mfa_attempts",
				"name":        "MFA Attempts",
				"description": "Multi-factor authentication attempts",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(formats)
}

// PreviewExport handles POST /api/admin/audit/export/preview
func (h *AuditExportHandler) PreviewExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// Parse request body
	var exportReq service.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&exportReq); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON in request body")
		return
	}

	// Limit preview to small number of records
	exportReq.MaxRecords = 100
	exportReq.Format = service.FormatJSON // Always return JSON for preview

	// Create a buffer to capture the export output
	var buffer strings.Builder

	result, err := h.exportSvc.ExportAuditData(r.Context(), &exportReq, &buffer)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "preview_failed", err.Error())
		return
	}

	// Parse the exported JSON to return as preview
	var previewData interface{}
	if err := json.Unmarshal([]byte(buffer.String()), &previewData); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "preview_parse_failed", "Failed to parse preview data")
		return
	}

	response := map[string]interface{}{
		"preview": previewData,
		"metadata": map[string]interface{}{
			"record_count":    result.RecordCount,
			"export_format":   result.Format,
			"estimated_size":  len(buffer.String()),
			"preview_limited": true,
			"max_records":     exportReq.MaxRecords,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetExportSchema handles GET /api/admin/audit/export/schema/{data_type}
func (h *AuditExportHandler) GetExportSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// Extract data type from URL path
	dataType := r.URL.Path[len("/api/admin/audit/export/schema/"):]

	var schema map[string]interface{}

	switch dataType {
	case "policy_decisions":
		schema = map[string]interface{}{
			"fields": []map[string]interface{}{
				{"name": "id", "type": "integer", "description": "Unique identifier"},
				{"name": "request_id", "type": "string", "description": "Request identifier"},
				{"name": "user_id", "type": "integer", "description": "User identifier"},
				{"name": "resource", "type": "string", "description": "Resource being accessed"},
				{"name": "action", "type": "string", "description": "Action being performed"},
				{"name": "decision", "type": "string", "description": "Policy decision (allow/deny)"},
				{"name": "policy_name", "type": "string", "description": "Name of the policy used"},
				{"name": "execution_time_ms", "type": "integer", "description": "Execution time in milliseconds"},
				{"name": "ip_address", "type": "string", "description": "Client IP address"},
				{"name": "user_agent", "type": "string", "description": "Client user agent"},
				{"name": "session_id", "type": "string", "description": "Session identifier"},
				{"name": "created_at", "type": "datetime", "description": "Timestamp of the decision"},
			},
		}
	case "violations":
		schema = map[string]interface{}{
			"fields": []map[string]interface{}{
				{"name": "id", "type": "integer", "description": "Unique identifier"},
				{"name": "audit_log_id", "type": "integer", "description": "Related audit log ID"},
				{"name": "violation_type", "type": "string", "description": "Type of security violation"},
				{"name": "severity", "type": "string", "description": "Severity level (low/medium/high/critical)"},
				{"name": "description", "type": "string", "description": "Violation description"},
				{"name": "is_investigated", "type": "boolean", "description": "Whether violation was investigated"},
				{"name": "investigated_by", "type": "integer", "description": "User who investigated"},
				{"name": "investigated_at", "type": "datetime", "description": "Investigation timestamp"},
				{"name": "notes", "type": "string", "description": "Investigation notes"},
				{"name": "created_at", "type": "datetime", "description": "Violation timestamp"},
			},
		}
	case "mfa_attempts":
		schema = map[string]interface{}{
			"fields": []map[string]interface{}{
				{"name": "id", "type": "integer", "description": "Unique identifier"},
				{"name": "user_id", "type": "integer", "description": "User identifier"},
				{"name": "attempt_type", "type": "string", "description": "Type of MFA attempt (totp/webauthn)"},
				{"name": "success", "type": "boolean", "description": "Whether attempt was successful"},
				{"name": "ip_address", "type": "string", "description": "Client IP address"},
				{"name": "user_agent", "type": "string", "description": "Client user agent"},
				{"name": "created_at", "type": "datetime", "description": "Attempt timestamp"},
			},
		}
	default:
		writeJSONError(w, http.StatusNotFound, "schema_not_found", fmt.Sprintf("Schema for data type '%s' not found", dataType))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// GetExportHistory handles GET /api/admin/audit/export/history
func (h *AuditExportHandler) GetExportHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET method is allowed")
		return
	}

	// Get pagination parameters
	page := 1
	limit := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// For now, return placeholder data
	// In a real implementation, you'd fetch export history from the database
	history := []map[string]interface{}{
		{
			"id":           "export_1703123456789",
			"format":       "json",
			"data_types":   []string{"policy_decisions", "violations"},
			"record_count": 1250,
			"file_size":    2048576,
			"status":       "completed",
			"created_at":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"completed_at": time.Now().Add(-2*time.Hour + 30*time.Second).Format(time.RFC3339),
		},
		{
			"id":           "export_1703120000000",
			"format":       "csv",
			"data_types":   []string{"policy_decisions"},
			"record_count": 500,
			"file_size":    512000,
			"status":       "completed",
			"created_at":   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"completed_at": time.Now().Add(-24*time.Hour + 45*time.Second).Format(time.RFC3339),
		},
	}

	response := map[string]interface{}{
		"exports": history,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       len(history),
			"total_pages": 1,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
