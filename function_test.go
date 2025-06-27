package function

import (
	"context"
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
	os.Setenv("SMTP_USER_SECRET", "projects/test/secrets/SMTP_USER")
	os.Setenv("SMTP_PASS_SECRET", "projects/test/secrets/SMTP_PASS")
	os.Setenv("SMTP_TO", "craftedlivefoundation@gmail.com")
	os.Setenv("SMTP_PROVIDERS", "gmail")

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

	// Patch getSecret to return dummy values for secrets
	getSecretOrig := getSecret
	getSecret = func(ctx context.Context, secretName string) (string, error) {
		if strings.Contains(secretName, "SMTP_USER") {
			return "craftedlivefoundation@gmail.com", nil
		}
		if strings.Contains(secretName, "SMTP_PASS") {
			return "Crafted001!", nil
		}
		return "", nil
	}
	defer func() { getSecret = getSecretOrig }()

	ContactForm(rw, req)

	resp := rw.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(body))
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
