// Command seed creates a single person + user account with a given role,
// so the login flow built in M4 and the RBAC gates built in M5 have
// something real to test against.
//
// This is NOT the M22 demo dataset (30+ students, staff, guardians, etc.)
// and is NOT how M6's "Admin creates staff, assigns role" flow will work —
// it's a stopgap so section 33's end-to-end flow ("Admin login -> ...") has
// an admin to log in as before either of those exist. Expect this command
// to be deleted once M6 provides a real way to create accounts.
//
// Usage (from backend/, with DATABASE_URL set):
//
//	go run ./cmd/seed -email admin@schoolhub.test -password "ChangeMe123!"
//	go run ./cmd/seed -role TEACHER -email teacher@schoolhub.test -password "ChangeMe123!"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/thunderlight247/schoolhub/backend/config"
	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
	"github.com/thunderlight247/schoolhub/backend/internal/database"
)

func main() {
	email := flag.String("email", "admin@schoolhub.test", "login email for the seeded user")
	password := flag.String("password", "ChangeMe123!", "login password for the seeded user (DEV ONLY — never use a fixed password like this in production)")
	firstName := flag.String("first-name", "Demo", "first name")
	lastName := flag.String("last-name", "Admin", "last name")
	role := flag.String("role", "ADMIN", "role to assign: ADMIN, TEACHER, STAFF, PARENT, or STUDENT")
	flag.Parse()

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	passwordHash, err := auth.HashPassword(*password)
	if err != nil {
		log.Fatalf("hashing password: %v", err)
	}

	person, err := queries.CreatePerson(ctx, db.CreatePersonParams{
		FirstName: *firstName,
		LastName:  *lastName,
	})
	if err != nil {
		log.Fatalf("creating person: %v", err)
	}

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		PersonID:     person.ID,
		Email:        *email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		log.Fatalf("creating user: %v", err)
	}

	roleRow, err := queries.GetRoleByName(ctx, *role)
	if err != nil {
		log.Fatalf("looking up role %q (did you run migrations? M2 seeds the roles table): %v", *role, err)
	}

	if err := queries.AssignRoleToUser(ctx, db.AssignRoleToUserParams{
		UserID: user.ID,
		RoleID: roleRow.ID,
	}); err != nil {
		log.Fatalf("assigning %s role: %v", *role, err)
	}

	fmt.Printf("Seeded %s user:\n", *role)
	fmt.Printf("  email:    %s\n", *email)
	fmt.Printf("  password: %s\n", *password)
	fmt.Println("DEV ONLY — do not use this command or these credentials in production.")
}
