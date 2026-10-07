// Package dashboard HTTP handlers (milestone M14).
// Exposes the admin dashboard metrics, KPI calculations, and quick actions
// under RBAC authorization.
package dashboard

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type ServiceAPI interface {
	GetAdminDashboard(ctx context.Context) (AdminDashboardData, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetAdminDashboard(c *gin.Context) {
	_, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	data, err := h.service.GetAdminDashboard(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load admin dashboard data.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
