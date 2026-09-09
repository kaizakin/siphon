package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPClient sends mail over plain SMTP, no auth. Meant for local testing
type SMTPClient struct {
	addr string // host:port
	from string
}

func NewSMTPClient(host string, port string, from string) *SMTPClient {
	return &SMTPClient{
		addr: fmt.Sprintf("%s:%s", host, port),
		from: from,
	}
}

func (c *SMTPClient) Send(to []string, subject string, html string) error {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		c.from, strings.Join(to, ", "), subject, html,
	)

	return smtp.SendMail(c.addr, nil, c.from, to, []byte(msg))
}
