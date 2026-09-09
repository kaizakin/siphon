package email

// Sender delivers a rendered email. ResendClient and SMTPClient both implement it.
type Sender interface {
	Send(to []string, subject string, html string) error
}

type Service struct {
	provider  Sender
	templates *TemplateManager
}

func NewService(provider Sender, templates *TemplateManager) *Service {
	return &Service{
		provider:  provider,
		templates: templates,
	}
}

func (s *Service) SendTemplate(templatename string, subject string, data any, to []string) error {
	html, err := s.templates.Render(templatename, data)
	if err != nil {
		return err
	}

	return s.provider.Send(to, subject, html)
}
