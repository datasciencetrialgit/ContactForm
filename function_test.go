package function

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestContactForm_Success(t *testing.T) {
	os.Setenv("SMTP_USER", "craftedlivefoundation@gmail.com")
	os.Setenv("SMTP_PASS", "Crafted001!")
	os.Setenv("SMTP_TO", "craftedlivefoundation@gmail.com")
	os.Setenv("SMTP_PROVIDERS", "gmail") // updated to match config-driven logic

	// Create a temporary smtp_providers.yaml file for the test
	tmpYaml := `smtp_providers:
  gmail:
    host: smtp.gmail.com
    port: "587"
  microsoft:
    host: smtp.office365.com
    port: "587"
`
	yamlPath := "smtp_providers.yaml"
	os.WriteFile(yamlPath, []byte(tmpYaml), 0644)
	defer os.Remove(yamlPath)

	form := url.Values{}
	form.Set("name", "Test User")
	form.Set("email", "test@user.com")
	form.Set("message", "Hello!")
	form.Set("website", "") // honeypot empty

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rw := httptest.NewRecorder()

	// Patch smtp.SendMail to avoid real email sending
	sendMailOrig := sendMail
	sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil // simulate success
	}
	defer func() { sendMail = sendMailOrig }()

	ContactForm(rw, req)

	resp := rw.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "Message sent successfully") {
		t.Errorf("unexpected response: %s", string(body))
	}
}

func TestContactForm_Honeypot(t *testing.T) {
	form := url.Values{}
	form.Set("name", "Test User")
	form.Set("email", "test@user.com")
	form.Set("message", "Hello!")
	form.Set("website", "spammy") // honeypot filled

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rw := httptest.NewRecorder()
	ContactForm(rw, req)

	resp := rw.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", resp.StatusCode)
	}
}

func TestContactForm_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()
	ContactForm(rw, req)
	resp := rw.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", resp.StatusCode)
	}
}
