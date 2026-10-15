// Package studentportal HTTP handlers (milestone M17).
package studentportal

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// Default and maximum attendance rows returned by the attendance endpoint;
// callers can lower the limit via ?limit= but never exceed the cap.
const (
	defaultAttendanceLimit = 50
	maxAttendanceLimit     = 200
)

type ServiceAPI interface {
	GetStudentOverview(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error)
	ListStudentAttendance(ctx context.Context, userID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error)
	ListStudentResultsHistory(ctx context.Context, userID uuid.UUID, termID, academicYearID uuid.UUID) ([]StudentResult, error)
	ListStudentEnrollments(ctx context.Context, userID uuid.UUID) ([]StudentEnrollmentHistoryItem, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetStudentOverview(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	data, err := h.service.GetStudentOverview(c.Request.Context(), authUser.User.ID)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve student portal overview.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetStudentAttendance(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	limit := int32(defaultAttendanceLimit)
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "limit must be a positive integer.", nil)
			return
		}
		if parsed > maxAttendanceLimit {
			parsed = maxAttendanceLimit
		}
		limit = int32(parsed)
	}

	records, err := h.service.ListStudentAttendance(c.Request.Context(), authUser.User.ID, limit)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve attendance records.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records})
}

func (h *Handler) GetStudentResultsHistory(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	// Optional scoping filters; both default to the full history.
	var termID, academicYearID uuid.UUID
	if raw := c.Query("term_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "term_id must be a valid UUID.", nil)
			return
		}
		termID = parsed
	}
	if raw := c.Query("academic_year_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "academic_year_id must be a valid UUID.", nil)
			return
		}
		academicYearID = parsed
	}

	results, err := h.service.ListStudentResultsHistory(c.Request.Context(), authUser.User.ID, termID, academicYearID)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve results history.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

func (h *Handler) GetStudentEnrollments(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	enrollments, err := h.service.ListStudentEnrollments(c.Request.Context(), authUser.User.ID)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve enrollment history.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": enrollments})
}
