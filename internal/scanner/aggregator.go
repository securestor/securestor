package scanner

import (
	"sort"
	"time"
)

// ResultAggregator aggregates results from multiple scanners
type ResultAggregator struct{}

// AggregatedScanResult represents the final aggregated result
type AggregatedScanResult struct {
	ArtifactPath    string               `json:"artifact_path"`
	ArtifactType    string               `json:"artifact_type"`
	WorkflowName    string               `json:"workflow_name"`
	Vulnerabilities []Vulnerability      `json:"vulnerabilities"`
	Summary         VulnerabilitySummary `json:"summary"`
	ScannerResults  []*ScanResult        `json:"scanner_results"`
	TotalScanners   int                  `json:"total_scanners"`
	SuccessfulScans int                  `json:"successful_scans"`
	FailedScans     int                  `json:"failed_scans"`
	ScanDuration    time.Duration        `json:"scan_duration"`
	Timestamp       time.Time            `json:"timestamp"`
}

// NewResultAggregator creates a new result aggregator
func NewResultAggregator() *ResultAggregator {
	return &ResultAggregator{}
}

// Aggregate combines results from multiple scanners
func (a *ResultAggregator) Aggregate(results []*ScanResult) *AggregatedScanResult {
	aggregated := &AggregatedScanResult{
		ScannerResults:  results,
		TotalScanners:   len(results),
		SuccessfulScans: len(results),
		Timestamp:       time.Now(),
	}

	// Collect all vulnerabilities
	var allVulnerabilities []Vulnerability
	totalDuration := time.Duration(0)

	for _, result := range results {
		allVulnerabilities = append(allVulnerabilities, result.Vulnerabilities...)
		totalDuration += time.Duration(result.ScanDuration * float64(time.Second))
	}

	aggregated.ScanDuration = totalDuration

	// Deduplicate vulnerabilities
	aggregated.Vulnerabilities = a.DeduplicateVulnerabilities(allVulnerabilities)

	// Calculate summary
	aggregated.Summary = a.CalculateSummary(aggregated.Vulnerabilities)

	// Sort vulnerabilities by severity and CVSS score
	aggregated.Vulnerabilities = a.SortVulnerabilities(aggregated.Vulnerabilities)

	return aggregated
}

// DeduplicateVulnerabilities removes duplicate vulnerabilities
func (a *ResultAggregator) DeduplicateVulnerabilities(vulnerabilities []Vulnerability) []Vulnerability {
	seen := make(map[string]bool)
	var unique []Vulnerability

	for _, vuln := range vulnerabilities {
		// Create a unique key based on vulnerability characteristics
		key := a.createVulnerabilityKey(vuln)

		if !seen[key] {
			seen[key] = true
			unique = append(unique, vuln)
		}
	}

	return unique
}

// createVulnerabilityKey creates a unique key for vulnerability deduplication
func (a *ResultAggregator) createVulnerabilityKey(vuln Vulnerability) string {
	// Use CVE if available, otherwise use a combination of fields
	if vuln.CVE != "" {
		return vuln.CVE
	}

	// For non-CVE vulnerabilities (like Bandit findings), use package + ID + title
	return vuln.Package + "|" + vuln.ID + "|" + vuln.Title
}

// CalculateSummary calculates vulnerability summary statistics
func (a *ResultAggregator) CalculateSummary(vulnerabilities []Vulnerability) VulnerabilitySummary {
	summary := VulnerabilitySummary{}

	for _, vuln := range vulnerabilities {
		summary.Total++

		switch vuln.Severity {
		case "CRITICAL":
			summary.Critical++
		case "HIGH":
			summary.High++
		case "MEDIUM":
			summary.Medium++
		case "LOW":
			summary.Low++
		}
	}

	return summary
}

// SortVulnerabilities sorts vulnerabilities by severity and CVSS score
func (a *ResultAggregator) SortVulnerabilities(vulnerabilities []Vulnerability) []Vulnerability {
	sorted := make([]Vulnerability, len(vulnerabilities))
	copy(sorted, vulnerabilities)

	sort.Slice(sorted, func(i, j int) bool {
		// First, sort by severity priority
		severityPriority := map[string]int{
			"CRITICAL": 4,
			"HIGH":     3,
			"MEDIUM":   2,
			"LOW":      1,
		}

		priorityI := severityPriority[sorted[i].Severity]
		priorityJ := severityPriority[sorted[j].Severity]

		if priorityI != priorityJ {
			return priorityI > priorityJ
		}

		// If same severity, sort by CVSS score (higher first)
		if sorted[i].CVSS != sorted[j].CVSS {
			return sorted[i].CVSS > sorted[j].CVSS
		}

		// If same CVSS, sort by CVE ID
		return sorted[i].CVE < sorted[j].CVE
	})

	return sorted
}

// FilterBySeverity filters vulnerabilities by minimum severity level
func (a *ResultAggregator) FilterBySeverity(vulnerabilities []Vulnerability, minSeverity string) []Vulnerability {
	severityLevels := map[string]int{
		"INFO":     1,
		"LOW":      2,
		"MEDIUM":   3,
		"HIGH":     4,
		"CRITICAL": 5,
	}

	minLevel := severityLevels[minSeverity]
	if minLevel == 0 {
		return vulnerabilities // Invalid severity, return all
	}

	var filtered []Vulnerability
	for _, vuln := range vulnerabilities {
		if severityLevels[vuln.Severity] >= minLevel {
			filtered = append(filtered, vuln)
		}
	}

	return filtered
}

// GetVulnerabilitiesByScanner returns vulnerabilities grouped by scanner
func (a *ResultAggregator) GetVulnerabilitiesByScanner(results []*ScanResult) map[string][]Vulnerability {
	byScanner := make(map[string][]Vulnerability)

	for _, result := range results {
		byScanner[result.ScannerName] = result.Vulnerabilities
	}

	return byScanner
}
