package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/kaizakin/siphon/internal/email"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
	resend_api_key string
	kafka_url      string
	from_email     string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error reading .env file")
	}

	cfg := Config{
		resend_api_key: config.Getenv("RESEND_API_KEY"),
		kafka_url:      config.Getenv("KAFKA_URL"),
		from_email:     config.Getenv("FROM_EMAIL"),
	}

	resendClient := email.NewResendClient(
		cfg.resend_api_key,
		cfg.from_email,
	)

	templates, err := email.NewTemplateManager()
	if err != nil {
		log.Fatal(err)
	}

	emailsvc := email.NewService(resendClient, templates)

	router := email.NewRouter(emailsvc)

	consumer := email.NewConsumer([]string{cfg.kafka_url}, "events", "email-service", router)

	log.Print("Starting the email service!..")
	if err := consumer.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
