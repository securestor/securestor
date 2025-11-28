package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PluginManager manages scanner plugins with advanced capabilities
type PluginManager struct {
	plugins        map[string]ScannerPlugin
	strategies     map[string]SelectionStrategy
	workflowEngine *WorkflowEngine
	registry       *PluginRegistry
	healthChecker  *HealthChecker
	mu             sync.RWMutex
	logger         Logger
}

// ScannerPlugin extends the Scanner interface with plugin capabilities
type ScannerPlugin interface {
	Scanner

	// Plugin metadata
	GetMetadata() PluginMetadata
	GetCapabilities() []Capability
	GetDependencies() []string

	// Plugin lifecycle
	Initialize(config PluginConfig) error
	Shutdown() error
	HealthCheck() HealthStatus

	// Configuration
	ValidateConfig(config PluginConfig) error
	GetDefaultConfig() PluginConfig
}

// PluginMetadata contains plugin information
type PluginMetadata struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Version     string               `json:"version"`
	Author      string               `json:"author"`
	Description string               `json:"description"`
	Homepage    string               `json:"homepage"`
	License     string               `json:"license"`
	Tags        []string             `json:"tags"`
	Categories  []ScannerCategory    `json:"categories"`
	Priority    int                  `json:"priority"` // Higher number = higher priority
	Resources   ResourceRequirements `json:"resources"`
}

// Capability represents what a scanner can do
type Capability struct {
	Type        CapabilityType `json:"type"`
	Description string         `json:"description"`
	Confidence  float64        `json:"confidence"` // 0.0 - 1.0
	Performance Performance    `json:"performance"`
}

// CapabilityType defines different scanner capabilities
type CapabilityType string

const (
	// Vulnerability detection
	CapabilityCVE             CapabilityType = "cve_detection"
	CapabilityOSVulnerability CapabilityType = "os_vulnerability"
	CapabilityDependencyVuln  CapabilityType = "dependency_vulnerability"
	CapabilityContainerVuln   CapabilityType = "container_vulnerability"

	// Code analysis
	CapabilityStaticAnalysis CapabilityType = "static_analysis"
	CapabilityCodeQuality    CapabilityType = "code_quality"
	CapabilityCodeComplexity CapabilityType = "code_complexity"

	// Security scanning
	CapabilitySecretDetection  CapabilityType = "secret_detection"
	CapabilityMalwareDetection CapabilityType = "malware_detection"
	CapabilityLicenseScanning  CapabilityType = "license_scanning"

	// Infrastructure
	CapabilityIaCScanning     CapabilityType = "iac_scanning"
	CapabilityConfigScanning  CapabilityType = "config_scanning"
	CapabilityComplianceCheck CapabilityType = "compliance_check"

	// SBOM generation
	CapabilitySBOMGeneration  CapabilityType = "sbom_generation"
	CapabilityDependencyGraph CapabilityType = "dependency_graph"
)

// ScannerCategory classifies scanners
type ScannerCategory string

const (
	CategoryVulnerability ScannerCategory = "vulnerability"
	CategorySecurity      ScannerCategory = "security"
	CategoryCompliance    ScannerCategory = "compliance"
	CategoryQuality       ScannerCategory = "quality"
	CategoryLicense       ScannerCategory = "license"
	CategorySBOM          ScannerCategory = "sbom"
)

// Performance metrics for capability assessment
type Performance struct {
	Speed          PerformanceLevel `json:"speed"`           // FAST, MEDIUM, SLOW
	Accuracy       PerformanceLevel `json:"accuracy"`        // HIGH, MEDIUM, LOW
	Coverage       PerformanceLevel `json:"coverage"`        // COMPREHENSIVE, MODERATE, BASIC
	FalsePositives PerformanceLevel `json:"false_positives"` // LOW, MEDIUM, HIGH
}

type PerformanceLevel string

const (
	PerformanceLow    PerformanceLevel = "low"
	PerformanceMedium PerformanceLevel = "medium"
	PerformanceHigh   PerformanceLevel = "high"

	PerformanceBasic         PerformanceLevel = "basic"
	PerformanceModerate      PerformanceLevel = "moderate"
	PerformanceComprehensive PerformanceLevel = "comprehensive"

	PerformanceFast PerformanceLevel = "fast"
	PerformanceSlow PerformanceLevel = "slow"
)

// ResourceRequirements defines resource needs
type ResourceRequirements struct {
	CPU     ResourceLevel `json:"cpu"`
	Memory  ResourceLevel `json:"memory"`
	Disk    ResourceLevel `json:"disk"`
	Network ResourceLevel `json:"network"`
}

type ResourceLevel string

const (
	ResourceLow    ResourceLevel = "low"
	ResourceMedium ResourceLevel = "medium"
	ResourceHigh   ResourceLevel = "high"
)

// PluginConfig holds plugin configuration
type PluginConfig struct {
	Settings map[string]interface{} `json:"settings"`
	Timeout  time.Duration          `json:"timeout"`
	Retries  int                    `json:"retries"`
	Enabled  bool                   `json:"enabled"`
}

// HealthStatus represents plugin health
type HealthStatus struct {
	Status    HealthStatusType       `json:"status"`
	Message   string                 `json:"message"`
	CheckedAt time.Time              `json:"checked_at"`
	Details   map[string]interface{} `json:"details"`
}

type HealthStatusType string

const (
	HealthHealthy   HealthStatusType = "healthy"
	HealthUnhealthy HealthStatusType = "unhealthy"
	HealthDegraded  HealthStatusType = "degraded"
	HealthUnknown   HealthStatusType = "unknown"
)

// SelectionStrategy defines how scanners are selected
type SelectionStrategy interface {
	SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error)
	GetName() string
	GetDescription() string
}

// ScanSelectionRequest contains selection criteria
type ScanSelectionRequest struct {
	ArtifactType         string                 `json:"artifact_type"`
	RequiredCapabilities []CapabilityType       `json:"required_capabilities"`
	PreferredCategories  []ScannerCategory      `json:"preferred_categories"`
	MaxScanners          int                    `json:"max_scanners"`
	MaxDuration          time.Duration          `json:"max_duration"`
	MinConfidence        float64                `json:"min_confidence"`
	ResourceLimits       ResourceRequirements   `json:"resource_limits"`
	StrategyName         string                 `json:"strategy_name"`
	CustomCriteria       map[string]interface{} `json:"custom_criteria"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(logger Logger) *PluginManager {
	pm := &PluginManager{
		plugins:       make(map[string]ScannerPlugin),
		strategies:    make(map[string]SelectionStrategy),
		registry:      NewPluginRegistry(),
		healthChecker: NewHealthChecker(),
		logger:        logger,
	}

	// Initialize workflow engine
	pm.workflowEngine = NewWorkflowEngine(pm, logger)

	// Register default selection strategies
	pm.registerDefaultStrategies()

	return pm
}

// RegisterPlugin registers a new scanner plugin
func (pm *PluginManager) RegisterPlugin(plugin ScannerPlugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metadata := plugin.GetMetadata()

	// Validate plugin
	if err := pm.validatePlugin(plugin); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	// Initialize plugin
	defaultConfig := plugin.GetDefaultConfig()
	if err := plugin.Initialize(defaultConfig); err != nil {
		return fmt.Errorf("plugin initialization failed: %w", err)
	}

	// Check health
	health := plugin.HealthCheck()
	if health.Status == HealthUnhealthy {
		return fmt.Errorf("plugin is unhealthy: %s", health.Message)
	}

	pm.plugins[metadata.ID] = plugin
	pm.registry.Register(metadata)

	pm.logger.Printf("[PLUGIN_MANAGER] Registered plugin: %s v%s (%s)",
		metadata.Name, metadata.Version, metadata.ID)

	return nil
}

// SelectScanners selects optimal scanners based on strategy and criteria
func (pm *PluginManager) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Get selection strategy
	strategy, exists := pm.strategies[request.StrategyName]
	if !exists {
		strategy = pm.strategies["optimal"] // Default strategy
	}

	// Select scanners using strategy
	selected, err := strategy.SelectScanners(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("scanner selection failed: %w", err)
	}

	// Filter by health status
	healthy := make([]ScannerPlugin, 0, len(selected))
	for _, plugin := range selected {
		health := plugin.HealthCheck()
		if health.Status == HealthHealthy || health.Status == HealthDegraded {
			healthy = append(healthy, plugin)
		}
	}

	return healthy, nil
}

// GetPlugin retrieves a plugin by ID
func (pm *PluginManager) GetPlugin(id string) (ScannerPlugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[id]
	return plugin, exists
}

// ListPlugins returns all registered plugins
func (pm *PluginManager) ListPlugins() []PluginMetadata {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]PluginMetadata, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin.GetMetadata())
	}

	return plugins
}

// GetPluginsByCategory returns plugins in a specific category
func (pm *PluginManager) GetPluginsByCategory(category ScannerCategory) []ScannerPlugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var plugins []ScannerPlugin
	for _, plugin := range pm.plugins {
		metadata := plugin.GetMetadata()
		for _, cat := range metadata.Categories {
			if cat == category {
				plugins = append(plugins, plugin)
				break
			}
		}
	}

	return plugins
}

// GetPluginsByCapability returns plugins with specific capability
func (pm *PluginManager) GetPluginsByCapability(capability CapabilityType) []ScannerPlugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var plugins []ScannerPlugin
	for _, plugin := range pm.plugins {
		capabilities := plugin.GetCapabilities()
		for _, cap := range capabilities {
			if cap.Type == capability {
				plugins = append(plugins, plugin)
				break
			}
		}
	}

	return plugins
}

// validatePlugin validates a plugin before registration
func (pm *PluginManager) validatePlugin(plugin ScannerPlugin) error {
	metadata := plugin.GetMetadata()

	if metadata.ID == "" {
		return fmt.Errorf("plugin ID cannot be empty")
	}

	if metadata.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if metadata.Version == "" {
		return fmt.Errorf("plugin version cannot be empty")
	}

	// Check if plugin is already registered
	if _, exists := pm.plugins[metadata.ID]; exists {
		return fmt.Errorf("plugin %s is already registered", metadata.ID)
	}

	// Validate configuration
	config := plugin.GetDefaultConfig()
	if err := plugin.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid default configuration: %w", err)
	}

	return nil
}

// registerDefaultStrategies registers built-in selection strategies
func (pm *PluginManager) registerDefaultStrategies() {
	pm.strategies["optimal"] = NewOptimalSelectionStrategy()
	pm.strategies["fastest"] = NewFastestSelectionStrategy()
	pm.strategies["comprehensive"] = NewComprehensiveSelectionStrategy()
	pm.strategies["balanced"] = NewBalancedSelectionStrategy()
	pm.strategies["security_focused"] = NewSecurityFocusedStrategy()
}

// Shutdown gracefully shuts down all plugins
func (pm *PluginManager) Shutdown(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []error

	for id, plugin := range pm.plugins {
		if err := plugin.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown plugin %s: %w", id, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}
