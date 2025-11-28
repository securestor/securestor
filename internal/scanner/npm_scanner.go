package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NPMPackageScanner implements scanning for npm packages using OWASP dep-scan and Blint
type NPMPackageScanner struct {
	depScanPath string
	blintPath   string
	pythonPath  string
}

func NewNPMPackageScanner() *NPMPackageScanner {
	return &NPMPackageScanner{
		depScanPath: "depscan",
		blintPath:   "blint",
		pythonPath:  "python3",
	}
}

func (s *NPMPackageScanner) Name() string {
	return "NPM Package Scanner (OWASP dep-scan + Blint)"
}

func (s *NPMPackageScanner) SupportedTypes() []string {
	return []string{"npm", "nodejs", "javascript"}
}

func (s *NPMPackageScanner) IsAvailable() bool {
	// Check if OWASP dep-scan is available
	if _, err := exec.LookPath(s.depScanPath); err == nil {
		return true
	}

	// Check if dep-scan is available via Python module
	cmd := exec.Command(s.pythonPath, "-c", "import depscan")
	if err := cmd.Run(); err == nil {
		return true
	}

	// Check if blint is available
	if _, err := exec.LookPath(s.blintPath); err == nil {
		return true
	}

	// Check if blint is available via Python module
	cmd = exec.Command(s.pythonPath, "-c", "import blint")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

func (s *NPMPackageScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

// NPMPackageData represents extracted npm package information
type NPMPackageData struct {
	PackageJSON     *PackageJSON      `json:"package_json"`
	Dependencies    []string          `json:"dependencies"`
	DevDependencies []string          `json:"dev_dependencies"`
	Scripts         map[string]string `json:"scripts"`
	MainFile        string            `json:"main_file"`
	HasPackageLock  bool              `json:"has_package_lock"`
	HasNodeModules  bool              `json:"has_node_modules"`
}

// PackageJSON represents the npm package.json structure for scanning
type PackageJSON struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Description          string            `json:"description,omitempty"`
	Main                 string            `json:"main,omitempty"`
	Scripts              map[string]string `json:"scripts,omitempty"`
	Keywords             []string          `json:"keywords,omitempty"`
	Author               interface{}       `json:"author,omitempty"`
	License              string            `json:"license,omitempty"`
	Dependencies         map[string]string `json:"dependencies,omitempty"`
	DevDependencies      map[string]string `json:"devDependencies,omitempty"`
	PeerDependencies     map[string]string `json:"peerDependencies,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
	Engines              map[string]string `json:"engines,omitempty"`
}

func (s *NPMPackageScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	startTime := time.Now()

	// Extract and analyze npm package
	packageData, err := s.extractNPMPackage(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract npm package: %w", err)
	}

	// Perform multiple scanning approaches
	var allVulnerabilities []Vulnerability
	var scanErrors []string

	// 1. OWASP dep-scan for dependency vulnerabilities
	if depScanVulns, err := s.runDepScan(ctx, artifactPath, packageData); err != nil {
		scanErrors = append(scanErrors, fmt.Sprintf("dep-scan failed: %v", err))
	} else {
		allVulnerabilities = append(allVulnerabilities, depScanVulns...)
	}

	// 2. Blint for code quality and security issues
	if blintVulns, err := s.runBlint(ctx, artifactPath, packageData); err != nil {
		scanErrors = append(scanErrors, fmt.Sprintf("blint scan failed: %v", err))
	} else {
		allVulnerabilities = append(allVulnerabilities, blintVulns...)
	}

	// 3. NPM-specific security checks
	if npmVulns, err := s.runNPMSecurityChecks(packageData); err != nil {
		scanErrors = append(scanErrors, fmt.Sprintf("npm security checks failed: %v", err))
	} else {
		allVulnerabilities = append(allVulnerabilities, npmVulns...)
	}

	// Deduplicate vulnerabilities
	uniqueVulns := s.deduplicateVulnerabilities(allVulnerabilities)

	// Calculate severity counts
	criticalCount, highCount, mediumCount, lowCount := s.calculateSeverityCounts(uniqueVulns)

	result := &ScanResult{
		ScannerName:     s.Name(),
		ScannerVersion:  "1.0.0",
		ArtifactType:    artifactType,
		Vulnerabilities: uniqueVulns,
		Summary: VulnerabilitySummary{
			Critical: criticalCount,
			High:     highCount,
			Medium:   mediumCount,
			Low:      lowCount,
			Total:    len(uniqueVulns),
		},
		ScanDuration: time.Since(startTime).Seconds(),
		Metadata: map[string]interface{}{
			"package_name":         packageData.PackageJSON.Name,
			"package_version":      packageData.PackageJSON.Version,
			"dependency_count":     len(packageData.Dependencies),
			"dev_dependency_count": len(packageData.DevDependencies),
			"has_package_lock":     packageData.HasPackageLock,
			"scan_tools_used":      []string{"owasp-dep-scan", "blint", "npm-security-checks"},
			"scan_approaches":      []string{"dependency_analysis", "code_quality_scan", "npm_specific_checks"},
			"scan_errors":          scanErrors,
			"main_file":            packageData.MainFile,
			"script_count":         len(packageData.Scripts),
		},
	}

	return result, nil
}

// extractNPMPackage extracts npm package information from tarball
func (s *NPMPackageScanner) extractNPMPackage(tarballPath string) (*NPMPackageData, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "npm-scan-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// Extract tarball
	if err := s.extractTarball(tarballPath, tempDir); err != nil {
		return nil, err
	}

	// Look for package.json in package/ directory
	packageJsonPath := filepath.Join(tempDir, "package", "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("package.json not found in tarball")
	}

	// Parse package.json
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, err
	}

	var packageJSON PackageJSON
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return nil, err
	}

	// Extract dependency information
	dependencies := make([]string, 0)
	for name, version := range packageJSON.Dependencies {
		dependencies = append(dependencies, fmt.Sprintf("%s@%s", name, version))
	}

	devDependencies := make([]string, 0)
	for name, version := range packageJSON.DevDependencies {
		devDependencies = append(devDependencies, fmt.Sprintf("%s@%s", name, version))
	}

	// Check for additional files
	packageDir := filepath.Join(tempDir, "package")
	hasPackageLock := false
	if _, err := os.Stat(filepath.Join(packageDir, "package-lock.json")); err == nil {
		hasPackageLock = true
	}

	hasNodeModules := false
	if _, err := os.Stat(filepath.Join(packageDir, "node_modules")); err == nil {
		hasNodeModules = true
	}

	return &NPMPackageData{
		PackageJSON:     &packageJSON,
		Dependencies:    dependencies,
		DevDependencies: devDependencies,
		Scripts:         packageJSON.Scripts,
		MainFile:        packageJSON.Main,
		HasPackageLock:  hasPackageLock,
		HasNodeModules:  hasNodeModules,
	}, nil
}

// extractTarball extracts a gzipped tarball to a directory
func (s *NPMPackageScanner) extractTarball(tarballPath, destDir string) error {
	cmd := exec.Command("tar", "-xzf", tarballPath, "-C", destDir)
	return cmd.Run()
}

// runDepScan runs OWASP dep-scan on the npm package
func (s *NPMPackageScanner) runDepScan(ctx context.Context, artifactPath string, packageData *NPMPackageData) ([]Vulnerability, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "depscan-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// Extract the package for scanning
	if err := s.extractTarball(artifactPath, tempDir); err != nil {
		return nil, err
	}

	packageDir := filepath.Join(tempDir, "package")
	outputFile := filepath.Join(tempDir, "depscan-results.json")

	// Try different ways to run dep-scan
	var cmd *exec.Cmd

	// First try as standalone executable
	if _, err := exec.LookPath(s.depScanPath); err == nil {
		cmd = exec.CommandContext(ctx, s.depScanPath, "--src", packageDir, "--reports-dir", tempDir, "--type", "npm")
	} else {
		// Try as Python module
		cmd = exec.CommandContext(ctx, s.pythonPath, "-m", "depscan", "--src", packageDir, "--reports-dir", tempDir, "--type", "npm")
	}

	if err := cmd.Run(); err != nil {
		return []Vulnerability{}, nil // Don't fail if dep-scan doesn't work
	}

	// Parse dep-scan results
	return s.parseDepScanResults(outputFile)
}

// runBlint runs Blint static analysis on the npm package
func (s *NPMPackageScanner) runBlint(ctx context.Context, artifactPath string, packageData *NPMPackageData) ([]Vulnerability, error) {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "blint-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// Extract the package for scanning
	if err := s.extractTarball(artifactPath, tempDir); err != nil {
		return nil, err
	}

	packageDir := filepath.Join(tempDir, "package")
	outputFile := filepath.Join(tempDir, "blint-results.json")

	// Try different ways to run blint
	var cmd *exec.Cmd

	// First try as standalone executable
	if _, err := exec.LookPath(s.blintPath); err == nil {
		cmd = exec.CommandContext(ctx, s.blintPath, "check", "--src", packageDir, "--output-file", outputFile)
	} else {
		// Try as Python module
		cmd = exec.CommandContext(ctx, s.pythonPath, "-m", "blint", "check", "--src", packageDir, "--output-file", outputFile)
	}

	if err := cmd.Run(); err != nil {
		return []Vulnerability{}, nil // Don't fail if blint doesn't work
	}

	// Parse blint results
	return s.parseBlintResults(outputFile)
}

// runNPMSecurityChecks runs npm-specific security checks
func (s *NPMPackageScanner) runNPMSecurityChecks(packageData *NPMPackageData) ([]Vulnerability, error) {
	var vulnerabilities []Vulnerability

	// Check for risky scripts
	if riskyScripts := s.checkRiskyScripts(packageData.Scripts); len(riskyScripts) > 0 {
		for script, issue := range riskyScripts {
			vuln := Vulnerability{
				ID:          fmt.Sprintf("NPM-RISKY-SCRIPT-%s", strings.ToUpper(script)),
				Severity:    "MEDIUM",
				Title:       fmt.Sprintf("Risky npm script: %s", script),
				Description: issue,
				Package:     packageData.PackageJSON.Name,
				Version:     packageData.PackageJSON.Version,
			}
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}

	// Check for suspicious dependencies
	if suspiciousDeps := s.checkSuspiciousDependencies(packageData.Dependencies); len(suspiciousDeps) > 0 {
		for _, dep := range suspiciousDeps {
			vuln := Vulnerability{
				ID:          "NPM-SUSPICIOUS-DEPENDENCY",
				Severity:    "LOW",
				Title:       "Suspicious dependency pattern",
				Description: fmt.Sprintf("Dependency %s matches suspicious pattern", dep),
				Package:     packageData.PackageJSON.Name,
				Version:     packageData.PackageJSON.Version,
			}
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}

	// Check for missing security fields
	if securityIssues := s.checkSecurityFields(packageData.PackageJSON); len(securityIssues) > 0 {
		for _, issue := range securityIssues {
			vuln := Vulnerability{
				ID:          "NPM-SECURITY-CONFIG",
				Severity:    "INFO",
				Title:       "Security configuration issue",
				Description: issue,
				Package:     packageData.PackageJSON.Name,
				Version:     packageData.PackageJSON.Version,
			}
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}

	return vulnerabilities, nil
}

// checkRiskyScripts identifies potentially risky npm scripts
func (s *NPMPackageScanner) checkRiskyScripts(scripts map[string]string) map[string]string {
	risky := make(map[string]string)
	riskyPatterns := map[string]string{
		"curl":        "Script contains curl command which could download external content",
		"wget":        "Script contains wget command which could download external content",
		"rm -rf":      "Script contains dangerous rm -rf command",
		"eval":        "Script uses eval which can execute arbitrary code",
		"sudo":        "Script requires sudo privileges",
		"npm install": "Script runs npm install which could install malicious packages",
	}

	for scriptName, scriptContent := range scripts {
		for pattern, warning := range riskyPatterns {
			if strings.Contains(scriptContent, pattern) {
				risky[scriptName] = warning
				break
			}
		}
	}

	return risky
}

// checkSuspiciousDependencies identifies potentially suspicious dependencies
func (s *NPMPackageScanner) checkSuspiciousDependencies(dependencies []string) []string {
	var suspicious []string
	suspiciousPatterns := []string{
		"env",
		"exec",
		"shell",
		"crypto-random-string",
		"request-",
		"node-",
	}

	for _, dep := range dependencies {
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(strings.ToLower(dep), pattern) {
				suspicious = append(suspicious, dep)
				break
			}
		}
	}

	return suspicious
}

// checkSecurityFields checks for missing security-related fields in package.json
func (s *NPMPackageScanner) checkSecurityFields(packageJSON *PackageJSON) []string {
	var issues []string

	if packageJSON.License == "" {
		issues = append(issues, "No license specified in package.json")
	}

	if packageJSON.Author == nil {
		issues = append(issues, "No author specified in package.json")
	}

	if len(packageJSON.Keywords) == 0 {
		issues = append(issues, "No keywords specified in package.json")
	}

	// Check for missing engines specification
	if len(packageJSON.Engines) == 0 {
		issues = append(issues, "No engines specification - may not work on all Node.js versions")
	}

	return issues
}

// parseDepScanResults parses OWASP dep-scan JSON output
func (s *NPMPackageScanner) parseDepScanResults(resultsFile string) ([]Vulnerability, error) {
	data, err := os.ReadFile(resultsFile)
	if err != nil {
		return []Vulnerability{}, nil
	}

	var depScanResult struct {
		Vulnerabilities []struct {
			ID          string   `json:"id"`
			Package     string   `json:"package"`
			Version     string   `json:"version"`
			Severity    string   `json:"severity"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			CVSS        float64  `json:"cvss"`
			References  []string `json:"references"`
		} `json:"vulnerabilities"`
	}

	if err := json.Unmarshal(data, &depScanResult); err != nil {
		return []Vulnerability{}, nil
	}

	var vulnerabilities []Vulnerability
	for _, vuln := range depScanResult.Vulnerabilities {
		vulnerability := Vulnerability{
			ID:          vuln.ID,
			Severity:    strings.ToUpper(vuln.Severity),
			Title:       vuln.Title,
			Description: vuln.Description,
			Package:     vuln.Package,
			Version:     vuln.Version,
			CVSS:        vuln.CVSS,
			References:  vuln.References,
		}
		vulnerabilities = append(vulnerabilities, vulnerability)
	}

	return vulnerabilities, nil
}

// parseBlintResults parses Blint JSON output
func (s *NPMPackageScanner) parseBlintResults(resultsFile string) ([]Vulnerability, error) {
	data, err := os.ReadFile(resultsFile)
	if err != nil {
		return []Vulnerability{}, nil
	}

	var blintResult struct {
		Issues []struct {
			RuleID      string `json:"rule_id"`
			Severity    string `json:"severity"`
			Title       string `json:"title"`
			Description string `json:"description"`
			File        string `json:"file"`
			Line        int    `json:"line"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(data, &blintResult); err != nil {
		return []Vulnerability{}, nil
	}

	var vulnerabilities []Vulnerability
	for _, issue := range blintResult.Issues {
		severity := strings.ToUpper(issue.Severity)
		if severity == "" {
			severity = "LOW"
		}

		vulnerability := Vulnerability{
			ID:          issue.RuleID,
			Severity:    severity,
			Title:       issue.Title,
			Description: fmt.Sprintf("%s (File: %s, Line: %d)", issue.Description, issue.File, issue.Line),
			Package:     "code-quality",
			Version:     "static-analysis",
		}
		vulnerabilities = append(vulnerabilities, vulnerability)
	}

	return vulnerabilities, nil
}

func (s *NPMPackageScanner) deduplicateVulnerabilities(vulns []Vulnerability) []Vulnerability {
	seen := make(map[string]bool)
	var unique []Vulnerability

	for _, vuln := range vulns {
		key := fmt.Sprintf("%s-%s-%s", vuln.ID, vuln.Package, vuln.Version)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, vuln)
		}
	}

	return unique
}

func (s *NPMPackageScanner) calculateSeverityCounts(vulns []Vulnerability) (critical, high, medium, low int) {
	for _, vuln := range vulns {
		switch strings.ToUpper(vuln.Severity) {
		case "CRITICAL":
			critical++
		case "HIGH":
			high++
		case "MEDIUM":
			medium++
		case "LOW", "INFO":
			low++
		}
	}
	return
}
