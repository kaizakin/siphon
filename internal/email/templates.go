package email

import (
	"bytes"
	"fmt"
	"html/template"
)

type TemplateManager struct {
	templates map[string]*template.Template
}

func NewTemplateManager() (*TemplateManager, error) {
	eventtemplates := []string{
		"order_cancelled",
		"order_failed",
		"order_success",
		"payment_failed",
		"payment_refunded",
		"payment_success",
		"signup_thankyou",
	}

	templates := make(map[string]*template.Template)

	for _, name := range eventtemplates {
		tmpl, err := template.ParseFiles(
			"templates/layout.html",
			fmt.Sprintf("templates/%s.html", name),
		)
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	return &TemplateManager{
		templates: templates,
	}, nil
}

func (t *TemplateManager) Render(name string, data any) (string, error) {
  tmpl := t.templates[name]

  var buf bytes.Buffer
  err := tmpl.Execute(&buf, data)
  if err != nil {
    return "", err
  }

  return buf.String(), nil
}
