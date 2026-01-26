package ldap

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if !config.Enabled {
		t.Error("Rate limiting should be enabled by default")
	}

	if config.QueriesPerSec <= 0 {
		t.Error("QueriesPerSec should be positive")
	}

	if config.BurstSize <= 0 {
		t.Error("BurstSize should be positive")
	}
}

func TestRateLimiterBasicOperation(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 10.0,
		BurstSize:    5,
	}

	limiter := NewRateLimiter(config)

	// Should be able to acquire burst size tokens immediately
	ctx := context.Background()
	start := time.Now()

	for i := 0; i < config.BurstSize; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Failed to acquire token %d: %v", i+1, err)
		}
	}

	elapsed := time.Since(start)

	// Should be very fast (< 100ms)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Burst tokens took too long: %v", elapsed)
	}
}

func TestRateLimiterRefill(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 10.0, // 10 tokens per second = 100ms per token
		BurstSize:    2,
	}

	limiter := NewRateLimiter(config)
	ctx := context.Background()

	// Consume burst tokens
	limiter.Wait(ctx)
	limiter.Wait(ctx)

	// Next token should require waiting
	start := time.Now()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Failed to wait for token: %v", err)
	}
	elapsed := time.Since(start)

	// Should wait approximately 100ms (1 token at 10 qps)
	// Allow some tolerance (50ms - 200ms)
	if elapsed < 50*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Wait time unexpected: %v (expected ~100ms)", elapsed)
	}
}

func TestRateLimiterTryAcquire(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 10.0,
		BurstSize:    3,
	}

	limiter := NewRateLimiter(config)

	// Should succeed for burst size
	for i := 0; i < config.BurstSize; i++ {
		if !limiter.TryAcquire() {
			t.Errorf("TryAcquire failed at iteration %d (within burst)", i+1)
		}
	}

	// Should fail when tokens exhausted
	if limiter.TryAcquire() {
		t.Error("TryAcquire should fail when tokens exhausted")
	}
}

func TestRateLimiterContextCancellation(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 1.0, // Very slow rate
		BurstSize:    1,
	}

	limiter := NewRateLimiter(config)

	// Consume the burst token
	ctx := context.Background()
	if err := limiter.Wait(ctx); err != nil {
		t.Fatalf("Failed to get initial token: %v", err)
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start waiting (should block since no tokens available)
	done := make(chan error, 1)
	go func() {
		done <- limiter.Wait(ctx)
	}()

	// Cancel context after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Should return context error
	select {
	case err := <-done:
		if err == nil {
			t.Error("Expected context cancellation error")
		}
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Wait did not respond to context cancellation")
	}
}

func TestRateLimiterStats(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 5.0,
		BurstSize:    10,
	}

	limiter := NewRateLimiter(config)

	stats := limiter.Stats()

	if !stats.Enabled {
		t.Error("Stats should show limiter as enabled")
	}

	if stats.MaxTokens != float64(config.BurstSize) {
		t.Errorf("Expected MaxTokens %d, got %.0f", config.BurstSize, stats.MaxTokens)
	}

	if stats.RefillRate != config.QueriesPerSec {
		t.Errorf("Expected RefillRate %.1f, got %.1f", config.QueriesPerSec, stats.RefillRate)
	}

	// Initial tokens should equal burst size
	if stats.AvailableTokens != float64(config.BurstSize) {
		t.Errorf("Expected AvailableTokens %d, got %.2f", config.BurstSize, stats.AvailableTokens)
	}

	// Consume some tokens
	ctx := context.Background()
	limiter.Wait(ctx)
	limiter.Wait(ctx)

	stats = limiter.Stats()
	if stats.AvailableTokens >= float64(config.BurstSize) {
		t.Error("AvailableTokens should decrease after consumption")
	}
}

func TestDisabledRateLimiter(t *testing.T) {
	config := RateLimitConfig{
		Enabled: false,
	}

	limiter := NewRateLimiter(config)

	// Should always succeed immediately
	ctx := context.Background()
	start := time.Now()

	for i := 0; i < 100; i++ {
		if err := limiter.Wait(ctx); err != nil {
			t.Fatalf("Disabled limiter should never error: %v", err)
		}
	}

	elapsed := time.Since(start)

	// Should be very fast (< 10ms for 100 iterations)
	if elapsed > 10*time.Millisecond {
		t.Errorf("Disabled limiter took too long: %v", elapsed)
	}

	// TryAcquire should always succeed
	for i := 0; i < 100; i++ {
		if !limiter.TryAcquire() {
			t.Error("Disabled limiter TryAcquire should always succeed")
		}
	}

	// Stats should show disabled
	stats := limiter.Stats()
	if stats.Enabled {
		t.Error("Stats should show limiter as disabled")
	}
}

func TestRateLimiterRefillAccuracy(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 5.0, // 5 tokens per second = 200ms per token
		BurstSize:    5,
	}

	limiter := NewRateLimiter(config)
	ctx := context.Background()

	// Consume all tokens
	for i := 0; i < config.BurstSize; i++ {
		limiter.Wait(ctx)
	}

	// Wait for 1 second (should refill 5 tokens)
	time.Sleep(1 * time.Second)

	stats := limiter.Stats()

	// Should have refilled close to burst size
	// Allow some tolerance (4-5 tokens)
	if stats.AvailableTokens < 4.0 || stats.AvailableTokens > float64(config.BurstSize) {
		t.Errorf("Expected ~5 tokens after 1s refill, got %.2f", stats.AvailableTokens)
	}
}

func TestRateLimiterBurstLimit(t *testing.T) {
	config := RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 100.0, // Fast refill rate
		BurstSize:    3,
	}

	limiter := NewRateLimiter(config)

	// Wait for tokens to accumulate
	time.Sleep(100 * time.Millisecond)

	stats := limiter.Stats()

	// Tokens should not exceed burst size even with fast refill
	if stats.AvailableTokens > float64(config.BurstSize) {
		t.Errorf("Tokens exceeded burst limit: %.2f > %d", stats.AvailableTokens, config.BurstSize)
	}
}
