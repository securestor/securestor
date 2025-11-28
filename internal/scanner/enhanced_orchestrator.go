package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// EnhancedOrchestrator orchestrates the complete scanning workflow with plugin architecture
type EnhancedOrchestrator struct {
	pluginManager       *PluginManager
	workflowEngine      *WorkflowEngine
	integrationManager  *PluginIntegrationManager
	policyClient        PolicyClient
	outputStore         OutputStore
	notificationService NotificationService
	tenantConfig        TenantConfigService
	logger              Logger

	// Performance enhancement components
	performanceEnhancer   *PerformanceEnhancer
	enablePerformanceMode bool
}

// TenantConfigService provides tenant-specific configuration
type TenantConfigService interface {
	GetTenantWorkflowConfig(tenantID string, workflowName string) (*TenantWorkflowConfig, error)
	IsWorkflowEnabledForTenant(tenantID string, workflowName string) bool
}

// TenantWorkflowConfig represents tenant-specific workflow overrides
type TenantWorkflowConfig struct {
	Enabled    bool                   `json:"enabled"`
	Scanners   []string               `json:"scanners,omitempty"`
	Execution  ExecutionConfig        `json:"execution,omitempty"`
	PolicyPath string                 `json:"policy_path,omitempty"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
}

// NotificationService handles policy violation notifications
type NotificationService interface {
	SendViolationNotification(ctx context.Context, notification ViolationNotification) error
}

// ViolationNotification represents a policy violation notification
type ViolationNotification struct {
	TenantID     string                 `json:"tenant_id"`
	ArtifactID   string                 `json:"artifact_id"`
	ArtifactType string                 `json:"artifact_type"`
	WorkflowName string                 `json:"workflow_name"`
	PolicyPath   string                 `json:"policy_path"`
	Decision     PolicyDecision         `json:"decision"`
	Violations   []Violation            `json:"violations"`
	ScanResults  *AggregatedScanResult  `json:"scan_results"`
	Timestamp    time.Time              `json:"timestamp"`
	Channels     []string               `json:"channels"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyDecision extends the basic Decision with additional metadata
type PolicyDecision struct {
	Decision                          // Embed the basic decision
	PolicyPath string                 `json:"policy_path"`
	Violations []*Violation           `json:"violations,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Violation represents a policy violation
type Violation struct {
	Rule     string                 `json:"rule"`
	Severity string                 `json:"severity"`
	Message  string                 `json:"message"`
	Resource string                 `json:"resource"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// EnhancedScanJob represents a comprehensive scan job
type EnhancedScanJob struct {
	JobID        string                 `json:"job_id"`
	ArtifactID   string                 `json:"artifact_id"`
	ArtifactPath string                 `json:"artifact_path"`
	ArtifactType string                 `json:"artifact_type"`
	TenantID     string                 `json:"tenant_id"`
	WorkflowName string                 `json:"workflow_name,omitempty"` // Optional override
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SubmittedAt  time.Time              `json:"submitted_at"`
	RequestedBy  string                 `json:"requested_by,omitempty"`
}

// ScanJobResult represents the complete result of a scan job
type ScanJobResult struct {
	Job              EnhancedScanJob       `json:"job"`
	WorkflowID       string                `json:"workflow_id"`
	WorkflowResult   *WorkflowResult       `json:"workflow_result"`
	AggregatedResult *AggregatedScanResult `json:"aggregated_result"`
	PolicyDecision   *PolicyDecision       `json:"policy_decision"`
	Status           ScanJobStatus         `json:"status"`
	StartedAt        time.Time             `json:"started_at"`
	CompletedAt      time.Time             `json:"completed_at"`
	Duration         time.Duration         `json:"duration"`
	Errors           []string              `json:"errors,omitempty"`
}

type ScanJobStatus string

const (
	JobStatusPending               ScanJobStatus = "pending"
	JobStatusInProgress            ScanJobStatus = "in_progress"
	JobStatusCompleted             ScanJobStatus = "completed"
	JobStatusCompletedWithWarnings ScanJobStatus = "completed_with_warnings"
	JobStatusBlocked               ScanJobStatus = "blocked"
	JobStatusQuarantined           ScanJobStatus = "quarantined"
	JobStatusFailed                ScanJobStatus = "failed"
	JobStatusPolicyError           ScanJobStatus = "policy_error"
)

// NewEnhancedOrchestrator creates a new enhanced orchestrator
func NewEnhancedOrchestrator(
	pluginManager *PluginManager,
	workflowEngine *WorkflowEngine,
	integrationManager *PluginIntegrationManager,
	policyClient PolicyClient,
	outputStore OutputStore,
	notificationService NotificationService,
	tenantConfig TenantConfigService,
	logger Logger,
) *EnhancedOrchestrator {
	return &EnhancedOrchestrator{
		pluginManager:         pluginManager,
		workflowEngine:        workflowEngine,
		integrationManager:    integrationManager,
		policyClient:          policyClient,
		outputStore:           outputStore,
		notificationService:   notificationService,
		tenantConfig:          tenantConfig,
		logger:                logger,
		performanceEnhancer:   NewPerformanceEnhancer(DefaultPerformanceConfig()),
		enablePerformanceMode: false,
	}
}

// EnablePerformanceMode enables high-performance scanning with optimizations
func (eo *EnhancedOrchestrator) EnablePerformanceMode(config *PerformanceConfig) {
	if config == nil {
		config = DefaultPerformanceConfig()
	}
	eo.performanceEnhancer = NewPerformanceEnhancer(config)
	eo.enablePerformanceMode = true
	eo.logger.Printf("[ORCHESTRATOR] Performance mode enabled with %d max concurrent scans",
		config.MaxConcurrentScans)
}

// NewEnhancedOrchestratorWithRealServices creates an enhanced orchestrator with real service implementations
func NewEnhancedOrchestratorWithRealServices(
	pluginManager *PluginManager,
	workflowEngine *WorkflowEngine,
	integrationManager *PluginIntegrationManager,
	policyClient PolicyClient,
	outputStore OutputStore,
	logger Logger,
) *EnhancedOrchestrator {
	// Create real service implementations
	notificationService := NewSimpleNotificationService(logger)
	tenantConfig := NewSimpleTenantConfigService(logger)

	return NewEnhancedOrchestrator(
		pluginManager,
		workflowEngine,
		integrationManager,
		policyClient,
		outputStore,
		notificationService,
		tenantConfig,
		logger,
	)
} // ExecuteJob executes the complete scanning workflow as described
func (eo *EnhancedOrchestrator) ExecuteJob(ctx context.Context, job EnhancedScanJob) (*ScanJobResult, error) {
	logger := logrus.WithFields(logrus.Fields{
		"artifact_id":   job.ArtifactID,
		"artifact_type": job.ArtifactType,
	})

	result := &ScanJobResult{
		Job:       job,
		Status:    JobStatusInProgress,
		StartedAt: time.Now(),
		Errors:    make([]string, 0),
	}

	logger.Info("Starting enhanced scan job execution")

	// Step 1: Artifact uploaded → Orchestrator identifies type
	eo.logger.Printf("[ORCHESTRATOR] Step 1: Identified artifact type: %s", job.ArtifactType)

	// Step 2: Load workflow for artifact type
	workflow, err := eo.selectWorkflowForJob(job)
	if err != nil {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, fmt.Sprintf("Workflow selection failed: %v", err))
		eo.completeJob(result)
		return result, err
	}

	result.WorkflowID = workflow.ID
	eo.logger.Printf("[ORCHESTRATOR] Step 2: Selected workflow: %s", workflow.Name)

	// Step 3: Check if enabled for tenant
	if !eo.isWorkflowEnabledForTenant(job.TenantID, workflow.Name) {
		result.Status = JobStatusBlocked
		result.Errors = append(result.Errors, "Workflow disabled for tenant")
		eo.completeJob(result)
		return result, fmt.Errorf("workflow %s is disabled for tenant %s", workflow.Name, job.TenantID)
	}

	eo.logger.Printf("[ORCHESTRATOR] Step 3: Workflow enabled for tenant %s", job.TenantID)

	// Step 4: Execute scanners using workflow engine
	workflowResult, err := eo.executeWorkflow(ctx, job, workflow)
	if err != nil {
		result.Status = JobStatusFailed
		result.Errors = append(result.Errors, fmt.Sprintf("Workflow execution failed: %v", err))
		eo.completeJob(result)
		return result, err
	}

	result.WorkflowResult = workflowResult
	result.AggregatedResult = workflowResult.AggregatedResult
	eo.logger.Printf("[ORCHESTRATOR] Step 4: Workflow executed successfully with %d stage results",
		len(workflowResult.StageResults))

	// Step 5: Submit results to OPA for policy evaluation
	policyDecision, err := eo.evaluatePolicy(ctx, job, workflow, result.AggregatedResult)
	if err != nil {
		result.Status = JobStatusPolicyError
		result.Errors = append(result.Errors, fmt.Sprintf("Policy evaluation failed: %v", err))
		eo.completeJob(result)
		return result, err
	}

	result.PolicyDecision = policyDecision
	eo.logger.Printf("[ORCHESTRATOR] Step 5: Policy evaluation completed - Action: %s", policyDecision.Action)

	// Step 6: Enforcement decision and status update
	result.Status = eo.determineJobStatus(policyDecision)
	eo.logger.Printf("[ORCHESTRATOR] Step 6: Job status determined: %s", result.Status)

	// Step 7: Send notifications if policy violated
	if eo.shouldSendNotification(policyDecision, result.Status) {
		if err := eo.sendViolationNotification(ctx, job, workflow, policyDecision, result); err != nil {
			eo.logger.Printf("[ORCHESTRATOR] Warning: Failed to send notification: %v", err)
			result.Errors = append(result.Errors, fmt.Sprintf("Notification failed: %v", err))
		} else {
			eo.logger.Printf("[ORCHESTRATOR] Step 7: Violation notification sent")
		}
	}

	// Step 8: Store results in unified schema
	if err := eo.storeResults(ctx, result); err != nil {
		eo.logger.Printf("[ORCHESTRATOR] Warning: Failed to store results: %v", err)
		result.Errors = append(result.Errors, fmt.Sprintf("Storage failed: %v", err))
	} else {
		eo.logger.Printf("[ORCHESTRATOR] Step 8: Results stored successfully")
	}

	eo.completeJob(result)
	eo.logger.Printf("[ORCHESTRATOR] Scan job %s completed with status: %s", job.JobID, result.Status)

	return result, nil
}

// selectWorkflowForJob selects the appropriate workflow for the job
func (eo *EnhancedOrchestrator) selectWorkflowForJob(job EnhancedScanJob) (*ScanWorkflow, error) {
	// If workflow name is specified in job, use that
	if job.WorkflowName != "" {
		workflow, exists := eo.workflowEngine.GetWorkflow(strings.ToLower(job.WorkflowName))
		if !exists {
			return nil, fmt.Errorf("specified workflow %s not found", job.WorkflowName)
		}
		return workflow, nil
	}

	// Otherwise, find workflows for artifact type
	workflows := eo.workflowEngine.GetWorkflowsForArtifactType(job.ArtifactType)
	if len(workflows) == 0 {
		return nil, fmt.Errorf("no workflows found for artifact type %s", job.ArtifactType)
	}

	// Select the first enabled workflow (could be enhanced with priority logic)
	for _, workflow := range workflows {
		if eo.isWorkflowEnabledForTenant(job.TenantID, workflow.Name) {
			return workflow, nil
		}
	}

	return nil, fmt.Errorf("no enabled workflows found for artifact type %s and tenant %s",
		job.ArtifactType, job.TenantID)
}

// isWorkflowEnabledForTenant checks if workflow is enabled for tenant
func (eo *EnhancedOrchestrator) isWorkflowEnabledForTenant(tenantID, workflowName string) bool {
	if eo.tenantConfig == nil {
		return true // Default to enabled if no tenant config service
	}

	return eo.tenantConfig.IsWorkflowEnabledForTenant(tenantID, workflowName)
}

// executeWorkflow executes the workflow using the workflow engine
func (eo *EnhancedOrchestrator) executeWorkflow(ctx context.Context, job EnhancedScanJob, workflow *ScanWorkflow) (*WorkflowResult, error) {
	return eo.integrationManager.ExecuteWithPlugins(ctx, job.ArtifactPath, job.ArtifactType, workflow.ID)
}

// evaluatePolicy evaluates the scan results against policy
func (eo *EnhancedOrchestrator) evaluatePolicy(ctx context.Context, job EnhancedScanJob, workflow *ScanWorkflow, results *AggregatedScanResult) (*PolicyDecision, error) {
	// Build policy input with enhanced context
	policyInput := map[string]interface{}{
		"artifact_id":     job.ArtifactID,
		"artifact_type":   job.ArtifactType,
		"artifact_path":   job.ArtifactPath,
		"tenant_id":       job.TenantID,
		"workflow_id":     workflow.ID,
		"workflow_name":   workflow.Name,
		"job_id":          job.JobID,
		"scan_results":    results,
		"vulnerabilities": results.Vulnerabilities,
		"summary":         results.Summary,
		"scanners_used":   eo.extractScannerNames(results),
		"scan_timestamp":  time.Now(),
		"metadata":        job.Metadata,
	}

	// Get tenant-specific policy path if available
	policyPath := workflow.Stages[0].ScannerRules[0].Requirements.StrategyName // Default from workflow
	if eo.tenantConfig != nil {
		if tenantConfig, err := eo.tenantConfig.GetTenantWorkflowConfig(job.TenantID, workflow.Name); err == nil && tenantConfig.PolicyPath != "" {
			policyPath = tenantConfig.PolicyPath
		}
	}

	// Use workflow policy path if no tenant override
	if policyPath == "" {
		// Extract policy path from workflow configuration (would need to be added to ScanWorkflow)
		policyPath = "/v1/data/securestor/default_policy"
	}

	eo.logger.Printf("[ORCHESTRATOR] Evaluating policy at path: %s", policyPath)

	decision, err := eo.policyClient.Evaluate(ctx, policyInput)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// Convert Decision to PolicyDecision
	policyDecision := &PolicyDecision{
		Decision:   decision,
		PolicyPath: policyPath,
		Violations: make([]*Violation, 0),
		Metadata:   make(map[string]interface{}),
	}

	return policyDecision, nil
}

// extractScannerNames extracts scanner names from aggregated results
func (eo *EnhancedOrchestrator) extractScannerNames(results *AggregatedScanResult) []string {
	var scanners []string
	scannerSet := make(map[string]bool)

	for _, result := range results.ScannerResults {
		if !scannerSet[result.ScannerName] {
			scanners = append(scanners, result.ScannerName)
			scannerSet[result.ScannerName] = true
		}
	}

	return scanners
}

// determineJobStatus determines final job status based on policy decision
func (eo *EnhancedOrchestrator) determineJobStatus(decision *PolicyDecision) ScanJobStatus {
	switch decision.Action {
	case "allow":
		return JobStatusCompleted
	case "warn":
		return JobStatusCompletedWithWarnings
	case "quarantine":
		return JobStatusQuarantined
	case "block", "deny":
		return JobStatusBlocked
	default:
		return JobStatusCompleted
	}
}

// shouldSendNotification determines if notification should be sent
func (eo *EnhancedOrchestrator) shouldSendNotification(decision *PolicyDecision, status ScanJobStatus) bool {
	return status == JobStatusBlocked || status == JobStatusQuarantined || status == JobStatusCompletedWithWarnings
}

// sendViolationNotification sends policy violation notification
func (eo *EnhancedOrchestrator) sendViolationNotification(
	ctx context.Context,
	job EnhancedScanJob,
	workflow *ScanWorkflow,
	decision *PolicyDecision,
	result *ScanJobResult,
) error {
	if eo.notificationService == nil {
		return nil // No notification service configured
	}

	// Extract violations from policy decision
	var violations []Violation
	if decision.Violations != nil {
		for _, v := range decision.Violations {
			violation := Violation{
				Rule:     v.Rule,
				Severity: v.Severity,
				Message:  v.Message,
				Resource: v.Resource,
				Details:  v.Details,
			}
			violations = append(violations, violation)
		}
	}

	notification := ViolationNotification{
		TenantID:     job.TenantID,
		ArtifactID:   job.ArtifactID,
		ArtifactType: job.ArtifactType,
		WorkflowName: workflow.Name,
		PolicyPath:   decision.PolicyPath,
		Decision:     *decision,
		Violations:   violations,
		ScanResults:  result.AggregatedResult,
		Timestamp:    time.Now(),
		Channels:     []string{"email", "slack"}, // Could be configured per tenant
		Metadata:     job.Metadata,
	}

	return eo.notificationService.SendViolationNotification(ctx, notification)
}

// storeResults stores the complete scan results
func (eo *EnhancedOrchestrator) storeResults(ctx context.Context, result *ScanJobResult) error {
	// Convert to legacy format for existing storage interface
	var scannerResults []ScannerResult

	if result.AggregatedResult != nil {
		for _, scanResult := range result.AggregatedResult.ScannerResults {
			// Convert ScanResult to ScannerResult
			summary := map[string]interface{}{
				"scanner_name":    scanResult.ScannerName,
				"scanner_version": scanResult.ScannerVersion,
				"artifact_type":   scanResult.ArtifactType,
				"vulnerabilities": scanResult.Vulnerabilities,
				"summary":         scanResult.Summary,
				"scan_duration":   scanResult.ScanDuration,
				"metadata":        scanResult.Metadata,
			}

			// Marshal the full scan result as JSON
			jsonData, _ := json.Marshal(scanResult)

			scannerResult := ScannerResult{
				Tool:      scanResult.ScannerName,
				OutputRaw: json.RawMessage(jsonData),
				Summary:   summary,
			}

			scannerResults = append(scannerResults, scannerResult)
		}
	}

	// Store individual scanner results
	if err := eo.outputStore.SaveScanResults(ctx, result.Job.JobID, scannerResults); err != nil {
		return fmt.Errorf("failed to save scanner results: %w", err)
	}

	// Mark job as completed with final status
	statusString := string(result.Status)
	if err := eo.outputStore.MarkJobCompleted(ctx, result.Job.JobID, statusString); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	return nil
}

// completeJob finalizes the job result
func (eo *EnhancedOrchestrator) completeJob(result *ScanJobResult) {
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
}

// GetJobStatus returns the current status of a job
func (eo *EnhancedOrchestrator) GetJobStatus(ctx context.Context, jobID string) (*ScanJobResult, error) {
	// This would typically query the storage system for job status
	// Implementation depends on the OutputStore interface
	return nil, fmt.Errorf("job status retrieval not implemented")
}

// ListAvailableWorkflows returns workflows available for a tenant and artifact type
func (eo *EnhancedOrchestrator) ListAvailableWorkflows(tenantID, artifactType string) []*ScanWorkflow {
	workflows := eo.workflowEngine.GetWorkflowsForArtifactType(artifactType)

	var available []*ScanWorkflow
	for _, workflow := range workflows {
		if eo.isWorkflowEnabledForTenant(tenantID, workflow.Name) {
			available = append(available, workflow)
		}
	}

	return available
}

// ========== PERFORMANCE ENHANCEMENT METHODS ==========

// ExecuteJobWithPerformanceOptimizations executes scan job with performance enhancements
func (eo *EnhancedOrchestrator) ExecuteJobWithPerformanceOptimizations(ctx context.Context, job EnhancedScanJob) (*ScanJobResult, error) {
	if !eo.enablePerformanceMode {
		// Fall back to standard execution
		return eo.ExecuteJob(ctx, job)
	}

	startTime := time.Now()
	eo.performanceEnhancer.metrics.mu.Lock()
	eo.performanceEnhancer.metrics.TotalScans++
	eo.performanceEnhancer.metrics.mu.Unlock()

	// Check cache first
	if eo.performanceEnhancer.config.EnableResultCache {
		if cachedResult := eo.checkResultCache(job); cachedResult != nil {
			eo.performanceEnhancer.metrics.mu.Lock()
			eo.performanceEnhancer.metrics.CacheHits++
			eo.performanceEnhancer.metrics.mu.Unlock()

			eo.logger.Printf("[ORCHESTRATOR] Cache hit for job %s", job.JobID)
			return cachedResult, nil
		}

		eo.performanceEnhancer.metrics.mu.Lock()
		eo.performanceEnhancer.metrics.CacheMisses++
		eo.performanceEnhancer.metrics.mu.Unlock()
	}

	// Use semaphore to limit concurrent scans
	select {
	case eo.performanceEnhancer.scanSemaphore <- struct{}{}:
		defer func() { <-eo.performanceEnhancer.scanSemaphore }()
	case <-ctx.Done():
		return nil, fmt.Errorf("scan canceled while waiting for slot: %w", ctx.Err())
	}

	// Memory optimization check
	if eo.performanceEnhancer.config.EnableGCOptimization {
		eo.checkMemoryAndOptimize()
	}

	// Execute with circuit breaker protection
	result, err := eo.executeWithCircuitBreaker(ctx, job)

	duration := time.Since(startTime)
	eo.updatePerformanceMetrics(duration, err != nil)

	// Cache successful results
	if err == nil && eo.performanceEnhancer.config.EnableResultCache {
		eo.cacheResult(job, result)
	}

	return result, err
}

// ExecuteParallelJobs executes multiple jobs in parallel with performance optimizations
func (eo *EnhancedOrchestrator) ExecuteParallelJobs(ctx context.Context, jobs []EnhancedScanJob) ([]*ScanJobResult, error) {
	if !eo.enablePerformanceMode {
		return eo.executeJobsSequentially(ctx, jobs)
	}

	resultChan := make(chan *jobResult, len(jobs))

	// Start parallel execution with limited concurrency
	for i, job := range jobs {
		go func(index int, scanJob EnhancedScanJob) {
			result, err := eo.ExecuteJobWithPerformanceOptimizations(ctx, scanJob)
			resultChan <- &jobResult{
				index:  index,
				result: result,
				err:    err,
			}
		}(i, job)
	}

	// Collect results
	results := make([]*ScanJobResult, len(jobs))
	var errors []error

	for i := 0; i < len(jobs); i++ {
		select {
		case jobRes := <-resultChan:
			results[jobRes.index] = jobRes.result
			if jobRes.err != nil {
				errors = append(errors, jobRes.err)
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("parallel execution canceled: %w", ctx.Err())
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("some jobs failed: %v", errors)
	}

	return results, nil
}

// checkResultCache checks for cached scan results
func (eo *EnhancedOrchestrator) checkResultCache(job EnhancedScanJob) *ScanJobResult {
	cacheKey := eo.generateCacheKey(job)

	if cached, ok := eo.performanceEnhancer.resultCache.Load(cacheKey); ok {
		cachedResult := cached.(*CachedResult)
		if !cachedResult.IsExpired() {
			return &ScanJobResult{
				Job: EnhancedScanJob{
					JobID:      job.JobID,
					ArtifactID: job.ArtifactID,
				},
				Status: JobStatusCompleted,
				AggregatedResult: &AggregatedScanResult{
					ScannerResults: []*ScanResult{cachedResult.Result},
				},
				StartedAt:   cachedResult.CachedAt,
				CompletedAt: time.Now(),
			}
		}
		// Remove expired cache entry
		eo.performanceEnhancer.resultCache.Delete(cacheKey)
	}

	return nil
}

// cacheResult stores scan result in cache
func (eo *EnhancedOrchestrator) cacheResult(job EnhancedScanJob, result *ScanJobResult) {
	if result.AggregatedResult == nil || len(result.AggregatedResult.ScannerResults) == 0 {
		return
	}

	cacheKey := eo.generateCacheKey(job)
	cachedResult := &CachedResult{
		Result:   result.AggregatedResult.ScannerResults[0], // Cache primary result
		CachedAt: time.Now(),
		TTL:      eo.performanceEnhancer.config.CacheTTL,
	}

	eo.performanceEnhancer.resultCache.Store(cacheKey, cachedResult)
}

// executeWithCircuitBreaker executes job with circuit breaker protection
func (eo *EnhancedOrchestrator) executeWithCircuitBreaker(ctx context.Context, job EnhancedScanJob) (*ScanJobResult, error) {
	if !eo.performanceEnhancer.config.CircuitBreakerEnabled {
		return eo.ExecuteJob(ctx, job)
	}

	// Get or create circuit breaker for this job type
	breakerKey := fmt.Sprintf("%s-%s", job.ArtifactType, job.TenantID)
	circuitBreaker := eo.getOrCreateCircuitBreaker(breakerKey)

	if !circuitBreaker.CanExecute() {
		eo.performanceEnhancer.metrics.mu.Lock()
		eo.performanceEnhancer.metrics.CircuitBreakEvents++
		eo.performanceEnhancer.metrics.mu.Unlock()

		return nil, fmt.Errorf("circuit breaker is open for %s", breakerKey)
	}

	result, err := eo.executeWithRetry(ctx, job)

	if err != nil {
		circuitBreaker.RecordFailure()
	} else {
		circuitBreaker.RecordSuccess()
	}

	return result, err
}

// executeWithRetry executes job with retry logic
func (eo *EnhancedOrchestrator) executeWithRetry(ctx context.Context, job EnhancedScanJob) (*ScanJobResult, error) {
	var lastError error

	for attempt := 0; attempt <= eo.performanceEnhancer.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * eo.performanceEnhancer.config.RetryBackoff
			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			eo.logger.Printf("[ORCHESTRATOR] Retrying job %s, attempt %d", job.JobID, attempt+1)
		}

		// Create timeout context for individual attempt
		attemptCtx, cancel := context.WithTimeout(ctx, eo.performanceEnhancer.config.ScanTimeout)
		result, err := eo.ExecuteJob(attemptCtx, job)
		cancel()

		if err == nil {
			return result, nil
		}

		lastError = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			break
		}
	}

	return nil, fmt.Errorf("job failed after %d attempts: %w", eo.performanceEnhancer.config.MaxRetries+1, lastError)
}

// Helper methods

func (eo *EnhancedOrchestrator) generateCacheKey(job EnhancedScanJob) string {
	return fmt.Sprintf("%s:%s:%s:%s", job.ArtifactType, job.ArtifactPath, job.TenantID, job.WorkflowName)
}

func (eo *EnhancedOrchestrator) getOrCreateCircuitBreaker(key string) *CircuitBreaker {
	if breaker, ok := eo.performanceEnhancer.circuitBreakers.Load(key); ok {
		return breaker.(*CircuitBreaker)
	}

	breaker := NewCircuitBreaker(
		eo.performanceEnhancer.config.FailureThreshold,
		60*time.Second, // Fixed timeout for now
	)
	eo.performanceEnhancer.circuitBreakers.Store(key, breaker)
	return breaker
}

func (eo *EnhancedOrchestrator) checkMemoryAndOptimize() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	usedMB := m.Alloc / 1024 / 1024
	thresholdMB := uint64(eo.performanceEnhancer.config.MaxMemoryMB) * uint64(eo.performanceEnhancer.config.GCThresholdPercent) / 100

	if usedMB > thresholdMB {
		runtime.GC()
		eo.logger.Printf("[ORCHESTRATOR] Triggered GC: %d MB used (threshold: %d MB)", usedMB, thresholdMB)
	}
}

func (eo *EnhancedOrchestrator) updatePerformanceMetrics(duration time.Duration, failed bool) {
	eo.performanceEnhancer.metrics.mu.Lock()
	defer eo.performanceEnhancer.metrics.mu.Unlock()

	if failed {
		eo.performanceEnhancer.metrics.FailedScans++
	}

	// Update rolling average duration
	totalScans := eo.performanceEnhancer.metrics.TotalScans
	if totalScans > 0 {
		currentAvg := eo.performanceEnhancer.metrics.AverageDuration
		eo.performanceEnhancer.metrics.AverageDuration = time.Duration(
			(int64(currentAvg)*int64(totalScans-1) + int64(duration)) / int64(totalScans),
		)
	}
}

func (eo *EnhancedOrchestrator) executeJobsSequentially(ctx context.Context, jobs []EnhancedScanJob) ([]*ScanJobResult, error) {
	results := make([]*ScanJobResult, len(jobs))

	for i, job := range jobs {
		result, err := eo.ExecuteJob(ctx, job)
		if err != nil {
			return results, fmt.Errorf("job %d failed: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// GetPerformanceMetrics returns current performance metrics
func (eo *EnhancedOrchestrator) GetPerformanceMetrics() *PerformanceMetrics {
	if !eo.enablePerformanceMode {
		return nil
	}
	return eo.performanceEnhancer.metrics
}

// jobResult is a helper struct for parallel execution
type jobResult struct {
	index  int
	result *ScanJobResult
	err    error
}
