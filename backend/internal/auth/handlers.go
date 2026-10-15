package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
)

// CookieConfig controls how the session cookie is set. Secure should be
// true in production (cookie only sent over HTTPS) and can be false for
// local HTTP development. CookieDomain left empty makes it a host-only
// cookie, which is correct as long as frontend and backend share a
// registrable domain. For a split-domain deployment (frontend on one
// registrable domain, API on another — e.g. a *.pages.dev frontend and a
// *.onrender.com API) the cookie must be SameSite=None; Secure: Lax
// cookies are dropped by the browser when set in a cross-site fetch's
// response, which would silently break login. SameSite carries that
// choice; see config.CookieSameSiteMode for how it is loaded.
type CookieConfig struct {
	Name     string
	Domain   string
	Secure   bool
	SameSite http.SameSite
}

// Handler holds the HTTP handlers for authentication endpoints.
type Handler struct {
	service       *Service
	cookie        CookieConfig
	clerkVerifier *ClerkVerifier
}

func NewHandler(service *Service, cookie CookieConfig) *Handler {
	return &Handler{service: service, cookie: cookie}
}

// NewHandlerWithClerk builds the handler with the optional Clerk bridge.
// A nil verifier leaves the classic flow untouched.
func NewHandlerWithClerk(service *Service, cookie CookieConfig, verifier *ClerkVerifier) *Handler {
	return &Handler{service: service, cookie: cookie, clerkVerifier: verifier}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// userDTO is what the API actually returns for "the current user" — never
// the generated db.User directly, which has a PasswordHash field that
// must never leave the server. Roles/Permissions are included (M5) so the
// frontend can render role-aware navigation; they are informational — the
// backend still enforces every permission server-side.
type userDTO struct {
	ID                 string   `json:"id"`
	Email              string   `json:"email"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	Status             string   `json:"status"`
	MustChangePassword bool     `json:"must_change_password"`
	Roles              []string `json:"roles"`
	Permissions        []string `json:"permissions"`
}

func toUserDTO(u AuthenticatedUser) userDTO {
	return userDTO{
		ID:                 u.User.ID.String(),
		Email:              u.User.Email,
		FirstName:          u.Person.FirstName,
		LastName:           u.Person.LastName,
		Status:             u.User.Status,
		MustChangePassword: u.User.MustChangePassword,
		Roles:              u.Roles,
		Permissions:        u.Permissions,
	}
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required.", nil)
		return
	}

	token, authUser, err := h.service.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		c.Request.UserAgent(),
		c.ClientIP(),
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountNotActive):
			apierror.Respond(c, http.StatusForbidden, "ACCOUNT_NOT_ACTIVE", "This account is not active.", nil)
		default:
			apierror.Respond(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password.", nil)
		}
		return
	}

	c.SetSameSite(h.cookie.SameSite)
	c.SetCookie(h.cookie.Name, token, int(SessionDuration.Seconds()), "/", h.cookie.Domain, h.cookie.Secure, true)

	c.JSON(http.StatusOK, gin.H{"user": toUserDTO(authUser)})
}

// Logout handles POST /api/v1/auth/logout. Always succeeds, even if the
// caller wasn't actually logged in — logging out twice should be harmless.
func (h *Handler) Logout(c *gin.Context) {
	if token, err := c.Cookie(h.cookie.Name); err == nil && token != "" {
		_ = h.service.Logout(c.Request.Context(), token)
	}

	c.SetSameSite(h.cookie.SameSite)
	c.SetCookie(h.cookie.Name, "", -1, "/", h.cookie.Domain, h.cookie.Secure, true)

	c.Status(http.StatusNoContent)
}

// Me handles GET /api/v1/me. Must be mounted behind RequireAuth.
func (h *Handler) Me(c *gin.Context) {
	authUser, ok := CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Not authenticated.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": toUserDTO(authUser)})
}

// clerkBridgePayload is what the frontend posts after a successful Clerk
// sign-in: the raw Clerk session JWT for server-side verification.
type clerkBridgePayload struct {
	Token string `json:"token" binding:"required"`
}

// ClerkSession handles POST /api/v1/auth/clerk-session — the hybrid-bridge
// endpoint. It verifies the Clerk JWT server-side (signature, expiry,
// verified primary email), finds the provisioned account with that email,
// and issues the same httpOnly session cookie the classic login does. Only
// mounted when CLERK_SECRET_KEY is configured.
func (h *Handler) ClerkSession(c *gin.Context) {
	var req clerkBridgePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Token is required.", nil)
		return
	}

	token, authUser, err := h.service.LoginWithClerk(
		c.Request.Context(),
		h.clerkVerifier,
		req.Token,
		c.Request.UserAgent(),
		c.ClientIP(),
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrAccountNotActive):
			apierror.Respond(c, http.StatusForbidden, "ACCOUNT_NOT_ACTIVE", "This account is not active.", nil)
		case errors.Is(err, ErrClerkNoAccount):
			apierror.Respond(c, http.StatusForbidden, "NO_ACCOUNT", "No account exists for this identity. Contact the school office.", nil)
		case errors.Is(err, ErrClerkNotConfigured):
			apierror.Respond(c, http.StatusNotFound, "NOT_CONFIGURED", "Clerk sign-in is not available on this deployment.", nil)
		default:
			// Invalid token, no verified email, session storage failure —
			// all surface as one undifferentiated rejection.
			apierror.Respond(c, http.StatusUnauthorized, "INVALID_TOKEN", "Sign-in could not be verified. Please try again.", nil)
		}
		return
	}

	c.SetSameSite(h.cookie.SameSite)
	c.SetCookie(h.cookie.Name, token, int(SessionDuration.Seconds()), "/", h.cookie.Domain, h.cookie.Secure, true)

	c.JSON(http.StatusOK, gin.H{"user": toUserDTO(authUser)})
}
