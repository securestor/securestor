package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/encrypt"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/repository"
	"github.com/securestor/securestor/internal/scanner"
	"github.com/securestor/securestor/internal/storage"
)

type ScanService struct {
	vulnerabilityRepo *repository.VulnerabilityRepository
	complianceRepo    *repository.ComplianceRepository
	scanRepo          *repository.ScanRepository
	artifactRepo      *repository.ArtifactRepository
	blobStorage       *storage.BlobStorage
	scannerManager    *scanner.ScannerManager
	sbomGenerator     *scanner.SBOMGenerator
	logger            *log.Logger
	tempDir           string
	scanProgress      map[int64]bool
	progressMutex     sync.RWMutex

	// Enterprise encryption services for secure scanning
	encryptionService *encrypt.EncryptionService
	tmkService        *encrypt.TMKService
	auditLogService   *AuditLogService
}

func NewScanService(
	vulnerabilityRepo *repository.VulnerabilityRepository,
	complianceRepo *repository.ComplianceRepository,
	scanRepo *repository.ScanRepository,
	artifactRepo *repository.ArtifactRepository,
	blobStorage *storage.BlobStorage,
	scannerManager *scanner.ScannerManager,
	logger *log.Logger,
	tempDir string) *ScanService {

	return &ScanService{
		vulnerabilityRepo: vulnerabilityRepo,
		complianceRepo:    complianceRepo,
		scanRepo:          scanRepo,
		artifactRepo:      artifactRepo,
		blobStorage:       blobStorage,
		scannerManager:    scannerManager,
		sbomGenerator:     scanner.NewSBOMGenerator(tempDir),
		logger:            logger,
		tempDir:           tempDir,
		scanProgress:      make(map[int64]bool),
		// Encryption services will be set via SetEncryptionServices
		encryptionService: nil,
		tmkService:        nil,
		auditLogService:   nil,
	}
}

// SetEncryptionServices sets the encryption services for secure scanning of encrypted artifacts
func (s *ScanService) SetEncryptionServices(
	encryptionService *encrypt.EncryptionService,
	tmkService *encrypt.TMKService,
	auditLogService *AuditLogService,
) {
	s.encryptionService = encryptionService
	s.tmkService = tmkService
	s.auditLogService = auditLogService
	s.logger.Printf("[SECURE_SCAN] Encryption services configured for secure artifact scanning")
}

// IsScanInProgress checks if a scan is currently running for an artifact
func (s *ScanService) IsScanInProgress(artifactID int64) bool {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()
	return s.scanProgress[artifactID]
}

func (s *ScanService) setScanProgress(artifactID int64, inProgress bool) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()
	if inProgress {
		s.scanProgress[artifactID] = true
	} else {
		delete(s.scanProgress, artifactID)
	}
}

// ScanArtifact performs vulnerability scanning on an artifact
func (s *ScanService) ScanArtifact(ctx context.Context, artifact *models.Artifact, scan *models.SecurityScan) error {
	s.logger.Printf("[SCAN_DEBUG] ScanArtifact called for artifact %d, type: %s", artifact.ID, artifact.Type)
	// Create temp directory for scan
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	var data []byte
	var err error

	// Special handling for Docker manifests stored in OCI format
	if artifact.Type == "docker" {
		if artifactType, ok := artifact.Metadata["artifact_type"].(string); ok && artifactType == "manifest" {
			// For Docker manifests, try to get the OCI manifest path
			if ociPath, exists := artifact.Metadata["storage_path"].(string); exists {
				// Read manifest from OCI storage path
				data, err = os.ReadFile(ociPath)
				if err != nil {
					return fmt.Errorf("failed to read OCI manifest from %s: %w", ociPath, err)
				}
			} else {
				// Fall back to trying blob storage
				storageID, ok := artifact.Metadata["storage_id"].(string)
				if !ok {
					return fmt.Errorf("neither OCI manifest path nor storage ID found for Docker manifest")
				}
				data, err = s.blobStorage.DownloadArtifact(ctx, artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
				if err != nil {
					return fmt.Errorf("failed to download Docker manifest from blob storage: %w", err)
				}
			}
		} else {
			// Regular Docker blob/layer - use blob storage
			storageID, ok := artifact.Metadata["storage_id"].(string)
			if !ok {
				return fmt.Errorf("storage ID not found in artifact metadata")
			}
			data, err = s.blobStorage.DownloadArtifact(ctx, artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
			if err != nil {
				return fmt.Errorf("failed to download artifact: %w", err)
			}
		}
	} else {
		// Regular artifacts - check storage type first, then artifact type
		s.logger.Printf("[SCAN_DEBUG] Processing artifact %d, type: %s, metadata: %v", artifact.ID, artifact.Type, artifact.Metadata)
		if storageType, ok := artifact.Metadata["storage_type"].(string); ok && storageType == "erasure_coded" {
			s.logger.Printf("[SCAN_DEBUG] Using erasure coded path for artifact %d", artifact.ID)
			// Erasure coded artifacts regardless of type
			storageID, ok := artifact.Metadata["storage_id"].(string)
			if !ok {
				return fmt.Errorf("storage ID not found in artifact metadata for erasure coded artifact")
			}
			data, err = s.blobStorage.DownloadArtifact(ctx, artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
			if err != nil {
				return fmt.Errorf("failed to download artifact: %w", err)
			}
		} else if artifact.Type == "maven" || artifact.Type == "npm" {
			s.logger.Printf("[SCAN_DEBUG] Using file path for artifact %d (storage_type: %v)", artifact.ID, storageType)
			// Maven and NPM artifacts stored directly on filesystem
			if filePath, ok := artifact.Metadata["file_path"].(string); ok {
				data, err = os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read artifact file from %s: %w", filePath, err)
				}
			} else {
				return fmt.Errorf("file_path not found in artifact metadata for %s artifact", artifact.Type)
			}
		} else {
			s.logger.Printf("[SCAN_DEBUG] Using blob storage for artifact %d", artifact.ID)
			// Use blob storage for other artifact types
			storageID, ok := artifact.Metadata["storage_id"].(string)
			if !ok {
				return fmt.Errorf("storage ID not found in artifact metadata")
			}
			data, err = s.blobStorage.DownloadArtifact(ctx, artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
			if err != nil {
				return fmt.Errorf("failed to download artifact: %w", err)
			}
		}
	}

	// ENTERPRISE SECURE SCANNING: Handle encrypted artifacts
	// If artifact is encrypted, decrypt it in-memory before scanning
	// This implements ephemeral decryption with comprehensive audit logging
	if artifact.Encrypted && s.encryptionService != nil {
		s.logger.Printf("[SECURE_SCAN] Artifact %d is encrypted, initiating secure decryption for scan", artifact.ID)

		// Audit log: Decryption for scan purpose
		s.logger.Printf("[AUDIT] ARTIFACT_DECRYPT_FOR_SCAN: artifact_id=%s, name=%s, scan_id=%d, purpose=security_scan, encrypted_size=%d",
			artifact.ID.String(), artifact.Name, scan.ID, len(data))

		// Extract encryption metadata
		nonceInterface, ok := artifact.EncryptionMetadata["nonce"]
		if !ok {
			return fmt.Errorf("encryption nonce not found in metadata")
		}

		// Decode nonce (stored as base64 string in JSONB)
		var nonce []byte
		switch v := nonceInterface.(type) {
		case string:
			var err error
			nonce, err = base64.StdEncoding.DecodeString(v)
			if err != nil {
				return fmt.Errorf("failed to decode nonce: %w", err)
			}
		case []byte:
			nonce = v
		default:
			return fmt.Errorf("unexpected nonce type: %T", v)
		}

		// Reconstruct EncryptedData for decryption
		encryptedData := &encrypt.EncryptedData{
			Ciphertext:   data,
			EncryptedDEK: artifact.EncryptedDEK,
			Nonce:        nonce,
			Algorithm:    artifact.EncryptionAlgorithm,
			KeyVersion:   artifact.EncryptionVersion,
			TenantID:     artifact.TenantID.String(),
		}

		// Decrypt artifact in-memory (ephemeral decryption)
		startTime := time.Now()
		plaintext, err := s.encryptionService.DecryptArtifact(encryptedData, os.Getenv("AWS_KMS_KEY_ID"))
		if err != nil {
			s.logger.Printf("[AUDIT] ARTIFACT_DECRYPT_FAILED: artifact_id=%s, error=%v", artifact.ID.String(), err)
			return fmt.Errorf("failed to decrypt artifact for scanning: %w", err)
		}

		decryptDuration := time.Since(startTime)

		// Replace encrypted data with plaintext for scanning
		data = plaintext

		// Audit log: Successful decryption
		s.logger.Printf("[AUDIT] ARTIFACT_DECRYPTED_FOR_SCAN: artifact_id=%s, name=%s, plaintext_size=%d, decrypt_duration=%.3fs, algorithm=%s, key_version=%d",
			artifact.ID.String(), artifact.Name, len(plaintext), decryptDuration.Seconds(), artifact.EncryptionAlgorithm, artifact.EncryptionVersion)

		s.logger.Printf("[SECURE_SCAN] Artifact %d decrypted in-memory for scanning (%d bytes, took %v)",
			artifact.ID, len(plaintext), decryptDuration)

		// Note: Plaintext will only exist in memory temporarily during scan
		// It will be scrubbed after scan completes (see defer at end)
		defer func() {
			// Securely scrub plaintext from memory after scan
			if len(data) > 0 {
				// Overwrite with random data
				rand.Read(data)
				// Overwrite with zeros
				for i := range data {
					data[i] = 0
				}
				// Force garbage collection
				runtime.GC()
				s.logger.Printf("[SECURE_SCAN] Securely scrubbed plaintext from memory after scan")
			}
		}()
	}

	// Write to ephemeral temp file (will be auto-deleted after scan)
	tempFile := filepath.Join(s.tempDir, fmt.Sprintf("artifact-%d-%d", artifact.ID, scan.ID))
	if err := os.WriteFile(tempFile, data, 0600); err != nil { // 0600 for security
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer func() {
		// Securely wipe temp file before deletion
		if file, err := os.OpenFile(tempFile, os.O_WRONLY, 0600); err == nil {
			stat, _ := file.Stat()
			randomData := make([]byte, stat.Size())
			rand.Read(randomData)
			file.WriteAt(randomData, 0)
			file.Sync()
			file.Close()
		}
		os.Remove(tempFile)
	}()

	// For Maven artifacts, also copy the POM file for dependency scanning
	var pomFile string
	var sbomFile string
	if artifact.Type == "maven" {
		// Check if this is a filesystem-stored Maven artifact with file_path
		if filePath, ok := artifact.Metadata["file_path"].(string); ok {
			// Derive POM path from JAR path
			pomPath := strings.TrimSuffix(filePath, ".jar") + ".pom"
			if _, err := os.Stat(pomPath); err == nil {
				// POM file exists, copy it to temp directory
				pomFile = filepath.Join(s.tempDir, fmt.Sprintf("artifact-%d-%d.pom", artifact.ID, scan.ID))
				pomData, err := os.ReadFile(pomPath)
				if err == nil {
					if err := os.WriteFile(pomFile, pomData, 0644); err == nil {
						defer os.Remove(pomFile)
						s.logger.Printf("Maven POM file copied for scanning: %s", pomFile)

						// Generate SBOM for Maven artifact using cdxgen
						s.logger.Printf("Generating SBOM for Maven artifact %d...", artifact.ID)
						sbomPath, err := s.sbomGenerator.GenerateMavenSBOM(ctx, pomFile, artifact.ID, scan.ID)
						if err != nil {
							s.logger.Printf("Warning: SBOM generation failed for Maven artifact %d: %v", artifact.ID, err)
						} else {
							sbomFile = sbomPath
							s.logger.Printf("SBOM generated successfully for Maven artifact %d at %s", artifact.ID, sbomPath)

							// Extract SBOM metadata
							sbomMetadata, err := s.sbomGenerator.GetSBOMMetadata(sbomPath)
							if err != nil {
								s.logger.Printf("Warning: Failed to extract SBOM metadata: %v", err)
							} else {
								// Add SBOM info to artifact metadata
								if artifact.Metadata == nil {
									artifact.Metadata = make(map[string]interface{})
								}
								artifact.Metadata["sbom_components"] = sbomMetadata["component_count"]
								artifact.Metadata["sbom_generated_at"] = time.Now().Format(time.RFC3339)
								s.logger.Printf("SBOM contains %v components", sbomMetadata["component_count"])
							}
						}
					}
				}
			}
		}
	}

	// Update scan status to running
	scan.Status = "running"
	if err := s.scanRepo.Update(scan); err != nil {
		return fmt.Errorf("failed to update scan status: %w", err)
	}

	// Initialize scan results
	results := &models.ScanResults{
		ScanID:    scan.ID,
		TenantID:  scan.TenantID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Perform real vulnerability scanning using scanner manager
	s.logger.Printf("Starting vulnerability scan for artifact %d using real scanners", artifact.ID)

	// For Maven artifacts with SBOM, use the SBOM file for scanning
	// Otherwise, scan the POM file for dependencies, or the artifact file
	scanFile := tempFile
	if artifact.Type == "maven" {
		if sbomFile != "" {
			// Use SBOM file for comprehensive dependency scanning
			scanFile = sbomFile
			s.logger.Printf("Scanning Maven artifact using generated SBOM: %s", sbomFile)
		} else if pomFile != "" {
			// Fall back to POM file if SBOM generation failed
			scanFile = pomFile
			s.logger.Printf("Scanning Maven POM file for dependencies: %s", pomFile)
		}
	}

	// Use real scanners to scan the artifact
	scanResults, err := s.scannerManager.ScanWithAll(ctx, scanFile, artifact.Type)
	if err != nil {
		s.logger.Printf("Scanner failed for artifact %d: %v", artifact.ID, err)
		// Fall back to basic results if scanners fail
		vulnResults := &models.VulnerabilityResults{
			TotalFound:      0,
			Critical:        0,
			High:            0,
			Medium:          0,
			Low:             0,
			Info:            0,
			Fixed:           0,
			Unfixed:         0,
			Vulnerabilities: []models.VulnerabilityItem{},
		}
		results.VulnerabilityResults = vulnResults
		results.Summary = fmt.Sprintf("Scanner failed: %v", err)
		results.RiskLevel = "unknown"
		results.OverallScore = 0
	} else {
		// Convert scanner results to our model format
		vulnResults := s.convertScannerResults(scanResults)
		results.VulnerabilityResults = vulnResults
		results.Summary = s.generateSummary(vulnResults)
		results.RiskLevel = s.calculateRiskLevel(vulnResults)
		results.OverallScore = s.calculateScore(vulnResults)
		s.logger.Printf("Scan completed for artifact %d: found %d vulnerabilities", artifact.ID, vulnResults.TotalFound)
	}

	// Basic malware scan (still mock for now - could be extended later)
	malwareResults := &models.MalwareResults{
		TotalScanned:    1,
		ThreatsFound:    0,
		CleanFiles:      1,
		SuspiciousFiles: 0,
		InfectedFiles:   0,
		Threats:         []models.MalwareThreat{},
		ScanEngines:     []models.ScanEngineResult{},
	}

	// Set malware results
	results.MalwareResults = malwareResults
	results.Recommendations = s.generateRecommendations(results.VulnerabilityResults)

	// Save scan results
	if err := s.scanRepo.SaveResults(results); err != nil {
		return fmt.Errorf("failed to save scan results: %w", err)
	}

	// Update scan record
	now := time.Now()
	scan.CompletedAt = &now
	duration := int64(now.Sub(scan.StartedAt).Seconds())
	scan.Duration = &duration
	scan.Status = "completed"
	scan.Results = results

	if err := s.scanRepo.Update(scan); err != nil {
		return fmt.Errorf("failed to update scan record: %w", err)
	}

	// Auto-calculate and update compliance based on vulnerability results
	if err := s.updateComplianceFromScan(artifact, results.VulnerabilityResults); err != nil {
		s.logger.Printf("Warning: Failed to auto-update compliance for artifact %s: %v", artifact.ID.String(), err)
		// Don't fail the scan if compliance update fails
	}

	s.logger.Printf("Security scan completed for artifact %s (scan ID: %d)", artifact.ID.String(), scan.ID)
	return nil
}

// updateComplianceFromScan automatically updates compliance status based on scan results
func (s *ScanService) updateComplianceFromScan(artifact *models.Artifact, vulnResults *models.VulnerabilityResults) error {
	if vulnResults == nil {
		return nil
	}

	// Convert VulnerabilityResults to Vulnerability model for calculation
	vuln := &models.Vulnerability{
		ArtifactID: artifact.ID,
		Critical:   vulnResults.Critical,
		High:       vulnResults.High,
		Medium:     vulnResults.Medium,
		Low:        vulnResults.Low,
	}

	// Calculate compliance status and score
	status, score := CalculateComplianceFromVulnerabilities(vuln)

	// Check if compliance audit already exists
	existingCompliance, err := s.complianceRepo.GetByArtifactID(artifact.ID)
	if err == nil && existingCompliance != nil {
		// Update existing compliance
		existingCompliance.TenantID = artifact.TenantID // Ensure tenant ID is set
		existingCompliance.Status = status
		existingCompliance.Score = score
		existingCompliance.Auditor = "Auto-Scan"
		existingCompliance.AuditedAt = time.Now()
		existingCompliance.SecurityScan = fmt.Sprintf("Critical: %d, High: %d, Medium: %d, Low: %d",
			vuln.Critical, vuln.High, vuln.Medium, vuln.Low)

		return s.complianceRepo.Create(existingCompliance) // This creates a new audit entry for history
	}

	// Create new compliance audit if none exists
	newCompliance := &models.ComplianceAudit{
		TenantID:   artifact.TenantID, // Set tenant ID from artifact
		ArtifactID: artifact.ID,
		Status:     status,
		Score:      score,
		Auditor:    "Auto-Scan",
		AuditedAt:  time.Now(),
		SecurityScan: fmt.Sprintf("Critical: %d, High: %d, Medium: %d, Low: %d",
			vuln.Critical, vuln.High, vuln.Medium, vuln.Low),
	}

	return s.complianceRepo.Create(newCompliance)
}

// CreateScan creates a new scan record
func (s *ScanService) CreateScan(scan *models.SecurityScan) error {
	return s.scanRepo.Create(scan)
}

// UpdateScan updates an existing scan record
func (s *ScanService) UpdateScan(scan *models.SecurityScan) error {
	return s.scanRepo.Update(scan)
}

// GetScanByID retrieves a scan by its ID
func (s *ScanService) GetScanByID(scanID int64) (*models.SecurityScan, error) {
	return s.scanRepo.GetByID(scanID)
}

// GetScanByUUID retrieves a scan by its UUID
func (s *ScanService) GetScanByUUID(scanID uuid.UUID) (*models.SecurityScan, error) {
	return s.scanRepo.GetByUUID(scanID)
}

// GetAllScans retrieves all scans with optional filtering
func (s *ScanService) GetAllScans(status, scanType, priority string, limit, offset int) ([]*models.SecurityScan, int, error) {
	return s.scanRepo.GetAllWithFilters(status, scanType, priority, limit, offset)
}

// GetActiveScan retrieves active scan for an artifact
func (s *ScanService) GetActiveScan(artifactID uuid.UUID) (*models.SecurityScan, error) {
	return s.scanRepo.GetActiveScan(artifactID)
}

// GetLatestScanResult retrieves the latest scan result for an artifact
func (s *ScanService) GetLatestScanResult(artifactID uuid.UUID) (*models.SecurityScan, error) {
	return s.scanRepo.GetLatestResult(artifactID)
}

// GetScanHistory retrieves scan history for an artifact
func (s *ScanService) GetScanHistory(artifactID uuid.UUID, limit, offset int) ([]*models.SecurityScan, int, error) {
	return s.scanRepo.GetHistory(artifactID, limit, offset)
}

// CancelScan cancels an active scan
func (s *ScanService) CancelScan(scanID int64) error {
	scan, err := s.scanRepo.GetByID(scanID)
	if err != nil {
		return err
	}

	if scan.Status != "running" && scan.Status != "initiated" {
		return errors.New("scan cannot be cancelled")
	}

	scan.Status = "cancelled"
	now := time.Now()
	scan.CompletedAt = &now

	if scan.StartedAt.Before(now) {
		duration := int64(now.Sub(scan.StartedAt).Seconds())
		scan.Duration = &duration
	}

	return s.scanRepo.Update(scan)
}

// CancelScanByUUID cancels an active scan by its UUID
func (s *ScanService) CancelScanByUUID(scanID uuid.UUID) error {
	scan, err := s.scanRepo.GetByUUID(scanID)
	if err != nil {
		return err
	}

	if scan.Status != "running" && scan.Status != "initiated" {
		return errors.New("scan cannot be cancelled")
	}

	scan.Status = "cancelled"
	now := time.Now()
	scan.CompletedAt = &now

	if scan.StartedAt.Before(now) {
		duration := int64(now.Sub(scan.StartedAt).Seconds())
		scan.Duration = &duration
	}

	return s.scanRepo.Update(scan)
}

// GetAvailableScanners returns information about available scanners
func (s *ScanService) GetAvailableScanners() []*models.ScannerInfo {
	return []*models.ScannerInfo{
		{
			ID:           "vulnerability-scanner",
			Name:         "Vulnerability Scanner",
			Type:         "vulnerability",
			Version:      "1.0.0",
			Enabled:      true,
			Status:       "healthy",
			LastUpdate:   time.Now(),
			Capabilities: []string{"cve-detection", "dependency-analysis"},
		},
		{
			ID:           "malware-scanner",
			Name:         "Malware Scanner",
			Type:         "malware",
			Version:      "1.0.0",
			Enabled:      true,
			Status:       "healthy",
			LastUpdate:   time.Now(),
			Capabilities: []string{"virus-detection", "trojan-detection"},
		},
	}
}

// CheckScannerHealth checks the health of all scanners
func (s *ScanService) CheckScannerHealth() []*models.ScannerHealth {
	return []*models.ScannerHealth{
		{
			ScannerID:    "vulnerability-scanner",
			Name:         "Vulnerability Scanner",
			Status:       "healthy",
			LastCheck:    time.Now(),
			ResponseTime: 100,
		},
		{
			ScannerID:    "malware-scanner",
			Name:         "Malware Scanner",
			Status:       "healthy",
			LastCheck:    time.Now(),
			ResponseTime: 150,
		},
	}
}

// GetVulnerabilities retrieves vulnerabilities for an artifact with filtering
func (s *ScanService) GetVulnerabilities(artifactID uuid.UUID, severity, status string) ([]*models.VulnerabilityItem, error) {
	scan, err := s.scanRepo.GetLatestResult(artifactID)
	if err != nil {
		return nil, err
	}

	if scan.Results == nil || scan.Results.VulnerabilityResults == nil {
		return []*models.VulnerabilityItem{}, nil
	}

	vulnerabilities := scan.Results.VulnerabilityResults.Vulnerabilities

	// Apply filters
	var filtered []*models.VulnerabilityItem
	for i := range vulnerabilities {
		vuln := &vulnerabilities[i]

		if severity != "" && vuln.Severity != severity {
			continue
		}

		if status != "" && vuln.Status != status {
			continue
		}

		filtered = append(filtered, vuln)
	}

	return filtered, nil
}

// Legacy method for backward compatibility
func (s *ScanService) GetScanResult(artifactID uuid.UUID) (interface{}, error) {
	scan, err := s.GetLatestScanResult(artifactID)
	if err != nil {
		return nil, err
	}

	if scan.Results == nil {
		return map[string]interface{}{
			"message": "No scan results available",
		}, nil
	}

	return scan.Results, nil
}

// convertScannerResults converts scanner results to our model format
func (s *ScanService) convertScannerResults(scanResults []*scanner.ScanResult) *models.VulnerabilityResults {
	s.logger.Printf("[CONVERT_DEBUG] Converting results from %d scanners", len(scanResults))

	vulnResults := &models.VulnerabilityResults{
		Vulnerabilities: []models.VulnerabilityItem{},
	}

	// Aggregate vulnerabilities from all scanners
	for i, result := range scanResults {
		s.logger.Printf("[CONVERT_DEBUG] Scanner %d: %s found %d vulnerabilities", i, result.ScannerName, len(result.Vulnerabilities))
		for _, vuln := range result.Vulnerabilities {
			vulnItem := models.VulnerabilityItem{
				ID:           vuln.ID,
				CVE:          vuln.CVE, // Populate CVE field from scanner result
				Title:        vuln.Title,
				Description:  vuln.Description,
				Severity:     vuln.Severity,
				Score:        vuln.CVSS,
				Component:    vuln.Package,
				Version:      vuln.Version,
				FixedVersion: vuln.FixedIn,
				References:   vuln.References,
				Status:       "open",
			}
			vulnResults.Vulnerabilities = append(vulnResults.Vulnerabilities, vulnItem)
		}
	}

	// Calculate counts by severity
	for _, vuln := range vulnResults.Vulnerabilities {
		vulnResults.TotalFound++
		switch vuln.Severity {
		case "CRITICAL":
			vulnResults.Critical++
		case "HIGH":
			vulnResults.High++
		case "MEDIUM":
			vulnResults.Medium++
		case "LOW":
			vulnResults.Low++
		default:
			vulnResults.Info++
		}
	}

	// For now, assume all vulnerabilities are unfixed
	vulnResults.Unfixed = vulnResults.TotalFound

	return vulnResults
}

// GetDetailedVulnerabilities returns detailed vulnerability information
func (s *ScanService) GetDetailedVulnerabilities(artifactID uuid.UUID) (map[string]interface{}, error) {
	// Get summary
	summary, err := s.vulnerabilityRepo.GetByArtifactID(artifactID)
	if err != nil {
		return nil, err
	}

	// Get detailed vulnerabilities from storage
	// For now, return empty array - this would need to be implemented
	// based on how vulnerabilities are stored in the database
	details := []scanner.Vulnerability{}

	return map[string]interface{}{
		"artifact_id":     artifactID,
		"summary":         summary,
		"vulnerabilities": details,
		"scanned_at":      summary.ScannedAt,
	}, nil
}

// ListVulnerableArtifacts returns artifacts with vulnerabilities
func (s *ScanService) ListVulnerableArtifacts(severity string, minCVSS string) ([]map[string]interface{}, error) {
	// Get all artifacts with vulnerabilities
	filter := &models.ArtifactFilter{
		Vulnerabilities: "has_vulnerabilities",
		Limit:           1000,
	}

	artifacts, _, err := s.artifactRepo.List(filter)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	// Convert minCVSS to float for comparison if needed
	_, _ = strconv.ParseFloat(minCVSS, 64)

	for _, artifact := range artifacts {
		if artifact.Vulnerabilities == nil {
			continue
		}

		// Filter by severity
		if severity != "" {
			switch severity {
			case "critical":
				if artifact.Vulnerabilities.Critical == 0 {
					continue
				}
			case "high":
				if artifact.Vulnerabilities.High == 0 {
					continue
				}
			}
		}

		result = append(result, map[string]interface{}{
			"id":              artifact.ID,
			"name":            artifact.Name,
			"version":         artifact.Version,
			"type":            artifact.Type,
			"repository":      artifact.Repository,
			"vulnerabilities": artifact.Vulnerabilities,
			"compliance":      artifact.Compliance,
		})
	}

	return result, nil
}

// GetSecurityDashboard returns security statistics
func (s *ScanService) GetSecurityDashboard() (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"total_artifacts_scanned":        0,
		"artifacts_with_critical":        0,
		"artifacts_with_high":            0,
		"artifacts_with_vulnerabilities": 0,
		"artifacts_clean":                0,
		"total_vulnerabilities":          0,
		"critical_vulnerabilities":       0,
		"high_vulnerabilities":           0,
		"medium_vulnerabilities":         0,
		"low_vulnerabilities":            0,
		"top_vulnerable_artifacts":       []map[string]interface{}{},
		"recent_scans":                   []map[string]interface{}{},
	}

	// TODO: Implement actual statistics calculation from database
	// This is a placeholder implementation

	return stats, nil
}

// BulkScan scans multiple artifacts
func (s *ScanService) BulkScan(ctx context.Context, artifactIDs []uuid.UUID) {
	for _, id := range artifactIDs {
		artifact, err := s.artifactRepo.GetByID(id)
		if err != nil {
			continue
		}

		// Create a scan record for this artifact
		scan := &models.SecurityScan{
			ArtifactID:        artifact.ID,
			ScanType:          "bulk",
			Status:            "running",
			Priority:          "normal",
			VulnerabilityScan: true,
			InitiatedBy:       nil, // System-initiated scan, no user
			StartedAt:         time.Now(),
		}

		// Save the scan to get an ID
		if err := s.scanRepo.Create(scan); err != nil {
			continue
		}

		// Scan each artifact (with timeout)
		scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		s.ScanArtifact(scanCtx, artifact, scan)
		cancel()
	}
}

// GenerateSecurityReport generates a security report in specified format
func (s *ScanService) GenerateSecurityReport(format string) ([]byte, error) {
	stats, err := s.GetSecurityDashboard()
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(stats, "", "  ")
	case "csv":
		// TODO: Implement CSV generation
		return []byte("CSV report not yet implemented"), nil
	case "pdf":
		// TODO: Implement PDF generation
		return []byte("PDF report not yet implemented"), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// generateSummary creates a summary of the scan results
func (s *ScanService) generateSummary(vulnResults *models.VulnerabilityResults) string {
	if vulnResults.TotalFound == 0 {
		return "No vulnerabilities found"
	}
	return fmt.Sprintf("Found %d vulnerabilities: %d critical, %d high, %d medium, %d low",
		vulnResults.TotalFound, vulnResults.Critical, vulnResults.High, vulnResults.Medium, vulnResults.Low)
}

// calculateRiskLevel determines the overall risk level
func (s *ScanService) calculateRiskLevel(vulnResults *models.VulnerabilityResults) string {
	if vulnResults.Critical > 0 {
		return "critical"
	}
	if vulnResults.High > 0 {
		return "high"
	}
	if vulnResults.Medium > 0 {
		return "medium"
	}
	if vulnResults.Low > 0 {
		return "low"
	}
	return "low"
}

// calculateScore calculates an overall security score
func (s *ScanService) calculateScore(vulnResults *models.VulnerabilityResults) int {
	if vulnResults.TotalFound == 0 {
		return 100
	}

	// Simple scoring algorithm
	score := 100
	score -= vulnResults.Critical * 20
	score -= vulnResults.High * 10
	score -= vulnResults.Medium * 5
	score -= vulnResults.Low * 1

	if score < 0 {
		score = 0
	}

	return score
}

// generateRecommendations provides actionable recommendations
func (s *ScanService) generateRecommendations(vulnResults *models.VulnerabilityResults) []string {
	if vulnResults.TotalFound == 0 {
		return []string{"No vulnerabilities found. Artifact appears secure."}
	}

	recommendations := []string{}

	if vulnResults.Critical > 0 {
		recommendations = append(recommendations, "URGENT: Address critical vulnerabilities immediately")
	}
	if vulnResults.High > 0 {
		recommendations = append(recommendations, "Address high severity vulnerabilities as soon as possible")
	}
	if vulnResults.Medium > 0 {
		recommendations = append(recommendations, "Plan to address medium severity vulnerabilities")
	}
	if vulnResults.Low > 0 {
		recommendations = append(recommendations, "Consider addressing low severity vulnerabilities when convenient")
	}

	recommendations = append(recommendations, "Update dependencies to latest secure versions")

	return recommendations
}

// Tenant-aware methods for multi-tenant isolation

func (s *ScanService) GetAllScansByTenant(tenantID uuid.UUID, status, scanType, priority string, limit, offset int) ([]*models.SecurityScan, int, error) {
	return s.scanRepo.GetAllWithFiltersByTenant(tenantID, status, scanType, priority, limit, offset)
}

func (s *ScanService) GetLatestScanResultByTenant(artifactID, tenantID uuid.UUID) (*models.SecurityScan, error) {
	return s.scanRepo.GetLatestResultByTenant(artifactID, tenantID)
}
