// Package assessments HTTP handlers (M12). Routes are wired in
// cmd/server/main.go on the dedicated assessment.* and result.* permission
// codes from section 11:
//   - assessment.view: list/get assessments
//   - assessment.create: create assessment header
//   - assessment.update: patch assessment header / delete assessment
//   - result.view: list assessment score sheet / get single result / list student results
//   - result.create: enter/record score sheet
//   - result.update: patch single result
package assessments

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
	CreateAssessment(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error)
	GetAssessment(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error)
	ListAssessments(ctx context.Context, f ListFilter) ([]db.ListAssessmentsRow, int64, error)
	UpdateAssessment(ctx context.Context, actorID, id uuid.UUID, in UpdateAssessmentInput) (db.GetAssessmentByIDRow, error)
	DeleteAssessment(ctx context.Context, actorID, id uuid.UUID) error
	RecordResults(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error)
	ListAssessmentResults(ctx context.Context, assessmentID uuid.UUID) ([]db.ListAssessmentResultsRow, error)
	GetResult(ctx context.Context, id uuid.UUID) (db.GetResultByIDRow, error)
	UpdateResult(ctx context.Context, actorID, id uuid.UUID, in UpdateResultInput) (db.GetResultByIDRow, error)
	ListStudentResults(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) ([]db.ListStudentResultsRow, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

type assessmentDTO struct {
	ID               string   `json:"id"`
	ClassID          string   `json:"class_id"`
	SubjectID        string   `json:"subject_id"`
	TermID           string   `json:"term_id"`
	Title            string   `json:"title"`
	Type             string   `json:"type"`
	MaxScore         float64  `json:"max_score"`
	Weight           *float64 `json:"weight,omitempty"`
	AssessmentDate   string   `json:"assessment_date"`
	ClassName        string   `json:"class_name,omitempty"`
	AcademicYearID   string   `json:"academic_year_id,omitempty"`
	AcademicYearName string   `json:"academic_year_name,omitempty"`
	SubjectCode      string   `json:"subject_code,omitempty"`
	SubjectName      string   `json:"subject_name,omitempty"`
	TermName         string   `json:"term_name,omitempty"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type resultItemDTO struct {
	ID            string   `json:"id"`
	AssessmentID  string   `json:"assessment_id"`
	StudentID     string   `json:"student_id"`
	StudentNumber string   `json:"student_number,omitempty"`
	StudentName   string   `json:"student_name,omitempty"`
	Score         *float64 `json:"score"`
	Grade         *string  `json:"grade"`
	Remarks       *string  `json:"remarks"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type resultDetailDTO struct {
	ID              string   `json:"id"`
	AssessmentID    string   `json:"assessment_id"`
	StudentID       string   `json:"student_id"`
	StudentNumber   string   `json:"student_number,omitempty"`
	StudentName     string   `json:"student_name,omitempty"`
	AssessmentTitle string   `json:"assessment_title"`
	AssessmentType  string   `json:"assessment_type"`
	MaxScore        float64  `json:"max_score"`
	AssessmentDate  string   `json:"assessment_date"`
	ClassID         string   `json:"class_id"`
	ClassName       string   `json:"class_name,omitempty"`
	SubjectCode     string   `json:"subject_code,omitempty"`
	SubjectName     string   `json:"subject_name,omitempty"`
	TermID          string   `json:"term_id"`
	TermName        string   `json:"term_name,omitempty"`
	Score           *float64 `json:"score"`
	Grade           *string  `json:"grade"`
	Remarks         *string  `json:"remarks"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type studentResultDTO struct {
	ID              string   `json:"id"`
	AssessmentID    string   `json:"assessment_id"`
	StudentID       string   `json:"student_id"`
	Score           *float64 `json:"score"`
	Grade           *string  `json:"grade"`
	Remarks         *string  `json:"remarks"`
	AssessmentTitle string   `json:"assessment_title"`
	AssessmentType  string   `json:"assessment_type"`
	MaxScore        float64  `json:"max_score"`
	AssessmentDate  string   `json:"assessment_date"`
	SubjectCode     string   `json:"subject_code,omitempty"`
	SubjectName     string   `json:"subject_name,omitempty"`
	ClassName       string   `json:"class_name,omitempty"`
	TermID          string   `json:"term_id"`
	TermName        string   `json:"term_name,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
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

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func toAssessmentDTO(row db.GetAssessmentByIDRow) assessmentDTO {
	return assessmentDTO{
		ID:               row.ID.String(),
		ClassID:          row.ClassID.String(),
		SubjectID:        row.SubjectID.String(),
		TermID:           row.TermID.String(),
		Title:            row.Title,
		Type:             row.Type,
		MaxScore:         NumericToFloat(row.MaxScore),
		Weight:           NumericToFloatPtr(row.Weight),
		AssessmentDate:   row.AssessmentDate.Time.Format("2006-01-02"),
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

func toAssessmentListItemDTO(row db.ListAssessmentsRow) assessmentDTO {
	return assessmentDTO{
		ID:               row.ID.String(),
		ClassID:          row.ClassID.String(),
		SubjectID:        row.SubjectID.String(),
		TermID:           row.TermID.String(),
		Title:            row.Title,
		Type:             row.Type,
		MaxScore:         NumericToFloat(row.MaxScore),
		Weight:           NumericToFloatPtr(row.Weight),
		AssessmentDate:   row.AssessmentDate.Time.Format("2006-01-02"),
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

func toResultItemDTO(row db.ListAssessmentResultsRow) resultItemDTO {
	return resultItemDTO{
		ID:            row.ID.String(),
		AssessmentID:  row.AssessmentID.String(),
		StudentID:     row.StudentID.String(),
		StudentNumber: row.StudentNumber,
		StudentName:   valueString(row.StudentName),
		Score:         NumericToFloatPtr(row.Score),
		Grade:         textPtr(row.Grade),
		Remarks:       textPtr(row.Remarks),
		CreatedAt:     row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:     row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toResultDetailDTO(row db.GetResultByIDRow) resultDetailDTO {
	return resultDetailDTO{
		ID:              row.ID.String(),
		AssessmentID:    row.AssessmentID.String(),
		StudentID:       row.StudentID.String(),
		StudentNumber:   row.StudentNumber,
		StudentName:     valueString(row.StudentName),
		AssessmentTitle: row.AssessmentTitle,
		AssessmentType:  row.AssessmentType,
		MaxScore:        NumericToFloat(row.MaxScore),
		AssessmentDate:  row.AssessmentDate.Time.Format("2006-01-02"),
		ClassID:         row.ClassID.String(),
		ClassName:       row.ClassName,
		SubjectCode:     row.SubjectCode,
		SubjectName:     row.SubjectName,
		TermID:          row.TermID.String(),
		TermName:        row.TermName,
		Score:           NumericToFloatPtr(row.Score),
		Grade:           textPtr(row.Grade),
		Remarks:         textPtr(row.Remarks),
		CreatedAt:       row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toStudentResultDTO(row db.ListStudentResultsRow) studentResultDTO {
	return studentResultDTO{
		ID:              row.ID.String(),
		AssessmentID:    row.AssessmentID.String(),
		StudentID:       row.StudentID.String(),
		Score:           NumericToFloatPtr(row.Score),
		Grade:           textPtr(row.Grade),
		Remarks:         textPtr(row.Remarks),
		AssessmentTitle: row.AssessmentTitle,
		AssessmentType:  row.AssessmentType,
		MaxScore:        NumericToFloat(row.MaxScore),
		AssessmentDate:  row.AssessmentDate.Time.Format("2006-01-02"),
		SubjectCode:     row.SubjectCode,
		SubjectName:     row.SubjectName,
		ClassName:       row.ClassName,
		TermID:          row.TermID.String(),
		TermName:        row.TermName,
		CreatedAt:       row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Time.Format(time.RFC3339),
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

func (h *Handler) ListAssessments(c *gin.Context) {
	filter := ListFilter{
		Type:     strings.TrimSpace(c.Query("type")),
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

	rows, total, err := h.service.ListAssessments(c.Request.Context(), filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]assessmentDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAssessmentListItemDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{
		"assessments": items,
		"pagination": gin.H{
			"page":      filter.Page,
			"page_size": filter.PageSize,
			"total":     total,
		},
	})
}

func (h *Handler) GetAssessment(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	row, err := h.service.GetAssessment(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"assessment": toAssessmentDTO(row)})
}

func (h *Handler) CreateAssessment(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var in CreateAssessmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid assessment payload.", nil)
		return
	}

	if in.ClassID == uuid.Nil || in.SubjectID == uuid.Nil || in.TermID == uuid.Nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "class_id, subject_id, and term_id are required.", nil)
		return
	}

	row, err := h.service.CreateAssessment(c.Request.Context(), actorID, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"assessment": toAssessmentDTO(row)})
}

func (h *Handler) UpdateAssessment(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var in UpdateAssessmentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid update payload.", nil)
		return
	}

	row, err := h.service.UpdateAssessment(c.Request.Context(), actorID, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"assessment": toAssessmentDTO(row)})
}

func (h *Handler) DeleteAssessment(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteAssessment(c.Request.Context(), actorID, id); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assessment deleted successfully."})
}

func (h *Handler) ListAssessmentResults(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	rows, err := h.service.ListAssessmentResults(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]resultItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toResultItemDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{"results": items})
}

func (h *Handler) RecordResults(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var in struct {
		Results []ResultInput `json:"results"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid results payload.", nil)
		return
	}

	rows, err := h.service.RecordResults(c.Request.Context(), actorID, id, in.Results)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]resultItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toResultItemDTO(row))
	}

	c.JSON(http.StatusCreated, gin.H{"results": items})
}

func (h *Handler) GetResult(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	row, err := h.service.GetResult(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": toResultDetailDTO(row)})
}

func (h *Handler) UpdateResult(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var in UpdateResultInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid result update payload.", nil)
		return
	}

	row, err := h.service.UpdateResult(c.Request.Context(), actorID, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": toResultDetailDTO(row)})
}

func (h *Handler) ListStudentResults(c *gin.Context) {
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

	rows, err := h.service.ListStudentResults(c.Request.Context(), id, termID, yearID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	items := make([]studentResultDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toStudentResultDTO(row))
	}

	c.JSON(http.StatusOK, gin.H{"results": items})
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAssessmentNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Assessment not found.", nil)
	case errors.Is(err, ErrResultNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Result not found.", nil)
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
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Assessment date falls outside the term's start and end dates.", nil)
	case errors.Is(err, ErrTitleRequired):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required.", nil)
	case errors.Is(err, ErrInvalidType):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid assessment type. Allowed types: QUIZ, TEST, ASSIGNMENT, PROJECT, MIDTERM, FINAL.", nil)
	case errors.Is(err, ErrInvalidMaxScore):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "max_score must be greater than zero.", nil)
	case errors.Is(err, ErrInvalidWeight):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "weight must be between 0 and 100.", nil)
	case errors.Is(err, ErrDuplicateAssessment):
		apierror.Respond(c, http.StatusConflict, "ALREADY_EXISTS", "An assessment with this title already exists for this class, subject, and term.", nil)
	case errors.Is(err, ErrStudentNotEnrolled):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "One or more students are not enrolled in this class.", nil)
	case errors.Is(err, ErrDuplicateStudent):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "A student appears more than once in the score sheet.", nil)
	case errors.Is(err, ErrNoResults):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "At least one result is required.", nil)
	case errors.Is(err, ErrInvalidScore):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Score must be between 0 and the assessment's max_score.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the assessment operation.", nil)
	}
}
