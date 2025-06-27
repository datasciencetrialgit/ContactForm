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

// Add a config for the redirect URL
var redirectURLConfig = os.Getenv("REDIRECT_URL")

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
			// Redirect with success message as URL parameter
			redirectURL := redirectURLConfig
			if redirectURL == "" {
				redirectURL = r.Referer()
			}
			if redirectURL == "" {
				redirectURL = "/Contact"
			}
			sep := "?"
			if strings.Contains(redirectURL, "?") {
				sep = "&"
			}
			redirectURL = redirectURL + sep + "status=success"
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
			return
		}
		lastErr = err
		log.Printf("SendMail error with %s: %v", provider, err)
	}
	if lastErr != nil {
		// Redirect with failure message as URL parameter
		redirectURL := redirectURLConfig
		if redirectURL == "" {
			redirectURL = r.Referer()
		}
		if redirectURL == "" {
			redirectURL = "/"
		}
		sep := "?"
		if strings.Contains(redirectURL, "?") {
			sep = "&"
		}
		redirectURL = redirectURL + sep + "status=error"
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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
