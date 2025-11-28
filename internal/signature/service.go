package signature

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles signature operations
type Service struct {
	db *sql.DB
}

// NewService creates a new signature service
func NewService(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

// StoreSignature stores a signature for an artifact
func (s *Service) StoreSignature(ctx context.Context, sig *ArtifactSignature) error {
	query := `
		INSERT INTO artifact_signatures (
			signature_id, tenant_id, artifact_id, repository_id,
			signature_type, signature_format, signature_data, signature_algorithm,
			signer_identity, signer_fingerprint, public_key, public_key_url,
			verified, verification_method, verification_status, verification_error,
			verified_at, verified_by,
			cosign_bundle, cosign_certificate, cosign_signature_digest, rekor_log_index, rekor_uuid,
			pgp_key_id, pgp_key_fingerprint, pgp_signature_version,
			sigstore_bundle, sigstore_predicate_type, attestation_payload,
			signature_storage_path, certificate_storage_path,
			signed_at, uploaded_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34
		)
		RETURNING signature_id, created_at, updated_at
	`

	cosignBundleJSON, _ := json.Marshal(sig.CosignBundle)
	sigstoreBundleJSON, _ := json.Marshal(sig.SigstoreBundle)
	attestationPayloadJSON, _ := json.Marshal(sig.AttestationPayload)

	err := s.db.QueryRowContext(ctx, query,
		sig.SignatureID, sig.TenantID, sig.ArtifactID, sig.RepositoryID,
		sig.SignatureType, sig.SignatureFormat, sig.SignatureData, sig.SignatureAlgo,
		sig.SignerIdentity, sig.SignerFingerprint, sig.PublicKey, sig.PublicKeyURL,
		sig.Verified, sig.VerificationMethod, sig.VerificationStatus, sig.VerificationError,
		sig.VerifiedAt, sig.VerifiedBy,
		cosignBundleJSON, sig.CosignCertificate, sig.CosignSignatureDigest, sig.RekorLogIndex, sig.RekorUUID,
		sig.PGPKeyID, sig.PGPKeyFingerprint, sig.PGPSignatureVersion,
		sigstoreBundleJSON, sig.SigstorePredicateType, attestationPayloadJSON,
		sig.SignatureStoragePath, sig.CertificateStoragePath,
		sig.SignedAt, time.Now(), sig.ExpiresAt,
	).Scan(&sig.SignatureID, &sig.CreatedAt, &sig.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store signature: %w", err)
	}

	return nil
}

// GetSignature retrieves a signature by ID
func (s *Service) GetSignature(ctx context.Context, tenantID, signatureID uuid.UUID) (*ArtifactSignature, error) {
	query := `
		SELECT 
			signature_id, tenant_id, artifact_id, repository_id,
			signature_type, signature_format, signature_data, signature_algorithm,
			signer_identity, signer_fingerprint, public_key, public_key_url,
			verified, verification_method, verification_status, verification_error,
			verified_at, verified_by,
			cosign_bundle, cosign_certificate, cosign_signature_digest, rekor_log_index, rekor_uuid,
			pgp_key_id, pgp_key_fingerprint, pgp_signature_version,
			sigstore_bundle, sigstore_predicate_type, attestation_payload,
			signature_storage_path, certificate_storage_path,
			signed_at, uploaded_at, expires_at, created_at, updated_at
		FROM artifact_signatures
		WHERE tenant_id = $1 AND signature_id = $2
	`

	var sig ArtifactSignature
	var cosignBundleJSON, sigstoreBundleJSON, attestationPayloadJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, tenantID, signatureID).Scan(
		&sig.SignatureID, &sig.TenantID, &sig.ArtifactID, &sig.RepositoryID,
		&sig.SignatureType, &sig.SignatureFormat, &sig.SignatureData, &sig.SignatureAlgo,
		&sig.SignerIdentity, &sig.SignerFingerprint, &sig.PublicKey, &sig.PublicKeyURL,
		&sig.Verified, &sig.VerificationMethod, &sig.VerificationStatus, &sig.VerificationError,
		&sig.VerifiedAt, &sig.VerifiedBy,
		&cosignBundleJSON, &sig.CosignCertificate, &sig.CosignSignatureDigest, &sig.RekorLogIndex, &sig.RekorUUID,
		&sig.PGPKeyID, &sig.PGPKeyFingerprint, &sig.PGPSignatureVersion,
		&sigstoreBundleJSON, &sig.SigstorePredicateType, &attestationPayloadJSON,
		&sig.SignatureStoragePath, &sig.CertificateStoragePath,
		&sig.SignedAt, &sig.UploadedAt, &sig.ExpiresAt, &sig.CreatedAt, &sig.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("signature not found")
		}
		return nil, fmt.Errorf("failed to get signature: %w", err)
	}

	// Parse JSON fields if present
	if cosignBundleJSON.Valid && cosignBundleJSON.String != "" {
		json.Unmarshal([]byte(cosignBundleJSON.String), &sig.CosignBundle)
	}
	if sigstoreBundleJSON.Valid && sigstoreBundleJSON.String != "" {
		json.Unmarshal([]byte(sigstoreBundleJSON.String), &sig.SigstoreBundle)
	}
	if attestationPayloadJSON.Valid && attestationPayloadJSON.String != "" {
		json.Unmarshal([]byte(attestationPayloadJSON.String), &sig.AttestationPayload)
	}

	return &sig, nil
}

// GetArtifactSignatures retrieves all signatures for an artifact
func (s *Service) GetArtifactSignatures(ctx context.Context, tenantID, artifactID uuid.UUID) ([]ArtifactSignature, error) {
	query := `
		SELECT 
			signature_id, tenant_id, artifact_id, repository_id,
			signature_type, signature_format, signature_data, signature_algorithm,
			signer_identity, signer_fingerprint, public_key, public_key_url,
			verified, verification_method, verification_status, verification_error,
			verified_at, verified_by,
			cosign_bundle, cosign_certificate, cosign_signature_digest, rekor_log_index, rekor_uuid,
			pgp_key_id, pgp_key_fingerprint, pgp_signature_version,
			sigstore_bundle, sigstore_predicate_type, attestation_payload,
			signature_storage_path, certificate_storage_path,
			signed_at, uploaded_at, expires_at, created_at, updated_at
		FROM artifact_signatures
		WHERE tenant_id = $1 AND artifact_id = $2
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to query signatures: %w", err)
	}
	defer rows.Close()

	var signatures []ArtifactSignature
	for rows.Next() {
		var sig ArtifactSignature
		var cosignBundleJSON, sigstoreBundleJSON, attestationPayloadJSON sql.NullString

		err := rows.Scan(
			&sig.SignatureID, &sig.TenantID, &sig.ArtifactID, &sig.RepositoryID,
			&sig.SignatureType, &sig.SignatureFormat, &sig.SignatureData, &sig.SignatureAlgo,
			&sig.SignerIdentity, &sig.SignerFingerprint, &sig.PublicKey, &sig.PublicKeyURL,
			&sig.Verified, &sig.VerificationMethod, &sig.VerificationStatus, &sig.VerificationError,
			&sig.VerifiedAt, &sig.VerifiedBy,
			&cosignBundleJSON, &sig.CosignCertificate, &sig.CosignSignatureDigest, &sig.RekorLogIndex, &sig.RekorUUID,
			&sig.PGPKeyID, &sig.PGPKeyFingerprint, &sig.PGPSignatureVersion,
			&sigstoreBundleJSON, &sig.SigstorePredicateType, &attestationPayloadJSON,
			&sig.SignatureStoragePath, &sig.CertificateStoragePath,
			&sig.SignedAt, &sig.UploadedAt, &sig.ExpiresAt, &sig.CreatedAt, &sig.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signature: %w", err)
		}

		// Parse JSON fields if present
		if cosignBundleJSON.Valid && cosignBundleJSON.String != "" {
			json.Unmarshal([]byte(cosignBundleJSON.String), &sig.CosignBundle)
		}
		if sigstoreBundleJSON.Valid && sigstoreBundleJSON.String != "" {
			json.Unmarshal([]byte(sigstoreBundleJSON.String), &sig.SigstoreBundle)
		}
		if attestationPayloadJSON.Valid && attestationPayloadJSON.String != "" {
			json.Unmarshal([]byte(attestationPayloadJSON.String), &sig.AttestationPayload)
		}

		signatures = append(signatures, sig)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating signatures: %w", err)
	}

	return signatures, nil
}

// VerifySignature verifies a signature against an artifact
func (s *Service) UpdateSignatureVerification(ctx context.Context, signatureID uuid.UUID, result *VerificationResult, verifiedBy *uuid.UUID) error {
	query := `
		UPDATE artifact_signatures
		SET 
			verified = $2,
			verification_status = $3,
			verification_method = $4,
			verification_error = $5,
			verified_at = $6,
			verified_by = $7,
			signer_identity = COALESCE(NULLIF($8, ''), signer_identity),
			signer_fingerprint = COALESCE(NULLIF($9, ''), signer_fingerprint),
			updated_at = CURRENT_TIMESTAMP
		WHERE signature_id = $1
	`

	errorMsg := ""
	if result.ErrorMessage != "" {
		errorMsg = result.ErrorMessage
	}

	_, err := s.db.ExecContext(ctx, query,
		signatureID,
		result.Verified,
		result.Status,
		result.VerificationMethod,
		errorMsg,
		result.VerifiedAt,
		verifiedBy,
		result.SignerIdentity,
		result.SignerFingerprint,
	)

	if err != nil {
		return fmt.Errorf("failed to update signature verification: %w", err)
	}

	return nil
}

// LogVerification logs a signature verification attempt
func (s *Service) LogVerification(ctx context.Context, log *VerificationLog) error {
	query := `
		INSERT INTO signature_verification_logs (
			log_id, tenant_id, artifact_id, signature_id,
			verification_type, verification_result, verification_status, verification_method,
			error_message, error_code, verified_by, client_ip, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	logID := uuid.New()
	_, err := s.db.ExecContext(ctx, query,
		logID, log.TenantID, log.ArtifactID, log.SignatureID,
		log.VerificationType, log.VerificationResult, log.VerificationStatus, log.VerificationMethod,
		log.ErrorMessage, log.ErrorCode, log.VerifiedBy, log.ClientIP, log.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to log verification: %w", err)
	}

	return nil
}

// GetRepositorySignaturePolicy retrieves the signature policy for a repository
func (s *Service) GetRepositorySignaturePolicy(ctx context.Context, tenantID, repositoryID uuid.UUID) (*RepositorySignaturePolicy, error) {
	query := `
		SELECT 
			repository_id, tenant_id,
			signature_policy, signature_verification_enabled,
			cosign_enabled, pgp_enabled, sigstore_enabled,
			allowed_signers
		FROM repositories
		WHERE tenant_id = $1 AND repository_id = $2
	`

	var policy RepositorySignaturePolicy
	var allowedSignersJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, tenantID, repositoryID).Scan(
		&policy.RepositoryID, &policy.TenantID,
		&policy.SignaturePolicy, &policy.SignatureVerificationEnabled,
		&policy.CosignEnabled, &policy.PGPEnabled, &policy.SigstoreEnabled,
		&allowedSignersJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository signature policy: %w", err)
	}

	if allowedSignersJSON.Valid && allowedSignersJSON.String != "" {
		json.Unmarshal([]byte(allowedSignersJSON.String), &policy.AllowedSigners)
	}

	return &policy, nil
}

// UpdateRepositorySignaturePolicy updates signature policy for a repository
func (s *Service) UpdateRepositorySignaturePolicy(ctx context.Context, policy *RepositorySignaturePolicy) error {
	query := `
		UPDATE repositories
		SET 
			signature_policy = $3,
			signature_verification_enabled = $4,
			cosign_enabled = $5,
			pgp_enabled = $6,
			sigstore_enabled = $7,
			allowed_signers = $8,
			updated_at = CURRENT_TIMESTAMP
		WHERE tenant_id = $1 AND repository_id = $2
	`

	_, err := s.db.ExecContext(ctx, query,
		policy.TenantID, policy.RepositoryID,
		policy.SignaturePolicy, policy.SignatureVerificationEnabled,
		policy.CosignEnabled, policy.PGPEnabled, policy.SigstoreEnabled,
		policy.AllowedSigners,
	)

	if err != nil {
		return fmt.Errorf("failed to update repository signature policy: %w", err)
	}

	return nil
}

// StorePublicKey stores a trusted public key
func (s *Service) StorePublicKey(ctx context.Context, key *PublicKey) error {
	query := `
		INSERT INTO public_keys (
			key_id, tenant_id, repository_id, key_name, key_type, key_format, public_key,
			key_fingerprint, key_id_short, key_algorithm, key_size,
			owner_email, owner_name, organization,
			trusted, enabled, valid_from, valid_until,
			description, key_source, key_source_url, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
		ON CONFLICT (tenant_id, key_fingerprint) 
		DO UPDATE SET
			key_name = EXCLUDED.key_name,
			trusted = EXCLUDED.trusted,
			enabled = EXCLUDED.enabled,
			updated_at = CURRENT_TIMESTAMP
		RETURNING key_id, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query,
		key.KeyID, key.TenantID, key.RepositoryID, key.KeyName, key.KeyType, key.KeyFormat, key.PublicKey,
		key.KeyFingerprint, key.KeyIDShort, key.KeyAlgorithm, key.KeySize,
		key.OwnerEmail, key.OwnerName, key.Organization,
		key.Trusted, key.Enabled, key.ValidFrom, key.ValidUntil,
		key.Description, key.KeySource, key.KeySourceURL, key.CreatedBy,
	).Scan(&key.KeyID, &key.CreatedAt, &key.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store public key: %w", err)
	}

	return nil
}

// GetTrustedKeys retrieves trusted public keys for a tenant/repository
func (s *Service) GetTrustedKeys(ctx context.Context, tenantID uuid.UUID, repositoryID *uuid.UUID) ([]*PublicKey, error) {
	query := `
		SELECT 
			key_id, tenant_id, repository_id, key_name, key_type, key_format, public_key,
			key_fingerprint, key_id_short, key_algorithm, key_size,
			owner_email, owner_name, organization,
			trusted, enabled, revoked, revoked_at, revocation_reason,
			valid_from, valid_until, description, key_source, key_source_url,
			created_at, updated_at, created_by
		FROM public_keys
		WHERE tenant_id = $1 
		AND (repository_id = $2 OR repository_id IS NULL)
		AND trusted = true 
		AND enabled = true
		AND revoked = false
		AND (valid_until IS NULL OR valid_until > CURRENT_TIMESTAMP)
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query trusted keys: %w", err)
	}
	defer rows.Close()

	var keys []*PublicKey
	for rows.Next() {
		var key PublicKey
		err := rows.Scan(
			&key.KeyID, &key.TenantID, &key.RepositoryID, &key.KeyName, &key.KeyType, &key.KeyFormat, &key.PublicKey,
			&key.KeyFingerprint, &key.KeyIDShort, &key.KeyAlgorithm, &key.KeySize,
			&key.OwnerEmail, &key.OwnerName, &key.Organization,
			&key.Trusted, &key.Enabled, &key.Revoked, &key.RevokedAt, &key.RevocationReason,
			&key.ValidFrom, &key.ValidUntil, &key.Description, &key.KeySource, &key.KeySourceURL,
			&key.CreatedAt, &key.UpdatedAt, &key.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan public key: %w", err)
		}
		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating keys: %w", err)
	}

	return keys, nil
}

// RepositorySignaturePolicy represents signature policy settings
type RepositorySignaturePolicy struct {
	RepositoryID                 uuid.UUID       `db:"repository_id"`
	TenantID                     uuid.UUID       `db:"tenant_id"`
	SignaturePolicy              SignaturePolicy `db:"signature_policy"`
	SignatureVerificationEnabled bool            `db:"signature_verification_enabled"`
	CosignEnabled                bool            `db:"cosign_enabled"`
	PGPEnabled                   bool            `db:"pgp_enabled"`
	SigstoreEnabled              bool            `db:"sigstore_enabled"`
	AllowedSigners               []string        `db:"allowed_signers"`
}

// CheckSignatureRequired checks if signature is required for upload
func (p *RepositorySignaturePolicy) CheckSignatureRequired() bool {
	return p.SignaturePolicy == SignaturePolicyRequired || p.SignaturePolicy == SignaturePolicyStrict
}

// CheckSignatureVerificationRequired checks if verification is required
func (p *RepositorySignaturePolicy) CheckSignatureVerificationRequired() bool {
	return p.SignatureVerificationEnabled && p.SignaturePolicy != SignaturePolicyDisabled
}
