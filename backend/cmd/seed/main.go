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
//	go run ./cmd/seed -email alice@example.com -password '…'
//	go run ./cmd/seed -role TEACHER -email bob@example.com -password '…'
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
	email := flag.String("email", "", "login email for the seeded user (required)")
	password := flag.String("password", "", "login password for the seeded user (required — pick a strong one; there is no fixed default)")
	firstName := flag.String("first-name", "QA", "first name")
	lastName := flag.String("last-name", "Admin", "last name")
	role := flag.String("role", "ADMIN", "role to assign: ADMIN, TEACHER, STAFF, PARENT, or STUDENT")
	flag.Parse()
	if *email == "" || *password == "" {
		log.Fatal("seed: -email and -password are required")
	}

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

	// Bootstrap invariant (mirrors migration 000014's promotion of every
	// pre-existing ADMIN): if you are seeding this account's first admin,
	// it must also be able to MANAGE admins — on a fresh database the
	// migration has already run, so the only remaining way to get a
	// MASTER_ADMIN would otherwise be none at all. Idempotent: skipped
	// when the role is already present.
	if *role == "ADMIN" {
		masterRole, err := queries.GetRoleByName(ctx, "MASTER_ADMIN")
		if err != nil {
			log.Fatalf("looking up MASTER_ADMIN role (did you run migrations?): %v", err)
		}
		var hasMaster bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role_id = $2)`,
			user.ID, masterRole.ID).Scan(&hasMaster); err != nil {
			log.Fatalf("checking existing MASTER_ADMIN role: %v", err)
		}
		if !hasMaster {
			if _, err := pool.Exec(ctx,
				`INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
				user.ID, masterRole.ID); err != nil {
				log.Fatalf("assigning MASTER_ADMIN role: %v", err)
			}
			fmt.Println("  roles:    ADMIN + MASTER_ADMIN (bootstrap)")
		}
	}

	fmt.Printf("Seeded %s user:\n", *role)
	fmt.Printf("  email:    %s\n", *email)
	fmt.Printf("  password: %s\n", *password)
	fmt.Println("DEV ONLY — do not use this command or these credentials in production.")
}
