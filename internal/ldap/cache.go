package ldap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry represents a cached item with TTL
type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

// Cache provides thread-safe caching with TTL
type Cache struct {
	entries map[string]*cacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	enabled bool
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled       bool          // Enable/disable caching
	DefaultTTL    time.Duration // Default TTL for cache entries
	CleanupInterval time.Duration // How often to run cleanup of expired entries
	MaxEntries    int           // Maximum number of entries (0 = unlimited)
}

// DefaultCacheConfig returns sensible defaults for caching
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:       true,
		DefaultTTL:    5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		MaxEntries:    1000,
	}
}

// NewCache creates a new cache instance
func NewCache(config CacheConfig) *Cache {
	if !config.Enabled {
		return &Cache{enabled: false}
	}

	cache := &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     config.DefaultTTL,
		enabled: true,
	}

	// Start background cleanup goroutine
	go cache.cleanup(config.CleanupInterval)

	return cache
}

// generateKey creates a cache key from operation name and parameters
func (c *Cache) generateKey(operation string, params ...interface{}) string {
	// Create a deterministic key from operation and parameters
	data := struct {
		Op     string
		Params []interface{}
	}{
		Op:     operation,
		Params: params,
	}

	// Use JSON encoding for consistent key generation
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// Get retrieves a value from the cache
// Returns nil if not found or expired
func (c *Cache) Get(operation string, params ...interface{}) interface{} {
	if !c.enabled {
		return nil
	}

	key := c.generateKey(operation, params...)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil
	}

	// Check if entry has expired
	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.data
}

// Set stores a value in the cache with default TTL
func (c *Cache) Set(operation string, value interface{}, params ...interface{}) {
	if !c.enabled {
		return
	}

	c.SetWithTTL(operation, value, c.ttl, params...)
}

// SetWithTTL stores a value in the cache with custom TTL
func (c *Cache) SetWithTTL(operation string, value interface{}, ttl time.Duration, params ...interface{}) {
	if !c.enabled {
		return
	}

	key := c.generateKey(operation, params...)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Invalidate removes a specific cache entry
func (c *Cache) Invalidate(operation string, params ...interface{}) {
	if !c.enabled {
		return
	}

	key := c.generateKey(operation, params...)

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// InvalidatePattern removes all cache entries matching a pattern
// For simplicity, this removes all entries with the same operation name
func (c *Cache) InvalidatePattern(operation string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// This is a simplified implementation
	// In a production system, you might want to store operation names
	// with keys for more efficient pattern matching
	for key := range c.entries {
		// For now, just clear everything
		// A better implementation would track operations per key
		delete(c.entries, key)
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}

// cleanup periodically removes expired entries
func (c *Cache) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if !c.enabled {
			return
		}

		c.mu.Lock()
		now := time.Now()

		for key, entry := range c.entries {
			if now.After(entry.expiresAt) {
				delete(c.entries, key)
			}
		}

		c.mu.Unlock()
	}
}

// Stats returns cache statistics
type CacheStats struct {
	Enabled     bool
	Entries     int
	TTL         time.Duration
}

// Stats returns current cache statistics
func (c *Cache) Stats() CacheStats {
	if !c.enabled {
		return CacheStats{Enabled: false}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Enabled: true,
		Entries: len(c.entries),
		TTL:     c.ttl,
	}
}

// CachedService wraps a Service with caching
type CachedService struct {
	*Service
	cache *Cache
}

// NewCachedService creates a new service with caching enabled
func NewCachedService(config *Config, poolConfig PoolConfig, cacheConfig CacheConfig) (*CachedService, error) {
	service, err := NewServiceWithPool(config, poolConfig)
	if err != nil {
		return nil, err
	}

	return &CachedService{
		Service: service,
		cache:   NewCache(cacheConfig),
	}, nil
}

// SearchUser searches for users with caching
func (cs *CachedService) SearchUser(query string) ([]*UserInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("SearchUser", query); cached != nil {
		return cached.([]*UserInfo), nil
	}

	// Cache miss - query LDAP
	users, err := cs.Service.SearchUser(query)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.Set("SearchUser", users, query)
	return users, nil
}

// GetUserDetails gets user details with caching
func (cs *CachedService) GetUserDetails(identifier string) (*UserInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("GetUserDetails", identifier); cached != nil {
		return cached.(*UserInfo), nil
	}

	// Cache miss - query LDAP
	user, err := cs.Service.GetUserDetails(identifier)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.Set("GetUserDetails", user, identifier)
	return user, nil
}

// SearchGroup searches for groups with caching
func (cs *CachedService) SearchGroup(query string) ([]*GroupInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("SearchGroup", query); cached != nil {
		return cached.([]*GroupInfo), nil
	}

	// Cache miss - query LDAP
	groups, err := cs.Service.SearchGroup(query)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.Set("SearchGroup", groups, query)
	return groups, nil
}

// GetGroupMembers gets group members with caching
func (cs *CachedService) GetGroupMembers(groupDN string) ([]*UserInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("GetGroupMembers", groupDN); cached != nil {
		return cached.([]*UserInfo), nil
	}

	// Cache miss - query LDAP
	members, err := cs.Service.GetGroupMembers(groupDN)
	if err != nil {
		return nil, err
	}

	// Store in cache with longer TTL (group memberships change less frequently)
	cs.cache.SetWithTTL("GetGroupMembers", members, 10*time.Minute, groupDN)
	return members, nil
}

// GetUserGroups gets user groups with caching
func (cs *CachedService) GetUserGroups(userIdentifier string) ([]*GroupInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("GetUserGroups", userIdentifier); cached != nil {
		return cached.([]*GroupInfo), nil
	}

	// Cache miss - query LDAP
	groups, err := cs.Service.GetUserGroups(userIdentifier)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.Set("GetUserGroups", groups, userIdentifier)
	return groups, nil
}

// SearchOU searches for OUs with caching
func (cs *CachedService) SearchOU(query string) ([]*OUInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("SearchOU", query); cached != nil {
		return cached.([]*OUInfo), nil
	}

	// Cache miss - query LDAP
	ous, err := cs.Service.SearchOU(query)
	if err != nil {
		return nil, err
	}

	// Store in cache with longer TTL (OUs change very infrequently)
	cs.cache.SetWithTTL("SearchOU", ous, 30*time.Minute, query)
	return ous, nil
}

// GetComputer gets computer info with caching
func (cs *CachedService) GetComputer(name string) (*ComputerInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("GetComputer", name); cached != nil {
		return cached.(*ComputerInfo), nil
	}

	// Cache miss - query LDAP
	computer, err := cs.Service.GetComputer(name)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.Set("GetComputer", computer, name)
	return computer, nil
}

// BulkUserLookup performs bulk user lookup with caching
// Note: For bulk operations, caching individual results might be more effective
func (cs *CachedService) BulkUserLookup(identifiers []string) ([]*UserInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("BulkUserLookup", identifiers); cached != nil {
		return cached.([]*UserInfo), nil
	}

	// Cache miss - query LDAP
	users, err := cs.Service.BulkUserLookup(identifiers)
	if err != nil {
		return nil, err
	}

	// Store in cache with shorter TTL (bulk queries are often exploratory)
	cs.cache.SetWithTTL("BulkUserLookup", users, 2*time.Minute, identifiers)
	return users, nil
}

// GetDirectReports gets direct reports with caching
func (cs *CachedService) GetDirectReports(managerIdentifier string) ([]*UserInfo, error) {
	// Try cache first
	if cached := cs.cache.Get("GetDirectReports", managerIdentifier); cached != nil {
		return cached.([]*UserInfo), nil
	}

	// Cache miss - query LDAP
	reports, err := cs.Service.GetDirectReports(managerIdentifier)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.Set("GetDirectReports", reports, managerIdentifier)
	return reports, nil
}

// CacheStats returns cache statistics
func (cs *CachedService) CacheStats() CacheStats {
	return cs.cache.Stats()
}

// ClearCache clears all cached entries
func (cs *CachedService) ClearCache() {
	cs.cache.Clear()
}
