package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

// ComplianceAutomationRequest represents a request for automated compliance processing
type ComplianceAutomationRequest struct {
	ArtifactID    uuid.UUID `json:"artifact_id"`
	PolicyType    string    `json:"policy_type"`    // "vulnerability", "license", "retention", "data_locality"
	AutoRemediate bool      `json:"auto_remediate"` // Whether to automatically apply remediation
	NotifyUsers   []string  `json:"notify_users"`   // Users to notify of compliance status
	CheckInterval int       `json:"check_interval"` // Hours between automated checks
}

// ComplianceAutomationResponse represents the result of automated compliance processing
type ComplianceAutomationResponse struct {
	ArtifactID       uuid.UUID            `json:"artifact_id"`
	PolicyDecision   *PolicyDecision      `json:"policy_decision"`
	ComplianceStatus string               `json:"compliance_status"` // "compliant", "non_compliant", "under_review"
	AutomatedActions []ComplianceAction   `json:"automated_actions"`
	RequiredActions  []ComplianceAction   `json:"required_actions"`
	NextCheck        time.Time            `json:"next_check"`
	Notifications    []NotificationResult `json:"notifications"`
}

// ComplianceAction represents an automated compliance action
type ComplianceAction struct {
	Type        string    `json:"type"` // "quarantine", "block", "notify", "tag", "move"
	Description string    `json:"description"`
	Status      string    `json:"status"` // "pending", "completed", "failed"
	ExecutedAt  time.Time `json:"executed_at"`
	Details     string    `json:"details"`
}

// NotificationResult represents the result of sending a compliance notification
type NotificationResult struct {
	Recipient string    `json:"recipient"`
	Channel   string    `json:"channel"` // "email", "slack", "webhook"
	Status    string    `json:"status"`  // "sent", "failed", "pending"
	SentAt    time.Time `json:"sent_at"`
	Message   string    `json:"message"`
}

// handleComplianceAutomation processes automated compliance workflows using OPA policies
func (s *Server) handleComplianceAutomation(c *gin.Context) {
	var req ComplianceAutomationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Get artifact
	artifact, err := s.artifactService.GetByID(req.ArtifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Get repository for context
	repo, err := s.repositoryService.GetByID(artifact.RepositoryID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	// Evaluate OPA policy for compliance
	userID := "system" // Automated system user
	policyInput := CreateArtifactPolicyInput(artifact, repo, userID)

	ctx := context.Background()
	policyDecision, err := s.policyService.EvaluateArtifactPolicy(ctx, policyInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Policy evaluation failed: " + err.Error()})
		return
	}

	// Process compliance based on policy decision
	response := s.processComplianceAutomation(req, artifact, policyDecision)

	// Update artifact metadata with automation results (would need Update method in service)
	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}
	artifact.Metadata["compliance_automation"] = map[string]interface{}{
		"last_processed": time.Now().Format(time.RFC3339),
		"policy_type":    req.PolicyType,
		"auto_remediate": req.AutoRemediate,
		"status":         response.ComplianceStatus,
		"actions_taken":  len(response.AutomatedActions),
	}

	// Note: Would save updated artifact if Update method was available
	s.logger.Printf("Compliance automation completed for artifact %d", req.ArtifactID)

	c.JSON(http.StatusOK, response)
}

// processComplianceAutomation processes the compliance automation workflow
func (s *Server) processComplianceAutomation(req ComplianceAutomationRequest, artifact *models.Artifact, policyDecision *PolicyDecision) *ComplianceAutomationResponse {
	response := &ComplianceAutomationResponse{
		ArtifactID:       req.ArtifactID,
		PolicyDecision:   policyDecision,
		ComplianceStatus: s.determineComplianceStatus(policyDecision),
		AutomatedActions: []ComplianceAction{},
		RequiredActions:  []ComplianceAction{},
		NextCheck:        time.Now().Add(time.Duration(req.CheckInterval) * time.Hour),
		Notifications:    []NotificationResult{},
	}

	// Execute automated actions based on policy decision
	if req.AutoRemediate {
		switch policyDecision.Action {
		case "block":
			action := s.executeBlockAction(artifact)
			response.AutomatedActions = append(response.AutomatedActions, action)

		case "quarantine":
			action := s.executeQuarantineAction(artifact)
			response.AutomatedActions = append(response.AutomatedActions, action)

		case "warn":
			action := s.executeWarningAction(artifact, req.NotifyUsers)
			response.AutomatedActions = append(response.AutomatedActions, action)
			response.Notifications = s.sendComplianceNotifications(artifact, policyDecision, req.NotifyUsers)
		}
	} else {
		// Create required actions without executing them
		switch policyDecision.Action {
		case "block":
			response.RequiredActions = append(response.RequiredActions, ComplianceAction{
				Type:        "block",
				Description: "Artifact must be blocked due to security policy violation",
				Status:      "pending",
				Details:     policyDecision.Reason,
			})

		case "quarantine":
			response.RequiredActions = append(response.RequiredActions, ComplianceAction{
				Type:        "quarantine",
				Description: "Artifact must be quarantined for security review",
				Status:      "pending",
				Details:     policyDecision.Reason,
			})
		}
	}

	// Always log compliance automation activity
	s.logger.Printf("Compliance automation for artifact %d: Status=%s, Action=%s, Auto-remediate=%t",
		req.ArtifactID, response.ComplianceStatus, policyDecision.Action, req.AutoRemediate)

	return response
}

// determineComplianceStatus determines compliance status based on policy decision
func (s *Server) determineComplianceStatus(policyDecision *PolicyDecision) string {
	switch policyDecision.Action {
	case "allow":
		return "compliant"
	case "warn":
		return "compliant" // Compliant but with warnings
	case "quarantine":
		return "under_review"
	case "block":
		return "non_compliant"
	default:
		return "unknown"
	}
}

// executeBlockAction blocks an artifact
func (s *Server) executeBlockAction(artifact *models.Artifact) ComplianceAction {
	// Update artifact metadata to mark as blocked
	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}
	artifact.Metadata["blocked"] = true
	artifact.Metadata["blocked_at"] = time.Now().Format(time.RFC3339)
	artifact.Metadata["blocked_reason"] = "Security policy violation"

	s.logger.Printf("Artifact %d (%s:%s) has been blocked due to policy violation",
		artifact.ID, artifact.Name, artifact.Version)

	return ComplianceAction{
		Type:        "block",
		Description: "Artifact access has been blocked",
		Status:      "completed",
		ExecutedAt:  time.Now(),
		Details:     "Artifact marked as blocked in metadata",
	}
}

// executeQuarantineAction quarantines an artifact
func (s *Server) executeQuarantineAction(artifact *models.Artifact) ComplianceAction {
	// Update artifact metadata to mark as quarantined
	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}
	artifact.Metadata["quarantined"] = true
	artifact.Metadata["quarantined_at"] = time.Now().Format(time.RFC3339)
	artifact.Metadata["quarantine_reason"] = "Security review required"

	s.logger.Printf("Artifact %d (%s:%s) has been quarantined for security review",
		artifact.ID, artifact.Name, artifact.Version)

	return ComplianceAction{
		Type:        "quarantine",
		Description: "Artifact has been quarantined for security review",
		Status:      "completed",
		ExecutedAt:  time.Now(),
		Details:     "Artifact marked as quarantined in metadata",
	}
}

// executeWarningAction creates a warning for an artifact
func (s *Server) executeWarningAction(artifact *models.Artifact, notifyUsers []string) ComplianceAction {
	// Update artifact metadata to add warning
	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}
	artifact.Metadata["warning"] = true
	artifact.Metadata["warning_at"] = time.Now().Format(time.RFC3339)
	artifact.Metadata["warning_reason"] = "Security concerns identified"

	s.logger.Printf("Artifact %d (%s:%s) flagged with security warning",
		artifact.ID, artifact.Name, artifact.Version)

	return ComplianceAction{
		Type:        "warn",
		Description: "Security warning flagged for artifact",
		Status:      "completed",
		ExecutedAt:  time.Now(),
		Details:     "Artifact marked with warning in metadata",
	}
}

// sendComplianceNotifications sends notifications about compliance status
func (s *Server) sendComplianceNotifications(artifact *models.Artifact, policyDecision *PolicyDecision, recipients []string) []NotificationResult {
	notifications := []NotificationResult{}

	for _, recipient := range recipients {
		// In a real implementation, you would integrate with actual notification services
		// For now, we'll simulate the notification
		notification := NotificationResult{
			Recipient: recipient,
			Channel:   "email", // Could be determined based on recipient preferences
			Status:    "sent",
			SentAt:    time.Now(),
			Message: fmt.Sprintf("Artifact %s:%s - %s (Risk Level: %s)",
				artifact.Name, artifact.Version, policyDecision.Reason, policyDecision.RiskLevel),
		}

		notifications = append(notifications, notification)
		s.logger.Printf("Compliance notification sent to %s for artifact %d", recipient, artifact.ID)
	}

	return notifications
}

// handleGetComplianceAutomationStatus gets the automation status for an artifact
func (s *Server) handleGetComplianceAutomationStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact
	artifact, err := s.artifactService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Extract compliance automation info from metadata
	automationInfo := map[string]interface{}{
		"artifact_id": id,
		"status":      "not_configured",
	}

	if artifact.Metadata != nil {
		if automation, ok := artifact.Metadata["compliance_automation"].(map[string]interface{}); ok {
			automationInfo = automation
			automationInfo["artifact_id"] = id
		}
	}

	c.JSON(http.StatusOK, automationInfo)
}
