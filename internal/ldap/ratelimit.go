package ldap

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens     float64   // Current number of tokens
	maxTokens  float64   // Maximum tokens (burst size)
	refillRate float64   // Tokens added per second
	lastRefill time.Time // Last time tokens were refilled
	mu         sync.Mutex
	enabled    bool
}

// RateLimitConfig represents rate limiter configuration
type RateLimitConfig struct {
	Enabled       bool    // Enable/disable rate limiting
	QueriesPerSec float64 // Maximum queries per second
	BurstSize     int     // Maximum burst size (queries that can be done instantly)
}

// DefaultRateLimitConfig returns sensible defaults for rate limiting
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:       true,
		QueriesPerSec: 10.0, // 10 queries per second
		BurstSize:     20,   // Allow bursts of up to 20 queries
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if !config.Enabled {
		return &RateLimiter{enabled: false}
	}

	return &RateLimiter{
		tokens:     float64(config.BurstSize),
		maxTokens:  float64(config.BurstSize),
		refillRate: config.QueriesPerSec,
		lastRefill: time.Now(),
		enabled:    true,
	}
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	// Add tokens based on elapsed time
	tokensToAdd := elapsed * rl.refillRate
	rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
	rl.lastRefill = now
}

// Wait blocks until a token is available or context is cancelled
// Returns error if context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if !rl.enabled {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for {
		rl.refill()

		// If we have tokens available, consume one and return
		if rl.tokens >= 1.0 {
			rl.tokens -= 1.0
			return nil
		}

		// Calculate wait time until next token is available
		tokensNeeded := 1.0 - rl.tokens
		waitTime := time.Duration(tokensNeeded/rl.refillRate*1000) * time.Millisecond

		// Limit max wait time to prevent excessive blocking
		if waitTime > 5*time.Second {
			waitTime = 5 * time.Second
		}

		// Release lock while waiting
		rl.mu.Unlock()

		// Wait with context support
		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			rl.mu.Lock()
			return ctx.Err()
		case <-timer.C:
			// Continue to next iteration
		}

		rl.mu.Lock()
	}
}

// TryAcquire attempts to acquire a token without blocking
// Returns true if token was acquired, false otherwise
func (rl *RateLimiter) TryAcquire() bool {
	if !rl.enabled {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}

// Stats returns current rate limiter statistics
type RateLimitStats struct {
	Enabled         bool
	AvailableTokens float64
	MaxTokens       float64
	RefillRate      float64
}

// Stats returns current rate limiter statistics
func (rl *RateLimiter) Stats() RateLimitStats {
	if !rl.enabled {
		return RateLimitStats{Enabled: false}
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	return RateLimitStats{
		Enabled:         true,
		AvailableTokens: rl.tokens,
		MaxTokens:       rl.maxTokens,
		RefillRate:      rl.refillRate,
	}
}

// RateLimitedService wraps a CachedService with rate limiting
type RateLimitedService struct {
	*CachedService
	limiter *RateLimiter
}

// NewRateLimitedService creates a new service with rate limiting
func NewRateLimitedService(config *Config, poolConfig PoolConfig, cacheConfig CacheConfig, rateLimitConfig RateLimitConfig) (*RateLimitedService, error) {
	cachedService, err := NewCachedService(config, poolConfig, cacheConfig)
	if err != nil {
		return nil, err
	}

	return &RateLimitedService{
		CachedService: cachedService,
		limiter:       NewRateLimiter(rateLimitConfig),
	}, nil
}

// withRateLimit wraps an operation with rate limiting
func (rls *RateLimitedService) withRateLimit(ctx context.Context, operation func() error) error {
	if err := rls.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit wait cancelled: %w", err)
	}
	return operation()
}

// SearchUser searches for users with rate limiting
func (rls *RateLimitedService) SearchUser(query string) ([]*UserInfo, error) {
	var users []*UserInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		users, err = rls.CachedService.SearchUser(query)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return users, err
}

// GetUserDetails gets user details with rate limiting
func (rls *RateLimitedService) GetUserDetails(identifier string) (*UserInfo, error) {
	var user *UserInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		user, err = rls.CachedService.GetUserDetails(identifier)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return user, err
}

// SearchGroup searches for groups with rate limiting
func (rls *RateLimitedService) SearchGroup(query string) ([]*GroupInfo, error) {
	var groups []*GroupInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		groups, err = rls.CachedService.SearchGroup(query)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return groups, err
}

// GetGroupMembers gets group members with rate limiting
func (rls *RateLimitedService) GetGroupMembers(groupDN string) ([]*UserInfo, error) {
	var members []*UserInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		members, err = rls.CachedService.GetGroupMembers(groupDN)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return members, err
}

// VerifyMembership verifies membership with rate limiting
func (rls *RateLimitedService) VerifyMembership(userIdentifier, groupIdentifier string) (bool, error) {
	var result bool
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		result, err = rls.CachedService.VerifyMembership(userIdentifier, groupIdentifier)
		return err
	})

	if opErr != nil {
		return false, opErr
	}

	return result, err
}

// SearchByFilter searches with filter with rate limiting
func (rls *RateLimitedService) SearchByFilter(filter, baseDN string, attributes []string) ([]*SearchResult, error) {
	var results []*SearchResult
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		results, err = rls.CachedService.SearchByFilter(filter, baseDN, attributes)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return results, err
}

// GetUserGroups gets user groups with rate limiting
func (rls *RateLimitedService) GetUserGroups(userIdentifier string) ([]*GroupInfo, error) {
	var groups []*GroupInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		groups, err = rls.CachedService.GetUserGroups(userIdentifier)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return groups, err
}

// SearchOU searches for OUs with rate limiting
func (rls *RateLimitedService) SearchOU(query string) ([]*OUInfo, error) {
	var ous []*OUInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		ous, err = rls.CachedService.SearchOU(query)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return ous, err
}

// GetComputer gets computer info with rate limiting
func (rls *RateLimitedService) GetComputer(name string) (*ComputerInfo, error) {
	var computer *ComputerInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		computer, err = rls.CachedService.GetComputer(name)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return computer, err
}

// BulkUserLookup performs bulk user lookup with rate limiting
func (rls *RateLimitedService) BulkUserLookup(identifiers []string) ([]*UserInfo, error) {
	var users []*UserInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		users, err = rls.CachedService.BulkUserLookup(identifiers)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return users, err
}

// GetDirectReports gets direct reports with rate limiting
func (rls *RateLimitedService) GetDirectReports(managerIdentifier string) ([]*UserInfo, error) {
	var reports []*UserInfo
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		reports, err = rls.CachedService.GetDirectReports(managerIdentifier)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return reports, err
}

// SearchByAttributes searches by attributes with rate limiting
func (rls *RateLimitedService) SearchByAttributes(attributes map[string]string, objectClass string) ([]*SearchResult, error) {
	var results []*SearchResult
	var err error

	ctx := context.Background()
	opErr := rls.withRateLimit(ctx, func() error {
		results, err = rls.CachedService.SearchByAttributes(attributes, objectClass)
		return err
	})

	if opErr != nil {
		return nil, opErr
	}

	return results, err
}

// RateLimitStats returns rate limiter statistics
func (rls *RateLimitedService) RateLimitStats() RateLimitStats {
	return rls.limiter.Stats()
}

// Helper function (Go 1.21+)
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
