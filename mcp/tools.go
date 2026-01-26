package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools registers all LDAP MCP tools
func (s *Server) RegisterTools() {
	// Tool: search_user
	s.server.AddTool(
		&mcp.Tool{
			Name:        "search_user",
			Description: "Search for users in LDAP by email, username, or display name",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search term (email, username, or name)",
					},
				},
				"required": []string{"query"},
			},
		},
		s.handleSearchUser,
	)

	// Tool: get_user_details
	s.server.AddTool(
		&mcp.Tool{
			Name:        "get_user_details",
			Description: "Get detailed information about a specific LDAP user",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"identifier": map[string]interface{}{
						"type":        "string",
						"description": "User identifier (email, username, or DN)",
					},
				},
				"required": []string{"identifier"},
			},
		},
		s.handleGetUserDetails,
	)

	// Tool: search_group
	s.server.AddTool(
		&mcp.Tool{
			Name:        "search_group",
			Description: "Search for LDAP groups by name or description",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Group name or description substring",
					},
				},
				"required": []string{"query"},
			},
		},
		s.handleSearchGroup,
	)

	// Tool: get_group_members
	s.server.AddTool(
		&mcp.Tool{
			Name:        "get_group_members",
			Description: "List all members of an LDAP group",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_dn": map[string]interface{}{
						"type":        "string",
						"description": "Distinguished name of the group",
					},
				},
				"required": []string{"group_dn"},
			},
		},
		s.handleGetGroupMembers,
	)

	// Tool: verify_membership
	s.server.AddTool(
		&mcp.Tool{
			Name:        "verify_membership",
			Description: "Check if a user is a member of a specific group",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_identifier": map[string]interface{}{
						"type":        "string",
						"description": "User identifier (email, username, or DN)",
					},
					"group_identifier": map[string]interface{}{
						"type":        "string",
						"description": "Group name or DN",
					},
				},
				"required": []string{"user_identifier", "group_identifier"},
			},
		},
		s.handleVerifyMembership,
	)

	// Tool: search_by_filter
	s.server.AddTool(
		&mcp.Tool{
			Name:        "search_by_filter",
			Description: "Execute a custom LDAP filter query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "LDAP filter (e.g., '(objectClass=person)')",
					},
					"base_dn": map[string]interface{}{
						"type":        "string",
						"description": "Base DN for search (optional, defaults to config)",
					},
					"attributes": map[string]interface{}{
						"type":        "array",
						"description": "Attributes to return (optional)",
						"items": map[string]string{
							"type": "string",
						},
					},
				},
				"required": []string{"filter"},
			},
		},
		s.handleSearchByFilter,
	)

	// Tool: get_user_groups
	s.server.AddTool(
		&mcp.Tool{
			Name:        "get_user_groups",
			Description: "Get all groups a user belongs to",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_identifier": map[string]interface{}{
						"type":        "string",
						"description": "User identifier (email, username, or DN)",
					},
				},
				"required": []string{"user_identifier"},
			},
		},
		s.handleGetUserGroups,
	)

	// Tool: search_ou
	s.server.AddTool(
		&mcp.Tool{
			Name:        "search_ou",
			Description: "Search for organizational units (OUs) in LDAP directory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "OU name or substring to search for",
					},
				},
				"required": []string{"query"},
			},
		},
		s.handleSearchOU,
	)

	// Tool: get_computer
	s.server.AddTool(
		&mcp.Tool{
			Name:        "get_computer",
			Description: "Get information about a computer object in Active Directory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Computer name",
					},
				},
				"required": []string{"name"},
			},
		},
		s.handleGetComputer,
	)

	// Tool: bulk_user_lookup
	s.server.AddTool(
		&mcp.Tool{
			Name:        "bulk_user_lookup",
			Description: "Efficiently lookup multiple users at once by email, username, or DN",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"identifiers": map[string]interface{}{
						"type":        "array",
						"description": "Array of user identifiers (emails, usernames, or DNs)",
						"items": map[string]string{
							"type": "string",
						},
					},
				},
				"required": []string{"identifiers"},
			},
		},
		s.handleBulkUserLookup,
	)

	// Tool: get_direct_reports
	s.server.AddTool(
		&mcp.Tool{
			Name:        "get_direct_reports",
			Description: "Get all direct reports (employees) reporting to a manager",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"manager_identifier": map[string]interface{}{
						"type":        "string",
						"description": "Manager's email, username, or DN",
					},
				},
				"required": []string{"manager_identifier"},
			},
		},
		s.handleGetDirectReports,
	)

	// Tool: search_by_attributes
	s.server.AddTool(
		&mcp.Tool{
			Name:        "search_by_attributes",
			Description: "Flexible search using multiple LDAP attributes",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"attributes": map[string]interface{}{
						"type":        "object",
						"description": "Map of attribute names to search values",
					},
					"object_class": map[string]interface{}{
						"type":        "string",
						"description": "Optional object class filter (e.g., 'user', 'group')",
					},
				},
				"required": []string{"attributes"},
			},
		},
		s.handleSearchByAttributes,
	)

	fmt.Println("Registered 12 LDAP query tools")
}

// Tool handlers

func (s *Server) handleSearchUser(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Query string `json:"query"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	users, err := s.ldapService.SearchUser(input.Query)
	if err != nil {
		return nil, fmt.Errorf("user search failed: %w", err)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleGetUserDetails(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Identifier string `json:"identifier"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	user, err := s.ldapService.GetUserDetails(input.Identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleSearchGroup(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Query string `json:"query"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	groups, err := s.ldapService.SearchGroup(input.Query)
	if err != nil {
		return nil, fmt.Errorf("group search failed: %w", err)
	}

	data, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleGetGroupMembers(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		GroupDN string `json:"group_dn"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	members, err := s.ldapService.GetGroupMembers(input.GroupDN)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}

	data, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleVerifyMembership(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		UserIdentifier  string `json:"user_identifier"`
		GroupIdentifier string `json:"group_identifier"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	isMember, err := s.ldapService.VerifyMembership(input.UserIdentifier, input.GroupIdentifier)
	if err != nil {
		return nil, fmt.Errorf("membership verification failed: %w", err)
	}

	result := map[string]interface{}{
		"is_member": isMember,
		"user":      input.UserIdentifier,
		"group":     input.GroupIdentifier,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleSearchByFilter(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Filter     string   `json:"filter"`
		BaseDN     string   `json:"base_dn"`
		Attributes []string `json:"attributes"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	results, err := s.ldapService.SearchByFilter(input.Filter, input.BaseDN, input.Attributes)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleGetUserGroups(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		UserIdentifier string `json:"user_identifier"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	groups, err := s.ldapService.GetUserGroups(input.UserIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	data, err := json.MarshalIndent(groups, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleSearchOU(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Query string `json:"query"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	ous, err := s.ldapService.SearchOU(input.Query)
	if err != nil {
		return nil, fmt.Errorf("OU search failed: %w", err)
	}

	data, err := json.MarshalIndent(ous, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleGetComputer(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Name string `json:"name"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	computer, err := s.ldapService.GetComputer(input.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get computer: %w", err)
	}

	data, err := json.MarshalIndent(computer, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleBulkUserLookup(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Identifiers []string `json:"identifiers"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	users, err := s.ldapService.BulkUserLookup(input.Identifiers)
	if err != nil {
		return nil, fmt.Errorf("bulk lookup failed: %w", err)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleGetDirectReports(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		ManagerIdentifier string `json:"manager_identifier"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	reports, err := s.ldapService.GetDirectReports(input.ManagerIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get direct reports: %w", err)
	}

	data, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

func (s *Server) handleSearchByAttributes(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input struct {
		Attributes  map[string]string `json:"attributes"`
		ObjectClass string            `json:"object_class"`
	}
	if err := parseArguments(request.Params.Arguments, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	results, err := s.ldapService.SearchByAttributes(input.Attributes, input.ObjectClass)
	if err != nil {
		return nil, fmt.Errorf("attribute search failed: %w", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(data),
			},
		},
	}, nil
}

// parseArguments helper to parse tool arguments from map[string]any
func parseArguments(args interface{}, target interface{}) error {
	// Convert to JSON and back to handle type conversions properly
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	return nil
}
