package observability

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"time"
)

// sendSMTP reuses the released checked dial boundary for the non-HTTP protocol.
// One deadline bounds connect, greeting, TLS, authentication and message delivery.
func (s *NotificationService) sendSMTP(host string, port int, username, password, from string, to []string, message []byte) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid SMTP port")
	}
	validator := *s.ssrfValidator
	validator.AllowedSchemes = []string{"smtp"}
	validator.AllowedPorts = []int{port}
	address := net.JoinHostPort(host, fmt.Sprint(port))
	target := url.URL{Scheme: "smtp", Host: address}
	if err := validator.ValidateURL(target.String()); err != nil {
		return err
	}
	client := validator.CreateHTTPClient(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("invalid transport")
	}
	conn, err := transport.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == 465 {
		secure := tls.Client(conn, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = secure
	}
	smtpClient, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer func() { _ = smtpClient.Close() }()
	if port != 465 {
		if ok, _ := smtpClient.Extension("STARTTLS"); ok {
			if err := smtpClient.StartTLS(tlsConfig); err != nil {
				return err
			}
		}
	}
	if username != "" || password != "" {
		if err := smtpClient.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return err
		}
	}
	if err := smtpClient.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := smtpClient.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := smtpClient.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return smtpClient.Quit()
}
