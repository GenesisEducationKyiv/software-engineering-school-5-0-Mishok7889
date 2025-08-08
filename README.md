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

## Caching and Performance

The application includes sophisticated caching with comprehensive monitoring:

### Cache Types
- **Memory Cache**: Default in-memory caching for single-instance deployments
- **Redis Cache**: Distributed caching for multi-instance deployments

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

## Observability and Monitoring

### Logging System

The application implements comprehensive structured logging with the following features:

#### Log Levels
- **Debug**: Detailed diagnostic information for troubleshooting
- **Info**: General operational messages and business events
- **Warn**: Potentially harmful situations and performance issues
- **Error**: Critical errors that need immediate attention

#### Log Sampling
To manage log volume in production, the system implements intelligent sampling:
- **Debug logs**: 1% sampling rate (high volume, used for detailed troubleshooting)
- **Info logs**: 10% sampling rate (general operations)
- **Warning logs**: 80% sampling rate (potential issues)
- **Error logs**: 100% sampling rate (never sampled - all errors are critical)

**Special Event Logging** (never sampled):
- Business events (subscriptions, cancellations)
- Security events (authentication failures, suspicious activity)
- Critical errors (service failures, data corruption)

#### Correlation ID Tracking
All requests are tracked with correlation IDs that flow through:
- HTTP requests via `X-Correlation-ID` header
- gRPC calls via metadata
- Message broker events
- Database operations

### Metrics Collection

The application exposes Prometheus metrics at `/metrics` endpoint covering:

#### HTTP Metrics
- `http_requests_total` - Total HTTP requests by method, endpoint, status
- `http_request_duration_seconds` - Request latency distribution
- `http_response_size_bytes` - Response size distribution

#### Weather Service Metrics
- `weather_requests_total` - Weather API requests by provider, city, status
- `weather_request_duration_seconds` - Weather API response times
- `weather_provider_failures_total` - Provider failure counts by type

#### Cache Metrics
- `cache_operations_total` - Cache operations (hit/miss) by type
- `cache_hit_ratio` - Real-time cache hit ratio
- `cache_size_bytes` - Current cache memory usage

#### Database Metrics
- `database_connections_active` - Active database connections
- `database_query_duration_seconds` - Query execution times
- `database_queries_total` - Query counts by operation and status

#### Message Broker Metrics
- `message_broker_published_total` - Published messages by topic
- `message_broker_consumed_total` - Consumed messages by topic and consumer group
- `message_broker_errors_total` - Message processing errors

#### Business Metrics
- `subscriptions_active` - Current active subscriptions
- `subscriptions_total` - Subscription operations by type and frequency
- `emails_sent_total` - Email delivery metrics by type and status

### Recommended Alerting Rules

Based on the implemented metrics and logging, the following alerts are crucial for production monitoring:

#### Critical Alerts (Immediate Response Required)

1. **Service Availability**
   - Alert: `http_requests_total` 5xx error rate > 5% for 2 minutes
   - Reason: High error rate indicates service degradation or outage
   - Action: Immediate investigation required

2. **Database Connectivity**
   - Alert: `database_connections_active` = 0 for 1 minute
   - Reason: Complete database failure will break all functionality
   - Action: Immediate database investigation

3. **Weather Provider Failures**
   - Alert: All weather providers failing for 5 minutes
   - Reason: Core service functionality unavailable
   - Action: Check provider APIs and failover mechanism

4. **Message Broker Down**
   - Alert: `message_broker_errors_total` spike or no messages processed for 10 minutes
   - Reason: Notification system will stop working
   - Action: Restart message broker, check queue health

#### Warning Alerts (Response Within 15-30 Minutes)

5. **High Response Times**
   - Alert: `http_request_duration_seconds` p95 > 2 seconds for 5 minutes
   - Reason: User experience degradation, potential performance issues
   - Action: Investigate slow queries, cache performance, external API delays

6. **Low Cache Hit Ratio**
   - Alert: `cache_hit_ratio` < 70% for 10 minutes
   - Reason: Increased load on weather APIs, higher costs
   - Action: Check cache configuration, Redis connectivity, TTL settings

7. **Email Delivery Issues**
   - Alert: `emails_sent_total` with status="error" > 10% for 15 minutes
   - Reason: Users not receiving notifications
   - Action: Check SMTP configuration, email service health

8. **High Database Query Times**
   - Alert: `database_query_duration_seconds` p95 > 1 second for 10 minutes
   - Reason: Performance degradation, potential database issues
   - Action: Analyze slow queries, check database resources

#### Informational Alerts (Response Within 1-4 Hours)

9. **Weather Provider Degradation**
   - Alert: Single weather provider failing for 30 minutes
   - Reason: Reduced redundancy, potential future issues
   - Action: Monitor provider status, consider provider rotation

10. **Subscription Trends**
    - Alert: `subscriptions_total` cancellation rate > 20% daily
    - Reason: Potential service quality issues
    - Action: Analyze user feedback, service performance

11. **Log Error Patterns**
    - Alert: Error log volume increase > 50% compared to baseline
    - Reason: Potential emerging issues
    - Action: Analyze error patterns and trends

### Log Retention Policy

Our log retention strategy balances operational needs, compliance requirements, and storage costs:

#### Debug Logs
- **Retention Period**: 7 days
- **Rationale**: High volume due to detailed diagnostic information. Short retention suitable for immediate troubleshooting. With 1% sampling, storage requirements remain manageable.
- **Storage**: Fast SSD storage for quick access during active debugging
- **Cleanup**: Automated daily cleanup of logs older than 7 days

#### Info Logs
- **Retention Period**: 30 days
- **Rationale**: Contains business events and operational information needed for monthly reporting and trend analysis. 10% sampling provides sufficient data for business intelligence.
- **Storage**: Standard storage, compressed after 7 days
- **Cleanup**: Automated weekly cleanup of logs older than 30 days

#### Warning Logs
- **Retention Period**: 90 days (3 months)
- **Rationale**: Warning patterns help identify recurring issues and service degradation trends. 80% sampling ensures comprehensive coverage of potential problems.
- **Storage**: Standard storage, compressed after 14 days
- **Cleanup**: Automated monthly cleanup of logs older than 90 days

#### Error Logs
- **Retention Period**: 1 year
- **Rationale**: Critical for post-incident analysis, compliance, and pattern recognition. 100% retention (no sampling) ensures no critical errors are missed.
- **Storage**: Archived to cold storage after 30 days, compressed
- **Cleanup**: Manual review before deletion, automated after 1 year

#### Special Event Logs (Business, Security, Critical)
- **Retention Period**: 2+ years
- **Rationale**: 
  - **Business Events**: Required for audit trails and compliance (subscriptions, payments, data access)
  - **Security Events**: Essential for security audits and incident investigation
  - **Critical Errors**: May be needed for legal or compliance purposes
- **Storage**: Archived to long-term cold storage after 90 days
- **Cleanup**: Manual review required, compliance team approval for deletion

#### Implementation Strategy

1. **Automated Cleanup**: Cron jobs and log rotation policies handle routine cleanup
2. **Compression**: Logs older than 7 days are compressed (gzip) to save 60-80% storage
3. **Archival**: Logs older than 30 days moved to cheaper cold storage
4. **Monitoring**: Alerts on log storage usage and cleanup failures
5. **Backup**: Critical logs backed up to separate geographic region
6. **Access Control**: Strict access controls on archived logs, audit trail for access

#### Cost Optimization

- **Sampling**: Reduces log volume by 80-90% while maintaining observability
- **Compression**: Saves 60-80% storage costs
- **Tiered Storage**: Hot → Warm → Cold → Archive reduces costs by up to 95%
- **Automated Lifecycle**: Prevents manual overhead and ensures compliance
