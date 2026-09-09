package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/kaizakin/siphon/internal/email"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
	email_provider string
	resend_api_key string
	smtp_host      string
	smtp_port      string
	kafka_url      string
	from_email     string
}

func main() {
	_ = godotenv.Load()

	cfg := Config{
		email_provider: config.GetenvDefault("EMAIL_PROVIDER", "resend"),
		resend_api_key: config.GetenvDefault("RESEND_API_KEY", ""),
		smtp_host:      config.GetenvDefault("SMTP_HOST", "localhost"),
		smtp_port:      config.GetenvDefault("SMTP_PORT", "1025"),
		kafka_url:      config.Getenv("KAFKA_URL"),
		from_email:     config.Getenv("FROM_EMAIL"),
	}

	var sender email.Sender
	switch cfg.email_provider {
	case "smtp":
		sender = email.NewSMTPClient(cfg.smtp_host, cfg.smtp_port, cfg.from_email)
	case "resend":
		sender = email.NewResendClient(cfg.resend_api_key, cfg.from_email)
	default:
		log.Fatalf("unknown EMAIL_PROVIDER %q (expected \"resend\" or \"smtp\")", cfg.email_provider)
	}

	templates, err := email.NewTemplateManager()
	if err != nil {
		log.Fatal(err)
	}

	emailsvc := email.NewService(sender, templates)

	router := email.NewRouter(emailsvc)

	consumer := email.NewConsumer([]string{cfg.kafka_url}, "events", "email-service", router)

	log.Print("Starting the email service!..")
	if err := consumer.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
