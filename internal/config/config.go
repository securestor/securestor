package config

import (
	"strconv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	Environment string
	StoragePath string
	MaxFileSize int64

	// Erasure Coding Configuration
	ErasureDataShards   int
	ErasureParityShards int

	// Redis Configuration
	RedisCacheTTL   int
	RedisSessionTTL int

	// Encryption Configuration
	EncryptionMode       string // mock, aws-kms, azure-keyvault, vault
	EncryptionEnabled    bool
	EncryptionEnforced   bool
	EncryptionMasterKey  string // Base64-encoded key for mock mode
	AWSKMSKeyID          string // ARN for AWS KMS
	AWSRegion            string
	AzureKeyVaultURL     string
	AzureKeyVaultKeyName string
	AzureTenantID        string
	VaultAddr            string
	VaultToken           string
	VaultKeyPath         string
	KeyCacheTTLMinutes   int
	KeyRotationDays      int
}

func Load() (*Config, error) {
	// Use the centralized environment loader
	LoadEnvOnce()

	maxFileSize, _ := strconv.ParseInt(GetEnvWithFallback("MAX_FILE_SIZE", "524288000"), 10, 64) // 500MB default
	dataShards, _ := strconv.Atoi(GetEnvWithFallback("ERASURE_DATA_SHARDS", "4"))
	parityShards, _ := strconv.Atoi(GetEnvWithFallback("ERASURE_PARITY_SHARDS", "2"))
	redisCacheTTL, _ := strconv.Atoi(GetEnvWithFallback("REDIS_CACHE_TTL", "3600"))
	redisSessionTTL, _ := strconv.Atoi(GetEnvWithFallback("REDIS_SESSION_TTL", "86400"))
	keyCacheTTL, _ := strconv.Atoi(GetEnvWithFallback("KEY_CACHE_TTL_MINUTES", "5"))
	keyRotationDays, _ := strconv.Atoi(GetEnvWithFallback("KEY_ROTATION_DAYS", "90"))
	encryptionEnabled, _ := strconv.ParseBool(GetEnvWithFallback("ENCRYPTION_ENABLED", "false"))
	encryptionEnforced, _ := strconv.ParseBool(GetEnvWithFallback("ENCRYPTION_ENFORCED", "false"))

	return &Config{
		Port:                GetEnvWithFallback("PORT", "8080"),
		DatabaseURL:         GetEnvWithFallback("DATABASE_URL", "postgresql://localhost:5432/securestor?sslmode=disable"),
		RedisURL:            GetEnvWithFallback("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:           GetEnvWithFallback("JWT_SECRET", "your-secret-key"),
		Environment:         GetEnvWithFallback("ENVIRONMENT", "development"),
		StoragePath:         GetEnvWithFallback("STORAGE_PATH", "./data"),
		MaxFileSize:         maxFileSize,
		ErasureDataShards:   dataShards,
		ErasureParityShards: parityShards,
		RedisCacheTTL:       redisCacheTTL,
		RedisSessionTTL:     redisSessionTTL,

		// Encryption Configuration
		EncryptionMode:       GetEnvWithFallback("ENCRYPTION_MODE", "mock"),
		EncryptionEnabled:    encryptionEnabled,
		EncryptionEnforced:   encryptionEnforced,
		EncryptionMasterKey:  GetEnvWithFallback("ENCRYPTION_MASTER_KEY", ""),
		AWSKMSKeyID:          GetEnvWithFallback("AWS_KMS_KEY_ID", ""),
		AWSRegion:            GetEnvWithFallback("AWS_REGION", "us-east-1"),
		AzureKeyVaultURL:     GetEnvWithFallback("AZURE_KEYVAULT_URL", ""),
		AzureKeyVaultKeyName: GetEnvWithFallback("AZURE_KEYVAULT_KEY_NAME", ""),
		AzureTenantID:        GetEnvWithFallback("AZURE_TENANT_ID", ""),
		VaultAddr:            GetEnvWithFallback("VAULT_ADDR", ""),
		VaultToken:           GetEnvWithFallback("VAULT_TOKEN", ""),
		VaultKeyPath:         GetEnvWithFallback("VAULT_KEY_PATH", ""),
		KeyCacheTTLMinutes:   keyCacheTTL,
		KeyRotationDays:      keyRotationDays,
	}, nil
}
