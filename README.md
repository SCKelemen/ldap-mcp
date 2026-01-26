# LDAP MCP Server

Model Context Protocol (MCP) server for querying LDAP directories in AI chat. Enables Claude and other AI agents to search users, groups, and organizational data from Active Directory or other LDAP servers.

## Features

### MCP Tools (12 total)

**User Queries:**
- **search_user**: Search for users by email, username, or name
- **get_user_details**: Get comprehensive information about a specific user
- **bulk_user_lookup**: Efficiently query multiple users at once
- **get_direct_reports**: Get all users reporting to a manager

**Group Queries:**
- **search_group**: Find groups by name or description
- **get_group_members**: List all members of a group
- **get_user_groups**: Get all groups a user belongs to
- **verify_membership**: Check if a user is a member of a group

**Directory Queries:**
- **search_ou**: Search for organizational units
- **get_computer**: Query computer objects in AD
- **search_by_attributes**: Flexible multi-attribute search
- **search_by_filter**: Execute custom LDAP filter queries

**Performance Features:**
- Connection pooling (configurable)
- Query caching with TTL (5-30 min depending on query type)
- Rate limiting (10 qps default, burst of 20)

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

### Claude Code CLI Configuration

Add to your `~/.claude.json`:

```json
{
  "mcpServers": {
    "ldap": {
      "type": "stdio",
      "command": "/path/to/ldap-mcp",
      "args": [],
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
[Uses get_group_members with group_identifier="Engineering"]
Engineering group has 15 members:
1. John Doe (john.doe@example.com)
2. Jane Smith (jane.smith@example.com)
...
```

**Note:** The `get_group_members` tool accepts either a simple group name (e.g., "Engineering") or a full distinguished name (e.g., "CN=Engineering,OU=Groups,DC=example,DC=com").

### Verify group membership

```
User: Is jane.smith a member of the Admins group?

Claude:
[Uses verify_membership tool]
Yes, jane.smith is a member of the Admins group.
```

### Advanced Examples

#### Search organizational units

```
User: Show me all OUs in the London office

Claude:
[Uses search_ou tool with query="London"]
Found 3 organizational units:
1. OU=Users,OU=London,DC=midasplayer,DC=local
2. OU=Groups,OU=London,DC=midasplayer,DC=local
3. OU=Computers,OU=London,DC=midasplayer,DC=local
```

#### Query computer objects

```
User: Find information about the computer named WS-ENG-001

Claude:
[Uses get_computer tool]
Computer: WS-ENG-001
- DNS Hostname: ws-eng-001.example.com
- Operating System: Windows 11 Enterprise
- OS Version: 10.0 (22631)
- Last Logon: 2026-01-25 14:32:15
- Status: Enabled
```

#### Bulk user lookup

```
User: Get information for users: john.doe, jane.smith, bob.jones

Claude:
[Uses bulk_user_lookup tool]
Found 3 users:
1. John Doe (john.doe@example.com) - Engineering
2. Jane Smith (jane.smith@example.com) - Product
3. Bob Jones (bob.jones@example.com) - Marketing
```

#### Get manager's direct reports

```
User: Who reports directly to sarah.manager?

Claude:
[Uses get_direct_reports tool]
Sarah Manager has 8 direct reports:
1. John Doe (john.doe@example.com) - Senior Engineer
2. Jane Smith (jane.smith@example.com) - Product Manager
3. Alice Brown (alice.brown@example.com) - Engineer
...
```

#### Search by custom attributes

```
User: Find all users in the Security department who are in Stockholm

Claude:
[Uses search_by_attributes tool]
{
  "attributes": {
    "department": "Security",
    "l": "Stockholm"
  },
  "objectClass": "user"
}

Found 12 users matching criteria:
1. Samuel Kelemen (samuel.kelemen@king.com) - Principal Security Engineer
2. ...
```

### Performance Characteristics

LDAP MCP includes advanced performance features:

#### Connection Pooling
- **Default**: 10 max connections, 2 min connections
- **Benefits**: Reduced latency, better throughput
- **Automatic**: Health checks and connection recycling

#### Caching
- **Default TTL**: 5 minutes for user/group queries
- **Performance**: 1,000,000x faster for cached queries (1.8s → 1.7µs)
- **Intelligent**: Different TTLs for different data types:
  * User/Group queries: 5 min
  * Group memberships: 10 min (less volatile)
  * OUs: 30 min (very stable)
  * Bulk queries: 2 min (exploratory)

#### Rate Limiting
- **Default**: 10 queries/second with burst of 20
- **Protection**: Prevents LDAP server overload
- **Token bucket**: Smooth rate limiting with burst support

## Configuration

### Full Configuration Example

`.ldap-mcp.yaml`:
```yaml
ldap:
  server: ldap.example.com:389
  use_tls: true
  bind_dn: cn=serviceaccount,dc=example,dc=com
  bind_password: ${LDAP_PASSWORD}
  base_dn: dc=example,dc=com
  timeout: 10s
```

### Custom Pool Configuration

```go
poolConfig := ldap.PoolConfig{
    MaxConns:    20,              // Max concurrent connections
    MinConns:    5,               // Minimum idle connections
    MaxIdleTime: 10 * time.Minute, // Idle timeout
    DialTimeout: 10 * time.Second, // Connection timeout
    HealthCheck: 30 * time.Second, // Health check interval
}
```

### Custom Cache Configuration

```go
cacheConfig := ldap.CacheConfig{
    Enabled:         true,
    DefaultTTL:      5 * time.Minute,  // Default cache TTL
    CleanupInterval: 1 * time.Minute,   // Cleanup frequency
    MaxEntries:      1000,              // Max cache entries
}
```

### Custom Rate Limit Configuration

```go
rateLimitConfig := ldap.RateLimitConfig{
    Enabled:       true,
    QueriesPerSec: 10.0, // Queries per second
    BurstSize:     20,   // Burst capacity
}
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
- `group_identifier` (string, required): Group name (CN) or full distinguished name (DN)

**Returns:** List of user objects for all group members

**Examples:**
```json
// Using group name
{
  "group_identifier": "Engineering"
}

// Using full DN
{
  "group_identifier": "CN=Engineering,OU=Groups,DC=example,DC=com"
}
```

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

### search_ou

Search for organizational units in the directory.

**Parameters:**
- `query` (string, required): OU name or substring to search for

**Returns:** List of OUs with location information (city, state, country)

**Example:**
```json
{
  "query": "London"
}
```

### get_computer

Query computer objects in Active Directory.

**Parameters:**
- `name` (string, required): Computer name (CN or hostname)

**Returns:** Computer object with system information (OS, version, last logon)

**Example:**
```json
{
  "name": "WS-ENG-001"
}
```

### bulk_user_lookup

Efficiently query multiple users in a single request.

**Parameters:**
- `identifiers` ([]string, required): List of user emails, usernames, or DNs

**Returns:** List of user objects for all found users

**Example:**
```json
{
  "identifiers": ["john.doe", "jane.smith", "bob.jones"]
}
```

**Note:** Uses optimized LDAP OR filter for better performance than multiple individual queries.

### get_direct_reports

Get all users who report directly to a manager.

**Parameters:**
- `manager_identifier` (string, required): Manager's email, username, or DN

**Returns:** List of direct report user objects

**Example:**
```json
{
  "manager_identifier": "sarah.manager"
}
```

### search_by_attributes

Flexible search using multiple LDAP attributes.

**Parameters:**
- `attributes` (map[string]string, required): Key-value pairs of attribute names and values
- `object_class` (string, optional): LDAP object class filter (default: "user")

**Returns:** List of matching LDAP entries

**Example:**
```json
{
  "attributes": {
    "department": "Engineering",
    "title": "Senior Engineer",
    "l": "Stockholm"
  },
  "object_class": "user"
}
```

**Note:** All attribute conditions are AND-ed together. Use `search_by_filter` for more complex queries.

## Security Considerations

1. **Credentials**: Store bind credentials securely (environment variables, secrets manager)
2. **TLS**: Enable TLS for production environments
3. **Least privilege**: Use a read-only service account with minimal permissions
4. **Rate limiting**: Built-in token bucket rate limiter prevents LDAP server overload
   - Default: 10 queries/second with burst of 20
   - Configurable per deployment
5. **Connection pooling**: Limits concurrent connections to prevent resource exhaustion
6. **Caching**: Reduces load on LDAP server and improves response times
7. **Logging**: Audit LDAP queries for security monitoring

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
