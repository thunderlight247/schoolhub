// Package reports HTTP handlers (milestone M18).
package reports

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
	GetEnrollmentReport(ctx context.Context, academicYearID uuid.UUID) (EnrollmentReport, error)
	GetClassAttendanceReport(ctx context.Context, classID, termID uuid.UUID, startDate, endDate string) (ClassAttendanceReport, error)
	GetClassPerformanceReport(ctx context.Context, classID, termID, subjectID uuid.UUID) (ClassPerformanceReport, error)
	GetStudentReportCard(ctx context.Context, studentID, termID uuid.UUID) (StudentReportCard, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

// GetEnrollmentReport handles GET /reports/enrollment
func (h *Handler) GetEnrollmentReport(c *gin.Context) {
	if _, ok := auth.CurrentUser(c); !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	var academicYearID uuid.UUID
	if raw := c.Query("academic_year_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "academic_year_id must be a valid UUID.", nil)
			return
		}
		academicYearID = parsed
	}

	report, err := h.service.GetEnrollmentReport(c.Request.Context(), academicYearID)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate enrollment report.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// GetClassAttendanceReport handles GET /reports/attendance/classes/:id
func (h *Handler) GetClassAttendanceReport(c *gin.Context) {
	if _, ok := auth.CurrentUser(c); !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	classID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a valid UUID.", nil)
		return
	}

	var termID uuid.UUID
	if raw := c.Query("term_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "term_id must be a valid UUID.", nil)
			return
		}
		termID = parsed
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	report, err := h.service.GetClassAttendanceReport(c.Request.Context(), classID, termID, startDate, endDate)
	if err != nil {
		switch {
		case errors.Is(err, ErrClassNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		case errors.Is(err, ErrInvalidDateRange):
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate class attendance report.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// GetClassPerformanceReport handles GET /reports/performance/classes/:id
func (h *Handler) GetClassPerformanceReport(c *gin.Context) {
	if _, ok := auth.CurrentUser(c); !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	classID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a valid UUID.", nil)
		return
	}

	var termID, subjectID uuid.UUID
	if raw := c.Query("term_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "term_id must be a valid UUID.", nil)
			return
		}
		termID = parsed
	}
	if raw := c.Query("subject_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "subject_id must be a valid UUID.", nil)
			return
		}
		subjectID = parsed
	}

	report, err := h.service.GetClassPerformanceReport(c.Request.Context(), classID, termID, subjectID)
	if err != nil {
		if errors.Is(err, ErrClassNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate class performance report.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// GetStudentReportCard handles GET /reports/students/:id/report-card
func (h *Handler) GetStudentReportCard(c *gin.Context) {
	if _, ok := auth.CurrentUser(c); !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a valid UUID.", nil)
		return
	}

	termRaw := c.Query("term_id")
	if termRaw == "" {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "term_id query parameter is required.", nil)
		return
	}
	termID, err := uuid.Parse(termRaw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "term_id must be a valid UUID.", nil)
		return
	}

	reportCard, err := h.service.GetStudentReportCard(c.Request.Context(), studentID, termID)
	if err != nil {
		switch {
		case errors.Is(err, ErrStudentNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		case errors.Is(err, ErrNoActiveEnrollment):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate student report card.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": reportCard})
}
