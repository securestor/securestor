package api

import (
	"fmt"
	"testing"
	"time"
)

// TestProxyCacheL1Hit tests L1 (Redis) cache hit scenario
func TestProxyCacheL1Hit(t *testing.T) {
	l2Storage, _ := NewL2StorageImpl("/tmp/test-l2", 0.01)
	testData := []byte("maven-central:commons-lang:3.12.0")
	key := "apache/commons/commons-lang/3.12.0/commons-lang-3.12.0.jar"
	
	if err := l2Storage.Set(key, testData, time.Hour); err != nil {
		t.Fatalf("Failed to set L1 cache: %v", err)
	}
	
	cached, err := l2Storage.Get(key)
	if err != nil {
		t.Fatalf("Failed to get from L1 cache: %v", err)
	}
	
	if string(cached.([]byte)) != string(testData) {
		t.Errorf("L1 cache mismatch")
	}
	t.Log("✓ L1 cache hit test passed")
}

// TestProxyCacheLRUEviction tests LRU eviction when L2 cache is full
func TestProxyCacheLRUEviction(t *testing.T) {
	l2Storage, _ := NewL2StorageImpl("/tmp/test-lru", 0.01)
	
	key1 := "item1"
	data1 := make([]byte, 7*1024*1024)
	l2Storage.Set(key1, data1, 1*time.Hour)
	time.Sleep(100 * time.Millisecond)
	
	l2Storage.Get(key1)
	time.Sleep(100 * time.Millisecond)
	
	key2 := "item2"
	data2 := make([]byte, 5*1024*1024)
	l2Storage.Set(key2, data2, 1*time.Hour)
	time.Sleep(100 * time.Millisecond)
	
	key3 := "item3"
	data3 := make([]byte, 4*1024*1024)
	l2Storage.Set(key3, data3, 1*time.Hour)
	
	_, err := l2Storage.Get(key2)
	if err == nil {
		t.Errorf("Expected item2 to be evicted")
	}
	
	if _, err := l2Storage.Get(key1); err != nil {
		t.Errorf("Item1 should exist after LRU eviction")
	}
	t.Log("✓ LRU eviction test passed")
}

// TestProxyCacheTTLExpiration tests TTL expiration
func TestProxyCacheTTLExpiration(t *testing.T) {
	l2Storage, _ := NewL2StorageImpl("/tmp/test-ttl", 1.0)
	key := "short-lived"
	data := []byte("temporary data")
	l2Storage.Set(key, data, 100*time.Millisecond)
	
	if _, err := l2Storage.Get(key); err != nil {
		t.Fatalf("Item should exist immediately")
	}
	
	time.Sleep(150 * time.Millisecond)
	
	_, err := l2Storage.Get(key)
	if err == nil {
		t.Errorf("Item should be expired after TTL")
	}
	t.Log("✓ TTL expiration test passed")
}

// TestProxyCleanup tests cleanup of expired entries
func TestProxyCleanup(t *testing.T) {
	l2Storage, _ := NewL2StorageImpl("/tmp/test-cleanup", 1.0)
	
	l2Storage.Set("expired", []byte("expires"), 100*time.Millisecond)
	l2Storage.Set("long-lived", []byte("stays"), 24*time.Hour)
	
	time.Sleep(150 * time.Millisecond)
	l2Storage.Cleanup()
	
	if _, err := l2Storage.Get("expired"); err == nil {
		t.Errorf("Expired entry should be cleaned up")
	}
	
	if _, err := l2Storage.Get("long-lived"); err != nil {
		t.Errorf("Long-lived entry should still exist")
	}
	t.Log("✓ Cleanup test passed")
}

// TestProxyStorageStats tests statistics collection
func TestProxyStorageStats(t *testing.T) {
	l2Storage, _ := NewL2StorageImpl("/tmp/test-stats", 1.0)
	
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("artifact-%d", i)
		l2Storage.Set(key, []byte(key), time.Hour)
	}
	
	stats := l2Storage.Stats()
	if stats["entry_count"] != 5 {
		t.Errorf("Expected 5 entries, got %v", stats["entry_count"])
	}
	t.Log("✓ Storage stats test passed")
}

// BenchmarkProxyL2CacheAccess benchmarks L2 cache performance
func BenchmarkProxyL2CacheAccess(b *testing.B) {
	l2Storage, _ := NewL2StorageImpl("/tmp/bench-l2", 1.0)
	key := "benchmark-key"
	data := make([]byte, 1024*1024)
	l2Storage.Set(key, data, time.Hour)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l2Storage.Get(key)
	}
}
