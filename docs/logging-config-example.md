# Logging and Observability Configuration
# Add these environment variables to your docker-compose.microservices.yml services

# Example environment variables for logging configuration:
# - ENVIRONMENT=production|development|staging
# - LOG_LEVEL=debug|info|warn|error  
# - LOG_SAMPLING_DEBUG=0.01    # Sample 1% of debug logs
# - LOG_SAMPLING_INFO=0.1      # Sample 10% of info logs  
# - LOG_SAMPLING_WARN=0.8      # Sample 80% of warn logs
# - LOG_SAMPLING_ERROR=1.0     # Sample 100% of error logs (never sample errors)

# For production deployments, add to each service:
environment:
  - ENVIRONMENT=production
  - LOG_LEVEL=info
  - LOG_SAMPLING_DEBUG=0.01
  - LOG_SAMPLING_INFO=0.1
  - LOG_SAMPLING_WARN=0.8
  - LOG_SAMPLING_ERROR=1.0

# For development, use:
environment:
  - ENVIRONMENT=development
  - LOG_LEVEL=debug
  - LOG_SAMPLING_DEBUG=1.0
  - LOG_SAMPLING_INFO=1.0
  - LOG_SAMPLING_WARN=1.0
  - LOG_SAMPLING_ERROR=1.0
