package enrollment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type ServiceAPI interface {
	CreateEnrollment(ctx context.Context, actorID uuid.UUID, in CreateEnrollmentInput) (db.Enrollment, error)
	GetEnrollment(ctx context.Context, id uuid.UUID) (db.GetEnrollmentByIDRow, error)
	ListEnrollments(ctx context.Context, f ListFilter) ([]db.ListEnrollmentsRow, int64, error)
	UpdateEnrollmentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (db.GetEnrollmentByIDRow, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

type enrollmentDTO struct {
	ID               string  `json:"id"`
	StudentID        string  `json:"student_id"`
	AcademicYearID   string  `json:"academic_year_id"`
	ClassID          string  `json:"class_id"`
	Status           string  `json:"status"`
	EnrollmentDate   string  `json:"enrollment_date"`
	ExitDate         *string `json:"exit_date,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	StudentName      string  `json:"student_name,omitempty"`
	AcademicYearName string  `json:"academic_year_name,omitempty"`
	ClassName        string  `json:"class_name,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type enrollmentListItemDTO struct {
	ID               string  `json:"id"`
	StudentID        string  `json:"student_id"`
	AcademicYearID   string  `json:"academic_year_id"`
	ClassID          string  `json:"class_id"`
	Status           string  `json:"status"`
	EnrollmentDate   string  `json:"enrollment_date"`
	ExitDate         *string `json:"exit_date,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	StudentNumber    string  `json:"student_number,omitempty"`
	StudentName      string  `json:"student_name,omitempty"`
	AcademicYearName string  `json:"academic_year_name,omitempty"`
	ClassName        string  `json:"class_name,omitempty"`
}

func valueString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}

func toEnrollmentDTO(row db.GetEnrollmentByIDRow) enrollmentDTO {
	var exitDate *string
	if row.ExitDate.Valid {
		s := row.ExitDate.Time.Format("2006-01-02")
		exitDate = &s
	}
	var notes *string
	if row.Notes.Valid {
		s := row.Notes.String
		notes = &s
	}
	return enrollmentDTO{
		ID:               row.ID.String(),
		StudentID:        row.StudentID.String(),
		AcademicYearID:   row.AcademicYearID.String(),
		ClassID:          row.ClassID.String(),
		Status:           row.Status,
		EnrollmentDate:   row.EnrollmentDate.Time.Format("2006-01-02"),
		ExitDate:         exitDate,
		Notes:            notes,
		StudentName:      valueString(row.StudentName),
		AcademicYearName: valueString(row.AcademicYearName),
		ClassName:        row.ClassName,
		CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:        row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toEnrollmentListItemDTO(row db.ListEnrollmentsRow) enrollmentListItemDTO {
	var exitDate *string
	if row.ExitDate.Valid {
		s := row.ExitDate.Time.Format("2006-01-02")
		exitDate = &s
	}
	var notes *string
	if row.Notes.Valid {
		s := row.Notes.String
		notes = &s
	}
	return enrollmentListItemDTO{
		ID:               row.ID.String(),
		StudentID:        row.StudentID.String(),
		AcademicYearID:   row.AcademicYearID.String(),
		ClassID:          row.ClassID.String(),
		Status:           row.Status,
		EnrollmentDate:   row.EnrollmentDate.Time.Format("2006-01-02"),
		ExitDate:         exitDate,
		Notes:            notes,
		StudentNumber:    row.StudentNumber,
		StudentName:      valueString(row.StudentName),
		AcademicYearName: valueString(row.AcademicYearName),
		ClassName:        row.ClassName,
	}
}

func parseUUIDFromQuery(c *gin.Context, key string) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a UUID.", nil)
		return nil, false
	}
	return &id, true
}

func parsePageParam(c *gin.Context, key string, fallback int) int {
	value := c.DefaultQuery(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func requireActor(c *gin.Context) (uuid.UUID, bool) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return uuid.Nil, false
	}
	return authUser.User.ID, true
}

func (h *Handler) ListEnrollments(c *gin.Context) {
	filter := ListFilter{
		Page:     parsePageParam(c, "page", 1),
		PageSize: parsePageParam(c, "page_size", 20),
	}
	if v, ok := parseUUIDFromQuery(c, "student_id"); ok && v != nil {
		filter.StudentID = v
	}
	// Nested route (/students/:id/enrollments): the path's student wins over
	// the query param. On /enrollments this param is empty and the query
	// value (if any) stands.
	if raw := c.Param("id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "student_id must be a UUID.", nil)
			return
		}
		filter.StudentID = &id
	}
	if v, ok := parseUUIDFromQuery(c, "academic_year_id"); ok && v != nil {
		filter.AcademicYearID = v
	}
	if v, ok := parseUUIDFromQuery(c, "class_id"); ok && v != nil {
		filter.ClassID = v
	}
	filter.Status = strings.TrimSpace(c.Query("status"))
	filter.Search = strings.TrimSpace(c.Query("search"))

	rows, total, err := h.service.ListEnrollments(c.Request.Context(), filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]enrollmentListItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toEnrollmentListItemDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{
		"enrollments": items,
		"pagination": gin.H{
			"page":      filter.Page,
			"page_size": filter.PageSize,
			"total":     total,
		},
	})
}

func (h *Handler) GetEnrollment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	row, err := h.service.GetEnrollment(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"enrollment": toEnrollmentDTO(row)})
}

func parseUUIDParam(c *gin.Context, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(key))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a UUID.", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) CreateEnrollment(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var in CreateEnrollmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid enrollment payload.", nil)
		return
	}
	// Nested route (/students/:id/enrollments): the path's student wins over
	// the body, so the URL stays authoritative.
	if raw := c.Param("id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "student_id must be a UUID.", nil)
			return
		}
		in.StudentID = id
	}
	if in.StudentID == uuid.Nil || in.AcademicYearID == uuid.Nil || in.ClassID == uuid.Nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "student_id, academic_year_id, and class_id are required.", nil)
		return
	}

	row, err := h.service.CreateEnrollment(c.Request.Context(), actorID, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"enrollment": toEnrollmentDTOFromEnrollment(row)})
}

func toEnrollmentDTOFromEnrollment(row db.Enrollment) enrollmentDTO {
	var exitDate *string
	if row.ExitDate.Valid {
		s := row.ExitDate.Time.Format("2006-01-02")
		exitDate = &s
	}
	var notes *string
	if row.Notes.Valid {
		s := row.Notes.String
		notes = &s
	}
	return enrollmentDTO{
		ID:             row.ID.String(),
		StudentID:      row.StudentID.String(),
		AcademicYearID: row.AcademicYearID.String(),
		ClassID:        row.ClassID.String(),
		Status:         row.Status,
		EnrollmentDate: row.EnrollmentDate.Time.Format("2006-01-02"),
		ExitDate:       exitDate,
		Notes:          notes,
		CreatedAt:      row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func (h *Handler) UpdateEnrollmentStatus(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid status payload.", nil)
		return
	}
	if strings.TrimSpace(payload.Status) == "" {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status is required.", nil)
		return
	}

	row, err := h.service.UpdateEnrollmentStatus(c.Request.Context(), actorID, id, payload.Status)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"enrollment": toEnrollmentDTO(row)})
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEnrollmentNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Enrollment not found.", nil)
	case errors.Is(err, ErrStudentNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Student not found.", nil)
	case errors.Is(err, ErrAcademicYearNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Academic year not found.", nil)
	case errors.Is(err, ErrClassNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Class not found.", nil)
	case errors.Is(err, ErrClassYearMismatch):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Class does not belong to the selected academic year.", nil)
	case errors.Is(err, ErrInvalidEnrollmentStatus):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Status must be one of ENROLLED, COMPLETED, TRANSFERRED, WITHDRAWN.", nil)
	case errors.Is(err, ErrDuplicateActiveEnrollment):
		apierror.Respond(c, http.StatusConflict, "ALREADY_ENROLLED", "Student already has an active enrollment in this academic year.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the enrollment operation.", nil)
	}
}

func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}
