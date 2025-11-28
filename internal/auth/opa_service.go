package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// OPAService handles Open Policy Agent integration for ABAC
type OPAService struct {
	db       *sql.DB
	opaURL   string
	logger   *logger.Logger
	auditSvc *service.AuditService
}

// NewOPAService creates a new OPA service instance
func NewOPAService(db *sql.DB, opaURL string, log *logger.Logger, auditSvc *service.AuditService) *OPAService {
	return &OPAService{
		db:       db,
		opaURL:   opaURL,
		logger:   log,
		auditSvc: auditSvc,
	}
}

// OPADecisionRequest represents a request to OPA for policy decision
type OPADecisionRequest struct {
	Input OPAInput `json:"input"`
}

// OPAInput represents the input data for OPA policy evaluation
type OPAInput struct {
	User     OPAUser     `json:"user"`
	Resource OPAResource `json:"resource"`
	Action   string      `json:"action"`
	Context  OPAContext  `json:"context"`
}

// OPAUser represents user information for OPA
type OPAUser struct {
	ID         string                 `json:"id"`
	Username   string                 `json:"username"`
	Email      string                 `json:"email"`
	Roles      []string               `json:"roles"`
	Attributes map[string]interface{} `json:"attributes"`
}

// OPAResource represents resource information for OPA
type OPAResource struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Path       string                 `json:"path"`
	Attributes map[string]interface{} `json:"attributes"`
}

// OPAContext represents contextual information for policy decisions
type OPAContext struct {
	Time      time.Time              `json:"time"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	SessionID string                 `json:"session_id"`
	RequestID string                 `json:"request_id"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// OPADecisionResponse represents OPA's decision response
type OPADecisionResponse struct {
	Result OPAResult `json:"result"`
}

// OPAResult represents the result of OPA policy evaluation
type OPAResult struct {
	Allow       bool                   `json:"allow"`
	Reason      string                 `json:"reason,omitempty"`
	PolicyName  string                 `json:"policy_name,omitempty"`
	Obligations []string               `json:"obligations,omitempty"`
	AuditLevel  string                 `json:"audit_level,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Authorize makes a policy decision using OPA
func (s *OPAService) Authorize(ctx context.Context, authContext *models.AuthContext, resource, action string, resourceAttribs map[string]interface{}) (*OPAResult, error) {
	startTime := time.Now()

	// Build OPA input
	input := s.buildOPAInput(authContext, resource, action, resourceAttribs, ctx)

	// Make decision request using HTTP client
	client := &http.Client{Timeout: 5 * time.Second}
	decision, err := s.makeDecisionRequest(ctx, client, input)
	if err != nil {
		s.logger.Error("OPA decision request failed", err)
		// Fall back to default deny
		decision = &OPAResult{
			Allow:      false,
			Reason:     "OPA service unavailable",
			PolicyName: "default_deny",
		}
	}

	// Log the decision
	executionTime := time.Since(startTime).Milliseconds()
	err = s.logPolicyDecision(authContext, input, decision, executionTime)
	if err != nil {
		s.logger.Error("Failed to log policy decision", err)
	}

	// Check for policy violations
	if !decision.Allow {
		s.checkForViolations(authContext, input, decision)
	}

	return decision, nil
}

// AuthorizeWithPolicy makes a policy decision using a specific policy
func (s *OPAService) AuthorizeWithPolicy(ctx context.Context, policyName string, authContext *models.AuthContext, resource, action string, resourceAttribs map[string]interface{}) (*OPAResult, error) {
	// Get policy from database
	policy, err := s.GetPolicy(policyName)
	if err != nil {
		return nil, fmt.Errorf("policy not found: %w", err)
	}

	if !policy.IsActive {
		return &OPAResult{
			Allow:      false,
			Reason:     "Policy is inactive",
			PolicyName: policyName,
		}, nil
	}

	// Use specific policy endpoint
	input := s.buildOPAInput(authContext, resource, action, resourceAttribs, ctx)
	decision, err := s.makeDecisionRequestWithPolicy(ctx, input, policyName)
	if err != nil {
		return nil, err
	}

	return decision, nil
}

// buildOPAInput builds the input structure for OPA
func (s *OPAService) buildOPAInput(authContext *models.AuthContext, resource, action string, resourceAttribs map[string]interface{}, ctx context.Context) *OPAInput {
	// Get user attributes (could be enhanced with database lookup)
	userAttributes := make(map[string]interface{})

	// Add basic user attributes
	if authContext.User != nil {
		userAttributes["department"] = "engineering"   // This would come from user profile
		userAttributes["clearance_level"] = "internal" // This would come from user profile
		userAttributes["employment_type"] = "full_time"
	}

	// Get resource attributes
	if resourceAttribs == nil {
		resourceAttribs = make(map[string]interface{})
	}

	// Add default resource attributes
	resourceAttribs["classification"] = "internal"
	resourceAttribs["sensitive"] = false

	// Extract context information
	requestID := ""
	ipAddress := ""
	userAgent := ""
	sessionID := authContext.SessionID

	// These would typically come from the HTTP request context
	if ctx != nil {
		if val := ctx.Value("request_id"); val != nil {
			requestID = val.(string)
		}
		if val := ctx.Value("ip_address"); val != nil {
			ipAddress = val.(string)
		}
		if val := ctx.Value("user_agent"); val != nil {
			userAgent = val.(string)
		}
	}

	return &OPAInput{
		User: OPAUser{
			ID:         authContext.UserID,
			Username:   authContext.Username,
			Email:      authContext.Email,
			Roles:      authContext.Roles,
			Attributes: userAttributes,
		},
		Resource: OPAResource{
			ID:         resource,
			Type:       s.getResourceType(resource),
			Path:       resource,
			Attributes: resourceAttribs,
		},
		Action: action,
		Context: OPAContext{
			Time:      time.Now(),
			IPAddress: ipAddress,
			UserAgent: userAgent,
			SessionID: sessionID,
			RequestID: requestID,
		},
	}
}

// makeDecisionRequest makes a request to OPA for policy decision
func (s *OPAService) makeDecisionRequest(ctx context.Context, client *http.Client, input *OPAInput) (*OPAResult, error) {
	request := OPADecisionRequest{Input: *input}

	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request to OPA's data API
	url := fmt.Sprintf("%s/v1/data/securestor/rbac", s.opaURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA returned status %d", resp.StatusCode)
	}

	var opaResp OPADecisionResponse
	err = json.NewDecoder(resp.Body).Decode(&opaResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &opaResp.Result, nil
}

// makeDecisionRequestWithPolicy makes a request using a specific policy
func (s *OPAService) makeDecisionRequestWithPolicy(ctx context.Context, input *OPAInput, policyName string) (*OPAResult, error) {
	// This would typically use OPA's policy-specific endpoints
	// For now, we'll use the general endpoint and add policy name to input
	requestBody := OPADecisionRequest{Input: *input}
	requestBody.Input.Context.Extra = map[string]interface{}{
		"policy_name": policyName,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	return s.makeDecisionRequest(ctx, client, input)
}

// logPolicyDecision logs a policy decision to the audit log
func (s *OPAService) logPolicyDecision(authContext *models.AuthContext, input *OPAInput, decision *OPAResult, executionTimeMs int64) error {
	inputJSON, _ := json.Marshal(input)
	outputJSON, _ := json.Marshal(decision)

	var userID *int64
	if authContext.UserID != "" {
		if id, err := parseUserID(authContext.UserID); err == nil {
			userID = &id
		}
	}

	decisionStr := "deny"
	if decision.Allow {
		decisionStr = "allow"
	}

	query := `
		INSERT INTO policy_audit_log 
		(request_id, user_id, resource, action, decision, policy_name, 
		 input_data, policy_output, execution_time_ms, ip_address, 
		 user_agent, session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := s.db.Exec(query,
		input.Context.RequestID, userID, input.Resource.Path, input.Action,
		decisionStr, decision.PolicyName, inputJSON, outputJSON,
		executionTimeMs, input.Context.IPAddress, input.Context.UserAgent,
		input.Context.SessionID,
	)

	return err
}

// checkForViolations checks if a denied request should be flagged as a violation
func (s *OPAService) checkForViolations(authContext *models.AuthContext, input *OPAInput, decision *OPAResult) {
	// Determine if this should be flagged as a security violation
	severity := s.determineSeverity(input, decision)
	if severity == "" {
		return // Not a violation
	}

	// Get the audit log ID
	auditLogID, err := s.getLatestAuditLogID(input.Context.RequestID)
	if err != nil {
		s.logger.Error("Failed to get audit log ID for violation", err)
		return
	}

	// Create violation record
	violationType := s.determineViolationType(input, decision)
	description := fmt.Sprintf("Access denied: %s attempted %s on %s",
		input.User.Username, input.Action, input.Resource.Path)

	query := `
		INSERT INTO policy_violations 
		(audit_log_id, violation_type, severity, description)
		VALUES ($1, $2, $3, $4)
	`

	_, err = s.db.Exec(query, auditLogID, violationType, severity, description)
	if err != nil {
		s.logger.Error("Failed to log policy violation", err)
	}
}

// Helper functions

func (s *OPAService) getResourceType(resource string) string {
	// Extract resource type from path
	if len(resource) > 0 && resource[0] == '/' {
		parts := splitPath(resource)
		if len(parts) > 2 {
			return parts[2] // /api/artifacts -> artifacts
		}
	}
	return "unknown"
}

func (s *OPAService) determineSeverity(input *OPAInput, decision *OPAResult) string {
	// Determine violation severity based on various factors
	if contains(input.User.Roles, "admin") {
		return "high" // Admin access denied is suspicious
	}

	if input.Action == "delete" {
		return "medium"
	}

	if input.Resource.Attributes["sensitive"] == true {
		return "medium"
	}

	return "low"
}

func (s *OPAService) determineViolationType(input *OPAInput, decision *OPAResult) string {
	if contains(input.User.Roles, "admin") {
		return "admin_access_denied"
	}

	if input.Action == "delete" {
		return "unauthorized_deletion_attempt"
	}

	return "unauthorized_access"
}

func (s *OPAService) getLatestAuditLogID(requestID string) (int64, error) {
	var id int64
	query := `SELECT id FROM policy_audit_log WHERE request_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := s.db.QueryRow(query, requestID).Scan(&id)
	return id, err
}

func parseUserID(userIDStr string) (int64, error) {
	// Implementation depends on how user IDs are formatted
	// This is a placeholder
	return 0, fmt.Errorf("not implemented")
}

func splitPath(path string) []string {
	// Simple path splitting implementation
	return []string{} // Placeholder
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetPolicy retrieves a policy by name
func (s *OPAService) GetPolicy(name string) (*models.OPAPolicy, error) {
	query := `
		SELECT id, name, description, policy_type, rego_policy, version, 
		       is_active, created_by, created_at, updated_at, activated_at
		FROM opa_policies 
		WHERE name = $1
	`

	policy := &models.OPAPolicy{}
	err := s.db.QueryRow(query, name).Scan(
		&policy.ID, &policy.Name, &policy.Description, &policy.PolicyType,
		&policy.RegoPolicy, &policy.Version, &policy.IsActive, &policy.CreatedBy,
		&policy.CreatedAt, &policy.UpdatedAt, &policy.ActivatedAt,
	)

	if err != nil {
		return nil, err
	}

	return policy, nil
}
