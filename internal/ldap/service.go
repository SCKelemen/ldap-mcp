package ldap

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

// Service handles LDAP connections and queries using a connection pool
type Service struct {
	config *Config
	pool   *ConnectionPool
}

// NewService creates a new LDAP service with connection pool
func NewService(config *Config) (*Service, error) {
	return NewServiceWithPool(config, DefaultPoolConfig())
}

// NewServiceWithPool creates a new LDAP service with custom pool configuration
func NewServiceWithPool(config *Config, poolConfig PoolConfig) (*Service, error) {
	pool, err := NewConnectionPool(config, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &Service{
		config: config,
		pool:   pool,
	}, nil
}

// withConnection executes a function with a connection from the pool
// Automatically handles acquire, release, and error handling
func (s *Service) withConnection(fn func(*ldap.Conn) error) error {
	conn, err := s.pool.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer s.pool.Release(conn)

	if err := fn(conn); err != nil {
		// Mark connection as unhealthy if there's an LDAP error
		if ldap.IsErrorWithCode(err, ldap.ErrorNetwork) ||
			ldap.IsErrorWithCode(err, ldap.LDAPResultUnwillingToPerform) {
			s.pool.MarkUnhealthy(conn)
		}
		return err
	}

	return nil
}

// Close closes the connection pool
func (s *Service) Close() error {
	if s.pool != nil {
		return s.pool.Close()
	}
	return nil
}

// Stats returns current connection pool statistics
func (s *Service) Stats() PoolStats {
	if s.pool != nil {
		return s.pool.Stats()
	}
	return PoolStats{}
}

// SearchUser searches for users by email, username, or display name
func (s *Service) SearchUser(query string) ([]*UserInfo, error) {
	var users []*UserInfo

	err := s.withConnection(func(conn *ldap.Conn) error {
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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		users = make([]*UserInfo, 0, len(sr.Entries))
		for _, entry := range sr.Entries {
			users = append(users, s.entryToUserInfo(entry))
		}

		return nil
	})

	return users, err
}

// GetUserDetails retrieves detailed information about a specific user
func (s *Service) GetUserDetails(identifier string) (*UserInfo, error) {
	var user *UserInfo

	err := s.withConnection(func(conn *ldap.Conn) error {
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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		if len(sr.Entries) == 0 {
			return fmt.Errorf("user not found: %s", identifier)
		}

		user = s.entryToUserInfo(sr.Entries[0])
		return nil
	})

	return user, err
}

// SearchGroup searches for groups by name or description
func (s *Service) SearchGroup(query string) ([]*GroupInfo, error) {
	var groups []*GroupInfo

	err := s.withConnection(func(conn *ldap.Conn) error {
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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		groups = make([]*GroupInfo, 0, len(sr.Entries))
		for _, entry := range sr.Entries {
			groups = append(groups, s.entryToGroupInfo(entry))
		}

		return nil
	})

	return groups, err
}

// GetGroupMembers retrieves all members of a group
// groupIdentifier can be a group name (CN) or full DN
func (s *Service) GetGroupMembers(groupIdentifier string) ([]*UserInfo, error) {
	var memberDNS []string

	// Resolve group identifier to DN
	var groupDN string
	if strings.Contains(groupIdentifier, "=") {
		// Already a DN
		groupDN = groupIdentifier
	} else {
		// Search for the group by name
		groups, err := s.SearchGroup(groupIdentifier)
		if err != nil || len(groups) == 0 {
			return nil, fmt.Errorf("failed to find group: %s", groupIdentifier)
		}
		groupDN = groups[0].DN
	}

	err := s.withConnection(func(conn *ldap.Conn) error {
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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("group search failed: %w", err)
		}

		if len(sr.Entries) == 0 {
			return fmt.Errorf("group not found: %s", groupDN)
		}

		memberDNS = sr.Entries[0].GetAttributeValues(AttrMember)
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(memberDNS) == 0 {
		return []*UserInfo{}, nil
	}

	// Retrieve user details for each member
	users := make([]*UserInfo, 0, len(memberDNS))
	for _, memberDN := range memberDNS {
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
	var results []*SearchResult

	err := s.withConnection(func(conn *ldap.Conn) error {
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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		results = make([]*SearchResult, 0, len(sr.Entries))
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

		return nil
	})

	return results, err
}

// GetUserGroups retrieves all groups a user belongs to
func (s *Service) GetUserGroups(userIdentifier string) ([]*GroupInfo, error) {
	user, err := s.GetUserDetails(userIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	var groups []*GroupInfo

	err = s.withConnection(func(conn *ldap.Conn) error {
		groups = make([]*GroupInfo, 0, len(user.MemberOf))
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

			sr, err := conn.Search(searchRequest)
			if err != nil {
				continue // Skip groups we can't access
			}

			if len(sr.Entries) > 0 {
				groups = append(groups, s.entryToGroupInfo(sr.Entries[0]))
			}
		}

		return nil
	})

	return groups, err
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

// SearchOU searches for organizational units by name
func (s *Service) SearchOU(query string) ([]*OUInfo, error) {
	var ous []*OUInfo

	err := s.withConnection(func(conn *ldap.Conn) error {
		filter := fmt.Sprintf("(&(objectClass=organizationalUnit)(ou=*%s*))",
			ldap.EscapeFilter(query),
		)

		attributes := []string{AttrOU, AttrDescription, AttrStreet, AttrL, AttrST, AttrC}

		searchRequest := ldap.NewSearchRequest(
			s.config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0, 0, false,
			filter,
			attributes,
			nil,
		)

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		ous = make([]*OUInfo, 0, len(sr.Entries))
		for _, entry := range sr.Entries {
			ous = append(ous, &OUInfo{
				DN:          entry.DN,
				Name:        entry.GetAttributeValue(AttrOU),
				Description: entry.GetAttributeValue(AttrDescription),
				Street:      entry.GetAttributeValue(AttrStreet),
				City:        entry.GetAttributeValue(AttrL),
				State:       entry.GetAttributeValue(AttrST),
				Country:     entry.GetAttributeValue(AttrC),
			})
		}

		return nil
	})

	return ous, err
}

// GetComputer retrieves information about a computer object
func (s *Service) GetComputer(name string) (*ComputerInfo, error) {
	var computer *ComputerInfo

	err := s.withConnection(func(conn *ldap.Conn) error {
		// Build filter for computer name
		filter := fmt.Sprintf("(&(objectClass=computer)(cn=%s))", ldap.EscapeFilter(name))

		attributes := []string{
			AttrCN, AttrDNSHostName, AttrOperatingSystem, AttrOSVersion,
			AttrDescription, AttrLastLogonTimestamp, AttrUserAccountControl,
		}

		searchRequest := ldap.NewSearchRequest(
			s.config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			1, 0, false,
			filter,
			attributes,
			nil,
		)

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		if len(sr.Entries) == 0 {
			return fmt.Errorf("computer not found: %s", name)
		}

		entry := sr.Entries[0]
		computer = &ComputerInfo{
			DN:              entry.DN,
			Name:            entry.GetAttributeValue(AttrCN),
			DNSHostName:     entry.GetAttributeValue(AttrDNSHostName),
			OperatingSystem: entry.GetAttributeValue(AttrOperatingSystem),
			OSVersion:       entry.GetAttributeValue(AttrOSVersion),
			Description:     entry.GetAttributeValue(AttrDescription),
			LastLogon:       entry.GetAttributeValue(AttrLastLogonTimestamp),
			Enabled:         true,
		}

		// Check if computer account is disabled
		if uacStr := entry.GetAttributeValue(AttrUserAccountControl); uacStr != "" {
			if uac, err := strconv.Atoi(uacStr); err == nil {
				computer.Enabled = (uac & UACAccountDisabled) == 0
			}
		}

		return nil
	})

	return computer, err
}

// BulkUserLookup efficiently retrieves multiple users by identifiers
func (s *Service) BulkUserLookup(identifiers []string) ([]*UserInfo, error) {
	if len(identifiers) == 0 {
		return []*UserInfo{}, nil
	}

	var users []*UserInfo

	err := s.withConnection(func(conn *ldap.Conn) error {
		// Build OR filter for all identifiers
		var filterParts []string
		for _, id := range identifiers {
			if strings.Contains(id, "@") {
				filterParts = append(filterParts, fmt.Sprintf("(mail=%s)", ldap.EscapeFilter(id)))
			} else if strings.Contains(id, "=") {
				filterParts = append(filterParts, fmt.Sprintf("(distinguishedName=%s)", ldap.EscapeFilter(id)))
			} else {
				filterParts = append(filterParts, fmt.Sprintf("(sAMAccountName=%s)", ldap.EscapeFilter(id)))
			}
		}

		filter := fmt.Sprintf("(&(objectClass=user)(|%s))", strings.Join(filterParts, ""))

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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP bulk search failed: %w", err)
		}

		users = make([]*UserInfo, 0, len(sr.Entries))
		for _, entry := range sr.Entries {
			users = append(users, s.entryToUserInfo(entry))
		}

		return nil
	})

	return users, err
}

// GetDirectReports retrieves all direct reports for a manager
func (s *Service) GetDirectReports(managerIdentifier string) ([]*UserInfo, error) {
	// First get the manager's DN
	manager, err := s.GetUserDetails(managerIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find manager: %w", err)
	}

	var reports []*UserInfo

	err = s.withConnection(func(conn *ldap.Conn) error {
		// Search for users where manager attribute matches this DN
		filter := fmt.Sprintf("(&(objectClass=user)(manager=%s))", ldap.EscapeFilter(manager.DN))

		attributes := []string{
			AttrSAMAccountName, AttrMail, AttrCN, AttrDisplayName,
			AttrGivenName, AttrSN, AttrTitle, AttrDepartment,
			AttrCompany, AttrUserAccountControl,
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

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		reports = make([]*UserInfo, 0, len(sr.Entries))
		for _, entry := range sr.Entries {
			reports = append(reports, s.entryToUserInfo(entry))
		}

		return nil
	})

	return reports, err
}

// SearchByAttributes searches LDAP using multiple attribute filters
func (s *Service) SearchByAttributes(attributes map[string]string, objectClass string) ([]*SearchResult, error) {
	if len(attributes) == 0 {
		return nil, fmt.Errorf("no search attributes provided")
	}

	var results []*SearchResult

	err := s.withConnection(func(conn *ldap.Conn) error {
		// Build filter from attributes
		var filterParts []string
		for attr, value := range attributes {
			filterParts = append(filterParts, fmt.Sprintf("(%s=*%s*)", ldap.EscapeFilter(attr), ldap.EscapeFilter(value)))
		}

		var filter string
		if objectClass != "" {
			filter = fmt.Sprintf("(&(objectClass=%s)(&%s))", objectClass, strings.Join(filterParts, ""))
		} else {
			filter = fmt.Sprintf("(&%s)", strings.Join(filterParts, ""))
		}

		searchRequest := ldap.NewSearchRequest(
			s.config.BaseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0, 0, false,
			filter,
			[]string{"*"}, // All attributes
			nil,
		)

		sr, err := conn.Search(searchRequest)
		if err != nil {
			return fmt.Errorf("LDAP search failed: %w", err)
		}

		results = make([]*SearchResult, 0, len(sr.Entries))
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

		return nil
	})

	return results, err
}
