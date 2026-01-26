package ldap

import (
	"testing"
	"time"
)

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	if !config.Enabled {
		t.Error("Cache should be enabled by default")
	}

	if config.DefaultTTL <= 0 {
		t.Error("DefaultTTL should be positive")
	}

	if config.CleanupInterval <= 0 {
		t.Error("CleanupInterval should be positive")
	}

	if config.MaxEntries < 0 {
		t.Error("MaxEntries should be non-negative")
	}
}

func TestCacheBasicOperations(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    1 * time.Second,
		CleanupInterval: 100 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)
	defer cache.Clear()

	// Test Set and Get
	cache.Set("test_op", "test_value", "param1")

	result := cache.Get("test_op", "param1")
	if result == nil {
		t.Error("Expected cached value, got nil")
	}

	if result.(string) != "test_value" {
		t.Errorf("Expected 'test_value', got %v", result)
	}

	// Test cache miss
	result = cache.Get("test_op", "different_param")
	if result != nil {
		t.Error("Expected cache miss, got value")
	}

	// Test different operation
	result = cache.Get("different_op", "param1")
	if result != nil {
		t.Error("Expected cache miss for different operation")
	}
}

func TestCacheExpiration(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)
	defer cache.Clear()

	cache.Set("test_op", "test_value", "param1")

	// Should be cached immediately
	result := cache.Get("test_op", "param1")
	if result == nil {
		t.Error("Expected cached value immediately after Set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	result = cache.Get("test_op", "param1")
	if result != nil {
		t.Error("Expected expired value to return nil")
	}
}

func TestCacheInvalidate(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    1 * time.Minute,
		CleanupInterval: 100 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)
	defer cache.Clear()

	cache.Set("test_op", "test_value", "param1")

	// Verify it's cached
	result := cache.Get("test_op", "param1")
	if result == nil {
		t.Error("Expected cached value")
	}

	// Invalidate
	cache.Invalidate("test_op", "param1")

	// Should be gone
	result = cache.Get("test_op", "param1")
	if result != nil {
		t.Error("Expected invalidated value to return nil")
	}
}

func TestCacheClear(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    1 * time.Minute,
		CleanupInterval: 100 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)

	// Add multiple entries
	cache.Set("op1", "value1", "param1")
	cache.Set("op2", "value2", "param2")
	cache.Set("op3", "value3", "param3")

	stats := cache.Stats()
	if stats.Entries != 3 {
		t.Errorf("Expected 3 entries, got %d", stats.Entries)
	}

	// Clear
	cache.Clear()

	stats = cache.Stats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.Entries)
	}

	// Verify entries are gone
	result := cache.Get("op1", "param1")
	if result != nil {
		t.Error("Expected cleared cache to return nil")
	}
}

func TestCacheCustomTTL(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    1 * time.Second,
		CleanupInterval: 100 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)
	defer cache.Clear()

	// Set with custom short TTL
	cache.SetWithTTL("test_op", "test_value", 50*time.Millisecond, "param1")

	// Should be cached immediately
	result := cache.Get("test_op", "param1")
	if result == nil {
		t.Error("Expected cached value")
	}

	// Wait for custom TTL to expire (but less than default TTL)
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	result = cache.Get("test_op", "param1")
	if result != nil {
		t.Error("Expected value with custom TTL to expire")
	}
}

func TestDisabledCache(t *testing.T) {
	config := CacheConfig{
		Enabled: false,
	}

	cache := NewCache(config)

	// Operations on disabled cache should be no-ops
	cache.Set("test_op", "test_value", "param1")

	result := cache.Get("test_op", "param1")
	if result != nil {
		t.Error("Disabled cache should always return nil")
	}

	stats := cache.Stats()
	if stats.Enabled {
		t.Error("Cache stats should show disabled")
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    1 * time.Minute,
		CleanupInterval: 100 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)
	defer cache.Clear()

	// Same operation, same params - should hit same cache entry
	cache.Set("test_op", "value1", "param1", "param2")
	result := cache.Get("test_op", "param1", "param2")
	if result == nil {
		t.Error("Expected cache hit with same params")
	}

	// Same operation, different param order - should be different entry
	// (This tests that parameter order matters)
	result = cache.Get("test_op", "param2", "param1")
	if result != nil {
		t.Error("Expected cache miss with different param order")
	}

	// Different number of params - should be different entry
	result = cache.Get("test_op", "param1")
	if result != nil {
		t.Error("Expected cache miss with different number of params")
	}
}

func TestCacheStats(t *testing.T) {
	config := CacheConfig{
		Enabled:       true,
		DefaultTTL:    5 * time.Minute,
		CleanupInterval: 100 * time.Millisecond,
		MaxEntries:    10,
	}

	cache := NewCache(config)
	defer cache.Clear()

	stats := cache.Stats()
	if !stats.Enabled {
		t.Error("Stats should show cache as enabled")
	}

	if stats.Entries != 0 {
		t.Error("New cache should have 0 entries")
	}

	if stats.TTL != config.DefaultTTL {
		t.Errorf("Expected TTL %v, got %v", config.DefaultTTL, stats.TTL)
	}

	// Add entries
	cache.Set("op1", "val1", "p1")
	cache.Set("op2", "val2", "p2")

	stats = cache.Stats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.Entries)
	}
}
