package observability

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEmailRejectsPrivateSMTPBeforeConnecting(t *testing.T) {
	service := NewNotificationService(nil)
	err := service.sendEmail(map[string]interface{}{
		"smtpHost": "127.0.0.1", "smtpPort": float64(25),
		"recipients": []interface{}{"recipient@example.invalid"},
		"fromAddr":   "sender@example.invalid",
	}, "fixture", "info", nil)
	if err == nil || !strings.Contains(err.Error(), "ssrf:") {
		t.Fatalf("private SMTP must be rejected by the SSRF boundary: %v", err)
	}
}

func TestSMTPUsesConfiguredPortAndDeliversWithoutChangingWebhookPolicy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			received <- "accept failed"
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		reader := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 localhost ESMTP\r\n")
		body := ""
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				received <- "read failed"
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"), strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				fmt.Fprint(conn, "250 OK\r\n")
			case strings.HasPrefix(line, "DATA"):
				fmt.Fprint(conn, "354 send data\r\n")
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						received <- "body failed"
						return
					}
					if line == ".\r\n" {
						break
					}
					body += line
				}
				fmt.Fprint(conn, "250 queued\r\n")
			case strings.HasPrefix(line, "QUIT"):
				fmt.Fprint(conn, "221 bye\r\n")
				received <- body
				return
			default:
				received <- "unexpected SMTP command"
				return
			}
		}
	}()
	service := NewNotificationService(nil)
	// Only this loopback fixture relaxes the policy; defaults still reject it.
	service.ssrfValidator.BlockPrivateIPs = false
	service.ssrfValidator.DNSRebindProtection = false
	oldTimeout := service.ssrfValidator.Timeout
	_, portString, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portString)
	err = service.sendSMTP("127.0.0.1", port, "", "", "sender@example.invalid", []string{"receiver@example.invalid"}, []byte("Subject: fixture\r\n\r\nhello\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if body := <-received; !strings.Contains(body, "hello") {
		t.Fatalf("SMTP payload missing: %q", body)
	}
	if service.ssrfValidator.Timeout != oldTimeout || service.ssrfValidator.AllowedSchemes[0] != "http" {
		t.Fatal("SMTP mutated webhook validator")
	}
	for _, port := range []int{0, -1, 65536} {
		if err := service.sendSMTP("127.0.0.1", port, "", "", "", nil, nil); err == nil {
			t.Fatalf("invalid port accepted: %d", port)
		}
	}
}
