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
	// Check for LDAP_PASSWORD environment variable
	if os.Getenv("LDAP_PASSWORD") == "" {
		fmt.Println("WARNING: LDAP_PASSWORD environment variable is not set!")
		fmt.Println("Please set it with: export LDAP_PASSWORD='your-password'")
		fmt.Println("Attempting to continue anyway...")
	}

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

	fmt.Println("\nTesting LDAP connection...")
	fmt.Printf("Server: %s\n", config.LDAP.Server)
	fmt.Printf("Base DN: %s\n", config.LDAP.BaseDN)
	fmt.Printf("Bind DN: %s\n", config.LDAP.BindDN)
	fmt.Printf("Use TLS: %v\n", config.LDAP.UseTLS)
	fmt.Printf("Timeout: %s\n", config.LDAP.Timeout)
	if config.LDAP.BindPassword != "" {
		fmt.Printf("Password: ****** (%d chars)\n", len(config.LDAP.BindPassword))
	} else {
		fmt.Println("Password: <empty>")
	}

	// Create service
	service, err := ldap.NewService(&config.LDAP)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer service.Close()

	fmt.Println("✓ Connection successful!")

	// Test search_user
	fmt.Println("\nTesting search_user for 'samuel.kelemen'...")
	users, err := service.SearchUser("samuel.kelemen")
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Found %d user(s)\n", len(users))
	for i, user := range users {
		fmt.Printf("\nUser %d:\n", i+1)
		fmt.Printf("  DN: %s\n", user.DN)
		fmt.Printf("  Username: %s\n", user.Username)
		fmt.Printf("  Email: %s\n", user.Email)
		fmt.Printf("  Display Name: %s\n", user.DisplayName)
		fmt.Printf("  Status: %s\n", user.Status)
		fmt.Printf("  Department: %s\n", user.Department)
	}

	// Test get_user_details
	if len(users) > 0 {
		fmt.Println("\nTesting get_user_details...")
		details, err := service.GetUserDetails(users[0].Username)
		if err != nil {
			log.Fatalf("GetUserDetails failed: %v", err)
		}

		fmt.Printf("Detailed info for %s:\n", details.Username)
		fmt.Printf("  First Name: %s\n", details.FirstName)
		fmt.Printf("  Last Name: %s\n", details.LastName)
		fmt.Printf("  Title: %s\n", details.Title)
		fmt.Printf("  Company: %s\n", details.Company)
		fmt.Printf("  Phone: %s\n", details.Phone)
		fmt.Printf("  Groups: %d\n", len(details.MemberOf))
	}

	// Test search_group
	fmt.Println("\nTesting search_group for 'platform'...")
	groups, err := service.SearchGroup("platform")
	if err != nil {
		log.Fatalf("SearchGroup failed: %v", err)
	}

	fmt.Printf("Found %d group(s)\n", len(groups))
	for i, group := range groups {
		if i >= 5 {
			fmt.Printf("... and %d more\n", len(groups)-5)
			break
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, group.Name, group.DN)
	}

	// Test new tools - search_ou
	fmt.Println("\nTesting search_ou...")
	ous, err := service.SearchOU("Users")
	if err != nil {
		log.Fatalf("SearchOU failed: %v", err)
	}

	fmt.Printf("Found %d OU(s)\n", len(ous))
	for i, ou := range ous {
		if i >= 3 {
			fmt.Printf("... and %d more\n", len(ous)-3)
			break
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, ou.Name, ou.DN)
	}

	// Test search_by_attributes
	fmt.Println("\nTesting search_by_attributes for department='Security'...")
	results, err := service.SearchByAttributes(
		map[string]string{"department": "Security"},
		"user",
	)
	if err != nil {
		log.Fatalf("SearchByAttributes failed: %v", err)
	}

	fmt.Printf("Found %d result(s)\n", len(results))
	for i, result := range results {
		if i >= 3 {
			fmt.Printf("... and %d more\n", len(results)-3)
			break
		}
		fmt.Printf("  %d. %s\n", i+1, result.DN)
	}

	fmt.Println("\n✓ All tests passed!")
}
