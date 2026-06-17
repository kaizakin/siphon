package config

import (
	"log"
	"os"
)

func Getenv(key string) string {
	value := os.Getenv(key)

	if value == "" {
		log.Fatalf("env variable %s not found", key)
	}

	return value
}