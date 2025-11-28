package scanner

import (
	"context"
	"fmt"
	"time"
)

// ScanPipeline represents a unified scanning pipeline
type ScanPipeline struct {
	registry   *ScannerRegistry
	executor   *ParallelExecutor
	aggregator *ResultAggregator
	logger     Logger
}

// ScanRequest encapsulates all scanning parameters
type ScanRequest struct {
	ArtifactPath string
	ArtifactType string
	WorkflowName string
	Scanners     []Scanner
	Concurrency  int
	Timeout      time.Duration
	Options      ScanOptions
}

// ScanOptions provides additional scanning configuration
type ScanOptions struct {
	FailFast        bool
	RetryCount      int
	SkipUnavailable bool
	Priority        []string // Scanner priority order
}

// Logger interface for pipeline logging
type Logger interface {
	Printf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// NewScanPipeline creates a new unified scanning pipeline
func NewScanPipeline(registry *ScannerRegistry, logger Logger) *ScanPipeline {
	return &ScanPipeline{
		registry:   registry,
		executor:   NewParallelExecutor(logger),
		aggregator: NewResultAggregator(),
		logger:     logger,
	}
}

// Execute runs the complete scanning pipeline
func (p *ScanPipeline) Execute(ctx context.Context, request *ScanRequest) (*AggregatedScanResult, error) {
	p.logger.Printf("[PIPELINE] Starting scan pipeline for %s (type: %s)", request.ArtifactPath, request.ArtifactType)

	// Step 1: Validate request
	if err := p.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid scan request: %w", err)
	}

	// Step 2: Select scanners based on workflow or type
	scanners, err := p.selectScanners(request)
	if err != nil {
		return nil, fmt.Errorf("scanner selection failed: %w", err)
	}

	p.logger.Printf("[PIPELINE] Selected %d scanners: %v", len(scanners), getScannerNames(scanners))

	// Step 3: Execute scanners in parallel
	results, err := p.executor.Execute(ctx, request, scanners)
	if err != nil {
		return nil, fmt.Errorf("parallel execution failed: %w", err)
	}

	p.logger.Printf("[PIPELINE] Parallel execution completed: %d results", len(results))

	// Step 4: Aggregate results
	aggregated := p.aggregator.Aggregate(results)
	aggregated.ArtifactPath = request.ArtifactPath
	aggregated.ArtifactType = request.ArtifactType
	aggregated.WorkflowName = request.WorkflowName

	p.logger.Printf("[PIPELINE] Pipeline completed: %d total vulnerabilities", len(aggregated.Vulnerabilities))

	return aggregated, nil
}

// validateRequest validates the scan request
func (p *ScanPipeline) validateRequest(request *ScanRequest) error {
	if request.ArtifactPath == "" {
		return fmt.Errorf("artifact path is required")
	}
	if request.ArtifactType == "" {
		return fmt.Errorf("artifact type is required")
	}
	if request.Timeout == 0 {
		request.Timeout = 5 * time.Minute // Default timeout
	}
	if request.Concurrency == 0 {
		request.Concurrency = 10 // Default concurrency
	}
	return nil
}

// selectScanners selects appropriate scanners based on workflow or artifact type
func (p *ScanPipeline) selectScanners(request *ScanRequest) ([]Scanner, error) {
	// If scanners are explicitly provided, use them
	if len(request.Scanners) > 0 {
		return request.Scanners, nil
	}

	// If workflow is specified, use workflow-based selection
	if request.WorkflowName != "" {
		return p.registry.GetScannersByWorkflow(request.WorkflowName)
	}

	// Fall back to type-based selection
	return p.registry.GetScannersForType(request.ArtifactType), nil
}

// getScannerNames extracts scanner names for logging
func getScannerNames(scanners []Scanner) []string {
	names := make([]string, len(scanners))
	for i, scanner := range scanners {
		names[i] = scanner.Name()
	}
	return names
}
