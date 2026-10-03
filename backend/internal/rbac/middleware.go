package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// RequirePermission returns Gin middleware that rejects any request from a
// user who doesn't hold the given permission code. Must be mounted after
// auth.RequireAuth, which is what populates the current user this reads.
// This is the server-side enforcement section 11 requires ("Hiding a
// button in the frontend is NOT security") — every domain handler from M6
// onward gates its mutating/sensitive routes with this, not just the UI.
func RequirePermission(service *Service, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUser, ok := auth.CurrentUser(c)
		if !ok {
			// Programming error (route mounted without auth.RequireAuth
			// first), not a normal user-facing case — but fail closed
			// rather than panic.
			apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
			c.Abort()
			return
		}

		has, err := service.HasPermission(c.Request.Context(), authUser.User.ID, code)
		if err != nil {
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not check permissions.", nil)
			c.Abort()
			return
		}
		if !has {
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to perform this action.", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole is like RequirePermission but checks role membership
// directly. Reserved for the handful of operations not covered by a
// section 11 permission code — currently just RBAC management itself
// (see handlers.go). Prefer RequirePermission for everything else; a
// role-based gate here and a permission-based gate there for the same
// kind of decision would be inconsistent.
func RequireRole(service *Service, roleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUser, ok := auth.CurrentUser(c)
		if !ok {
			apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
			c.Abort()
			return
		}

		has, err := service.HasRole(c.Request.Context(), authUser.User.ID, roleName)
		if err != nil {
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not check role.", nil)
			c.Abort()
			return
		}
		if !has {
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to perform this action.", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
