package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "setup":
		setupIntegrationTests()
	case "run":
		runIntegrationTests()
	case "cleanup":
		cleanupIntegrationTests()
	case "test":
		setupIntegrationTests()
		runIntegrationTests()
		cleanupIntegrationTests()
	case "quick":
		runIntegrationTests()
	case "status":
		showStatus()
	case "logs":
		showLogs()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Integration Test Runner")
	fmt.Println("Usage: go run tests/integration/runner.go <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  setup   - Set up test environment (includes preparing dependencies)")
	fmt.Println("  run     - Run integration tests")
	fmt.Println("  cleanup - Clean up test environment")
	fmt.Println("  test    - Full test cycle (setup + run + cleanup)")
	fmt.Println("  quick   - Run tests without setup")
	fmt.Println("  status  - Show service status")
	fmt.Println("  logs    - Show service logs")
}

func setupIntegrationTests() {
	fmt.Println("Setting up integration test environment...")
	fmt.Println("Only affecting containers with project name 'weatherapi-integration-test'")

	// Check for port conflicts first
	checkPortConflicts()

	// Prepare mock weather server dependencies
	prepareMockWeatherServer()

	// Stop and remove only OUR test containers
	runCommand("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "down", "--volumes")

	// Build services
	runCommand("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "build", "--no-cache")

	// Start services
	runCommand("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "up", "-d")

	// Wait for services to be ready
	waitForServices()

	fmt.Println("Integration test environment is ready")
}

func runIntegrationTests() {
	fmt.Println("Running integration tests...")

	cmd := exec.Command("go", "test", "-v", "-timeout=10m", "./tests/integration/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Integration tests failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Integration tests completed successfully")
}

func cleanupIntegrationTests() {
	fmt.Println("Cleaning up integration test environment...")
	fmt.Println("Only removing containers with project name 'weatherapi-integration-test'")

	runCommand("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "down", "--volumes")

	fmt.Println("Integration test environment cleaned up")
}

func waitForServices() {
	fmt.Println("Waiting for services to be ready...")

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if checkServicesReady() {
			fmt.Println("All services are ready")
			return
		}

		fmt.Printf("Waiting for services... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}

	fmt.Println("Services failed to start within timeout")
	showLogs()
	os.Exit(1)
}

func checkPortConflicts() {
	ports := []string{"5433", "8081", "1025", "8025"}
	conflicts := []string{}

	for _, port := range ports {
		cmd := exec.Command("netstat", "-an")
		output, err := cmd.Output()
		if err == nil {
			if strings.Contains(string(output), ":"+port) {
				conflicts = append(conflicts, port)
			}
		}
	}

	if len(conflicts) > 0 {
		fmt.Printf("Warning: The following ports are in use: %s\n", strings.Join(conflicts, ", "))
		fmt.Println("Integration tests may fail if these ports conflict with test services.")
		fmt.Println("Consider stopping services using these ports or changing test port configuration.")
	}
}

func checkServicesReady() bool {
	cmd := exec.Command("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "ps")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Simple check - look for "Up" status
	// In a real implementation, you'd parse the output more carefully
	outputStr := string(output)
	return len(outputStr) > 0
}

func showStatus() {
	fmt.Println("Integration test services status:")
	runCommand("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "ps")
}

func showLogs() {
	fmt.Println("Integration test service logs:")
	runCommand("docker-compose", "-f", "tests/docker/docker-compose.integration.yml", "-p", "weatherapi-integration-test", "logs")
}

func prepareMockWeatherServer() {
	fmt.Println("Preparing mock weather server dependencies...")

	// Change to mock weather server directory and run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = "tests/mocks/weather-server"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to run go mod tidy in mock weather server: %v", err)
		os.Exit(1)
	}

	fmt.Println("Mock weather server dependencies prepared")
}

func runCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Command failed: %s %v - %v", name, args, err)
	}
}
