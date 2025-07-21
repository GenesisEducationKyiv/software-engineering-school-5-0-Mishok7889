package ports

// EmailBuilder defines the contract for building email content
type EmailBuilder interface {
	BuildConfirmationEmail(city, token string) (EmailParams, error)
	BuildWelcomeEmail(city, frequency, unsubscribeToken string) (EmailParams, error)
	BuildUnsubscribeEmail(city string) (EmailParams, error)
}
