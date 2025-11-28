package encrypt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// RewrapService handles re-wrapping DEKs after TMK rotation
// without re-encrypting the actual blob ciphertext
type RewrapService struct {
	db            *sql.DB
	tmkService    *TMKService
	kmsClient     KMSClient
	storagePath   string
	batchSize     int
	delayBetween  time.Duration
	progressMutex sync.RWMutex
	activeJobs    map[uuid.UUID]*RewrapJob
	auditCallback func(ctx context.Context, event RewrapAuditEvent)
}

// RewrapJob tracks progress of a re-wrap operation
type RewrapJob struct {
	JobID           uuid.UUID
	TenantID        uuid.UUID
	OldTMKVersion   int
	NewTMKVersion   int
	StartedAt       time.Time
	CompletedAt     time.Time
	Status          string // "running", "completed", "failed", "cancelled"
	TotalArtifacts  int64
	ProcessedCount  int64
	SuccessCount    int64
	FailedCount     int64
	CurrentBatch    int
	Errors          []RewrapError
	LastProcessedID uuid.UUID
}

// RewrapError represents a failure during re-wrap
type RewrapError struct {
	ArtifactID uuid.UUID
	Error      string
	Timestamp  time.Time
	Retryable  bool
}

// RewrapAuditEvent tracks audit events for re-wrap operations
type RewrapAuditEvent struct {
	JobID      uuid.UUID
	TenantID   uuid.UUID
	ArtifactID uuid.UUID
	Operation  string // "rewrap_start", "rewrap_success", "rewrap_failure"
	OldKEKID   string
	NewKEKID   string
	Timestamp  time.Time
	Error      string
}

// ArtifactMetadata represents the meta.json structure
type ArtifactMetadata struct {
	TenantID        string    `json:"tenant_id"`
	ArtifactID      string    `json:"artifact_id"`
	RepositoryID    string    `json:"repository_id"`
	WrappedDEK      string    `json:"wrapped_dek"`
	KEKKeyID        string    `json:"kek_key_id"`
	TMKVersion      int       `json:"tmk_version"`
	EncryptionAlgo  string    `json:"encryption_algorithm"`
	Nonce           string    `json:"nonce"`
	LastRewrappedAt time.Time `json:"last_rewrapped_at,omitempty"`
}

// RewrapConfig contains configuration for the re-wrap service
type RewrapConfig struct {
	BatchSize      int           // Number of artifacts to process per batch
	DelayBetween   time.Duration // Delay between batches to avoid overwhelming KMS
	MaxRetries     int           // Maximum retries for failed artifacts
	RetryDelay     time.Duration // Delay before retrying failed artifacts
	MaxConcurrency int           // Maximum concurrent re-wrap operations
}

// NewRewrapService creates a new re-wrap service
func NewRewrapService(db *sql.DB, tmkService *TMKService, kmsClient KMSClient, storagePath string, config RewrapConfig) *RewrapService {
	batchSize := config.BatchSize
	if batchSize == 0 {
		batchSize = 100 // Default batch size
	}

	delayBetween := config.DelayBetween
	if delayBetween == 0 {
		delayBetween = 1 * time.Second // Default delay
	}

	return &RewrapService{
		db:           db,
		tmkService:   tmkService,
		kmsClient:    kmsClient,
		storagePath:  storagePath,
		batchSize:    batchSize,
		delayBetween: delayBetween,
		activeJobs:   make(map[uuid.UUID]*RewrapJob),
	}
}

// StartRewrapJob initiates a re-wrap job for a tenant after TMK rotation
func (rs *RewrapService) StartRewrapJob(ctx context.Context, tenantID uuid.UUID, oldTMKVersion, newTMKVersion int) (*RewrapJob, error) {
	// Create job record
	job := &RewrapJob{
		JobID:         uuid.New(),
		TenantID:      tenantID,
		OldTMKVersion: oldTMKVersion,
		NewTMKVersion: newTMKVersion,
		StartedAt:     time.Now(),
		Status:        "running",
		Errors:        make([]RewrapError, 0),
	}

	// Count total artifacts to process
	var totalCount int64
	err := rs.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM artifacts WHERE tenant_id = $1 AND encrypted = true`,
		tenantID,
	).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count artifacts: %w", err)
	}

	job.TotalArtifacts = totalCount

	// Store job in active jobs map
	rs.progressMutex.Lock()
	rs.activeJobs[job.JobID] = job
	rs.progressMutex.Unlock()

	// Audit job start
	if rs.auditCallback != nil {
		rs.auditCallback(ctx, RewrapAuditEvent{
			JobID:     job.JobID,
			TenantID:  tenantID,
			Operation: "rewrap_start",
			Timestamp: time.Now(),
		})
	}

	// Start background goroutine to process re-wrap
	go rs.processRewrapJob(context.Background(), job)

	return job, nil
}

// processRewrapJob processes artifacts in batches
func (rs *RewrapService) processRewrapJob(ctx context.Context, job *RewrapJob) {
	defer func() {
		job.CompletedAt = time.Now()
		if job.SuccessCount == job.TotalArtifacts {
			job.Status = "completed"
		} else {
			job.Status = "failed"
		}
	}()

	// Get new TMK
	newTMK, err := rs.tmkService.GetActiveTMK(ctx, job.TenantID)
	if err != nil {
		job.Status = "failed"
		job.Errors = append(job.Errors, RewrapError{
			Error:     fmt.Sprintf("failed to get new TMK: %v", err),
			Timestamp: time.Now(),
			Retryable: true,
		})
		return
	}
	defer zeroBytes(newTMK)

	// Process in batches
	var lastID uuid.UUID
	batchNum := 0

	for {
		batchNum++
		job.CurrentBatch = batchNum

		// Get next batch of artifacts
		rows, err := rs.db.QueryContext(ctx, `
			SELECT artifact_id, repository_id, name, version
			FROM artifacts
			WHERE tenant_id = $1 
			  AND encrypted = true
			  AND artifact_id > $2
			ORDER BY artifact_id
			LIMIT $3
		`, job.TenantID, lastID, rs.batchSize)

		if err != nil {
			job.Errors = append(job.Errors, RewrapError{
				Error:     fmt.Sprintf("failed to query batch %d: %v", batchNum, err),
				Timestamp: time.Now(),
				Retryable: true,
			})
			break
		}

		artifacts := make([]struct {
			ID           uuid.UUID
			RepositoryID uuid.UUID
			Name         string
			Version      string
		}, 0, rs.batchSize)

		for rows.Next() {
			var art struct {
				ID           uuid.UUID
				RepositoryID uuid.UUID
				Name         string
				Version      string
			}
			if err := rows.Scan(&art.ID, &art.RepositoryID, &art.Name, &art.Version); err != nil {
				continue
			}
			artifacts = append(artifacts, art)
		}
		rows.Close()

		if len(artifacts) == 0 {
			break // No more artifacts to process
		}

		// Process each artifact in the batch
		for _, art := range artifacts {
			if err := rs.rewrapArtifact(ctx, job, art.ID, art.RepositoryID, newTMK); err != nil {
				atomic.AddInt64(&job.FailedCount, 1)
				job.Errors = append(job.Errors, RewrapError{
					ArtifactID: art.ID,
					Error:      err.Error(),
					Timestamp:  time.Now(),
					Retryable:  true,
				})
			} else {
				atomic.AddInt64(&job.SuccessCount, 1)
			}

			atomic.AddInt64(&job.ProcessedCount, 1)
			lastID = art.ID
			job.LastProcessedID = art.ID
		}

		// Delay between batches to avoid overwhelming KMS
		time.Sleep(rs.delayBetween)
	}
}

// rewrapArtifact re-wraps a single artifact's DEK
func (rs *RewrapService) rewrapArtifact(ctx context.Context, job *RewrapJob, artifactID, repositoryID uuid.UUID, newTMK []byte) error {
	// Build path to meta.json
	metaPath := filepath.Join(rs.storagePath, job.TenantID.String(), "artifacts", artifactID.String(), "meta.json")

	// Read current metadata
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read meta.json: %w", err)
	}

	var meta ArtifactMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return fmt.Errorf("failed to parse meta.json: %w", err)
	}

	// Skip if already using new TMK version
	if meta.TMKVersion >= job.NewTMKVersion {
		return nil
	}

	// Step 1: Unwrap DEK with old KEK
	oldKEK, err := rs.deriveKEK(job.TenantID, repositoryID, job.OldTMKVersion)
	if err != nil {
		return fmt.Errorf("failed to derive old KEK: %w", err)
	}
	defer zeroBytes(oldKEK)

	dek, err := rs.kmsClient.Decrypt(meta.KEKKeyID, []byte(meta.WrappedDEK))
	if err != nil {
		return fmt.Errorf("failed to unwrap DEK: %w", err)
	}
	defer zeroBytes(dek)

	// Step 2: Re-wrap DEK with new KEK
	newKEK, err := rs.deriveKEKFromTMK(newTMK, repositoryID.String())
	if err != nil {
		return fmt.Errorf("failed to derive new KEK: %w", err)
	}
	defer zeroBytes(newKEK)

	newKEKID := fmt.Sprintf("kek-%s-v%d", repositoryID.String(), job.NewTMKVersion)
	newWrappedDEK, err := rs.kmsClient.Encrypt(newKEKID, dek)
	if err != nil {
		return fmt.Errorf("failed to re-wrap DEK: %w", err)
	}

	// Step 3: Update meta.json atomically
	meta.WrappedDEK = string(newWrappedDEK)
	meta.KEKKeyID = newKEKID
	meta.TMKVersion = job.NewTMKVersion
	meta.LastRewrappedAt = time.Now()

	newMetaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated meta: %w", err)
	}

	// Write to temporary file first (atomic update)
	tempPath := metaPath + ".tmp"
	if err := os.WriteFile(tempPath, newMetaBytes, 0600); err != nil {
		return fmt.Errorf("failed to write temp meta: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, metaPath); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename meta.json: %w", err)
	}

	// Audit success
	if rs.auditCallback != nil {
		rs.auditCallback(ctx, RewrapAuditEvent{
			JobID:      job.JobID,
			TenantID:   job.TenantID,
			ArtifactID: artifactID,
			Operation:  "rewrap_success",
			OldKEKID:   meta.KEKKeyID,
			NewKEKID:   newKEKID,
			Timestamp:  time.Now(),
		})
	}

	return nil
}

// deriveKEK derives KEK from old TMK version (for unwrapping)
func (rs *RewrapService) deriveKEK(tenantID, repositoryID uuid.UUID, tmkVersion int) ([]byte, error) {
	// Get old TMK version from database
	var encryptedKey []byte
	var kmsKeyID string
	err := rs.db.QueryRow(`
		SELECT encrypted_key, kms_key_id
		FROM tenant_master_keys
		WHERE tenant_id = $1 AND key_version = $2
	`, tenantID, tmkVersion).Scan(&encryptedKey, &kmsKeyID)

	if err != nil {
		return nil, fmt.Errorf("failed to get old TMK: %w", err)
	}

	// Decrypt old TMK
	oldTMK, err := rs.kmsClient.Decrypt(kmsKeyID, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt old TMK: %w", err)
	}
	defer zeroBytes(oldTMK)

	// Derive KEK
	return rs.deriveKEKFromTMK(oldTMK, repositoryID.String())
}

// deriveKEKFromTMK derives KEK using HKDF
func (rs *RewrapService) deriveKEKFromTMK(tmk []byte, context string) ([]byte, error) {
	// Use encryption service for key derivation
	encService := &EncryptionService{}
	return encService.DeriveKEK(tmk, context)
}

// GetJobStatus returns the current status of a re-wrap job
func (rs *RewrapService) GetJobStatus(jobID uuid.UUID) (*RewrapJob, error) {
	rs.progressMutex.RLock()
	defer rs.progressMutex.RUnlock()

	job, exists := rs.activeJobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// CancelJob attempts to cancel a running re-wrap job
func (rs *RewrapService) CancelJob(jobID uuid.UUID) error {
	rs.progressMutex.Lock()
	defer rs.progressMutex.Unlock()

	job, exists := rs.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status != "running" {
		return fmt.Errorf("job is not running (status: %s)", job.Status)
	}

	job.Status = "cancelled"
	return nil
}

// SetAuditCallback sets a callback function for audit events
func (rs *RewrapService) SetAuditCallback(callback func(ctx context.Context, event RewrapAuditEvent)) {
	rs.auditCallback = callback
}
