package function

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var sendMail = smtp.SendMail

type SMTPProvider struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type SMTPConfig struct {
	Providers map[string]SMTPProvider `yaml:"smtp_providers"`
}

// loadSMTPConfig loads SMTP providers from a YAML config file
func loadSMTPConfig(path string) (*SMTPConfig, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(exePath)
	fullPath := filepath.Join(dir, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var cfg SMTPConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
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
	cfg, err := loadSMTPConfig("smtp_providers.yaml")
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
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Message sent successfully!")
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
