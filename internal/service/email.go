package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"github.com/readygeneration/readygeneration-backend/internal/config"
)

// EmailService sends transactional emails via SMTP.
type EmailService struct {
	cfg config.EmailConfig
}

// NewEmailService creates an EmailService.
func NewEmailService(cfg config.EmailConfig) *EmailService {
	return &EmailService{cfg: cfg}
}

// Send sends a plain-text email to the given recipient.
func (s *EmailService) Send(to, subject, body string) error {
	if !s.cfg.Enabled {
		return fmt.Errorf("email not enabled")
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	msg := []byte(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, s.cfg.From, subject, body))

	if s.cfg.Port == 587 {
		return s.sendStartTLS(addr, to, msg)
	}
	return smtp.SendMail(addr, smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host), s.cfg.From, []string{to}, msg)
}

func (s *EmailService) sendStartTLS(addr, to string, msg []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	if err := client.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(s.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
