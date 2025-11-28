// internal/validation/artifact.go
package validation

import (
	"errors"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

func ValidateArtifact(artifact *models.Artifact) error {
	if artifact.Name == "" {
		return errors.New("artifact name is required")
	}
	if artifact.Version == "" {
		return errors.New("version is required")
	}
	if artifact.Size <= 0 {
		return errors.New("size must be positive")
	}
	if artifact.RepositoryID == uuid.Nil {
		return errors.New("valid repository ID is required")
	}
	return nil
}
