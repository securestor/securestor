package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WorkflowConfig represents the workflow configuration structure
type WorkflowConfig struct {
	Workflows []WorkflowDefinition `json:"workflows"`
	Policies  PolicyConfig         `json:"policies"`
}

// WorkflowDefinition defines a scanning workflow
type WorkflowDefinition struct {
	Name          string   `json:"name"`
	ArtifactTypes []string `json:"artifact_types"`
	Scanners      []string `json:"scanners"`
	PolicyPath    string   `json:"policy_path"`
	Description   string   `json:"description"`
	Enabled       bool     `json:"enabled"`
}

// PolicyConfig holds policy configuration
type PolicyConfig struct {
	BaseURL    string `json:"base_url"`
	DefaultTTL int    `json:"default_ttl"`
}

// LoadWorkflowConfig loads workflow configuration from file or creates default
func LoadWorkflowConfig(configPath string) (*WorkflowConfig, error) {
	if configPath == "" {
		configPath = "configs/workflows.json"
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default configuration
		defaultConfig := getDefaultWorkflowConfig()

		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		// Save default config
		if err := saveWorkflowConfig(configPath, defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}

		return defaultConfig, nil
	}

	// Load existing config
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config WorkflowConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

// getDefaultWorkflowConfig returns the default workflow configuration
func getDefaultWorkflowConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Workflows: []WorkflowDefinition{
			{
				Name:          "DockerWorkflow",
				ArtifactTypes: []string{"docker", "oci"},
				Scanners:      []string{"syft", "grype", "trivy"},
				PolicyPath:    "/v1/data/securestor/docker_policy",
				Description:   "Comprehensive scanning workflow for Docker and OCI images",
				Enabled:       true,
			},
			{
				Name:          "MavenWorkflow",
				ArtifactTypes: []string{"maven"},
				Scanners:      []string{"syft", "depscan", "grype"},
				PolicyPath:    "/v1/data/securestor/maven_policy",
				Description:   "Security scanning workflow for Maven artifacts",
				Enabled:       true,
			},
			{
				Name:          "NPMWorkflow",
				ArtifactTypes: []string{"npm", "nodejs"},
				Scanners:      []string{"syft", "depscan"},
				PolicyPath:    "/v1/data/securestor/npm_policy",
				Description:   "Security scanning workflow for NPM packages",
				Enabled:       true,
			},
			{
				Name:          "PythonWorkflow",
				ArtifactTypes: []string{"python", "pypi"},
				Scanners:      []string{"syft", "depscan", "grype", "bandit", "trufflehog"},
				PolicyPath:    "/v1/data/securestor/python_policy",
				Description:   "Comprehensive security scanning workflow for Python packages with Bandit code analysis",
				Enabled:       true,
			},
			{
				Name:          "GenericWorkflow",
				ArtifactTypes: []string{"generic", "binary"},
				Scanners:      []string{"trivy", "trufflehog"},
				PolicyPath:    "/v1/data/securestor/generic_policy",
				Description:   "Basic security scanning for generic artifacts",
				Enabled:       true,
			},
		},
		Policies: PolicyConfig{
			BaseURL:    "http://opa:8181",
			DefaultTTL: 3600,
		},
	}
}

// saveWorkflowConfig saves workflow configuration to file
func saveWorkflowConfig(configPath string, config *WorkflowConfig) error {
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}
