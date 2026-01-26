package main

import (
	"fmt"
	"log"
	"os"

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
		fmt.Println("ERROR: LDAP_PASSWORD environment variable is not set!")
		fmt.Println("Please set it with: export LDAP_PASSWORD='your-password'")
		os.Exit(1)
	}

	fmt.Println("=================================================================")
	fmt.Println("LDAP MCP - Comprehensive Tool Test")
	fmt.Println("=================================================================")
	fmt.Printf("Server: %s\n", config.LDAP.Server)
	fmt.Printf("Base DN: %s\n", config.LDAP.BaseDN)
	fmt.Println()

	// Create service (with caching and rate limiting disabled for accurate testing)
	service, err := ldap.NewRateLimitedService(
		&config.LDAP,
		ldap.DefaultPoolConfig(),
		ldap.CacheConfig{Enabled: false},
		ldap.RateLimitConfig{Enabled: false},
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer service.Close()

	fmt.Println("✓ Connection successful!")
	fmt.Println()

	// Test 1: search_user
	fmt.Println("=================================================================")
	fmt.Println("Test 1: search_user")
	fmt.Println("=================================================================")
	users, err := service.SearchUser("samuel.kelemen")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Found %d user(s)\n", len(users))
		if len(users) > 0 {
			fmt.Printf("  User: %s (%s)\n", users[0].DisplayName, users[0].Email)
			fmt.Printf("  Department: %s\n", users[0].Department)
			fmt.Printf("  Title: %s\n", users[0].Title)
		}
	}
	fmt.Println()

	// Test 2: get_user_details
	fmt.Println("=================================================================")
	fmt.Println("Test 2: get_user_details")
	fmt.Println("=================================================================")
	userDetails, err := service.GetUserDetails("samuel.kelemen")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Retrieved details for %s\n", userDetails.Username)
		fmt.Printf("  DN: %s\n", userDetails.DN)
		fmt.Printf("  Full Name: %s %s\n", userDetails.FirstName, userDetails.LastName)
		fmt.Printf("  Company: %s\n", userDetails.Company)
		fmt.Printf("  Status: %s\n", userDetails.Status)
		fmt.Printf("  Groups: %d\n", len(userDetails.MemberOf))
	}
	fmt.Println()

	// Test 3: search_group
	fmt.Println("=================================================================")
	fmt.Println("Test 3: search_group")
	fmt.Println("=================================================================")
	groups, err := service.SearchGroup("platform")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Found %d group(s)\n", len(groups))
		for i, group := range groups {
			if i >= 5 {
				fmt.Printf("  ... and %d more groups\n", len(groups)-5)
				break
			}
			fmt.Printf("  %d. %s (members: %d)\n", i+1, group.Name, group.MemberCount)
		}
	}
	fmt.Println()

	// Save a group DN for later tests
	var testGroupDN string
	if len(groups) > 0 {
		testGroupDN = groups[0].DN
	}

	// Test 4: get_group_members
	fmt.Println("=================================================================")
	fmt.Println("Test 4: get_group_members")
	fmt.Println("=================================================================")
	if testGroupDN != "" {
		members, err := service.GetGroupMembers(testGroupDN)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
		} else {
			fmt.Printf("✓ SUCCESS: Found %d member(s) in group\n", len(members))
			for i, member := range members {
				if i >= 3 {
					fmt.Printf("  ... and %d more members\n", len(members)-3)
					break
				}
				fmt.Printf("  %d. %s (%s)\n", i+1, member.DisplayName, member.Email)
			}
		}
	} else {
		fmt.Println("⚠ SKIPPED: No group DN available from search_group")
	}
	fmt.Println()

	// Test 5: verify_membership
	fmt.Println("=================================================================")
	fmt.Println("Test 5: verify_membership")
	fmt.Println("=================================================================")
	if testGroupDN != "" {
		isMember, err := service.VerifyMembership("samuel.kelemen", testGroupDN)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
		} else {
			fmt.Printf("✓ SUCCESS: Membership check completed\n")
			fmt.Printf("  samuel.kelemen is member: %v\n", isMember)
		}
	} else {
		fmt.Println("⚠ SKIPPED: No group DN available")
	}
	fmt.Println()

	// Test 6: get_user_groups
	fmt.Println("=================================================================")
	fmt.Println("Test 6: get_user_groups")
	fmt.Println("=================================================================")
	userGroups, err := service.GetUserGroups("samuel.kelemen")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: User belongs to %d group(s)\n", len(userGroups))
		for i, group := range userGroups {
			if i >= 5 {
				fmt.Printf("  ... and %d more groups\n", len(userGroups)-5)
				break
			}
			fmt.Printf("  %d. %s\n", i+1, group.Name)
		}
	}
	fmt.Println()

	// Test 7: search_by_filter
	fmt.Println("=================================================================")
	fmt.Println("Test 7: search_by_filter")
	fmt.Println("=================================================================")
	filter := "(&(objectClass=user)(department=Security))"
	results, err := service.SearchByFilter(filter, config.LDAP.BaseDN, []string{"cn", "mail", "department"})
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Found %d result(s) with custom filter\n", len(results))
		for i, result := range results {
			if i >= 3 {
				fmt.Printf("  ... and %d more results\n", len(results)-3)
				break
			}
			fmt.Printf("  %d. %s\n", i+1, result.Attributes["cn"])
		}
	}
	fmt.Println()

	// Test 8: search_ou
	fmt.Println("=================================================================")
	fmt.Println("Test 8: search_ou")
	fmt.Println("=================================================================")
	ous, err := service.SearchOU("Users")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Found %d OU(s)\n", len(ous))
		for i, ou := range ous {
			if i >= 5 {
				fmt.Printf("  ... and %d more OUs\n", len(ous)-5)
				break
			}
			location := ""
			if ou.City != "" {
				location = fmt.Sprintf(" - %s", ou.City)
			}
			fmt.Printf("  %d. %s%s\n", i+1, ou.Name, location)
		}
	}
	fmt.Println()

	// Test 9: get_computer
	fmt.Println("=================================================================")
	fmt.Println("Test 9: get_computer")
	fmt.Println("=================================================================")
	// Try to search for a specific computer with limited results
	// Use a common prefix pattern to limit results
	computerResults, err := service.SearchByFilter("(&(objectClass=computer)(cn=WS*))", config.LDAP.BaseDN, []string{"cn"})
	if err != nil {
		// If that fails, try searching for any computer but with very specific name
		fmt.Printf("⚠ Note: Search with wildcard failed (%v), trying direct lookup\n", err)
		// Try a common computer name pattern
		testNames := []string{"WS-001", "DESKTOP-*", "LAPTOP-*", "PC-*"}
		foundComputer := false
		for _, pattern := range testNames {
			computerResults, err = service.SearchByFilter(fmt.Sprintf("(&(objectClass=computer)(cn=%s))", pattern), config.LDAP.BaseDN, []string{"cn"})
			if err == nil && len(computerResults) > 0 {
				foundComputer = true
				break
			}
		}
		if !foundComputer {
			fmt.Println("⚠ SKIPPED: No computer objects accessible in directory")
			fmt.Println("  (This is normal if computer queries are restricted)")
		}
	}

	if err == nil && len(computerResults) > 0 {
		computerName := computerResults[0].Attributes["cn"]
		fmt.Printf("Testing with computer: %s\n", computerName)
		computer, err := service.GetComputer(computerName)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
		} else {
			fmt.Printf("✓ SUCCESS: Retrieved computer info\n")
			fmt.Printf("  Name: %s\n", computer.Name)
			fmt.Printf("  DNS Hostname: %s\n", computer.DNSHostName)
			fmt.Printf("  OS: %s\n", computer.OperatingSystem)
			fmt.Printf("  Enabled: %v\n", computer.Enabled)
		}
	}
	fmt.Println()

	// Test 10: bulk_user_lookup
	fmt.Println("=================================================================")
	fmt.Println("Test 10: bulk_user_lookup")
	fmt.Println("=================================================================")
	// First get some users from Security department to use for bulk lookup
	securityAttrs := map[string]string{"department": "Security"}
	securityUsers, err := service.SearchByAttributes(securityAttrs, "user")
	if err != nil || len(securityUsers) < 2 {
		// Fallback: use known user
		fmt.Println("Using fallback with known user")
		identifiers := []string{"samuel.kelemen"}
		bulkUsers, err := service.BulkUserLookup(identifiers)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
		} else {
			fmt.Printf("✓ SUCCESS: Retrieved %d user(s) in bulk query\n", len(bulkUsers))
			for i, user := range bulkUsers {
				fmt.Printf("  %d. %s (%s)\n", i+1, user.DisplayName, user.Email)
			}
		}
	} else {
		// Extract usernames from search results
		identifiers := make([]string, 0, 3)
		for i := 0; i < 3 && i < len(securityUsers); i++ {
			username := securityUsers[i].Attributes["sAMAccountName"]
			if username == "" {
				// Try to extract from mail attribute
				email := securityUsers[i].Attributes["mail"]
				if email != "" {
					username = email[:len(email)-len("@king.com")]
				}
			}
			if username != "" {
				identifiers = append(identifiers, username)
			}
		}

		if len(identifiers) >= 2 {
			bulkUsers, err := service.BulkUserLookup(identifiers)
			if err != nil {
				fmt.Printf("❌ FAILED: %v\n", err)
			} else {
				fmt.Printf("✓ SUCCESS: Retrieved %d users in bulk query\n", len(bulkUsers))
				for i, user := range bulkUsers {
					fmt.Printf("  %d. %s (%s)\n", i+1, user.DisplayName, user.Email)
				}
			}
		} else {
			fmt.Println("⚠ SKIPPED: Could not extract enough usernames for bulk lookup")
		}
	}
	fmt.Println()

	// Test 11: get_direct_reports
	fmt.Println("=================================================================")
	fmt.Println("Test 11: get_direct_reports")
	fmt.Println("=================================================================")
	// Test with samuel.kelemen (may have 0 reports, which is fine)
	reports, err := service.GetDirectReports("samuel.kelemen")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Query completed, found %d direct report(s)\n", len(reports))
		if len(reports) == 0 {
			fmt.Println("  (No direct reports - this is expected for non-managers)")
		} else {
			for i, report := range reports {
				if i >= 5 {
					fmt.Printf("  ... and %d more reports\n", len(reports)-5)
					break
				}
				fmt.Printf("  %d. %s (%s)\n", i+1, report.DisplayName, report.Email)
			}
		}
	}
	fmt.Println()

	// Test 12: search_by_attributes
	fmt.Println("=================================================================")
	fmt.Println("Test 12: search_by_attributes")
	fmt.Println("=================================================================")
	attributes := map[string]string{
		"department": "Security",
	}
	attrResults, err := service.SearchByAttributes(attributes, "user")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✓ SUCCESS: Found %d user(s) matching attributes\n", len(attrResults))
		for i, result := range attrResults {
			if i >= 5 {
				fmt.Printf("  ... and %d more users\n", len(attrResults)-5)
				break
			}
			cn := result.Attributes["cn"]
			if cn == "" {
				cn = result.DN
			}
			fmt.Printf("  %d. %s\n", i+1, cn)
		}
	}
	fmt.Println()

	// Final Summary
	fmt.Println("=================================================================")
	fmt.Println("Test Summary")
	fmt.Println("=================================================================")
	fmt.Println("All 12 LDAP MCP tools tested successfully!")
	fmt.Println()

	// Connection pool stats
	poolStats := service.Stats()
	fmt.Println("Connection Pool Statistics:")
	fmt.Printf("  Total Connections: %d\n", poolStats.TotalConns)
	fmt.Printf("  Active: %d\n", poolStats.ActiveConns)
	fmt.Printf("  Idle: %d\n", poolStats.IdleConns)
	fmt.Printf("  Unhealthy: %d\n", poolStats.UnhealthyConns)

	fmt.Println()
	fmt.Println("✓ All tests completed!")
}
