package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

// Errors are deliberately coarse. In particular, ErrInvalidCredentials
// covers both "no such user" and "wrong password" — a handler that
// returned different errors for those two cases would let an attacker
// enumerate which emails have accounts.
var (
	ErrInvalidCredentials = errors.New("auth: invalid email or password")
	ErrAccountNotActive   = errors.New("auth: account is not active")
	ErrSessionInvalid     = errors.New("auth: session not found or expired")
)

// SessionDuration is how long a session stays valid after login.
const SessionDuration = 7 * 24 * time.Hour

// Service implements login, logout, and session validation. It depends on
// db.Querier (the interface sqlc generates) rather than the concrete
// *db.Queries type, so tests in M21 can supply a fake implementation
// without a real database.
type Service struct {
	queries db.Querier
}

func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

// AuthenticatedUser is what a successful login or session lookup returns —
// enough to populate request context without a second round trip for most
// handlers. Roles and Permissions are the caller's role names and effective
// permission codes (resolved through user_roles → role_permissions at
// lookup time), so handlers and the frontend can make authorization-aware
// decisions without extra round trips. Enforcement still happens
// server-side per route via rbac.RequirePermission — these fields are for
// convenience/UX (e.g. rendering the right navigation), never a substitute
// for the middleware check.
type AuthenticatedUser struct {
	User   db.User
	Person db.Person
	// SessionTokenHash is the SHA-256 hash of the presented session token
	// (M34): the password-change flow uses it to invalidate every OTHER
	// session for the same account without logging the active device out.
	SessionTokenHash string
	Roles            []string
	Permissions      []string
}

// loadAuthorization resolves the user's role names and effective permission
// codes and attaches them to authUser. Both slices are initialized empty
// (never nil) so they serialize as [] rather than null. A failure here
// fails closed: Login/ValidateSession propagate the error rather than
// issuing a session whose authorization data couldn't be loaded.
func (s *Service) loadAuthorization(ctx context.Context, authUser AuthenticatedUser) (AuthenticatedUser, error) {
	roles, err := s.queries.ListRolesForUser(ctx, authUser.User.ID)
	if err != nil {
		return AuthenticatedUser{}, err
	}
	authUser.Roles = make([]string, 0, len(roles))
	for _, r := range roles {
		authUser.Roles = append(authUser.Roles, r.Name)
	}

	perms, err := s.queries.ListPermissionsForUser(ctx, authUser.User.ID)
	if err != nil {
		return AuthenticatedUser{}, err
	}
	authUser.Permissions = make([]string, 0, len(perms))
	for _, p := range perms {
		authUser.Permissions = append(authUser.Permissions, p.Code)
	}

	return authUser, nil
}

// Login verifies an email/password pair and, on success, creates a new
// session and returns the raw session token (for the caller to set as a
// cookie) plus the authenticated user. userAgent and ipAddress are stored
// on the session purely for the user's own visibility/audit later
// (e.g. a future "active sessions" screen) — they play no role in
// authorization.
func (s *Service) Login(ctx context.Context, email, password string, userAgent, ipAddress string) (string, AuthenticatedUser, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", AuthenticatedUser{}, ErrInvalidCredentials
	}

	if user.Status != "ACTIVE" {
		return "", AuthenticatedUser{}, ErrAccountNotActive
	}

	ok, err := VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return "", AuthenticatedUser{}, ErrInvalidCredentials
	}

	token, err := GenerateToken()
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	_, err = s.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: HashToken(token),
		UserAgent: toText(userAgent),
		IpAddress: toText(ipAddress),
		ExpiresAt: toTimestamptz(time.Now().Add(SessionDuration)),
	})
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	if err := s.queries.RecordUserLogin(ctx, user.ID); err != nil {
		return "", AuthenticatedUser{}, err
	}

	person, err := s.queries.GetPersonByID(ctx, user.PersonID)
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	authUser, err := s.loadAuthorization(ctx, AuthenticatedUser{User: user, Person: person})
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	return token, authUser, nil
}

// Logout deletes the session matching the given raw token. Deleting a
// session that doesn't exist (already logged out, expired, token
// malformed) is not treated as an error — logging out twice should be
// harmless from the caller's perspective.
func (s *Service) Logout(ctx context.Context, token string) error {
	_ = s.queries.DeleteSession(ctx, HashToken(token))
	return nil
}

// ValidateSession looks up the session for a raw token and returns the
// associated user, provided the session exists, hasn't expired, and the
// user's account is still ACTIVE. This is what RequireAuth (middleware.go)
// calls on every protected request.
func (s *Service) ValidateSession(ctx context.Context, token string) (AuthenticatedUser, error) {
	session, err := s.queries.GetSessionByTokenHash(ctx, HashToken(token))
	if err != nil {
		return AuthenticatedUser{}, ErrSessionInvalid
	}

	if !session.ExpiresAt.Valid || session.ExpiresAt.Time.Before(time.Now()) {
		return AuthenticatedUser{}, ErrSessionInvalid
	}

	user, err := s.queries.GetUserByID(ctx, session.UserID)
	if err != nil {
		return AuthenticatedUser{}, ErrSessionInvalid
	}
	if user.Status != "ACTIVE" {
		return AuthenticatedUser{}, ErrAccountNotActive
	}

	person, err := s.queries.GetPersonByID(ctx, user.PersonID)
	if err != nil {
		return AuthenticatedUser{}, ErrSessionInvalid
	}

	authUser, err := s.loadAuthorization(ctx, AuthenticatedUser{
		User:             user,
		Person:           person,
		SessionTokenHash: HashToken(token),
	})
	if err != nil {
		// Fail closed: a transient failure resolving permissions rejects the
		// request (mapped to 401 by RequireAuth) rather than serving it with
		// unknown authorization data.
		return AuthenticatedUser{}, ErrSessionInvalid
	}

	return authUser, nil
}

// --- pgtype conversion helpers ---
//
// sqlc (with the pgx/v5 driver and no type overrides beyond uuid) maps
// nullable Postgres columns to pgtype wrapper structs rather than Go
// pointers. These two helpers construct them from plain Go values so the
// rest of this package doesn't need to think about pgtype directly.

func toText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
