// Package parentportal HTTP handlers (milestone M16).
package parentportal

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type ServiceAPI interface {
	GetParentOverview(ctx context.Context, userID uuid.UUID) (ParentOverviewData, error)
	GetChildDetail(ctx context.Context, userID uuid.UUID, studentID uuid.UUID) (ChildDetail, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetParentOverview(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	data, err := h.service.GetParentOverview(c.Request.Context(), authUser.User.ID)
	if err != nil {
		if errors.Is(err, ErrGuardianNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve parent portal overview.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetChildDetail(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a valid UUID.", nil)
		return
	}

	detail, err := h.service.GetChildDetail(c.Request.Context(), authUser.User.ID, studentID)
	if err != nil {
		switch {
		case errors.Is(err, ErrChildAccessDenied):
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		case errors.Is(err, ErrGuardianNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve child detail.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail})
}
