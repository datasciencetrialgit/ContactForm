# ContactForm Google Cloud Function

This project provides a secure Google Cloud Function written in Go for handling contact form submissions from static sites. It sends emails via one or more configurable SMTP providers (Gmail, Microsoft, etc.) and includes anti-spam protection.

## Features
- Written in Go (Golang)
- Accepts POST requests from your contact form
- Honeypot field for spam protection
- Config-driven: supports multiple SMTP providers (failover if one fails)
- Reads SMTP provider config from `smtp_providers.yaml`
- Ready for deployment to Google Cloud Functions
- Example GitHub Actions workflow for CI/CD
- Includes local/unit test scripts

## Setup

### 1. Configure Environment Variables
Set the following environment variables when deploying or testing:
- `SMTP_USER`: Your email address (Gmail, Microsoft, etc.)
- `SMTP_PASS`: Your app password (not your main password)
- `SMTP_TO`: The email address to receive contact form messages
- `SMTP_PROVIDERS`: Comma-separated list of provider keys to try in order (e.g., `gmail,microsoft`). Defaults to `gmail,microsoft`.

### 2. Configure SMTP Providers
Edit `smtp_providers.yaml` to add or modify SMTP providers. Example:
```yaml
smtp_providers:
  gmail:
    host: smtp.gmail.com
    port: "587"
  microsoft:
    host: smtp.office365.com
    port: "587"
  # Add more providers as needed
```

### 3. Deploy to Google Cloud Functions

```
gcloud functions deploy ContactForm \
  --runtime go121 \
  --trigger-http \
  --allow-unauthenticated \
  --set-env-vars "SMTP_USER=you@gmail.com,SMTP_PASS=your-app-password,SMTP_TO=you@gmail.com,SMTP_PROVIDERS=gmail,microsoft"
```

> Note: For Google Cloud Functions, you must include the `smtp_providers.yaml` file in your deployment package.

### 4. Update Your Contact Form
- Set the form `action` to your deployed function's URL (e.g., `https://REGION-PROJECT_ID.cloudfunctions.net/ContactForm`)
- Add a hidden honeypot field:
  ```html
  <input type="text" name="website" style="display:none">
  ```

### 5. Local Testing
- Run `go mod init contactform` if you haven't already.
- Run `go get gopkg.in/yaml.v3` to install the YAML package.
- Run tests with `go test` (no real emails sent; SMTP is mocked).
- For real local testing, create a `main.go` to run an HTTP server and POST to it.

### 6. (Optional) GitHub Actions CI/CD
See `.github/workflows/deploy.yml` for an example workflow to deploy on push.

## Security
- Uses a honeypot field to block most bots.
- For more security, add reCAPTCHA and validate it in the function.

## Google Cloud IAM Permissions

Your service account must have the following roles to deploy and manage Cloud Functions:
- `roles/cloudfunctions.developer`
- `roles/cloudfunctions.viewer`
- `roles/iam.serviceAccountUser` (if deploying as another service account)

You can grant these roles in the [Google Cloud Console IAM page](https://console.cloud.google.com/iam-admin/iam).

## License
MIT
