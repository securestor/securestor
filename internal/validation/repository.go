package validation

import (
	"errors"
	"regexp"
	"strings"

	"github.com/securestor/securestor/internal/models"
)

var repositoryNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

func ValidateCreateRepository(req *models.CreateRepositoryRequest) error {
	// Validate name
	if req.Name == "" {
		return errors.New("repository name is required")
	}

	if len(req.Name) < 3 || len(req.Name) > 255 {
		return errors.New("repository name must be between 3 and 255 characters")
	}

	if !repositoryNameRegex.MatchString(req.Name) {
		return errors.New("repository name must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}

	// Validate type
	validTypes := []string{"docker", "npm", "maven", "pypi", "helm", "generic"}
	if !contains(validTypes, req.Type) {
		return errors.New("invalid repository type")
	}

	// Validate repository type
	validRepoTypes := []string{"local", "remote", "cloud"}
	if !contains(validRepoTypes, req.RepositoryType) {
		return errors.New("invalid repository type")
	}

	// Validate remote URL for remote repositories
	if req.RepositoryType == "remote" {
		if req.RemoteURL == "" {
			return errors.New("remote URL is required for remote repositories")
		}
		if !strings.HasPrefix(req.RemoteURL, "http://") && !strings.HasPrefix(req.RemoteURL, "https://") {
			return errors.New("remote URL must start with http:// or https://")
		}
	}

	// Validate cloud storage settings
	if req.RepositoryType == "cloud" {
		if req.CloudProvider == "" {
			return errors.New("cloud provider is required for cloud repositories")
		}

		validCloudProviders := []string{"s3", "s3-compatible", "github", "aws-ecr", "azure", "gcp"}
		if !contains(validCloudProviders, req.CloudProvider) {
			return errors.New("invalid cloud provider")
		}

		// Validate S3 settings
		if req.CloudProvider == "s3" || req.CloudProvider == "s3-compatible" {
			if req.BucketName == "" {
				return errors.New("bucket name is required for S3 storage")
			}
			if req.AccessKeyID == "" || req.SecretAccessKey == "" {
				return errors.New("access credentials are required for S3 storage")
			}
			if req.CloudProvider == "s3" && req.Region == "" {
				return errors.New("region is required for AWS S3")
			}
			if req.CloudProvider == "s3-compatible" && req.Endpoint == "" {
				return errors.New("endpoint is required for S3-compatible storage")
			}
		}

		// Validate GitHub settings
		if req.CloudProvider == "github" {
			if req.GithubToken == "" {
				return errors.New("GitHub token is required")
			}
			if req.GithubOrg == "" {
				return errors.New("GitHub organization/username is required")
			}
		}

		// Validate AWS ECR settings
		if req.CloudProvider == "aws-ecr" {
			if req.AccessKeyID == "" || req.SecretAccessKey == "" {
				return errors.New("AWS credentials are required for ECR")
			}
			if req.Region == "" {
				return errors.New("region is required for AWS ECR")
			}
		}
	}

	// Validate replication settings
	if req.EnableReplication && req.RepositoryType == "local" {
		if len(req.ReplicationBuckets) == 0 {
			return errors.New("at least one replication target is required when replication is enabled")
		}

		validSyncFrequencies := []string{"realtime", "hourly", "daily", "weekly"}
		if req.SyncFrequency != "" && !contains(validSyncFrequencies, req.SyncFrequency) {
			return errors.New("invalid sync frequency")
		}
	}

	// Validate storage limits
	if req.MaxStorageGB < 0 {
		return errors.New("max storage must be positive")
	}

	if req.RetentionDays < 0 {
		return errors.New("retention days must be positive")
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
