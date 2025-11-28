package service

import (
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/repository"
)

type RepositoryService struct {
	repo *repository.RepositoryRepository
}

func NewRepositoryService(repo *repository.RepositoryRepository) *RepositoryService {
	return &RepositoryService{repo: repo}
}

func (s *RepositoryService) Create(req *models.CreateRepositoryRequest) (*models.RepositoryResponse, error) {
	return s.repo.Create(req)
}

func (s *RepositoryService) GetByIDWithStats(id uuid.UUID) (*models.RepositoryResponse, error) {
	return s.repo.GetByIDWithStats(id)
}

func (s *RepositoryService) ListWithStats() ([]models.RepositoryResponse, error) {
	return s.repo.ListWithStats()
}

func (s *RepositoryService) GetStats() (*models.RepositoryStats, error) {
	return s.repo.GetStats()
}

func (s *RepositoryService) GetByID(id uuid.UUID) (*models.Repository, error) {
	return s.repo.GetByID(id)
}

func (s *RepositoryService) List() ([]models.Repository, error) {
	return s.repo.List()
}

func (s *RepositoryService) ListByTenant(tenantID uuid.UUID) ([]models.Repository, error) {
	return s.repo.ListByTenant(tenantID)
}

func (s *RepositoryService) ListWithStatsByTenant(tenantID uuid.UUID) ([]models.RepositoryResponse, error) {
	return s.repo.ListWithStatsByTenant(tenantID)
}

func (s *RepositoryService) GetStatsByTenant(tenantID uuid.UUID) (*models.RepositoryStats, error) {
	return s.repo.GetStatsByTenant(tenantID)
}

func (s *RepositoryService) CreateWithTenant(tenantID uuid.UUID, req *models.CreateRepositoryRequest) (*models.RepositoryResponse, error) {
	return s.repo.CreateWithTenant(tenantID, req)
}

func (s *RepositoryService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *RepositoryService) GetRepositoryTenantID(id uuid.UUID) (uuid.UUID, error) {
	return s.repo.GetRepositoryTenantID(id)
}
