package ldap

// Config represents LDAP connection configuration
type Config struct {
	Server       string `json:"server" yaml:"server"`               // LDAP server address (host:port)
	UseTLS       bool   `json:"use_tls" yaml:"use_tls"`             // Use TLS connection
	BindDN       string `json:"bind_dn" yaml:"bind_dn"`             // Service account DN for binding
	BindPassword string `json:"bind_password" yaml:"bind_password"` // Service account password
	BaseDN       string `json:"base_dn" yaml:"base_dn"`             // Base DN for searches
	Timeout      string `json:"timeout" yaml:"timeout"`             // Connection timeout (e.g., "10s")
}

// UserInfo represents detailed user information from LDAP
type UserInfo struct {
	DN             string            `json:"dn"`
	Username       string            `json:"username"`             // sAMAccountName
	Email          string            `json:"email"`                // mail
	DisplayName    string            `json:"display_name"`         // cn or displayName
	FirstName      string            `json:"first_name"`           // givenName
	LastName       string            `json:"last_name"`            // sn
	Title          string            `json:"title,omitempty"`      // title
	Department     string            `json:"department,omitempty"` // department
	Company        string            `json:"company,omitempty"`    // company
	Phone          string            `json:"phone,omitempty"`      // telephoneNumber
	Mobile         string            `json:"mobile,omitempty"`     // mobile
	Manager        string            `json:"manager,omitempty"`    // manager DN
	Status         string            `json:"status"`               // "Active", "Inactive", "Disabled"
	AccountControl int               `json:"account_control"`      // userAccountControl
	MemberOf       []string          `json:"member_of,omitempty"`  // Direct group memberships
	Attributes     map[string]string `json:"attributes,omitempty"` // Additional attributes
}

// GroupInfo represents LDAP group information
type GroupInfo struct {
	DN          string   `json:"dn"`
	Name        string   `json:"name"`                  // cn
	Description string   `json:"description,omitempty"` // description
	MemberCount int      `json:"member_count"`          // Number of members
	Members     []string `json:"members,omitempty"`     // Member DNs
	GroupType   string   `json:"group_type"`            // "Security", "Distribution", etc.
}

// SearchResult represents a generic LDAP search result
type SearchResult struct {
	DN         string            `json:"dn"`
	Attributes map[string]string `json:"attributes"`
}

// OUInfo represents organizational unit information
type OUInfo struct {
	DN          string `json:"dn"`
	Name        string `json:"name"` // ou
	Description string `json:"description,omitempty"`
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Country     string `json:"country,omitempty"`
}

// ComputerInfo represents computer object information
type ComputerInfo struct {
	DN              string `json:"dn"`
	Name            string `json:"name"`                       // cn
	DNSHostName     string `json:"dns_hostname"`               // dNSHostName
	OperatingSystem string `json:"operating_system,omitempty"` // operatingSystem
	OSVersion       string `json:"os_version,omitempty"`       // operatingSystemVersion
	Description     string `json:"description,omitempty"`
	LastLogon       string `json:"last_logon,omitempty"` // lastLogonTimestamp
	Enabled         bool   `json:"enabled"`
}

// Common LDAP attribute names
const (
	AttrSAMAccountName     = "sAMAccountName"
	AttrMail               = "mail"
	AttrCN                 = "cn"
	AttrDisplayName        = "displayName"
	AttrGivenName          = "givenName"
	AttrSN                 = "sn"
	AttrTitle              = "title"
	AttrDepartment         = "department"
	AttrCompany            = "company"
	AttrTelephoneNumber    = "telephoneNumber"
	AttrMobile             = "mobile"
	AttrManager            = "manager"
	AttrUserAccountControl = "userAccountControl"
	AttrMemberOf           = "memberOf"
	AttrMember             = "member"
	AttrDescription        = "description"
	AttrGroupType          = "groupType"
	AttrObjectClass        = "objectClass"
	AttrOU                 = "ou"
	AttrStreet             = "street"
	AttrL                  = "l"  // locality/city
	AttrST                 = "st" // state
	AttrC                  = "c"  // country
	AttrDNSHostName        = "dNSHostName"
	AttrOperatingSystem    = "operatingSystem"
	AttrOSVersion          = "operatingSystemVersion"
	AttrLastLogonTimestamp = "lastLogonTimestamp"
	AttrDirectReports      = "directReports"
)

// Common LDAP object classes
const (
	ObjectClassPerson             = "person"
	ObjectClassUser               = "user"
	ObjectClassGroup              = "group"
	ObjectClassOrganizationalUnit = "organizationalUnit"
	ObjectClassComputer           = "computer"
)

// UserAccountControl flags
const (
	UACAccountDisabled = 0x0002
	UACNormalAccount   = 0x0200
)
