package scanner

import (
	"context"
	"fmt"
	"time"
)

// PluginAdapter adapts existing scanners to the plugin interface
type PluginAdapter struct {
	scanner  Scanner
	metadata PluginMetadata
	health   *HealthStatus
}

// NewPluginAdapter creates a new plugin adapter for an existing scanner
func NewPluginAdapter(scanner Scanner, metadata PluginMetadata) *PluginAdapter {
	return &PluginAdapter{
		scanner:  scanner,
		metadata: metadata,
		health: &HealthStatus{
			Status:    HealthHealthy,
			CheckedAt: time.Now(),
			Message:   "Initialized",
		},
	}
}

// Scanner interface implementation
func (pa *PluginAdapter) Name() string {
	return pa.scanner.Name()
}

func (pa *PluginAdapter) SupportedTypes() []string {
	return pa.scanner.SupportedTypes()
}

func (pa *PluginAdapter) Scan(ctx context.Context, artifactPath, artifactType string) (*ScanResult, error) {
	return pa.scanner.Scan(ctx, artifactPath, artifactType)
}

func (pa *PluginAdapter) IsAvailable() bool {
	return pa.scanner.IsAvailable()
}

func (pa *PluginAdapter) Supports(artifactType string) bool {
	return pa.scanner.Supports(artifactType)
}

// ScannerPlugin interface implementation
func (pa *PluginAdapter) GetMetadata() PluginMetadata {
	return pa.metadata
}

func (pa *PluginAdapter) GetCapabilities() []Capability {
	// Convert simplified metadata to capabilities
	capabilities := make([]Capability, 0)

	// Add basic capabilities based on scanner type
	switch pa.scanner.Name() {
	case "syft":
		capabilities = append(capabilities, Capability{
			Type:        CapabilitySBOMGeneration,
			Description: "Generate Software Bill of Materials",
			Confidence:  0.95,
		})
	case "grype":
		capabilities = append(capabilities, Capability{
			Type:        CapabilityCVE,
			Description: "Detect CVE vulnerabilities",
			Confidence:  0.92,
		})
	case "trivy":
		capabilities = append(capabilities, []Capability{
			{
				Type:        CapabilityCVE,
				Description: "Detect vulnerabilities",
				Confidence:  0.88,
			},
			{
				Type:        CapabilitySecretDetection,
				Description: "Detect secrets",
				Confidence:  0.85,
			},
		}...)
	case "bandit":
		capabilities = append(capabilities, Capability{
			Type:        CapabilityStaticAnalysis,
			Description: "Python security analysis",
			Confidence:  0.90,
		})
	}

	return capabilities
}

func (pa *PluginAdapter) GetDependencies() []string {
	// Most existing scanners don't have dependencies
	return []string{}
}

func (pa *PluginAdapter) Initialize(config PluginConfig) error {
	// Most existing scanners don't need explicit initialization
	pa.health.Status = HealthHealthy
	pa.health.Message = "Initialized successfully"
	pa.health.CheckedAt = time.Now()
	return nil
}

func (pa *PluginAdapter) Shutdown() error {
	// Most existing scanners don't need explicit shutdown
	return nil
}

func (pa *PluginAdapter) HealthCheck() HealthStatus {
	// Update health status based on scanner availability
	if !pa.scanner.IsAvailable() {
		pa.health.Status = HealthUnhealthy
		pa.health.Message = "Scanner not available"
	} else {
		pa.health.Status = HealthHealthy
		pa.health.Message = "Scanner available"
	}

	pa.health.CheckedAt = time.Now()
	return *pa.health
}

func (pa *PluginAdapter) ValidateConfig(config PluginConfig) error {
	// Basic validation - check if scanner is available
	if !pa.scanner.IsAvailable() {
		return fmt.Errorf("scanner %s is not available", pa.scanner.Name())
	}
	return nil
}

func (pa *PluginAdapter) GetDefaultConfig() PluginConfig {
	return PluginConfig{
		Settings: make(map[string]interface{}),
		Timeout:  5 * time.Minute,
		Retries:  3,
		Enabled:  true,
	}
}

// PluginAdapterFactory creates plugin adapters for all existing scanners
type PluginAdapterFactory struct {
	scannerRegistry *ScannerRegistry
}

// NewPluginAdapterFactory creates a new adapter factory
func NewPluginAdapterFactory(registry *ScannerRegistry) *PluginAdapterFactory {
	return &PluginAdapterFactory{
		scannerRegistry: registry,
	}
}

// CreateAdapters creates plugin adapters for all registered scanners
func (factory *PluginAdapterFactory) CreateAdapters() []ScannerPlugin {
	var adapters []ScannerPlugin

	// Create adapters for all known scanner types
	scannerNames := []string{"syft", "grype", "trivy", "bandit"}

	for _, name := range scannerNames {
		// Get scanners that support different artifact types to find this scanner
		allTypes := []string{"docker", "npm", "maven", "python", "generic"}
		var scanner Scanner

		for _, artifactType := range allTypes {
			scanners := factory.scannerRegistry.GetScannersForType(artifactType)
			for _, s := range scanners {
				if s.Name() == name {
					scanner = s
					break
				}
			}
			if scanner != nil {
				break
			}
		}

		if scanner != nil {
			metadata := factory.createMetadataForScanner(scanner)
			adapter := NewPluginAdapter(scanner, metadata)
			adapters = append(adapters, adapter)
		}
	}

	return adapters
}

// createMetadataForScanner creates metadata for a scanner based on its type
func (factory *PluginAdapterFactory) createMetadataForScanner(scanner Scanner) PluginMetadata {
	scannerName := scanner.Name()

	switch scannerName {
	case "syft":
		return factory.createSyftMetadata()
	case "grype":
		return factory.createGrypeMetadata()
	case "trivy":
		return factory.createTrivyMetadata()
	case "bandit":
		return factory.createBanditMetadata()
	default:
		return factory.createGenericMetadata(scanner)
	}
}

// Scanner-specific metadata creation
func (factory *PluginAdapterFactory) createSyftMetadata() PluginMetadata {
	return PluginMetadata{
		ID:          "syft-adapter",
		Name:        "Syft Scanner",
		Version:     "1.0.0",
		Description: "SBOM generation and component analysis scanner",
		Author:      "Anchore",
		Categories:  []ScannerCategory{CategorySBOM},
		Tags:        []string{"sbom", "packages", "dependencies"},
		Priority:    7,
		Resources: ResourceRequirements{
			CPU:     ResourceMedium,
			Memory:  ResourceMedium,
			Disk:    ResourceLow,
			Network: ResourceLow,
		},
	}
}

func (factory *PluginAdapterFactory) createGrypeMetadata() PluginMetadata {
	return PluginMetadata{
		ID:          "grype-adapter",
		Name:        "Grype Scanner",
		Version:     "1.0.0",
		Description: "Vulnerability scanner for container images and filesystems",
		Author:      "Anchore",
		Categories:  []ScannerCategory{CategoryVulnerability},
		Tags:        []string{"vulnerability", "cve", "security"},
		Priority:    8,
		Resources: ResourceRequirements{
			CPU:     ResourceMedium,
			Memory:  ResourceMedium,
			Disk:    ResourceMedium,
			Network: ResourceMedium,
		},
	}
}

func (factory *PluginAdapterFactory) createTrivyMetadata() PluginMetadata {
	return PluginMetadata{
		ID:          "trivy-adapter",
		Name:        "Trivy Scanner",
		Version:     "1.0.0",
		Description: "Comprehensive security scanner for vulnerabilities, misconfigurations, secrets, and licenses",
		Author:      "Aqua Security",
		Categories:  []ScannerCategory{CategoryVulnerability, CategorySecurity},
		Tags:        []string{"vulnerability", "secrets", "misconfig", "compliance"},
		Priority:    6, // Lower due to current JSON parsing issues
		Resources: ResourceRequirements{
			CPU:     ResourceHigh,
			Memory:  ResourceHigh,
			Disk:    ResourceMedium,
			Network: ResourceMedium,
		},
	}
}

func (factory *PluginAdapterFactory) createBanditMetadata() PluginMetadata {
	return PluginMetadata{
		ID:          "bandit-adapter",
		Name:        "Bandit Scanner",
		Version:     "1.0.0",
		Description: "Security linter for Python code",
		Author:      "OpenStack Security",
		Categories:  []ScannerCategory{CategorySecurity},
		Tags:        []string{"python", "sast", "security", "linting"},
		Priority:    9,
		Resources: ResourceRequirements{
			CPU:     ResourceLow,
			Memory:  ResourceLow,
			Disk:    ResourceLow,
			Network: ResourceLow,
		},
	}
}

func (factory *PluginAdapterFactory) createGenericMetadata(scanner Scanner) PluginMetadata {
	return PluginMetadata{
		ID:          fmt.Sprintf("%s-adapter", scanner.Name()),
		Name:        fmt.Sprintf("%s Scanner", scanner.Name()),
		Version:     "1.0.0",
		Description: fmt.Sprintf("Generic adapter for %s scanner", scanner.Name()),
		Author:      "SecureStore",
		Categories:  []ScannerCategory{CategorySecurity},
		Tags:        []string{"generic"},
		Priority:    5,
		Resources: ResourceRequirements{
			CPU:     ResourceMedium,
			Memory:  ResourceMedium,
			Disk:    ResourceMedium,
			Network: ResourceLow,
		},
	}
}

// PluginIntegrationManager manages the integration of plugins with existing systems
type PluginIntegrationManager struct {
	pluginManager  *PluginManager
	workflowEngine *WorkflowEngine
	adapterFactory *PluginAdapterFactory
	logger         Logger
}

// NewPluginIntegrationManager creates a new integration manager
func NewPluginIntegrationManager(
	pluginManager *PluginManager,
	workflowEngine *WorkflowEngine,
	scannerRegistry *ScannerRegistry,
	logger Logger,
) *PluginIntegrationManager {
	return &PluginIntegrationManager{
		pluginManager:  pluginManager,
		workflowEngine: workflowEngine,
		adapterFactory: NewPluginAdapterFactory(scannerRegistry),
		logger:         logger,
	}
}

// Initialize sets up the plugin system with existing scanners
func (pim *PluginIntegrationManager) Initialize() error {
	pim.logger.Printf("[PLUGIN_INTEGRATION] Initializing plugin system")

	// Create adapters for existing scanners
	adapters := pim.adapterFactory.CreateAdapters()

	// Register all adapters as plugins
	for _, adapter := range adapters {
		if err := pim.pluginManager.RegisterPlugin(adapter); err != nil {
			pim.logger.Printf("[PLUGIN_INTEGRATION] Failed to register plugin %s: %v", adapter.Name(), err)
			continue
		}
		pim.logger.Printf("[PLUGIN_INTEGRATION] Registered plugin: %s", adapter.Name())
	}

	pim.logger.Printf("[PLUGIN_INTEGRATION] Plugin system initialized with %d plugins", len(adapters))
	return nil
}

// GetCompatibleWorkflows returns workflows compatible with available plugins
func (pim *PluginIntegrationManager) GetCompatibleWorkflows(artifactType string) []*ScanWorkflow {
	workflows := pim.workflowEngine.GetWorkflowsForArtifactType(artifactType)

	// Filter workflows based on plugin availability
	var compatible []*ScanWorkflow
	for _, workflow := range workflows {
		if pim.isWorkflowCompatible(workflow) {
			compatible = append(compatible, workflow)
		}
	}

	return compatible
}

// isWorkflowCompatible checks if a workflow can be executed with available plugins
func (pim *PluginIntegrationManager) isWorkflowCompatible(workflow *ScanWorkflow) bool {
	// Check if we have plugins that can satisfy the workflow requirements
	// Get all plugins by checking each category
	availablePlugins := pim.pluginManager.GetPluginsByCategory(CategoryVulnerability)
	availablePlugins = append(availablePlugins, pim.pluginManager.GetPluginsByCategory(CategorySecurity)...)

	// Simple compatibility check - ensure we have at least one plugin
	// More sophisticated logic would check specific capability requirements
	return len(availablePlugins) > 0
}

// ExecuteWithPlugins executes a scan using the plugin system
func (pim *PluginIntegrationManager) ExecuteWithPlugins(
	ctx context.Context,
	artifactPath string,
	artifactType string,
	workflowID string,
) (*WorkflowResult, error) {
	request := WorkflowExecutionRequest{
		ArtifactPath: artifactPath,
		ArtifactType: artifactType,
		Options:      make(map[string]interface{}),
	}

	return pim.workflowEngine.ExecuteWorkflow(ctx, workflowID, request)
}

// GetPluginStatus returns the status of all plugins
func (pim *PluginIntegrationManager) GetPluginStatus() map[string]HealthStatus {
	status := make(map[string]HealthStatus)

	// Get plugins from all categories
	categories := []ScannerCategory{CategoryVulnerability, CategorySecurity, CategorySBOM}
	for _, category := range categories {
		plugins := pim.pluginManager.GetPluginsByCategory(category)
		for _, plugin := range plugins {
			status[plugin.GetMetadata().ID] = plugin.HealthCheck()
		}
	}

	return status
}
