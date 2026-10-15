// HTTP layer for the admin/account access-control endpoints (expert pass
// follow-up). Route mounting and permission gating live in
// cmd/server/main.go:
//
//	PUT  /users/:id/admin-role   (admin.manage + MASTER_ADMIN role)
//	PUT  /users/:id/status       (account.block)
//	GET  /admins                 (admin.manage)
//	GET  /accounts?role=...      (account.block)
//
// Every handler assumes auth.RequireAuth has run (audit attribution) and
// the matching RequirePermission middleware has authorized the request;
// the MASTER_ADMIN-role double-check for admin management happens in the
// service (defense in depth) and again via middleware for good UX (403s
// for plain admins without a pointless round trip).
package adminaccounts

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// ServiceAPI is what the handlers need. An interface (not *Service) so
// handler tests can stub it, matching internal/staff's convention — the
// concrete service's transactional paths need a real database.
type ServiceAPI interface {
	ListAdmins(ctx context.Context) ([]AdminView, error)
	ListAccountsByRole(ctx context.Context, roleName string) ([]AccountView, error)
	ListInvitees(ctx context.Context, inviterID uuid.UUID) ([]Invitee, error)
	BlockInvitee(ctx context.Context, inviterID, targetUserID uuid.UUID) error
	UnblockInvitee(ctx context.Context, inviterID, targetUserID uuid.UUID) error
	RevokeAdmin(ctx context.Context, actorID, targetUserID uuid.UUID) error
	Block(ctx context.Context, actorID, targetUserID uuid.UUID) error
	Unblock(ctx context.Context, actorID, targetUserID uuid.UUID) error
	CreateAdmin(ctx context.Context, actorID uuid.UUID, in CreateAdminInput) (AdminView, error)
}

// Handler exposes the access-control endpoints.
type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

// --- DTOs ---

// adminDTO is one row of GET /admins.
type adminDTO struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Status      string  `json:"status"`
	IsMaster    bool    `json:"is_master"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
}

// accountDTO is one row of GET /accounts?role=....
type accountDTO struct {
	ID          string  `json:"id"`
	PersonID    string  `json:"person_id"`
	Email       string  `json:"email"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Status      string  `json:"status"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
}

// inviteeDTO is one row of GET /my-invitees.
type inviteeDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Roles     string `json:"roles"`
	Status    string `json:"status"`
}

// ListMyInvitees handles GET /api/v1/my-invitees — every inviter (teacher
// included) sees the accounts they personally provisioned. Gated only by
// RequireAuth; the service filter (users.invited_by = caller) is the access
// control — you can only ever see your own invitees.
func (h *Handler) ListMyInvitees(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	invitees, err := h.service.ListInvitees(c.Request.Context(), actor)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	dtos := make([]inviteeDTO, 0, len(invitees))
	for _, v := range invitees {
		dtos = append(dtos, inviteeDTO{
			ID:        v.UserID.String(),
			Email:     v.Email,
			FirstName: v.FirstName,
			LastName:  v.LastName,
			Roles:     v.Roles,
			Status:    v.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{"invitees": dtos})
}

// InviteeStatus handles PUT /api/v1/my-invitees/:id/status — an inviter
// suspending/restoring an account THEY provisioned (teachers without
// account.block get their lockout power from here, strictly scoped to
// their own invitees).
func (h *Handler) InviteeStatus(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	target, ok := parseTargetID(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status is required (ACTIVE or SUSPENDED).", nil)
		return
	}
	var svcErr error
	switch req.Status {
	case "SUSPENDED":
		svcErr = h.service.BlockInvitee(c.Request.Context(), actor, target)
	case "ACTIVE":
		svcErr = h.service.UnblockInvitee(c.Request.Context(), actor, target)
	default:
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status must be ACTIVE or SUSPENDED.", nil)
		return
	}
	if svcErr != nil {
		respondServiceError(c, svcErr)
		return
	}
	c.Status(http.StatusNoContent)
}

// respondServiceError maps service sentinel errors to structured
// responses; anything unmapped stays a details-free 500.
func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Account not found.", nil)
	case errors.Is(err, ErrNotAdmin):
		apierror.Respond(c, http.StatusConflict, "NOT_AN_ADMIN", "That account holds no admin role.", nil)
	case errors.Is(err, ErrLastAdmin):
		apierror.Respond(c, http.StatusConflict, "LAST_ADMIN", "Refusing to remove the last active admin.", nil)
	case errors.Is(err, ErrSelfDemotion):
		apierror.Respond(c, http.StatusConflict, "SELF_DEMOTION", "You cannot revoke your own admin access.", nil)
	case errors.Is(err, ErrSelfBlock):
		apierror.Respond(c, http.StatusConflict, "SELF_BLOCK", "You cannot block your own account.", nil)
	case errors.Is(err, ErrAdminNotBlockable):
		apierror.Respond(c, http.StatusConflict, "ADMIN_NOT_BLOCKABLE", "Admin accounts can only be suspended by a master admin or the account's inviter.", nil)
	case errors.Is(err, ErrNotYourInvitee):
		apierror.Respond(c, http.StatusForbidden, "NOT_YOUR_INVITEE", "You can only manage accounts you invited.", nil)
	case errors.Is(err, ErrLastMaster):
		apierror.Respond(c, http.StatusConflict, "LAST_MASTER", "Refusing to suspend the last active master admin.", nil)
	case errors.Is(err, ErrUnknownRole):
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Unknown role.", nil)
	case errors.Is(err, ErrEmailTaken):
		apierror.Respond(c, http.StatusConflict, "EMAIL_TAKEN", "That email already belongs to an account.", nil)
	default:
		// Never leave a 500 without a server-side trace — operators need
		// the underlying cause in the logs.
		log.Printf("adminaccounts: unmapped service error: %v", err)
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not complete the access-control operation.", nil)
	}
}

// requireActor returns the authenticated user's id, or responds 401 and
// returns ok=false (mirroring internal/staff's helper).
func requireActor(c *gin.Context) (uuid.UUID, bool) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return uuid.Nil, false
	}
	return authUser.User.ID, true
}

// --- reads ---

// ListAdmins handles GET /api/v1/admins (admin.manage; effectively
// master-admin-only because that's who holds the permission — the
// middleware gates the route).
func (h *Handler) ListAdmins(c *gin.Context) {
	views, err := h.service.ListAdmins(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}

	dtos := make([]adminDTO, 0, len(views))
	for _, v := range views {
		var lastLogin *string
		if v.LastLoginAt != nil {
			s := v.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
			lastLogin = &s
		}
		dtos = append(dtos, adminDTO{
			ID:          v.UserID.String(),
			Email:       v.Email,
			FirstName:   v.FirstName,
			LastName:    v.LastName,
			Status:      v.Status,
			IsMaster:    v.IsMaster,
			LastLoginAt: lastLogin,
		})
	}

	c.JSON(http.StatusOK, gin.H{"admins": dtos})
}

// ListAccountsByRole handles GET /api/v1/accounts?role=TEACHER etc.
func (h *Handler) ListAccountsByRole(c *gin.Context) {
	role := c.Query("role")
	if role != "TEACHER" && role != "STAFF" && role != "PARENT" && role != "STUDENT" {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "role must be one of TEACHER, STAFF, PARENT, STUDENT.", nil)
		return
	}

	views, err := h.service.ListAccountsByRole(c.Request.Context(), role)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	dtos := make([]accountDTO, 0, len(views))
	for _, v := range views {
		var lastLogin *string
		if v.LastLoginAt != nil {
			s := v.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
			lastLogin = &s
		}
		dtos = append(dtos, accountDTO{
			ID:          v.UserID.String(),
			PersonID:    v.PersonID.String(),
			Email:       v.Email,
			FirstName:   v.FirstName,
			LastName:    v.LastName,
			Status:      v.Status,
			LastLoginAt: lastLogin,
		})
	}

	c.JSON(http.StatusOK, gin.H{"role": role, "accounts": dtos})
}

// --- mutations ---

// Parse target id shared by both mutation handlers.
func parseTargetID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid user id.", nil)
		return uuid.Nil, false
	}
	return id, true
}

// RevokeAdmin handles PUT /api/v1/users/:id/admin-role (admin.manage).
// The MASTER_ADMIN middleware in main.go plus the service's own checks
// mean a successful call always came from a master admin acting on
// someone else.
func (h *Handler) RevokeAdmin(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	target, ok := parseTargetID(c)
	if !ok {
		return
	}

	if err := h.service.RevokeAdmin(c.Request.Context(), actor, target); err != nil {
		respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateAdmin handles POST /api/v1/admins (M34, MASTER_ADMIN only): adds a
// new admin account (person + user + ADMIN role), sends the Clerk
// invitation best-effort, and returns the row the admins list would show.
func (h *Handler) CreateAdmin(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}

	var req struct {
		Email     string `json:"email" binding:"required,email"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"email, first_name and last_name are required (email must be valid).", nil)
		return
	}

	view, err := h.service.CreateAdmin(c.Request.Context(), actor, CreateAdminInput{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}
	// Same DTO shape as GET /admins' row so the frontend can push the new
	// admin into its list without re-listing except for last_login (nil).
	c.JSON(http.StatusCreated, gin.H{"admin": adminDTO{
		ID:        view.UserID.String(),
		Email:     view.Email,
		FirstName: view.FirstName,
		LastName:  view.LastName,
		Status:    view.Status,
		IsMaster:  view.IsMaster,
	}})
}

// Block handles PUT /api/v1/users/:id/status (account.block).
func (h *Handler) Block(c *gin.Context) {
	actor, ok := requireActor(c)
	if !ok {
		return
	}
	target, ok := parseTargetID(c)
	if !ok {
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status is required (ACTIVE or SUSPENDED).", nil)
		return
	}

	var svcErr error
	switch req.Status {
	case "SUSPENDED":
		svcErr = h.service.Block(c.Request.Context(), actor, target)
	case "ACTIVE":
		svcErr = h.service.Unblock(c.Request.Context(), actor, target)
	default:
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "status must be ACTIVE or SUSPENDED.", nil)
		return
	}
	if svcErr != nil {
		respondServiceError(c, svcErr)
		return
	}
	c.Status(http.StatusNoContent)
}
