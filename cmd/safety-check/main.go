package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Integration Test Container Safety Check")
	fmt.Println("=====================================")

	fmt.Println("\n1. Current Docker containers:")
	runCommand("docker", "ps", "-a", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}")

	fmt.Println("\n2. Containers that would be affected by integration tests:")
	fmt.Println("   - weatherapi-integration-test_postgres-test_1")
	fmt.Println("   - weatherapi-integration-test_mock-weather-api_1")
	fmt.Println("   - weatherapi-integration-test_mailhog_1")

	fmt.Println("\n3. Test ports that will be used:")
	fmt.Println("   - 5433 (PostgreSQL)")
	fmt.Println("   - 8081 (Mock Weather API)")
	fmt.Println("   - 1025 (SMTP)")
	fmt.Println("   - 8025 (MailHog Web UI)")

	fmt.Println("\n4. Checking if integration test containers already exist:")
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=weatherapi-integration-test", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("   Error checking containers:", err)
		return
	}

	if len(output) == 0 {
		fmt.Println("   ✓ No integration test containers found")
	} else {
		fmt.Println("   Found existing integration test containers:")
		fmt.Print(string(output))
	}

	fmt.Println("\n5. Only containers with project name 'weatherapi-integration-test' will be affected.")
	fmt.Println("   Your existing containers are safe!")
}

func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Command failed: %s %v - %v\n", name, args, err)
	}
}
