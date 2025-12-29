package email

// EmailService interface for sending emails
// This abstraction allows easy switching between email providers (Resend, S3, SES, etc.)
type EmailService interface {
	SendPasswordResetEmail(to, resetLink string, userName string) error
}

