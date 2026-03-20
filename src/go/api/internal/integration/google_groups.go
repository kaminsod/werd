package integration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// GoogleGroupsCredentials holds the sender email, app password, and target group.
type GoogleGroupsCredentials struct {
	Email      string `json:"email"`
	AppPassword string `json:"app_password"`
	GroupEmail  string `json:"group_email"`
}

// GoogleGroups implements PlatformAdapter for posting to Google Groups via SMTP.
type GoogleGroups struct{}

func NewGoogleGroups() *GoogleGroups {
	return &GoogleGroups{}
}

func (g *GoogleGroups) ValidateCredentials(ctx context.Context, credentials json.RawMessage) error {
	creds, err := g.parseCreds(credentials)
	if err != nil {
		return err
	}

	// Verify SMTP auth (same as Gmail — uses same Gmail SMTP).
	conn, err := net.DialTimeout("tcp", gmailSMTPAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("google_groups: connecting to SMTP: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, gmailSMTPHost)
	if err != nil {
		return fmt.Errorf("google_groups: SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: gmailSMTPHost}); err != nil {
		return fmt.Errorf("google_groups: STARTTLS: %w", err)
	}

	auth := smtp.PlainAuth("", creds.Email, creds.AppPassword, gmailSMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("google_groups: authentication failed — check email and app password: %w", err)
	}

	return nil
}

func (g *GoogleGroups) Publish(ctx context.Context, content PublishContent, credentials json.RawMessage) (*PublishResult, error) {
	creds, err := g.parseCreds(credentials)
	if err != nil {
		return nil, err
	}

	subject := content.Title
	if subject == "" {
		subject = "Post from Werd"
	}

	body := content.Body
	if content.URL != "" {
		if body != "" {
			body += "\n\n"
		}
		body += content.URL
	}

	auth := smtp.PlainAuth("", creds.Email, creds.AppPassword, gmailSMTPHost)
	messageID, err := sendEmail(gmailSMTPAddr, auth, creds.Email, creds.GroupEmail, subject, body)
	if err != nil {
		return nil, fmt.Errorf("google_groups: %w", err)
	}

	return &PublishResult{
		PlatformPostID: messageID,
		URL:            "",
	}, nil
}

func (g *GoogleGroups) parseCreds(raw json.RawMessage) (*GoogleGroupsCredentials, error) {
	var creds GoogleGroupsCredentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("google_groups: invalid credentials JSON: %w", err)
	}
	if creds.Email == "" {
		return nil, fmt.Errorf("google_groups: email is required")
	}
	if creds.AppPassword == "" {
		return nil, fmt.Errorf("google_groups: app_password is required")
	}
	if creds.GroupEmail == "" {
		return nil, fmt.Errorf("google_groups: group_email is required")
	}
	if !strings.HasSuffix(creds.GroupEmail, "@googlegroups.com") {
		return nil, fmt.Errorf("google_groups: group_email must end with @googlegroups.com")
	}
	return &creds, nil
}
