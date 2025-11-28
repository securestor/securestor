package service

import (
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/repository"
)

type ArtifactService struct {
	repo *repository.ArtifactRepository
}

func NewArtifactService(repo *repository.ArtifactRepository) *ArtifactService {
	return &ArtifactService{repo: repo}
}

func (s *ArtifactService) Create(artifact *models.Artifact) error {
	return s.repo.Create(artifact)
}

func (s *ArtifactService) GetByID(id uuid.UUID) (*models.Artifact, error) {
	return s.repo.GetByID(id)
}

func (s *ArtifactService) List(filter *models.ArtifactFilter) ([]models.Artifact, int, error) {
	return s.repo.List(filter)
}

func (s *ArtifactService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

// Tenant-aware methods for multi-tenant isolation
func (s *ArtifactService) ListByTenant(tenantID uuid.UUID, filter *models.ArtifactFilter) ([]models.Artifact, int, error) {
	return s.repo.ListByTenant(tenantID, filter)
}

func (s *ArtifactService) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Artifact, error) {
	return s.repo.GetByIDAndTenant(id, tenantID)
}

func (s *ArtifactService) CreateWithTenant(tenantID uuid.UUID, artifact *models.Artifact) error {
	artifact.TenantID = tenantID
	return s.repo.Create(artifact)
}

func (s *ArtifactService) DeleteByTenant(id, tenantID uuid.UUID) error {
	return s.repo.DeleteByTenant(id, tenantID)
}

func (s *ArtifactService) IncrementDownloads(id uuid.UUID) error {
	return s.repo.IncrementDownloads(id)
}

func (s *ArtifactService) GetByNameVersionType(name, version, artifactType string) ([]models.Artifact, error) {
	// Use search to find artifacts matching name, version and type
	searchQuery := name + " " + version
	filter := &models.ArtifactFilter{
		Search: searchQuery,
		Types:  []string{artifactType},
		Limit:  10,
	}
	artifacts, _, err := s.repo.List(filter)
	if err != nil {
		return nil, err
	}

	// Filter results to exact matches
	var exactMatches []models.Artifact
	for _, artifact := range artifacts {
		if artifact.Name == name && artifact.Version == version && artifact.Type == artifactType {
			exactMatches = append(exactMatches, artifact)
		}
	}

	return exactMatches, nil
}

func (s *ArtifactService) GetDashboardStats() (map[string]interface{}, error) {
	// TODO: Implement actual statistics calculation
	stats := map[string]interface{}{
		"totalArtifacts": 27504,
		"totalStorage":   "335.5 GB",
		"downloadsToday": 15847,
		"activeUsers":    142,
	}
	return stats, nil
}
