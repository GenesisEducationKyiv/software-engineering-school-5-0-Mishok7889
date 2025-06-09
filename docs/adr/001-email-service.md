# ADR-001: Choosing an Email Service 

---

**Status**: Accepted  
**Author**: Mykhailo Okhrimenko  
**Date**: 01.06.2025

---

## Context

The Weather Forecast API requires an email delivery system for subscription confirmations, welcome emails, weather updates, and unsubscribe notices. The service must be reliable, cost-effective, easy to integrate with Go, and secure.

## Solution

### Options Considered

1. **MailSlurp**: Testing-oriented service with API capabilities
2. **Gmail SMTP**: Uses Google's SMTP servers with standard protocols
3. **SendGrid**: Professional email delivery service with API
4. **AWS SES**: Cloud-based email service with pay-as-you-go pricing

### Evaluation

**MailSlurp**
- Pros:
  - Good for testing purposes (provides temporary email addresses and API for automated tests)
  - API-driven approach
  - Allows verification of email content in tests
- Cons:
  - "426 Upgrade Required" error when integrating with Go
  - Limited production use cases

**Gmail SMTP**
- Pros:
  - Reliable delivery
  - Simple integration with standard protocols
  - No additional service costs
- Cons:
  - Requires app password setup
  - Daily sending limits (2,000/day)
  - Tied to Google account

**SendGrid**
- Pros:
  - Built for production use cases
  - Strong deliverability
  - Comprehensive API
- Cons:
  - Additional costs
  - Extra service to manage

**AWS SES**
- Pros:
  - Excellent scalability
  - Cost-effective at higher volumes
- Cons:
  - More complex setup
  - Requires AWS knowledge

## Decision

We chose **Gmail SMTP** for the following reasons:

1. Simple integration with Go's standard library
2. Sufficient deliverability rates
3. No additional service costs

## Security Considerations

1. Using Google's App Password system instead of main account password
2. Storing credentials in environment variables
3. Communication using TLS encryption

## Consequences

**Positive**:
- Reliable delivery with simple implementation
- Sufficient for current project needs

**Negative**:
- Limited to Google's sending limits (2,000/day)
- Requires management of Google account security
- May need migration if application scales
