# ContactForm Google Cloud Function

This project provides a secure Google Cloud Function written in Go for handling contact form submissions from static sites. It sends emails via one or more configurable SMTP providers (Gmail, Microsoft, etc.) and includes anti-spam protection.

## Features
- Written in Go (Golang)
- Accepts POST requests from your contact form
- Honeypot field for spam protection
- Config-driven: supports multiple SMTP providers (failover if one fails)
- Uses Google Cloud Secret Manager for SMTP credentials
- Ready for deployment to Google Cloud Functions
- Example GitHub Actions workflow for CI/CD
- Includes local/unit test scripts

## Setup

### 1. Configure Environment Variables
Set the following environment variables when deploying or testing:
- `SMTP_USER_SECRET`: The resource name of your SMTP user secret in Secret Manager (e.g., `projects/PROJECT_ID/secrets/SMTP_USER`)
- `SMTP_PASS_SECRET`: The resource name of your SMTP password secret in Secret Manager (e.g., `projects/PROJECT_ID/secrets/SMTP_PASS`)
- `SMTP_TO`: The email address to receive contact form messages
- `SMTP_PROVIDERS`: Comma-separated list of provider keys to try in order (e.g., `gmail,microsoft`). Defaults to `gmail,microsoft`.

### 2. Deploy to Google Cloud Functions

```
gcloud functions deploy ContactForm \
  --runtime go121 \
  --trigger-http \
  --allow-unauthenticated \
  --set-env-vars "SMTP_USER_SECRET=projects/PROJECT_ID/secrets/SMTP_USER,SMTP_PASS_SECRET=projects/PROJECT_ID/secrets/SMTP_PASS,SMTP_TO=you@gmail.com,SMTP_PROVIDERS=gmail,microsoft"
```

### 3. Update Your Contact Form
- Set the form `action` to your deployed function's URL (e.g., `https://REGION-PROJECT_ID.cloudfunctions.net/ContactForm`)
- Add a hidden honeypot field:
  ```html
  <input type="text" name="website" style="display:none">
  ```

### 4. Local Testing
- Run `go mod init contactform` if you haven't already.
- Run tests with `go test` (no real emails sent; SMTP is mocked).
- For real local testing, create a `main.go` to run an HTTP server and POST to it.

### 5. (Optional) GitHub Actions CI/CD
See `.github/workflows/deploy.yml` for an example workflow to deploy on push.

## Security
- Uses a honeypot field to block most bots.
- SMTP credentials are stored securely in Google Cloud Secret Manager, not in environment variables.
- For more advanced security, consider adding a math challenge or reCAPTCHA on your frontend and validating it before submitting to this function.

## Google Cloud Setup Summary

### APIs/Services to Enable
- **Cloud Functions API** (`cloudfunctions.googleapis.com`)
- **Cloud Run API** (`run.googleapis.com`)
- **Cloud Build API** (`cloudbuild.googleapis.com`)
- **IAM Service Account Credentials API** (`iamcredentials.googleapis.com`)
- **Secret Manager API** (`secretmanager.googleapis.com`)

Enable these in the [Google Cloud Console API Library](https://console.cloud.google.com/apis/library).

### Required IAM Roles for Service Account
- `roles/cloudfunctions.developer` (Cloud Functions Developer)
- `roles/cloudfunctions.viewer` (Cloud Functions Viewer)
- `roles/iam.serviceAccountUser` (Service Account User, if deploying as another service account)
- `roles/run.invoker` (Cloud Run Invoker, for HTTP triggers)
- `roles/secretmanager.secretAccessor` (Secret Manager Secret Accessor)

Assign these roles to the service account used for deployment in the [IAM page](https://console.cloud.google.com/iam-admin/iam).

## License
MIT
