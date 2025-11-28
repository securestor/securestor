package scanner

import (
	"context"
	"sort"
	"time"
)

// OptimalSelectionStrategy balances speed, accuracy, and coverage
type OptimalSelectionStrategy struct{}

func NewOptimalSelectionStrategy() *OptimalSelectionStrategy {
	return &OptimalSelectionStrategy{}
}

func (s *OptimalSelectionStrategy) GetName() string {
	return "optimal"
}

func (s *OptimalSelectionStrategy) GetDescription() string {
	return "Balances speed, accuracy, and coverage for optimal results"
}

func (s *OptimalSelectionStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	// Get all eligible scanners
	eligible := s.getEligibleScanners(request)

	// Score each scanner
	scored := s.scoreScannersOptimal(eligible, request)

	// Sort by score (highest first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Select top scanners within limits
	return s.selectTopScanners(scored, request), nil
}

func (s *OptimalSelectionStrategy) getEligibleScanners(request ScanSelectionRequest) []ScannerPlugin {
	// This would be implemented to filter by artifact type, capabilities, etc.
	// For now, returning empty slice - implementation would get from plugin manager
	return []ScannerPlugin{}
}

func (s *OptimalSelectionStrategy) scoreScannersOptimal(scanners []ScannerPlugin, request ScanSelectionRequest) []ScoredScanner {
	var scored []ScoredScanner

	for _, scanner := range scanners {
		score := s.calculateOptimalScore(scanner, request)
		scored = append(scored, ScoredScanner{
			Plugin: scanner,
			Score:  score,
		})
	}

	return scored
}

func (s *OptimalSelectionStrategy) calculateOptimalScore(scanner ScannerPlugin, request ScanSelectionRequest) float64 {
	metadata := scanner.GetMetadata()
	capabilities := scanner.GetCapabilities()

	score := 0.0

	// Base priority score (0-100)
	score += float64(metadata.Priority)

	// Capability matching score (0-200)
	capabilityScore := 0.0
	if len(request.RequiredCapabilities) > 0 {
		for _, reqCap := range request.RequiredCapabilities {
			for _, cap := range capabilities {
				if cap.Type == reqCap {
					capabilityScore += cap.Confidence * 50 // Max 50 per capability
					break
				}
			}
		}
	}
	score += capabilityScore

	// Performance score (0-100)
	if len(capabilities) > 0 {
		avgPerformance := 0.0
		for _, cap := range capabilities {
			perfScore := s.calculatePerformanceScore(cap.Performance)
			avgPerformance += perfScore
		}
		score += avgPerformance / float64(len(capabilities))
	}

	// Resource efficiency score (0-50)
	resourceScore := s.calculateResourceScore(metadata.Resources, request.ResourceLimits)
	score += resourceScore

	return score
}

func (s *OptimalSelectionStrategy) calculatePerformanceScore(perf Performance) float64 {
	score := 0.0

	// Speed score
	switch perf.Speed {
	case PerformanceFast:
		score += 30
	case PerformanceMedium:
		score += 20
	case PerformanceSlow:
		score += 10
	}

	// Accuracy score
	switch perf.Accuracy {
	case PerformanceHigh:
		score += 25
	case PerformanceMedium:
		score += 15
	case PerformanceLow:
		score += 5
	}

	// Coverage score
	switch perf.Coverage {
	case PerformanceComprehensive:
		score += 25
	case PerformanceModerate:
		score += 15
	case PerformanceBasic:
		score += 5
	}

	// False positive penalty
	switch perf.FalsePositives {
	case PerformanceLow:
		score += 20
	case PerformanceMedium:
		score += 10
	case PerformanceHigh:
		score -= 10
	}

	return score
}

func (s *OptimalSelectionStrategy) calculateResourceScore(required, available ResourceRequirements) float64 {
	score := 50.0 // Base score

	// Penalize high resource usage
	if required.CPU == ResourceHigh {
		score -= 15
	} else if required.CPU == ResourceMedium {
		score -= 5
	}

	if required.Memory == ResourceHigh {
		score -= 15
	} else if required.Memory == ResourceMedium {
		score -= 5
	}

	return score
}

func (s *OptimalSelectionStrategy) selectTopScanners(scored []ScoredScanner, request ScanSelectionRequest) []ScannerPlugin {
	maxScanners := request.MaxScanners
	if maxScanners <= 0 {
		maxScanners = 5 // Default limit
	}

	selected := make([]ScannerPlugin, 0, maxScanners)
	for i, scanner := range scored {
		if i >= maxScanners {
			break
		}
		selected = append(selected, scanner.Plugin)
	}

	return selected
}

// FastestSelectionStrategy prioritizes speed over accuracy
type FastestSelectionStrategy struct{}

func NewFastestSelectionStrategy() *FastestSelectionStrategy {
	return &FastestSelectionStrategy{}
}

func (s *FastestSelectionStrategy) GetName() string {
	return "fastest"
}

func (s *FastestSelectionStrategy) GetDescription() string {
	return "Prioritizes speed over accuracy for quick scans"
}

func (s *FastestSelectionStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	// Implementation similar to optimal but weights speed heavily
	eligible := s.getEligibleScanners(request)
	scored := s.scoreScannersFastest(eligible, request)

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return s.selectTopScanners(scored, request), nil
}

func (s *FastestSelectionStrategy) getEligibleScanners(request ScanSelectionRequest) []ScannerPlugin {
	return []ScannerPlugin{} // Implementation would filter scanners
}

func (s *FastestSelectionStrategy) scoreScannersFastest(scanners []ScannerPlugin, request ScanSelectionRequest) []ScoredScanner {
	var scored []ScoredScanner

	for _, scanner := range scanners {
		score := s.calculateSpeedScore(scanner, request)
		scored = append(scored, ScoredScanner{
			Plugin: scanner,
			Score:  score,
		})
	}

	return scored
}

func (s *FastestSelectionStrategy) calculateSpeedScore(scanner ScannerPlugin, request ScanSelectionRequest) float64 {
	capabilities := scanner.GetCapabilities()
	score := 0.0

	for _, cap := range capabilities {
		switch cap.Performance.Speed {
		case PerformanceFast:
			score += 100
		case PerformanceMedium:
			score += 50
		case PerformanceSlow:
			score += 10
		}
	}

	return score / float64(len(capabilities))
}

func (s *FastestSelectionStrategy) selectTopScanners(scored []ScoredScanner, request ScanSelectionRequest) []ScannerPlugin {
	maxScanners := request.MaxScanners
	if maxScanners <= 0 {
		maxScanners = 3 // Fewer scanners for speed
	}

	selected := make([]ScannerPlugin, 0, maxScanners)
	for i, scanner := range scored {
		if i >= maxScanners {
			break
		}
		selected = append(selected, scanner.Plugin)
	}

	return selected
}

// ComprehensiveSelectionStrategy prioritizes coverage and accuracy
type ComprehensiveSelectionStrategy struct{}

func NewComprehensiveSelectionStrategy() *ComprehensiveSelectionStrategy {
	return &ComprehensiveSelectionStrategy{}
}

func (s *ComprehensiveSelectionStrategy) GetName() string {
	return "comprehensive"
}

func (s *ComprehensiveSelectionStrategy) GetDescription() string {
	return "Prioritizes coverage and accuracy for thorough analysis"
}

func (s *ComprehensiveSelectionStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	eligible := s.getEligibleScanners(request)
	scored := s.scoreScannersComprehensive(eligible, request)

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return s.selectTopScanners(scored, request), nil
}

func (s *ComprehensiveSelectionStrategy) getEligibleScanners(request ScanSelectionRequest) []ScannerPlugin {
	return []ScannerPlugin{}
}

func (s *ComprehensiveSelectionStrategy) scoreScannersComprehensive(scanners []ScannerPlugin, request ScanSelectionRequest) []ScoredScanner {
	var scored []ScoredScanner

	for _, scanner := range scanners {
		score := s.calculateComprehensiveScore(scanner, request)
		scored = append(scored, ScoredScanner{
			Plugin: scanner,
			Score:  score,
		})
	}

	return scored
}

func (s *ComprehensiveSelectionStrategy) calculateComprehensiveScore(scanner ScannerPlugin, request ScanSelectionRequest) float64 {
	capabilities := scanner.GetCapabilities()
	score := 0.0

	for _, cap := range capabilities {
		// Heavily weight coverage and accuracy
		coverageScore := 0.0
		switch cap.Performance.Coverage {
		case PerformanceComprehensive:
			coverageScore = 100
		case PerformanceModerate:
			coverageScore = 60
		case PerformanceBasic:
			coverageScore = 20
		}

		accuracyScore := 0.0
		switch cap.Performance.Accuracy {
		case PerformanceHigh:
			accuracyScore = 100
		case PerformanceMedium:
			accuracyScore = 60
		case PerformanceLow:
			accuracyScore = 20
		}

		score += (coverageScore + accuracyScore) * cap.Confidence
	}

	return score / float64(len(capabilities))
}

func (s *ComprehensiveSelectionStrategy) selectTopScanners(scored []ScoredScanner, request ScanSelectionRequest) []ScannerPlugin {
	maxScanners := request.MaxScanners
	if maxScanners <= 0 {
		maxScanners = 8 // More scanners for comprehensive coverage
	}

	selected := make([]ScannerPlugin, 0, maxScanners)
	for i, scanner := range scored {
		if i >= maxScanners {
			break
		}
		selected = append(selected, scanner.Plugin)
	}

	return selected
}

// BalancedSelectionStrategy balances all factors equally
type BalancedSelectionStrategy struct{}

func NewBalancedSelectionStrategy() *BalancedSelectionStrategy {
	return &BalancedSelectionStrategy{}
}

func (s *BalancedSelectionStrategy) GetName() string {
	return "balanced"
}

func (s *BalancedSelectionStrategy) GetDescription() string {
	return "Balances speed, accuracy, coverage, and resource usage equally"
}

func (s *BalancedSelectionStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	// Implementation similar to optimal but with equal weighting
	return []ScannerPlugin{}, nil
}

// SecurityFocusedStrategy prioritizes security-specific scanners
type SecurityFocusedStrategy struct{}

func NewSecurityFocusedStrategy() *SecurityFocusedStrategy {
	return &SecurityFocusedStrategy{}
}

func (s *SecurityFocusedStrategy) GetName() string {
	return "security_focused"
}

func (s *SecurityFocusedStrategy) GetDescription() string {
	return "Prioritizes security-focused scanners and capabilities"
}

func (s *SecurityFocusedStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	// Focus on security categories and capabilities
	return []ScannerPlugin{}, nil
}

// ScoredScanner represents a scanner with its selection score
type ScoredScanner struct {
	Plugin ScannerPlugin
	Score  float64
}

// WorkflowBasedStrategy selects scanners based on predefined workflows
type WorkflowBasedStrategy struct {
	workflows map[string][]string // artifact type -> scanner IDs
}

func NewWorkflowBasedStrategy() *WorkflowBasedStrategy {
	wbs := &WorkflowBasedStrategy{
		workflows: make(map[string][]string),
	}

	// Initialize default workflows
	wbs.initializeWorkflows()

	return wbs
}

func (s *WorkflowBasedStrategy) GetName() string {
	return "workflow_based"
}

func (s *WorkflowBasedStrategy) GetDescription() string {
	return "Uses predefined workflows optimized for specific artifact types"
}

func (s *WorkflowBasedStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	_, exists := s.workflows[request.ArtifactType]
	if !exists {
		// Fallback to generic workflow
		_ = s.workflows["generic"]
	}

	// Convert scanner IDs to plugins (would need access to plugin manager)
	// This is a simplified implementation
	return []ScannerPlugin{}, nil
}

func (s *WorkflowBasedStrategy) initializeWorkflows() {
	// Docker / OCI Image workflow
	s.workflows["docker"] = []string{
		"syft_scanner",
		"trivy_scanner",
		"grype_scanner",
		"trufflehog_scanner",
	}

	// Maven (Java JAR/WAR) workflow
	s.workflows["maven"] = []string{
		"syft_scanner",
		"depscan_scanner",
		"trufflehog_scanner",
	}

	// NPM (Node.js) workflow
	s.workflows["npm"] = []string{
		"syft_scanner",
		"depscan_scanner",
		"trufflehog_scanner",
	}

	// PyPI (Python) workflow
	s.workflows["pypi"] = []string{
		"syft_scanner",
		"bandit_scanner",
		"depscan_scanner",
		"trufflehog_scanner",
	}

	// Helm Charts / K8s YAML workflow
	s.workflows["helm"] = []string{
		"syft_scanner",
		"trivy_iac_scanner",
		"checkov_scanner",
	}

	// Generic Binary / Tarball workflow
	s.workflows["generic"] = []string{
		"syft_scanner",
		"trivy_scanner",
		"trufflehog_scanner",
	}
}

// AdaptiveSelectionStrategy learns from past scan results to improve selection
type AdaptiveSelectionStrategy struct {
	history        []ScanHistory
	learningWeight float64
}

type ScanHistory struct {
	ArtifactType  string
	ScannersUsed  []string
	Performance   ScanPerformance
	Effectiveness float64
	Timestamp     time.Time
}

type ScanPerformance struct {
	Duration       time.Duration
	VulnsFound     int
	FalsePositives int
	Coverage       float64
}

func NewAdaptiveSelectionStrategy() *AdaptiveSelectionStrategy {
	return &AdaptiveSelectionStrategy{
		history:        make([]ScanHistory, 0),
		learningWeight: 0.3, // 30% weight for historical data
	}
}

func (s *AdaptiveSelectionStrategy) GetName() string {
	return "adaptive"
}

func (s *AdaptiveSelectionStrategy) GetDescription() string {
	return "Learns from past scan results to continuously improve scanner selection"
}

func (s *AdaptiveSelectionStrategy) SelectScanners(ctx context.Context, request ScanSelectionRequest) ([]ScannerPlugin, error) {
	// Analyze historical performance for this artifact type
	_ = s.analyzeHistory(request.ArtifactType)

	// Combine with current scoring
	// Implementation would merge historical data with current scanner capabilities

	return []ScannerPlugin{}, nil
}

func (s *AdaptiveSelectionStrategy) analyzeHistory(artifactType string) []string {
	// Analyze past performance and return best performing scanner combinations
	return []string{}
}

func (s *AdaptiveSelectionStrategy) RecordScanResult(history ScanHistory) {
	s.history = append(s.history, history)

	// Keep only recent history (last 1000 scans)
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}
}
