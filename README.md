# Weather Forecast API

A REST API service that allows users to subscribe to regular weather updates for their chosen cities.

## Project Overview

This service enables users to:
- Get current weather for any city
- Subscribe to weather updates (hourly or daily)
- Confirm subscriptions via email
- Unsubscribe from updates when no longer needed

Weather data is fetched from multiple providers with automatic failover and delivered to subscribers via email according to their preferred frequency.

## Technologies Used

- Go with Gin framework for API handling
- PostgreSQL for data storage
- GORM as ORM
- Multiple weather providers (WeatherAPI.com, OpenWeatherMap, AccuWeather) with automatic failover
- Redis for distributed caching (with memory cache fallback)
- Prometheus for metrics collection and monitoring
- Gmail SMTP for email delivery
- Docker and Docker Compose for containerization

## Architecture

The application implements Gang of Four design patterns:
- **Chain of Responsibility**: Automatic failover between weather providers
- **Proxy Pattern**: Response caching to reduce API calls
- **Decorator Pattern**: Request/response logging

## Setup and Installation

### Prerequisites

- Go 1.21+
- PostgreSQL
- Docker and Docker Compose (optional)
- WeatherAPI.com API key
- Gmail account with app password for SMTP

### Configuration

Copy a `.env.example` file to `.env` in the root directory and update it with values of your preferences.

### Running with Docker

```bash
docker-compose up -d
```

### Running Locally

```bash
# Install dependencies
go mod download

# Run the application
go run main.go
```

## API Endpoints

- `GET /api/weather?city=cityname` - Get current weather for a city
- `POST /api/subscribe` - Subscribe to weather updates
- `GET /api/confirm/:token` - Confirm email subscription
- `GET /api/unsubscribe/:token` - Unsubscribe from weather updates
- `GET /api/metrics` - Get cache performance metrics (JSON format)
- `GET /metrics` - Prometheus metrics endpoint

## Caching and Monitoring

The application includes sophisticated caching with comprehensive monitoring:

### Cache Types
- **Memory Cache**: Default in-memory caching for single-instance deployments
- **Redis Cache**: Distributed caching for multi-instance deployments

### Metrics and Monitoring
- **Cache hit/miss ratios** with real-time tracking
- **Cache operation latency** monitoring
- **Prometheus metrics** for integration with monitoring systems
- **JSON metrics endpoint** for custom dashboards

### Cache Configuration

Configure caching via environment variables:

```bash
# Cache type (memory or redis)
CACHE_TYPE=redis

# Redis settings (when CACHE_TYPE=redis)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=your_redis_password
REDIS_DB=0
REDIS_DIAL_TIMEOUT=5
REDIS_READ_TIMEOUT=3
REDIS_WRITE_TIMEOUT=3
```

## Development

### Linting

This project uses golangci-lint v2 for code quality assurance.

Configuration notes:
- Linting is automatically run on all branches and pull requests via GitHub Actions

To run the linter locally:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6

# Run linter
golangci-lint run ./...
```

### Pre-commit Hooks

This repository includes pre-commit hooks to automatically run linting before each commit.

To set up pre-commit hooks:

```bash
# Install pre-commit
pip install pre-commit

# Install the git hooks
pre-commit install
```

## Problems during development

### Email Service

Initially, MailSlurp was considered for email delivery, but I encountered issues with error "426 Upgrade Required" when sending emails using their standard library fo Golang. I decided that Gmail SMTP provides reliable delivery for this application. However I should use personal account for deployment.

To use Gmail for sending emails:
1. Create a Google account or use an existing one
2. Enable 2-Step Verification
3. Create an App Password (Settings → Security → App passwords)
4. Use this password in the EMAIL_SMTP_PASSWORD environment variable

### Database Initialization

The application automatically handles database migrations on startup. However, ensure your PostgreSQL instance is properly configured and accessible before starting.

## Deployment

The application is deployed using **Google Cloud Platform (GCP)**. For the purpose of this project, the infrastructure was set up manually using a VM instance rather than Infrastructure as Code tools like Terraform or Ansible.

### Access Information

- **API URL**: [http://34.71.35.254:8080/](http://34.71.35.254:8080/)
- The above link also provides access to the **web interface** for subscribing to weather forecast notifications.
