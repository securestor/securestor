package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SimpleTenantConfigService provides basic tenant configuration
type SimpleTenantConfigService struct {
	tenantConfigs map[string]*TenantConfig
	logger        Logger
}

// TenantConfig represents complete tenant configuration
type TenantConfig struct {
	TenantID          string                           `json:"tenant_id"`
	Name              string                           `json:"name"`
	WorkflowOverrides map[string]*TenantWorkflowConfig `json:"workflow_overrides"`
	DefaultSettings   map[string]interface{}           `json:"default_settings"`
	Enabled           bool                             `json:"enabled"`
}

// NewSimpleTenantConfigService creates a new tenant config service
func NewSimpleTenantConfigService(logger Logger) *SimpleTenantConfigService {
	service := &SimpleTenantConfigService{
		tenantConfigs: make(map[string]*TenantConfig),
		logger:        logger,
	}

	// Initialize with default tenant configurations
	service.initializeDefaultConfigs()

	return service
}

// initializeDefaultConfigs sets up default tenant configurations
func (s *SimpleTenantConfigService) initializeDefaultConfigs() {
	// Default tenant with all workflows enabled
	defaultTenant := &TenantConfig{
		TenantID: "default",
		Name:     "Default Tenant",
		WorkflowOverrides: map[string]*TenantWorkflowConfig{
			"DockerWorkflow": {
				Enabled:  true,
				Scanners: []string{"syft", "grype", "trivy"},
				Execution: ExecutionConfig{
					Strategy:      "parallel",
					FailurePolicy: "continue",
				},
			},
			"PythonWorkflow": {
				Enabled:  true,
				Scanners: []string{"syft", "bandit", "grype"},
				Execution: ExecutionConfig{
					Strategy:      "parallel",
					FailurePolicy: "continue",
				},
			},
		},
		Enabled: true,
	}

	// Example enterprise tenant with custom configuration
	enterpriseTenant := &TenantConfig{
		TenantID: "acme-inc",
		Name:     "ACME Corporation",
		WorkflowOverrides: map[string]*TenantWorkflowConfig{
			"DockerWorkflow": {
				Enabled:  true,
				Scanners: []string{"syft", "grype"}, // Trivy disabled for this tenant
				Execution: ExecutionConfig{
					Strategy:      "sequential", // Different execution strategy
					FailurePolicy: "stop_on_critical",
				},
				PolicyPath: "/v1/data/acme/docker_policy", // Custom policy
			},
			"PythonWorkflow": {
				Enabled: false, // Python scanning disabled
			},
		},
		DefaultSettings: map[string]interface{}{
			"notification_channels": []string{"slack", "email"},
			"scan_timeout":          "10m",
			"max_concurrent_scans":  3,
		},
		Enabled: true,
	}

	s.tenantConfigs["default"] = defaultTenant
	s.tenantConfigs["acme-inc"] = enterpriseTenant

	s.logger.Printf("[TENANT_CONFIG] Initialized %d tenant configurations", len(s.tenantConfigs))
}

// GetTenantWorkflowConfig returns tenant-specific workflow configuration
func (s *SimpleTenantConfigService) GetTenantWorkflowConfig(tenantID string, workflowName string) (*TenantWorkflowConfig, error) {
	tenant, exists := s.tenantConfigs[tenantID]
	if !exists {
		// Fall back to default tenant
		tenant, exists = s.tenantConfigs["default"]
		if !exists {
			return nil, fmt.Errorf("no configuration found for tenant %s", tenantID)
		}
	}

	if !tenant.Enabled {
		return nil, fmt.Errorf("tenant %s is disabled", tenantID)
	}

	workflowConfig, exists := tenant.WorkflowOverrides[workflowName]
	if !exists {
		// Return default enabled config if no override
		return &TenantWorkflowConfig{Enabled: true}, nil
	}

	return workflowConfig, nil
}

// IsWorkflowEnabledForTenant checks if a workflow is enabled for a tenant
func (s *SimpleTenantConfigService) IsWorkflowEnabledForTenant(tenantID string, workflowName string) bool {
	config, err := s.GetTenantWorkflowConfig(tenantID, workflowName)
	if err != nil {
		s.logger.Printf("[TENANT_CONFIG] Error getting config for tenant %s workflow %s: %v", tenantID, workflowName, err)
		return false
	}

	return config.Enabled
}

// AddTenant adds a new tenant configuration
func (s *SimpleTenantConfigService) AddTenant(tenant *TenantConfig) {
	s.tenantConfigs[tenant.TenantID] = tenant
	s.logger.Printf("[TENANT_CONFIG] Added tenant: %s (%s)", tenant.TenantID, tenant.Name)
}

// SimpleNotificationService provides basic notification functionality
type SimpleNotificationService struct {
	logger Logger
}

// NewSimpleNotificationService creates a new notification service
func NewSimpleNotificationService(logger Logger) *SimpleNotificationService {
	return &SimpleNotificationService{
		logger: logger,
	}
}

// SendViolationNotification sends a policy violation notification
func (s *SimpleNotificationService) SendViolationNotification(ctx context.Context, notification ViolationNotification) error {
	s.logger.Printf("[NOTIFICATION] Policy violation detected for tenant %s, artifact %s",
		notification.TenantID, notification.ArtifactID)

	// Build notification message
	message := s.buildNotificationMessage(notification)

	// Send to configured channels
	for _, channel := range notification.Channels {
		if err := s.sendToChannel(ctx, channel, message, notification); err != nil {
			s.logger.Printf("[NOTIFICATION] Failed to send to %s: %v", channel, err)
			continue
		}
		s.logger.Printf("[NOTIFICATION] Sent violation notification to %s", channel)
	}

	return nil
}

// buildNotificationMessage creates a human-readable notification message
func (s *SimpleNotificationService) buildNotificationMessage(notification ViolationNotification) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("🚨 Security Policy Violation Detected\n\n"))
	builder.WriteString(fmt.Sprintf("**Tenant:** %s\n", notification.TenantID))
	builder.WriteString(fmt.Sprintf("**Artifact:** %s (%s)\n", notification.ArtifactID, notification.ArtifactType))
	builder.WriteString(fmt.Sprintf("**Workflow:** %s\n", notification.WorkflowName))
	builder.WriteString(fmt.Sprintf("**Policy Decision:** %s\n", notification.Decision.Action))
	builder.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", notification.Timestamp.Format("2006-01-02 15:04:05 UTC")))

	if len(notification.Violations) > 0 {
		builder.WriteString("**Violations:**\n")
		for i, violation := range notification.Violations {
			builder.WriteString(fmt.Sprintf("%d. **%s** (%s): %s\n",
				i+1, violation.Rule, violation.Severity, violation.Message))
		}
		builder.WriteString("\n")
	}

	if notification.ScanResults != nil {
		builder.WriteString("**Scan Summary:**\n")
		builder.WriteString(fmt.Sprintf("- Total Scanners: %d\n", notification.ScanResults.TotalScanners))
		builder.WriteString(fmt.Sprintf("- Successful Scans: %d\n", notification.ScanResults.SuccessfulScans))
		builder.WriteString(fmt.Sprintf("- Vulnerabilities Found: %d\n", len(notification.ScanResults.Vulnerabilities)))

		if notification.ScanResults.Summary.Critical > 0 {
			builder.WriteString(fmt.Sprintf("- 🔴 Critical: %d\n", notification.ScanResults.Summary.Critical))
		}
		if notification.ScanResults.Summary.High > 0 {
			builder.WriteString(fmt.Sprintf("- 🟠 High: %d\n", notification.ScanResults.Summary.High))
		}
		if notification.ScanResults.Summary.Medium > 0 {
			builder.WriteString(fmt.Sprintf("- 🟡 Medium: %d\n", notification.ScanResults.Summary.Medium))
		}
		if notification.ScanResults.Summary.Low > 0 {
			builder.WriteString(fmt.Sprintf("- 🟢 Low: %d\n", notification.ScanResults.Summary.Low))
		}
	}

	return builder.String()
}

// sendToChannel sends notification to a specific channel
func (s *SimpleNotificationService) sendToChannel(ctx context.Context, channel string, message string, notification ViolationNotification) error {
	switch channel {
	case "slack":
		return s.sendSlackNotification(ctx, message, notification)
	case "email":
		return s.sendEmailNotification(ctx, message, notification)
	case "webhook":
		return s.sendWebhookNotification(ctx, notification)
	default:
		s.logger.Printf("[NOTIFICATION] Unknown notification channel: %s", channel)
		return nil
	}
}

// sendSlackNotification sends notification to Slack (mock implementation)
func (s *SimpleNotificationService) sendSlackNotification(ctx context.Context, message string, notification ViolationNotification) error {
	// Mock Slack notification - in real implementation, would use Slack API
	s.logger.Printf("[NOTIFICATION] [SLACK] %s", message)
	return nil
}

// sendEmailNotification sends notification via email (mock implementation)
func (s *SimpleNotificationService) sendEmailNotification(ctx context.Context, message string, notification ViolationNotification) error {
	// Mock email notification - in real implementation, would use email service
	s.logger.Printf("[NOTIFICATION] [EMAIL] %s", message)
	return nil
}

// sendWebhookNotification sends notification via webhook (mock implementation)
func (s *SimpleNotificationService) sendWebhookNotification(ctx context.Context, notification ViolationNotification) error {
	// Mock webhook notification - in real implementation, would make HTTP request
	payload, _ := json.Marshal(notification)
	s.logger.Printf("[NOTIFICATION] [WEBHOOK] %s", string(payload))
	return nil
}

// OrchestrationExample demonstrates how everything works together
func OrchestrationExample(logger Logger) {
	logger.Printf("=== SecureStor Enhanced Orchestration Example ===\n")

	// 1. Initialize all components
	pluginManager := NewPluginManager(logger)
	scannerRegistry := NewScannerRegistry()
	workflowEngine := NewWorkflowEngine(pluginManager, logger)

	// Create integration manager
	integrationManager := NewPluginIntegrationManager(
		pluginManager,
		workflowEngine,
		scannerRegistry,
		logger,
	)

	// Initialize plugin system
	if err := integrationManager.Initialize(); err != nil {
		logger.Printf("Failed to initialize plugin system: %v", err)
		return
	}

	// Create services
	tenantConfig := NewSimpleTenantConfigService(logger)
	notificationService := NewSimpleNotificationService(logger)

	// Mock policy client and output store (would be real implementations)
	var policyClient PolicyClient = &MockPolicyClient{logger: logger}
	var outputStore OutputStore = &MockOutputStore{logger: logger}

	// 2. Create enhanced orchestrator
	orchestrator := NewEnhancedOrchestrator(
		pluginManager,
		workflowEngine,
		integrationManager,
		policyClient,
		outputStore,
		notificationService,
		tenantConfig,
		logger,
	)

	// 3. Simulate artifact upload and scanning
	ctx := context.Background()

	// Example 1: Docker image scan for default tenant
	dockerJob := EnhancedScanJob{
		JobID:        "job-001",
		ArtifactID:   "docker-image-123",
		ArtifactPath: "/path/to/docker/image.tar",
		ArtifactType: "docker",
		TenantID:     "default",
		Priority:     1,
		SubmittedAt:  time.Now(),
		RequestedBy:  "user@example.com",
	}

	logger.Printf("\n--- Example 1: Docker Image Scan (Default Tenant) ---")
	result1, err := orchestrator.ExecuteJob(ctx, dockerJob)
	if err != nil {
		logger.Printf("Docker scan failed: %v", err)
	} else {
		logger.Printf("Docker scan completed with status: %s", result1.Status)
	}

	// Example 2: Python package scan for enterprise tenant
	pythonJob := EnhancedScanJob{
		JobID:        "job-002",
		ArtifactID:   "python-package-456",
		ArtifactPath: "/path/to/python/package.tar.gz",
		ArtifactType: "python",
		TenantID:     "acme-inc",
		Priority:     2,
		SubmittedAt:  time.Now(),
		RequestedBy:  "admin@acme.com",
	}

	logger.Printf("\n--- Example 2: Python Package Scan (Enterprise Tenant) ---")
	result2, err := orchestrator.ExecuteJob(ctx, pythonJob)
	if err != nil {
		logger.Printf("Python scan failed: %v", err)
	} else {
		logger.Printf("Python scan completed with status: %s", result2.Status)
	}

	// 4. Show available workflows for different scenarios
	logger.Printf("\n--- Available Workflows ---")
	dockerWorkflows := orchestrator.ListAvailableWorkflows("default", "docker")
	logger.Printf("Docker workflows for default tenant: %d", len(dockerWorkflows))

	pythonWorkflows := orchestrator.ListAvailableWorkflows("acme-inc", "python")
	logger.Printf("Python workflows for acme-inc tenant: %d", len(pythonWorkflows))

	logger.Printf("\n=== Orchestration Example Complete ===")
}

// Mock implementations for testing
type MockPolicyClient struct {
	logger Logger
}

func (m *MockPolicyClient) Evaluate(ctx context.Context, input map[string]interface{}) (Decision, error) {
	// Mock policy evaluation - in real implementation, would call OPA
	m.logger.Printf("[POLICY] Evaluating policy for artifact: %v", input["artifact_id"])

	// Simulate policy decision based on vulnerabilities
	decision := Decision{
		Allow:  true,
		Action: "allow",
		Reason: "No policy violations detected",
	}

	// Mock: if artifact type is python and tenant is acme-inc, simulate block
	if input["artifact_type"] == "python" && input["tenant_id"] == "acme-inc" {
		decision.Allow = false
		decision.Action = "block"
		decision.Reason = "Python packages blocked for tenant acme-inc"
	}

	return decision, nil
}

type MockOutputStore struct {
	logger Logger
}

func (m *MockOutputStore) SaveScanResults(ctx context.Context, jobID string, results []ScannerResult) error {
	m.logger.Printf("[STORAGE] Saving %d scan results for job %s", len(results), jobID)
	return nil
}

func (m *MockOutputStore) MarkJobCompleted(ctx context.Context, jobID string, status string) error {
	m.logger.Printf("[STORAGE] Marking job %s as %s", jobID, status)
	return nil
}
