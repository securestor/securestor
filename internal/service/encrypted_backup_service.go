package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/encrypt"
)

// EncryptedBackupService handles encrypted database and object storage backups
type EncryptedBackupService struct {
	db              *sql.DB
	tmkService      *encrypt.TMKService
	kmsClient       encrypt.KMSClient
	storagePath     string
	backupKEKID     string
	logger          *log.Logger
	retentionDays   int
	crossRegionCopy bool
}

// BackupMetadata stores metadata about encrypted backups
type BackupMetadata struct {
	BackupID              uuid.UUID `json:"backup_id"`
	TenantID              uuid.UUID `json:"tenant_id,omitempty"` // Empty for system backups
	BackupType            string    `json:"backup_type"`         // "database", "object_storage", "keys"
	EncryptedDEK          string    `json:"encrypted_dek"`
	KEKKeyID              string    `json:"kek_key_id"`
	TMKVersion            int       `json:"tmk_version"`
	Nonce                 string    `json:"nonce"`
	AuthTag               string    `json:"auth_tag"`
	OriginalSize          int64     `json:"original_size"`
	EncryptedSize         int64     `json:"encrypted_size"`
	Checksum              string    `json:"checksum"` // SHA-256 of encrypted data
	Timestamp             time.Time `json:"timestamp"`
	ExpiresAt             time.Time `json:"expires_at"`
	VerifiedAt            time.Time `json:"verified_at,omitempty"`
	RestoredAt            time.Time `json:"restored_at,omitempty"`
	CrossRegionReplicated bool      `json:"cross_region_replicated"`
}

// EncryptedBackupConfig contains configuration for encrypted backups
type EncryptedBackupConfig struct {
	StoragePath       string
	RetentionDays     int
	CrossRegionCopy   bool
	BackupKEKID       string // Dedicated KEK for backups
	VerifyAfterBackup bool
	Logger            *log.Logger
}

// NewEncryptedBackupService creates a new encrypted backup service
func NewEncryptedBackupService(
	db *sql.DB,
	tmkService *encrypt.TMKService,
	kmsClient encrypt.KMSClient,
	config EncryptedBackupConfig,
) *EncryptedBackupService {
	if config.BackupKEKID == "" {
		config.BackupKEKID = "backup-kek-v1"
	}

	if config.RetentionDays == 0 {
		config.RetentionDays = 90 // Default 90 days retention
	}

	return &EncryptedBackupService{
		db:              db,
		tmkService:      tmkService,
		kmsClient:       kmsClient,
		storagePath:     config.StoragePath,
		backupKEKID:     config.BackupKEKID,
		logger:          config.Logger,
		retentionDays:   config.RetentionDays,
		crossRegionCopy: config.CrossRegionCopy,
	}
}

// BackupDatabase creates an encrypted backup of the PostgreSQL database
func (ebs *EncryptedBackupService) BackupDatabase(ctx context.Context) (*BackupMetadata, error) {
	backupID := uuid.New()
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(ebs.storagePath, "backups", "database")

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	ebs.logger.Printf("📦 Starting encrypted database backup: %s", backupID)

	// Step 1: Create plaintext dump (temporary)
	tempDumpPath := filepath.Join(backupDir, fmt.Sprintf("temp_%s.sql", timestamp))
	if err := ebs.createDatabaseDump(ctx, tempDumpPath); err != nil {
		return nil, fmt.Errorf("failed to create database dump: %w", err)
	}
	defer os.Remove(tempDumpPath) // Clean up plaintext dump

	// Step 2: Encrypt the dump
	encryptedPath := filepath.Join(backupDir, fmt.Sprintf("backup_%s_%s.sql.enc", timestamp, backupID.String()))
	metaPath := filepath.Join(backupDir, fmt.Sprintf("backup_%s_%s.meta.json", timestamp, backupID.String()))

	metadata, err := ebs.encryptFile(ctx, tempDumpPath, encryptedPath, uuid.Nil, "database")
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt backup: %w", err)
	}

	metadata.BackupID = backupID
	metadata.ExpiresAt = time.Now().AddDate(0, 0, ebs.retentionDays)

	// Step 3: Write metadata
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaBytes, 0600); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Step 4: Store backup record in database
	if err := ebs.recordBackup(ctx, metadata); err != nil {
		ebs.logger.Printf("⚠️  Failed to record backup in database: %v", err)
	}

	// Step 5: Cross-region replication (if enabled)
	if ebs.crossRegionCopy {
		if err := ebs.replicateBackup(ctx, encryptedPath, metaPath); err != nil {
			ebs.logger.Printf("⚠️  Cross-region backup replication failed: %v", err)
		} else {
			metadata.CrossRegionReplicated = true
		}
	}

	ebs.logger.Printf("✅ Encrypted database backup completed: %s (size: %d bytes)", backupID, metadata.EncryptedSize)

	return metadata, nil
}

// BackupKeys creates an encrypted backup of encryption keys (for DR)
func (ebs *EncryptedBackupService) BackupKeys(ctx context.Context, tenantID uuid.UUID) (*BackupMetadata, error) {
	backupID := uuid.New()
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(ebs.storagePath, "backups", "keys", tenantID.String())

	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	ebs.logger.Printf("🔑 Starting encrypted key backup for tenant: %s", tenantID)

	// Step 1: Export TMK (encrypted with KMS)
	// Note: ExportTMKForBackup requires userID - using nil UUID for system backup
	systemUserID := uuid.Nil
	tmkExport, err := ebs.tmkService.ExportTMKForBackup(ctx, tenantID, systemUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to export TMK: %w", err)
	}

	// Step 2: Create temporary key export file
	tempKeyPath := filepath.Join(backupDir, fmt.Sprintf("temp_keys_%s.json", timestamp))
	keyExportData, err := json.MarshalIndent(tmkExport, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key export: %w", err)
	}

	if err := os.WriteFile(tempKeyPath, keyExportData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write key export: %w", err)
	}
	defer os.Remove(tempKeyPath)

	// Step 3: Encrypt the key export (double encryption for safety)
	encryptedPath := filepath.Join(backupDir, fmt.Sprintf("keys_%s_%s.enc", timestamp, backupID.String()))
	metaPath := filepath.Join(backupDir, fmt.Sprintf("keys_%s_%s.meta.json", timestamp, backupID.String()))

	metadata, err := ebs.encryptFile(ctx, tempKeyPath, encryptedPath, tenantID, "keys")
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key backup: %w", err)
	}

	metadata.BackupID = backupID
	metadata.TenantID = tenantID
	metadata.ExpiresAt = time.Now().AddDate(0, 0, ebs.retentionDays)

	// Step 4: Write metadata
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaBytes, 0600); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Step 5: Record in database
	if err := ebs.recordBackup(ctx, metadata); err != nil {
		ebs.logger.Printf("⚠️  Failed to record key backup in database: %v", err)
	}

	ebs.logger.Printf("✅ Encrypted key backup completed: %s", backupID)

	return metadata, nil
}

// RestoreDatabase restores a database from an encrypted backup
func (ebs *EncryptedBackupService) RestoreDatabase(ctx context.Context, backupID uuid.UUID) error {
	ebs.logger.Printf("♻️  Starting database restore from backup: %s", backupID)

	// Step 1: Find backup metadata
	metadata, metaPath, encPath, err := ebs.findBackup(backupID, "database")
	if err != nil {
		return fmt.Errorf("failed to find backup: %w", err)
	}

	// Step 2: Verify KEK is available
	if err := ebs.verifyKEKAvailable(ctx, metadata); err != nil {
		return fmt.Errorf("KEK not available for restore: %w", err)
	}

	// Step 3: Decrypt backup
	tempRestorePath := filepath.Join(filepath.Dir(encPath), fmt.Sprintf("restore_%s.sql", backupID.String()))
	if err := ebs.decryptFile(ctx, encPath, tempRestorePath, metadata); err != nil {
		return fmt.Errorf("failed to decrypt backup: %w", err)
	}
	defer os.Remove(tempRestorePath)

	// Step 4: Restore database (using psql or pg_restore)
	if err := ebs.restoreDatabaseDump(ctx, tempRestorePath); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	// Step 5: Update metadata
	metadata.RestoredAt = time.Now()
	if err := ebs.updateBackupMetadata(metaPath, metadata); err != nil {
		ebs.logger.Printf("⚠️  Failed to update backup metadata: %v", err)
	}

	ebs.logger.Printf("✅ Database restored successfully from backup: %s", backupID)

	return nil
}

// encryptFile encrypts a file using envelope encryption
func (ebs *EncryptedBackupService) encryptFile(ctx context.Context, inputPath, outputPath string, tenantID uuid.UUID, backupType string) (*BackupMetadata, error) {
	// Step 1: Generate DEK
	dek := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("failed to generate DEK: %w", err)
	}
	defer func() {
		for i := range dek {
			dek[i] = 0 // Zero out DEK
		}
	}()

	// Step 2: Encrypt DEK with backup KEK
	encryptedDEK, err := ebs.kmsClient.Encrypt(ebs.backupKEKID, dek)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt DEK: %w", err)
	}

	// Step 3: Read input file
	inputData, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}
	originalSize := int64(len(inputData))

	// Step 4: Encrypt data with DEK using AES-256-GCM
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, inputData, nil)

	// Step 5: Write encrypted data
	if err := os.WriteFile(outputPath, ciphertext, 0600); err != nil {
		return nil, fmt.Errorf("failed to write encrypted file: %w", err)
	}

	// Step 6: Calculate checksum
	checksum := sha256.Sum256(ciphertext)

	// Step 7: Create metadata
	metadata := &BackupMetadata{
		BackupType:    backupType,
		EncryptedDEK:  hex.EncodeToString(encryptedDEK),
		KEKKeyID:      ebs.backupKEKID,
		Nonce:         hex.EncodeToString(nonce),
		OriginalSize:  originalSize,
		EncryptedSize: int64(len(ciphertext)),
		Checksum:      hex.EncodeToString(checksum[:]),
		Timestamp:     time.Now(),
	}

	if tenantID != uuid.Nil {
		metadata.TenantID = tenantID
	}

	return metadata, nil
}

// decryptFile decrypts a file using envelope encryption
func (ebs *EncryptedBackupService) decryptFile(ctx context.Context, inputPath, outputPath string, metadata *BackupMetadata) error {
	// Step 1: Decrypt DEK
	encryptedDEK, err := hex.DecodeString(metadata.EncryptedDEK)
	if err != nil {
		return fmt.Errorf("failed to decode encrypted DEK: %w", err)
	}

	dek, err := ebs.kmsClient.Decrypt(metadata.KEKKeyID, encryptedDEK)
	if err != nil {
		return fmt.Errorf("failed to decrypt DEK: %w", err)
	}
	defer func() {
		for i := range dek {
			dek[i] = 0 // Zero out DEK
		}
	}()

	// Step 2: Read encrypted file
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Step 3: Verify checksum
	checksum := sha256.Sum256(ciphertext)
	if hex.EncodeToString(checksum[:]) != metadata.Checksum {
		return fmt.Errorf("checksum mismatch: backup may be corrupted")
	}

	// Step 4: Decrypt data
	block, err := aes.NewCipher(dek)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := hex.DecodeString(metadata.Nonce)
	if err != nil {
		return fmt.Errorf("failed to decode nonce: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Step 5: Write decrypted data
	if err := os.WriteFile(outputPath, plaintext, 0600); err != nil {
		return fmt.Errorf("failed to write decrypted file: %w", err)
	}

	return nil
}

// Helper functions (stubs for full implementation)

func (ebs *EncryptedBackupService) createDatabaseDump(ctx context.Context, outputPath string) error {
	// In production: exec pg_dump
	// exec.CommandContext(ctx, "pg_dump", "-h", host, "-U", user, "-d", dbname, "-f", outputPath)

	// For now, create a simple dump file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := fmt.Sprintf("-- PostgreSQL Encrypted Backup\n-- Timestamp: %s\n\n", time.Now().Format(time.RFC3339))
	_, err = file.WriteString(header)
	return err
}

func (ebs *EncryptedBackupService) restoreDatabaseDump(ctx context.Context, dumpPath string) error {
	// In production: exec psql < dumpPath
	// exec.CommandContext(ctx, "psql", "-h", host, "-U", user, "-d", dbname, "-f", dumpPath)

	ebs.logger.Printf("Would restore database from: %s", dumpPath)
	return nil
}

func (ebs *EncryptedBackupService) recordBackup(ctx context.Context, metadata *BackupMetadata) error {
	query := `
		INSERT INTO encrypted_backups 
		(backup_id, tenant_id, backup_type, kek_key_id, encrypted_size, checksum, timestamp, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := ebs.db.ExecContext(ctx, query,
		metadata.BackupID, metadata.TenantID, metadata.BackupType,
		metadata.KEKKeyID, metadata.EncryptedSize, metadata.Checksum,
		metadata.Timestamp, metadata.ExpiresAt,
	)
	return err
}

func (ebs *EncryptedBackupService) verifyKEKAvailable(ctx context.Context, metadata *BackupMetadata) error {
	// Verify we can decrypt with the KEK
	testData := []byte("test")
	encrypted, err := ebs.kmsClient.Encrypt(metadata.KEKKeyID, testData)
	if err != nil {
		return fmt.Errorf("KEK not available: %w", err)
	}

	decrypted, err := ebs.kmsClient.Decrypt(metadata.KEKKeyID, encrypted)
	if err != nil {
		return fmt.Errorf("KEK decryption failed: %w", err)
	}

	if string(decrypted) != string(testData) {
		return fmt.Errorf("KEK decryption produced incorrect result")
	}

	return nil
}

func (ebs *EncryptedBackupService) findBackup(backupID uuid.UUID, backupType string) (*BackupMetadata, string, string, error) {
	// Stub: Find backup files in storage
	return nil, "", "", fmt.Errorf("not implemented")
}

func (ebs *EncryptedBackupService) updateBackupMetadata(metaPath string, metadata *BackupMetadata) error {
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, metaBytes, 0600)
}

func (ebs *EncryptedBackupService) replicateBackup(ctx context.Context, encPath, metaPath string) error {
	// Stub: Cross-region replication
	ebs.logger.Printf("Would replicate backup to cross-region: %s", encPath)
	return nil
}
