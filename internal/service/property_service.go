package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/repository"
)

// PropertyService handles business logic for artifact properties
type PropertyService struct {
	repo         *repository.PropertyRepository
	auditService *AuditService
	log          *logger.Logger
	// Encryption key cache per tenant (simplified - in production use TMKService)
	encryptionKeys map[uuid.UUID][]byte
}

// NewPropertyService creates a new property service
func NewPropertyService(
	repo *repository.PropertyRepository,
	auditService *AuditService,
	log *logger.Logger,
) *PropertyService {
	return &PropertyService{
		repo:           repo,
		auditService:   auditService,
		log:            log,
		encryptionKeys: make(map[uuid.UUID][]byte),
	}
}

var (
	// Property key must be alphanumeric with dots, dashes, underscores
	propertyKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,255}$`)

	// System property prefixes (read-only, managed by system)
	systemPropertyPrefixes = []string{
		"securestor.",
		"artifact.",
		"scanner.",
		"compliance.",
		"signature.",
	}
)

// Validation constants
const (
	MaxValueSize       = 64 * 1024 // 64KB max value size
	MaxDescriptionSize = 1024      // 1KB max description
	MaxTagsCount       = 20        // Maximum 20 tags per property
	MaxBatchSize       = 100       // Maximum 100 properties in batch operation
)

// ValidatePropertyKey validates property key format
func ValidatePropertyKey(key string) error {
	if key == "" {
		return fmt.Errorf("property key cannot be empty")
	}
	if len(key) > 255 {
		return fmt.Errorf("property key too long (max 255 characters)")
	}
	if !propertyKeyRegex.MatchString(key) {
		return fmt.Errorf("property key must contain only alphanumeric characters, dots, dashes, and underscores")
	}
	return nil
}

// IsSystemProperty checks if a key is a system property
func IsSystemProperty(key string) bool {
	for _, prefix := range systemPropertyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// ValidatePropertyValue validates property value size
func ValidatePropertyValue(value string) error {
	if len(value) > MaxValueSize {
		return fmt.Errorf("property value too large (max %d bytes)", MaxValueSize)
	}
	return nil
}

// ValidateDescription validates description size
func ValidateDescription(description string) error {
	if len(description) > MaxDescriptionSize {
		return fmt.Errorf("description too long (max %d bytes)", MaxDescriptionSize)
	}
	return nil
}

// ValidateTags validates tags count
func ValidateTags(tags []string) error {
	if len(tags) > MaxTagsCount {
		return fmt.Errorf("too many tags (max %d)", MaxTagsCount)
	}
	return nil
}

// getOrCreateEncryptionKey gets or creates an encryption key for a tenant
func (s *PropertyService) getOrCreateEncryptionKey(tenantID uuid.UUID) []byte {
	// Check cache
	if key, exists := s.encryptionKeys[tenantID]; exists {
		return key
	}

	// Generate new key (32 bytes for AES-256)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(fmt.Sprintf("failed to generate encryption key: %v", err))
	}

	s.encryptionKeys[tenantID] = key
	return key
}

// encryptValue encrypts a property value using AES-256-GCM
func (s *PropertyService) encryptValue(tenantID uuid.UUID, value string) (string, string, string, error) {
	key := s.getOrCreateEncryptionKey(tenantID)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(value), nil)

	encryptedValue := base64.StdEncoding.EncodeToString(ciphertext)
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	keyID := tenantID.String() // Use tenant ID as key ID for now

	return encryptedValue, keyID, nonceB64, nil
}

// decryptValue decrypts a property value using AES-256-GCM
func (s *PropertyService) decryptValue(tenantID uuid.UUID, encryptedValue, nonce string) (string, error) {
	key := s.getOrCreateEncryptionKey(tenantID)

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// CreateProperty creates a new property with validation and encryption
func (s *PropertyService) CreateProperty(ctx context.Context, tenantID uuid.UUID, repositoryID, artifactID string, req *models.CreatePropertyRequest, userID uuid.UUID) (*models.ArtifactProperty, error) {
	// Validate key
	if err := ValidatePropertyKey(req.Key); err != nil {
		return nil, fmt.Errorf("invalid property key: %w", err)
	}

	// Validate value
	if err := ValidatePropertyValue(req.Value); err != nil {
		return nil, fmt.Errorf("invalid property value: %w", err)
	}

	// Validate description
	if req.Description != "" {
		if err := ValidateDescription(req.Description); err != nil {
			return nil, fmt.Errorf("invalid description: %w", err)
		}
	}

	// Validate tags
	if err := ValidateTags(req.Tags); err != nil {
		return nil, fmt.Errorf("invalid tags: %w", err)
	}

	// Check if trying to create system property
	if IsSystemProperty(req.Key) {
		return nil, fmt.Errorf("cannot create system property (key starts with reserved prefix)")
	}

	// Check if property already exists (for single-value properties)
	if !req.IsMultiValue {
		existing, err := s.repo.GetByKey(ctx, tenantID, artifactID, req.Key)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("property already exists (use multi-value mode or update existing)")
		}
	}

	property := &models.ArtifactProperty{
		TenantID:     tenantID,
		RepositoryID: repositoryID,
		ArtifactID:   artifactID,
		Key:          req.Key,
		Value:        req.Value,
		ValueType:    req.ValueType,
		IsSensitive:  req.IsSensitive,
		IsSystem:     false,
		IsMultiValue: req.IsMultiValue,
		Tags:         req.Tags,
		CreatedBy:    &userID,
		UpdatedBy:    &userID,
	}

	// Set description if provided
	if req.Description != "" {
		property.Description = &req.Description
	}

	// Encrypt sensitive values
	if req.IsSensitive {
		encryptedValue, keyID, nonce, err := s.encryptValue(tenantID, req.Value)
		if err != nil {
			s.log.Error("Failed to encrypt sensitive property", err)
			return nil, fmt.Errorf("failed to encrypt sensitive value: %w", err)
		}
		property.EncryptedValue = &encryptedValue
		property.EncryptionKeyID = &keyID
		algo := "AES-256-GCM"
		property.EncryptionAlgorithm = &algo
		property.Nonce = &nonce
		// Clear plaintext value for sensitive properties
		property.Value = ""
	}

	// Create in database
	if err := s.repo.Create(ctx, property); err != nil {
		s.log.Error("Failed to create property", err)
		return nil, fmt.Errorf("failed to create property: %w", err)
	}

	s.log.Info(fmt.Sprintf("Property created: tenant=%s artifact=%s key=%s sensitive=%v", tenantID, artifactID, req.Key, req.IsSensitive))

	return property, nil
}

// GetProperty retrieves a property and decrypts if needed
func (s *PropertyService) GetProperty(ctx context.Context, tenantID, propertyID uuid.UUID) (*models.PropertyResponse, error) {
	property, err := s.repo.Get(ctx, tenantID, propertyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get property: %w", err)
	}

	response := s.toPropertyResponse(property, false)

	// Decrypt sensitive values if needed
	if property.IsSensitive && property.EncryptedValue != nil && property.Nonce != nil {
		decryptedValue, err := s.decryptValue(tenantID, *property.EncryptedValue, *property.Nonce)
		if err != nil {
			s.log.Error("Failed to decrypt sensitive property", err)
			return nil, fmt.Errorf("failed to decrypt sensitive value: %w", err)
		}
		response.Value = decryptedValue
		response.Masked = false
	}

	return response, nil
}

// GetPropertiesByArtifact retrieves all properties for an artifact
func (s *PropertyService) GetPropertiesByArtifact(ctx context.Context, tenantID uuid.UUID, artifactID string, limit, offset int, maskSensitive bool) ([]*models.PropertyResponse, int, error) {
	properties, total, err := s.repo.GetByArtifact(ctx, tenantID, artifactID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get properties: %w", err)
	}

	responses := make([]*models.PropertyResponse, len(properties))
	for i, property := range properties {
		response := s.toPropertyResponse(property, maskSensitive)

		// Decrypt sensitive values if not masking
		if !maskSensitive && property.IsSensitive && property.EncryptedValue != nil && property.Nonce != nil {
			decryptedValue, err := s.decryptValue(tenantID, *property.EncryptedValue, *property.Nonce)
			if err != nil {
				s.log.Error("Failed to decrypt sensitive property", err)
				// Continue with masked value on decryption error
			} else {
				response.Value = decryptedValue
				response.Masked = false
			}
		}

		responses[i] = response
	}

	return responses, total, nil
}

// UpdateProperty updates a property with validation and encryption
func (s *PropertyService) UpdateProperty(ctx context.Context, tenantID, propertyID uuid.UUID, req *models.UpdatePropertyRequest, userID uuid.UUID) (*models.ArtifactProperty, error) {
	// Get existing property
	property, err := s.repo.Get(ctx, tenantID, propertyID)
	if err != nil {
		return nil, fmt.Errorf("property not found: %w", err)
	}

	// Prevent updating system properties
	if property.IsSystem {
		return nil, fmt.Errorf("cannot update system property")
	}

	// Validate value if provided
	if req.Value != nil {
		if err := ValidatePropertyValue(*req.Value); err != nil {
			return nil, fmt.Errorf("invalid property value: %w", err)
		}
		property.Value = *req.Value
	}

	// Validate description if provided
	if req.Description != nil {
		if err := ValidateDescription(*req.Description); err != nil {
			return nil, fmt.Errorf("invalid description: %w", err)
		}
		property.Description = req.Description
	}

	// Validate tags if provided
	if req.Tags != nil {
		if err := ValidateTags(req.Tags); err != nil {
			return nil, fmt.Errorf("invalid tags: %w", err)
		}
		property.Tags = req.Tags
	}

	// Update other fields
	if req.ValueType != nil {
		property.ValueType = *req.ValueType
	}
	if req.IsSensitive != nil {
		property.IsSensitive = *req.IsSensitive
	}

	property.UpdatedBy = &userID

	// Re-encrypt if sensitive and value changed
	if property.IsSensitive && req.Value != nil {
		encryptedValue, keyID, nonce, err := s.encryptValue(tenantID, *req.Value)
		if err != nil {
			s.log.Error("Failed to encrypt sensitive property", err)
			return nil, fmt.Errorf("failed to encrypt sensitive value: %w", err)
		}
		property.EncryptedValue = &encryptedValue
		property.EncryptionKeyID = &keyID
		algo := "AES-256-GCM"
		property.EncryptionAlgorithm = &algo
		property.Nonce = &nonce
		property.Value = "" // Clear plaintext
	}

	// Update in database
	if err := s.repo.Update(ctx, property); err != nil {
		s.log.Error("Failed to update property", err)
		return nil, fmt.Errorf("failed to update property: %w", err)
	}

	s.log.Info(fmt.Sprintf("Property updated: tenant=%s property_id=%s key=%s", tenantID, propertyID, property.Key))

	return property, nil
}

// DeleteProperty deletes a property
func (s *PropertyService) DeleteProperty(ctx context.Context, tenantID, propertyID, userID uuid.UUID) error {
	// Get property first to check if it's system property
	property, err := s.repo.Get(ctx, tenantID, propertyID)
	if err != nil {
		return fmt.Errorf("property not found: %w", err)
	}

	// Prevent deleting system properties
	if property.IsSystem {
		return fmt.Errorf("cannot delete system property")
	}

	// Delete from database
	if err := s.repo.Delete(ctx, tenantID, propertyID); err != nil {
		s.log.Error("Failed to delete property", err)
		return fmt.Errorf("failed to delete property: %w", err)
	}

	s.log.Info(fmt.Sprintf("Property deleted: tenant=%s property_id=%s key=%s", tenantID, propertyID, property.Key))

	return nil
}

// SearchProperties searches for properties with decryption support
func (s *PropertyService) SearchProperties(ctx context.Context, tenantID uuid.UUID, req *models.SearchPropertiesRequest, maskSensitive bool) ([]*models.PropertySearchResult, int, error) {
	results, total, err := s.repo.Search(ctx, tenantID, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search properties: %w", err)
	}

	// Mask sensitive values
	for _, result := range results {
		if result.IsSensitive {
			result.Value = "***MASKED***"
		}
	}

	return results, total, nil
}

// BatchCreateProperties creates multiple properties in a transaction
func (s *PropertyService) BatchCreateProperties(ctx context.Context, tenantID uuid.UUID, repositoryID string, properties []struct {
	ArtifactID string
	Request    models.CreatePropertyRequest
}, userID uuid.UUID) error {
	if len(properties) == 0 {
		return fmt.Errorf("no properties provided")
	}
	if len(properties) > MaxBatchSize {
		return fmt.Errorf("too many properties (max %d)", MaxBatchSize)
	}

	propModels := make([]*models.ArtifactProperty, len(properties))

	for i, prop := range properties {
		// Validate each property
		if err := ValidatePropertyKey(prop.Request.Key); err != nil {
			return fmt.Errorf("property %d: invalid key: %w", i, err)
		}
		if err := ValidatePropertyValue(prop.Request.Value); err != nil {
			return fmt.Errorf("property %d: invalid value: %w", i, err)
		}
		if IsSystemProperty(prop.Request.Key) {
			return fmt.Errorf("property %d: cannot create system property", i)
		}

		property := &models.ArtifactProperty{
			TenantID:     tenantID,
			RepositoryID: repositoryID,
			ArtifactID:   prop.ArtifactID,
			Key:          prop.Request.Key,
			Value:        prop.Request.Value,
			ValueType:    prop.Request.ValueType,
			IsSensitive:  prop.Request.IsSensitive,
			IsSystem:     false,
			IsMultiValue: prop.Request.IsMultiValue,
			Tags:         prop.Request.Tags,
			CreatedBy:    &userID,
			UpdatedBy:    &userID,
		}

		if prop.Request.Description != "" {
			property.Description = &prop.Request.Description
		}

		// Encrypt sensitive values
		if prop.Request.IsSensitive {
			encryptedValue, keyID, nonce, err := s.encryptValue(tenantID, prop.Request.Value)
			if err != nil {
				s.log.Error("Failed to encrypt sensitive property in batch", err)
				return fmt.Errorf("property %d: failed to encrypt: %w", i, err)
			}
			property.EncryptedValue = &encryptedValue
			property.EncryptionKeyID = &keyID
			algo := "AES-256-GCM"
			property.EncryptionAlgorithm = &algo
			property.Nonce = &nonce
			property.Value = ""
		}

		propModels[i] = property
	}

	// Batch create
	if err := s.repo.BatchCreate(ctx, propModels); err != nil {
		s.log.Error("Failed to batch create properties", err)
		return fmt.Errorf("failed to batch create properties: %w", err)
	}

	s.log.Info(fmt.Sprintf("Properties batch created: tenant=%s count=%d", tenantID, len(propModels)))

	return nil
}

// BatchDeleteProperties deletes multiple properties in a transaction
func (s *PropertyService) BatchDeleteProperties(ctx context.Context, tenantID uuid.UUID, propertyIDs []uuid.UUID, userID uuid.UUID) error {
	if len(propertyIDs) == 0 {
		return fmt.Errorf("no property IDs provided")
	}
	if len(propertyIDs) > MaxBatchSize {
		return fmt.Errorf("too many property IDs (max %d)", MaxBatchSize)
	}

	// Check for system properties (cannot delete)
	for _, propertyID := range propertyIDs {
		property, err := s.repo.Get(ctx, tenantID, propertyID)
		if err != nil {
			continue // Skip non-existent properties
		}
		if property.IsSystem {
			return fmt.Errorf("cannot delete system property: %s", property.Key)
		}
	}

	// Batch delete
	if err := s.repo.BatchDelete(ctx, tenantID, propertyIDs); err != nil {
		s.log.Error("Failed to batch delete properties", err)
		return fmt.Errorf("failed to batch delete properties: %w", err)
	}

	s.log.Info(fmt.Sprintf("Properties batch deleted: tenant=%s count=%d", tenantID, len(propertyIDs)))

	return nil
}

// GetStatistics returns property statistics for a tenant
func (s *PropertyService) GetStatistics(ctx context.Context, tenantID uuid.UUID) (*models.PropertyStatistics, error) {
	stats, err := s.repo.GetStatistics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return stats, nil
}

// toPropertyResponse converts ArtifactProperty to PropertyResponse
func (s *PropertyService) toPropertyResponse(property *models.ArtifactProperty, maskSensitive bool) *models.PropertyResponse {
	response := &models.PropertyResponse{
		ID:           property.ID,
		TenantID:     property.TenantID,
		RepositoryID: property.RepositoryID,
		ArtifactID:   property.ArtifactID,
		Key:          property.Key,
		Value:        property.Value,
		ValueType:    property.ValueType,
		IsSensitive:  property.IsSensitive,
		IsSystem:     property.IsSystem,
		IsMultiValue: property.IsMultiValue,
		Description:  property.Description,
		Tags:         property.Tags,
		CreatedBy:    property.CreatedBy,
		CreatedAt:    property.CreatedAt,
		UpdatedBy:    property.UpdatedBy,
		UpdatedAt:    property.UpdatedAt,
		Version:      property.Version,
		Masked:       false,
	}

	// Mask sensitive values if requested
	if maskSensitive && property.IsSensitive {
		response.Value = "***MASKED***"
		response.Masked = true
	}

	return response
}
