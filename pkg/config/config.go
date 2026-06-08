package config

import (
	"log"
	"os"
)

type Config struct {
	Port string
	DB_URL string
}

func Load() Config {
	port := os.Getenv("PORT")
	dbUrl := os.Getenv("DATABASE_URL")

	if port == "" {
		port = "8080"
	}

	if dbUrl == "" {
		log.Fatal("Database Url is not found!")
	}

	return Config{
		Port: port,
		DB_URL: dbUrl,
	}
}
