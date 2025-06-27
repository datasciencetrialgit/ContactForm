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

// getSecret fetches the latest version of a secret from Google Cloud Secret Manager.
func getSecret(ctx context.Context, secretName string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("%s/versions/latest", secretName),
	}
	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}
	return string(result.Payload.Data), nil
}

// ContactForm handles POST requests from a contact form and sends email via configured SMTP providers.
func ContactForm(w http.ResponseWriter, r *http.Request) {
	wd, _ := os.Getwd()
	log.Printf("Current working directory: %s", wd)
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.FormValue("website") != "" {
		http.Error(w, "Spam detected", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	message := r.FormValue("message")

	if name == "" || email == "" || message == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	subject := "Contact Form Submission"
	body := fmt.Sprintf("Name: %s\nEmail: %s\nMessage:\n%s", name, email, message)

	// Fetch SMTP credentials from Secret Manager
	ctx := r.Context()
	smtpUserSecret := os.Getenv("SMTP_USER_SECRET")
	smtpPassSecret := os.Getenv("SMTP_PASS_SECRET")
	if smtpUserSecret == "" || smtpPassSecret == "" {
		http.Error(w, "SMTP secret names not set", http.StatusInternalServerError)
		return
	}
	smtpUser, err := getSecret(ctx, smtpUserSecret)
	if err != nil {
		log.Printf("Failed to get SMTP_USER secret: %v", err)
		http.Error(w, "Server config error", http.StatusInternalServerError)
		return
	}
	smtpPass, err := getSecret(ctx, smtpPassSecret)
	if err != nil {
		log.Printf("Failed to get SMTP_PASS secret: %v", err)
		http.Error(w, "Server config error", http.StatusInternalServerError)
		return
	}

	msg := "From: " + smtpUser + "\n" +
		"To: " + os.Getenv("SMTP_TO") + "\n" +
		"Subject: " + subject + "\n\n" + body

	// Load SMTP providers config
	cfg, err := loadSMTPConfig("")
	if err != nil {
		log.Printf("Failed to load SMTP config: %v", err)
		http.Error(w, "Server config error", http.StatusInternalServerError)
		return
	}

	providers := os.Getenv("SMTP_PROVIDERS") // comma-separated, e.g. "gmail,microsoft"
	if providers == "" {
		providers = "gmail,microsoft" // default order
	}

	if err := trySendMailWithProviders(cfg, providers, msg, smtpUser, smtpPass); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
