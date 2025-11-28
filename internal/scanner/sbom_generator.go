package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// SBOMFormat represents the format of the generated SBOM
type SBOMFormat string

const (
	CycloneDXFormat SBOMFormat = "cyclonedx"
	SPDXFormat      SBOMFormat = "spdx"
)

// SBOMGenerator generates Software Bill of Materials (SBOM) for artifacts
type SBOMGenerator struct {
	tempDir string
}

// NewSBOMGenerator creates a new SBOM generator
func NewSBOMGenerator(tempDir string) *SBOMGenerator {
	return &SBOMGenerator{
		tempDir: tempDir,
	}
}

// GenerateMavenSBOM generates an SBOM for a Maven project using cdxgen
func (g *SBOMGenerator) GenerateMavenSBOM(ctx context.Context, pomFilePath string, artifactID uuid.UUID, scanID uuid.UUID) (string, error) {
	// Create a project directory for Maven to work with
	projectDir := filepath.Join(g.tempDir, fmt.Sprintf("maven-project-%s-%s", artifactID.String(), scanID.String()))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	// Copy POM file to project directory as pom.xml
	pomData, err := os.ReadFile(pomFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read POM file: %w", err)
	}

	pomDestPath := filepath.Join(projectDir, "pom.xml")
	if err := os.WriteFile(pomDestPath, pomData, 0644); err != nil {
		return "", fmt.Errorf("failed to write pom.xml: %w", err)
	}

	// Generate SBOM using cdxgen
	bomPath := filepath.Join(projectDir, "bom.json")

	// Set a timeout for SBOM generation (e.g., 5 minutes)
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Run cdxgen to generate the SBOM
	cmd := exec.CommandContext(cmdCtx, "cdxgen", "-t", "java", "-o", bomPath, projectDir)
	cmd.Dir = projectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cdxgen failed: %w, output: %s", err, string(output))
	}

	// Verify SBOM was created and is valid JSON
	if _, err := os.Stat(bomPath); os.IsNotExist(err) {
		return "", fmt.Errorf("cdxgen did not create SBOM file")
	}

	// Validate JSON
	bomData, err := os.ReadFile(bomPath)
	if err != nil {
		return "", fmt.Errorf("failed to read generated SBOM: %w", err)
	}

	var bomJSON map[string]interface{}
	if err := json.Unmarshal(bomData, &bomJSON); err != nil {
		return "", fmt.Errorf("generated SBOM is not valid JSON: %w", err)
	}

	// Check for components
	components, ok := bomJSON["components"].([]interface{})
	if !ok || len(components) == 0 {
		return "", fmt.Errorf("SBOM contains no components - Maven may have failed to resolve dependencies")
	}

	return bomPath, nil
}

// GenerateNPMSBOM generates an SBOM for an NPM package using cdxgen
func (g *SBOMGenerator) GenerateNPMSBOM(ctx context.Context, packagePath string, artifactID, scanID int64) (string, error) {
	// Create a project directory for NPM to work with
	projectDir := filepath.Join(g.tempDir, fmt.Sprintf("npm-project-%d-%d", artifactID, scanID))
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}

	// Copy package contents to project directory
	// For NPM, we expect a package.json or a tarball
	bomPath := filepath.Join(projectDir, "bom.json")

	// Set a timeout for SBOM generation
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Run cdxgen to generate the SBOM
	cmd := exec.CommandContext(cmdCtx, "cdxgen", "-t", "npm", "-o", bomPath, packagePath)
	cmd.Dir = filepath.Dir(packagePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cdxgen failed: %w, output: %s", err, string(output))
	}

	// Verify SBOM was created
	if _, err := os.Stat(bomPath); os.IsNotExist(err) {
		return "", fmt.Errorf("cdxgen did not create SBOM file")
	}

	// Validate JSON
	bomData, err := os.ReadFile(bomPath)
	if err != nil {
		return "", fmt.Errorf("failed to read generated SBOM: %w", err)
	}

	var bomJSON map[string]interface{}
	if err := json.Unmarshal(bomData, &bomJSON); err != nil {
		return "", fmt.Errorf("generated SBOM is not valid JSON: %w", err)
	}

	return bomPath, nil
}

// GetSBOMMetadata extracts metadata from a generated SBOM
func (g *SBOMGenerator) GetSBOMMetadata(bomPath string) (map[string]interface{}, error) {
	bomData, err := os.ReadFile(bomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBOM: %w", err)
	}

	var bomJSON map[string]interface{}
	if err := json.Unmarshal(bomData, &bomJSON); err != nil {
		return nil, fmt.Errorf("failed to parse SBOM JSON: %w", err)
	}

	metadata := make(map[string]interface{})

	// Extract component count
	if components, ok := bomJSON["components"].([]interface{}); ok {
		metadata["component_count"] = len(components)

		// Extract component names and versions
		componentList := make([]map[string]string, 0)
		for _, comp := range components {
			if compMap, ok := comp.(map[string]interface{}); ok {
				name, _ := compMap["name"].(string)
				version, _ := compMap["version"].(string)
				componentList = append(componentList, map[string]string{
					"name":    name,
					"version": version,
				})
			}
		}
		metadata["components"] = componentList
	}

	// Extract SBOM format and spec version
	if bomFormat, ok := bomJSON["bomFormat"].(string); ok {
		metadata["bom_format"] = bomFormat
	}
	if specVersion, ok := bomJSON["specVersion"].(string); ok {
		metadata["spec_version"] = specVersion
	}

	// Extract metadata section if present
	if metaSection, ok := bomJSON["metadata"].(map[string]interface{}); ok {
		if timestamp, ok := metaSection["timestamp"].(string); ok {
			metadata["generated_at"] = timestamp
		}
		if component, ok := metaSection["component"].(map[string]interface{}); ok {
			if name, ok := component["name"].(string); ok {
				metadata["project_name"] = name
			}
			if version, ok := component["version"].(string); ok {
				metadata["project_version"] = version
			}
		}
	}

	return metadata, nil
}

// IsCDXGenAvailable checks if cdxgen is installed and available
func IsCDXGenAvailable() bool {
	cmd := exec.Command("cdxgen", "--version")
	return cmd.Run() == nil
}
