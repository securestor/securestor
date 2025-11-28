package service

import (
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/repository"
)

type ComplianceService struct {
	complianceRepo    *repository.ComplianceRepository
	vulnerabilityRepo *repository.VulnerabilityRepository
}

func NewComplianceService(complianceRepo *repository.ComplianceRepository, vulnerabilityRepo *repository.VulnerabilityRepository) *ComplianceService {
	return &ComplianceService{
		complianceRepo:    complianceRepo,
		vulnerabilityRepo: vulnerabilityRepo,
	}
}

func (s *ComplianceService) CreateAudit(audit *models.ComplianceAudit) error {
	return s.complianceRepo.Create(audit)
}

func (s *ComplianceService) GetByArtifactID(artifactID uuid.UUID) (*models.ComplianceAudit, error) {
	return s.complianceRepo.GetByArtifactID(artifactID)
}

func (s *ComplianceService) CreateVulnerabilityScan(vuln *models.Vulnerability) error {
	return s.vulnerabilityRepo.Create(vuln)
}

func (s *ComplianceService) GetVulnerabilities(artifactID uuid.UUID) (*models.Vulnerability, error) {
	return s.vulnerabilityRepo.GetByArtifactID(artifactID)
}

func (s *ComplianceService) GenerateReport() (map[string]interface{}, error) {
	// TODO: Implement actual report generation
	report := map[string]interface{}{
		"totalArtifacts": 100,
		"compliant":      85,
		"review":         10,
		"nonCompliant":   5,
	}
	return report, nil
}
