package mcp

import (
	"context"
	"log"

	"github.com/SCKelemen/ldap-mcp/internal/ldap"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents the LDAP MCP server
type Server struct {
	server      *mcp.Server
	ldapService *ldap.RateLimitedService
}

// NewServer creates a new LDAP MCP server with default configurations
func NewServer(ldapConfig *ldap.Config) (*Server, error) {
	return NewServerWithConfigs(
		ldapConfig,
		ldap.DefaultPoolConfig(),
		ldap.DefaultCacheConfig(),
		ldap.DefaultRateLimitConfig(),
	)
}

// NewServerWithConfigs creates a new LDAP MCP server with custom configurations
func NewServerWithConfigs(
	ldapConfig *ldap.Config,
	poolConfig ldap.PoolConfig,
	cacheConfig ldap.CacheConfig,
	rateLimitConfig ldap.RateLimitConfig,
) (*Server, error) {
	// Create LDAP service with caching and rate limiting
	ldapService, err := ldap.NewRateLimitedService(ldapConfig, poolConfig, cacheConfig, rateLimitConfig)
	if err != nil {
		return nil, err
	}

	// Create MCP server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ldap-mcp",
			Version: "0.3.0", // Bumped version for rate limiting support
		},
		nil, // ServerOptions
	)

	s := &Server{
		server:      mcpServer,
		ldapService: ldapService,
	}

	// Register tools
	s.RegisterTools()

	log.Printf("LDAP MCP Server initialized:")
	log.Printf("  Connection Pool: max=%d, min=%d", poolConfig.MaxConns, poolConfig.MinConns)
	log.Printf("  Cache: enabled=%v, ttl=%s", cacheConfig.Enabled, cacheConfig.DefaultTTL)
	log.Printf("  Rate Limit: enabled=%v, qps=%.1f, burst=%d",
		rateLimitConfig.Enabled, rateLimitConfig.QueriesPerSec, rateLimitConfig.BurstSize)

	return s, nil
}

// Run starts the MCP server using stdio transport
func (s *Server) Run(ctx context.Context) error {
	log.Println("Starting LDAP MCP server...")
	log.Println("Listening on stdio...")

	// Run server with stdio transport
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

// Close closes the server and LDAP connection
func (s *Server) Close() error {
	if s.ldapService != nil {
		return s.ldapService.Close()
	}
	return nil
}

// GetMCPServer returns the underlying MCP server instance
func (s *Server) GetMCPServer() *mcp.Server {
	return s.server
}
