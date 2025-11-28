package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Artifact struct {
	ID                  uuid.UUID              `json:"id"`
	TenantID            uuid.UUID              `json:"tenant_id"`
	Name                string                 `json:"name"`
	Version             string                 `json:"version"`
	Type                string                 `json:"type"`
	RepositoryID        uuid.UUID              `json:"repository_id"`
	Repository          string                 `json:"repository"`
	Size                int64                  `json:"size"`
	SizeFormatted       string                 `json:"size_formatted"`
	Checksum            string                 `json:"checksum"`
	UploadedBy          *string                `json:"uploaded_by,omitempty"`
	UploadedAt          time.Time              `json:"uploaded_at"`
	Downloads           int                    `json:"downloads"`
	License             *string                `json:"license,omitempty"`
	Metadata            map[string]interface{} `json:"metadata"`
	Tags                []string               `json:"tags"`
	Compliance          *ComplianceAudit       `json:"compliance,omitempty"`
	Vulnerabilities     *Vulnerability         `json:"vulnerabilities,omitempty"`
	Indexing            *ArtifactIndexing      `json:"indexing,omitempty"`
	Encrypted           bool                   `json:"encrypted"`
	EncryptionVersion   int                    `json:"encryption_version,omitempty"`
	EncryptedDEK        []byte                 `json:"encrypted_dek,omitempty"`
	EncryptionAlgorithm string                 `json:"encryption_algorithm,omitempty"`
	EncryptionMetadata  map[string]interface{} `json:"encryption_metadata,omitempty"`

	// Integrity and signing fields
	SHA256Hash          string     `json:"sha256_hash,omitempty"`
	SHA512Hash          string     `json:"sha512_hash,omitempty"`
	HashAlgorithm       string     `json:"hash_algorithm,omitempty"`
	SignatureRequired   bool       `json:"signature_required"`
	SignatureVerified   bool       `json:"signature_verified"`
	SignatureVerifiedAt *time.Time `json:"signature_verified_at,omitempty"`
	SignatureCount      int        `json:"signature_count,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ComplianceAudit struct {
	AuditID           uuid.UUID         `json:"audit_id"`
	TenantID          uuid.UUID         `json:"tenant_id"`
	ArtifactID        uuid.UUID         `json:"artifact_id"`
	Status            string            `json:"status"` // compliant, review, non-compliant
	Score             int               `json:"score"`
	Auditor           string            `json:"auditor"`
	LicenseCompliance string            `json:"license_compliance"`
	SecurityScan      string            `json:"security_scan"`
	CodeQuality       string            `json:"code_quality"`
	DataPrivacy       string            `json:"data_privacy"`
	AuditedAt         time.Time         `json:"audited_at"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	LastAudit         string            `json:"last_audit"`
	Checks            map[string]string `json:"checks"`
}

type Vulnerability struct {
	ID         uuid.UUID `json:"id"`
	ArtifactID uuid.UUID `json:"artifact_id"`
	Critical   int       `json:"critical"`
	High       int       `json:"high"`
	Medium     int       `json:"medium"`
	Low        int       `json:"low"`
	Total      int       `json:"total"`
	ScannedAt  time.Time `json:"scanned_at"`
	LastScan   string    `json:"last_scan"`
}

type ArtifactIndexing struct {
	IndexingID    uuid.UUID      `json:"indexing_id"`
	TenantID      uuid.UUID      `json:"tenant_id"`
	ArtifactID    uuid.UUID      `json:"artifact_id"`
	IndexStatus   string         `json:"index_status"` // pending, indexing, completed, failed
	SearchContent *string        `json:"search_content,omitempty"`
	Keywords      pq.StringArray `json:"keywords,omitempty"`
	IndexedAt     *time.Time     `json:"indexed_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Repository struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	RepositoryType string    `json:"repository_type"`
	Description    string    `json:"description"`
	PublicAccess   bool      `json:"public_access"`
	EnableIndexing bool      `json:"enable_indexing"`
	RemoteURL      string    `json:"remote_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ArtifactFilter struct {
	Search           string     `json:"search"`
	Types            []string   `json:"types"`
	Repositories     []string   `json:"repositories"`
	RepositoryID     *uuid.UUID `json:"repository_id"`
	ComplianceStatus []string   `json:"compliance_status"`
	Vulnerabilities  string     `json:"vulnerabilities"`
	DateRange        string     `json:"date_range"`
	Tags             []string   `json:"tags"`
	SortBy           string     `json:"sort_by"`
	SortOrder        string     `json:"sort_order"`
	CreatedAfter     *time.Time `json:"created_after"`
	CreatedBefore    *time.Time `json:"created_before"`
	MinSize          *int64     `json:"min_size"`
	MaxSize          *int64     `json:"max_size"`
	Limit            int        `json:"limit"`
	Offset           int        `json:"offset"`
}

type DeployArtifactRequest struct {
	Name         string                 `json:"name" validate:"required"`
	Version      string                 `json:"version" validate:"required"`
	RepositoryID int64                  `json:"repository_id" validate:"required"`
	Description  string                 `json:"description"`
	Tags         []string               `json:"tags"`
	License      string                 `json:"license"`
	Metadata     map[string]interface{} `json:"metadata"`
	// File will be uploaded via multipart form
}

type DeployArtifactResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Type          string    `json:"type"`
	Repository    string    `json:"repository"`
	Size          int64     `json:"size"`
	SizeFormatted string    `json:"size_formatted"`
	Checksum      string    `json:"checksum"`
	UploadedBy    string    `json:"uploaded_by"`
	UploadedAt    string    `json:"uploaded_at"`
	StorageType   string    `json:"storage_type"`
	ShardCount    int       `json:"shard_count"`
	Message       string    `json:"message"`
}
