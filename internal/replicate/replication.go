package replicate

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/securestor/securestor/internal/config"
	"github.com/securestor/securestor/internal/logger"
)

// ReplicationService handles 3-way synchronous replication
type ReplicationService struct {
	storageNodes []StorageNode
	logger       *logger.Logger
	mutex        sync.RWMutex
	healthStatus map[string]NodeHealth
	db           *sql.DB
	tenantID     string // Current tenant context
}

// StorageNode represents a single storage node
type StorageNode struct {
	ID           string
	NodeName     string
	Path         string
	Priority     int
	NodeType     string // 'local', 's3', 'minio', 'gcs', 'azure'
	S3Endpoint   string
	S3Region     string
	S3Bucket     string
	S3AccessKey  string
	S3SecretKey  string
	S3UseSSL     bool
	S3PathPrefix string
}

// NodeHealth tracks node health status
type NodeHealth struct {
	IsHealthy     bool
	LastCheck     time.Time
	FailureCount  int
	AvailableSize int64
}

// ReplicationResult contains result of replication operation
type ReplicationResult struct {
	Success          bool
	ReplicatedNodes  []string
	FailedNodes      []string
	Checksum         string
	ReplicationTime  time.Duration
	RequiredReplicas int
	ActualReplicas   int
}

var (
	replicationService *ReplicationService
	once               sync.Once
	log                *logger.Logger
)

// InitReplicationService initializes the replication service (singleton)
func InitReplicationService(l *logger.Logger) *ReplicationService {
	once.Do(func() {
		log = l
		config.LoadEnvOnce()

		nodes := parseStorageNodes()
		replicationService = &ReplicationService{
			storageNodes: nodes,
			logger:       l,
			healthStatus: make(map[string]NodeHealth),
		}

		// Initialize health status
		for _, node := range nodes {
			replicationService.healthStatus[node.ID] = NodeHealth{
				IsHealthy:    true,
				LastCheck:    time.Now(),
				FailureCount: 0,
			}
		}

		log.Printf("Replication service initialized with %d storage nodes", len(nodes))

		// Start health checker
		go replicationService.startHealthChecker()
	})

	return replicationService
}

// InitReplicationServiceWithDB initializes replication service with database support
func InitReplicationServiceWithDB(l *logger.Logger, db *sql.DB, tenantID string) *ReplicationService {
	rs := &ReplicationService{
		logger:       l,
		mutex:        sync.RWMutex{},
		healthStatus: make(map[string]NodeHealth),
		db:           db,
		tenantID:     tenantID,
	}

	// Load nodes from database
	rs.loadNodesFromDB()

	return rs
}

// GetInstance returns the singleton replication service
func GetInstance() *ReplicationService {
	if replicationService == nil {
		log.Printf("WARNING: ReplicationService not initialized, initializing now")
		return InitReplicationService(log)
	}
	return replicationService
}

// parseStorageNodes parses storage nodes from environment
func parseStorageNodes() []StorageNode {
	bucketsEnv := config.GetEnvWithFallback("REPLICATION_BUCKETS", "")
	if bucketsEnv == "" {
		// Default to 3-way replication
		return []StorageNode{
			{ID: "node1", Path: "/storage/ssd1/securestor", Priority: 1},
			{ID: "node2", Path: "/storage/ssd2/securestor", Priority: 2},
			{ID: "node3", Path: "/storage/ssd3/securestor", Priority: 3},
		}
	}

	paths := filepath.SplitList(bucketsEnv)
	nodes := make([]StorageNode, 0, len(paths))

	for i, path := range paths {
		nodes = append(nodes, StorageNode{
			ID:       fmt.Sprintf("node%d", i+1),
			Path:     path,
			Priority: i + 1,
		})
	}

	return nodes
}

// loadNodesFromDB loads replication nodes from database
func (rs *ReplicationService) loadNodesFromDB() {
	if rs.db == nil {
		rs.logger.Printf("WARNING: No database connection, using default local nodes")
		rs.storageNodes = parseStorageNodes()
		return
	}

	query := `
		SELECT 
			id, node_name, node_path, priority, node_type,
			s3_endpoint, s3_region, s3_bucket, s3_access_key, s3_secret_key,
			s3_use_ssl, s3_path_prefix
		FROM replication_nodes
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY priority ASC
	`

	rows, err := rs.db.Query(query, rs.tenantID)
	if err != nil {
		rs.logger.Printf("ERROR: Failed to load nodes from DB: %v", err)
		rs.storageNodes = parseStorageNodes()
		return
	}
	defer rows.Close()

	nodes := []StorageNode{}
	for rows.Next() {
		var node StorageNode
		var nodeType, s3Endpoint, s3Region, s3Bucket, s3AccessKey, s3SecretKey, s3PathPrefix sql.NullString
		var s3UseSSL sql.NullBool

		err := rows.Scan(
			&node.ID, &node.NodeName, &node.Path, &node.Priority, &nodeType,
			&s3Endpoint, &s3Region, &s3Bucket, &s3AccessKey, &s3SecretKey,
			&s3UseSSL, &s3PathPrefix,
		)
		if err != nil {
			rs.logger.Printf("ERROR: Failed to scan node: %v", err)
			continue
		}

		node.NodeType = nodeType.String
		if node.NodeType == "" {
			node.NodeType = "local"
		}

		node.S3Endpoint = s3Endpoint.String
		node.S3Region = s3Region.String
		node.S3Bucket = s3Bucket.String
		node.S3AccessKey = s3AccessKey.String
		node.S3SecretKey = s3SecretKey.String
		node.S3UseSSL = s3UseSSL.Bool
		node.S3PathPrefix = s3PathPrefix.String

		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		rs.logger.Printf("WARNING: No nodes found in DB, using default local nodes")
		rs.storageNodes = parseStorageNodes()
		return
	}

	rs.storageNodes = nodes
	rs.logger.Printf("Loaded %d replication nodes from database", len(nodes))

	// Initialize health status
	for _, node := range nodes {
		rs.healthStatus[node.ID] = NodeHealth{
			IsHealthy:    true,
			LastCheck:    time.Now(),
			FailureCount: 0,
		}
	}
}

// ReplicateFile performs 3-way synchronous replication
func (rs *ReplicationService) ReplicateFile(ctx context.Context, bucketName, filename string, fileBytes []byte) (*ReplicationResult, error) {
	// Reload nodes from DB if DB is available
	if rs.db != nil {
		rs.loadNodesFromDB()
	}
	start := time.Now()
	result := &ReplicationResult{
		ReplicatedNodes:  []string{},
		FailedNodes:      []string{},
		RequiredReplicas: 2, // Need at least 2/3 replicas
	}

	// Calculate checksum
	hash := md5.Sum(fileBytes)
	result.Checksum = fmt.Sprintf("%x", hash)

	// Replicate to all healthy nodes in parallel
	replicaChan := make(chan string, len(rs.storageNodes))
	var wg sync.WaitGroup

	for _, node := range rs.storageNodes {
		wg.Add(1)
		go func(n StorageNode) {
			defer wg.Done()

			if err := rs.replicateToNode(ctx, n, bucketName, filename, fileBytes); err != nil {
				result.FailedNodes = append(result.FailedNodes, n.ID)
				replicaChan <- ""
				rs.recordNodeFailure(n.ID)
				rs.logger.Printf("ERROR: Failed to replicate to %s: %v", n.ID, err)
			} else {
				result.ReplicatedNodes = append(result.ReplicatedNodes, n.ID)
				replicaChan <- n.ID
				rs.recordNodeSuccess(n.ID)
			}
		}(node)
	}

	wg.Wait()
	close(replicaChan)

	// Count successful replications
	successCount := 0
	for range replicaChan {
		successCount++
	}
	result.ActualReplicas = successCount

	// Check if we have minimum required replicas
	if result.ActualReplicas < result.RequiredReplicas {
		result.Success = false
		return result, fmt.Errorf("insufficient replicas: got %d, need %d", result.ActualReplicas, result.RequiredReplicas)
	}

	result.Success = true
	result.ReplicationTime = time.Since(start)

	rs.logger.Printf("Replication successful: file=%s, replicas=%d, time=%v", filename, result.ActualReplicas, result.ReplicationTime)
	return result, nil
}

// replicateToNode replicates file to a specific node
func (rs *ReplicationService) replicateToNode(ctx context.Context, node StorageNode, bucketName, filename string, fileBytes []byte) error {
	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Route to appropriate replication method based on node type
	switch node.NodeType {
	case "s3", "minio":
		return rs.replicateToS3(ctx, node, bucketName, filename, fileBytes)
	case "local", "":
		return rs.replicateToLocal(ctx, node, bucketName, filename, fileBytes)
	default:
		return fmt.Errorf("unsupported node type: %s", node.NodeType)
	}
}

// replicateToLocal replicates file to local filesystem
func (rs *ReplicationService) replicateToLocal(ctx context.Context, node StorageNode, bucketName, filename string, fileBytes []byte) error {
	// Create directory structure
	replicaPath := filepath.Join(node.Path, bucketName)
	if err := os.MkdirAll(replicaPath, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file atomically (write to temp, then rename)
	fullPath := filepath.Join(replicaPath, filename)
	tempPath := fullPath + ".tmp"

	// Write to temporary file
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := file.Write(fileBytes); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Sync to disk for durability
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to sync file: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tempPath, fullPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// replicateToS3 replicates file to S3-compatible storage (S3, MinIO, etc.)
func (rs *ReplicationService) replicateToS3(ctx context.Context, node StorageNode, bucketName, filename string, fileBytes []byte) error {
	// Create S3 client
	client, err := rs.createS3Client(ctx, node)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Build S3 key (path)
	key := filename
	if node.S3PathPrefix != "" {
		key = filepath.Join(node.S3PathPrefix, bucketName, filename)
	} else {
		key = filepath.Join(bucketName, filename)
	}

	// Upload to S3
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(node.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileBytes),
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	rs.logger.Printf("Successfully replicated to S3: bucket=%s, key=%s, size=%d", node.S3Bucket, key, len(fileBytes))
	return nil
}

// createS3Client creates an S3 client for the given node
func (rs *ReplicationService) createS3Client(ctx context.Context, node StorageNode) (*s3.Client, error) {
	// Create custom endpoint resolver for MinIO/S3-compatible storage
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if node.S3Endpoint != "" {
			return aws.Endpoint{
				URL:               node.S3Endpoint,
				HostnameImmutable: true,
				Source:            aws.EndpointSourceCustom,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	// Build AWS config
	var cfg aws.Config
	var err error

	if node.S3AccessKey != "" && node.S3SecretKey != "" {
		// Use provided credentials
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(node.S3Region),
			awsconfig.WithEndpointResolverWithOptions(customResolver),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				node.S3AccessKey,
				node.S3SecretKey,
				"",
			)),
		)
	} else {
		// Use default credentials (IAM role, env vars, etc.)
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(node.S3Region),
			awsconfig.WithEndpointResolverWithOptions(customResolver),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with path-style addressing for MinIO compatibility
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for MinIO
	})

	return client, nil
}

// ReadFile reads from the fastest healthy node
func (rs *ReplicationService) ReadFile(ctx context.Context, bucketName, filename string) ([]byte, error) {
	// Sort nodes by health and priority
	rs.mutex.RLock()
	healthyNodes := make([]StorageNode, 0)
	for _, node := range rs.storageNodes {
		if health, ok := rs.healthStatus[node.ID]; ok && health.IsHealthy {
			healthyNodes = append(healthyNodes, node)
		}
	}
	rs.mutex.RUnlock()

	if len(healthyNodes) == 0 {
		return nil, fmt.Errorf("no healthy nodes available")
	}

	// Try to read from each healthy node
	for _, node := range healthyNodes {
		filePath := filepath.Join(node.Path, bucketName, filename)

		data, err := os.ReadFile(filePath)
		if err == nil {
			return data, nil
		}

		rs.logger.Printf("DEBUG: Failed to read from %s: %v", node.ID, err)
	}

	return nil, fmt.Errorf("could not read file from any node")
}

// VerifyReplication checks data consistency across replicas
func (rs *ReplicationService) VerifyReplication(bucketName, filename string) (*ReplicationResult, error) {
	result := &ReplicationResult{
		ReplicatedNodes: []string{},
		FailedNodes:     []string{},
	}

	checksums := make(map[string]string)

	// Calculate checksum on each node
	for _, node := range rs.storageNodes {
		filePath := filepath.Join(node.Path, bucketName, filename)

		data, err := os.ReadFile(filePath)
		if err != nil {
			result.FailedNodes = append(result.FailedNodes, node.ID)
			continue
		}

		hash := md5.Sum(data)
		checksum := fmt.Sprintf("%x", hash)
		checksums[node.ID] = checksum
		result.ReplicatedNodes = append(result.ReplicatedNodes, node.ID)
	}

	// Verify all checksums match
	if len(checksums) == 0 {
		result.Success = false
		return result, fmt.Errorf("no replicas found")
	}

	firstChecksum := ""
	for _, checksum := range checksums {
		if firstChecksum == "" {
			firstChecksum = checksum
			result.Checksum = checksum
		} else if checksum != firstChecksum {
			result.Success = false
			return result, fmt.Errorf("checksum mismatch detected")
		}
	}

	result.Success = true
	result.ActualReplicas = len(result.ReplicatedNodes)

	return result, nil
}

// startHealthChecker periodically checks node health
func (rs *ReplicationService) startHealthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, node := range rs.storageNodes {
			go rs.checkNodeHealth(node)
		}
	}
}

// checkNodeHealth verifies a node is accessible and has space
func (rs *ReplicationService) checkNodeHealth(node StorageNode) {
	// Check if path exists and is writable
	testFile := filepath.Join(node.Path, ".health_check")

	err := os.WriteFile(testFile, []byte("health_check"), 0600)
	if err != nil {
		rs.recordNodeFailure(node.ID)
		return
	}

	os.Remove(testFile)

	// Check available space
	var stat os.FileInfo
	if stat, err = os.Stat(node.Path); err != nil || !stat.IsDir() {
		rs.recordNodeFailure(node.ID)
		return
	}

	rs.recordNodeSuccess(node.ID)
}

// recordNodeSuccess records successful health check
func (rs *ReplicationService) recordNodeSuccess(nodeID string) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	health := rs.healthStatus[nodeID]
	health.IsHealthy = true
	health.LastCheck = time.Now()
	health.FailureCount = 0
	rs.healthStatus[nodeID] = health
}

// recordNodeFailure records failed health check
func (rs *ReplicationService) recordNodeFailure(nodeID string) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	health := rs.healthStatus[nodeID]
	health.FailureCount++

	// Mark unhealthy after 3 consecutive failures
	if health.FailureCount >= 3 {
		health.IsHealthy = false
	}

	health.LastCheck = time.Now()
	rs.healthStatus[nodeID] = health
}

// GetHealthStatus returns health status of all nodes
func (rs *ReplicationService) GetHealthStatus() map[string]NodeHealth {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	// Return a copy
	status := make(map[string]NodeHealth)
	for k, v := range rs.healthStatus {
		status[k] = v
	}

	return status
}

// Deprecated: Use GetInstance().ReplicateFile instead
func ReplicateFile(bucketName, filename string, fileBytes []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if replicationService != nil {
		replicationService.ReplicateFile(ctx, bucketName, filename, fileBytes)
	}
}
