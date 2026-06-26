package email

import (
	"log"

	"github.com/resend/resend-go/v3"
)

type Event struct {
	EventType string         `json:"event_type"`
	Recipient string         `json:"recipient"`
	Data      map[string]any `json:"data"`
}

type ResendClient struct {
	client *resend.Client
	from   string
}

func NewResendClient(apikey string, from string) *ResendClient {
	return &ResendClient{
		client: resend.NewClient(apikey),
		from:   from,
	}
}

// send email
func (r *ResendClient) Send(to []string, subject string, html string) error {
	params := &resend.SendEmailRequest{
		From:    r.from,
		To:      to,
		Subject: subject,
		Html:    html,
	}

	sent, err := r.client.Emails.Send(params)
	if err != nil {
		log.Fatalf("failed to send email: %v", err)
	}

	log.Printf("email sent: %s", sent.Id)

	return err
}
