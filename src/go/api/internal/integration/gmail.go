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

const (
	gmailSMTPHost = "smtp.gmail.com"
	gmailSMTPAddr = "smtp.gmail.com:587"
)

// GmailCredentials holds the email and app password for Gmail SMTP.
type GmailCredentials struct {
	Email            string `json:"email"`
	AppPassword      string `json:"app_password"`
	DefaultRecipient string `json:"default_recipient"`
}

// Gmail implements PlatformAdapter for sending email via Gmail SMTP.
type Gmail struct{}

func NewGmail() *Gmail {
	return &Gmail{}
}

func (g *Gmail) ValidateCredentials(ctx context.Context, credentials json.RawMessage) error {
	creds, err := g.parseCreds(credentials)
	if err != nil {
		return err
	}

	// Verify SMTP auth by connecting, doing STARTTLS, and authenticating.
	conn, err := net.DialTimeout("tcp", gmailSMTPAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("gmail: connecting to SMTP: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, gmailSMTPHost)
	if err != nil {
		return fmt.Errorf("gmail: SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: gmailSMTPHost}); err != nil {
		return fmt.Errorf("gmail: STARTTLS: %w", err)
	}

	auth := smtp.PlainAuth("", creds.Email, creds.AppPassword, gmailSMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("gmail: authentication failed — check email and app password: %w", err)
	}

	return nil
}

func (g *Gmail) Publish(ctx context.Context, content PublishContent, credentials json.RawMessage) (*PublishResult, error) {
	creds, err := g.parseCreds(credentials)
	if err != nil {
		return nil, err
	}

	recipient := creds.DefaultRecipient
	if recipient == "" {
		return nil, fmt.Errorf("gmail: no recipient — set default_recipient in credentials")
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
	messageID, err := sendEmail(gmailSMTPAddr, auth, creds.Email, recipient, subject, body)
	if err != nil {
		return nil, fmt.Errorf("gmail: %w", err)
	}

	return &PublishResult{
		PlatformPostID: messageID,
		URL:            "",
	}, nil
}

func (g *Gmail) parseCreds(raw json.RawMessage) (*GmailCredentials, error) {
	var creds GmailCredentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("gmail: invalid credentials JSON: %w", err)
	}
	if creds.Email == "" {
		return nil, fmt.Errorf("gmail: email is required")
	}
	if creds.AppPassword == "" {
		return nil, fmt.Errorf("gmail: app_password is required")
	}
	return &creds, nil
}

// sendEmail constructs an RFC 2822 message and sends it via SMTP with STARTTLS.
func sendEmail(addr string, auth smtp.Auth, from, to, subject, body string) (string, error) {
	messageID := fmt.Sprintf("<%d.werd@%s>", time.Now().UnixNano(), strings.SplitN(from, "@", 2)[1])

	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"Message-ID: " + messageID,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return "", fmt.Errorf("sending email: %w", err)
	}

	return messageID, nil
}
