// Package demo implements the one-click demo sign-in: instead of sharing
// credentials, a demo deployment lets a visitor open a real session for a
// designated persona (admin, teacher, or parent) with a single click. Every
// persona still holds its real role and its real row-level scoping — the
// teacher sees their own classes, the parent only their own children —
// because no authorization rule is bypassed; only the credential check of
// Login is replaced by a fixed, server-side persona resolution.
//
// SECURITY: this flow is only mounted when config.DemoMode is set (env
// DEMO_MODE). It lets any internet visitor act as the demo school's admin,
// so it must never be enabled on a deployment holding real data.
package demo

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// Role is the visitor-chosen persona. "admin" resolves the demo school's
// master administrator, "teacher" a teaching staff member, and "parent" a
// guardian with linked children.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleParent  Role = "parent"
)

// ErrUnknownRole is returned for any role string other than the three
// supported personas.
var ErrUnknownRole = errors.New("demo: unknown role")

// Service resolves (and if necessary provisions) the account behind each
// demo persona, then hands the account to the auth service to start a real
// session. It talks to the pool directly for persona resolution (small
// one-off lookups, same pattern as auth's password-change service) and to
// the generated auth queries for account mutation.
type Service struct {
	pool     *pgxpool.Pool
	queries  authQuerier
	sessions *auth.Service
}

// authQuerier is the slice of the generated Querier the demo service uses.
// Declared structurally (like the rbac interfaces) rather than taking the
// concrete *db.Queries, so tests can fake it.
type authQuerier interface {
	GetUserByPersonID(ctx context.Context, personID uuid.UUID) (db.User, error)
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	SetUserMustChangePassword(ctx context.Context, arg db.SetUserMustChangePasswordParams) error
	UpdateUserStatus(ctx context.Context, arg db.UpdateUserStatusParams) error
	GetPersonByID(ctx context.Context, personID uuid.UUID) (db.Person, error)
}

// NewService builds the demo sign-in service. sessions is the shared auth
// service: sessions issued here are ordinary sessions, validated by the
// ordinary RequireAuth middleware, and revoked by the ordinary logout.
func NewService(pool *pgxpool.Pool, queries authQuerier, sessions *auth.Service) *Service {
	return &Service{pool: pool, queries: queries, sessions: sessions}
}

// Login resolves the persona for role, ensures an ACTIVE account with the
// role's backing exists for it, and starts a session. The returned user's
// must_change_password flag is always cleared before a session is issued —
// a demo visitor must not be dropped into the forced password-change gate.
func (s *Service) Login(ctx context.Context, role Role, userAgent, ipAddress string) (string, auth.AuthenticatedUser, error) {
	personID, roleName, err := s.resolvePersona(ctx, role)
	if err != nil {
		return "", auth.AuthenticatedUser{}, err
	}

	user, provisioned, err := s.ensureAccount(ctx, personID, roleName)
	if err != nil {
		return "", auth.AuthenticatedUser{}, err
	}
	if provisioned {
		log.Printf("demo: provisioned %s persona account %s (person %s)", role, user.Email, personID)
	}

	// The password gate would block a flagged account behind the
	// change-password screen, defeating the one-click sign-in. Demo
	// accounts are not secrets — their passwords are never told to anyone
	// — so clearing the flag here is safe.
	if user.MustChangePassword {
		if err := s.queries.SetUserMustChangePassword(ctx, db.SetUserMustChangePasswordParams{
			MustChangePassword: false,
			ID:                 user.ID,
		}); err != nil {
			return "", auth.AuthenticatedUser{}, fmt.Errorf("demo: clearing password flag: %w", err)
		}
		user.MustChangePassword = false
	}

	token, authUser, err := s.sessions.StartSessionForUser(ctx, user, userAgent, ipAddress)
	if err != nil {
		return "", auth.AuthenticatedUser{}, fmt.Errorf("demo: starting session: %w", err)
	}
	return token, authUser, nil
}

// resolvePersona maps the chosen role to a person ID and the role name that
// person's account must hold. Every resolution picks the persona that makes
// the demo story work: the school's oldest master administrator, a teaching
// staff member with actual class assignments, and the guardian linked to
// the most students.
func (s *Service) resolvePersona(ctx context.Context, role Role) (uuid.UUID, string, error) {
	switch role {
	case RoleAdmin:
		var personID uuid.UUID
		err := s.pool.QueryRow(ctx,
			`SELECT u.person_id
			 FROM users u
			 JOIN user_roles ur ON ur.user_id = u.id
			 JOIN roles r ON r.id = ur.role_id
			 WHERE r.name = 'MASTER_ADMIN' AND u.status = 'ACTIVE'
			 ORDER BY u.created_at
			 LIMIT 1`).Scan(&personID)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("demo: no master admin found (was the bootstrap admin created?): %w", err)
		}
		return personID, "ADMIN", nil

	case RoleTeacher:
		// STF-001 (Ada Obi) teaches Mathematics & Science across both demo
		// classes — the richest teacher story. Falls back to any active
		// staff member so a freshly seeded but renumbered deployment still
		// works.
		var personID uuid.UUID
		err := s.pool.QueryRow(ctx,
			`SELECT person_id FROM staff
			 WHERE employment_status = 'ACTIVE'
			 ORDER BY (staff_number = 'STF-001') DESC, staff_number
			 LIMIT 1`).Scan(&personID)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("demo: no staff member found (was demo-seed run?): %w", err)
		}
		return personID, "TEACHER", nil

	case RoleParent:
		// The guardian with the most linked students — in the demo dataset
		// that is Abena Boateng, linked to both children.
		var personID uuid.UUID
		err := s.pool.QueryRow(ctx,
			`SELECT g.person_id
			 FROM guardians g
			 LEFT JOIN student_guardians sg ON sg.guardian_id = g.id
			 GROUP BY g.person_id, g.created_at
			 ORDER BY count(sg.id) DESC, g.created_at
			 LIMIT 1`).Scan(&personID)
		if err != nil {
			return uuid.Nil, "", fmt.Errorf("demo: no guardian found (was demo-seed run?): %w", err)
		}
		return personID, "PARENT", nil

	default:
		return uuid.Nil, "", ErrUnknownRole
	}
}

// ensureAccount returns the ACTIVE account backing the persona, creating
// one (with an impossible, never-disclosed password) and granting the role
// if the person doesn't have an account yet. It also re-activates a
// suspended account and guarantees the persona role is held — demo mode
// tolerates touching the flagged/suspended state because the accounts it
// resolves are demo data, not real users.
func (s *Service) ensureAccount(ctx context.Context, personID uuid.UUID, roleName string) (db.User, bool, error) {
	user, err := s.queries.GetUserByPersonID(ctx, personID)
	if err == nil {
		if user.Status != "ACTIVE" {
			if err := s.queries.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
				Status: "ACTIVE",
				ID:     user.ID,
			}); err != nil {
				return db.User{}, false, fmt.Errorf("demo: reactivating persona account: %w", err)
			}
			user.Status = "ACTIVE"
		}
		if err := s.grantRole(ctx, user.ID, roleName); err != nil {
			return db.User{}, false, err
		}
		return user, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, false, fmt.Errorf("demo: persona account lookup: %w", err)
	}

	person, err := s.queries.GetPersonByID(ctx, personID)
	if err != nil {
		return db.User{}, false, fmt.Errorf("demo: persona person lookup: %w", err)
	}

	// The password is a fresh random secret that is hashed, stored, and
	// immediately discarded — nobody ever learns it, so the account can
	// only be entered through this demo flow.
	password, err := randomSecret()
	if err != nil {
		return db.User{}, false, fmt.Errorf("demo: generating persona password: %w", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.User{}, false, fmt.Errorf("demo: hashing persona password: %w", err)
	}

	email := fmt.Sprintf("%s.%s@demo.schoolhub", slug(person.FirstName), slug(person.LastName))
	created, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		PersonID:     personID,
		Email:        email,
		PasswordHash: hash,
	})
	if err != nil {
		// Duplicate email can only mean another person took the same
		// slug — surface it rather than guessing further.
		return db.User{}, false, fmt.Errorf("demo: creating persona account %q: %w", email, err)
	}

	if err := s.grantRole(ctx, created.ID, roleName); err != nil {
		return db.User{}, false, err
	}
	return created, true, nil
}

// grantRole inserts user_roles (idempotent). Role IDs come from the seeded
// roles table (migration 000002).
func (s *Service) grantRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id)
		 SELECT $1, id FROM roles WHERE name = $2
		 ON CONFLICT (user_id, role_id) DO NOTHING`,
		userID, roleName)
	if err != nil {
		return fmt.Errorf("demo: granting role %s: %w", roleName, err)
	}
	if tag.RowsAffected() > 0 {
		log.Printf("demo: granted role %s to user %s", roleName, userID)
	}
	return nil
}

// randomSecret returns 32 bytes of crypto/rand, base64-encoded (~43 chars).
func randomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// slug lowercases a name and keeps only a-z0-9, for use in a demo email
// local part.
func slug(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
		}
	}
	return string(out)
}
