// Package mailer sends email via SMTP, with a log-only implementation for
// local development.
package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"go.uber.org/zap"
)

// Message is an email to send.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer sends email.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// SMTPMailer delivers mail through an SMTP server (Mailhog, SendGrid, etc.).
type SMTPMailer struct {
	addr string // host:port
	auth smtp.Auth
	from string
}

// NewSMTPMailer builds an SMTPMailer. If username is empty, no auth is used
// (fine for local relays like Mailhog).
func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	return &SMTPMailer{
		addr: fmt.Sprintf("%s:%d", host, port),
		auth: auth,
		from: from,
	}
}

// Send delivers a plain-text message.
func (m *SMTPMailer) Send(_ context.Context, msg Message) error {
	if err := smtp.SendMail(m.addr, m.auth, m.from, []string{msg.To}, buildMIME(m.from, msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func buildMIME(from string, msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	b.WriteString("\r\n")
	return []byte(b.String())
}

// LogMailer logs messages instead of sending them — used when SMTP is not
// configured, so local runs don't require a mail server.
type LogMailer struct{ log *zap.Logger }

// NewLogMailer builds a LogMailer.
func NewLogMailer(log *zap.Logger) *LogMailer { return &LogMailer{log: log} }

// Send logs the message.
func (m *LogMailer) Send(_ context.Context, msg Message) error {
	m.log.Info("email (dev mode, not sent)",
		zap.String("to", msg.To),
		zap.String("subject", msg.Subject),
	)
	return nil
}
