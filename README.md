# Weather Forecast API

A REST API service that allows users to subscribe to regular weather updates for their chosen cities.

## Project Overview

This service enables users to:
- Get current weather for any city
- Subscribe to weather updates (hourly or daily)
- Confirm subscriptions via email
- Unsubscribe from updates when no longer needed

Weather data is fetched from WeatherAPI.com and delivered to subscribers via email according to their preferred frequency.

## Technologies Used

- Go with Gin framework for API handling
- PostgreSQL for data storage
- GORM as ORM
- WeatherAPI.com for weather data
- Gmail SMTP for email delivery
- Docker and Docker Compose for containerization

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