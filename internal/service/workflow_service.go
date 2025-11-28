package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/config"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/repository"
	"github.com/securestor/securestor/internal/scanner"
	"github.com/securestor/securestor/internal/storage"
)

// WorkflowService handles workflow-based artifact processing
type WorkflowService struct {
	registry          *scanner.WorkflowRegistry
	config            *config.WorkflowConfig
	artifactRepo      *repository.ArtifactRepository
	scanRepo          *repository.ScanRepository
	vulnerabilityRepo *repository.VulnerabilityRepository
	blobStorage       *storage.BlobStorage
	logger            *log.Logger
	tempDir           string
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(
	workflowConfig *config.WorkflowConfig,
	artifactRepo *repository.ArtifactRepository,
	scanRepo *repository.ScanRepository,
	vulnerabilityRepo *repository.VulnerabilityRepository,
	blobStorage *storage.BlobStorage,
	logger *log.Logger,
	tempDir string,
) (*WorkflowService, error) {
	registry, err := scanner.BuildWorkflowRegistry(workflowConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow registry: %w", err)
	}

	return &WorkflowService{
		registry:          registry,
		config:            workflowConfig,
		artifactRepo:      artifactRepo,
		scanRepo:          scanRepo,
		vulnerabilityRepo: vulnerabilityRepo,
		blobStorage:       blobStorage,
		logger:            logger,
		tempDir:           tempDir,
	}, nil
}

// WorkflowInfo represents workflow information
type WorkflowInfo struct {
	Name          string   `json:"name"`
	ArtifactTypes []string `json:"artifact_types"`
	Scanners      []string `json:"scanners"`
	PolicyPath    string   `json:"policy_path"`
	Description   string   `json:"description"`
	Enabled       bool     `json:"enabled"`
	Available     bool     `json:"available"`
}

// ProcessArtifact processes an artifact using the appropriate workflow
func (ws *WorkflowService) ProcessArtifact(ctx context.Context, artifactID uuid.UUID) (*models.ScanResult, error) {
	ws.logger.Printf("Processing artifact %s with workflow system", artifactID.String())

	// Get artifact details
	artifact, err := ws.artifactRepo.GetByID(artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	// Determine artifact type
	artifactType := ws.detectArtifactType(artifact)
	ws.logger.Printf("Detected artifact type: %s for artifact %s", artifactType, artifactID.String())

	// Get workflow for artifact type
	workflow := ws.registry.Get(artifactType)
	if workflow == nil {
		return nil, fmt.Errorf("no workflow found for artifact type: %s", artifactType)
	}

	ws.logger.Printf("Using workflow: %s for artifact %d", workflow.Name, artifactID)

	// Download artifact to temp location
	tempFile, err := ws.downloadArtifactToTemp(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}
	defer ws.cleanup(tempFile)

	// Create scan result record
	scanResult := &models.ScanResult{
		ArtifactID:   artifactID,
		ScanType:     "workflow",
		Status:       "in_progress",
		StartTime:    time.Now(),
		WorkflowName: workflow.Name,
		Metadata: map[string]interface{}{
			"workflow_name": workflow.Name,
			"artifact_type": artifactType,
			"scanners":      ws.getScannerNames(workflow.Scanners),
			"policy_path":   workflow.PolicyPath,
		},
	}

	ws.logger.Printf("Starting workflow scan for artifact %d with workflow: %s", artifactID, workflow.Name)

	// Execute workflow scanners
	results := make([]*scanner.ScanResult, 0, len(workflow.Scanners))
	availableScanners := 0

	for _, s := range workflow.Scanners {
		if !s.IsAvailable() {
			ws.logger.Printf("Scanner %s not available, skipping", s.Name())
			continue
		}

		availableScanners++
		ws.logger.Printf("Running scanner: %s", s.Name())

		result, err := s.Scan(ctx, tempFile, artifactType)
		if err != nil {
			ws.logger.Printf("Scanner %s failed: %v", s.Name(), err)
			continue
		}

		results = append(results, result)
		ws.logger.Printf("Scanner %s completed, found %d vulnerabilities",
			s.Name(), len(result.Vulnerabilities))
	}

	if availableScanners == 0 {
		scanResult.Status = "failed"
		scanResult.Error = "No available scanners for this artifact type"
		endTime := time.Now()
		scanResult.EndTime = &endTime
		ws.logger.Printf("No available scanners for artifact type: %s", artifactType)
		return scanResult, fmt.Errorf("no available scanners for artifact type: %s", artifactType)
	}

	// Aggregate results
	aggregatedResult := ws.aggregateResults(results, workflow)

	// Update scan result
	scanResult.Status = "completed"
	endTime := time.Now()
	scanResult.EndTime = &endTime
	scanResult.VulnerabilityCount = len(aggregatedResult.Vulnerabilities)
	scanResult.CriticalCount = ws.countBySeverity(aggregatedResult.Vulnerabilities, "CRITICAL")
	scanResult.HighCount = ws.countBySeverity(aggregatedResult.Vulnerabilities, "HIGH")
	scanResult.MediumCount = ws.countBySeverity(aggregatedResult.Vulnerabilities, "MEDIUM")
	scanResult.LowCount = ws.countBySeverity(aggregatedResult.Vulnerabilities, "LOW")

	// Log vulnerability results
	ws.logger.Printf("Workflow scan completed for artifact %d: found %d vulnerabilities",
		artifactID, len(aggregatedResult.Vulnerabilities))

	if len(aggregatedResult.Vulnerabilities) > 0 {
		ws.logger.Printf("Vulnerability breakdown - Critical: %d, High: %d, Medium: %d, Low: %d",
			scanResult.CriticalCount, scanResult.HighCount, scanResult.MediumCount, scanResult.LowCount)
	}

	return scanResult, nil
}

// GetAvailableWorkflows returns available workflows with their status
func (ws *WorkflowService) GetAvailableWorkflows() []WorkflowInfo {
	workflows := make([]WorkflowInfo, 0, len(ws.config.Workflows))

	for _, workflowDef := range ws.config.Workflows {
		// Check if workflow is available (all scanners available)
		workflow := ws.registry.Get(workflowDef.ArtifactTypes[0])
		available := workflow != nil && ws.isWorkflowAvailable(workflow)

		workflows = append(workflows, WorkflowInfo{
			Name:          workflowDef.Name,
			ArtifactTypes: workflowDef.ArtifactTypes,
			Scanners:      workflowDef.Scanners,
			PolicyPath:    workflowDef.PolicyPath,
			Description:   workflowDef.Description,
			Enabled:       workflowDef.Enabled,
			Available:     available,
		})
	}

	return workflows
}

// GetWorkflowForArtifactType returns workflow info for specific artifact type
func (ws *WorkflowService) GetWorkflowForArtifactType(artifactType string) *WorkflowInfo {
	workflow := ws.registry.Get(artifactType)
	if workflow == nil {
		return nil
	}

	// Find config definition
	for _, workflowDef := range ws.config.Workflows {
		if workflowDef.Name == workflow.Name {
			return &WorkflowInfo{
				Name:          workflowDef.Name,
				ArtifactTypes: workflowDef.ArtifactTypes,
				Scanners:      workflowDef.Scanners,
				PolicyPath:    workflowDef.PolicyPath,
				Description:   workflowDef.Description,
				Enabled:       workflowDef.Enabled,
				Available:     ws.isWorkflowAvailable(workflow),
			}
		}
	}

	return nil
}

// detectArtifactType determines the artifact type from various sources
func (ws *WorkflowService) detectArtifactType(artifact *models.Artifact) string {
	// Check if explicitly set
	if artifact.Type != "" {
		return artifact.Type
	}

	// Detect from name/extension
	filename := strings.ToLower(artifact.Name)

	// Docker/OCI detection
	if strings.Contains(filename, "docker") || strings.Contains(filename, "oci") ||
		strings.HasSuffix(filename, ".tar") && (strings.Contains(filename, "image") || strings.Contains(filename, "container")) {
		return "docker"
	}

	// Maven detection
	if strings.HasSuffix(filename, ".jar") || strings.HasSuffix(filename, ".war") ||
		strings.HasSuffix(filename, ".pom") || strings.Contains(filename, "maven") {
		return "maven"
	}

	// NPM detection
	if strings.Contains(filename, "package.json") || strings.HasSuffix(filename, ".tgz") ||
		strings.Contains(filename, "node_modules") || strings.Contains(filename, "npm") {
		return "npm"
	}

	// Python detection
	if strings.HasSuffix(filename, ".whl") || strings.HasSuffix(filename, ".tar.gz") ||
		strings.Contains(filename, "requirements.txt") || strings.Contains(filename, "pypi") {
		return "python"
	}

	// Default to generic
	return "generic"
}

// Helper functions

func (ws *WorkflowService) downloadArtifactToTemp(ctx context.Context, artifact *models.Artifact) (string, error) {
	tempFile := filepath.Join(ws.tempDir, fmt.Sprintf("artifact_%d_%s", artifact.ID, artifact.Name))

	// Get storage ID from artifact metadata
	storageID, ok := artifact.Metadata["storage_id"].(string)
	if !ok {
		return "", fmt.Errorf("storage ID not found in artifact metadata")
	}
	data, err := ws.blobStorage.DownloadArtifact(ctx, artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
	if err != nil {
		return "", err
	}

	// Write to temporary file
	file, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return "", err
	}

	return tempFile, nil
}

func (ws *WorkflowService) cleanup(filePath string) {
	if err := os.Remove(filePath); err != nil {
		ws.logger.Printf("Failed to cleanup temp file %s: %v", filePath, err)
	}
}

func (ws *WorkflowService) getScannerNames(scanners []scanner.Scanner) []string {
	names := make([]string, len(scanners))
	for i, s := range scanners {
		names[i] = s.Name()
	}
	return names
}

func (ws *WorkflowService) isWorkflowAvailable(workflow *scanner.Workflow) bool {
	for _, s := range workflow.Scanners {
		if !s.IsAvailable() {
			return false
		}
	}
	return true
}

func (ws *WorkflowService) aggregateResults(results []*scanner.ScanResult, workflow *scanner.Workflow) *scanner.ScanResult {
	if len(results) == 0 {
		return &scanner.ScanResult{
			ScannerName:     workflow.Name,
			ScannerVersion:  "1.0.0",
			Vulnerabilities: []scanner.Vulnerability{},
			Summary: scanner.VulnerabilitySummary{
				Total: 0,
			},
		}
	}

	// Combine all vulnerabilities
	var allVulns []scanner.Vulnerability
	for _, result := range results {
		allVulns = append(allVulns, result.Vulnerabilities...)
	}

	// Deduplicate vulnerabilities by ID
	vulnMap := make(map[string]scanner.Vulnerability)
	for _, vuln := range allVulns {
		if existing, exists := vulnMap[vuln.ID]; !exists || vuln.CVSS > existing.CVSS {
			vulnMap[vuln.ID] = vuln
		}
	}

	// Convert back to slice
	finalVulns := make([]scanner.Vulnerability, 0, len(vulnMap))
	for _, vuln := range vulnMap {
		finalVulns = append(finalVulns, vuln)
	}

	// Calculate summary
	summary := scanner.VulnerabilitySummary{
		Critical: ws.countBySeverity(finalVulns, "CRITICAL"),
		High:     ws.countBySeverity(finalVulns, "HIGH"),
		Medium:   ws.countBySeverity(finalVulns, "MEDIUM"),
		Low:      ws.countBySeverity(finalVulns, "LOW"),
		Total:    len(finalVulns),
	}

	return &scanner.ScanResult{
		ScannerName:     workflow.Name,
		ScannerVersion:  "1.0.0",
		ArtifactType:    results[0].ArtifactType,
		Vulnerabilities: finalVulns,
		Summary:         summary,
	}
}

func (ws *WorkflowService) countBySeverity(vulns []scanner.Vulnerability, severity string) int {
	count := 0
	for _, vuln := range vulns {
		if vuln.Severity == severity {
			count++
		}
	}
	return count
}
