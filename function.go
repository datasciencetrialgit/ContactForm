package function

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var sendMail = smtp.SendMail

type SMTPProvider struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type SMTPConfig struct {
	Providers map[string]SMTPProvider `yaml:"smtp_providers"`
}

// SMTP provider config is now hardcoded in the code for deployment simplicity
var smtpConfig = SMTPConfig{
	Providers: map[string]SMTPProvider{
		"gmail": {
			Host: "smtp.gmail.com",
			Port: "587",
		},
		"microsoft": {
			Host: "smtp.office365.com",
			Port: "587",
		},
		// Add more providers as needed
	},
}

// loadSMTPConfig returns the hardcoded SMTPConfig
func loadSMTPConfig(_ string) (*SMTPConfig, error) {
	return &smtpConfig, nil
}

// GetSecretValue fetches the secret value from Google Secret Manager.
func GetSecretValue(ctx context.Context, client *secretmanager.Client, secretName string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %v", err)
	}

	return string(result.Payload.Data), nil
}

// ContactForm handles POST requests from a contact form and sends email via configured SMTP providers.
const serverConfigErrorMsg = "Server config error"

func ContactForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple honeypot anti-bot check
	if r.FormValue("website") != "" {
		http.Error(w, "Spam detected", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	message := strings.TrimSpace(r.FormValue("message"))

	// Basic input validation
	if name == "" || email == "" || message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}
	if len(message) > 5000 {
		http.Error(w, "Message too long", http.StatusBadRequest)
		return
	}
	if !strings.Contains(email, "@") {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	subject := "Contact Form Submission"
	body := fmt.Sprintf("Name: %s\nEmail: %s\nMessage:\n%s", name, email, message)

	// Fetch SMTP credentials from Secret Manager
	ctx := r.Context()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create new client: %v", err)
		http.Error(w, serverConfigErrorMsg, http.StatusInternalServerError)
		return
	}
	defer client.Close()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Printf("GOOGLE_CLOUD_PROJECT not set")
		http.Error(w, "Server config error", http.StatusInternalServerError)
		return
	}

	smtpUserSecret := fmt.Sprintf("projects/%s/secrets/SMTP_USER_SECRET/versions/1", projectID)
    smtpPassSecret := fmt.Sprintf("projects/%s/secrets/SMTP_PASS_SECRET/versions/1", projectID)
	if smtpUserSecret == "" || smtpPassSecret == "" {
		log.Printf("SMTP secret names not set")
		http.Error(w, serverConfigErrorMsg, http.StatusInternalServerError)
		return
	}
	smtpUser, err := GetSecretValue(ctx, client, smtpUserSecret)
	if err != nil {
		log.Printf("Failed to get SMTP_USER secret: %v", err)
		http.Error(w, serverConfigErrorMsg, http.StatusInternalServerError)
		return
	}
	smtpPass, err := GetSecretValue(ctx, client, smtpPassSecret)
	if err != nil {
		log.Printf("Failed to get SMTP_PASS secret: %v", err)
		http.Error(w, serverConfigErrorMsg, http.StatusInternalServerError)
		return
	}

	msg := "From: " + smtpUser + "\n" +
		"To: " + os.Getenv("SMTP_TO") + "\n" +
		"Subject: " + subject + "\n\n" + body
	cfg, err := loadSMTPConfig("")
	if err != nil {
		log.Printf("Failed to load SMTP config: %v", err)
		http.Error(w, serverConfigErrorMsg, http.StatusInternalServerError)
		return
	}

	providers := os.Getenv("SMTP_PROVIDERS")
	if providers == "" {
		providers = "gmail,microsoft"
	}

	// TODO: Add rate limiting and/or Origin/Referer checks here if needed

	if err := trySendMailWithProviders(cfg, providers, msg, smtpUser, smtpPass); err != nil {
		log.Printf("Failed to send mail: %v", err)
		http.Error(w, "Failed to send message. Please try again later.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Message sent successfully!")
}

func splitAndTrim(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// trySendMailWithProviders attempts to send the email using the configured SMTP providers.
func trySendMailWithProviders(cfg *SMTPConfig, providers string, msg string, smtpUser string, smtpPass string) error {
	var lastErr error
	providerList := splitAndTrim(providers)
	smtpTo := os.Getenv("SMTP_TO")

	if smtpUser == "" || smtpPass == "" {
		log.Printf("SMTP_USER or SMTP_PASS not set")
		return fmt.Errorf("SMTP credentials not set")
	}
	if smtpTo == "" {
		log.Printf("SMTP_TO not set")
		return fmt.Errorf("SMTP_TO not set")
	}

	for _, provider := range providerList {
		p, ok := cfg.Providers[provider]
		if !ok {
			continue
		}
		auth := smtp.PlainAuth("", smtpUser, smtpPass, p.Host)
		err := sendMail(p.Host+":"+p.Port, auth, smtpUser, []string{smtpTo}, []byte(msg))
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("SendMail error with %s: %v", provider, err)
	}
	if lastErr != nil {
		return fmt.Errorf("failed to send email with all providers")
	}
	return fmt.Errorf("no valid SMTP provider configured")
}
