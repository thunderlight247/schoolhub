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
// registrable domain; a split-domain production deployment (e.g. a Vercel
// domain for the frontend and a separate host for the API) will need this
// revisited — see the note in M4's report, not solved here.
type CookieConfig struct {
	Name   string
	Domain string
	Secure bool
}

// Handler holds the HTTP handlers for authentication endpoints.
type Handler struct {
	service *Service
	cookie  CookieConfig
}

func NewHandler(service *Service, cookie CookieConfig) *Handler {
	return &Handler{service: service, cookie: cookie}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// userDTO is what the API actually returns for "the current user" — never
// the generated db.User directly, which has a PasswordHash field that
// must never leave the server.
type userDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Status    string `json:"status"`
}

func toUserDTO(u AuthenticatedUser) userDTO {
	return userDTO{
		ID:        u.User.ID.String(),
		Email:     u.User.Email,
		FirstName: u.Person.FirstName,
		LastName:  u.Person.LastName,
		Status:    u.User.Status,
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

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookie.Name, token, int(SessionDuration.Seconds()), "/", h.cookie.Domain, h.cookie.Secure, true)

	c.JSON(http.StatusOK, gin.H{"user": toUserDTO(authUser)})
}

// Logout handles POST /api/v1/auth/logout. Always succeeds, even if the
// caller wasn't actually logged in — logging out twice should be harmless.
func (h *Handler) Logout(c *gin.Context) {
	if token, err := c.Cookie(h.cookie.Name); err == nil && token != "" {
		_ = h.service.Logout(c.Request.Context(), token)
	}

	c.SetSameSite(http.SameSiteLaxMode)
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
