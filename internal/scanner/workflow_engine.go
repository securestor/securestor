package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

// PluginRegistry manages plugin metadata and discovery
type PluginRegistry struct {
	plugins map[string]PluginMetadata
	indices map[string][]string // category/capability -> plugin IDs
	mu      sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]PluginMetadata),
		indices: make(map[string][]string),
	}
}

// Register adds plugin metadata to the registry
func (r *PluginRegistry) Register(metadata PluginMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins[metadata.ID] = metadata

	// Update indices
	r.updateIndices(metadata)
}

// updateIndices updates search indices for the plugin
func (r *PluginRegistry) updateIndices(metadata PluginMetadata) {
	// Category index
	for _, category := range metadata.Categories {
		key := fmt.Sprintf("category:%s", category)
		r.indices[key] = append(r.indices[key], metadata.ID)
	}

	// Tag index
	for _, tag := range metadata.Tags {
		key := fmt.Sprintf("tag:%s", tag)
		r.indices[key] = append(r.indices[key], metadata.ID)
	}
}

// FindByCategory returns plugin IDs for a category
func (r *PluginRegistry) FindByCategory(category ScannerCategory) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("category:%s", category)
	return r.indices[key]
}

// FindByTag returns plugin IDs for a tag
func (r *PluginRegistry) FindByTag(tag string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("tag:%s", tag)
	return r.indices[key]
}

// Get returns plugin metadata by ID
func (r *PluginRegistry) Get(id string) (PluginMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.plugins[id]
	return metadata, exists
}

// List returns all registered plugin metadata
func (r *PluginRegistry) List() []PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]PluginMetadata, 0, len(r.plugins))
	for _, metadata := range r.plugins {
		plugins = append(plugins, metadata)
	}

	return plugins
}

// WorkflowEngine manages scanning workflows
type WorkflowEngine struct {
	pluginManager *PluginManager
	workflows     map[string]*ScanWorkflow
	logger        Logger
	mu            sync.RWMutex
}

// ScanWorkflow defines a complete scanning workflow
type ScanWorkflow struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	ArtifactTypes []string              `json:"artifact_types"`
	Stages        []WorkflowStage       `json:"stages"`
	Configuration WorkflowConfiguration `json:"configuration"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// WorkflowStage represents a stage in the workflow
type WorkflowStage struct {
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	ScannerRules    []ScannerRule    `json:"scanner_rules"`
	Parallel        bool             `json:"parallel"`
	ContinueOnError bool             `json:"continue_on_error"`
	Timeout         time.Duration    `json:"timeout"`
	Dependencies    []string         `json:"dependencies"` // Stage names this depends on
	Conditions      []StageCondition `json:"conditions"`
}

// ScannerRule defines how scanners are selected for a stage
type ScannerRule struct {
	Type         ScannerRuleType      `json:"type"`
	Strategy     string               `json:"strategy"`
	Requirements ScanSelectionRequest `json:"requirements"`
	Fallback     *ScannerRule         `json:"fallback,omitempty"`
}

type ScannerRuleType string

const (
	RuleTypeFixed       ScannerRuleType = "fixed"       // Use specific scanners
	RuleTypeStrategy    ScannerRuleType = "strategy"    // Use selection strategy
	RuleTypeConditional ScannerRuleType = "conditional" // Conditional selection
)

// StageCondition defines when a stage should run
type StageCondition struct {
	Type      ConditionType     `json:"type"`
	Parameter string            `json:"parameter"`
	Operator  ConditionOperator `json:"operator"`
	Value     interface{}       `json:"value"`
}

type ConditionType string

const (
	ConditionArtifactType   ConditionType = "artifact_type"
	ConditionFileSize       ConditionType = "file_size"
	ConditionPreviousResult ConditionType = "previous_result"
	ConditionCustom         ConditionType = "custom"
)

type ConditionOperator string

const (
	OpEquals      ConditionOperator = "equals"
	OpNotEquals   ConditionOperator = "not_equals"
	OpGreaterThan ConditionOperator = "greater_than"
	OpLessThan    ConditionOperator = "less_than"
	OpContains    ConditionOperator = "contains"
	OpMatches     ConditionOperator = "matches" // Regex
)

// WorkflowConfiguration defines workflow behavior
type WorkflowConfiguration struct {
	MaxConcurrency    int               `json:"max_concurrency"`
	GlobalTimeout     time.Duration     `json:"global_timeout"`
	FailFast          bool              `json:"fail_fast"`
	RetryPolicy       RetryPolicy       `json:"retry_policy"`
	ResultAggregation AggregationConfig `json:"result_aggregation"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	BackoffFactor float64       `json:"backoff_factor"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
}

// AggregationConfig defines how results are aggregated
type AggregationConfig struct {
	Strategy      AggregationStrategy `json:"strategy"`
	WeightByType  map[string]float64  `json:"weight_by_type"`
	Deduplication bool                `json:"deduplication"`
}

type AggregationStrategy string

const (
	AggregationMerge    AggregationStrategy = "merge"
	AggregationUnion    AggregationStrategy = "union"
	AggregationWeighted AggregationStrategy = "weighted"
)

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(pluginManager *PluginManager, logger Logger) *WorkflowEngine {
	we := &WorkflowEngine{
		pluginManager: pluginManager,
		workflows:     make(map[string]*ScanWorkflow),
		logger:        logger,
	}

	// Initialize default workflows
	we.initializeDefaultWorkflows()

	return we
}

// RegisterWorkflow registers a new workflow
func (we *WorkflowEngine) RegisterWorkflow(workflow *ScanWorkflow) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if err := we.validateWorkflow(workflow); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	workflow.UpdatedAt = time.Now()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = time.Now()
	}

	we.workflows[workflow.ID] = workflow

	we.logger.Printf("[WORKFLOW_ENGINE] Registered workflow: %s (%s)", workflow.Name, workflow.ID)

	return nil
}

// ExecuteWorkflow executes a workflow for an artifact
func (we *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflowID string, request WorkflowExecutionRequest) (*WorkflowResult, error) {
	we.mu.RLock()
	workflow, exists := we.workflows[workflowID]
	we.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	we.logger.Printf("[WORKFLOW_ENGINE] Executing workflow: %s for artifact: %s", workflow.Name, request.ArtifactPath)

	// Create execution context
	execCtx := &WorkflowExecutionContext{
		Workflow:      workflow,
		Request:       request,
		Results:       make(map[string]*StageResult),
		StartTime:     time.Now(),
		Logger:        we.logger,
		PluginManager: we.pluginManager,
	}

	// Execute stages
	result, err := we.executeStages(ctx, execCtx)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	return result, nil
}

// WorkflowExecutionRequest contains execution parameters
type WorkflowExecutionRequest struct {
	ArtifactPath string                 `json:"artifact_path"`
	ArtifactType string                 `json:"artifact_type"`
	Options      map[string]interface{} `json:"options"`
}

// WorkflowResult contains workflow execution results
type WorkflowResult struct {
	WorkflowID       string                  `json:"workflow_id"`
	Status           WorkflowStatus          `json:"status"`
	StartTime        time.Time               `json:"start_time"`
	EndTime          time.Time               `json:"end_time"`
	Duration         time.Duration           `json:"duration"`
	StageResults     map[string]*StageResult `json:"stage_results"`
	AggregatedResult *AggregatedScanResult   `json:"aggregated_result"`
	Errors           []string                `json:"errors,omitempty"`
}

type WorkflowStatus string

const (
	StatusPending   WorkflowStatus = "pending"
	StatusRunning   WorkflowStatus = "running"
	StatusCompleted WorkflowStatus = "completed"
	StatusFailed    WorkflowStatus = "failed"
	StatusCanceled  WorkflowStatus = "canceled"
)

// StageResult contains stage execution results
type StageResult struct {
	StageName      string         `json:"stage_name"`
	Status         WorkflowStatus `json:"status"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	Duration       time.Duration  `json:"duration"`
	ScannerResults []*ScanResult  `json:"scanner_results"`
	Errors         []string       `json:"errors,omitempty"`
}

// WorkflowExecutionContext holds execution state
type WorkflowExecutionContext struct {
	Workflow      *ScanWorkflow
	Request       WorkflowExecutionRequest
	Results       map[string]*StageResult
	StartTime     time.Time
	Logger        Logger
	PluginManager *PluginManager
}

// executeStages executes all workflow stages
func (we *WorkflowEngine) executeStages(ctx context.Context, execCtx *WorkflowExecutionContext) (*WorkflowResult, error) {
	result := &WorkflowResult{
		WorkflowID:   execCtx.Workflow.ID,
		Status:       StatusRunning,
		StartTime:    execCtx.StartTime,
		StageResults: make(map[string]*StageResult),
	}

	// Execute stages in order, respecting dependencies
	for _, stage := range execCtx.Workflow.Stages {
		// Check if stage should run
		if !we.shouldRunStage(stage, execCtx) {
			we.logger.Printf("[WORKFLOW] Skipping stage: %s (conditions not met)", stage.Name)
			continue
		}

		// Wait for dependencies
		if err := we.waitForDependencies(ctx, stage, execCtx); err != nil {
			return nil, fmt.Errorf("dependency wait failed for stage %s: %w", stage.Name, err)
		}

		// Execute stage
		stageResult, err := we.executeStage(ctx, stage, execCtx)
		if err != nil {
			if !stage.ContinueOnError {
				result.Status = StatusFailed
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result, fmt.Errorf("stage %s failed: %w", stage.Name, err)
			}

			// Continue with error
			stageResult.Errors = append(stageResult.Errors, err.Error())
		}

		result.StageResults[stage.Name] = stageResult
		execCtx.Results[stage.Name] = stageResult
	}

	result.Status = StatusCompleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Aggregate results
	if aggregated, err := we.aggregateResults(execCtx); err == nil {
		result.AggregatedResult = aggregated
	}

	return result, nil
}

// shouldRunStage checks if stage conditions are met
func (we *WorkflowEngine) shouldRunStage(stage WorkflowStage, execCtx *WorkflowExecutionContext) bool {
	for _, condition := range stage.Conditions {
		if !we.evaluateCondition(condition, execCtx) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a stage condition
func (we *WorkflowEngine) evaluateCondition(condition StageCondition, execCtx *WorkflowExecutionContext) bool {
	switch condition.Type {
	case ConditionArtifactType:
		return we.evaluateStringCondition(execCtx.Request.ArtifactType, condition.Operator, condition.Value)
	case ConditionFileSize:
		// Implementation would check file size
		return true
	case ConditionPreviousResult:
		// Implementation would check previous stage results
		return true
	default:
		return true
	}
}

// evaluateStringCondition evaluates string-based conditions
func (we *WorkflowEngine) evaluateStringCondition(value string, operator ConditionOperator, expected interface{}) bool {
	expectedStr, ok := expected.(string)
	if !ok {
		return false
	}

	switch operator {
	case OpEquals:
		return value == expectedStr
	case OpNotEquals:
		return value != expectedStr
	case OpContains:
		return value == expectedStr // Simplified
	default:
		return false
	}
}

// waitForDependencies waits for stage dependencies to complete
func (we *WorkflowEngine) waitForDependencies(ctx context.Context, stage WorkflowStage, execCtx *WorkflowExecutionContext) error {
	for _, dep := range stage.Dependencies {
		// Wait for dependency stage to complete
		for {
			if result, exists := execCtx.Results[dep]; exists && result.Status == StatusCompleted {
				break
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				// Continue waiting
			}
		}
	}
	return nil
}

// executeStage executes a single workflow stage
func (we *WorkflowEngine) executeStage(ctx context.Context, stage WorkflowStage, execCtx *WorkflowExecutionContext) (*StageResult, error) {
	stageResult := &StageResult{
		StageName: stage.Name,
		Status:    StatusRunning,
		StartTime: time.Now(),
	}

	we.logger.Printf("[WORKFLOW] Executing stage: %s", stage.Name)

	// Apply stage timeout
	stageCtx := ctx
	if stage.Timeout > 0 {
		var cancel context.CancelFunc
		stageCtx, cancel = context.WithTimeout(ctx, stage.Timeout)
		defer cancel()
	}

	// Execute scanner rules
	var allResults []*ScanResult
	for _, rule := range stage.ScannerRules {
		results, err := we.executeScannerRule(stageCtx, rule, execCtx)
		if err != nil {
			stageResult.Errors = append(stageResult.Errors, err.Error())
			continue
		}
		allResults = append(allResults, results...)
	}

	stageResult.ScannerResults = allResults
	stageResult.Status = StatusCompleted
	stageResult.EndTime = time.Now()
	stageResult.Duration = stageResult.EndTime.Sub(stageResult.StartTime)

	return stageResult, nil
}

// executeScannerRule executes a scanner rule
func (we *WorkflowEngine) executeScannerRule(ctx context.Context, rule ScannerRule, execCtx *WorkflowExecutionContext) ([]*ScanResult, error) {
	switch rule.Type {
	case RuleTypeStrategy:
		return we.executeStrategyRule(ctx, rule, execCtx)
	case RuleTypeFixed:
		return we.executeFixedRule(ctx, rule, execCtx)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.Type)
	}
}

// executeStrategyRule executes a strategy-based scanner rule
func (we *WorkflowEngine) executeStrategyRule(ctx context.Context, rule ScannerRule, execCtx *WorkflowExecutionContext) ([]*ScanResult, error) {
	// Set artifact type for selection
	rule.Requirements.ArtifactType = execCtx.Request.ArtifactType
	rule.Requirements.StrategyName = rule.Strategy

	// Select scanners using strategy
	plugins, err := we.pluginManager.SelectScanners(ctx, rule.Requirements)
	if err != nil {
		return nil, fmt.Errorf("scanner selection failed: %w", err)
	}

	// Execute selected scanners
	var results []*ScanResult
	for _, plugin := range plugins {
		result, err := plugin.Scan(ctx, execCtx.Request.ArtifactPath, execCtx.Request.ArtifactType)
		if err != nil {
			we.logger.Printf("[WORKFLOW] Scanner %s failed: %v", plugin.Name(), err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// executeFixedRule executes a fixed scanner rule
func (we *WorkflowEngine) executeFixedRule(ctx context.Context, rule ScannerRule, execCtx *WorkflowExecutionContext) ([]*ScanResult, error) {
	// Implementation would execute specific scanners
	return []*ScanResult{}, nil
}

// aggregateResults aggregates all stage results
func (we *WorkflowEngine) aggregateResults(execCtx *WorkflowExecutionContext) (*AggregatedScanResult, error) {
	var allResults []*ScanResult

	for _, stageResult := range execCtx.Results {
		allResults = append(allResults, stageResult.ScannerResults...)
	}

	// Use result aggregator to combine results
	aggregator := NewResultAggregator()
	return aggregator.Aggregate(allResults), nil
}

// validateWorkflow validates workflow configuration
func (we *WorkflowEngine) validateWorkflow(workflow *ScanWorkflow) error {
	if workflow.ID == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}

	if workflow.Name == "" {
		return fmt.Errorf("workflow name cannot be empty")
	}

	if len(workflow.Stages) == 0 {
		return fmt.Errorf("workflow must have at least one stage")
	}

	// Validate stage dependencies
	stageNames := make(map[string]bool)
	for _, stage := range workflow.Stages {
		stageNames[stage.Name] = true
	}

	for _, stage := range workflow.Stages {
		for _, dep := range stage.Dependencies {
			if !stageNames[dep] {
				return fmt.Errorf("stage %s depends on undefined stage: %s", stage.Name, dep)
			}
		}
	}

	return nil
}

// WorkflowDefinition represents a workflow configuration from JSON
type WorkflowDefinition struct {
	Name          string              `json:"name"`
	ArtifactTypes []string            `json:"artifact_types"`
	Scanners      []ScannerDefinition `json:"scanners"`
	PolicyPath    string              `json:"policy_path"`
	Description   string              `json:"description"`
	Execution     ExecutionConfig     `json:"execution"`
	Notifications NotificationConfig  `json:"notifications"`
	Enabled       bool                `json:"enabled"`
}

// ScannerDefinition represents scanner configuration in workflows
type ScannerDefinition struct {
	Name string `json:"name"`
}

// ExecutionConfig defines workflow execution parameters
type ExecutionConfig struct {
	Strategy      string `json:"strategy"`
	FailurePolicy string `json:"failure_policy"`
}

// NotificationConfig defines notification settings
type NotificationConfig struct {
	OnViolation []string `json:"on_violation"`
}

// WorkflowsConfiguration represents the complete workflows.json structure
type WorkflowsConfiguration struct {
	Version          string               `json:"version"`
	Metadata         WorkflowMetadata     `json:"metadata"`
	Workflows        []WorkflowDefinition `json:"workflows"`
	Policies         PolicyConfig         `json:"policies"`
	DefaultExecution DefaultExecution     `json:"default_execution"`
	Tenants          TenantsConfig        `json:"tenants"`
}

// WorkflowMetadata contains workflow registry metadata
type WorkflowMetadata struct {
	Description string `json:"description"`
	Maintainer  string `json:"maintainer"`
	LastUpdated string `json:"last_updated"`
}

// PolicyConfig defines policy evaluation settings
type PolicyConfig struct {
	BaseURL        string `json:"base_url"`
	DefaultTTL     int    `json:"default_ttl"`
	EvaluationMode string `json:"evaluation_mode"`
}

// DefaultExecution defines default execution parameters
type DefaultExecution struct {
	Strategy          string   `json:"strategy"`
	FailurePolicy     string   `json:"failure_policy"`
	NotifyOnViolation bool     `json:"notify_on_violation"`
	Triggers          []string `json:"triggers"`
}

// TenantsConfig defines tenant-specific configurations
type TenantsConfig struct {
	AllowOverride     bool                   `json:"allow_override"`
	DefaultPolicyMode string                 `json:"default_policy_mode"`
	ExampleOverride   map[string]interface{} `json:"example_override"`
}

// initializeDefaultWorkflows creates default workflows
func (we *WorkflowEngine) initializeDefaultWorkflows() {
	// Load workflows from configuration file
	if err := we.loadWorkflowsFromConfig(); err != nil {
		we.logger.Printf("[WORKFLOW_ENGINE] Failed to load workflows from config: %v", err)
		// Fall back to creating basic default workflows
		we.createBasicDefaultWorkflows()
	}
}

// loadWorkflowsFromConfig loads workflows from the workflows.json configuration file
func (we *WorkflowEngine) loadWorkflowsFromConfig() error {
	configPath := "configs/workflows.json"

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("workflows config file not found: %s", configPath)
	}

	// Read configuration file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read workflows config: %w", err)
	}

	// Parse configuration
	var config WorkflowsConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse workflows config: %w", err)
	}

	// Convert configuration workflows to engine workflows
	for _, workflowConfig := range config.Workflows {
		if !workflowConfig.Enabled {
			continue
		}

		workflow := we.convertConfigToWorkflow(workflowConfig)
		if err := we.RegisterWorkflow(workflow); err != nil {
			we.logger.Printf("[WORKFLOW_ENGINE] Failed to register workflow %s: %v", workflowConfig.Name, err)
			continue
		}

		we.logger.Printf("[WORKFLOW_ENGINE] Loaded workflow: %s for types: %v",
			workflowConfig.Name, workflowConfig.ArtifactTypes)
	}

	return nil
}

// convertConfigToWorkflow converts a WorkflowDefinition to a ScanWorkflow
func (we *WorkflowEngine) convertConfigToWorkflow(config WorkflowDefinition) *ScanWorkflow {
	// Create a simple workflow ID from the name
	workflowID := strings.ToLower(strings.ReplaceAll(config.Name, " ", "_"))

	// Extract scanner names from scanner definitions
	var scannerNames []string
	for _, scanner := range config.Scanners {
		scannerNames = append(scannerNames, scanner.Name)
	}

	// Create stages based on scanners
	stages := we.createStagesFromScanners(scannerNames, config.ArtifactTypes)

	// Convert execution strategy
	var executionConfig WorkflowConfiguration
	if config.Execution.Strategy == "parallel" {
		executionConfig.MaxConcurrency = 5
	} else {
		executionConfig.MaxConcurrency = 1
	}

	// Set failure policy
	failFast := config.Execution.FailurePolicy == "stop_on_critical"

	executionConfig = WorkflowConfiguration{
		MaxConcurrency: executionConfig.MaxConcurrency,
		GlobalTimeout:  15 * time.Minute,
		FailFast:       failFast,
		RetryPolicy: RetryPolicy{
			MaxRetries:    2,
			BackoffFactor: 1.5,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
		},
		ResultAggregation: AggregationConfig{
			Strategy:      AggregationMerge,
			Deduplication: true,
		},
	}

	return &ScanWorkflow{
		ID:            workflowID,
		Name:          config.Name,
		Description:   config.Description,
		ArtifactTypes: config.ArtifactTypes,
		Stages:        stages,
		Configuration: executionConfig,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// createStagesFromScanners creates workflow stages from scanner names
func (we *WorkflowEngine) createStagesFromScanners(scanners []string, artifactTypes []string) []WorkflowStage {
	var stages []WorkflowStage

	// Group scanners by execution priority
	fastScanners := []string{}
	thoroughScanners := []string{}

	for _, scanner := range scanners {
		switch scanner {
		case "syft":
			fastScanners = append(fastScanners, scanner)
		case "bandit", "trufflehog":
			fastScanners = append(fastScanners, scanner)
		default:
			thoroughScanners = append(thoroughScanners, scanner)
		}
	}

	// Create fast scan stage if we have fast scanners
	if len(fastScanners) > 0 {
		stages = append(stages, WorkflowStage{
			Name:        "fast_scan",
			Description: "Quick security scan",
			ScannerRules: []ScannerRule{
				{
					Type:     RuleTypeStrategy,
					Strategy: "fastest",
					Requirements: ScanSelectionRequest{
						MaxScanners: len(fastScanners),
						MaxDuration: 3 * time.Minute,
					},
				},
			},
			Parallel:        true,
			ContinueOnError: true,
			Timeout:         5 * time.Minute,
		})
	}

	// Create thorough scan stage if we have thorough scanners
	if len(thoroughScanners) > 0 {
		dependencies := []string{}
		if len(fastScanners) > 0 {
			dependencies = append(dependencies, "fast_scan")
		}

		stages = append(stages, WorkflowStage{
			Name:        "comprehensive_scan",
			Description: "Comprehensive security analysis",
			ScannerRules: []ScannerRule{
				{
					Type:     RuleTypeStrategy,
					Strategy: "comprehensive",
					Requirements: ScanSelectionRequest{
						MaxScanners: len(thoroughScanners),
						MaxDuration: 10 * time.Minute,
					},
				},
			},
			Parallel:        true,
			ContinueOnError: true,
			Timeout:         12 * time.Minute,
			Dependencies:    dependencies,
		})
	}

	return stages
}

// createBasicDefaultWorkflows creates fallback workflows if config loading fails
func (we *WorkflowEngine) createBasicDefaultWorkflows() {
	// Docker workflow
	dockerWorkflow := &ScanWorkflow{
		ID:            "docker_comprehensive",
		Name:          "Docker Comprehensive Scan",
		Description:   "Comprehensive security scanning for Docker images",
		ArtifactTypes: []string{"docker"},
		Stages: []WorkflowStage{
			{
				Name:        "fast_scan",
				Description: "Quick vulnerability scan",
				ScannerRules: []ScannerRule{
					{
						Type:     RuleTypeStrategy,
						Strategy: "fastest",
						Requirements: ScanSelectionRequest{
							MaxScanners: 2,
							MaxDuration: 2 * time.Minute,
						},
					},
				},
				Parallel:        true,
				ContinueOnError: true,
				Timeout:         3 * time.Minute,
			},
			{
				Name:        "comprehensive_scan",
				Description: "Thorough security analysis",
				ScannerRules: []ScannerRule{
					{
						Type:     RuleTypeStrategy,
						Strategy: "comprehensive",
						Requirements: ScanSelectionRequest{
							MaxScanners: 5,
							MaxDuration: 10 * time.Minute,
						},
					},
				},
				Parallel:        true,
				ContinueOnError: true,
				Timeout:         15 * time.Minute,
				Dependencies:    []string{"fast_scan"},
			},
		},
		Configuration: WorkflowConfiguration{
			MaxConcurrency: 4,
			GlobalTimeout:  20 * time.Minute,
			FailFast:       false,
		},
	}

	we.RegisterWorkflow(dockerWorkflow)

	// Add more basic workflows for other types if needed
	we.logger.Printf("[WORKFLOW_ENGINE] Created basic fallback workflows")
}

// GetWorkflow returns a workflow by ID
func (we *WorkflowEngine) GetWorkflow(id string) (*ScanWorkflow, bool) {
	we.mu.RLock()
	defer we.mu.RUnlock()

	workflow, exists := we.workflows[id]
	return workflow, exists
}

// ListWorkflows returns all registered workflows
func (we *WorkflowEngine) ListWorkflows() []*ScanWorkflow {
	we.mu.RLock()
	defer we.mu.RUnlock()

	workflows := make([]*ScanWorkflow, 0, len(we.workflows))
	for _, workflow := range we.workflows {
		workflows = append(workflows, workflow)
	}

	return workflows
}

// GetWorkflowsForArtifactType returns workflows suitable for an artifact type
func (we *WorkflowEngine) GetWorkflowsForArtifactType(artifactType string) []*ScanWorkflow {
	we.mu.RLock()
	defer we.mu.RUnlock()

	var matching []*ScanWorkflow
	for _, workflow := range we.workflows {
		for _, supportedType := range workflow.ArtifactTypes {
			if supportedType == artifactType {
				matching = append(matching, workflow)
				break
			}
		}
	}

	return matching
}
