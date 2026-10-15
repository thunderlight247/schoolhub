package auth

// Clerk bridge (optional). When CLERK_SECRET_KEY is configured, the backend
// can verify a Clerk session JWT and issue our standard httpOnly session
// cookie for the provisioned account with the matching email — the "hybrid
// bridge" pattern: Clerk owns the sign-in UX, our M4 sessions and M5 RBAC own
// everything downstream.
//
// When CLERK_SECRET_KEY is empty, the bridge endpoint returns 404-style
// "not configured" and the app uses the classic email/password flow only.
//
// Accounts cannot be created through this endpoint: the email from the
// verified Clerk token must already exist in the users table (accounts are
// provisioned by admins in the staff directory). This matches how school
// MIS systems work — staff/parents/students don't self-register.

import (
	"context"
	"errors"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

var (
	// ErrClerkNotConfigured means the caller should fall back to default
	// session auth (the endpoint isn't mounted in that case, so this is a
	// defensive guard).
	ErrClerkNotConfigured = errors.New("auth: clerk bridge is not configured")
	// ErrClerkTokenInvalid covers any rejected token (expired, wrong
	// audience, failed signature) without leaking which specific reason.
	ErrClerkTokenInvalid = errors.New("auth: clerk token invalid")
	// ErrClerkNoEmail means the verified Clerk user carries no verified
	// primary email address.
	ErrClerkNoEmail = errors.New("auth: clerk user has no verified email")
	// ErrClerkNoAccount means the verified identity has no provisioned
	// account in this deployment.
	ErrClerkNoAccount = errors.New("auth: no provisioned account for this identity")
)

// ClerkVerifier validates Clerk session tokens when configured. A nil
// verifier means the bridge is disabled.
type ClerkVerifier struct {
	enabled bool
}

// NewClerkVerifier wires the Clerk SDK to the deployment's secret key. An
// empty key disables the verifier (and the bridge endpoint with it).
func NewClerkVerifier(secretKey string) *ClerkVerifier {
	if secretKey == "" {
		return &ClerkVerifier{enabled: false}
	}
	clerk.SetKey(secretKey)
	return &ClerkVerifier{enabled: true}
}

// Enabled reports whether the bridge is active.
func (v *ClerkVerifier) Enabled() bool {
	return v != nil && v.enabled
}

// VerifyToken checks the Clerk JWT signature/validity server-side and
// returns the verified primary email address of the Clerk user it belongs
// to. Token rejection is deliberately undifferentiated.
func (v *ClerkVerifier) VerifyToken(ctx context.Context, token string) (string, error) {
	if !v.Enabled() {
		return "", ErrClerkNotConfigured
	}

	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{Token: token})
	if err != nil {
		return "", ErrClerkTokenInvalid
	}

	clerkUser, err := user.Get(ctx, claims.Subject)
	if err != nil {
		return "", ErrClerkTokenInvalid
	}

	// Require a verified primary email before an account lookup is allowed.
	primary := clerkUser.PrimaryEmailAddressID
	if primary == nil {
		return "", ErrClerkNoEmail
	}
	for _, addr := range clerkUser.EmailAddresses {
		email := addr.EmailAddress
		if addr.ID == *primary && addr.Verification != nil && addr.Verification.Status == "verified" {
			return email, nil
		}
	}
	return "", ErrClerkNoEmail
}

// LoginWithClerk verifies the token and issues a backend session for the
// provisioned account with the verified email — the same session shape the
// classic Login produces.
func (s *Service) LoginWithClerk(ctx context.Context, verifier *ClerkVerifier, token, userAgent, ipAddress string) (string, AuthenticatedUser, error) {
	email, err := verifier.VerifyToken(ctx, token)
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	userRow, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", AuthenticatedUser{}, ErrClerkNoAccount
	}
	if userRow.Status != "ACTIVE" {
		return "", AuthenticatedUser{}, ErrAccountNotActive
	}

	sessionToken, err := GenerateToken()
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	_, err = s.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    userRow.ID,
		TokenHash: HashToken(sessionToken),
		UserAgent: toText(userAgent),
		IpAddress: toText(ipAddress),
		ExpiresAt: toTimestamptz(time.Now().Add(SessionDuration)),
	})
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	if err := s.queries.RecordUserLogin(ctx, userRow.ID); err != nil {
		return "", AuthenticatedUser{}, err
	}

	person, err := s.queries.GetPersonByID(ctx, userRow.PersonID)
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	authUser, err := s.loadAuthorization(ctx, AuthenticatedUser{User: userRow, Person: person})
	if err != nil {
		return "", AuthenticatedUser{}, err
	}

	return sessionToken, authUser, nil
}
