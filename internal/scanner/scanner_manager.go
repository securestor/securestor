package scanner

import (
	"context"
	"fmt"
	"os"
)

// ScannerManager manages multiple vulnerability scanners using unified pipeline
type ScannerManager struct {
	registry *ScannerRegistry
	pipeline *ScanPipeline
	logger   SimpleLogger
}

// SimpleLogger implements the Logger interface for ScannerManager
type SimpleLogger struct{}

func (l SimpleLogger) Printf(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (l SimpleLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func (l SimpleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// NewScannerManager creates a new scanner manager with unified pipeline
func NewScannerManager() *ScannerManager {
	fmt.Fprintf(os.Stderr, "[SCANNER_DEBUG] NewScannerManager called\n")

	logger := SimpleLogger{}
	registry := NewScannerRegistry()

	sm := &ScannerManager{
		registry: registry,
		pipeline: NewScanPipeline(registry, logger),
		logger:   logger,
	}

	fmt.Fprintf(os.Stderr, "[SCANNER_DEBUG] Starting scanner registration\n")

	// Register all available scanners
	sm.registerAllScanners()

	return sm
}

// registerAllScanners registers all available scanners in the registry
func (sm *ScannerManager) registerAllScanners() {
	// SBOM generation & CVE scanning
	sm.RegisterScanner(NewSyftScanner()) // Syft: SBOM generation + Grype CVE scan
	sm.RegisterScanner(NewOSVScanner())  // OSV-Scanner: SBOM-based CVE scan

	// Dependency-level CVE scanning
	sm.RegisterScanner(NewDepScanScanner()) // dep-scan: Application package scanning

	// OS & container scanning
	sm.RegisterScanner(NewTrivyScanner())          // Trivy: OS package vulnerabilities
	sm.RegisterScanner(NewDockerManifestScanner()) // Docker/OCI manifest scanning

	// Secret detection
	sm.RegisterScanner(NewTruffleHogScanner()) // TruffleHog: Secrets and credentials

	// Code security analysis
	sm.RegisterScanner(NewBanditScanner()) // Bandit: Python code security analysis

	// Package-specific scanners
	sm.RegisterScanner(NewNPMPackageScanner()) // NPM package vulnerabilities

	// Helm chart scanner
	sm.RegisterScanner(NewHelmChartScanner()) // Helm: Chart security and best practices
}

// RegisterScanner adds a new scanner to the registry
func (sm *ScannerManager) RegisterScanner(scanner Scanner) {
	// Write debug info to stderr and a file
	fmt.Fprintf(os.Stderr, "[SCANNER_DEBUG] Checking scanner: %s\n", scanner.Name())
	os.WriteFile("/app/scanner_debug.log", []byte(fmt.Sprintf("Checking scanner: %s\n", scanner.Name())), os.ModeAppend|0644)

	if !scanner.IsAvailable() {
		fmt.Fprintf(os.Stderr, "[SCANNER_DEBUG] Scanner %s is NOT AVAILABLE\n", scanner.Name())
		os.WriteFile("/app/scanner_debug.log", []byte(fmt.Sprintf("Scanner %s is NOT AVAILABLE\n", scanner.Name())), os.ModeAppend|0644)
		return // Skip unavailable scanners
	}

	fmt.Fprintf(os.Stderr, "[SCANNER_DEBUG] REGISTERING scanner: %s, supported types: %v\n", scanner.Name(), scanner.SupportedTypes())
	os.WriteFile("/app/scanner_debug.log", []byte(fmt.Sprintf("REGISTERING scanner: %s, supported types: %v\n", scanner.Name(), scanner.SupportedTypes())), os.ModeAppend|0644)

	// Register scanner in the registry
	err := sm.registry.RegisterScanner(scanner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SCANNER_DEBUG] Failed to register scanner %s: %v\n", scanner.Name(), err)
	}
}

// GetScannersForType returns all scanners that support the given artifact type
func (sm *ScannerManager) GetScannersForType(artifactType string) []Scanner {
	fmt.Printf("[DEBUG] GetScannersForType: Looking for scanners for type '%s'\n", artifactType)

	scanners, err := sm.registry.GetScannersByArtifactType(artifactType)
	if err != nil {
		fmt.Printf("[DEBUG] Error getting scanners for artifact type %s: %v\n", artifactType, err)
		return []Scanner{}
	}

	// Extract scanner names for debug logging
	scannerNames := make([]string, len(scanners))
	for i, scanner := range scanners {
		scannerNames[i] = scanner.Name()
	}

	fmt.Printf("[DEBUG] Found %d matching scanners for type '%s': %v\n", len(scanners), artifactType, scannerNames)
	return scanners
}

// ScanWithAll scans artifact with all available scanners using unified pipeline
func (sm *ScannerManager) ScanWithAll(ctx context.Context, artifactPath string, artifactType string) ([]*ScanResult, error) {
	fmt.Printf("[SCAN_DEBUG] ScanWithAll called: path=%s, type=%s\n", artifactPath, artifactType)

	// Get the appropriate workflow for this artifact type
	workflow := sm.registry.GetWorkflowForType(artifactType)
	workflowName := "generic" // fallback
	if workflow != nil {
		workflowName = workflow.Name
	}

	// Create scan request for the unified pipeline
	request := &ScanRequest{
		ArtifactPath: artifactPath,
		ArtifactType: artifactType,
		WorkflowName: workflowName,
	}

	// Execute unified scanning pipeline
	aggregated, err := sm.pipeline.Execute(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Return the individual scanner results from the aggregated result
	if aggregated != nil && len(aggregated.ScannerResults) > 0 {
		return aggregated.ScannerResults, nil
	}

	return []*ScanResult{}, nil
}

// AggregateScanResults combines results from multiple scanners using unified aggregator
func (sm *ScannerManager) AggregateScanResults(results []*ScanResult) *ScanResult {
	if len(results) == 0 {
		return nil
	}

	// Use the unified aggregator
	aggregated := sm.pipeline.aggregator.Aggregate(results)

	// Convert AggregatedScanResult to ScanResult for backward compatibility
	return &ScanResult{
		ScannerName:     "Aggregated",
		ArtifactType:    aggregated.ArtifactType,
		Vulnerabilities: aggregated.Vulnerabilities,
		Summary:         aggregated.Summary,
		ScanDuration:    aggregated.ScanDuration.Seconds(),
		Metadata: map[string]interface{}{
			"scanners_used":    extractScannerNames(aggregated.ScannerResults),
			"scanner_count":    aggregated.TotalScanners,
			"successful_scans": aggregated.SuccessfulScans,
			"failed_scans":     aggregated.FailedScans,
			"workflow_name":    aggregated.WorkflowName,
		},
	}
}

// extractScannerNames extracts scanner names from results
func extractScannerNames(results []*ScanResult) []string {
	names := make([]string, len(results))
	for i, result := range results {
		names[i] = result.ScannerName
	}
	return names
}
