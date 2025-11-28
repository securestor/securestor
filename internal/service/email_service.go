package service

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

// EmailService handles sending emails
type EmailService struct {
	dialer   *gomail.Dialer
	from     string
	fromName string
	baseURL  string
}

// EmailConfig contains SMTP configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	BaseURL      string
	TLSEnabled   bool
}

// NewEmailService creates a new email service
func NewEmailService(config EmailConfig) *EmailService {
	dialer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	if !config.TLSEnabled {
		dialer.TLSConfig = nil
	}

	return &EmailService{
		dialer:   dialer,
		from:     config.FromEmail,
		fromName: config.FromName,
		baseURL:  config.BaseURL,
	}
}

// LoadEmailConfigFromEnv loads email configuration from environment variables
func LoadEmailConfigFromEnv() EmailConfig {
	port, _ := strconv.Atoi(getEnvWithDefault("SMTP_PORT", "587"))
	tlsEnabled, _ := strconv.ParseBool(getEnvWithDefault("SMTP_TLS", "true"))

	return EmailConfig{
		SMTPHost:     getEnvWithDefault("SMTP_HOST", "localhost"),
		SMTPPort:     port,
		SMTPUsername: getEnvWithDefault("SMTP_USERNAME", ""),
		SMTPPassword: getEnvWithDefault("SMTP_PASSWORD", ""),
		FromEmail:    getEnvWithDefault("FROM_EMAIL", "noreply@securestor.local"),
		FromName:     getEnvWithDefault("FROM_NAME", "SecurStor"),
		BaseURL:      getEnvWithDefault("BASE_URL", "http://localhost:3000"),
		TLSEnabled:   tlsEnabled,
	}
}

// SendUserInvitation sends an invitation email to a new user
func (s *EmailService) SendUserInvitation(email, firstName, lastName, inviteToken string) error {
	subject := "You're invited to SecurStor"

	// Create invitation data
	data := struct {
		FirstName   string
		LastName    string
		Email       string
		InviteURL   string
		CompanyName string
		BaseURL     string
	}{
		FirstName:   firstName,
		LastName:    lastName,
		Email:       email,
		InviteURL:   fmt.Sprintf("%s/invite/accept?token=%s", s.baseURL, inviteToken),
		CompanyName: "SecurStor",
		BaseURL:     s.baseURL,
	}

	// Generate email content
	htmlContent, err := s.generateInvitationHTML(data)
	if err != nil {
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	textContent := s.generateInvitationText(data)

	return s.sendEmail(email, subject, htmlContent, textContent)
}

// SendPasswordReset sends a password reset email
func (s *EmailService) SendPasswordReset(email, firstName, resetToken string) error {
	subject := "Reset your SecurStor password"

	data := struct {
		FirstName   string
		Email       string
		ResetURL    string
		CompanyName string
		BaseURL     string
	}{
		FirstName:   firstName,
		Email:       email,
		ResetURL:    fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, resetToken),
		CompanyName: "SecurStor",
		BaseURL:     s.baseURL,
	}

	htmlContent, err := s.generatePasswordResetHTML(data)
	if err != nil {
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	textContent := s.generatePasswordResetText(data)

	return s.sendEmail(email, subject, htmlContent, textContent)
}

// sendEmail sends an email using the configured SMTP settings
func (s *EmailService) sendEmail(to, subject, htmlContent, textContent string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(s.from, s.fromName))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", textContent)
	m.AddAlternative("text/html", htmlContent)

	// Check if SMTP is configured
	if s.dialer.Host == "localhost" && s.dialer.Username == "" {
		// In development mode without SMTP, just log the email
		fmt.Printf("\n=== EMAIL WOULD BE SENT ===\n")
		fmt.Printf("To: %s\n", to)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Content:\n%s\n", textContent)
		fmt.Printf("========================\n\n")
		return nil
	}

	return s.dialer.DialAndSend(m)
}

// generateInvitationHTML generates HTML content for invitation emails
func (s *EmailService) generateInvitationHTML(data interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>You're invited to {{.CompanyName}}</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2563eb; color: white; padding: 20px; text-align: center; }
        .content { padding: 30px 20px; background: #f9fafb; }
        .button { display: inline-block; padding: 12px 24px; background: #2563eb; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { padding: 20px; text-align: center; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to {{.CompanyName}}</h1>
        </div>
        <div class="content">
            <h2>Hello {{.FirstName}}!</h2>
            <p>You've been invited to join <strong>{{.CompanyName}}</strong>, a secure artifact storage and management platform.</p>
            <p>Click the button below to accept your invitation and set up your account:</p>
            <p style="text-align: center;">
                <a href="{{.InviteURL}}" class="button">Accept Invitation</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #2563eb;">{{.InviteURL}}</p>
            <p><strong>Note:</strong> This invitation will expire in 7 days for security reasons.</p>
        </div>
        <div class="footer">
            <p>If you didn't expect this invitation, you can safely ignore this email.</p>
            <p>© {{.CompanyName}} - Secure Artifact Storage</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("invitation").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateInvitationText generates plain text content for invitation emails
func (s *EmailService) generateInvitationText(data interface{}) string {
	d := data.(struct {
		FirstName   string
		LastName    string
		Email       string
		InviteURL   string
		CompanyName string
		BaseURL     string
	})

	return fmt.Sprintf(`Welcome to %s!

Hello %s,

You've been invited to join %s, a secure artifact storage and management platform.

To accept your invitation and set up your account, please visit:
%s

This invitation will expire in 7 days for security reasons.

If you didn't expect this invitation, you can safely ignore this email.

Best regards,
The %s Team`, d.CompanyName, d.FirstName, d.CompanyName, d.InviteURL, d.CompanyName)
}

// generatePasswordResetHTML generates HTML content for password reset emails
func (s *EmailService) generatePasswordResetHTML(data interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset your password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #dc2626; color: white; padding: 20px; text-align: center; }
        .content { padding: 30px 20px; background: #f9fafb; }
        .button { display: inline-block; padding: 12px 24px; background: #dc2626; color: white; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { padding: 20px; text-align: center; color: #6b7280; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset</h1>
        </div>
        <div class="content">
            <h2>Hello {{.FirstName}}!</h2>
            <p>We received a request to reset your password for your {{.CompanyName}} account.</p>
            <p>Click the button below to reset your password:</p>
            <p style="text-align: center;">
                <a href="{{.ResetURL}}" class="button">Reset Password</a>
            </p>
            <p>Or copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #dc2626;">{{.ResetURL}}</p>
            <p><strong>Note:</strong> This link will expire in 1 hour for security reasons.</p>
        </div>
        <div class="footer">
            <p>If you didn't request this password reset, you can safely ignore this email.</p>
            <p>© {{.CompanyName}} - Secure Artifact Storage</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("reset").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generatePasswordResetText generates plain text content for password reset emails
func (s *EmailService) generatePasswordResetText(data interface{}) string {
	d := data.(struct {
		FirstName   string
		Email       string
		ResetURL    string
		CompanyName string
		BaseURL     string
	})

	return fmt.Sprintf(`Password Reset Request

Hello %s,

We received a request to reset your password for your %s account.

To reset your password, please visit:
%s

This link will expire in 1 hour for security reasons.

If you didn't request this password reset, you can safely ignore this email.

Best regards,
The %s Team`, d.FirstName, d.CompanyName, d.ResetURL, d.CompanyName)
}

// getEnvWithDefault gets environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
