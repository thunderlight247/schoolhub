// M34: forced first-login password change.
//
// Bootstrap accounts (the master admin) are created with a temporary
// password the developer knows, with users.must_change_password = true.
// While the flag is set the auth middleware only lets the caller reach
// change-password / logout / me — nothing else — so the temporary
// credential cannot be used to explore the system. After the change
// (which clears the flag and kills every OTHER session) even the developer
// cannot log the bootstrap account in anymore.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
)

// ErrPasswordChange rejects indistinguishably: wrong current password,
// too-short new password, no-op rotation — the response is the same.
var ErrPasswordChange = errors.New("auth: password change rejected")

// Allowed while must_change_password is set — matched against the route's
// FullPath (the pattern as mounted: "/api/v1/...") plus method.
func passwordChangeAllowed(method, fullPath string) bool {
	if !strings.HasPrefix(fullPath, "/api/v1/auth/") && !strings.HasPrefix(fullPath, "/api/v1/me") {
		return false
	}
	if fullPath == "/api/v1/me" && method == http.MethodGet {
		return true
	}
	if method != http.MethodPost {
		return false
	}
	switch fullPath {
	case "/api/v1/auth/change-password", "/api/v1/auth/clerk-password-changed", "/api/v1/auth/logout", "/api/v1/auth/clerk-session":
		return true
	}
	return false
}

// PasswordChangeService owns the change flow.
type PasswordChangeService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPasswordChangeService(pool *pgxpool.Pool) *PasswordChangeService {
	return &PasswordChangeService{pool: pool, queries: db.New(pool)}
}

// Change verifies currentPassword, stores the new Argon2id hash, clears
// must_change_password, and invalidates every other session for the user.
func (s *PasswordChangeService) Change(ctx context.Context, authUser AuthenticatedUser, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordChange
	}
	currentOK, err := VerifyPassword(currentPassword, authUser.User.PasswordHash)
	if err != nil || !currentOK {
		return ErrPasswordChange
	}
	// Pointless rotation: refusing "change to the same password" is simpler
	// than telling the user to pick yet another one.
	sameOK, _ := VerifyPassword(newPassword, authUser.User.PasswordHash)
	if sameOK {
		return ErrPasswordChange
	}
	if len(newPassword) > 128 {
		return ErrPasswordChange
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	if err := q.UpdateUserPasswordHash(ctx, db.UpdateUserPasswordHashParams{
		ID:           authUser.User.ID,
		PasswordHash: hash,
	}); err != nil {
		return err
	}
	if err := q.SetUserMustChangePassword(ctx, db.SetUserMustChangePasswordParams{
		ID:                 authUser.User.ID,
		MustChangePassword: false,
	}); err != nil {
		return err
	}
	// Kill other live sessions: another device holding the bootstrap
	// password must not keep a working session across the change.
	if authUser.SessionTokenHash != "" {
		if _, err := q.DeleteOtherSessionsForUser(ctx, db.DeleteOtherSessionsForUserParams{
			UserID:    authUser.User.ID,
			TokenHash: authUser.SessionTokenHash,
		}); err != nil {
			return err
		}
	}
	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  authUser.User.ID,
		Action:       audit.ActionPasswordChanged,
		ResourceType: audit.ResourceUser,
		ResourceID:   authUser.User.ID,
		Metadata:     map[string]any{"first_login": true},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ClearForClerkSession clears the flag when the frontend updated the
// password through Clerk (the Clerk session itself proved identity, so no
// current password is checked here). Used in Clerk deployment mode.
func (s *PasswordChangeService) ClearForClerkSession(ctx context.Context, authUser AuthenticatedUser) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	if err := q.SetUserMustChangePassword(ctx, db.SetUserMustChangePasswordParams{
		ID:                 authUser.User.ID,
		MustChangePassword: false,
	}); err != nil {
		return err
	}
	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  authUser.User.ID,
		Action:       audit.ActionPasswordChanged,
		ResourceType: audit.ResourceUser,
		ResourceID:   authUser.User.ID,
		Metadata:     map[string]any{"via": "clerk", "first_login": true},
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ChangeHandler handles POST /api/v1/auth/change-password (session auth).
type ChangeHandler struct {
	change *PasswordChangeService
}

func NewChangeHandler(change *PasswordChangeService) *ChangeHandler {
	return &ChangeHandler{change: change}
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *ChangeHandler) ChangePassword(c *gin.Context) {
	authUser, ok := CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION", "current_password and new_password are required.", nil)
		return
	}
	if err := h.change.Change(c.Request.Context(), authUser, req.CurrentPassword, req.NewPassword); err != nil {
		if err == ErrPasswordChange {
			apierror.Respond(c, http.StatusBadRequest, "PASSWORD_CHANGE_REJECTED",
				"Password change rejected: at least 8 characters, current password required, new password must differ.", nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL", "Password change failed. Try again.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"must_change_password": false},
	})
}

// ClerkPasswordChangedHandler handles POST /api/v1/auth/clerk-password-changed:
// the frontend already rotated the password through Clerk; the backend just
// clears the flag so the gate opens.
type ClerkPasswordChangedHandler struct {
	change *PasswordChangeService
}

func NewClerkPasswordChangedHandler(change *PasswordChangeService) *ClerkPasswordChangedHandler {
	return &ClerkPasswordChangedHandler{change: change}
}

func (h *ClerkPasswordChangedHandler) ClerkPasswordChanged(c *gin.Context) {
	authUser, ok := CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}
	if err := h.change.ClearForClerkSession(c.Request.Context(), authUser); err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL", "Could not confirm the password change. Try again.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"must_change_password": false},
	})
}
