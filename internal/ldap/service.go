package ldap

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// Service handles LDAP connections and queries
type Service struct {
	config *Config
	conn   *ldap.Conn
}

// NewService creates a new LDAP service with connection
func NewService(config *Config) (*Service, error) {
	// Parse timeout
	timeout := 10 * time.Second
	if config.Timeout != "" {
		if t, err := time.ParseDuration(config.Timeout); err == nil {
			timeout = t
		}
	}

	// Prepare LDAP URL
	ldapURL := "ldap://" + config.Server
	if config.UseTLS {
		ldapURL = "ldaps://" + config.Server
	}

	// Connect to LDAP server
	var conn *ldap.Conn
	var err error

	dialer := &net.Dialer{Timeout: timeout}

	if config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         strings.Split(config.Server, ":")[0],
		}
		conn, err = ldap.DialURL(ldapURL, ldap.DialWithDialer(dialer), ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(ldapURL, ldap.DialWithDialer(dialer))
	}

	if err != nil {
		return nil, fmt.Errorf("LDAP connection failed: %w", err)
	}

	// Bind with service account
	if err := conn.Bind(config.BindDN, config.BindPassword); err != nil {
		conn.Close()
		return nil, fmt.Errorf("LDAP bind failed: %w", err)
	}

	return &Service{
		config: config,
		conn:   conn,
	}, nil
}

// Close closes the LDAP connection
func (s *Service) Close() error {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	return nil
}

// SearchUser searches for users by email, username, or display name
func (s *Service) SearchUser(query string) ([]*UserInfo, error) {
	// Build flexible search filter
	filter := fmt.Sprintf("(&(objectClass=user)(|(sAMAccountName=*%s*)(mail=*%s*)(cn=*%s*)(displayName=*%s*)))",
		ldap.EscapeFilter(query),
		ldap.EscapeFilter(query),
		ldap.EscapeFilter(query),
		ldap.EscapeFilter(query),
	)

	attributes := []string{
		AttrSAMAccountName, AttrMail, AttrCN, AttrDisplayName,
		AttrGivenName, AttrSN, AttrTitle, AttrDepartment,
		AttrCompany, AttrTelephoneNumber, AttrMobile, AttrManager,
		AttrUserAccountControl, AttrMemberOf,
	}

	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		attributes,
		nil,
	)

	sr, err := s.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	users := make([]*UserInfo, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		users = append(users, s.entryToUserInfo(entry))
	}

	return users, nil
}

// GetUserDetails retrieves detailed information about a specific user
func (s *Service) GetUserDetails(identifier string) (*UserInfo, error) {
	// Try to determine identifier type and build appropriate filter
	var filter string
	if strings.Contains(identifier, "@") {
		// Email address
		filter = fmt.Sprintf("(&(objectClass=user)(mail=%s))", ldap.EscapeFilter(identifier))
	} else if strings.Contains(identifier, "=") {
		// Likely a DN, search by DN
		filter = fmt.Sprintf("(&(objectClass=user)(distinguishedName=%s))", ldap.EscapeFilter(identifier))
	} else {
		// Username
		filter = fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", ldap.EscapeFilter(identifier))
	}

	attributes := []string{
		AttrSAMAccountName, AttrMail, AttrCN, AttrDisplayName,
		AttrGivenName, AttrSN, AttrTitle, AttrDepartment,
		AttrCompany, AttrTelephoneNumber, AttrMobile, AttrManager,
		AttrUserAccountControl, AttrMemberOf,
	}

	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false, // Limit to 1 result
		filter,
		attributes,
		nil,
	)

	sr, err := s.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", identifier)
	}

	return s.entryToUserInfo(sr.Entries[0]), nil
}

// SearchGroup searches for groups by name or description
func (s *Service) SearchGroup(query string) ([]*GroupInfo, error) {
	filter := fmt.Sprintf("(&(objectClass=group)(|(cn=*%s*)(description=*%s*)))",
		ldap.EscapeFilter(query),
		ldap.EscapeFilter(query),
	)

	attributes := []string{AttrCN, AttrDescription, AttrMember, AttrGroupType}

	searchRequest := ldap.NewSearchRequest(
		s.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		attributes,
		nil,
	)

	sr, err := s.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	groups := make([]*GroupInfo, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		groups = append(groups, s.entryToGroupInfo(entry))
	}

	return groups, nil
}

// GetGroupMembers retrieves all members of a group
func (s *Service) GetGroupMembers(groupDN string) ([]*UserInfo, error) {
	// First get the group entry to retrieve member DNs
	searchRequest := ldap.NewSearchRequest(
		groupDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=group)",
		[]string{AttrMember},
		nil,
	)

	sr, err := s.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("group search failed: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("group not found: %s", groupDN)
	}

	memberDNs := sr.Entries[0].GetAttributeValues(AttrMember)
	if len(memberDNs) == 0 {
		return []*UserInfo{}, nil
	}

	// Retrieve user details for each member
	users := make([]*UserInfo, 0, len(memberDNs))
	for _, memberDN := range memberDNs {
		user, err := s.GetUserDetails(memberDN)
		if err != nil {
			// Skip members that can't be retrieved (might be other groups, etc.)
			continue
		}
		users = append(users, user)
	}

	return users, nil
}

// VerifyMembership checks if a user is a member of a group
func (s *Service) VerifyMembership(userIdentifier, groupIdentifier string) (bool, error) {
	// Get user details to get DN
	user, err := s.GetUserDetails(userIdentifier)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	// Get group DN if not already a DN
	var groupDN string
	if strings.Contains(groupIdentifier, "=") {
		groupDN = groupIdentifier
	} else {
		groups, err := s.SearchGroup(groupIdentifier)
		if err != nil || len(groups) == 0 {
			return false, fmt.Errorf("failed to find group: %s", groupIdentifier)
		}
		groupDN = groups[0].DN
	}

	// Check if user's memberOf contains the group DN
	for _, memberOf := range user.MemberOf {
		if strings.EqualFold(memberOf, groupDN) {
			return true, nil
		}
	}

	return false, nil
}

// SearchByFilter executes a custom LDAP filter query
func (s *Service) SearchByFilter(filter, baseDN string, attributes []string) ([]*SearchResult, error) {
	if baseDN == "" {
		baseDN = s.config.BaseDN
	}

	if len(attributes) == 0 {
		attributes = []string{"*"} // All attributes
	}

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		attributes,
		nil,
	)

	sr, err := s.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	results := make([]*SearchResult, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		result := &SearchResult{
			DN:         entry.DN,
			Attributes: make(map[string]string),
		}
		for _, attr := range entry.Attributes {
			if len(attr.Values) > 0 {
				result.Attributes[attr.Name] = attr.Values[0]
			}
		}
		results = append(results, result)
	}

	return results, nil
}

// GetUserGroups retrieves all groups a user belongs to
func (s *Service) GetUserGroups(userIdentifier string) ([]*GroupInfo, error) {
	user, err := s.GetUserDetails(userIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	groups := make([]*GroupInfo, 0, len(user.MemberOf))
	for _, groupDN := range user.MemberOf {
		// Get group details
		searchRequest := ldap.NewSearchRequest(
			groupDN,
			ldap.ScopeBaseObject,
			ldap.NeverDerefAliases,
			0, 0, false,
			"(objectClass=group)",
			[]string{AttrCN, AttrDescription, AttrGroupType},
			nil,
		)

		sr, err := s.conn.Search(searchRequest)
		if err != nil {
			continue // Skip groups we can't access
		}

		if len(sr.Entries) > 0 {
			groups = append(groups, s.entryToGroupInfo(sr.Entries[0]))
		}
	}

	return groups, nil
}

// Helper function to convert LDAP entry to UserInfo
func (s *Service) entryToUserInfo(entry *ldap.Entry) *UserInfo {
	user := &UserInfo{
		DN:             entry.DN,
		Username:       entry.GetAttributeValue(AttrSAMAccountName),
		Email:          entry.GetAttributeValue(AttrMail),
		DisplayName:    entry.GetAttributeValue(AttrDisplayName),
		FirstName:      entry.GetAttributeValue(AttrGivenName),
		LastName:       entry.GetAttributeValue(AttrSN),
		Title:          entry.GetAttributeValue(AttrTitle),
		Department:     entry.GetAttributeValue(AttrDepartment),
		Company:        entry.GetAttributeValue(AttrCompany),
		Phone:          entry.GetAttributeValue(AttrTelephoneNumber),
		Mobile:         entry.GetAttributeValue(AttrMobile),
		Manager:        entry.GetAttributeValue(AttrManager),
		MemberOf:       entry.GetAttributeValues(AttrMemberOf),
		Status:         "Unknown",
		AccountControl: 0,
	}

	// Use CN if DisplayName is empty
	if user.DisplayName == "" {
		user.DisplayName = entry.GetAttributeValue(AttrCN)
	}

	// Parse userAccountControl to determine status
	if uacStr := entry.GetAttributeValue(AttrUserAccountControl); uacStr != "" {
		if uac, err := strconv.Atoi(uacStr); err == nil {
			user.AccountControl = uac
			if uac&UACAccountDisabled != 0 {
				user.Status = "Disabled"
			} else {
				user.Status = "Active"
			}
		}
	}

	return user
}

// Helper function to convert LDAP entry to GroupInfo
func (s *Service) entryToGroupInfo(entry *ldap.Entry) *GroupInfo {
	members := entry.GetAttributeValues(AttrMember)
	return &GroupInfo{
		DN:          entry.DN,
		Name:        entry.GetAttributeValue(AttrCN),
		Description: entry.GetAttributeValue(AttrDescription),
		MemberCount: len(members),
		Members:     members,
		GroupType:   entry.GetAttributeValue(AttrGroupType),
	}
}
