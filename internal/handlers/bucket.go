package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/securestor/securestor/internal/config"
)

var storageDir string

// List of buckets for replication
var replicationBuckets []string

func init() {
	// Use the centralized environment loader
	config.LoadEnvOnce()

	storageDir = config.GetEnvWithFallback("STORAGE_DIR", "storage")

	bucketsEnv := config.GetEnvWithFallback("REPLICATION_BUCKETS", "")
	if bucketsEnv == "" {
		replicationBuckets = []string{"replica1", "replica2"}
	} else {
		replicationBuckets = filepath.SplitList(bucketsEnv)
	}
}

func CreateBucketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	bucketName := r.URL.Query().Get("bucket")
	if bucketName == "" {
		http.Error(w, "Bucket name is required", http.StatusBadRequest)
		return
	}

	bucketPath := filepath.Join(storageDir, bucketName)
	if err := os.MkdirAll(bucketPath, os.ModePerm); err != nil {
		http.Error(w, "Failed to create bucket", http.StatusInternalServerError)
		return
	}

	// Replicate the bucket
	for _, replica := range replicationBuckets {
		replicaPath := filepath.Join(replica, bucketName)
		if err := os.MkdirAll(replicaPath, os.ModePerm); err != nil {
			http.Error(w, "Failed to create replica bucket", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Bucket created successfully: %s\n", bucketName)
}
