package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/channels"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
)

// EmailChannel delivers notifications via email (SMTP)
type EmailChannel struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
	useTLS   bool
	enabled  bool
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
	UseTLS   bool
}

func NewEmailChannel(cfg EmailConfig) *EmailChannel {
	return &EmailChannel{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
		fromName: cfg.FromName,
		useTLS:   cfg.UseTLS,
		enabled:  cfg.Host != "",
	}
}

func (e *EmailChannel) Name() string {
	return "email"
}

func (e *EmailChannel) Send(ctx context.Context, notification *models.Notification, recipient *channels.ChannelRecipient) error {
	if !e.enabled || recipient.Email == "" {
		return fmt.Errorf("email not configured or recipient email missing")
	}

	// Build email
	subject := notification.Title
	body := notification.Body

	message := fmt.Sprintf("From: %s <%s>\r\n", e.fromName, e.from)
	message += fmt.Sprintf("To: %s\r\n", recipient.Email)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += fmt.Sprintf("<html><body><h2>%s</h2><p>%s</p></body></html>", subject, body)

	// Setup authentication
	auth := smtp.PlainAuth("", e.username, e.password, e.host)

	// Send email
	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	if e.useTLS {
		// TLS configuration
		tlsConfig := &tls.Config{
			ServerName: e.host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.host)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer client.Close()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if err := client.Mail(e.from); err != nil {
			return err
		}

		if err := client.Rcpt(recipient.Email); err != nil {
			return err
		}

		w, err := client.Data()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		return client.Quit()
	}

	// Non-TLS
	return smtp.SendMail(addr, auth, e.from, []string{recipient.Email}, []byte(message))
}

func (e *EmailChannel) SendBatch(ctx context.Context, notifications []*models.Notification, recipients []*channels.ChannelRecipient) error {
	for i, notif := range notifications {
		if err := e.Send(ctx, notif, recipients[i]); err != nil {
			return err
		}
	}
	return nil
}

func (e *EmailChannel) IsEnabled() bool {
	return e.enabled
}

func (e *EmailChannel) HealthCheck(ctx context.Context) error {
	if !e.enabled {
		return fmt.Errorf("email channel not configured")
	}
	return nil
}
