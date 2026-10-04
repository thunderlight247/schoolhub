package staff

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

// ServiceAPI is what the handlers need from the staff service. An interface
// (not the concrete *Service) so handler tests can stub it — the concrete
// service's transactional paths need a real database (see service.go).
type ServiceAPI interface {
	CreateStaff(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error)
	GetStaff(ctx context.Context, id uuid.UUID) (StaffDetail, error)
	ListStaff(ctx context.Context, f ListFilter) ([]db.ListStaffRow, int64, error)
	UpdateStaff(ctx context.Context, actorID, id uuid.UUID, in UpdateStaffInput) (StaffDetail, error)
	SetAssignment(ctx context.Context, actorID, id uuid.UUID, in SetAssignmentInput) (StaffDetail, error)
	SetEmploymentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (StaffDetail, error)
	CreateAccount(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error)
	SetAccountRoles(ctx context.Context, actorID, staffID uuid.UUID, roleNames []string) ([]string, error)
	ListDepartments(ctx context.Context) ([]db.Department, error)
	CreateDepartment(ctx context.Context, actorID uuid.UUID, name string, description *string) (db.Department, error)
	ListStaffPositions(ctx context.Context) ([]db.StaffPosition, error)
	CreateStaffPosition(ctx context.Context, actorID uuid.UUID, title, category string, description *string) (db.StaffPosition, error)
}

// Handler exposes the staff-management endpoints (section 12/25). Route
// mounting and permission gating happen in cmd/server/main.go; every
// handler here assumes auth.RequireAuth has run (it reads the authenticated
// user for audit actor attribution) and that the matching staff.* permission
// middleware has already authorized the request.
type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

// --- DTOs and conversions ---

// sqlc maps nullable columns to pgtype wrappers; the API surface uses
// *string/*uuid strings so absent optional fields simply don't appear in
// the JSON (omitempty).

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func uuidPtr(t pgtype.UUID) *string {
	if !t.Valid {
		return nil
	}
	s := uuid.UUID(t.Bytes).String()
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

type accountDTO struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Status string   `json:"status"`
	Roles  []string `json:"roles"`
}

func toAccountDTO(a *Account) *accountDTO {
	if a == nil {
		return nil
	}
	return &accountDTO{ID: a.ID.String(), Email: a.Email, Status: a.Status, Roles: a.Roles}
}

// staffDTO is the full staff profile (the GetStaffByIDRow plus the
// optional account).
type staffDTO struct {
	ID               string      `json:"id"`
	StaffNumber      string      `json:"staff_number"`
	FirstName        string      `json:"first_name"`
	LastName         string      `json:"last_name"`
	OtherNames       *string     `json:"other_names,omitempty"`
	Gender           *string     `json:"gender,omitempty"`
	DateOfBirth      *string     `json:"date_of_birth,omitempty"`
	Phone            *string     `json:"phone,omitempty"`
	Email            *string     `json:"email,omitempty"`
	Address          *string     `json:"address,omitempty"`
	DepartmentID     *string     `json:"department_id,omitempty"`
	DepartmentName   *string     `json:"department_name,omitempty"`
	PositionID       *string     `json:"position_id,omitempty"`
	PositionTitle    *string     `json:"position_title,omitempty"`
	PositionCategory *string     `json:"position_category,omitempty"`
	HiredDate        *string     `json:"hired_date,omitempty"`
	EmploymentStatus string      `json:"employment_status"`
	Account          *accountDTO `json:"account,omitempty"`
	CreatedAt        string      `json:"created_at"`
	UpdatedAt        string      `json:"updated_at"`
}

func toStaffDTO(d StaffDetail) staffDTO {
	r := d.Row
	return staffDTO{
		ID:               r.ID.String(),
		StaffNumber:      r.StaffNumber,
		FirstName:        r.FirstName,
		LastName:         r.LastName,
		OtherNames:       textPtr(r.OtherNames),
		Gender:           textPtr(r.Gender),
		DateOfBirth:      datePtr(r.DateOfBirth),
		Phone:            textPtr(r.Phone),
		Email:            textPtr(r.Email),
		Address:          textPtr(r.Address),
		DepartmentID:     uuidPtr(r.DepartmentID),
		DepartmentName:   textPtr(r.DepartmentName),
		PositionID:       uuidPtr(r.PositionID),
		PositionTitle:    textPtr(r.PositionTitle),
		PositionCategory: textPtr(r.PositionCategory),
		HiredDate:        datePtr(r.HiredDate),
		EmploymentStatus: r.EmploymentStatus,
		Account:          toAccountDTO(d.Account),
		CreatedAt:        tsString(r.CreatedAt),
		UpdatedAt:        tsString(r.UpdatedAt),
	}
}

// staffListItemDTO is the list-view row: the columns the list screens show,
// plus has_account so the UI can offer "create account" only where needed.
type staffListItemDTO struct {
	ID               string  `json:"id"`
	StaffNumber      string  `json:"staff_number"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	OtherNames       *string `json:"other_names,omitempty"`
	DepartmentName   *string `json:"department_name,omitempty"`
	PositionTitle    *string `json:"position_title,omitempty"`
	PositionCategory *string `json:"position_category,omitempty"`
	EmploymentStatus string  `json:"employment_status"`
	HasAccount       bool    `json:"has_account"`
	CreatedAt        string  `json:"created_at"`
}

func toListItemDTO(r db.ListStaffRow) staffListItemDTO {
	return staffListItemDTO{
		ID:               r.ID.String(),
		StaffNumber:      r.StaffNumber,
		FirstName:        r.FirstName,
		LastName:         r.LastName,
		OtherNames:       textPtr(r.OtherNames),
		DepartmentName:   textPtr(r.DepartmentName),
		PositionTitle:    textPtr(r.PositionTitle),
		PositionCategory: textPtr(r.PositionCategory),
		EmploymentStatus: r.EmploymentStatus,
		HasAccount:       r.HasAccount,
		CreatedAt:        tsString(r.CreatedAt),
	}
}

// respondServiceError maps service sentinel errors to the API's structured
// error responses; anything unmapped is a 500 whose details stay in the
// server logs, never in the response body.
func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrStaffNotFound), errors.Is(err, ErrUserNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Staff member not found.", nil)
	case errors.Is(err, ErrStaffNumberTaken):
		apierror.Respond(c, http.StatusConflict, "STAFF_NUMBER_TAKEN", "That staff number is already in use.", nil)
	case errors.Is(err, ErrEmailTaken):
		apierror.Respond(c, http.StatusConflict, "EMAIL_TAKEN", "That email is already in use by an account.", nil)
	case errors.Is(err, ErrAccountExists):
		apierror.Respond(c, http.StatusConflict, "ACCOUNT_EXISTS", "This staff member already has a user account.", nil)
	case errors.Is(err, ErrUnknownDepartment):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Unknown department.", nil)
	case errors.Is(err, ErrUnknownPosition):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Unknown staff position.", nil)
	case errors.Is(err, ErrUnknownRole):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Unknown role name.", nil)
	case errors.Is(err, ErrInvalidStatus):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employment status.", nil)
	case errors.Is(err, ErrInvalidCategory):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid position category.", nil)
	case errors.Is(err, ErrReferenceNameTaken):
		apierror.Respond(c, http.StatusConflict, "NAME_TAKEN", "A department or position with that name already exists.", nil)
	default:
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the staff operation.", nil)
	}
}

// actorID returns the authenticated user's id for audit attribution, or
// ok=false when the route wasn't mounted behind auth.RequireAuth (a wiring
// bug — fail closed rather than write an anonymous audit row).
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

// --- reads ---

// ListStaff handles GET /staff?page=&page_size=&department_id=&employment_status=&position_category=&search=
func (h *Handler) ListStaff(c *gin.Context) {
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

	if raw := c.Query("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "department_id must be a UUID.", nil)
			return
		}
		filter.DepartmentID = &id
	}
	if raw := c.Query("employment_status"); raw != "" {
		if !contains(EmploymentStatuses, raw) {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employment_status filter.", nil)
			return
		}
		filter.EmploymentStatus = raw
	}
	if raw := c.Query("position_category"); raw != "" {
		if !contains(PositionCategories, raw) {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid position_category filter.", nil)
			return
		}
		filter.PositionCategory = raw
	}

	rows, total, err := h.service.ListStaff(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list staff.", nil)
		return
	}

	page, pageSize := NormalizePagination(filter.Page, filter.PageSize)
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	dtos := make([]staffListItemDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, toListItemDTO(r))
	}

	c.JSON(http.StatusOK, gin.H{
		"staff": dtos,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetStaff handles GET /staff/:id.
func (h *Handler) GetStaff(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid staff id.", nil)
		return
	}

	detail, err := h.service.GetStaff(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"staff": toStaffDTO(detail)})
}

// --- mutations ---

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

// parseUUIDParam converts an optional client-supplied UUID string.
func parseUUIDParam(c *gin.Context, raw, field string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", field+" must be a UUID.", nil)
		return uuid.Nil, false
	}
	return id, true
}

type createStaffRequest struct {
	StaffNumber  string  `json:"staff_number" binding:"required"`
	FirstName    string  `json:"first_name" binding:"required"`
	LastName     string  `json:"last_name" binding:"required"`
	OtherNames   *string `json:"other_names"`
	Gender       *string `json:"gender"`
	DateOfBirth  *string `json:"date_of_birth"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	Address      *string `json:"address"`
	DepartmentID *string `json:"department_id"`
	PositionID   *string `json:"position_id"`
	HiredDate    *string `json:"hired_date"`
}

// CreateStaff handles POST /staff. Employment status always starts ACTIVE;
// status changes are a separate, separately-gated operation.
func (h *Handler) CreateStaff(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	var req createStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "staff_number, first_name, and last_name are required.", nil)
		return
	}

	in := CreateStaffInput{
		StaffNumber: req.StaffNumber,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		OtherNames:  req.OtherNames,
		Gender:      req.Gender,
		Phone:       req.Phone,
		Email:       req.Email,
		Address:     req.Address,
	}

	if req.DateOfBirth != nil {
		t, ok := parseDateParam(c, *req.DateOfBirth, "date_of_birth")
		if !ok {
			return
		}
		in.DateOfBirth = &t
	}
	if req.DepartmentID != nil {
		id, ok := parseUUIDParam(c, *req.DepartmentID, "department_id")
		if !ok {
			return
		}
		in.DepartmentID = &id
	}
	if req.PositionID != nil {
		id, ok := parseUUIDParam(c, *req.PositionID, "position_id")
		if !ok {
			return
		}
		in.PositionID = &id
	}
	if req.HiredDate != nil {
		t, ok := parseDateParam(c, *req.HiredDate, "hired_date")
		if !ok {
			return
		}
		in.HiredDate = &t
	}

	detail, err := h.service.CreateStaff(c.Request.Context(), act, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"staff": toStaffDTO(detail)})
}

type updateStaffRequest struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	OtherNames  *string `json:"other_names"`
	Gender      *string `json:"gender"`
	DateOfBirth *string `json:"date_of_birth"`
	Phone       *string `json:"phone"`
	Email       *string `json:"email"`
	Address     *string `json:"address"`
}

// UpdateStaff handles PATCH /staff/:id. Absent fields are left unchanged;
// the assignment triple and employment status live on their own endpoints.
func (h *Handler) UpdateStaff(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid staff id.", nil)
		return
	}

	var req updateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	in := UpdateStaffInput{
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

	detail, err := h.service.UpdateStaff(c.Request.Context(), act, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"staff": toStaffDTO(detail)})
}

type setAssignmentRequest struct {
	DepartmentID *string `json:"department_id"`
	PositionID   *string `json:"position_id"`
	HiredDate    *string `json:"hired_date"`
}

// SetAssignment handles PUT /staff/:id/assignment. Full-replace semantics:
// every field is sent explicitly, and null (or an absent key) unassigns.
func (h *Handler) SetAssignment(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid staff id.", nil)
		return
	}

	var req setAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body.", nil)
		return
	}

	in := SetAssignmentInput{}
	if req.DepartmentID != nil {
		depID, ok := parseUUIDParam(c, *req.DepartmentID, "department_id")
		if !ok {
			return
		}
		in.DepartmentID = &depID
	}
	if req.PositionID != nil {
		posID, ok := parseUUIDParam(c, *req.PositionID, "position_id")
		if !ok {
			return
		}
		in.PositionID = &posID
	}
	if req.HiredDate != nil {
		t, ok := parseDateParam(c, *req.HiredDate, "hired_date")
		if !ok {
			return
		}
		in.HiredDate = &t
	}

	detail, err := h.service.SetAssignment(c.Request.Context(), act, id, in)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"staff": toStaffDTO(detail)})
}

type setStatusRequest struct {
	EmploymentStatus string `json:"employment_status" binding:"required"`
}

// SetEmploymentStatus handles PUT /staff/:id/status (activate/deactivate,
// suspend, leave — section 12). This route carries the staff.deactivate
// permission, stricter than staff.update.
func (h *Handler) SetEmploymentStatus(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid staff id.", nil)
		return
	}

	var req setStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "employment_status is required.", nil)
		return
	}

	detail, err := h.service.SetEmploymentStatus(c.Request.Context(), act, id, req.EmploymentStatus)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"staff": toStaffDTO(detail)})
}

type createAccountRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Roles    []string `json:"roles" binding:"min=1,dive,required"`
}

// CreateAccount handles POST /staff/:id/account — create the login account
// with its initial system roles (section 12). The staff member must not
// already have an account; the email must be unused.
func (h *Handler) CreateAccount(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid staff id.", nil)
		return
	}

	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "email, a password of at least 8 characters, and at least one role are required.", nil)
		return
	}

	account, err := h.service.CreateAccount(c.Request.Context(), act, id, CreateAccountInput{
		Email:     req.Email,
		Password:  req.Password,
		RoleNames: req.Roles,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"account": toAccountDTO(&account)})
}

type setAccountRolesRequest struct {
	Roles []string `json:"roles" binding:"dive,required"`
}

// SetAccountRoles handles PUT /staff/:id/account/roles — replace the
// account's roles wholesale (an empty list removes all roles, which is a
// legitimate way to lock a member out without touching account status).
func (h *Handler) SetAccountRoles(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid staff id.", nil)
		return
	}

	var req setAccountRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "roles must be a list of role names.", nil)
		return
	}

	roles, err := h.service.SetAccountRoles(c.Request.Context(), act, id, req.Roles)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}
