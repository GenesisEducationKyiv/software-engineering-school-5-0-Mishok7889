#!/bin/bash
set -e

echo "[E2E] Starting E2E Test Runner..."

# Find project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# Verify we're in the right place
if [ ! -f "go.mod" ]; then
    echo "[ERROR] Cannot find go.mod - not in project root"
    echo "[ERROR] Current directory: $PWD"
    exit 1
fi

echo "[E2E] Project root found: $PWD"

# Configuration
COMPOSE_FILE="tests/e2e/docker/docker-compose.e2e.yml"
PROJECT_NAME="weatherapi-e2e"
E2E_DIR="tests/e2e"

# Check if compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "[ERROR] Docker compose file not found: $COMPOSE_FILE"
    exit 1
fi

echo "[E2E] Using compose file: $COMPOSE_FILE"

# Print usage if no command provided
print_usage() {
    echo "E2E Test Runner for Weather API"
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  setup       - Set up environment"
    echo "  test        - Run tests"
    echo "  full        - Setup + Test + Cleanup"
    echo "  cleanup     - Clean up"
    echo "  status      - Show status"
    echo "  logs        - Show logs"
}

# Check command line argument
if [ $# -eq 0 ]; then
    print_usage
    exit 1
fi

# Execute command based on argument
case "$1" in
    "setup")
        echo "[E2E] Setting up environment..."
        
        # Install Node dependencies
        echo "[E2E] Installing Node.js dependencies..."
        cd "$E2E_DIR"
        if [ ! -d "node_modules" ]; then
            npm install
        fi
        npx playwright install
        
        # Return to project root
        cd "$PROJECT_ROOT"
        
        # Setup Docker
        echo "[E2E] Setting up Docker services..."
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down --volumes --remove-orphans || true
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" build --no-cache
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d
        
        # Wait for services
        echo "[E2E] Waiting for services to be ready..."
        sleep 30
        echo "[E2E] Setup complete!"
        ;;
        
    "test")
        echo "[E2E] Running E2E tests..."
        cd "$E2E_DIR"
        npx playwright test
        if [ $? -ne 0 ]; then
            echo "[ERROR] Tests failed"
            exit 1
        fi
        echo "[E2E] Tests completed successfully!"
        ;;
        
    "full")
        echo "[E2E] Running full E2E cycle..."
        "$0" setup
        if [ $? -ne 0 ]; then exit 1; fi
        
        "$0" test
        if [ $? -ne 0 ]; then exit 1; fi
        
        "$0" cleanup
        ;;
        
    "cleanup")
        echo "[E2E] Cleaning up..."
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down --volumes --remove-orphans
        echo "[E2E] Cleanup complete!"
        ;;
        
    "status")
        echo "[E2E] Service status:"
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps
        ;;
        
    "logs")
        echo "[E2E] Service logs:"
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs
        ;;
        
    *)
        echo "[ERROR] Unknown command: $1"
        print_usage
        exit 1
        ;;
esac

echo "[E2E] Done."
