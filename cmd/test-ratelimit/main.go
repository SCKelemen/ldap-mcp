package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/SCKelemen/ldap-mcp/internal/ldap"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LDAP ldap.Config `yaml:"ldap"`
}

func main() {
	// Read config
	data, err := os.ReadFile(".ldap-mcp.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Expand environment variables
	config.LDAP.BindPassword = os.ExpandEnv(config.LDAP.BindPassword)

	if config.LDAP.BindPassword == "" {
		fmt.Println("WARNING: LDAP_PASSWORD environment variable is not set!")
		fmt.Println("Please set it with: export LDAP_PASSWORD='your-password'")
		os.Exit(1)
	}

	fmt.Println("Testing LDAP Rate Limiting...")
	fmt.Printf("Server: %s\n", config.LDAP.Server)

	// Create rate-limited service with strict limits for testing
	rateLimitConfig := ldap.RateLimitConfig{
		Enabled:      true,
		QueriesPerSec: 5.0,  // 5 queries per second
		BurstSize:    10,    // Allow burst of 10
	}

	cacheConfig := ldap.CacheConfig{
		Enabled:    false, // Disable cache to test rate limiting accurately
	}

	service, err := ldap.NewRateLimitedService(
		&config.LDAP,
		ldap.DefaultPoolConfig(),
		cacheConfig,
		rateLimitConfig,
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer service.Close()

	fmt.Println("✓ Connection successful!")
	fmt.Printf("\nRate Limit Configuration:\n")
	fmt.Printf("  Queries/sec: %.1f\n", rateLimitConfig.QueriesPerSec)
	fmt.Printf("  Burst size: %d\n", rateLimitConfig.BurstSize)

	// Test 1: Burst capacity
	fmt.Println("\n=== Test 1: Burst Capacity ===")
	fmt.Printf("Sending %d rapid queries (within burst limit)...\n", rateLimitConfig.BurstSize)

	start := time.Now()
	for i := 0; i < rateLimitConfig.BurstSize; i++ {
		_, err := service.SearchUser("test")
		if err != nil {
			log.Printf("Query %d failed: %v", i+1, err)
		}
	}
	elapsed := time.Since(start)

	fmt.Printf("Time for %d queries: %v\n", rateLimitConfig.BurstSize, elapsed)
	fmt.Printf("Average: %v per query\n", elapsed/time.Duration(rateLimitConfig.BurstSize))

	// Test 2: Rate limiting kicks in
	fmt.Println("\n=== Test 2: Rate Limiting (Exceeding Burst) ===")
	fmt.Println("Sending 5 more queries (should be rate limited)...")

	start = time.Now()
	queryTimes := make([]time.Duration, 0, 5)

	for i := 0; i < 5; i++ {
		queryStart := time.Now()
		_, err := service.SearchUser("test")
		queryElapsed := time.Since(queryStart)
		queryTimes = append(queryTimes, queryElapsed)

		if err != nil {
			log.Printf("Query %d failed: %v", i+1, err)
		}

		fmt.Printf("  Query %d: %v\n", i+1, queryElapsed)
	}

	totalElapsed := time.Since(start)
	fmt.Printf("Total time for 5 queries: %v\n", totalElapsed)
	fmt.Printf("Expected minimum time: ~%.1fs (at %.1f qps)\n",
		5.0/rateLimitConfig.QueriesPerSec,
		rateLimitConfig.QueriesPerSec)

	// Test 3: Concurrent requests
	fmt.Println("\n=== Test 3: Concurrent Requests ===")
	fmt.Println("Sending 20 concurrent queries...")

	var wg sync.WaitGroup
	start = time.Now()
	concurrentQueries := 20

	queryResults := make([]time.Duration, concurrentQueries)
	var resultsMu sync.Mutex

	for i := 0; i < concurrentQueries; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			queryStart := time.Now()
			_, err := service.SearchUser("test")
			queryElapsed := time.Since(queryStart)

			resultsMu.Lock()
			queryResults[index] = queryElapsed
			resultsMu.Unlock()

			if err != nil {
				log.Printf("Query %d failed: %v", index+1, err)
			}
		}(i)
	}

	wg.Wait()
	totalElapsed = time.Since(start)

	fmt.Printf("Total time for %d concurrent queries: %v\n", concurrentQueries, totalElapsed)
	fmt.Printf("Expected minimum time: ~%.1fs (%.1f qps with burst)\n",
		float64(concurrentQueries-rateLimitConfig.BurstSize)/rateLimitConfig.QueriesPerSec,
		rateLimitConfig.QueriesPerSec)

	// Show distribution
	var fastQueries, slowQueries int
	for _, d := range queryResults {
		if d < 100*time.Millisecond {
			fastQueries++
		} else {
			slowQueries++
		}
	}

	fmt.Printf("\nQuery distribution:\n")
	fmt.Printf("  Fast (< 100ms): %d (burst capacity)\n", fastQueries)
	fmt.Printf("  Slow (> 100ms): %d (rate limited)\n", slowQueries)

	// Test 4: Rate limiter statistics
	fmt.Println("\n=== Test 4: Rate Limiter Statistics ===")
	stats := service.RateLimitStats()
	fmt.Printf("Enabled: %v\n", stats.Enabled)
	fmt.Printf("Available tokens: %.2f\n", stats.AvailableTokens)
	fmt.Printf("Max tokens: %.0f\n", stats.MaxTokens)
	fmt.Printf("Refill rate: %.1f tokens/sec\n", stats.RefillRate)

	// Wait for token refill
	fmt.Println("\nWaiting for token bucket to refill (2 seconds)...")
	time.Sleep(2 * time.Second)

	stats = service.RateLimitStats()
	fmt.Printf("Available tokens after wait: %.2f\n", stats.AvailableTokens)

	// Final stats
	fmt.Println("\n=== Final Statistics ===")
	poolStats := service.Stats()
	fmt.Printf("Connection Pool:\n")
	fmt.Printf("  Total: %d\n", poolStats.TotalConns)
	fmt.Printf("  Active: %d\n", poolStats.ActiveConns)
	fmt.Printf("  Idle: %d\n", poolStats.IdleConns)

	fmt.Println("\n✓ All rate limiting tests completed!")
}
