// Package assignments HTTP handlers (M10). Routes are wired in
// cmd/server/main.go: reads under class.view, create/remove under
// class.update (same gating as the class-subject mappings in M8, which
// this domain sits next to).
package assignments

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

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type ServiceAPI interface {
	CreateTeacherAssignment(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error)
	GetTeacherAssignment(ctx context.Context, id uuid.UUID) (db.GetTeacherAssignmentByIDRow, error)
	ListTeacherAssignments(ctx context.Context, f ListFilter) ([]db.ListTeacherAssignmentsRow, int64, error)
	DeleteTeacherAssignment(ctx context.Context, actorID, id uuid.UUID) error
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

type teacherAssignmentDTO struct {
	ID               string `json:"id"`
	TeacherID        string `json:"teacher_id"`
	ClassID          string `json:"class_id"`
	SubjectID        string `json:"subject_id"`
	StaffNumber      string `json:"staff_number,omitempty"`
	TeacherName      string `json:"teacher_name,omitempty"`
	AcademicYearID   string `json:"academic_year_id,omitempty"`
	AcademicYearName string `json:"academic_year_name,omitempty"`
	ClassName        string `json:"class_name,omitempty"`
	SubjectCode      string `json:"subject_code,omitempty"`
	SubjectName      string `json:"subject_name,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type teacherAssignmentListItemDTO struct {
	ID               string `json:"id"`
	TeacherID        string `json:"teacher_id"`
	ClassID          string `json:"class_id"`
	SubjectID        string `json:"subject_id"`
	StaffNumber      string `json:"staff_number,omitempty"`
	TeacherName      string `json:"teacher_name,omitempty"`
	AcademicYearID   string `json:"academic_year_id,omitempty"`
	AcademicYearName string `json:"academic_year_name,omitempty"`
	ClassName        string `json:"class_name,omitempty"`
	SubjectCode      string `json:"subject_code,omitempty"`
	SubjectName      string `json:"subject_name,omitempty"`
}

// valueString renders sqlc's interface{} columns (the teacher_name
// concatenation — sqlc cannot infer its nullability). Same approach as the
// enrollment handlers use for student_name.
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

func toTeacherAssignmentDTO(row db.GetTeacherAssignmentByIDRow) teacherAssignmentDTO {
	return teacherAssignmentDTO{
		ID:               row.ID.String(),
		TeacherID:        row.TeacherID.String(),
		ClassID:          row.ClassID.String(),
		SubjectID:        row.SubjectID.String(),
		StaffNumber:      row.StaffNumber,
		TeacherName:      valueString(row.TeacherName),
		AcademicYearID:   row.AcademicYearID.String(),
		AcademicYearName: row.AcademicYearName,
		ClassName:        row.ClassName,
		SubjectCode:      row.SubjectCode,
		SubjectName:      row.SubjectName,
		CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:        row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toTeacherAssignmentListItemDTO(row db.ListTeacherAssignmentsRow) teacherAssignmentListItemDTO {
	return teacherAssignmentListItemDTO{
		ID:               row.ID.String(),
		TeacherID:        row.TeacherID.String(),
		ClassID:          row.ClassID.String(),
		SubjectID:        row.SubjectID.String(),
		StaffNumber:      row.StaffNumber,
		TeacherName:      valueString(row.TeacherName),
		AcademicYearID:   row.AcademicYearID.String(),
		AcademicYearName: row.AcademicYearName,
		ClassName:        row.ClassName,
		SubjectCode:      row.SubjectCode,
		SubjectName:      row.SubjectName,
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

func parseUUIDParam(c *gin.Context, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(key))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a UUID.", nil)
		return uuid.Nil, false
	}
	return id, true
}

func requireActor(c *gin.Context) (uuid.UUID, bool) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return uuid.Nil, false
	}
	return authUser.User.ID, true
}

// pathTeacherID returns the :id path segment when the handler is serving a
// nested route (/teachers/:id/assignments); it returns ok=false (with an
// error already written) when the segment is present but not a UUID.
func pathTeacherID(c *gin.Context) (*uuid.UUID, bool) {
	raw := c.Param("id")
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "teacher_id must be a UUID.", nil)
		return nil, false
	}
	return &id, true
}

func (h *Handler) ListTeacherAssignments(c *gin.Context) {
	filter := ListFilter{
		Page:     parsePageParam(c, "page", 1),
		PageSize: parsePageParam(c, "page_size", 20),
	}

	// Nested route: the path's teacher wins over the query param.
	teacherID, ok := pathTeacherID(c)
	if !ok {
		return
	}
	if teacherID != nil {
		filter.TeacherID = teacherID
	} else if v, ok := parseUUIDFromQuery(c, "teacher_id"); ok && v != nil {
		filter.TeacherID = v
	}

	if v, ok := parseUUIDFromQuery(c, "class_id"); ok && v != nil {
		filter.ClassID = v
	}
	if v, ok := parseUUIDFromQuery(c, "subject_id"); ok && v != nil {
		filter.SubjectID = v
	}
	if v, ok := parseUUIDFromQuery(c, "academic_year_id"); ok && v != nil {
		filter.AcademicYearID = v
	}
	filter.Search = strings.TrimSpace(c.Query("search"))

	rows, total, err := h.service.ListTeacherAssignments(c.Request.Context(), filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]teacherAssignmentListItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toTeacherAssignmentListItemDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{
		"assignments": items,
		"pagination": gin.H{
			"page":      filter.Page,
			"page_size": filter.PageSize,
			"total":     total,
		},
	})
}

func (h *Handler) GetTeacherAssignment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	row, err := h.service.GetTeacherAssignment(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignment": toTeacherAssignmentDTO(row)})
}

func (h *Handler) CreateTeacherAssignment(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var in CreateTeacherAssignmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid teacher assignment payload.", nil)
		return
	}

	// Nested route (/teachers/:id/assignments) carries the teacher in the
	// path; it wins over the body so the URL stays authoritative.
	if teacherID, ok := pathTeacherID(c); !ok {
		return
	} else if teacherID != nil {
		in.TeacherID = *teacherID
	}

	if in.TeacherID == uuid.Nil || in.ClassID == uuid.Nil || in.SubjectID == uuid.Nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "teacher_id, class_id, and subject_id are required.", nil)
		return
	}

	row, err := h.service.CreateTeacherAssignment(c.Request.Context(), actorID, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"assignment": toTeacherAssignmentDTO(row)})
}

func (h *Handler) DeleteTeacherAssignment(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteTeacherAssignment(c.Request.Context(), actorID, id); err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAssignmentNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Teacher assignment not found.", nil)
	case errors.Is(err, ErrTeacherNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Teacher (staff record) not found.", nil)
	case errors.Is(err, ErrClassNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Class not found.", nil)
	case errors.Is(err, ErrSubjectNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Subject not found.", nil)
	case errors.Is(err, ErrSubjectNotInClass):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Subject is not assigned to this class; assign it to the class first.", nil)
	case errors.Is(err, ErrDuplicateAssignment):
		apierror.Respond(c, http.StatusConflict, "ALREADY_ASSIGNED", "This teacher already has an assignment for this class and subject.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the teacher assignment operation.", nil)
	}
}
