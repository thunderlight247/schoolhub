package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
)

const contextKeyUser = "schoolhub.auth.user"

// RequireAuth is Gin middleware that rejects any request without a valid,
// unexpired session whose user account is still ACTIVE. On success it
// stores the authenticated user in the request context for handlers to
// retrieve via CurrentUser. Authorization (does this user have permission
// X) is a separate, later concern — see M5 — this only establishes who
// the caller is.
func RequireAuth(service *Service, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
			c.Abort()
			return
		}

		authUser, err := service.ValidateSession(c.Request.Context(), token)
		if err != nil {
			apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Session is invalid or has expired.", nil)
			c.Abort()
			return
		}

		c.Set(contextKeyUser, authUser)
		c.Next()
	}
}

// CurrentUser retrieves the authenticated user set by RequireAuth. The
// second return value is false if called on a route not mounted behind
// RequireAuth — callers should treat that as a programming error, not a
// user-facing 401, since it means the route was wired up wrong.
func CurrentUser(c *gin.Context) (AuthenticatedUser, bool) {
	val, exists := c.Get(contextKeyUser)
	if !exists {
		return AuthenticatedUser{}, false
	}
	authUser, ok := val.(AuthenticatedUser)
	return authUser, ok
}
