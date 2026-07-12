package application

import (
	"context"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/waaris/waaris/services/enrollment/internal/domain"
)

type SMTPNotifier struct {
	address string
	from    string
	timeout time.Duration
}

func NewSMTPNotifier(address, from string) *SMTPNotifier {
	return &SMTPNotifier{address: address, from: from, timeout: 5 * time.Second}
}

func (n *SMTPNotifier) Send(ctx context.Context, notification domain.Notification) error {
	connection, err := (&net.Dialer{Timeout: n.timeout}).DialContext(ctx, "tcp", n.address)
	if err != nil {
		return err
	}
	defer connection.Close()
	if err = connection.SetDeadline(time.Now().Add(n.timeout)); err != nil {
		return err
	}

	host, _, err := net.SplitHostPort(n.address)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err = client.Mail(n.from); err != nil {
		return err
	}
	if err = client.Rcpt(notification.RecipientEmail); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := strings.Join([]string{
		"From: " + n.from,
		"To: " + notification.RecipientEmail,
		"Subject: " + notification.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		notification.Body,
	}, "\r\n")
	if _, err = writer.Write([]byte(message)); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
