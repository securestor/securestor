package scanner

import (
	"strings"
)

func normalizeSeverity(severity string) string {
	severity = strings.ToUpper(strings.TrimSpace(severity))

	switch severity {
	case "CRITICAL", "CRIT":
		return "CRITICAL"
	case "HIGH", "H":
		return "HIGH"
	case "MEDIUM", "MED", "M", "MODERATE":
		return "MEDIUM"
	case "LOW", "L":
		return "LOW"
	default:
		return "UNKNOWN"
	}
}

func calculateSummary(vulnerabilities []Vulnerability) VulnerabilitySummary {
	summary := VulnerabilitySummary{}

	for _, vuln := range vulnerabilities {
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
		summary.Total++
	}

	return summary
}
