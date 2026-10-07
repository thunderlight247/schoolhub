// Package attendance HTTP handlers (M11). Routes are wired in
// cmd/server/main.go on the dedicated attendance.* permission codes from
// section 11: reads under attendance.view (PARENT/STUDENT hold it too for
// the portals), register capture under attendance.record, mark corrections
// under attendance.update.
package attendance

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
	CreateSession(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error)
	GetSession(ctx context.Context, id uuid.UUID) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error)
	ListSessions(ctx context.Context, f ListFilter) ([]db.ListAttendanceSessionsRow, int64, error)
	UpdateRecord(ctx context.Context, actorID, id uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error)
	StudentSummary(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) (Summary, error)
	ClassSummary(ctx context.Context, classID uuid.UUID, termID, subjectID *uuid.UUID) (Summary, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

type attendanceSessionDTO struct {
	ID               string `json:"id"`
	ClassID          string `json:"class_id"`
	SubjectID        string `json:"subject_id"`
	TermID           string `json:"term_id"`
	SessionDate      string `json:"session_date"`
	ClassName        string `json:"class_name,omitempty"`
	AcademicYearID   string `json:"academic_year_id,omitempty"`
	AcademicYearName string `json:"academic_year_name,omitempty"`
	SubjectCode      string `json:"subject_code,omitempty"`
	SubjectName      string `json:"subject_name,omitempty"`
	TermName         string `json:"term_name,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type attendanceRecordDTO struct {
	ID            string  `json:"id"`
	SessionID     string  `json:"session_id"`
	StudentID     string  `json:"student_id"`
	StudentNumber string  `json:"student_number,omitempty"`
	StudentName   string  `json:"student_name,omitempty"`
	Status        string  `json:"status"`
	Notes         *string `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type attendanceRecordDetailDTO struct {
	ID            string  `json:"id"`
	SessionID     string  `json:"session_id"`
	StudentID     string  `json:"student_id"`
	StudentNumber string  `json:"student_number,omitempty"`
	StudentName   string  `json:"student_name,omitempty"`
	Status        string  `json:"status"`
	Notes         *string `json:"notes,omitempty"`
	SessionDate   string  `json:"session_date"`
	ClassID       string  `json:"class_id"`
	ClassName     string  `json:"class_name,omitempty"`
	SubjectCode   string  `json:"subject_code,omitempty"`
	SubjectName   string  `json:"subject_name,omitempty"`
	TermID        string  `json:"term_id"`
	TermName      string  `json:"term_name,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type summaryDTO struct {
	StudentID      string           `json:"student_id,omitempty"`
	ClassID        string           `json:"class_id,omitempty"`
	SubjectID      string           `json:"subject_id,omitempty"`
	TermID         string           `json:"term_id,omitempty"`
	AcademicYearID string           `json:"academic_year_id,omitempty"`
	Counts         map[string]int64 `json:"counts"`
	TotalRecords   int64            `json:"total_records"`
}

// valueString renders sqlc's interface{} columns (the student_name
// concatenation — sqlc cannot infer its nullability). Same approach as the
// enrollment and teacher-assignment handlers.
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

func notesPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func toSessionDTO(row db.GetAttendanceSessionByIDRow) attendanceSessionDTO {
	return attendanceSessionDTO{
		ID:               row.ID.String(),
		ClassID:          row.ClassID.String(),
		SubjectID:        row.SubjectID.String(),
		TermID:           row.TermID.String(),
		SessionDate:      row.SessionDate.Time.Format("2006-01-02"),
		ClassName:        row.ClassName,
		AcademicYearID:   row.AcademicYearID.String(),
		AcademicYearName: row.AcademicYearName,
		SubjectCode:      row.SubjectCode,
		SubjectName:      row.SubjectName,
		TermName:         row.TermName,
		CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:        row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toSessionListItemDTO(row db.ListAttendanceSessionsRow) attendanceSessionDTO {
	return attendanceSessionDTO{
		ID:               row.ID.String(),
		ClassID:          row.ClassID.String(),
		SubjectID:        row.SubjectID.String(),
		TermID:           row.TermID.String(),
		SessionDate:      row.SessionDate.Time.Format("2006-01-02"),
		ClassName:        row.ClassName,
		AcademicYearID:   row.AcademicYearID.String(),
		AcademicYearName: row.AcademicYearName,
		SubjectCode:      row.SubjectCode,
		SubjectName:      row.SubjectName,
		TermName:         row.TermName,
		CreatedAt:        row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:        row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toRecordDTO(row db.ListAttendanceSessionRecordsRow) attendanceRecordDTO {
	return attendanceRecordDTO{
		ID:            row.ID.String(),
		SessionID:     row.SessionID.String(),
		StudentID:     row.StudentID.String(),
		StudentNumber: row.StudentNumber,
		StudentName:   valueString(row.StudentName),
		Status:        row.Status,
		Notes:         notesPtr(row.Notes),
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toRecordDetailDTO(row db.GetAttendanceRecordByIDRow) attendanceRecordDetailDTO {
	return attendanceRecordDetailDTO{
		ID:            row.ID.String(),
		SessionID:     row.SessionID.String(),
		StudentID:     row.StudentID.String(),
		StudentNumber: row.StudentNumber,
		StudentName:   valueString(row.StudentName),
		Status:        row.Status,
		Notes:         notesPtr(row.Notes),
		SessionDate:   row.SessionDate.Time.Format("2006-01-02"),
		ClassID:       row.ClassID.String(),
		ClassName:     row.ClassName,
		SubjectCode:   row.SubjectCode,
		SubjectName:   row.SubjectName,
		TermID:        row.TermID.String(),
		TermName:      row.TermName,
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toSummaryDTO(sum Summary) summaryDTO {
	dto := summaryDTO{Counts: sum.Counts, TotalRecords: sum.TotalRecords}
	if sum.StudentID != nil {
		dto.StudentID = sum.StudentID.String()
	}
	if sum.ClassID != nil {
		dto.ClassID = sum.ClassID.String()
	}
	if sum.SubjectID != nil {
		dto.SubjectID = sum.SubjectID.String()
	}
	if sum.TermID != nil {
		dto.TermID = sum.TermID.String()
	}
	if sum.AcademicYearID != nil {
		dto.AcademicYearID = sum.AcademicYearID.String()
	}
	return dto
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

func parseDateFromQuery(c *gin.Context, key string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a date (YYYY-MM-DD).", nil)
		return nil, false
	}
	return &parsed, true
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

func (h *Handler) ListSessions(c *gin.Context) {
	filter := ListFilter{
		Page:     parsePageParam(c, "page", 1),
		PageSize: parsePageParam(c, "page_size", 20),
	}

	if v, ok := parseUUIDFromQuery(c, "class_id"); ok && v != nil {
		filter.ClassID = v
	}
	if v, ok := parseUUIDFromQuery(c, "subject_id"); ok && v != nil {
		filter.SubjectID = v
	}
	if v, ok := parseUUIDFromQuery(c, "term_id"); ok && v != nil {
		filter.TermID = v
	}
	if v, ok := parseDateFromQuery(c, "date_from"); ok {
		filter.DateFrom = v
	}
	if v, ok := parseDateFromQuery(c, "date_to"); ok {
		filter.DateTo = v
	}

	rows, total, err := h.service.ListSessions(c.Request.Context(), filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]attendanceSessionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toSessionListItemDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": items,
		"pagination": gin.H{
			"page":      filter.Page,
			"page_size": filter.PageSize,
			"total":     total,
		},
	})
}

func (h *Handler) GetSession(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	sessionRow, records, err := h.service.GetSession(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]attendanceRecordDTO, 0, len(records))
	for _, row := range records {
		items = append(items, toRecordDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{
		"session": toSessionDTO(sessionRow),
		"records": items,
	})
}

func (h *Handler) CreateSession(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var in CreateSessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid attendance capture payload.", nil)
		return
	}

	if in.ClassID == uuid.Nil || in.SubjectID == uuid.Nil || in.TermID == uuid.Nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "class_id, subject_id, and term_id are required.", nil)
		return
	}

	sessionRow, records, err := h.service.CreateSession(c.Request.Context(), actorID, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]attendanceRecordDTO, 0, len(records))
	for _, row := range records {
		items = append(items, toRecordDTO(row))
	}

	c.JSON(http.StatusCreated, gin.H{
		"session": toSessionDTO(sessionRow),
		"records": items,
	})
}

func (h *Handler) UpdateRecord(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var in struct {
		Status string  `json:"status"`
		Notes  *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid attendance record payload.", nil)
		return
	}

	row, err := h.service.UpdateRecord(c.Request.Context(), actorID, id, in.Status, in.Notes)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": toRecordDetailDTO(row)})
}

func (h *Handler) StudentSummary(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var termID, yearID *uuid.UUID
	if v, ok := parseUUIDFromQuery(c, "term_id"); ok && v != nil {
		termID = v
	}
	if v, ok := parseUUIDFromQuery(c, "academic_year_id"); ok && v != nil {
		yearID = v
	}

	sum, err := h.service.StudentSummary(c.Request.Context(), id, termID, yearID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": toSummaryDTO(sum)})
}

func (h *Handler) ClassSummary(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var termID, subjectID *uuid.UUID
	if v, ok := parseUUIDFromQuery(c, "term_id"); ok && v != nil {
		termID = v
	}
	if v, ok := parseUUIDFromQuery(c, "subject_id"); ok && v != nil {
		subjectID = v
	}

	sum, err := h.service.ClassSummary(c.Request.Context(), id, termID, subjectID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": toSummaryDTO(sum)})
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrSessionNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Attendance session not found.", nil)
	case errors.Is(err, ErrRecordNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Attendance record not found.", nil)
	case errors.Is(err, ErrClassNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Class not found.", nil)
	case errors.Is(err, ErrSubjectNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Subject not found.", nil)
	case errors.Is(err, ErrTermNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Term not found.", nil)
	case errors.Is(err, ErrStudentNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Student not found.", nil)
	case errors.Is(err, ErrSubjectNotInClass):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Subject is not assigned to this class; assign it to the class first.", nil)
	case errors.Is(err, ErrTermYearMismatch):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Term does not belong to the class's academic year.", nil)
	case errors.Is(err, ErrDateOutsideTerm):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Session date falls outside the term's start and end dates.", nil)
	case errors.Is(err, ErrInvalidStatus):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status must be one of PRESENT, ABSENT, LATE, EXCUSED.", nil)
	case errors.Is(err, ErrDuplicateSession):
		apierror.Respond(c, http.StatusConflict, "ALREADY_RECORDED", "Attendance has already been captured for this class, subject, term, and date.", nil)
	case errors.Is(err, ErrStudentNotEnrolled):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "One or more students are not enrolled in this class.", nil)
	case errors.Is(err, ErrDuplicateStudent):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "A student appears more than once in the register.", nil)
	case errors.Is(err, ErrNoRecords):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "At least one attendance record is required.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the attendance operation.", nil)
	}
}
