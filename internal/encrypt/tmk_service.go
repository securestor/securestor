package encrypt

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TenantMasterKey represents a tenant's master encryption key
type TenantMasterKey struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	EncryptedKey []byte
	KMSKeyID     string
	KeyVersion   int
	IsActive     bool
	CreatedAt    time.Time
	RotatedAt    *time.Time
	CreatedBy    uuid.UUID
}

// TMKService manages tenant master keys
type TMKService struct {
	db        *sql.DB
	kmsClient KMSClient
	keyCache  *KeyCache
}

// NewTMKService creates a new tenant master key service
func NewTMKService(db *sql.DB, kmsClient KMSClient) *TMKService {
	return &TMKService{
		db:        db,
		kmsClient: kmsClient,
		keyCache:  NewKeyCache(300), // 5 minutes in seconds
	}
}

// CreateTMK generates and stores a new tenant master key
func (s *TMKService) CreateTMK(ctx context.Context, tenantID, createdBy uuid.UUID, kmsKeyID string) (*TenantMasterKey, error) {
	// Generate random 32-byte TMK
	tmkPlaintext := make([]byte, 32)
	if _, err := rand.Read(tmkPlaintext); err != nil {
		return nil, fmt.Errorf("failed to generate TMK: %w", err)
	}

	// Encrypt TMK with KMS
	encryptedTMK, err := s.kmsClient.Encrypt(kmsKeyID, tmkPlaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TMK with KMS: %w", err)
	}

	// Store encrypted TMK in database
	tmk := &TenantMasterKey{
		ID:           uuid.New(),
		TenantID:     tenantID,
		EncryptedKey: encryptedTMK,
		KMSKeyID:     kmsKeyID,
		KeyVersion:   1,
		IsActive:     true,
		CreatedAt:    time.Now(),
		CreatedBy:    createdBy,
	}

	query := `
		INSERT INTO tenant_master_keys 
		(id, tenant_id, encrypted_key, kms_key_id, key_version, is_active, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	// Use sql.NullString to properly handle nil UUID
	var createdByParam interface{}
	if createdBy == uuid.Nil {
		createdByParam = nil
	} else {
		createdByParam = createdBy
	}

	err = s.db.QueryRowContext(ctx, query,
		tmk.ID, tmk.TenantID, tmk.EncryptedKey, tmk.KMSKeyID,
		tmk.KeyVersion, tmk.IsActive, tmk.CreatedAt, createdByParam,
	).Scan(&tmk.ID, &tmk.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to store TMK: %w", err)
	}

	// Cache the decrypted TMK
	cacheKey := fmt.Sprintf("tmk:%s:v%d", tenantID.String(), tmk.KeyVersion)
	s.keyCache.Set(cacheKey, tmkPlaintext)

	// Zero out plaintext from memory
	for i := range tmkPlaintext {
		tmkPlaintext[i] = 0
	}

	// Log TMK creation to audit log
	s.logKeyAudit(ctx, tenantID, createdBy, "TMK", tmk.ID.String(), "generate", true, nil, 0)

	return tmk, nil
}

// GetActiveTMK retrieves the active TMK for a tenant
func (s *TMKService) GetActiveTMK(ctx context.Context, tenantID uuid.UUID) ([]byte, error) {
	// Check cache first
	var tmkVersion int
	cacheKey := ""

	// Query database for active TMK metadata
	query := `
		SELECT id, encrypted_key, kms_key_id, key_version 
		FROM tenant_master_keys 
		WHERE tenant_id = $1 AND is_active = true
		LIMIT 1
	`

	var tmkID uuid.UUID
	var encryptedKey []byte
	var kmsKeyID string

	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tmkID, &encryptedKey, &kmsKeyID, &tmkVersion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active TMK found for tenant %s", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query TMK: %w", err)
	}

	// Check cache with version
	cacheKey = fmt.Sprintf("tmk:%s:v%d", tenantID.String(), tmkVersion)
	if cachedKey, found := s.keyCache.Get(cacheKey); found {
		s.logKeyAudit(ctx, tenantID, uuid.Nil, "TMK", tmkID.String(), "access", true, nil, 0)
		return cachedKey, nil
	}

	// Decrypt TMK with KMS
	startTime := time.Now()
	tmkPlaintext, err := s.kmsClient.Decrypt(kmsKeyID, encryptedKey)
	duration := int(time.Since(startTime).Milliseconds())

	if err != nil {
		s.logKeyAudit(ctx, tenantID, uuid.Nil, "TMK", tmkID.String(), "decrypt", false, err, duration)
		return nil, fmt.Errorf("failed to decrypt TMK: %w", err)
	}

	// Cache decrypted TMK
	s.keyCache.Set(cacheKey, tmkPlaintext)

	// Log successful decryption
	s.logKeyAudit(ctx, tenantID, uuid.Nil, "TMK", tmkID.String(), "decrypt", true, nil, duration)

	return tmkPlaintext, nil
}

// RotateTMK creates a new TMK version and marks the old one inactive
func (s *TMKService) RotateTMK(ctx context.Context, tenantID, rotatedBy uuid.UUID, kmsKeyID string) (*TenantMasterKey, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current active TMK version
	var currentVersion int
	err = tx.QueryRowContext(ctx,
		"SELECT key_version FROM tenant_master_keys WHERE tenant_id = $1 AND is_active = true",
		tenantID,
	).Scan(&currentVersion)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query current TMK version: %w", err)
	}

	// Mark current TMK as inactive
	if err != sql.ErrNoRows {
		_, err = tx.ExecContext(ctx,
			"UPDATE tenant_master_keys SET is_active = false, rotated_at = $1 WHERE tenant_id = $2 AND is_active = true",
			time.Now(), tenantID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to deactivate old TMK: %w", err)
		}

		// Invalidate cache for old TMK
		oldCacheKey := fmt.Sprintf("tmk:%s:v%d", tenantID.String(), currentVersion)
		s.keyCache.Delete(oldCacheKey)
	}

	// Generate new TMK
	tmkPlaintext := make([]byte, 32)
	if _, err := rand.Read(tmkPlaintext); err != nil {
		return nil, fmt.Errorf("failed to generate new TMK: %w", err)
	}

	// Encrypt with KMS
	encryptedTMK, err := s.kmsClient.Encrypt(kmsKeyID, tmkPlaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt new TMK: %w", err)
	}

	// Insert new TMK
	newTMK := &TenantMasterKey{
		ID:           uuid.New(),
		TenantID:     tenantID,
		EncryptedKey: encryptedTMK,
		KMSKeyID:     kmsKeyID,
		KeyVersion:   currentVersion + 1,
		IsActive:     true,
		CreatedAt:    time.Now(),
		CreatedBy:    rotatedBy,
	}

	query := `
		INSERT INTO tenant_master_keys 
		(id, tenant_id, encrypted_key, kms_key_id, key_version, is_active, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = tx.ExecContext(ctx, query,
		newTMK.ID, newTMK.TenantID, newTMK.EncryptedKey, newTMK.KMSKeyID,
		newTMK.KeyVersion, newTMK.IsActive, newTMK.CreatedAt, newTMK.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new TMK: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit TMK rotation: %w", err)
	}

	// Cache new TMK
	newCacheKey := fmt.Sprintf("tmk:%s:v%d", tenantID.String(), newTMK.KeyVersion)
	s.keyCache.Set(newCacheKey, tmkPlaintext)

	// Zero out plaintext
	for i := range tmkPlaintext {
		tmkPlaintext[i] = 0
	}

	// Log rotation
	s.logKeyAudit(ctx, tenantID, rotatedBy, "TMK", newTMK.ID.String(), "rotate", true, nil, 0)

	return newTMK, nil
}

// GetTMKStatus returns encryption status for a tenant
func (s *TMKService) GetTMKStatus(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT 
			key_version, 
			created_at, 
			rotated_at,
			EXTRACT(DAY FROM NOW() - COALESCE(rotated_at, created_at)) as days_since_rotation,
			access_count
		FROM tenant_master_keys
		WHERE tenant_id = $1 AND is_active = true
	`

	var version int
	var createdAt time.Time
	var rotatedAt sql.NullTime
	var daysSinceRotation float64
	var accessCount int64

	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
		&version, &createdAt, &rotatedAt, &daysSinceRotation, &accessCount,
	)

	if err == sql.ErrNoRows {
		return map[string]interface{}{
			"has_tmk": false,
			"message": "No TMK found for tenant",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query TMK status: %w", err)
	}

	status := map[string]interface{}{
		"has_tmk":              true,
		"key_version":          version,
		"created_at":           createdAt,
		"days_since_rotation":  int(daysSinceRotation),
		"access_count":         accessCount,
		"rotation_recommended": daysSinceRotation > 90,
	}

	if rotatedAt.Valid {
		status["last_rotated"] = rotatedAt.Time
	}

	return status, nil
}

// logKeyAudit logs key operations to audit log
func (s *TMKService) logKeyAudit(ctx context.Context, tenantID, userID uuid.UUID, keyType, keyID, operation string, success bool, err error, durationMS int) {
	var errorMsg *string
	if err != nil {
		msg := err.Error()
		errorMsg = &msg
	}

	var userIDPtr *uuid.UUID
	if userID != uuid.Nil {
		userIDPtr = &userID
	}

	query := `
		INSERT INTO key_audit_log 
		(tenant_id, user_id, key_type, key_id, operation, success, error_message, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	// Fire and forget - don't block on audit logging
	go func() {
		_, _ = s.db.ExecContext(ctx, query,
			tenantID, userIDPtr, keyType, keyID, operation, success, errorMsg, durationMS,
		)
	}()
}

// ExportTMKForBackup exports encrypted TMK (for disaster recovery)
func (s *TMKService) ExportTMKForBackup(ctx context.Context, tenantID, requestedBy uuid.UUID) (string, error) {
	query := `
		SELECT encrypted_key, kms_key_id, key_version
		FROM tenant_master_keys
		WHERE tenant_id = $1 AND is_active = true
	`

	var encryptedKey []byte
	var kmsKeyID string
	var version int

	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(&encryptedKey, &kmsKeyID, &version)
	if err != nil {
		return "", fmt.Errorf("failed to export TMK: %w", err)
	}

	// Log export operation
	s.logKeyAudit(ctx, tenantID, requestedBy, "TMK", "", "export", true, nil, 0)

	// Return base64-encoded encrypted key with metadata
	export := fmt.Sprintf("v%d:%s:%s", version, kmsKeyID, base64.StdEncoding.EncodeToString(encryptedKey))
	return export, nil
}
