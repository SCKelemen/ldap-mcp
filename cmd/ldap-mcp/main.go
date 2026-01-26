package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SCKelemen/ldap-mcp/internal/ldap"
	"github.com/SCKelemen/ldap-mcp/mcp"
)

func main() {
	// Load configuration from environment variables
	config := &ldap.Config{
		Server:       getEnv("LDAP_SERVER", ""),
		UseTLS:       getEnv("LDAP_USE_TLS", "false") == "true",
		BindDN:       getEnv("LDAP_BIND_DN", ""),
		BindPassword: getEnv("LDAP_BIND_PASSWORD", ""),
		BaseDN:       getEnv("LDAP_BASE_DN", ""),
		Timeout:      getEnv("LDAP_TIMEOUT", "10s"),
	}

	// Validate required configuration
	if config.Server == "" {
		log.Fatal("LDAP_SERVER environment variable is required")
	}
	if config.BindDN == "" {
		log.Fatal("LDAP_BIND_DN environment variable is required")
	}
	if config.BindPassword == "" {
		log.Fatal("LDAP_BIND_PASSWORD environment variable is required")
	}
	if config.BaseDN == "" {
		log.Fatal("LDAP_BASE_DN environment variable is required")
	}

	// Create MCP server
	server, err := mcp.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}
	defer server.Close()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Run MCP server
	if err := server.Run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
