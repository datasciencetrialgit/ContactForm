package function

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
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
	msg := "From: " + os.Getenv("SMTP_USER") + "\n" +
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
	var lastErr error
	for _, provider := range splitAndTrim(providers) {
		p, ok := cfg.Providers[provider]
		if !ok {
			continue
		}
		smtpUser := os.Getenv("SMTP_USER")
		smtpPass := os.Getenv("SMTP_PASS")
		if smtpUser == "" || smtpPass == "" {
			log.Printf("SMTP_USER or SMTP_PASS not set")
			continue
		}
		auth := smtp.PlainAuth("", smtpUser, smtpPass, p.Host)
		err := sendMail(p.Host+":"+p.Port, auth, smtpUser, []string{os.Getenv("SMTP_TO")}, []byte(msg))
		if err == nil {
			// Show message and redirect after 1 second using HTML/JS
			redirectURL := r.Referer()
			if redirectURL == "" {
				redirectURL = "/" // fallback if no referer
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<html><body><p>Message sent successfully!</p><script>setTimeout(function(){window.location.href='%s';}, 1000);</script></body></html>`, redirectURL)
			return
		}
		lastErr = err
		log.Printf("SendMail error with %s: %v", provider, err)
	}
	if lastErr != nil {
		http.Error(w, "Failed to send email with all providers", http.StatusInternalServerError)
	} else {
		http.Error(w, "No valid SMTP provider configured", http.StatusInternalServerError)
	}
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
