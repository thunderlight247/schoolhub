package auth

// Clerk admin-issued invitations (expert-pass access-control follow-up).
// When an admin provisions a parent/teacher login account, the backend
// asks Clerk (via the Backend API) to email that person an official
// "set your password / sign up" invitation for the SAME email address the
// account was provisioned with. The recipient clicks the email link,
// creates their Clerk identity with that email, and the existing hybrid
// bridge (POST /auth/clerk-session → email match → backend session) lets
// them straight in. Classic email/password login keeps working regardless.
//
// The flow is deliberately best-effort and optional: without
// CLERK_SECRET_KEY nothing happens, and any Clerk API failure is logged
// and swallowed — an account must never fail to provision because of a
// mail/identity-provider outage (same posture as the Resend mailer).

import (
	"context"
	"errors"
	"log"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/invitation"
)

// ClerkInviter creates and revokes Clerk invitations when the bridge is
// configured. A nil/zero inviter is a valid no-op.
type ClerkInviter struct {
	enabled bool
	// redirectURL sends the invitation's "create account" link back to the
	// app's register page (optional; Clerk's dashboard default applies
	// when empty).
	redirectURL string
}

// NewClerkInviter wires the inviter to the deployment's secret key. An
// empty key disables it (mirrors NewClerkVerifier's posture).
func NewClerkInviter(secretKey, redirectURL string) *ClerkInviter {
	if secretKey == "" {
		return &ClerkInviter{enabled: false}
	}
	// clerk.SetKey is already called by NewClerkVerifier; calling it again
	// is harmless (idempotent global) and keeps this constructor standalone
	// for tests.
	clerk.SetKey(secretKey)
	return &ClerkInviter{enabled: true, redirectURL: redirectURL}
}

// Enabled reports whether invitations can be issued.
func (v *ClerkInviter) Enabled() bool {
	return v != nil && v.enabled
}

// Create sends (or re-sends) a Clerk invitation to the email. Returns the
// Clerk invitation id ("inv_...") so callers can log/link it, or "" when
// the inviter is disabled or the provider rejected the call (logged,
// non-fatal). ignoreExisting=true means an email that already has a
// pending invitation just gets a fresh one rather than an error.
func (v *ClerkInviter) Create(ctx context.Context, email string) string {
	if !v.Enabled() {
		return ""
	}
	params := &invitation.CreateParams{
		EmailAddress:   email,
		Notify:         boolPtr(true),
		IgnoreExisting: boolPtr(true),
	}
	if v.redirectURL != "" {
		params.RedirectURL = &v.redirectURL
	}

	// The SDK's default HTTP client has a 5-second total timeout, and the
	// first TLS handshake from this host occasionally exceeds it (observed
	// once in dev). One retry absorbs a cold-start hiccup without turning
	// mail into a latency problem — the caller is already in a goroutine.
	var inv *clerk.Invitation
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		inv, err = invitation.Create(ctx, params)
		if err == nil {
			return inv.ID
		}
		if isSkippableClerkErr(err) {
			// 4xx param rejections won't improve on retry.
			break
		}
	}
	log.Printf("clerk: invitation create for %s failed: %v", email, err)
	return ""
}

// isSkippableClerkErr reports whether an error is a Clerk API 4xx-style
// rejection (APIErrorResponse, which the SDK returns for well-formed API
// errors) as opposed to a transport error — retrying the former would not
// fix anything.
func isSkippableClerkErr(err error) bool {
	var apiErr *clerk.APIErrorResponse
	return errors.As(err, &apiErr)
}

// Revoke a pending invitation (best-effort). Used when an account is
// blocked before ever being accepted, so a stale "create your account"
// email can't linger. Failures are logged and swallowed.
func (v *ClerkInviter) Revoke(ctx context.Context, invitationID string) {
	if !v.Enabled() || invitationID == "" {
		return
	}
	if _, err := invitation.Revoke(ctx, invitationID); err != nil {
		log.Printf("clerk: invitation revoke for %s failed: %v", invitationID, err)
	}
}

func boolPtr(b bool) *bool { return &b }
