package adapters

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/securestor/securestor/internal/storage"
)

// AdapterFactory creates and manages database and storage adapters
type AdapterFactory struct {
	db     *sql.DB
	logger *log.Logger
	config AdapterConfig
}

// AdapterConfig contains configuration for all adapters
type AdapterConfig struct {
	Database DatabaseConfig `json:"database"`
	Storage  StorageConfig  `json:"storage"`
}

// DatabaseConfig contains database adapter configuration
type DatabaseConfig struct {
	Type         string `json:"type"` // postgresql, mysql, sqlite
	MaxConns     int    `json:"max_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
	EnableAudit  bool   `json:"enable_audit"`
}

// StorageConfig contains storage adapter configuration
type StorageConfig struct {
	Type              string                    `json:"type"`      // local, s3, gcp, azure, minio
	BasePath          string                    `json:"base_path"` // for local storage
	EnableEncryption  bool                      `json:"enable_encryption"`
	EnableCompression bool                      `json:"enable_compression"`
	ErasureCoding     ErasureCodingConfig       `json:"erasure_coding"`
	CloudConfig       CloudStorageConfig        `json:"cloud_config"`
	LocalConfig       LocalStorageFactoryConfig `json:"local_config"`
}

// ErasureCodingConfig contains erasure coding settings
type ErasureCodingConfig struct {
	Enabled      bool `json:"enabled"`
	DataShards   int  `json:"data_shards"`
	ParityShards int  `json:"parity_shards"`
}

// CloudStorageConfig contains cloud-specific settings
type CloudStorageConfig struct {
	Region          string            `json:"region"`
	Bucket          string            `json:"bucket"`
	Endpoint        string            `json:"endpoint"`
	AccessKeyID     string            `json:"access_key_id"`
	SecretAccessKey string            `json:"secret_access_key"`
	ExtraConfig     map[string]string `json:"extra_config"`
}

// LocalStorageFactoryConfig contains local storage settings for the factory
type LocalStorageFactoryConfig struct {
	BasePath          string `json:"base_path"`
	DataShards        int    `json:"data_shards"`
	ParityShards      int    `json:"parity_shards"`
	EnableEncryption  bool   `json:"enable_encryption"`
	EnableCompression bool   `json:"enable_compression"`
	MaxConcurrentOps  int    `json:"max_concurrent_ops"`
}

// AdapterSet contains all initialized adapters
type AdapterSet struct {
	DB      DBAdapter
	Storage StorageAdapter
	Logger  *log.Logger
}

// NewAdapterFactory creates a new adapter factory
func NewAdapterFactory(db *sql.DB, config AdapterConfig, logger *log.Logger) *AdapterFactory {
	return &AdapterFactory{
		db:     db,
		config: config,
		logger: logger,
	}
}

// CreateAdapters creates and initializes all adapters
func (f *AdapterFactory) CreateAdapters() (*AdapterSet, error) {
	// Create database adapter
	dbAdapter, err := f.createDatabaseAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to create database adapter: %w", err)
	}

	// Create storage adapter
	storageAdapter, err := f.createStorageAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage adapter: %w", err)
	}

	return &AdapterSet{
		DB:      dbAdapter,
		Storage: storageAdapter,
		Logger:  f.logger,
	}, nil
}

// createDatabaseAdapter creates the appropriate database adapter
func (f *AdapterFactory) createDatabaseAdapter() (DBAdapter, error) {
	switch f.config.Database.Type {
	case "postgresql", "postgres":
		return NewPostgreSQLAdapter(f.db, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", f.config.Database.Type)
	}
}

// createStorageAdapter creates the appropriate storage adapter
func (f *AdapterFactory) createStorageAdapter() (StorageAdapter, error) {
	switch f.config.Storage.Type {
	case "local":
		return f.createLocalStorageAdapter()
	case "s3":
		return f.createS3StorageAdapter()
	case "gcp":
		return f.createGCPStorageAdapter()
	case "azure":
		return f.createAzureStorageAdapter()
	case "minio":
		return f.createMinioStorageAdapter()
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", f.config.Storage.Type)
	}
}

// createLocalStorageAdapter creates a local storage adapter
func (f *AdapterFactory) createLocalStorageAdapter() (StorageAdapter, error) {
	// Create erasure coder wrapper that implements the interface
	erasureCoder, err := storage.NewErasureCoder(
		f.config.Storage.ErasureCoding.DataShards,
		f.config.Storage.ErasureCoding.ParityShards,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create erasure coder: %w", err)
	}

	// Wrap the storage erasure coder to implement our interface
	erasureCoderWrapper := &ErasureCoderWrapper{erasureCoder}

	// Create encryptor (placeholder - would need actual implementation)
	encryptor := &NullEncryptor{} // Placeholder

	// Create local storage config
	localConfig := LocalStorageConfig{
		BasePath:          f.config.Storage.BasePath,
		DataShards:        f.config.Storage.ErasureCoding.DataShards,
		ParityShards:      f.config.Storage.ErasureCoding.ParityShards,
		EnableEncryption:  f.config.Storage.EnableEncryption,
		EnableCompression: f.config.Storage.EnableCompression,
		MaxConcurrentOps:  10, // Default value
	}

	return NewLocalStorageAdapter(localConfig, erasureCoderWrapper, encryptor, f.logger), nil
}

// createS3StorageAdapter creates an S3 storage adapter
func (f *AdapterFactory) createS3StorageAdapter() (StorageAdapter, error) {
	f.logger.Printf("Creating S3 storage adapter")
	return nil, fmt.Errorf("S3 storage adapter not implemented yet")
}

// createGCPStorageAdapter creates a GCP storage adapter
func (f *AdapterFactory) createGCPStorageAdapter() (StorageAdapter, error) {
	f.logger.Printf("Creating GCP storage adapter")
	return nil, fmt.Errorf("GCP storage adapter not implemented yet")
}

// createAzureStorageAdapter creates an Azure storage adapter
func (f *AdapterFactory) createAzureStorageAdapter() (StorageAdapter, error) {
	f.logger.Printf("Creating Azure storage adapter")
	return nil, fmt.Errorf("Azure storage adapter not implemented yet")
}

// createMinioStorageAdapter creates a MinIO storage adapter
func (f *AdapterFactory) createMinioStorageAdapter() (StorageAdapter, error) {
	f.logger.Printf("Creating MinIO storage adapter")
	return nil, fmt.Errorf("MinIO storage adapter not implemented yet")
}

// ValidateConfig validates the adapter configuration
func (f *AdapterFactory) ValidateConfig() error {
	// Validate database config
	if f.config.Database.Type == "" {
		return fmt.Errorf("database type is required")
	}

	// Validate storage config
	if f.config.Storage.Type == "" {
		return fmt.Errorf("storage type is required")
	}

	// Validate erasure coding config
	if f.config.Storage.ErasureCoding.Enabled {
		if f.config.Storage.ErasureCoding.DataShards <= 0 {
			return fmt.Errorf("data shards must be positive")
		}
		if f.config.Storage.ErasureCoding.ParityShards <= 0 {
			return fmt.Errorf("parity shards must be positive")
		}
		if f.config.Storage.ErasureCoding.DataShards+f.config.Storage.ErasureCoding.ParityShards > 256 {
			return fmt.Errorf("total shards cannot exceed 256")
		}
	}

	// Validate storage-specific config
	switch f.config.Storage.Type {
	case "local":
		if f.config.Storage.BasePath == "" {
			return fmt.Errorf("base path is required for local storage")
		}
	case "s3", "minio":
		if f.config.Storage.CloudConfig.Bucket == "" {
			return fmt.Errorf("bucket is required for S3/MinIO storage")
		}
		if f.config.Storage.CloudConfig.AccessKeyID == "" || f.config.Storage.CloudConfig.SecretAccessKey == "" {
			return fmt.Errorf("access credentials are required for S3/MinIO storage")
		}
	case "gcp":
		if f.config.Storage.CloudConfig.Bucket == "" {
			return fmt.Errorf("bucket is required for GCP storage")
		}
	case "azure":
		if f.config.Storage.CloudConfig.Bucket == "" {
			return fmt.Errorf("container is required for Azure storage")
		}
	}

	return nil
}

// GetDefaultConfig returns a default adapter configuration
func GetDefaultConfig() AdapterConfig {
	return AdapterConfig{
		Database: DatabaseConfig{
			Type:         "postgresql",
			MaxConns:     25,
			MaxIdleConns: 5,
			EnableAudit:  true,
		},
		Storage: StorageConfig{
			Type:              "local",
			BasePath:          "./data/storage",
			EnableEncryption:  true,
			EnableCompression: false,
			ErasureCoding: ErasureCodingConfig{
				Enabled:      true,
				DataShards:   6,
				ParityShards: 3,
			},
			LocalConfig: LocalStorageFactoryConfig{
				BasePath:          "./data/storage",
				DataShards:        6,
				ParityShards:      3,
				EnableEncryption:  true,
				EnableCompression: false,
				MaxConcurrentOps:  10,
			},
		},
	}
}

// NullEncryptor is a placeholder encryptor that does nothing
type NullEncryptor struct{}

func (n *NullEncryptor) Encrypt(data []byte, key []byte) ([]byte, error) {
	return data, nil // No encryption - just pass through
}

func (n *NullEncryptor) Decrypt(encryptedData []byte, key []byte) ([]byte, error) {
	return encryptedData, nil // No decryption - just pass through
}

func (n *NullEncryptor) GenerateKey() []byte {
	return make([]byte, 32) // Return empty 32-byte key
}

// ErasureCoderWrapper wraps the storage.ErasureCoder to implement our ErasureCoder interface
type ErasureCoderWrapper struct {
	coder *storage.ErasureCoder
}

func (w *ErasureCoderWrapper) Encode(data []byte) ([][]byte, error) {
	return w.coder.Encode(data)
}

func (w *ErasureCoderWrapper) Decode(shards [][]byte, originalSize int) ([]byte, error) {
	return w.coder.Decode(shards, originalSize)
}

func (w *ErasureCoderWrapper) GetDataShards() int {
	// We'll need to expose this from the storage.ErasureCoder
	return 6 // Default value for now
}

func (w *ErasureCoderWrapper) GetParityShards() int {
	// We'll need to expose this from the storage.ErasureCoder
	return 3 // Default value for now
}

// AdapterManager manages adapter lifecycle
type AdapterManager struct {
	adapters *AdapterSet
	factory  *AdapterFactory
}

// NewAdapterManager creates a new adapter manager
func NewAdapterManager(db *sql.DB, config AdapterConfig, logger *log.Logger) (*AdapterManager, error) {
	factory := NewAdapterFactory(db, config, logger)

	// Validate configuration
	if err := factory.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid adapter configuration: %w", err)
	}

	// Create adapters
	adapters, err := factory.CreateAdapters()
	if err != nil {
		return nil, fmt.Errorf("failed to create adapters: %w", err)
	}

	return &AdapterManager{
		adapters: adapters,
		factory:  factory,
	}, nil
}

// GetDBAdapter returns the database adapter
func (m *AdapterManager) GetDBAdapter() DBAdapter {
	return m.adapters.DB
}

// GetStorageAdapter returns the storage adapter
func (m *AdapterManager) GetStorageAdapter() StorageAdapter {
	return m.adapters.Storage
}

// GetAdapters returns all adapters
func (m *AdapterManager) GetAdapters() *AdapterSet {
	return m.adapters
}

// Close closes all adapters and releases resources
func (m *AdapterManager) Close() error {
	if m.adapters.Storage != nil {
		if err := m.adapters.Storage.Close(); err != nil {
			m.adapters.Logger.Printf("Error closing storage adapter: %v", err)
		}
	}

	m.adapters.Logger.Printf("Adapter manager closed")
	return nil
}
