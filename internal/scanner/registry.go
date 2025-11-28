package scanner

import (
	"fmt"
	"sync"
)

// ScannerRegistry manages scanner registration and selection
type ScannerRegistry struct {
	scanners  map[string]Scanner
	workflows map[string]*WorkflowConfig
	mu        sync.RWMutex
}

// WorkflowConfig defines scanner workflows for different artifact types
type WorkflowConfig struct {
	Name          string
	ArtifactTypes []string
	Scanners      []string
	Parallel      bool
	Timeout       string
	Priority      []string
}

// NewScannerRegistry creates a new scanner registry
func NewScannerRegistry() *ScannerRegistry {
	registry := &ScannerRegistry{
		scanners:  make(map[string]Scanner),
		workflows: make(map[string]*WorkflowConfig),
	}

	// Initialize default workflows
	registry.initializeDefaultWorkflows()

	return registry
}

// RegisterScanner registers a new scanner
func (r *ScannerRegistry) RegisterScanner(scanner Scanner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := scanner.Name()
	if _, exists := r.scanners[name]; exists {
		return fmt.Errorf("scanner %s already registered", name)
	}

	// Only register if scanner is available
	if !scanner.IsAvailable() {
		return fmt.Errorf("scanner %s is not available", name)
	}

	r.scanners[name] = scanner
	fmt.Printf("[REGISTRY] Registered scanner: %s, types: %v\n", name, scanner.SupportedTypes())
	return nil
}

// GetScannersForType returns all scanners that support the given artifact type
func (r *ScannerRegistry) GetScannersForType(artifactType string) []Scanner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchingScanners []Scanner
	for _, scanner := range r.scanners {
		for _, supportedType := range scanner.SupportedTypes() {
			if supportedType == artifactType || supportedType == "generic" {
				matchingScanners = append(matchingScanners, scanner)
				break
			}
		}
	}

	return matchingScanners
}

// GetScannersByWorkflow returns scanners for a specific workflow
func (r *ScannerRegistry) GetScannersByWorkflow(workflowName string) ([]Scanner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workflow, exists := r.workflows[workflowName]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowName)
	}

	var scanners []Scanner
	for _, scannerName := range workflow.Scanners {
		if scanner, exists := r.scanners[scannerName]; exists {
			scanners = append(scanners, scanner)
		}
	}

	return scanners, nil
}

// RegisterWorkflow registers a new workflow configuration
func (r *ScannerRegistry) RegisterWorkflow(workflow *WorkflowConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.workflows[workflow.Name] = workflow
	fmt.Printf("[REGISTRY] Registered workflow: %s for types: %v\n", workflow.Name, workflow.ArtifactTypes)
}

// initializeDefaultWorkflows sets up the default scanning workflows
func (r *ScannerRegistry) initializeDefaultWorkflows() {
	// PyPI/Python Workflow - Comprehensive Python security scanning
	r.workflows["python"] = &WorkflowConfig{
		Name:          "python",
		ArtifactTypes: []string{"pypi", "python", "wheel", "sdist"},
		Scanners:      []string{"Syft/Grype", "OSV-Scanner (SBOM-based CVE)", "OWASP Dep-Scan", "Bandit", "TruffleHog Secret Scanner"},
		Parallel:      true,
		Timeout:       "5m",
		Priority:      []string{"Bandit", "Syft/Grype", "OWASP Dep-Scan"},
	}

	// Docker Workflow - Container security scanning
	r.workflows["docker"] = &WorkflowConfig{
		Name:          "docker",
		ArtifactTypes: []string{"docker"},
		Scanners:      []string{"Docker/OCI Manifest Scanner (Syft+Grype+Trivy)", "Syft/Grype", "Trivy OS/Container Scanner", "TruffleHog Secret Scanner"},
		Parallel:      true,
		Timeout:       "10m",
		Priority:      []string{"Trivy OS/Container Scanner", "Syft/Grype"},
	}

	// Maven Workflow - Java dependency scanning
	r.workflows["maven"] = &WorkflowConfig{
		Name:          "maven",
		ArtifactTypes: []string{"maven"},
		Scanners:      []string{"Syft/Grype", "OSV-Scanner (SBOM-based CVE)", "OWASP Dep-Scan", "TruffleHog Secret Scanner"},
		Parallel:      true,
		Timeout:       "5m",
		Priority:      []string{"Syft/Grype", "OWASP Dep-Scan"},
	}

	// NPM Workflow - JavaScript/Node.js security scanning
	r.workflows["npm"] = &WorkflowConfig{
		Name:          "npm",
		ArtifactTypes: []string{"npm", "nodejs", "javascript"},
		Scanners:      []string{"NPM Package Scanner (OWASP dep-scan + Blint)", "Syft/Grype", "OSV-Scanner (SBOM-based CVE)", "TruffleHog Secret Scanner"},
		Parallel:      true,
		Timeout:       "5m",
		Priority:      []string{"NPM Package Scanner (OWASP dep-scan + Blint)", "Syft/Grype"},
	}

	// Generic Workflow - Fallback for unknown types
	r.workflows["generic"] = &WorkflowConfig{
		Name:          "generic",
		ArtifactTypes: []string{"generic"},
		Scanners:      []string{"TruffleHog Secret Scanner"},
		Parallel:      false,
		Timeout:       "2m",
		Priority:      []string{"TruffleHog Secret Scanner"},
	}
}

// GetWorkflowForType returns the appropriate workflow for an artifact type
func (r *ScannerRegistry) GetWorkflowForType(artifactType string) *WorkflowConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, workflow := range r.workflows {
		for _, supportedType := range workflow.ArtifactTypes {
			if supportedType == artifactType {
				return workflow
			}
		}
	}

	// Return generic workflow as fallback
	return r.workflows["generic"]
}

// GetScannersByArtifactType returns scanners for a specific artifact type by finding the appropriate workflow
func (r *ScannerRegistry) GetScannersByArtifactType(artifactType string) ([]Scanner, error) {
	workflow := r.GetWorkflowForType(artifactType)
	if workflow == nil {
		return []Scanner{}, fmt.Errorf("no workflow found for artifact type: %s", artifactType)
	}

	return r.GetScannersByWorkflow(workflow.Name)
}
