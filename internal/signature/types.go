package signature

import (
	"time"

	"github.com/google/uuid"
)

// SignatureType represents the type of signature
type SignatureType string

const (
	SignatureTypeCosign   SignatureType = "cosign"
	SignatureTypePGP      SignatureType = "pgp"
	SignatureTypeSigstore SignatureType = "sigstore"
	SignatureTypeX509     SignatureType = "x509"
	SignatureTypeSSH      SignatureType = "ssh"
)

// SignatureFormat represents the format of signature data
type SignatureFormat string

const (
	SignatureFormatBinary     SignatureFormat = "binary"
	SignatureFormatASCIIArmor SignatureFormat = "ascii-armor"
	SignatureFormatJSON       SignatureFormat = "json"
	SignatureFormatPEM        SignatureFormat = "pem"
	SignatureFormatDER        SignatureFormat = "der"
)

// VerificationStatus represents the verification status
type VerificationStatus string

const (
	VerificationStatusPending   VerificationStatus = "pending"
	VerificationStatusValid     VerificationStatus = "valid"
	VerificationStatusInvalid   VerificationStatus = "invalid"
	VerificationStatusExpired   VerificationStatus = "expired"
	VerificationStatusRevoked   VerificationStatus = "revoked"
	VerificationStatusUntrusted VerificationStatus = "untrusted"
)

// SignaturePolicy represents repository signature enforcement policy
type SignaturePolicy string

const (
	SignaturePolicyDisabled SignaturePolicy = "disabled" // No signature checking
	SignaturePolicyOptional SignaturePolicy = "optional" // Signatures accepted but not required
	SignaturePolicyRequired SignaturePolicy = "required" // Signature required but trust not enforced
	SignaturePolicyStrict   SignaturePolicy = "strict"   // Signature required and must be from trusted signer
)

// ArtifactSignature represents a signature for an artifact
type ArtifactSignature struct {
	SignatureID        uuid.UUID          `json:"signature_id" db:"signature_id"`
	TenantID           uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	ArtifactID         uuid.UUID          `json:"artifact_id" db:"artifact_id"`
	RepositoryID       uuid.UUID          `json:"repository_id" db:"repository_id"`
	SignatureType      SignatureType      `json:"signature_type" db:"signature_type"`
	SignatureFormat    SignatureFormat    `json:"signature_format" db:"signature_format"`
	SignatureData      []byte             `json:"signature_data" db:"signature_data"`
	SignatureAlgo      string             `json:"signature_algorithm,omitempty" db:"signature_algorithm"`
	SignerIdentity     string             `json:"signer_identity,omitempty" db:"signer_identity"`
	SignerFingerprint  string             `json:"signer_fingerprint,omitempty" db:"signer_fingerprint"`
	PublicKey          string             `json:"public_key,omitempty" db:"public_key"`
	PublicKeyURL       string             `json:"public_key_url,omitempty" db:"public_key_url"`
	Verified           bool               `json:"verified" db:"verified"`
	VerificationMethod string             `json:"verification_method,omitempty" db:"verification_method"`
	VerificationStatus VerificationStatus `json:"verification_status" db:"verification_status"`
	VerificationError  string             `json:"verification_error,omitempty" db:"verification_error"`
	VerifiedAt         *time.Time         `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedBy         *uuid.UUID         `json:"verified_by,omitempty" db:"verified_by"`

	// Cosign-specific fields
	CosignBundle          map[string]interface{} `json:"cosign_bundle,omitempty" db:"cosign_bundle"`
	CosignCertificate     string                 `json:"cosign_certificate,omitempty" db:"cosign_certificate"`
	CosignSignatureDigest string                 `json:"cosign_signature_digest,omitempty" db:"cosign_signature_digest"`
	RekorLogIndex         *int64                 `json:"rekor_log_index,omitempty" db:"rekor_log_index"`
	RekorUUID             string                 `json:"rekor_uuid,omitempty" db:"rekor_uuid"`

	// PGP-specific fields
	PGPKeyID            string `json:"pgp_key_id,omitempty" db:"pgp_key_id"`
	PGPKeyFingerprint   string `json:"pgp_key_fingerprint,omitempty" db:"pgp_key_fingerprint"`
	PGPSignatureVersion *int   `json:"pgp_signature_version,omitempty" db:"pgp_signature_version"`

	// Sigstore attestation fields
	SigstoreBundle        map[string]interface{} `json:"sigstore_bundle,omitempty" db:"sigstore_bundle"`
	SigstorePredicateType string                 `json:"sigstore_predicate_type,omitempty" db:"sigstore_predicate_type"`
	AttestationPayload    map[string]interface{} `json:"attestation_payload,omitempty" db:"attestation_payload"`

	// Storage paths
	SignatureStoragePath   string `json:"signature_storage_path,omitempty" db:"signature_storage_path"`
	CertificateStoragePath string `json:"certificate_storage_path,omitempty" db:"certificate_storage_path"`

	// Timestamps
	SignedAt   *time.Time `json:"signed_at,omitempty" db:"signed_at"`
	UploadedAt time.Time  `json:"uploaded_at" db:"uploaded_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// PublicKey represents a trusted public key
type PublicKey struct {
	KeyID            uuid.UUID  `json:"key_id" db:"key_id"`
	TenantID         uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	RepositoryID     *uuid.UUID `json:"repository_id,omitempty" db:"repository_id"`
	KeyName          string     `json:"key_name" db:"key_name"`
	KeyType          string     `json:"key_type" db:"key_type"`
	KeyFormat        string     `json:"key_format" db:"key_format"`
	PublicKey        string     `json:"public_key" db:"public_key"`
	KeyFingerprint   string     `json:"key_fingerprint" db:"key_fingerprint"`
	KeyIDShort       string     `json:"key_id_short,omitempty" db:"key_id_short"`
	KeyAlgorithm     string     `json:"key_algorithm,omitempty" db:"key_algorithm"`
	KeySize          *int       `json:"key_size,omitempty" db:"key_size"`
	OwnerEmail       string     `json:"owner_email,omitempty" db:"owner_email"`
	OwnerName        string     `json:"owner_name,omitempty" db:"owner_name"`
	Organization     string     `json:"organization,omitempty" db:"organization"`
	Trusted          bool       `json:"trusted" db:"trusted"`
	Enabled          bool       `json:"enabled" db:"enabled"`
	Revoked          bool       `json:"revoked" db:"revoked"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevocationReason string     `json:"revocation_reason,omitempty" db:"revocation_reason"`
	ValidFrom        *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidUntil       *time.Time `json:"valid_until,omitempty" db:"valid_until"`
	Description      string     `json:"description,omitempty" db:"description"`
	KeySource        string     `json:"key_source,omitempty" db:"key_source"`
	KeySourceURL     string     `json:"key_source_url,omitempty" db:"key_source_url"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// VerificationLog represents a signature verification log entry
type VerificationLog struct {
	LogID              uuid.UUID  `json:"log_id" db:"log_id"`
	TenantID           uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	ArtifactID         uuid.UUID  `json:"artifact_id" db:"artifact_id"`
	SignatureID        *uuid.UUID `json:"signature_id,omitempty" db:"signature_id"`
	VerificationType   string     `json:"verification_type" db:"verification_type"`
	VerificationResult string     `json:"verification_result" db:"verification_result"`
	VerificationStatus string     `json:"verification_status" db:"verification_status"`
	VerificationMethod string     `json:"verification_method,omitempty" db:"verification_method"`
	ErrorMessage       string     `json:"error_message,omitempty" db:"error_message"`
	ErrorCode          string     `json:"error_code,omitempty" db:"error_code"`
	VerifiedBy         *uuid.UUID `json:"verified_by,omitempty" db:"verified_by"`
	ClientIP           string     `json:"client_ip,omitempty" db:"client_ip"`
	UserAgent          string     `json:"user_agent,omitempty" db:"user_agent"`
	VerifiedAt         time.Time  `json:"verified_at" db:"verified_at"`
}

// VerificationResult represents the result of signature verification
type VerificationResult struct {
	Verified           bool               `json:"verified"`
	Status             VerificationStatus `json:"status"`
	SignerIdentity     string             `json:"signer_identity,omitempty"`
	SignerFingerprint  string             `json:"signer_fingerprint,omitempty"`
	SignatureAlgorithm string             `json:"signature_algorithm,omitempty"`
	VerificationMethod string             `json:"verification_method,omitempty"`
	TrustedSigner      bool               `json:"trusted_signer"`
	ErrorMessage       string             `json:"error_message,omitempty"`
	VerifiedAt         time.Time          `json:"verified_at"`

	// Transparency log information
	RekorLogIndex *int64                 `json:"rekor_log_index,omitempty"`
	RekorUUID     string                 `json:"rekor_uuid,omitempty"`
	LogEntry      map[string]interface{} `json:"log_entry,omitempty"`

	// Certificate information
	CertificateSubject string     `json:"certificate_subject,omitempty"`
	CertificateIssuer  string     `json:"certificate_issuer,omitempty"`
	CertificateExpiry  *time.Time `json:"certificate_expiry,omitempty"`
}

// UploadSignatureRequest represents request to upload a signature
type UploadSignatureRequest struct {
	ArtifactID        uuid.UUID              `json:"artifact_id" validate:"required"`
	SignatureType     SignatureType          `json:"signature_type" validate:"required,oneof=cosign pgp sigstore x509 ssh"`
	SignatureFormat   SignatureFormat        `json:"signature_format" validate:"required"`
	SignatureData     []byte                 `json:"signature_data" validate:"required"`
	SignatureAlgo     string                 `json:"signature_algorithm,omitempty"`
	PublicKey         string                 `json:"public_key,omitempty"`
	Certificate       string                 `json:"certificate,omitempty"`
	CosignBundle      map[string]interface{} `json:"cosign_bundle,omitempty"`
	SigstoreBundle    map[string]interface{} `json:"sigstore_bundle,omitempty"`
	VerifyImmediately bool                   `json:"verify_immediately"`
}

// VerifySignatureRequest represents request to verify a signature
type VerifySignatureRequest struct {
	SignatureID uuid.UUID `json:"signature_id" validate:"required"`
	TrustedKeys []string  `json:"trusted_keys,omitempty"` // Override trusted keys for verification
}

// IntegrityHash represents computed hashes for an artifact
type IntegrityHash struct {
	SHA256    string `json:"sha256"`
	SHA512    string `json:"sha512,omitempty"`
	Algorithm string `json:"algorithm"`
}
