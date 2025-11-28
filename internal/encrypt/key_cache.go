package encrypt

import (
	"sync"
	"time"
)

// KeyCache provides an in-memory cache for encryption keys with TTL
type KeyCache struct {
	cache map[string]*cachedKey
	mutex sync.RWMutex
	ttl   time.Duration
}

// cachedKey represents a cached encryption key with expiration
type cachedKey struct {
	key       []byte
	expiresAt time.Time
}

// NewKeyCache creates a new key cache with specified TTL in seconds
func NewKeyCache(ttlSeconds int) *KeyCache {
	cache := &KeyCache{
		cache: make(map[string]*cachedKey),
		ttl:   time.Duration(ttlSeconds) * time.Second,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a key from cache
func (kc *KeyCache) Get(keyID string) ([]byte, bool) {
	kc.mutex.RLock()
	defer kc.mutex.RUnlock()

	cached, exists := kc.cache[keyID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.expiresAt) {
		return nil, false
	}

	// Return copy to prevent modification
	keyCopy := make([]byte, len(cached.key))
	copy(keyCopy, cached.key)
	return keyCopy, true
}

// Set stores a key in cache
func (kc *KeyCache) Set(keyID string, key []byte) {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	// Store copy to prevent external modification
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	kc.cache[keyID] = &cachedKey{
		key:       keyCopy,
		expiresAt: time.Now().Add(kc.ttl),
	}
}

// Delete removes a key from cache
func (kc *KeyCache) Delete(keyID string) {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	// Zero out key before deleting
	if cached, exists := kc.cache[keyID]; exists {
		zeroBytes(cached.key)
		delete(kc.cache, keyID)
	}
}

// Clear removes all keys from cache
func (kc *KeyCache) Clear() {
	kc.mutex.Lock()
	defer kc.mutex.Unlock()

	// Zero out all keys
	for _, cached := range kc.cache {
		zeroBytes(cached.key)
	}

	kc.cache = make(map[string]*cachedKey)
}

// cleanup periodically removes expired keys
func (kc *KeyCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		kc.mutex.Lock()
		now := time.Now()
		for keyID, cached := range kc.cache {
			if now.After(cached.expiresAt) {
				zeroBytes(cached.key)
				delete(kc.cache, keyID)
			}
		}
		kc.mutex.Unlock()
	}
}

// Size returns the number of keys in cache
func (kc *KeyCache) Size() int {
	kc.mutex.RLock()
	defer kc.mutex.RUnlock()
	return len(kc.cache)
}
