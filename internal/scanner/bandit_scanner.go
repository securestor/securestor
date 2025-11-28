package scanner

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BanditScanner implements Python code security scanning using Bandit
type BanditScanner struct {
	banditPath string
}

func NewBanditScanner() *BanditScanner {
	return &BanditScanner{
		banditPath: "bandit", // assumes bandit is in PATH
	}
}

func (s *BanditScanner) Name() string {
	return "Bandit"
}

func (s *BanditScanner) SupportedTypes() []string {
	return []string{"python", "pypi", "wheel", "sdist"}
}

func (s *BanditScanner) IsAvailable() bool {
	// First try to find bandit in PATH
	_, err := exec.LookPath(s.banditPath)
	if err == nil {
		return true
	}

	// If not found, try using python3 -m bandit
	cmd := exec.Command("python3", "-m", "bandit", "--help")
	err = cmd.Run()
	return err == nil
}

func (s *BanditScanner) Supports(artifactType string) bool {
	supportedTypes := s.SupportedTypes()
	for _, t := range supportedTypes {
		if t == artifactType {
			return true
		}
	}
	return false
}

func (s *BanditScanner) Scan(ctx context.Context, artifactPath string, artifactType string) (*ScanResult, error) {
	fmt.Printf("[BANDIT_DEBUG] Starting Bandit scan for %s (type: %s)\n", artifactPath, artifactType)
	startTime := time.Now()

	// Determine scan target - for Python packages, extract and scan contents
	scanTarget := artifactPath
	var tempDir string
	var cleanup func()

	// For Python packages (tar.gz, whl), extract first
	if strings.HasSuffix(artifactPath, ".tar.gz") || strings.HasSuffix(artifactPath, ".whl") || artifactType == "pypi" {
		fmt.Printf("[BANDIT_DEBUG] Extracting Python package: %s\n", artifactPath)
		var err error
		tempDir, err = s.extractPythonPackage(artifactPath)
		if err != nil {
			fmt.Printf("[BANDIT_DEBUG] Failed to extract package: %v\n", err)
			return nil, fmt.Errorf("failed to extract Python package: %w", err)
		}
		scanTarget = tempDir
		fmt.Printf("[BANDIT_DEBUG] Extracted to temp directory: %s\n", tempDir)

		// List ALL files in the extracted directory for debugging
		fmt.Printf("[BANDIT_DEBUG] Directory contents:\n")
		err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fmt.Printf("[BANDIT_DEBUG] File: %s\n", path)
				if strings.HasSuffix(path, ".py") {
					fmt.Printf("[BANDIT_DEBUG] *** Found Python file: %s\n", path)
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("[BANDIT_DEBUG] Error walking directory: %v\n", err)
		}

		cleanup = func() {
			os.RemoveAll(tempDir)
		}
		defer cleanup()
	}

	// Try to run bandit - first via PATH, then via python module
	var cmd *exec.Cmd
	if _, err := exec.LookPath(s.banditPath); err == nil {
		// Bandit available in PATH
		fmt.Printf("[BANDIT_DEBUG] Using bandit from PATH: %s\n", s.banditPath)
		cmd = exec.CommandContext(ctx, s.banditPath, "-r", scanTarget, "-f", "json", "--exit-zero")
	} else {
		// Use python3 -m bandit
		fmt.Printf("[BANDIT_DEBUG] Using python3 -m bandit\n")
		cmd = exec.CommandContext(ctx, "python3", "-m", "bandit", "-r", scanTarget, "-f", "json", "--exit-zero")
	}

	fmt.Printf("[BANDIT_DEBUG] Running command: %s\n", cmd.String())
	output, err := cmd.CombinedOutput()
	fmt.Printf("[BANDIT_DEBUG] Command output length: %d, error: %v\n", len(output), err)
	if len(output) > 0 {
		maxLen := 500
		if len(output) < maxLen {
			maxLen = len(output)
		}
		fmt.Printf("[BANDIT_DEBUG] First %d chars of output: %s\n", maxLen, string(output[:maxLen]))
	}

	if err != nil {
		return nil, fmt.Errorf("bandit command failed: %w, output: %s", err, string(output))
	}

	vulnerabilities, err := s.parseBanditOutput(output)
	fmt.Printf("[BANDIT_DEBUG] Parsed %d vulnerabilities from output\n", len(vulnerabilities))
	if err != nil {
		fmt.Printf("[BANDIT_DEBUG] Failed to parse output: %v\n", err)
		return nil, fmt.Errorf("failed to parse bandit output: %w", err)
	}

	summary := calculateSummary(vulnerabilities)
	duration := time.Since(startTime).Seconds()

	return &ScanResult{
		ScannerName:     s.Name(),
		ScannerVersion:  s.getVersion(),
		ArtifactType:    artifactType,
		Vulnerabilities: vulnerabilities,
		Summary:         summary,
		ScanDuration:    duration,
	}, nil
}

func (s *BanditScanner) parseBanditOutput(output []byte) ([]Vulnerability, error) {
	// Filter out log lines before JSON - Bandit outputs log messages that break JSON parsing
	outputStr := string(output)

	// Find the first occurrence of '{' which should be the start of JSON
	jsonStart := strings.Index(outputStr, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in bandit output")
	}

	// Extract only the JSON part
	jsonOutput := outputStr[jsonStart:]
	fmt.Printf("[BANDIT_DEBUG] Filtered JSON output (first 200 chars): %s\n", jsonOutput[:min(200, len(jsonOutput))])

	var banditResult struct {
		Results []struct {
			TestID          string `json:"test_id"`
			TestName        string `json:"test_name"`
			IssueText       string `json:"issue_text"`
			IssueSeverity   string `json:"issue_severity"`
			IssueConfidence string `json:"issue_confidence"`
			Filename        string `json:"filename"`
			LineNumber      int    `json:"line_number"`
			Code            string `json:"code"`
			MoreInfo        string `json:"more_info"`
			IssueCWE        struct {
				ID   int    `json:"id"`
				Link string `json:"link"`
			} `json:"issue_cwe"`
		} `json:"results"`
		Metrics struct {
			Total struct {
				ConfidenceHigh   int `json:"CONFIDENCE.HIGH"`
				ConfidenceMedium int `json:"CONFIDENCE.MEDIUM"`
				ConfidenceLow    int `json:"CONFIDENCE.LOW"`
				SeverityHigh     int `json:"SEVERITY.HIGH"`
				SeverityMedium   int `json:"SEVERITY.MEDIUM"`
				SeverityLow      int `json:"SEVERITY.LOW"`
			} `json:"_totals"`
		} `json:"metrics"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &banditResult); err != nil {
		return nil, fmt.Errorf("failed to parse bandit JSON output: %w", err)
	}

	var vulnerabilities []Vulnerability
	for _, result := range banditResult.Results {
		severity := strings.ToUpper(result.IssueSeverity)

		// Create vulnerability ID combining test ID with CWE if available
		vulnID := result.TestID
		if result.IssueCWE.ID != 0 {
			vulnID = fmt.Sprintf("%s-CWE-%d", result.TestID, result.IssueCWE.ID)
		}

		vuln := Vulnerability{
			ID:          vulnID,
			Severity:    severity,
			Title:       result.TestName,
			Description: result.IssueText,
			Package:     extractPackageFromPath(result.Filename),
			Version:     "unknown",
			References: []string{
				result.MoreInfo,
				result.IssueCWE.Link,
			},
			CVSS: s.severityToScore(severity),
		}

		vulnerabilities = append(vulnerabilities, vuln)
	}

	return vulnerabilities, nil
}

// extractPackageFromPath attempts to extract a package name from file path
func extractPackageFromPath(filePath string) string {
	// Extract just the directory or file name without full path
	parts := strings.Split(filePath, "/")
	if len(parts) > 0 {
		fileName := parts[len(parts)-1]
		// Remove file extension if present
		if strings.Contains(fileName, ".") {
			return strings.Split(fileName, ".")[0]
		}
		return fileName
	}
	return "unknown"
}

// severityToScore converts Bandit severity to CVSS-like score
func (s *BanditScanner) severityToScore(severity string) float64 {
	switch strings.ToUpper(severity) {
	case "HIGH":
		return 7.5
	case "MEDIUM":
		return 5.0
	case "LOW":
		return 2.5
	default:
		return 0.0
	}
}

func (s *BanditScanner) getVersion() string {
	// Try bandit --version first
	if _, err := exec.LookPath(s.banditPath); err == nil {
		cmd := exec.Command(s.banditPath, "--version")
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}

	// Fallback to python3 -m bandit --version
	cmd := exec.Command("python3", "-m", "bandit", "--version")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}

	return "unknown"
}

// extractPythonPackage extracts a Python package (tar.gz or whl) to a temporary directory for scanning
func (s *BanditScanner) extractPythonPackage(packagePath string) (string, error) {
	fmt.Printf("[BANDIT_DEBUG] extractPythonPackage called with: %s\n", packagePath)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "bandit-scan-")
	if err != nil {
		fmt.Printf("[BANDIT_DEBUG] Failed to create temp dir: %v\n", err)
		return "", err
	}
	fmt.Printf("[BANDIT_DEBUG] Created temp directory: %s\n", tempDir)

	// Check file type by reading magic bytes instead of just relying on extension
	file, err := os.Open(packagePath)
	if err != nil {
		fmt.Printf("[BANDIT_DEBUG] Failed to open file for magic byte check: %v\n", err)
		return tempDir, nil
	}
	defer file.Close()

	// Read first few bytes to detect file type
	magic := make([]byte, 10)
	n, err := file.Read(magic)
	if err != nil && n == 0 {
		fmt.Printf("[BANDIT_DEBUG] Failed to read magic bytes: %v\n", err)
		return tempDir, nil
	}
	fmt.Printf("[BANDIT_DEBUG] Read %d magic bytes: %x\n", n, magic[:n])

	// Check for gzip magic bytes (1f 8b)
	isGzip := n >= 2 && magic[0] == 0x1f && magic[1] == 0x8b
	// Check for ZIP magic bytes (50 4b) for .whl files
	isZip := n >= 2 && magic[0] == 0x50 && magic[1] == 0x4b

	fmt.Printf("[BANDIT_DEBUG] File type detection - isGzip: %v, isZip: %v\n", isGzip, isZip)

	// Handle .whl files (which are ZIP archives) or files with ZIP magic
	if strings.HasSuffix(packagePath, ".whl") || isZip {
		fmt.Printf("[BANDIT_DEBUG] Detected wheel/ZIP file, using ZIP extraction\n")
		return s.extractZipPackage(packagePath, tempDir)
	}

	// Handle .tar.gz files or files with gzip magic
	if strings.HasSuffix(packagePath, ".tar.gz") || isGzip {
		fmt.Printf("[BANDIT_DEBUG] Detected tar.gz file, using tar extraction\n")
		return s.extractTarGzPackage(packagePath, tempDir)
	}

	fmt.Printf("[BANDIT_DEBUG] No matching extraction method for file: %s (not gzip or zip)\n", packagePath)
	return tempDir, nil
}

// extractTarGzPackage extracts a .tar.gz file to the specified directory
func (s *BanditScanner) extractTarGzPackage(packagePath, destDir string) (string, error) {
	fmt.Printf("[BANDIT_DEBUG] Opening package file: %s\n", packagePath)
	file, err := os.Open(packagePath)
	if err != nil {
		fmt.Printf("[BANDIT_DEBUG] Failed to open package: %v\n", err)
		return "", err
	}
	defer file.Close()

	fmt.Printf("[BANDIT_DEBUG] Creating gzip reader\n")
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		fmt.Printf("[BANDIT_DEBUG] Failed to create gzip reader: %v\n", err)
		return "", err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	extractedFiles := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("[BANDIT_DEBUG] Error reading tar header: %v\n", err)
			return "", err
		}

		targetPath := filepath.Join(destDir, header.Name)
		fmt.Printf("[BANDIT_DEBUG] Processing tar entry: %s -> %s\n", header.Name, targetPath)

		// Ensure we don't extract outside the destination directory (path traversal protection)
		if !strings.HasPrefix(targetPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			fmt.Printf("[BANDIT_DEBUG] Creating directory: %s\n", targetPath)
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				fmt.Printf("[BANDIT_DEBUG] Failed to create directory: %v\n", err)
				return "", err
			}
		case tar.TypeReg:
			fmt.Printf("[BANDIT_DEBUG] Extracting file: %s (size: %d)\n", targetPath, header.Size)
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				fmt.Printf("[BANDIT_DEBUG] Failed to create parent directory: %v\n", err)
				return "", err
			}

			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				fmt.Printf("[BANDIT_DEBUG] Failed to create file: %v\n", err)
				return "", err
			}

			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				fmt.Printf("[BANDIT_DEBUG] Failed to copy file content: %v\n", err)
				return "", err
			}
			extractedFiles++
		}
	}

	fmt.Printf("[BANDIT_DEBUG] Extraction complete. Files extracted: %d\n", extractedFiles)
	return destDir, nil
}

// extractZipPackage extracts a .whl (ZIP) file to the specified directory
func (s *BanditScanner) extractZipPackage(packagePath, destDir string) (string, error) {
	// For now, use unzip command if available
	cmd := exec.Command("unzip", "-q", packagePath, "-d", destDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract wheel file: %w", err)
	}
	return destDir, nil
}
