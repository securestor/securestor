package service

import (
	"github.com/securestor/securestor/internal/models"
)

// CalculateComplianceFromVulnerabilities automatically calculates compliance status and score
// based on vulnerability scan results following industry-standard security policies
func CalculateComplianceFromVulnerabilities(vulnerabilities *models.Vulnerability) (status string, score int) {
	if vulnerabilities == nil {
		return "pending", 0
	}

	critical := vulnerabilities.Critical
	high := vulnerabilities.High
	medium := vulnerabilities.Medium
	low := vulnerabilities.Low

	// Calculate base score (100 = perfect, 0 = worst)
	// Severity weights: Critical=-40, High=-20, Medium=-10, Low=-5
	score = 100
	score -= critical * 40
	score -= high * 20
	score -= medium * 10
	score -= low * 5

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	// Determine compliance status based on severity and score
	// Critical vulnerabilities = immediate non-compliance
	if critical > 0 {
		status = "non-compliant"
	} else if high > 2 {
		// More than 2 high vulnerabilities = non-compliant
		status = "non-compliant"
	} else if high > 0 || medium > 5 {
		// Any high or more than 5 medium = needs review
		status = "review"
	} else if score >= 80 {
		// Score >= 80 with no critical/high = compliant
		status = "compliant"
	} else if score >= 60 {
		// Score 60-79 = needs review
		status = "review"
	} else {
		// Score < 60 = non-compliant
		status = "non-compliant"
	}

	return status, score
}

// CalculateOverallScore calculates an overall security score from scan results
func CalculateOverallScore(scanResults *models.ScanResults) int {
	if scanResults == nil {
		return 0
	}

	// Use the scan's overall score if available
	if scanResults.OverallScore > 0 {
		return scanResults.OverallScore
	}

	// Otherwise calculate from vulnerability results
	if scanResults.VulnerabilityResults != nil {
		_, score := CalculateComplianceFromVulnerabilities(&models.Vulnerability{
			Critical: scanResults.VulnerabilityResults.Critical,
			High:     scanResults.VulnerabilityResults.High,
			Medium:   scanResults.VulnerabilityResults.Medium,
			Low:      scanResults.VulnerabilityResults.Low,
		})
		return score
	}

	return 50 // Default neutral score
}
