package email

import (
	"fmt"
	"log"
	"os"

	"github.com/resend/resend-go/v2"
)

// ResendEmailService implements EmailService using Resend Go SDK
type ResendEmailService struct {
	client    *resend.Client
	fromEmail string
	fromName  string
}

// NewResendEmailService creates a new Resend email service using the official SDK
func NewResendEmailService(apiKey, fromEmail, fromName string) EmailService {
	return &ResendEmailService{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (r *ResendEmailService) SendPasswordResetEmail(to, resetLink string, userName string) error {
	if r.client == nil {
		return fmt.Errorf("resend client not initialized")
	}

	if r.fromEmail == "" {
		return fmt.Errorf("email from address not configured")
	}

	subject := "Reset Your Password"
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <title>Password Reset Request</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f5f7fa; line-height: 1.6;">
  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" style="background-color: #f5f7fa; padding: 40px 20px;">
    <tr>
      <td align="center">
        <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="600" style="max-width: 600px; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); overflow: hidden;">
          <!-- Header -->
          <tr>
            <td style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 40px 40px 30px; text-align: center;">
              <h1 style="margin: 0; color: #ffffff; font-size: 28px; font-weight: 600; letter-spacing: -0.5px;">🔒 Password Reset</h1>
            </td>
          </tr>
          
          <!-- Content -->
          <tr>
            <td style="padding: 40px;">
              <h2 style="margin: 0 0 20px 0; color: #1a202c; font-size: 22px; font-weight: 600;">Hello %s,</h2>
              
              <p style="margin: 0 0 20px 0; color: #4a5568; font-size: 16px;">
                We received a request to reset your password for your account. Click the button below to create a new password:
              </p>
              
              <!-- CTA Button -->
              <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%%" style="margin: 30px 0;">
                <tr>
                  <td align="center">
                    <a href="%s" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #ffffff; text-decoration: none; padding: 16px 40px; border-radius: 8px; font-weight: 600; font-size: 16px; box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4); transition: all 0.3s ease;">Reset Password</a>
                  </td>
                </tr>
              </table>
              
              <!-- Alternative Link -->
              <div style="background-color: #f7fafc; border-left: 4px solid #667eea; padding: 16px; border-radius: 4px; margin: 30px 0;">
                <p style="margin: 0 0 10px 0; color: #4a5568; font-size: 14px; font-weight: 600;">Can't click the button? Copy and paste this link:</p>
                <p style="margin: 0; word-break: break-all; color: #667eea; font-size: 13px; font-family: 'Courier New', monospace;">%s</p>
              </div>
              
              <!-- Security Notice -->
              <div style="background-color: #fff5f5; border: 1px solid #feb2b2; border-radius: 6px; padding: 16px; margin: 30px 0;">
                <p style="margin: 0; color: #c53030; font-size: 14px; line-height: 1.5;">
                  <strong>⏰ Important:</strong> This link will expire in <strong>1 hour</strong> for your security. If you didn't request a password reset, you can safely ignore this email and your password will remain unchanged.
                </p>
              </div>
              
              <!-- Footer Info -->
              <p style="margin: 30px 0 0 0; color: #718096; font-size: 13px; text-align: center; border-top: 1px solid #e2e8f0; padding-top: 20px;">
                This is an automated message. Please do not reply to this email.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, userName, resetLink, resetLink)

	// TODO: Update to use dynamic values (fromField, to) when going to production
	// Format FROM field exactly like Next.js: "Name <email@domain.com>"
	// fromField := r.fromEmail
	// if r.fromName != "" {
	// 	fromField = fmt.Sprintf("%s <%s>", r.fromName, r.fromEmail)
	// }

	// Use the official Resend Go SDK
	params := &resend.SendEmailRequest{
		From:    " Chalachew Tsegaye Portfolio <onboarding@resend.dev>", // fromField
		To:      []string{"tsegayechalachew0@gmail.com"},                 // []string{to},
		Subject: subject,
		Html:    htmlBody,
	}

	sent, err := r.client.Emails.Send(params)
	if err != nil {
		log.Printf("[EMAIL] [ERROR] Failed to send email: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	if sent != nil && sent.Id != "" {
		log.Printf("[EMAIL] [SUCCESS] Email sent successfully! Resend ID: %s", sent.Id)
	}

	return nil
}

// stripQuotes removes surrounding quotes from a string if present
func stripQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// NewEmailServiceFromConfig creates an email service based on configuration
// This makes it easy to switch between providers
func NewEmailServiceFromConfig(provider, apiKey, fromEmail, fromName string) EmailService {
	if provider == "" {
		provider = "resend" // Default
	}

	// Strip quotes from config values (common in config files)
	apiKey = stripQuotes(apiKey)
	fromEmail = stripQuotes(fromEmail)
	fromName = stripQuotes(fromName)
	if fromName == "" {
		fromName = "Employee List"
	}

	log.Printf("[EMAIL_INIT] Email service initialized - Provider: %s, From: %s", provider, fromEmail)

	switch provider {
	case "resend":
		return NewResendEmailService(apiKey, fromEmail, fromName)
	case "s3":
		// Future: Implement S3 email service
		// return NewS3EmailService(...)
		log.Printf("[EMAIL_INIT] [ERROR] S3 email provider not yet implemented, falling back to Resend")
		return NewResendEmailService(apiKey, fromEmail, fromName)
	default:
		// Default to Resend if not specified
		log.Printf("[EMAIL_INIT] Unknown provider '%s', defaulting to Resend", provider)
		return NewResendEmailService(apiKey, fromEmail, fromName)
	}
}

// NewEmailServiceFromEnv creates an email service based on environment configuration
// This makes it easy to switch between providers
func NewEmailServiceFromEnv() EmailService {
	provider := os.Getenv("EMAIL_PROVIDER")
	if provider == "" {
		provider = "resend" // Default
	}
	
	// Strip quotes from env vars (common in .env files)
	apiKey := stripQuotes(os.Getenv("RESEND_API_KEY"))
	fromEmail := stripQuotes(os.Getenv("EMAIL_FROM"))
	fromName := stripQuotes(os.Getenv("EMAIL_FROM_NAME"))
	if fromName == "" {
		fromName = "Employee List"
	}
	
	log.Printf("[EMAIL_INIT] Email service initialized - Provider: %s, From: %s", provider, fromEmail)
	
	switch provider {
	case "resend":
		return NewResendEmailService(apiKey, fromEmail, fromName)
	case "s3":
		// Future: Implement S3 email service
		// return NewS3EmailService(...)
		log.Printf("[EMAIL_INIT] [ERROR] S3 email provider not yet implemented, falling back to Resend")
		return NewResendEmailService(apiKey, fromEmail, fromName)
	default:
		// Default to Resend if not specified
		log.Printf("[EMAIL_INIT] Unknown provider '%s', defaulting to Resend", provider)
		return NewResendEmailService(apiKey, fromEmail, fromName)
	}
}

