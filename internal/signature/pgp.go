package signature

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// PGPVerifier handles PGP signature verification
type PGPVerifier struct {
	keyring openpgp.EntityList
}

// NewPGPVerifier creates a new PGP verifier
func NewPGPVerifier() *PGPVerifier {
	return &PGPVerifier{
		keyring: make(openpgp.EntityList, 0),
	}
}

// AddPublicKey adds a public key to the keyring
func (pv *PGPVerifier) AddPublicKey(publicKeyPEM string) error {
	var keyReader *bytes.Reader

	// Check if it's ASCII-armored
	if strings.Contains(publicKeyPEM, "BEGIN PGP PUBLIC KEY") {
		block, err := armor.Decode(strings.NewReader(publicKeyPEM))
		if err != nil {
			return fmt.Errorf("failed to decode armored key: %w", err)
		}
		keyData, err := io.ReadAll(block.Body)
		if err != nil {
			return fmt.Errorf("failed to read armored key: %w", err)
		}
		keyReader = bytes.NewReader(keyData)
	} else {
		keyReader = bytes.NewReader([]byte(publicKeyPEM))
	}

	entityList, err := openpgp.ReadKeyRing(keyReader)
	if err != nil {
		return fmt.Errorf("failed to read PGP key: %w", err)
	}

	pv.keyring = append(pv.keyring, entityList...)
	return nil
}

// AddPublicKeys adds multiple public keys to the keyring
func (pv *PGPVerifier) AddPublicKeys(keys []*PublicKey) error {
	for _, key := range keys {
		if key.KeyType == "pgp" && key.Enabled && !key.Revoked {
			if err := pv.AddPublicKey(key.PublicKey); err != nil {
				return fmt.Errorf("failed to add key %s: %w", key.KeyName, err)
			}
		}
	}
	return nil
}

// VerifyPGPSignature verifies a PGP signature
func (pv *PGPVerifier) VerifyPGPSignature(ctx context.Context, sig *ArtifactSignature, artifactData []byte) (*VerificationResult, error) {
	result := &VerificationResult{
		Verified:           false,
		Status:             VerificationStatusInvalid,
		VerificationMethod: "pgp",
		VerifiedAt:         time.Now(),
	}

	// Validate signature format
	if sig.SignatureType != SignatureTypePGP {
		result.ErrorMessage = "not a PGP signature"
		return result, fmt.Errorf("invalid signature type")
	}

	if len(sig.SignatureData) == 0 {
		result.ErrorMessage = "signature data is empty"
		return result, fmt.Errorf("empty signature data")
	}

	// Decode signature if ASCII-armored
	signatureReader := bytes.NewReader(sig.SignatureData)
	if sig.SignatureFormat == SignatureFormatASCIIArmor {
		block, err := armor.Decode(signatureReader)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to decode armored signature: %v", err)
			return result, err
		}
		sigData, err := io.ReadAll(block.Body)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to read armored signature: %v", err)
			return result, err
		}
		signatureReader = bytes.NewReader(sigData)
	}

	// Parse signature packet
	reader := packet.NewReader(signatureReader)
	pkt, err := reader.Next()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to parse signature: %v", err)
		return result, err
	}

	sigPacket, ok := pkt.(*packet.Signature)
	if !ok {
		result.ErrorMessage = "not a valid PGP signature packet"
		return result, fmt.Errorf("invalid packet type")
	}

	// If public key is provided in signature, add it to keyring
	if sig.PublicKey != "" {
		if err := pv.AddPublicKey(sig.PublicKey); err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to add public key: %v", err)
			return result, err
		}
	}

	// Find the signer's key in keyring
	var signer *openpgp.Entity
	keyID := sigPacket.IssuerKeyId
	if keyID != nil {
		for _, entity := range pv.keyring {
			if entity.PrimaryKey.KeyId == *keyID {
				signer = entity
				break
			}
		}
	}

	if signer == nil {
		result.ErrorMessage = "signer's public key not found in keyring"
		result.Status = VerificationStatusUntrusted
		return result, fmt.Errorf("unknown signer")
	}

	// Verify the signature
	dataReader := bytes.NewReader(artifactData)
	_, err = openpgp.CheckDetachedSignature(pv.keyring, dataReader, signatureReader)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("signature verification failed: %v", err)
		return result, err
	}

	// Signature is valid
	result.Verified = true
	result.Status = VerificationStatusValid
	result.SignerIdentity = getSignerIdentity(signer)
	result.SignerFingerprint = fmt.Sprintf("%X", signer.PrimaryKey.Fingerprint)
	result.SignatureAlgorithm = fmt.Sprintf("PGP-%d", sigPacket.PubKeyAlgo)
	result.TrustedSigner = true // Key is in our trusted keyring

	return result, nil
}

// VerifyPGPSignatureDetached verifies a detached PGP signature
func (pv *PGPVerifier) VerifyPGPSignatureDetached(signatureData, artifactData []byte) (*VerificationResult, error) {
	result := &VerificationResult{
		Verified:           false,
		Status:             VerificationStatusInvalid,
		VerificationMethod: "pgp-detached",
		VerifiedAt:         time.Now(),
	}

	signatureReader := bytes.NewReader(signatureData)
	dataReader := bytes.NewReader(artifactData)

	signer, err := openpgp.CheckDetachedSignature(pv.keyring, dataReader, signatureReader)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("signature verification failed: %v", err)
		return result, err
	}

	if signer == nil {
		result.ErrorMessage = "no signer found"
		result.Status = VerificationStatusUntrusted
		return result, fmt.Errorf("unknown signer")
	}

	result.Verified = true
	result.Status = VerificationStatusValid
	result.SignerIdentity = getSignerIdentity(signer)
	result.SignerFingerprint = fmt.Sprintf("%X", signer.PrimaryKey.Fingerprint)
	result.TrustedSigner = true

	return result, nil
}

// ExtractPGPKeyInfo extracts information from a PGP public key
func ExtractPGPKeyInfo(publicKeyPEM string) (*PublicKey, error) {
	var keyReader *bytes.Reader

	// Check if it's ASCII-armored
	if strings.Contains(publicKeyPEM, "BEGIN PGP PUBLIC KEY") {
		block, err := armor.Decode(strings.NewReader(publicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to decode armored key: %w", err)
		}
		keyData, err := io.ReadAll(block.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read armored key: %w", err)
		}
		keyReader = bytes.NewReader(keyData)
	} else {
		keyReader = bytes.NewReader([]byte(publicKeyPEM))
	}

	entityList, err := openpgp.ReadKeyRing(keyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PGP key: %w", err)
	}

	if len(entityList) == 0 {
		return nil, fmt.Errorf("no keys found")
	}

	entity := entityList[0]

	pubKey := &PublicKey{
		KeyID:          uuid.New(),
		KeyType:        "pgp",
		KeyFormat:      "ascii-armor",
		PublicKey:      publicKeyPEM,
		KeyFingerprint: fmt.Sprintf("%X", entity.PrimaryKey.Fingerprint),
		KeyIDShort:     fmt.Sprintf("%X", entity.PrimaryKey.KeyId),
		KeyAlgorithm:   fmt.Sprintf("PGP-%d", entity.PrimaryKey.PubKeyAlgo),
		Enabled:        true,
		Trusted:        false,
		KeySource:      "manual",
	}

	// Extract key size
	switch entity.PrimaryKey.PubKeyAlgo {
	case packet.PubKeyAlgoRSA, packet.PubKeyAlgoRSAEncryptOnly, packet.PubKeyAlgoRSASignOnly:
		bits, err := entity.PrimaryKey.BitLength()
		if err == nil {
			bitSize := int(bits)
			pubKey.KeySize = &bitSize
		}
	}

	// Extract identity information
	for _, ident := range entity.Identities {
		if ident.UserId != nil {
			pubKey.OwnerName = ident.UserId.Name
			pubKey.OwnerEmail = ident.UserId.Email
			pubKey.KeyName = ident.UserId.Name
			break
		}
	}

	// Set validity period
	if entity.PrimaryKey.CreationTime.Unix() > 0 {
		pubKey.ValidFrom = &entity.PrimaryKey.CreationTime
	}

	return pubKey, nil
}

// getSignerIdentity extracts identity from PGP entity
func getSignerIdentity(entity *openpgp.Entity) string {
	for _, ident := range entity.Identities {
		if ident.UserId != nil {
			if ident.UserId.Email != "" {
				return ident.UserId.Email
			}
			return ident.UserId.Name
		}
	}
	return fmt.Sprintf("Key ID: %X", entity.PrimaryKey.KeyId)
}
