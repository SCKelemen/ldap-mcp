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
	ldapService *ldap.Service
}

// NewServer creates a new LDAP MCP server
func NewServer(ldapConfig *ldap.Config) (*Server, error) {
	// Create LDAP service
	ldapService, err := ldap.NewService(ldapConfig)
	if err != nil {
		return nil, err
	}

	// Create MCP server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ldap-mcp",
			Version: "0.1.0",
		},
		nil, // ServerOptions
	)

	s := &Server{
		server:      mcpServer,
		ldapService: ldapService,
	}

	// Register tools
	s.RegisterTools()

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
