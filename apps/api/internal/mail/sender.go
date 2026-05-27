package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Config holds SMTP connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
}

// IsConfigured returns true when enough SMTP settings are present to send mail.
func (c Config) IsConfigured() bool {
	return c.Host != "" && c.Port > 0 && c.From != ""
}

// Message is a simple outgoing email.
// Set HTML to send an HTML body; Body is used as plain-text fallback.
type Message struct {
	To      []string
	Subject string
	Body    string // plain text (always set; used as fallback when HTML is set)
	HTML    string // optional HTML body; if set, sends multipart/alternative
}

// Sender sends transactional emails via SMTP.
// All sends are blocking — callers should invoke from a goroutine when needed.
type Sender struct {
	cfg Config
}

func New(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

// Send delivers a message via SMTP. Returns nil when SMTP is not configured
// (graceful no-op for deployments without email setup).
func (s *Sender) Send(msg Message) error {
	if !s.cfg.IsConfigured() {
		return nil
	}
	if len(msg.To) == 0 {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	body := buildBody(s.cfg.From, msg)

	var auth smtp.Auth
	if s.cfg.User != "" {
		auth = smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	}

	if s.cfg.TLS {
		return s.sendTLS(addr, auth, body, msg.To)
	}
	return smtp.SendMail(addr, auth, s.cfg.From, msg.To, body)
}

func (s *Sender) sendTLS(addr string, auth smtp.Auth, body []byte, to []string) error {
	host, _, _ := net.SplitHostPort(addr)
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close() //nolint:errcheck

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Quit() //nolint:errcheck

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp MAIL: %w", err)
	}
	for _, r := range to {
		if err := client.Rcpt(r); err != nil {
			return fmt.Errorf("smtp RCPT %s: %w", r, err)
		}
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	defer wc.Close() //nolint:errcheck
	_, err = wc.Write(body)
	return err
}

func buildBody(from string, msg Message) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))

	if msg.HTML != "" {
		boundary := "==OpsOwlBoundary=="
		sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q\r\n", boundary))
		sb.WriteString("\r\n")
		// plain text part
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.Body)
		sb.WriteString("\r\n")
		// HTML part
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.HTML)
		sb.WriteString("\r\n")
		sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		sb.WriteString("\r\n")
		sb.WriteString(msg.Body)
	}

	return []byte(sb.String())
}
