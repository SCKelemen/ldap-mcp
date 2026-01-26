# LDAP MCP Server

Model Context Protocol (MCP) server for querying LDAP directories in AI chat. Enables Claude and other AI agents to search users, groups, and organizational data from Active Directory or other LDAP servers.

## Features

### MCP Tools (7 total)

- **search_user**: Search for users by email, username, or name
- **get_user_details**: Get detailed information about a specific user
- **search_group**: Search for groups by name or description
- **get_group_members**: List all members of a group
- **verify_membership**: Check if a user is a member of a group
- **search_by_filter**: Execute custom LDAP filter queries
- **get_user_groups**: Get all groups a user belongs to

## Installation

### Build from source

```bash
go build -o ldap-mcp ./cmd/ldap-mcp
```

### Configuration

Create a configuration file `.ldap-mcp.yaml`:

```yaml
ldap:
  server: ldap.example.com:389
  use_tls: false
  bind_dn: cn=serviceaccount,dc=example,dc=com
  bind_password: your-password
  base_dn: dc=example,dc=com
  timeout: 10s
```

Or use environment variables:

```bash
export LDAP_SERVER=ldap.example.com:389
export LDAP_BIND_DN=cn=serviceaccount,dc=example,dc=com
export LDAP_BIND_PASSWORD=your-password
export LDAP_BASE_DN=dc=example,dc=com
```

### Claude Desktop Configuration

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "ldap": {
      "command": "ldap-mcp",
      "env": {
        "LDAP_SERVER": "ldap.example.com:389",
        "LDAP_BIND_DN": "cn=serviceaccount,dc=example,dc=com",
        "LDAP_BIND_PASSWORD": "your-password",
        "LDAP_BASE_DN": "dc=example,dc=com"
      }
    }
  }
}
```

## Usage Examples

### Search for a user

```
User: Find the user with email john.doe@example.com

Claude:
[Uses search_user tool]
Found user: John Doe (john.doe)
- Email: john.doe@example.com
- Display Name: John Doe
- Department: Engineering
- Status: Active
```

### Get group members

```
User: Who are the members of the Engineering team?

Claude:
[Uses search_group to find "Engineering", then get_group_members]
Engineering group has 15 members:
1. John Doe (john.doe@example.com)
2. Jane Smith (jane.smith@example.com)
...
```

### Verify group membership

```
User: Is jane.smith a member of the Admins group?

Claude:
[Uses verify_membership tool]
Yes, jane.smith is a member of the Admins group.
```

## Architecture

```
ldap-mcp/
├── cmd/
│   └── ldap-mcp/       # Main MCP server binary
├── internal/
│   └── ldap/           # LDAP connection and query logic
├── mcp/                # MCP server and tool implementations
├── .ldap-mcp.yaml      # Configuration file (not in repo)
└── go.mod
```

### Design Principles

Following MCP best practices:

- **Data-source specific**: Specialized for LDAP queries
- **Stateful connections**: Maintains LDAP connection pool
- **Configuration-driven**: Flexible connection settings
- **Security-aware**: Supports TLS, credential management
- **Composable**: Works with other MCP servers

## Tool Details

### search_user

Search for users by various attributes.

**Parameters:**
- `query` (string, required): Search term (email, username, or display name)
- `attributes` ([]string, optional): Additional attributes to return

**Returns:** List of matching users with basic info

### get_user_details

Get comprehensive information about a specific user.

**Parameters:**
- `identifier` (string, required): User email, username, or DN

**Returns:** Full user object with all available attributes

### search_group

Search for groups by name or description.

**Parameters:**
- `query` (string, required): Group name or description substring

**Returns:** List of matching groups

### get_group_members

List all members of a group.

**Parameters:**
- `group_dn` (string, required): Distinguished name of the group

**Returns:** List of user objects for all group members

### verify_membership

Check if a user is a member of a group.

**Parameters:**
- `user_identifier` (string, required): User email, username, or DN
- `group_identifier` (string, required): Group name or DN

**Returns:** Boolean indicating membership status

### search_by_filter

Execute a custom LDAP filter query.

**Parameters:**
- `filter` (string, required): LDAP filter (e.g., "(objectClass=person)")
- `base_dn` (string, optional): Base DN for search (defaults to config)
- `attributes` ([]string, optional): Attributes to return

**Returns:** List of matching LDAP entries

### get_user_groups

Get all groups a user belongs to.

**Parameters:**
- `user_identifier` (string, required): User email, username, or DN

**Returns:** List of group names and DNs

## Security Considerations

1. **Credentials**: Store bind credentials securely (environment variables, secrets manager)
2. **TLS**: Enable TLS for production environments
3. **Least privilege**: Use a read-only service account with minimal permissions
4. **Rate limiting**: Consider implementing query rate limits for large directories
5. **Logging**: Audit LDAP queries for security monitoring

## Comparison with other approaches

### Why MCP Tool (not a Skill)?

LDAP MCP is a **tool** because it:
- ✅ Provides data retrieval from a specific source (LDAP)
- ✅ Stateful connection management required
- ✅ Works as independent service
- ✅ Composable with other MCPs (e.g., combine with Jira, Slack MCPs)

A **skill** would be appropriate for:
- ❌ Opinionated workflows (e.g., "provision new user account")
- ❌ Multi-step orchestration (LDAP + ticketing + email)
- ❌ Claude Code-specific features

## Development

### Run tests

```bash
go test ./...
```

### Run locally

```bash
export LDAP_SERVER=ldap.example.com:389
export LDAP_BIND_DN=cn=serviceaccount,dc=example,dc=com
export LDAP_BIND_PASSWORD=your-password
export LDAP_BASE_DN=dc=example,dc=com

./ldap-mcp
```

## License

BearWare 1.0 (MIT Compatible) 🐻

See [LICENSE](LICENSE) for the full bear-framed license text.

## Credits

Based on LDAP implementation patterns from [Knugen](https://github.com/King/knugen).
