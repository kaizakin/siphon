package email

import (
	"context"
	"encoding/json"
)

type TemplateHandler[T any] struct {
	email        *Service
	templateName string
	subject      string
}

func NewTemplateHandler[T any](email *Service, templateName string, subject string) *TemplateHandler[T] {
	return &TemplateHandler[T]{
		email:        email,
		templateName: templateName,
		subject:      subject,
	}
}

func (h *TemplateHandler[T]) Send(ctx context.Context, event Event) error {
	_ = ctx

	var payload T
	if err := decodeEventData(event.Data, &payload); err != nil {
		return err
	}

	return h.email.SendTemplate(h.templateName, h.subject, payload, []string{event.Recipient})
}

func decodeEventData(data map[string]any, target any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}
