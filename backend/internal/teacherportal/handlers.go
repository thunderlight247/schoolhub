// Package teacherportal HTTP handlers (milestone M15).
package teacherportal

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
	GetTeacherOverview(ctx context.Context, userID uuid.UUID) (TeacherOverviewData, error)
	GetTeacherClasses(ctx context.Context, userID uuid.UUID) ([]AssignedClass, error)
	GetClassRoster(ctx context.Context, userID uuid.UUID, classID uuid.UUID) ([]StudentRosterItem, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

func parseUUIDParam(c *gin.Context, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(key))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a valid UUID.", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) GetTeacherOverview(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	data, err := h.service.GetTeacherOverview(c.Request.Context(), authUser.User.ID)
	if err != nil {
		if errors.Is(err, ErrTeacherNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve teacher portal overview.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetTeacherClasses(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	classes, err := h.service.GetTeacherClasses(c.Request.Context(), authUser.User.ID)
	if err != nil {
		if errors.Is(err, ErrTeacherNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve teacher classes.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": classes})
}

func (h *Handler) GetClassRoster(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	classID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	roster, err := h.service.GetClassRoster(c.Request.Context(), authUser.User.ID, classID)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbiddenClassAccess):
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		case errors.Is(err, ErrTeacherNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve class roster.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": roster})
}
