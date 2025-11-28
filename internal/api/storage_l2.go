package api

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// L2StorageImpl implements local disk-based caching with LRU eviction and TTL management
type L2StorageImpl struct {
	basePath string
	maxSize  int64 // Maximum total size in bytes
	mu       sync.RWMutex
	entries  map[string]*L2Entry
	usage    int64 // Current total usage in bytes
}

// L2Entry represents metadata for cached items
type L2Entry struct {
	Key        string
	Data       []byte
	Size       int64
	CreatedAt  time.Time
	AccessedAt time.Time
	TTL        time.Duration
	Hash       string
}

// NewL2StorageImpl creates a new L2 storage instance
func NewL2StorageImpl(basePath string, maxSizeGB float64) (*L2StorageImpl, error) {
	maxSizeBytes := int64(maxSizeGB * 1024 * 1024 * 1024)

	// Create base directory if not exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create L2 storage directory: %w", err)
	}

	l2 := &L2StorageImpl{
		basePath: basePath,
		maxSize:  maxSizeBytes,
		entries:  make(map[string]*L2Entry),
		usage:    0,
	}

	// Load existing entries from disk
	if err := l2.loadEntriesFromDisk(); err != nil {
		// Log warning but don't fail initialization
		fmt.Printf("warning: failed to load L2 entries from disk: %v\n", err)
	}

	return l2, nil
}

// Get retrieves data from L2 storage
func (l2 *L2StorageImpl) Get(key string) (interface{}, error) {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	entry, exists := l2.entries[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check TTL
	if entry.TTL > 0 && time.Since(entry.CreatedAt) > entry.TTL {
		// Entry expired, remove it
		if err := l2.removeEntry(entry); err != nil {
			fmt.Printf("warning: failed to remove expired entry: %v\n", err)
		}
		delete(l2.entries, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}

	// Update access time for LRU tracking
	entry.AccessedAt = time.Now()

	// Verify data integrity with hash
	if err := l2.verifyDataIntegrity(entry); err != nil {
		if err := l2.removeEntry(entry); err != nil {
			fmt.Printf("warning: failed to remove corrupted entry: %v\n", err)
		}
		delete(l2.entries, key)
		return nil, fmt.Errorf("data integrity check failed: %w", err)
	}

	return entry.Data, nil
}

// Set stores data in L2 storage with optional TTL
func (l2 *L2StorageImpl) Set(key string, data interface{}, ttl time.Duration) error {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	// Convert data to bytes
	dataBytes, err := l2.serializeData(data)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}

	size := int64(len(dataBytes))

	// Check if adding this would exceed max size
	if l2.usage+size > l2.maxSize {
		// Evict LRU entries until we have space
		if err := l2.evictLRU(l2.usage + size - l2.maxSize); err != nil {
			return fmt.Errorf("failed to evict entries: %w", err)
		}
	}

	// Create entry
	entry := &L2Entry{
		Key:        key,
		Data:       dataBytes,
		Size:       size,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		TTL:        ttl,
		Hash:       l2.hashData(dataBytes),
	}

	// Write to disk
	if err := l2.writeEntryToDisk(entry); err != nil {
		return fmt.Errorf("failed to write entry to disk: %w", err)
	}

	// Remove old entry if exists
	if oldEntry, exists := l2.entries[key]; exists {
		l2.usage -= oldEntry.Size
		if err := l2.removeEntry(oldEntry); err != nil {
			fmt.Printf("warning: failed to remove old entry: %v\n", err)
		}
	}

	// Add to in-memory index
	l2.entries[key] = entry
	l2.usage += size

	return nil
}

// Delete removes an entry from L2 storage
func (l2 *L2StorageImpl) Delete(key string) error {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	entry, exists := l2.entries[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	if err := l2.removeEntry(entry); err != nil {
		return fmt.Errorf("failed to remove entry: %w", err)
	}

	l2.usage -= entry.Size
	delete(l2.entries, key)

	return nil
}

// Stats returns storage statistics
func (l2 *L2StorageImpl) Stats() map[string]interface{} {
	l2.mu.RLock()
	defer l2.mu.RUnlock()

	return map[string]interface{}{
		"max_size_gb":  float64(l2.maxSize) / (1024 * 1024 * 1024),
		"used_size_gb": float64(l2.usage) / (1024 * 1024 * 1024),
		"entry_count":  len(l2.entries),
		"utilization":  float64(l2.usage) / float64(l2.maxSize) * 100,
	}
}

// Cleanup removes expired entries
func (l2 *L2StorageImpl) Cleanup() error {
	l2.mu.Lock()
	defer l2.mu.Unlock()

	expiredKeys := make([]string, 0)

	for key, entry := range l2.entries {
		if entry.TTL > 0 && time.Since(entry.CreatedAt) > entry.TTL {
			expiredKeys = append(expiredKeys, key)
			l2.usage -= entry.Size
			if err := l2.removeEntry(entry); err != nil {
				fmt.Printf("warning: failed to remove expired entry during cleanup: %v\n", err)
			}
		}
	}

	for _, key := range expiredKeys {
		delete(l2.entries, key)
	}

	return nil
}

// Private helper methods

// serializeData converts interface{} to bytes
func (l2 *L2StorageImpl) serializeData(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// deserializeData converts bytes back to interface{}
func (l2 *L2StorageImpl) deserializeData(data []byte) (interface{}, error) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var result interface{}
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// hashData creates MD5 hash of data for integrity verification
func (l2 *L2StorageImpl) hashData(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// verifyDataIntegrity checks if stored data matches hash
func (l2 *L2StorageImpl) verifyDataIntegrity(entry *L2Entry) error {
	currentHash := l2.hashData(entry.Data)
	if currentHash != entry.Hash {
		return fmt.Errorf("data hash mismatch: expected %s, got %s", entry.Hash, currentHash)
	}
	return nil
}

// writeEntryToDisk persists entry metadata and data
func (l2 *L2StorageImpl) writeEntryToDisk(entry *L2Entry) error {
	// Write data file
	dataPath := l2.getDataPath(entry.Key)
	if err := os.MkdirAll(filepath.Dir(dataPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(dataPath, entry.Data, 0644); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	// Write metadata file
	metaPath := l2.getMetaPath(entry.Key)
	metaData := fmt.Sprintf("%s|%d|%d|%s\n", entry.Hash, entry.CreatedAt.Unix(), entry.TTL.Milliseconds(), entry.Key)
	if err := os.WriteFile(metaPath, []byte(metaData), 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// removeEntry deletes entry files from disk
func (l2 *L2StorageImpl) removeEntry(entry *L2Entry) error {
	dataPath := l2.getDataPath(entry.Key)
	metaPath := l2.getMetaPath(entry.Key)

	// Remove files (ignore errors if they don't exist)
	_ = os.Remove(dataPath)
	_ = os.Remove(metaPath)

	return nil
}

// evictLRU removes least recently used entries to free up space
func (l2 *L2StorageImpl) evictLRU(spaceNeeded int64) error {
	// Create sorted list of entries by access time (oldest first)
	entries := make([]*L2Entry, 0, len(l2.entries))
	for _, entry := range l2.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AccessedAt.Before(entries[j].AccessedAt)
	})

	// Remove entries until we have enough space
	freedSpace := int64(0)
	for _, entry := range entries {
		if freedSpace >= spaceNeeded {
			break
		}

		if err := l2.removeEntry(entry); err != nil {
			fmt.Printf("warning: failed to remove entry during LRU eviction: %v\n", err)
		}

		l2.usage -= entry.Size
		freedSpace += entry.Size
		delete(l2.entries, entry.Key)
	}

	return nil
}

// loadEntriesFromDisk reads existing entries from disk (recovery on startup)
func (l2 *L2StorageImpl) loadEntriesFromDisk() error {
	// Scan for metadata files
	err := filepath.Walk(l2.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".meta" {
			// Parse metadata and reconstruct entry
			_, err := os.ReadFile(path)
			if err != nil {
				return nil // Skip corrupted files
			}

			// Extract key from path
			relPath, _ := filepath.Rel(l2.basePath, path)
			key := relPath[:len(relPath)-5] // Remove ".meta"

			// Read data file
			dataPath := l2.getDataPath(key)
			data, err := os.ReadFile(dataPath)
			if err != nil {
				return nil // Skip if data file missing
			}

			entry := &L2Entry{
				Key:        key,
				Data:       data,
				Size:       int64(len(data)),
				Hash:       l2.hashData(data),
				CreatedAt:  time.Now(),
				AccessedAt: time.Now(),
			}

			l2.entries[key] = entry
			l2.usage += entry.Size
		}

		return nil
	})

	return err
}

// getDataPath returns the file path for entry data
func (l2 *L2StorageImpl) getDataPath(key string) string {
	// Use hash of key to avoid path issues
	hash := md5.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(l2.basePath, hashStr[:2], hashStr[2:], "data")
}

// getMetaPath returns the file path for entry metadata
func (l2 *L2StorageImpl) getMetaPath(key string) string {
	hash := md5.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])
	return filepath.Join(l2.basePath, hashStr[:2], hashStr[2:], "meta")
}
