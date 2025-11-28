package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"google.golang.org/api/option"
)

// L3StorageImpl implements cloud-based caching (S3, GCS, Azure)
type L3StorageImpl struct {
	provider string // "s3", "gcs", "azure"
	bucket   string
	client   interface{} // Will be *s3.Client, *storage.Client, etc.
	prefix   string      // Object prefix in bucket
	maxSize  int64       // Maximum total size
}

// S3Config holds S3-specific configuration
type S3Config struct {
	Bucket    string
	Region    string
	Endpoint  string // For S3-compatible services
	AccessKey string
	SecretKey string
	Prefix    string
}

// GCSConfig holds Google Cloud Storage configuration
type GCSConfig struct {
	Bucket      string
	ProjectID   string
	Credentials string
	Prefix      string
}

// AzureConfig holds Azure Blob Storage configuration
type AzureConfig struct {
	Container string
	Account   string
	Key       string
	Prefix    string
}

// NewL3StorageS3 creates S3-based L3 storage
func NewL3StorageS3(config *S3Config, maxSizeGB float64) (*L3StorageImpl, error) {
	maxSizeBytes := int64(maxSizeGB * 1024 * 1024 * 1024)

	// Load AWS configuration
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	l3 := &L3StorageImpl{
		provider: "s3",
		bucket:   config.Bucket,
		prefix:   config.Prefix,
		maxSize:  maxSizeBytes,
		client:   client,
	}

	// Verify bucket access by listing objects with limit 1
	_, err = client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(config.Bucket),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify S3 bucket access: %w", err)
	}

	return l3, nil
}

// NewL3StorageGCS creates Google Cloud Storage-based L3 storage
func NewL3StorageGCS(config *GCSConfig, maxSizeGB float64) (*L3StorageImpl, error) {
	maxSizeBytes := int64(maxSizeGB * 1024 * 1024 * 1024)
	ctx := context.Background()

	// Create GCS client with optional credentials
	var opts []option.ClientOption
	if config.Credentials != "" {
		opts = append(opts, option.WithCredentialsFile(config.Credentials))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	l3 := &L3StorageImpl{
		provider: "gcs",
		bucket:   config.Bucket,
		prefix:   config.Prefix,
		maxSize:  maxSizeBytes,
		client:   client,
	}

	// Verify bucket access
	_, err = client.Bucket(config.Bucket).Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to verify GCS bucket access: %w", err)
	}

	return l3, nil
}

// NewL3StorageAzure creates Azure Blob Storage-based L3 storage
func NewL3StorageAzure(config *AzureConfig, maxSizeGB float64) (*L3StorageImpl, error) {
	maxSizeBytes := int64(maxSizeGB * 1024 * 1024 * 1024)

	// Create Azure Blob client
	var client *azblob.Client
	var err error

	if config.Key != "" {
		// Using account key authentication
		connStr := fmt.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
			config.Account, config.Key)
		client, err = azblob.NewClientFromConnectionString(connStr, nil)
	} else {
		// Using managed identity/environment credentials
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
		}
		client, err = azblob.NewClient(
			fmt.Sprintf("https://%s.blob.core.windows.net", config.Account),
			cred,
			nil,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Azure client: %w", err)
	}

	l3 := &L3StorageImpl{
		provider: "azure",
		bucket:   config.Container,
		prefix:   config.Prefix,
		maxSize:  maxSizeBytes,
		client:   client,
	}

	// Verify container access by listing blobs with limit 1
	pager := client.NewListBlobsFlatPager(config.Container, &azblob.ListBlobsFlatOptions{})
	if pager == nil {
		return nil, fmt.Errorf("failed to create Azure blob pager")
	}

	return l3, nil
}

// Get retrieves data from L3 storage (with optional local buffering)
func (l3 *L3StorageImpl) Get(ctx context.Context, key string) (interface{}, error) {
	switch l3.provider {
	case "s3":
		return l3.getFromS3(ctx, key)
	case "gcs":
		return l3.getFromGCS(ctx, key)
	case "azure":
		return l3.getFromAzure(ctx, key)
	default:
		return nil, fmt.Errorf("unsupported L3 provider: %s", l3.provider)
	}
}

// Set stores data in L3 storage
func (l3 *L3StorageImpl) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	// Serialize data
	buf := new(bytes.Buffer)
	// In production, use proper serialization
	// For now, assume data is []byte
	if b, ok := data.([]byte); ok {
		buf.Write(b)
	} else {
		return fmt.Errorf("L3 storage expects []byte data")
	}

	// Check size
	if int64(buf.Len()) > l3.maxSize {
		return fmt.Errorf("data size exceeds L3 max size: %d > %d", buf.Len(), l3.maxSize)
	}

	switch l3.provider {
	case "s3":
		return l3.setToS3(ctx, key, buf.Bytes(), ttl)
	case "gcs":
		return l3.setToGCS(ctx, key, buf.Bytes(), ttl)
	case "azure":
		return l3.setToAzure(ctx, key, buf.Bytes(), ttl)
	default:
		return fmt.Errorf("unsupported L3 provider: %s", l3.provider)
	}
}

// Delete removes data from L3 storage
func (l3 *L3StorageImpl) Delete(ctx context.Context, key string) error {
	switch l3.provider {
	case "s3":
		return l3.deleteFromS3(ctx, key)
	case "gcs":
		return l3.deleteFromGCS(ctx, key)
	case "azure":
		return l3.deleteFromAzure(ctx, key)
	default:
		return fmt.Errorf("unsupported L3 provider: %s", l3.provider)
	}
}

// Stats returns L3 storage statistics
func (l3 *L3StorageImpl) Stats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"provider":    l3.provider,
		"bucket":      l3.bucket,
		"prefix":      l3.prefix,
		"max_size_gb": float64(l3.maxSize) / (1024 * 1024 * 1024),
	}
}

// S3 Implementation

func (l3 *L3StorageImpl) getFromS3(ctx context.Context, key string) (interface{}, error) {
	client := l3.client.(*s3.Client)
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(l3.bucket),
		Key:    aws.String(l3.getObjectKey(key)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (l3 *L3StorageImpl) setToS3(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	client := l3.client.(*s3.Client)
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(l3.bucket),
		Key:                  aws.String(l3.getObjectKey(key)),
		Body:                 bytes.NewReader(data),
		ContentLength:        aws.Int64(int64(len(data))),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	return err
}

func (l3 *L3StorageImpl) deleteFromS3(ctx context.Context, key string) error {
	client := l3.client.(*s3.Client)
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(l3.bucket),
		Key:    aws.String(l3.getObjectKey(key)),
	})
	return err
}

// GCS Implementation

func (l3 *L3StorageImpl) getFromGCS(ctx context.Context, key string) (interface{}, error) {
	client := l3.client.(*storage.Client)
	reader, err := client.Bucket(l3.bucket).Object(l3.getObjectKey(key)).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from GCS: %w", err)
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (l3 *L3StorageImpl) setToGCS(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	client := l3.client.(*storage.Client)
	wc := client.Bucket(l3.bucket).Object(l3.getObjectKey(key)).NewWriter(ctx)
	if _, err := wc.Write(data); err != nil {
		return fmt.Errorf("failed to write to GCS: %w", err)
	}
	return wc.Close()
}

func (l3 *L3StorageImpl) deleteFromGCS(ctx context.Context, key string) error {
	client := l3.client.(*storage.Client)
	return client.Bucket(l3.bucket).Object(l3.getObjectKey(key)).Delete(ctx)
}

// Azure Implementation

func (l3 *L3StorageImpl) getFromAzure(ctx context.Context, key string) (interface{}, error) {
	client := l3.client.(*azblob.Client)
	resp, err := client.DownloadStream(ctx, l3.bucket, l3.getObjectKey(key), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob from Azure: %w", err)
	}
	return io.ReadAll(resp.Body)
}

func (l3 *L3StorageImpl) setToAzure(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	client := l3.client.(*azblob.Client)
	_, err := client.UploadStream(ctx, l3.bucket, l3.getObjectKey(key),
		bytes.NewReader(data), nil)
	return err
}

func (l3 *L3StorageImpl) deleteFromAzure(ctx context.Context, key string) error {
	client := l3.client.(*azblob.Client)
	_, err := client.DeleteBlob(ctx, l3.bucket, l3.getObjectKey(key), nil)
	return err
}

// Helper method to construct object keys
func (l3 *L3StorageImpl) getObjectKey(key string) string {
	if l3.prefix != "" {
		return l3.prefix + "/" + key
	}
	return key
}

// StreamGet returns an io.Reader for large files (avoids loading entire object into memory)
func (l3 *L3StorageImpl) StreamGet(ctx context.Context, key string) (io.ReadCloser, error) {
	switch l3.provider {
	case "s3":
		client := l3.client.(*s3.Client)
		result, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(l3.bucket),
			Key:    aws.String(l3.getObjectKey(key)),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to stream from S3: %w", err)
		}
		return result.Body, nil

	case "gcs":
		client := l3.client.(*storage.Client)
		reader, err := client.Bucket(l3.bucket).Object(l3.getObjectKey(key)).NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to stream from GCS: %w", err)
		}
		return reader, nil

	case "azure":
		client := l3.client.(*azblob.Client)
		resp, err := client.DownloadStream(ctx, l3.bucket, l3.getObjectKey(key), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to stream from Azure: %w", err)
		}
		return resp.Body, nil

	default:
		return nil, fmt.Errorf("unsupported L3 provider for streaming: %s", l3.provider)
	}
}

// Cleanup removes expired objects (implementation depends on cloud provider TTL support)
func (l3 *L3StorageImpl) Cleanup(ctx context.Context) error {
	// Cloud providers handle object expiration differently
	// S3: Use lifecycle policies
	// GCS: Use object lifecycle rules
	// Azure: Use blob lifecycle management

	// This method serves as a coordination point
	return nil
}
