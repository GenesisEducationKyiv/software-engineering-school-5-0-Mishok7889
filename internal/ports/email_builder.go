package ports

// EmailBuilder defines the contract for building email content
type EmailBuilder interface {
	BuildConfirmationEmail(city, confirmationURL string) (EmailParams, error)
	BuildWelcomeEmail(city, frequency, unsubscribeURL string) (EmailParams, error)
	BuildUnsubscribeEmail(city string) (EmailParams, error)
}
