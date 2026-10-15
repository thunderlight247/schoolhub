package students

import (
	"context"
	"errors"
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

// ServiceAPI is what the handlers need from the students service. An
// interface (not the concrete *Service) so handler tests can stub it —
// same reasoning as internal/staff.ServiceAPI.
type ServiceAPI interface {
	CreateStudent(ctx context.Context, actorID uuid.UUID, in CreateStudentInput) (StudentDetail, error)
	GetStudent(ctx context.Context, id uuid.UUID) (StudentDetail, error)
	ListStudents(ctx context.Context, f ListFilter) ([]db.ListStudentsRow, int64, error)
	UpdateStudent(ctx context.Context, actorID, id uuid.UUID, in UpdateStudentInput) (StudentDetail, error)
	SetStudentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (StudentDetail, error)
	LinkGuardianToStudent(ctx context.Context, actorID, studentID uuid.UUID, in LinkGuardianInput) ([]db.ListGuardiansForStudentRow, error)
	UnlinkGuardianFromStudent(ctx context.Context, actorID, studentID, guardianID uuid.UUID) ([]db.ListGuardiansForStudentRow, error)
	CreateGuardian(ctx context.Context, actorID uuid.UUID, in CreateGuardianInput) (GuardianDetail, error)
	GetGuardian(ctx context.Context, id uuid.UUID) (GuardianDetail, error)
	ListGuardians(ctx context.Context, f GuardianListFilter) ([]db.ListGuardiansRow, int64, error)
	UpdateGuardian(ctx context.Context, actorID, id uuid.UUID, in UpdateGuardianInput) (GuardianDetail, error)
	CreateGuardianAccount(ctx context.Context, actorID, guardianID uuid.UUID, in GuardianAccountInput) error
}

// Handler exposes the students/guardians endpoints (sections 14-15/25).
// Route mounting and permission gating happen in cmd/server/main.go; every
// handler here assumes auth.RequireAuth has run (it reads the
// authenticated user for audit actor attribution) and that the matching
// student.* permission middleware has already authorized the request.
type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

// --- pgtype -> JSON helpers (duplicated from internal/staff; see that
// package's comment on why these aren't shared) ---

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func datePtr(t pgtype.Date) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format("2006-01-02")
	return &s
}

func tsString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

// --- DTOs and conversions ---

type guardianLinkDTO struct {
	GuardianID   string  `json:"guardian_id"`
	Relationship string  `json:"relationship"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	OtherNames   *string `json:"other_names,omitempty"`
	Occupation   *string `json:"occupation,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	Email        *string `json:"email,omitempty"`
}

func toGuardianLinkDTO(r db.ListGuardiansForStudentRow) guardianLinkDTO {
	return guardianLinkDTO{
		GuardianID:   r.ID.String(),
		Relationship: r.Relationship,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		OtherNames:   textPtr(r.OtherNames),
		Occupation:   textPtr(r.Occupation),
		Phone:        textPtr(r.Phone),
		Email:        textPtr(r.Email),
	}
}

func toGuardianLinkDTOs(rows []db.ListGuardiansForStudentRow) []guardianLinkDTO {
	dtos := make([]guardianLinkDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, toGuardianLinkDTO(r))
	}
	return dtos
}

type studentLinkDTO struct {
	StudentID     string  `json:"student_id"`
	StudentNumber string  `json:"student_number"`
	Status        string  `json:"status"`
	Relationship  string  `json:"relationship"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	OtherNames    *string `json:"other_names,omitempty"`
}

func toStudentLinkDTO(r db.ListStudentsForGuardianRow) studentLinkDTO {
	return studentLinkDTO{
		StudentID:     r.ID.String(),
		StudentNumber: r.StudentNumber,
		Status:        r.Status,
		Relationship:  r.Relationship,
		FirstName:     r.FirstName,
		LastName:      r.LastName,
		OtherNames:    textPtr(r.OtherNames),
	}
}

// studentDTO is the full student profile: the joined person+student row
// plus linked guardians.
type studentDTO struct {
	ID            string            `json:"id"`
	StudentNumber string            `json:"student_number"`
	FirstName     string            `json:"first_name"`
	LastName      string            `json:"last_name"`
	OtherNames    *string           `json:"other_names,omitempty"`
	Gender        *string           `json:"gender,omitempty"`
	DateOfBirth   *string           `json:"date_of_birth,omitempty"`
	Phone         *string           `json:"phone,omitempty"`
	Email         *string           `json:"email,omitempty"`
	Address       *string           `json:"address,omitempty"`
	AdmissionDate *string           `json:"admission_date,omitempty"`
	Status        string            `json:"status"`
	Guardians     []guardianLinkDTO `json:"guardians"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

func toStudentDTO(d StudentDetail) studentDTO {
	r := d.Row
	return studentDTO{
		ID:            r.ID.String(),
		StudentNumber: r.StudentNumber,
		FirstName:     r.FirstName,
		LastName:      r.LastName,
		OtherNames:    textPtr(r.OtherNames),
		Gender:        textPtr(r.Gender),
		DateOfBirth:   datePtr(r.DateOfBirth),
		Phone:         textPtr(r.Phone),
		Email:         textPtr(r.Email),
		Address:       textPtr(r.Address),
		AdmissionDate: datePtr(r.AdmissionDate),
		Status:        r.Status,
		Guardians:     toGuardianLinkDTOs(d.Guardians),
		CreatedAt:     tsString(r.CreatedAt),
		UpdatedAt:     tsString(r.UpdatedAt),
	}
}

// studentListItemDTO is the list-view row: the columns the list screens
// show, no guardians (fetch the full profile for that).
type studentListItemDTO struct {
	ID            string  `json:"id"`
	StudentNumber string  `json:"student_number"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	OtherNames    *string `json:"other_names,omitempty"`
	Status        string  `json:"status"`
	AdmissionDate *string `json:"admission_date,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

func toStudentListItemDTO(r db.ListStudentsRow) studentListItemDTO {
	return studentListItemDTO{
		ID:            r.ID.String(),
		StudentNumber: r.StudentNumber,
		FirstName:     r.FirstName,
		LastName:      r.LastName,
		OtherNames:    textPtr(r.OtherNames),
		Status:        r.Status,
		AdmissionDate: datePtr(r.AdmissionDate),
		CreatedAt:     tsString(r.CreatedAt),
	}
}

// guardianDTO is the full guardian profile: the joined person+guardian row
// plus linked students.
type guardianDTO struct {
	ID          string           `json:"id"`
	FirstName   string           `json:"first_name"`
	LastName    string           `json:"last_name"`
	OtherNames  *string          `json:"other_names,omitempty"`
	Gender      *string          `json:"gender,omitempty"`
	DateOfBirth *string          `json:"date_of_birth,omitempty"`
	Phone       *string          `json:"phone,omitempty"`
	Email       *string          `json:"email,omitempty"`
	Address     *string          `json:"address,omitempty"`
	Occupation  *string          `json:"occupation,omitempty"`
	Students    []studentLinkDTO `json:"students"`
	// Optional login account (nil until an admin creates parent access).
	Account   *accountDTO `json:"account,omitempty"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

// accountDTO is the parent-access summary shown on the guardian profile.
type accountDTO struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Status string   `json:"status"`
	Roles  []string `json:"roles"`
}

func toGuardianDTO(d GuardianDetail) guardianDTO {
	r := d.Row
	students := make([]studentLinkDTO, 0, len(d.Students))
	for _, st := range d.Students {
		students = append(students, toStudentLinkDTO(st))
	}
	var account *accountDTO
	if r.AccountID.Valid {
		account = &accountDTO{
			ID:     uuid.UUID(r.AccountID.Bytes).String(),
			Email:  r.AccountEmail.String,
			Status: r.AccountStatus.String,
			Roles:  d.AccountRoles,
		}
	}
	return guardianDTO{
		ID:          r.ID.String(),
		FirstName:   r.FirstName,
		LastName:    r.LastName,
		OtherNames:  textPtr(r.OtherNames),
		Gender:      textPtr(r.Gender),
		DateOfBirth: datePtr(r.DateOfBirth),
		Phone:       textPtr(r.Phone),
		Email:       textPtr(r.Email),
		Address:     textPtr(r.Address),
		Occupation:  textPtr(r.Occupation),
		Students:    students,
		Account:     account,
		CreatedAt:   tsString(r.CreatedAt),
		UpdatedAt:   tsString(r.UpdatedAt),
	}
}

type guardianListItemDTO struct {
	ID         string  `json:"id"`
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	OtherNames *string `json:"other_names,omitempty"`
	Occupation *string `json:"occupation,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Email      *string `json:"email,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

func toGuardianListItemDTO(r db.ListGuardiansRow) guardianListItemDTO {
	return guardianListItemDTO{
		ID:         r.ID.String(),
		FirstName:  r.FirstName,
		LastName:   r.LastName,
		OtherNames: textPtr(r.OtherNames),
		Occupation: textPtr(r.Occupation),
		Phone:      textPtr(r.Phone),
		Email:      textPtr(r.Email),
		CreatedAt:  tsString(r.CreatedAt),
	}
}

// respondServiceError maps service sentinel errors to the API's structured
// error responses; anything unmapped is a 500 whose details stay in the
// server logs, never in the response body.
func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrStudentNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Student not found.", nil)
	case errors.Is(err, ErrGuardianNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Guardian not found.", nil)
	case errors.Is(err, ErrStudentNumberTaken):
		apierror.Respond(c, http.StatusConflict, "STUDENT_NUMBER_TAKEN", "That student number is already in use.", nil)
	case errors.Is(err, ErrInvalidStatus):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid status.", nil)
	case errors.Is(err, ErrGuardianAlreadyLinked):
		apierror.Respond(c, http.StatusConflict, "GUARDIAN_ALREADY_LINKED", "That guardian is already linked to this student.", nil)
	case errors.Is(err, ErrGuardianLinkNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "That guardian is not linked to this student.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the student operation.", nil)
	}
}

// actorID returns the authenticated user's id for audit attribution, or
// ok=false when the route wasn't mounted behind auth.RequireAuth.
func actorID(c *gin.Context) (uuid.UUID, bool) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		return uuid.Nil, false
	}
	return authUser.User.ID, true
}

// requireActor is the shared guard for mutating handlers.
func requireActor(c *gin.Context) (uuid.UUID, bool) {
	id, ok := actorID(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
	}
	return id, ok
}

// CreateGuardianAccount handles POST /api/v1/guardians/:id/account — the
// "Create parent access" action on a guardian's profile. Provisions the
// PARENT login (Argon2id-hashed password, audit entry) and fires the
// best-effort welcome email + Clerk invitation after commit.
func (h *Handler) CreateGuardianAccount(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	id, ok := parseUUIDParam(c, c.Param("id"), "guardian id")
	if !ok {
		return
	}

	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "email (valid address) and password (min 8 chars) are required.", nil)
		return
	}

	in := GuardianAccountInput{Email: req.Email, Password: req.Password, RoleName: "PARENT"}
	if req.Role != "" && req.Role != "PARENT" {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Parent access is always granted with the PARENT role.", nil)
		return
	}

	if err := h.service.CreateGuardianAccount(c.Request.Context(), actor, id, in); err != nil {
		switch {
		case errors.Is(err, ErrGuardianNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Guardian not found.", nil)
		case errors.Is(err, ErrAccountExists):
			apierror.Respond(c, http.StatusConflict, "ACCOUNT_EXISTS", "This guardian already has a user account.", nil)
		case errors.Is(err, ErrEmailTaken):
			apierror.Respond(c, http.StatusConflict, "EMAIL_TAKEN", "That email is already in use by an account.", nil)
		case errors.Is(err, ErrUnknownRole):
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Unknown role.", nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not create the parent account.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"account": gin.H{
		"email":  req.Email,
		"status": "ACTIVE",
		"roles":  []string{"PARENT"},
	}})
}

// parseDateParam converts a client-supplied YYYY-MM-DD string, writing the
// validation error itself (so every date field fails identically).
func parseDateParam(c *gin.Context, raw, field string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", field+" must be a date in YYYY-MM-DD format.", nil)
		return time.Time{}, false
	}
	return t, true
}

// parseUUIDParam converts a client-supplied UUID string.
func parseUUIDParam(c *gin.Context, raw, field string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", field+" must be a UUID.", nil)
		return uuid.Nil, false
	}
	return id, true
}

// --- students: reads ---

// ListStudents handles GET /students?page=&page_size=&status=&search=
func (h *Handler) ListStudents(c *gin.Context) {
	filter := ListFilter{Search: strings.TrimSpace(c.Query("search"))}

	if raw := c.Query("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "page must be an integer.", nil)
			return
		}
		filter.Page = page
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "page_size must be an integer.", nil)
			return
		}
		filter.PageSize = pageSize
	}
	if raw := c.Query("status"); raw != "" {
		if !contains(StudentStatuses, raw) {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid status filter.", nil)
			return
		}
		filter.Status = raw
	}

	rows, total, err := h.service.ListStudents(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list students.", nil)
		return
	}

	page, pageSize := NormalizePagination(filter.Page, filter.PageSize)
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	dtos := make([]studentListItemDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, toStudentListItemDTO(r))
	}

	c.JSON(http.StatusOK, gin.H{
		"students": dtos,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetStudent handles GET /students/:id.
func (h *Handler) GetStudent(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	detail, err := h.service.GetStudent(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"student": toStudentDTO(detail)})
}

// --- students: mutations ---

type createStudentRequest struct {
	StudentNumber string  `json:"student_number" binding:"required"`
	FirstName     string  `json:"first_name" binding:"required"`
	LastName      string  `json:"last_name" binding:"required"`
	OtherNames    *string `json:"other_names"`
	Gender        *string `json:"gender"`
	DateOfBirth   *string `json:"date_of_birth"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Address       *string `json:"address"`
	AdmissionDate *string `json:"admission_date"`
}

// CreateStudent handles POST /students. Status always starts ACTIVE.
func (h *Handler) CreateStudent(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	var req createStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "student_number, first_name, and last_name are required.", nil)
		return
	}

	in := CreateStudentInput{
		StudentNumber: req.StudentNumber,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		OtherNames:    req.OtherNames,
		Gender:        req.Gender,
		Phone:         req.Phone,
		Email:         req.Email,
		Address:       req.Address,
	}

	if req.DateOfBirth != nil {
		t, ok := parseDateParam(c, *req.DateOfBirth, "date_of_birth")
		if !ok {
			return
		}
		in.DateOfBirth = &t
	}
	if req.AdmissionDate != nil {
		t, ok := parseDateParam(c, *req.AdmissionDate, "admission_date")
		if !ok {
			return
		}
		in.AdmissionDate = &t
	}

	detail, err := h.service.CreateStudent(c.Request.Context(), act, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"student": toStudentDTO(detail)})
}

type updateStudentRequest struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	OtherNames  *string `json:"other_names"`
	Gender      *string `json:"gender"`
	DateOfBirth *string `json:"date_of_birth"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	Address     *string `json:"address"`
}

// UpdateStudent handles PATCH /students/:id. Absent fields are left
// unchanged; status changes live on their own endpoint.
func (h *Handler) UpdateStudent(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	in := UpdateStudentInput{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		OtherNames: req.OtherNames,
		Gender:     req.Gender,
		Phone:      req.Phone,
		Email:      req.Email,
		Address:    req.Address,
	}
	if req.DateOfBirth != nil {
		t, ok := parseDateParam(c, *req.DateOfBirth, "date_of_birth")
		if !ok {
			return
		}
		in.DateOfBirth = &t
	}

	detail, err := h.service.UpdateStudent(c.Request.Context(), act, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"student": toStudentDTO(detail)})
}

type setStudentStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// SetStudentStatus handles PUT /students/:id/status (section 14's
// archive/activate). Gated by student.archive, stricter than
// student.update.
func (h *Handler) SetStudentStatus(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req setStudentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status is required.", nil)
		return
	}

	detail, err := h.service.SetStudentStatus(c.Request.Context(), act, id, req.Status)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"student": toStudentDTO(detail)})
}

// --- guardian links on a student ---

type linkGuardianRequest struct {
	GuardianID   string `json:"guardian_id" binding:"required"`
	Relationship string `json:"relationship" binding:"required"`
}

// LinkGuardian handles POST /students/:id/guardians — link an EXISTING
// guardian (create them first via POST /guardians if they don't exist
// yet). Returns the student's full updated guardian list.
func (h *Handler) LinkGuardian(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	studentID, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req linkGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "guardian_id and relationship are required.", nil)
		return
	}

	guardianID, ok := parseUUIDParam(c, req.GuardianID, "guardian_id")
	if !ok {
		return
	}

	guardians, err := h.service.LinkGuardianToStudent(c.Request.Context(), act, studentID, LinkGuardianInput{
		GuardianID:   guardianID,
		Relationship: req.Relationship,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"guardians": toGuardianLinkDTOs(guardians)})
}

// UnlinkGuardian handles DELETE /students/:id/guardians/:guardian_id.
// Removes the association only — the guardian record (and any other
// student they're linked to) is untouched.
func (h *Handler) UnlinkGuardian(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	studentID, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}
	guardianID, ok := parseUUIDParam(c, c.Param("guardian_id"), "guardian_id")
	if !ok {
		return
	}

	guardians, err := h.service.UnlinkGuardianFromStudent(c.Request.Context(), act, studentID, guardianID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"guardians": toGuardianLinkDTOs(guardians)})
}

// --- guardians (standalone) ---

// ListGuardians handles GET /guardians?page=&page_size=&search= — the
// guardians directory, and the "search existing guardian to link" lookup.
func (h *Handler) ListGuardians(c *gin.Context) {
	filter := GuardianListFilter{Search: strings.TrimSpace(c.Query("search"))}

	if raw := c.Query("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "page must be an integer.", nil)
			return
		}
		filter.Page = page
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "page_size must be an integer.", nil)
			return
		}
		filter.PageSize = pageSize
	}

	rows, total, err := h.service.ListGuardians(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list guardians.", nil)
		return
	}

	page, pageSize := NormalizePagination(filter.Page, filter.PageSize)
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	dtos := make([]guardianListItemDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, toGuardianListItemDTO(r))
	}

	c.JSON(http.StatusOK, gin.H{
		"guardians": dtos,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetGuardian handles GET /guardians/:id.
func (h *Handler) GetGuardian(c *gin.Context) {
	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	detail, err := h.service.GetGuardian(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"guardian": toGuardianDTO(detail)})
}

type createGuardianRequest struct {
	FirstName   string  `json:"first_name" binding:"required"`
	LastName    string  `json:"last_name" binding:"required"`
	OtherNames  *string `json:"other_names"`
	Gender      *string `json:"gender"`
	DateOfBirth *string `json:"date_of_birth"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	Address     *string `json:"address"`
	Occupation  *string `json:"occupation"`
}

// CreateGuardian handles POST /guardians — registers a new guardian,
// standalone (not yet linked to any student; link separately via
// POST /students/:id/guardians).
func (h *Handler) CreateGuardian(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	var req createGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "first_name and last_name are required.", nil)
		return
	}

	in := CreateGuardianInput{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		OtherNames: req.OtherNames,
		Gender:     req.Gender,
		Phone:      req.Phone,
		Email:      req.Email,
		Address:    req.Address,
		Occupation: req.Occupation,
	}

	if req.DateOfBirth != nil {
		t, ok := parseDateParam(c, *req.DateOfBirth, "date_of_birth")
		if !ok {
			return
		}
		in.DateOfBirth = &t
	}

	detail, err := h.service.CreateGuardian(c.Request.Context(), act, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"guardian": toGuardianDTO(detail)})
}

type updateGuardianRequest struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	OtherNames  *string `json:"other_names"`
	Gender      *string `json:"gender"`
	DateOfBirth *string `json:"date_of_birth"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	Address     *string `json:"address"`
	Occupation  *string `json:"occupation"`
}

// UpdateGuardian handles PATCH /guardians/:id. Absent fields are left
// unchanged.
func (h *Handler) UpdateGuardian(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, ok := parseUUIDParam(c, c.Param("id"), "id")
	if !ok {
		return
	}

	var req updateGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	in := UpdateGuardianInput{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		OtherNames: req.OtherNames,
		Gender:     req.Gender,
		Phone:      req.Phone,
		Email:      req.Email,
		Address:    req.Address,
		Occupation: req.Occupation,
	}
	if req.DateOfBirth != nil {
		t, ok := parseDateParam(c, *req.DateOfBirth, "date_of_birth")
		if !ok {
			return
		}
		in.DateOfBirth = &t
	}

	detail, err := h.service.UpdateGuardian(c.Request.Context(), act, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"guardian": toGuardianDTO(detail)})
}
