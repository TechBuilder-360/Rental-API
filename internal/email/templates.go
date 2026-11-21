package email

import (
	"bytes"
	"fmt"
	"html/template"
	"time"
)

var otpHTMLTemplate = template.Must(template.New("otp").Parse(`<!doctype html>
<html>
  <body style="font-family: -apple-system, Segoe UI, Roboto, sans-serif; color: #111;">
    <p>Your sign-in code is:</p>
    <p style="font-size: 28px; font-weight: bold; letter-spacing: 6px;">{{.Code}}</p>
    <p>This code expires in {{.Minutes}} minutes. If you didn't request it, you can ignore this email.</p>
  </body>
</html>`))

// OTPMessage builds the sign-in code email.
func OTPMessage(to, code string, ttl time.Duration) (Message, error) {
	minutes := int(ttl.Round(time.Minute) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}

	var html bytes.Buffer
	if err := otpHTMLTemplate.Execute(&html, struct {
		Code    string
		Minutes int
	}{code, minutes}); err != nil {
		return Message{}, fmt.Errorf("render otp email: %w", err)
	}

	return Message{
		To:      to,
		Subject: "Your sign-in code",
		TextBody: fmt.Sprintf(
			"Your sign-in code is %s.\n\nThis code expires in %d minutes. If you didn't request it, you can ignore this email.\n",
			code, minutes),
		HTMLBody: html.String(),
		Tag:      "otp",
	}, nil
}
