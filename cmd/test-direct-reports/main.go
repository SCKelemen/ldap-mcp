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
	data, err := os.ReadFile(".ldap-mcp.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	config.LDAP.BindPassword = os.ExpandEnv(config.LDAP.BindPassword)

	service, err := ldap.NewService(&config.LDAP)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer service.Close()

	fmt.Println("=================================================================")
	fmt.Println("Testing get_direct_reports with Gabriel Linero")
	fmt.Println("=================================================================")

	// First, get Gabriel's details
	fmt.Println("\n1. Getting manager details...")
	manager, err := service.GetUserDetails("gabriel.linero")
	if err != nil {
		log.Fatalf("Failed to get manager details: %v", err)
	}

	fmt.Printf("Manager: %s (%s)\n", manager.DisplayName, manager.Email)
	fmt.Printf("Title: %s\n", manager.Title)
	fmt.Printf("Department: %s\n", manager.Department)
	fmt.Printf("DN: %s\n", manager.DN)

	// Get direct reports
	fmt.Println("\n2. Getting direct reports...")
	reports, err := service.GetDirectReports("gabriel.linero")
	if err != nil {
		log.Fatalf("Failed to get direct reports: %v", err)
	}

	fmt.Printf("\n✓ Found %d direct report(s):\n\n", len(reports))

	for i, report := range reports {
		fmt.Printf("%d. %s\n", i+1, report.DisplayName)
		fmt.Printf("   Email: %s\n", report.Email)
		fmt.Printf("   Username: %s\n", report.Username)
		if report.Title != "" {
			fmt.Printf("   Title: %s\n", report.Title)
		}
		if report.Department != "" {
			fmt.Printf("   Department: %s\n", report.Department)
		}
		fmt.Println()
	}

	// Verify samuel.kelemen is in the list
	fmt.Println("3. Verifying samuel.kelemen is in direct reports...")
	found := false
	for _, report := range reports {
		if report.Username == "samuel.kelemen" {
			found = true
			break
		}
	}

	if found {
		fmt.Println("✓ samuel.kelemen IS a direct report of gabriel.linero")
	} else {
		fmt.Println("⚠ samuel.kelemen is NOT in the direct reports list")
		fmt.Println("  (This might be expected if the reporting structure is different)")
	}

	// Also check Samuel's manager attribute
	fmt.Println("\n4. Checking samuel.kelemen's manager attribute...")
	samuel, err := service.GetUserDetails("samuel.kelemen")
	if err != nil {
		log.Fatalf("Failed to get samuel.kelemen details: %v", err)
	}

	fmt.Printf("Samuel's manager DN: %s\n", samuel.Manager)
	fmt.Printf("Gabriel's DN: %s\n", manager.DN)

	if samuel.Manager == manager.DN {
		fmt.Println("✓ samuel.kelemen's manager attribute matches gabriel.linero")
	} else {
		fmt.Println("⚠ Manager DNs don't match")
		if samuel.Manager != "" {
			fmt.Printf("  Samuel's manager is: %s\n", samuel.Manager)
		} else {
			fmt.Println("  Samuel has no manager attribute set")
		}
	}

	fmt.Println("\n✓ Direct reports test completed!")
}
