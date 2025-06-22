# Testing Guide

## Prerequisites
- Git, Docker, Go 1.23+, Node.js (for E2E tests)

## Quick Start

### Install Dependencies
```bash
make -f Makefile.testing test-deps
```

### Run Tests

#### Individual Test Types
```bash
# Unit tests (fast, no dependencies)
make -f Makefile.testing test-unit

# Integration tests (with auto Docker setup)
make -f Makefile.testing test-integration

# E2E tests (full browser automation)
make -f Makefile.testing test-e2e
```

#### All Tests
```bash
# Run complete test suite
make -f Makefile.testing test-all
```

#### Code Quality
```bash
# Run linting
make -f Makefile.testing lint
```

#### Cleanup
```bash
# Clean up all test environments
make -f Makefile.testing test-clean
```

## Help
```bash
# See all available commands
make -f Makefile.testing help
```

Each command automatically handles Docker service setup and cleanup.
