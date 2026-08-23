package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/kaizakin/siphon/internal/auth/sqlc"
	"github.com/kaizakin/siphon/pkg/config"
)

func main() {
	emailFlag := flag.String("email", "", "Email of the user to promote or create as admin")
	passwordFlag := flag.String("password", "", "Password if creating a new admin user (min 8 chars)")
	flag.Parse()

	email := *emailFlag
	if email == "" && len(flag.Args()) > 0 {
		email = flag.Arg(0)
	}

	if email == "" {
		fmt.Println("Email not provided")
		os.Exit(1)
	}

	// Load .env if present
	_ = godotenv.Load()

	dbURL := config.Getenv("DATABASE_URL")
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	// Check if user already exists
	existingUser, err := queries.GetUserByEmail(ctx, email)
	if err == nil {
		// User exists, promote to admin
		updatedUser, err := queries.UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{
			Email: email,
			Role:  "admin",
		})
		if err != nil {
			log.Fatalf("Failed to update user role: %v", err)
		}
		log.Printf("Successfully set role to 'admin' for existing user: %s (ID: %s, Previous Role: %s)", updatedUser.Email, updatedUser.ID.String(), existingUser.Role)
		return
	}

	// User does not exist, check if password was provided to create new admin
	if *passwordFlag == "" {
		log.Fatalf("User with email %q does not exist. Provide -password <pass> to create a new admin user directly.", email)
	}

	if len(*passwordFlag) < 8 {
		log.Fatalf("Password must be at least 8 characters long")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*passwordFlag), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	userID := pgtype.UUID{
		Bytes: uuid.New(),
		Valid: true,
	}

	newUser, err := queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           userID,
		Email:        email,
		PasswordHash: string(hash),
		Role:         "admin",
	})
	if err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Printf("Successfully created new admin user: %s (ID: %s, Role: %s)", newUser.Email, newUser.ID.String(), newUser.Role)
}
