package academics

import (
	"context"
	"errors"
	"net/http"
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
	CreateAcademicYear(ctx context.Context, actorID uuid.UUID, in CreateAcademicYearInput) (db.AcademicYear, error)
	GetAcademicYear(ctx context.Context, id uuid.UUID) (db.AcademicYear, error)
	GetCurrentAcademicYear(ctx context.Context) (db.AcademicYear, error)
	ListAcademicYears(ctx context.Context) ([]db.AcademicYear, error)
	UpdateAcademicYear(ctx context.Context, actorID, id uuid.UUID, in UpdateAcademicYearInput) (db.AcademicYear, error)
	SetCurrentAcademicYear(ctx context.Context, actorID, id uuid.UUID) (db.AcademicYear, error)

	CreateTerm(ctx context.Context, actorID uuid.UUID, in CreateTermInput) (db.Term, error)
	GetTerm(ctx context.Context, id uuid.UUID) (db.GetTermByIDRow, error)
	GetCurrentTerm(ctx context.Context) (db.GetCurrentTermRow, error)
	ListTerms(ctx context.Context, academicYearID *uuid.UUID) ([]db.ListAllTermsRow, error)
	UpdateTerm(ctx context.Context, actorID, id uuid.UUID, in UpdateTermInput) (db.Term, error)
	SetCurrentTerm(ctx context.Context, actorID, id uuid.UUID) (db.Term, error)

	CreateProgramme(ctx context.Context, actorID uuid.UUID, in CreateProgrammeInput) (db.Programme, error)
	GetProgramme(ctx context.Context, id uuid.UUID) (db.Programme, error)
	ListProgrammes(ctx context.Context) ([]db.Programme, error)
	UpdateProgramme(ctx context.Context, actorID, id uuid.UUID, in UpdateProgrammeInput) (db.Programme, error)

	CreateSubject(ctx context.Context, actorID uuid.UUID, in CreateSubjectInput) (db.Subject, error)
	GetSubject(ctx context.Context, id uuid.UUID) (db.GetSubjectByIDRow, error)
	ListSubjects(ctx context.Context, f SubjectListFilter) ([]db.ListSubjectsRow, error)
	UpdateSubject(ctx context.Context, actorID, id uuid.UUID, in UpdateSubjectInput) (db.Subject, error)

	CreateClass(ctx context.Context, actorID uuid.UUID, in CreateClassInput) (db.Class, error)
	GetClass(ctx context.Context, id uuid.UUID) (db.GetClassByIDRow, error)
	ListClasses(ctx context.Context, f ClassListFilter) ([]db.ListClassesRow, error)
	UpdateClass(ctx context.Context, actorID, id uuid.UUID, in UpdateClassInput) (db.Class, error)
	AssignSubjectToClass(ctx context.Context, actorID, classID, subjectID uuid.UUID) (db.ClassSubject, error)
	RemoveSubjectFromClass(ctx context.Context, actorID, classID, subjectID uuid.UUID) error
	ListSubjectsForClass(ctx context.Context, classID uuid.UUID) ([]db.ListSubjectsForClassRow, error)
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

// Helpers

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func tsString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func uuidPtrString(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes).String()
	return &id
}

func int4Ptr(i pgtype.Int4) *int32 {
	if !i.Valid {
		return nil
	}
	v := i.Int32
	return &v
}

func parseDateParam(c *gin.Context, raw, field string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", field+" must be a date in YYYY-MM-DD format.", nil)
		return time.Time{}, false
	}
	return t, true
}

func parseUUIDParam(c *gin.Context, raw, field string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", field+" must be a UUID.", nil)
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

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAcademicYearNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Academic year not found.", nil)
	case errors.Is(err, ErrAcademicYearNameTaken):
		apierror.Respond(c, http.StatusConflict, "NAME_TAKEN", "An academic year with that name already exists.", nil)
	case errors.Is(err, ErrInvalidDateRange):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "End date must be greater than or equal to start date.", nil)
	case errors.Is(err, ErrTermNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Term not found.", nil)
	case errors.Is(err, ErrTermNameTaken):
		apierror.Respond(c, http.StatusConflict, "NAME_TAKEN", "A term with that name already exists in this academic year.", nil)
	case errors.Is(err, ErrProgrammeNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Programme not found.", nil)
	case errors.Is(err, ErrProgrammeCodeTaken):
		apierror.Respond(c, http.StatusConflict, "CODE_TAKEN", "A programme with that code already exists.", nil)
	case errors.Is(err, ErrProgrammeNameTaken):
		apierror.Respond(c, http.StatusConflict, "NAME_TAKEN", "A programme with that name already exists.", nil)
	case errors.Is(err, ErrSubjectNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Subject not found.", nil)
	case errors.Is(err, ErrSubjectCodeTaken):
		apierror.Respond(c, http.StatusConflict, "CODE_TAKEN", "A subject with that code already exists.", nil)
	case errors.Is(err, ErrInvalidSubjectType):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Subject type must be CORE or ELECTIVE.", nil)
	case errors.Is(err, ErrClassNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Class not found.", nil)
	case errors.Is(err, ErrClassNameTaken):
		apierror.Respond(c, http.StatusConflict, "NAME_TAKEN", "A class with that name already exists in this academic year.", nil)
	case errors.Is(err, ErrInvalidCapacity):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Capacity must be a positive integer.", nil)
	case errors.Is(err, ErrSubjectAlreadyAssigned):
		apierror.Respond(c, http.StatusConflict, "ALREADY_ASSIGNED", "Subject is already assigned to this class.", nil)
	case errors.Is(err, ErrSubjectNotAssigned):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Subject is not assigned to this class.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the academic operation.", nil)
	}
}

// --- DTOs ---

type academicYearDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	IsCurrent bool   `json:"is_current"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toAcademicYearDTO(y db.AcademicYear) academicYearDTO {
	return academicYearDTO{
		ID:        y.ID.String(),
		Name:      y.Name,
		StartDate: dateString(y.StartDate),
		EndDate:   dateString(y.EndDate),
		IsCurrent: y.IsCurrent,
		CreatedAt: tsString(y.CreatedAt),
		UpdatedAt: tsString(y.UpdatedAt),
	}
}

type termDTO struct {
	ID               string `json:"id"`
	AcademicYearID   string `json:"academic_year_id"`
	AcademicYearName string `json:"academic_year_name,omitempty"`
	Name             string `json:"name"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	IsCurrent        bool   `json:"is_current"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toTermDTO(t db.Term, yearName string) termDTO {
	return termDTO{
		ID:               t.ID.String(),
		AcademicYearID:   t.AcademicYearID.String(),
		AcademicYearName: yearName,
		Name:             t.Name,
		StartDate:        dateString(t.StartDate),
		EndDate:          dateString(t.EndDate),
		IsCurrent:        t.IsCurrent,
		CreatedAt:        tsString(t.CreatedAt),
		UpdatedAt:        tsString(t.UpdatedAt),
	}
}

type programmeDTO struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toProgrammeDTO(p db.Programme) programmeDTO {
	return programmeDTO{
		ID:          p.ID.String(),
		Code:        p.Code,
		Name:        p.Name,
		Description: textPtr(p.Description),
		CreatedAt:   tsString(p.CreatedAt),
		UpdatedAt:   tsString(p.UpdatedAt),
	}
}

type subjectDTO struct {
	ID             string  `json:"id"`
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Description    *string `json:"description,omitempty"`
	Type           string  `json:"type"`
	DepartmentID   *string `json:"department_id,omitempty"`
	DepartmentName *string `json:"department_name,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func toSubjectDTO(s db.Subject, deptName *string) subjectDTO {
	return subjectDTO{
		ID:             s.ID.String(),
		Code:           s.Code,
		Name:           s.Name,
		Description:    textPtr(s.Description),
		Type:           s.Type,
		DepartmentID:   uuidPtrString(s.DepartmentID),
		DepartmentName: deptName,
		CreatedAt:      tsString(s.CreatedAt),
		UpdatedAt:      tsString(s.UpdatedAt),
	}
}

type classDTO struct {
	ID               string  `json:"id"`
	AcademicYearID   string  `json:"academic_year_id"`
	AcademicYearName string  `json:"academic_year_name,omitempty"`
	ProgrammeID      *string `json:"programme_id,omitempty"`
	ProgrammeName    *string `json:"programme_name,omitempty"`
	ProgrammeCode    *string `json:"programme_code,omitempty"`
	Name             string  `json:"name"`
	Level            *string `json:"level,omitempty"`
	Capacity         *int32  `json:"capacity,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func toClassDTO(c db.Class, yearName string, progName, progCode *string) classDTO {
	return classDTO{
		ID:               c.ID.String(),
		AcademicYearID:   c.AcademicYearID.String(),
		AcademicYearName: yearName,
		ProgrammeID:      uuidPtrString(c.ProgrammeID),
		ProgrammeName:    progName,
		ProgrammeCode:    progCode,
		Name:             c.Name,
		Level:            textPtr(c.Level),
		Capacity:         int4Ptr(c.Capacity),
		CreatedAt:        tsString(c.CreatedAt),
		UpdatedAt:        tsString(c.UpdatedAt),
	}
}

// --- Academic Years Endpoints ---

func (h *Handler) ListAcademicYears(c *gin.Context) {
	years, err := h.service.ListAcademicYears(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list academic years.", nil)
		return
	}
	dtos := make([]academicYearDTO, 0, len(years))
	for _, y := range years {
		dtos = append(dtos, toAcademicYearDTO(y))
	}
	c.JSON(http.StatusOK, gin.H{"academic_years": dtos})
}

func (h *Handler) GetCurrentAcademicYear(c *gin.Context) {
	year, err := h.service.GetCurrentAcademicYear(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"academic_year": toAcademicYearDTO(year)})
}

func (h *Handler) GetAcademicYear(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	year, err := h.service.GetAcademicYear(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"academic_year": toAcademicYearDTO(year)})
}

type createAcademicYearRequest struct {
	Name      string `json:"name" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
	IsCurrent bool   `json:"is_current"`
}

func (h *Handler) CreateAcademicYear(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var req createAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "name, start_date, and end_date are required.", nil)
		return
	}

	startDate, ok := parseDateParam(c, req.StartDate, "start_date")
	if !ok {
		return
	}
	endDate, ok := parseDateParam(c, req.EndDate, "end_date")
	if !ok {
		return
	}

	year, err := h.service.CreateAcademicYear(c.Request.Context(), actorID, CreateAcademicYearInput{
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		IsCurrent: req.IsCurrent,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"academic_year": toAcademicYearDTO(year)})
}

type updateAcademicYearRequest struct {
	Name      *string `json:"name"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

func (h *Handler) UpdateAcademicYear(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	in := UpdateAcademicYearInput{Name: req.Name}
	if req.StartDate != nil {
		t, ok := parseDateParam(c, *req.StartDate, "start_date")
		if !ok {
			return
		}
		in.StartDate = &t
	}
	if req.EndDate != nil {
		t, ok := parseDateParam(c, *req.EndDate, "end_date")
		if !ok {
			return
		}
		in.EndDate = &t
	}

	year, err := h.service.UpdateAcademicYear(c.Request.Context(), actorID, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"academic_year": toAcademicYearDTO(year)})
}

func (h *Handler) SetCurrentAcademicYear(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	year, err := h.service.SetCurrentAcademicYear(c.Request.Context(), actorID, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"academic_year": toAcademicYearDTO(year)})
}

// --- Terms Endpoints ---

func (h *Handler) ListTerms(c *gin.Context) {
	var yearIDPtr *uuid.UUID
	if raw := c.Query("academic_year_id"); raw != "" {
		id, ok := parseUUIDParam(c, raw, "academic_year_id")
		if !ok {
			return
		}
		yearIDPtr = &id
	}

	rows, err := h.service.ListTerms(c.Request.Context(), yearIDPtr)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list terms.", nil)
		return
	}

	dtos := make([]termDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, termDTO{
			ID:               r.ID.String(),
			AcademicYearID:   r.AcademicYearID.String(),
			AcademicYearName: r.AcademicYearName,
			Name:             r.Name,
			StartDate:        dateString(r.StartDate),
			EndDate:          dateString(r.EndDate),
			IsCurrent:        r.IsCurrent,
			CreatedAt:        tsString(r.CreatedAt),
			UpdatedAt:        tsString(r.UpdatedAt),
		})
	}
	c.JSON(http.StatusOK, gin.H{"terms": dtos})
}

func (h *Handler) GetCurrentTerm(c *gin.Context) {
	r, err := h.service.GetCurrentTerm(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"term": termDTO{
		ID:               r.ID.String(),
		AcademicYearID:   r.AcademicYearID.String(),
		AcademicYearName: r.AcademicYearName,
		Name:             r.Name,
		StartDate:        dateString(r.StartDate),
		EndDate:          dateString(r.EndDate),
		IsCurrent:        r.IsCurrent,
		CreatedAt:        tsString(r.CreatedAt),
		UpdatedAt:        tsString(r.UpdatedAt),
	}})
}

func (h *Handler) GetTerm(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	r, err := h.service.GetTerm(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"term": termDTO{
		ID:               r.ID.String(),
		AcademicYearID:   r.AcademicYearID.String(),
		AcademicYearName: r.AcademicYearName,
		Name:             r.Name,
		StartDate:        dateString(r.StartDate),
		EndDate:          dateString(r.EndDate),
		IsCurrent:        r.IsCurrent,
		CreatedAt:        tsString(r.CreatedAt),
		UpdatedAt:        tsString(r.UpdatedAt),
	}})
}

type createTermRequest struct {
	AcademicYearID string `json:"academic_year_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	StartDate      string `json:"start_date" binding:"required"`
	EndDate        string `json:"end_date" binding:"required"`
	IsCurrent      bool   `json:"is_current"`
}

func (h *Handler) CreateTerm(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var req createTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "academic_year_id, name, start_date, and end_date are required.", nil)
		return
	}

	yearID, ok := parseUUIDParam(c, req.AcademicYearID, "academic_year_id")
	if !ok {
		return
	}
	startDate, ok := parseDateParam(c, req.StartDate, "start_date")
	if !ok {
		return
	}
	endDate, ok := parseDateParam(c, req.EndDate, "end_date")
	if !ok {
		return
	}

	term, err := h.service.CreateTerm(c.Request.Context(), actorID, CreateTermInput{
		AcademicYearID: yearID,
		Name:           req.Name,
		StartDate:      startDate,
		EndDate:        endDate,
		IsCurrent:      req.IsCurrent,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"term": toTermDTO(term, "")})
}

type updateTermRequest struct {
	Name      *string `json:"name"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

func (h *Handler) UpdateTerm(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateTermRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	in := UpdateTermInput{Name: req.Name}
	if req.StartDate != nil {
		t, ok := parseDateParam(c, *req.StartDate, "start_date")
		if !ok {
			return
		}
		in.StartDate = &t
	}
	if req.EndDate != nil {
		t, ok := parseDateParam(c, *req.EndDate, "end_date")
		if !ok {
			return
		}
		in.EndDate = &t
	}

	term, err := h.service.UpdateTerm(c.Request.Context(), actorID, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"term": toTermDTO(term, "")})
}

func (h *Handler) SetCurrentTerm(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	term, err := h.service.SetCurrentTerm(c.Request.Context(), actorID, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"term": toTermDTO(term, "")})
}

// --- Programmes Endpoints ---

func (h *Handler) ListProgrammes(c *gin.Context) {
	progs, err := h.service.ListProgrammes(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list programmes.", nil)
		return
	}
	dtos := make([]programmeDTO, 0, len(progs))
	for _, p := range progs {
		dtos = append(dtos, toProgrammeDTO(p))
	}
	c.JSON(http.StatusOK, gin.H{"programmes": dtos})
}

func (h *Handler) GetProgramme(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	p, err := h.service.GetProgramme(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"programme": toProgrammeDTO(p)})
}

type createProgrammeRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

func (h *Handler) CreateProgramme(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var req createProgrammeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "code and name are required.", nil)
		return
	}

	p, err := h.service.CreateProgramme(c.Request.Context(), actorID, CreateProgrammeInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"programme": toProgrammeDTO(p)})
}

type updateProgrammeRequest struct {
	Code        *string `json:"code"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *Handler) UpdateProgramme(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateProgrammeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	p, err := h.service.UpdateProgramme(c.Request.Context(), actorID, id, UpdateProgrammeInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"programme": toProgrammeDTO(p)})
}

// --- Subjects Endpoints ---

func (h *Handler) ListSubjects(c *gin.Context) {
	filter := SubjectListFilter{
		Type:   strings.TrimSpace(c.Query("type")),
		Search: strings.TrimSpace(c.Query("search")),
	}
	if raw := c.Query("department_id"); raw != "" {
		id, ok := parseUUIDParam(c, raw, "department_id")
		if !ok {
			return
		}
		filter.DepartmentID = &id
	}

	rows, err := h.service.ListSubjects(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list subjects.", nil)
		return
	}

	dtos := make([]subjectDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, subjectDTO{
			ID:             r.ID.String(),
			Code:           r.Code,
			Name:           r.Name,
			Description:    textPtr(r.Description),
			Type:           r.Type,
			DepartmentID:   uuidPtrString(r.DepartmentID),
			DepartmentName: textPtr(r.DepartmentName),
			CreatedAt:      tsString(r.CreatedAt),
			UpdatedAt:      tsString(r.UpdatedAt),
		})
	}
	c.JSON(http.StatusOK, gin.H{"subjects": dtos})
}

func (h *Handler) GetSubject(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	r, err := h.service.GetSubject(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"subject": subjectDTO{
		ID:             r.ID.String(),
		Code:           r.Code,
		Name:           r.Name,
		Description:    textPtr(r.Description),
		Type:           r.Type,
		DepartmentID:   uuidPtrString(r.DepartmentID),
		DepartmentName: textPtr(r.DepartmentName),
		CreatedAt:      tsString(r.CreatedAt),
		UpdatedAt:      tsString(r.UpdatedAt),
	}})
}

type createSubjectRequest struct {
	Code         string  `json:"code" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	Description  *string `json:"description"`
	Type         string  `json:"type"`
	DepartmentID *string `json:"department_id"`
}

func (h *Handler) CreateSubject(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var req createSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "code and name are required.", nil)
		return
	}

	var deptID *uuid.UUID
	if req.DepartmentID != nil && *req.DepartmentID != "" {
		id, ok := parseUUIDParam(c, *req.DepartmentID, "department_id")
		if !ok {
			return
		}
		deptID = &id
	}

	s, err := h.service.CreateSubject(c.Request.Context(), actorID, CreateSubjectInput{
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		DepartmentID: deptID,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"subject": toSubjectDTO(s, nil)})
}

type updateSubjectRequest struct {
	Code         *string `json:"code"`
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Type         *string `json:"type"`
	DepartmentID *string `json:"department_id"`
}

func (h *Handler) UpdateSubject(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	var deptID *uuid.UUID
	if req.DepartmentID != nil {
		if *req.DepartmentID != "" {
			d, ok := parseUUIDParam(c, *req.DepartmentID, "department_id")
			if !ok {
				return
			}
			deptID = &d
		}
	}

	s, err := h.service.UpdateSubject(c.Request.Context(), actorID, id, UpdateSubjectInput{
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		DepartmentID: deptID,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"subject": toSubjectDTO(s, nil)})
}

// --- Classes Endpoints ---

func (h *Handler) ListClasses(c *gin.Context) {
	filter := ClassListFilter{
		Level: strings.TrimSpace(c.Query("level")),
	}
	if raw := c.Query("academic_year_id"); raw != "" {
		id, ok := parseUUIDParam(c, raw, "academic_year_id")
		if !ok {
			return
		}
		filter.AcademicYearID = &id
	}
	if raw := c.Query("programme_id"); raw != "" {
		id, ok := parseUUIDParam(c, raw, "programme_id")
		if !ok {
			return
		}
		filter.ProgrammeID = &id
	}

	rows, err := h.service.ListClasses(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list classes.", nil)
		return
	}

	dtos := make([]classDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, classDTO{
			ID:               r.ID.String(),
			AcademicYearID:   r.AcademicYearID.String(),
			AcademicYearName: r.AcademicYearName,
			ProgrammeID:      uuidPtrString(r.ProgrammeID),
			ProgrammeName:    textPtr(r.ProgrammeName),
			ProgrammeCode:    textPtr(r.ProgrammeCode),
			Name:             r.Name,
			Level:            textPtr(r.Level),
			Capacity:         int4Ptr(r.Capacity),
			CreatedAt:        tsString(r.CreatedAt),
			UpdatedAt:        tsString(r.UpdatedAt),
		})
	}
	c.JSON(http.StatusOK, gin.H{"classes": dtos})
}

func (h *Handler) GetClass(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	r, err := h.service.GetClass(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"class": classDTO{
		ID:               r.ID.String(),
		AcademicYearID:   r.AcademicYearID.String(),
		AcademicYearName: r.AcademicYearName,
		ProgrammeID:      uuidPtrString(r.ProgrammeID),
		ProgrammeName:    textPtr(r.ProgrammeName),
		ProgrammeCode:    textPtr(r.ProgrammeCode),
		Name:             r.Name,
		Level:            textPtr(r.Level),
		Capacity:         int4Ptr(r.Capacity),
		CreatedAt:        tsString(r.CreatedAt),
		UpdatedAt:        tsString(r.UpdatedAt),
	}})
}

type createClassRequest struct {
	AcademicYearID string  `json:"academic_year_id" binding:"required"`
	ProgrammeID    *string `json:"programme_id"`
	Name           string  `json:"name" binding:"required"`
	Level          *string `json:"level"`
	Capacity       *int32  `json:"capacity"`
}

func (h *Handler) CreateClass(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}

	var req createClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "academic_year_id and name are required.", nil)
		return
	}

	yearID, ok := parseUUIDParam(c, req.AcademicYearID, "academic_year_id")
	if !ok {
		return
	}

	var progID *uuid.UUID
	if req.ProgrammeID != nil && *req.ProgrammeID != "" {
		id, ok := parseUUIDParam(c, *req.ProgrammeID, "programme_id")
		if !ok {
			return
		}
		progID = &id
	}

	cl, err := h.service.CreateClass(c.Request.Context(), actorID, CreateClassInput{
		AcademicYearID: yearID,
		ProgrammeID:    progID,
		Name:           req.Name,
		Level:          req.Level,
		Capacity:       req.Capacity,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"class": toClassDTO(cl, "", nil, nil)})
}

type updateClassRequest struct {
	ProgrammeID *string `json:"programme_id"`
	Name        *string `json:"name"`
	Level       *string `json:"level"`
	Capacity    *int32  `json:"capacity"`
}

func (h *Handler) UpdateClass(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	var progID *uuid.UUID
	if req.ProgrammeID != nil {
		if *req.ProgrammeID != "" {
			p, ok := parseUUIDParam(c, *req.ProgrammeID, "programme_id")
			if !ok {
				return
			}
			progID = &p
		}
	}

	cl, err := h.service.UpdateClass(c.Request.Context(), actorID, id, UpdateClassInput{
		ProgrammeID: progID,
		Name:        req.Name,
		Level:       req.Level,
		Capacity:    req.Capacity,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"class": toClassDTO(cl, "", nil, nil)})
}

type assignSubjectRequest struct {
	SubjectID string `json:"subject_id" binding:"required"`
}

func (h *Handler) AssignSubjectToClass(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	classID, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req assignSubjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "subject_id is required.", nil)
		return
	}
	subjectID, ok := parseUUIDParam(c, req.SubjectID, "subject_id")
	if !ok {
		return
	}

	cs, err := h.service.AssignSubjectToClass(c.Request.Context(), actorID, classID, subjectID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"class_id":    cs.ClassID.String(),
		"subject_id":  cs.SubjectID.String(),
		"assigned_at": tsString(cs.CreatedAt),
	})
}

func (h *Handler) RemoveSubjectFromClass(c *gin.Context) {
	actorID, ok := requireActor(c)
	if !ok {
		return
	}
	classID, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	subjectID, ok := parseUUIDParam(c, c.Param("subject_id"), "subject_id")
	if !ok {
		return
	}

	if err := h.service.RemoveSubjectFromClass(c.Request.Context(), actorID, classID, subjectID); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}

type classSubjectDTO struct {
	ID           string  `json:"id"`
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Description  *string `json:"description,omitempty"`
	Type         string  `json:"type"`
	DepartmentID *string `json:"department_id,omitempty"`
	AssignedAt   string  `json:"assigned_at"`
}

func (h *Handler) ListSubjectsForClass(c *gin.Context) {
	classID, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	rows, err := h.service.ListSubjectsForClass(c.Request.Context(), classID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	dtos := make([]classSubjectDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, classSubjectDTO{
			ID:           r.ID.String(),
			Code:         r.Code,
			Name:         r.Name,
			Description:  textPtr(r.Description),
			Type:         r.Type,
			DepartmentID: uuidPtrString(r.DepartmentID),
			AssignedAt:   tsString(r.AssignedAt),
		})
	}

	c.JSON(http.StatusOK, gin.H{"subjects": dtos})
}
