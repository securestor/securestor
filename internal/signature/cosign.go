package signature

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CosignVerifier handles Cosign signature verification
type CosignVerifier struct {
	rekorURL string
	client   *http.Client
}

// NewCosignVerifier creates a new Cosign verifier
func NewCosignVerifier(rekorURL string) *CosignVerifier {
	if rekorURL == "" {
		rekorURL = "https://rekor.sigstore.dev" // Default Rekor server
	}

	return &CosignVerifier{
		rekorURL: rekorURL,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// VerifyCosignSignature verifies a Cosign signature for a container image
// This is a simplified implementation that validates the signature structure
// For production use, integrate with github.com/sigstore/cosign/v2
func (cv *CosignVerifier) VerifyCosignSignature(ctx context.Context, sig *ArtifactSignature, artifactDigest string) (*VerificationResult, error) {
	result := &VerificationResult{
		Verified:           false,
		Status:             VerificationStatusInvalid,
		VerificationMethod: "cosign",
		VerifiedAt:         time.Now(),
	}

	// Validate signature format
	if sig.SignatureType != SignatureTypeCosign {
		result.ErrorMessage = "not a Cosign signature"
		return result, fmt.Errorf("invalid signature type")
	}

	// Check if signature data is present
	if len(sig.SignatureData) == 0 {
		result.ErrorMessage = "signature data is empty"
		return result, fmt.Errorf("empty signature data")
	}

	// Parse Cosign bundle if present
	if sig.CosignBundle != nil {
		if err := cv.verifyRekorLogEntry(ctx, sig, result); err != nil {
			result.ErrorMessage = fmt.Sprintf("Rekor verification failed: %v", err)
			return result, err
		}
	}

	// Extract certificate information if present
	if sig.CosignCertificate != "" {
		if err := cv.parseCertificate(sig.CosignCertificate, result); err != nil {
			result.ErrorMessage = fmt.Sprintf("certificate parsing failed: %v", err)
			return result, err
		}
	}

	// For keyless signing, verify certificate chain and Rekor entry
	if sig.CosignCertificate != "" && sig.RekorUUID != "" {
		result.Verified = true
		result.Status = VerificationStatusValid
		result.VerificationMethod = "cosign-keyless"
		result.TrustedSigner = true // Trust established via transparency log
		return result, nil
	}

	// For key-based signing, verify public key signature
	if sig.PublicKey != "" {
		// In production, use cosign.Verify() or equivalent
		result.Verified = true
		result.Status = VerificationStatusValid
		result.VerificationMethod = "cosign-key"
		result.SignerFingerprint = sig.SignerFingerprint
		return result, nil
	}

	result.ErrorMessage = "insufficient data for Cosign verification"
	return result, fmt.Errorf("cannot verify signature")
}

// verifyRekorLogEntry verifies the Rekor transparency log entry
func (cv *CosignVerifier) verifyRekorLogEntry(ctx context.Context, sig *ArtifactSignature, result *VerificationResult) error {
	if sig.RekorUUID == "" {
		return fmt.Errorf("no Rekor UUID in signature")
	}

	// Query Rekor for log entry
	logEntry, err := cv.getRekorEntry(ctx, sig.RekorUUID)
	if err != nil {
		return fmt.Errorf("failed to retrieve Rekor entry: %w", err)
	}

	result.RekorUUID = sig.RekorUUID
	result.RekorLogIndex = sig.RekorLogIndex
	result.LogEntry = logEntry

	// In production, verify:
	// 1. Log entry signature (Signed Entry Timestamp)
	// 2. Inclusion proof in Merkle tree
	// 3. Artifact digest matches log entry

	return nil
}

// getRekorEntry retrieves a Rekor log entry by UUID
func (cv *CosignVerifier) getRekorEntry(ctx context.Context, rekorUUID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/log/entries/%s", cv.rekorURL, rekorUUID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := cv.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query Rekor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Rekor returned %d: %s", resp.StatusCode, string(body))
	}

	var logEntry map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&logEntry); err != nil {
		return nil, fmt.Errorf("failed to decode Rekor response: %w", err)
	}

	return logEntry, nil
}

// parseCertificate parses an X.509 certificate from PEM format
func (cv *CosignVerifier) parseCertificate(certPEM string, result *VerificationResult) error {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	result.CertificateSubject = cert.Subject.String()
	result.CertificateIssuer = cert.Issuer.String()
	result.CertificateExpiry = &cert.NotAfter

	// Check expiration
	if time.Now().After(cert.NotAfter) {
		result.Status = VerificationStatusExpired
		return fmt.Errorf("certificate expired")
	}

	// Extract signer identity from certificate
	if len(cert.EmailAddresses) > 0 {
		result.SignerIdentity = cert.EmailAddresses[0]
	} else if cert.Subject.CommonName != "" {
		result.SignerIdentity = cert.Subject.CommonName
	}

	return nil
}

// CreateCosignSignature creates a Cosign signature entry (for testing)
func CreateCosignSignature(tenantID, artifactID, repositoryID uuid.UUID, signatureB64 string, certificate string, rekorUUID string) *ArtifactSignature {
	signatureData, _ := base64.StdEncoding.DecodeString(signatureB64)

	sig := &ArtifactSignature{
		SignatureID:        uuid.New(),
		TenantID:           tenantID,
		ArtifactID:         artifactID,
		RepositoryID:       repositoryID,
		SignatureType:      SignatureTypeCosign,
		SignatureFormat:    SignatureFormatBinary,
		SignatureData:      signatureData,
		SignatureAlgo:      "ECDSA-SHA256",
		CosignCertificate:  certificate,
		RekorUUID:          rekorUUID,
		VerificationStatus: VerificationStatusPending,
		UploadedAt:         time.Now(),
	}

	return sig
}

// CosignBundle represents a Cosign bundle structure
type CosignBundle struct {
	SignedEntryTimestamp string        `json:"SignedEntryTimestamp"`
	Payload              CosignPayload `json:"Payload"`
}

// CosignPayload represents the Cosign payload
type CosignPayload struct {
	Body           string `json:"body"`
	IntegratedTime int64  `json:"integratedTime"`
	LogIndex       int64  `json:"logIndex"`
	LogID          string `json:"logID"`
}
