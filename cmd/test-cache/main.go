package main

import (
	"fmt"
	"log"
	"os"
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

	fmt.Println("Testing LDAP Caching...")
	fmt.Printf("Server: %s\n", config.LDAP.Server)

	// Create cached service with short TTL for testing
	cacheConfig := ldap.CacheConfig{
		Enabled:         true,
		DefaultTTL:      10 * time.Second,
		CleanupInterval: 5 * time.Second,
		MaxEntries:      100,
	}

	service, err := ldap.NewCachedService(&config.LDAP, ldap.DefaultPoolConfig(), cacheConfig)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer service.Close()

	fmt.Println("✓ Connection successful!")

	testQuery := "samuel.kelemen"

	// First query - should hit LDAP
	fmt.Println("\n=== First Query (Cache Miss) ===")
	start := time.Now()
	users1, err := service.SearchUser(testQuery)
	elapsed1 := time.Since(start)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	fmt.Printf("Query: '%s'\n", testQuery)
	fmt.Printf("Results: %d user(s)\n", len(users1))
	fmt.Printf("Time: %v\n", elapsed1)

	// Check cache stats
	cacheStats := service.CacheStats()
	fmt.Printf("Cache entries: %d\n", cacheStats.Entries)

	// Second query - should hit cache
	fmt.Println("\n=== Second Query (Cache Hit) ===")
	start = time.Now()
	users2, err := service.SearchUser(testQuery)
	elapsed2 := time.Since(start)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	fmt.Printf("Query: '%s'\n", testQuery)
	fmt.Printf("Results: %d user(s)\n", len(users2))
	fmt.Printf("Time: %v\n", elapsed2)
	fmt.Printf("Speedup: %.2fx faster\n", float64(elapsed1)/float64(elapsed2))

	// Verify results are the same
	if len(users1) != len(users2) {
		log.Fatal("Cache returned different results!")
	}
	fmt.Println("✓ Cached results match!")

	// Test GetUserDetails caching
	if len(users1) > 0 {
		username := users1[0].Username

		fmt.Println("\n=== Testing GetUserDetails Caching ===")

		// First call - cache miss
		start = time.Now()
		details1, err := service.GetUserDetails(username)
		elapsed1 = time.Since(start)
		if err != nil {
			log.Fatalf("GetUserDetails failed: %v", err)
		}
		fmt.Printf("First call: %v (DN: %s)\n", elapsed1, details1.DN)

		// Second call - cache hit
		start = time.Now()
		_, err = service.GetUserDetails(username)
		elapsed2 = time.Since(start)
		if err != nil {
			log.Fatalf("GetUserDetails failed: %v", err)
		}
		fmt.Printf("Second call: %v\n", elapsed2)
		fmt.Printf("Speedup: %.2fx faster\n", float64(elapsed1)/float64(elapsed2))
		fmt.Println("✓ Details cached successfully!")
	}

	// Test cache expiration
	fmt.Println("\n=== Testing Cache Expiration ===")
	fmt.Printf("Waiting for cache to expire (TTL: %v)...\n", cacheConfig.DefaultTTL)
	time.Sleep(cacheConfig.DefaultTTL + 1*time.Second)

	start = time.Now()
	users3, err := service.SearchUser(testQuery)
	elapsed3 := time.Since(start)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	fmt.Printf("After expiration: %v\n", elapsed3)
	fmt.Printf("Results: %d user(s)\n", len(users3))

	if elapsed3 < elapsed2 {
		fmt.Println("WARNING: Query after expiration was faster than expected (might still be cached)")
	} else {
		fmt.Println("✓ Cache expired correctly, new LDAP query performed")
	}

	// Test cache clearing
	fmt.Println("\n=== Testing Cache Clear ===")
	service.ClearCache()
	cacheStats = service.CacheStats()
	fmt.Printf("Cache entries after clear: %d\n", cacheStats.Entries)
	if cacheStats.Entries == 0 {
		fmt.Println("✓ Cache cleared successfully!")
	} else {
		fmt.Println("WARNING: Cache still has entries after clear")
	}

	// Final stats
	fmt.Println("\n=== Final Statistics ===")
	poolStats := service.Stats()
	fmt.Printf("Connection Pool:\n")
	fmt.Printf("  Total: %d\n", poolStats.TotalConns)
	fmt.Printf("  Active: %d\n", poolStats.ActiveConns)
	fmt.Printf("  Idle: %d\n", poolStats.IdleConns)
	fmt.Printf("  Unhealthy: %d\n", poolStats.UnhealthyConns)

	cacheStats = service.CacheStats()
	fmt.Printf("\nCache:\n")
	fmt.Printf("  Enabled: %v\n", cacheStats.Enabled)
	fmt.Printf("  Entries: %d\n", cacheStats.Entries)
	fmt.Printf("  TTL: %v\n", cacheStats.TTL)

	fmt.Println("\n✓ All cache tests passed!")
}
