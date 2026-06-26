package email

type Service struct {
	provider  *ResendClient
	templates *TemplateManager
}

func NewService(provider *ResendClient, templates *TemplateManager) *Service {
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
