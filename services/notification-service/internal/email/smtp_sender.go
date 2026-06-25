package email

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// sendMailFunc matches net/smtp.SendMail; it is a field so tests can substitute
// a fake transport instead of dialing a real server.
type sendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// SMTPSender delivers email over SMTP using the standard library. It negotiates
// STARTTLS automatically when the server advertises it (net/smtp.SendMail
// behavior). Implicit TLS (port 465) is not supported in v1; use a STARTTLS
// port such as 587. SMTP_PASSWORD is never logged and the message body is never
// logged at info level.
type SMTPSender struct {
	cfg  SMTPConfig
	log  *zap.Logger
	send sendMailFunc
}

// NewSMTPSender constructs the SMTP sender, requiring at least a host and a
// from-address (validated again here as defence in depth even though config
// load already enforces it).
func NewSMTPSender(cfg SMTPConfig, log *zap.Logger) (*SMTPSender, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("smtp sender: SMTP_HOST is required")
	}
	if strings.TrimSpace(cfg.FromEmail) == "" {
		return nil, fmt.Errorf("smtp sender: SMTP_FROM_EMAIL is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return &SMTPSender{cfg: cfg, log: log, send: smtp.SendMail}, nil
}

// Send builds the MIME message and hands it to the SMTP transport. Auth is used
// only when a username is configured.
func (s *SMTPSender) Send(_ context.Context, msg EmailMessage) error {
	if err := msg.Validate(); err != nil {
		return err
	}

	raw, err := buildMIMEMessage(s.cfg, msg)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	var auth smtp.Auth
	if strings.TrimSpace(s.cfg.Username) != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	if err := s.send(addr, auth, s.cfg.FromEmail, []string{msg.ToEmail}, raw); err != nil {
		// Error text may include the address but never the password or body.
		return fmt.Errorf("smtp send to %s: %w", MaskEmail(msg.ToEmail), err)
	}

	s.log.Info("email send (smtp)",
		zap.String("provider", ProviderSMTP),
		zap.String("to", MaskEmail(msg.ToEmail)),
		zap.String("subject", msg.Subject),
	)
	return nil
}

// buildMIMEMessage renders an RFC 5322 message with CRLF line endings. When an
// HTML body is present it is sent as a multipart/alternative with a text/plain
// part first (clients pick the richest part they support); otherwise a single
// text/plain part is used.
func buildMIMEMessage(cfg SMTPConfig, msg EmailMessage) ([]byte, error) {
	from := (&mail.Address{Name: cfg.FromName, Address: cfg.FromEmail}).String()
	to := (&mail.Address{Name: msg.ToName, Address: msg.ToEmail}).String()

	var b strings.Builder
	writeHeader(&b, "From", from)
	writeHeader(&b, "To", to)
	writeHeader(&b, "Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	writeHeader(&b, "MIME-Version", "1.0")

	html := strings.TrimSpace(msg.HTMLBody)
	if html == "" {
		writeHeader(&b, "Content-Type", "text/plain; charset=\"utf-8\"")
		b.WriteString("\r\n")
		b.WriteString(normalizeCRLF(msg.TextBody))
		return []byte(b.String()), nil
	}

	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}
	writeHeader(&b, "Content-Type", "multipart/alternative; boundary=\""+boundary+"\"")
	b.WriteString("\r\n")

	writePart(&b, boundary, "text/plain; charset=\"utf-8\"", msg.TextBody)
	writePart(&b, boundary, "text/html; charset=\"utf-8\"", msg.HTMLBody)
	b.WriteString("--" + boundary + "--\r\n")

	return []byte(b.String()), nil
}

func writeHeader(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\r\n")
}

func writePart(b *strings.Builder, boundary, contentType, body string) {
	b.WriteString("--" + boundary + "\r\n")
	writeHeader(b, "Content-Type", contentType)
	b.WriteString("\r\n")
	b.WriteString(normalizeCRLF(body))
	b.WriteString("\r\n")
}

// normalizeCRLF converts any line endings to CRLF as required by SMTP.
func normalizeCRLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func randomBoundary() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate mime boundary: %w", err)
	}
	return "boundary_" + hex.EncodeToString(buf), nil
}
