package demo

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	pgx "github.com/jackc/pgx/v5"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// Handler exposes the demo sign-in endpoints:
//
//	GET  /api/v1/auth/demo-mode   → {"demo_mode": true|false}
//	POST /api/v1/auth/demo-login  → {"as": "admin"|"teacher"|"parent"}
//
// demo-mode is the harmless capability probe the frontend uses to decide
// whether to show the one-click persona picker; it can be mounted even when
// demo mode is off (it just reports false). demo-login is only mounted on
// deployments with config.DemoMode set.
// LoginService is the slice of *Service the handler needs, extracted so
// handler tests can fake persona resolution without a database.
type LoginService interface {
	Login(ctx context.Context, role Role, userAgent, ipAddress string) (string, auth.AuthenticatedUser, error)
}

type Handler struct {
	service LoginService
	cookie  auth.CookieConfig
}

func NewHandler(service LoginService, cookie auth.CookieConfig) *Handler {
	return &Handler{service: service, cookie: cookie}
}

// Mode handles GET /api/v1/auth/demo-mode. Mounted always; the body tells
// the frontend whether the one-click sign-in exists on this deployment.
func Mode(demoMode bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"demo_mode": demoMode})
	}
}

// parseRole accepts exactly the three supported personas.
func parseRole(raw string) (Role, bool) {
	switch Role(raw) {
	case RoleAdmin, RoleTeacher, RoleParent:
		return Role(raw), true
	}
	return "", false
}

type demoLoginRequest struct {
	As string `json:"as" binding:"required"`
}

// Login handles POST /api/v1/auth/demo-login. It behaves exactly like the
// password login on success: same httpOnly session cookie, same {"user":
// …} body shape, so the frontend's session handling needs no special case.
func (h *Handler) Login(c *gin.Context) {
	var req demoLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Choose a demo role: admin, teacher, or parent.", nil)
		return
	}

	role, ok := parseRole(req.As)
	if !ok {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Unknown demo role. Use admin, teacher, or parent.", nil)
		return
	}

	token, authUser, err := h.service.Login(
		c.Request.Context(),
		role,
		c.Request.UserAgent(),
		c.ClientIP(),
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnknownRole):
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR",
				"Unknown demo role. Use admin, teacher, or parent.", nil)
		case errors.Is(err, pgx.ErrNoRows):
			apierror.Respond(c, http.StatusServiceUnavailable, "DEMO_DATA_MISSING",
				"Demo data has not been seeded on this deployment.", nil)
		default:
			log.Printf("demo: demo-login (%s) failed: %v", role, err)
			apierror.Respond(c, http.StatusInternalServerError, "DEMO_LOGIN_FAILED",
				"Demo sign-in failed. Please try again.", nil)
		}
		return
	}

	c.SetSameSite(h.cookie.SameSite)
	c.SetCookie(h.cookie.Name, token, int(auth.SessionDuration.Seconds()), "/", h.cookie.Domain, h.cookie.Secure, true)

	c.JSON(http.StatusOK, gin.H{"user": auth.ToUserDTO(authUser)})
}
