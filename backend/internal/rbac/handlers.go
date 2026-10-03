package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
)

// Handler exposes read access to roles/permissions and the ability to
// customize which permissions a role holds, on top of the default mapping
// seeded by migration 000004. All routes using this handler are mounted
// behind RequireRole(service, "ADMIN") in main.go.
type Handler struct {
	queries db.Querier
}

func NewHandler(queries db.Querier) *Handler {
	return &Handler{queries: queries}
}

type roleDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type permissionDTO struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

func toRoleDTO(r db.Role) roleDTO {
	return roleDTO{ID: r.ID.String(), Name: r.Name, Description: r.Description.String}
}

func toPermissionDTO(p db.Permission) permissionDTO {
	return permissionDTO{ID: p.ID.String(), Code: p.Code, Description: p.Description.String}
}

// ListRoles handles GET /api/v1/roles.
func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.queries.ListRoles(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list roles.", nil)
		return
	}

	dtos := make([]roleDTO, 0, len(roles))
	for _, r := range roles {
		dtos = append(dtos, toRoleDTO(r))
	}
	c.JSON(http.StatusOK, gin.H{"roles": dtos})
}

// ListPermissions handles GET /api/v1/permissions.
func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.queries.ListPermissions(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list permissions.", nil)
		return
	}

	dtos := make([]permissionDTO, 0, len(perms))
	for _, p := range perms {
		dtos = append(dtos, toPermissionDTO(p))
	}
	c.JSON(http.StatusOK, gin.H{"permissions": dtos})
}

// ListRolePermissions handles GET /api/v1/roles/:id/permissions.
func (h *Handler) ListRolePermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role id.", nil)
		return
	}

	perms, err := h.queries.ListPermissionsForRole(c.Request.Context(), roleID)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list role permissions.", nil)
		return
	}

	dtos := make([]permissionDTO, 0, len(perms))
	for _, p := range perms {
		dtos = append(dtos, toPermissionDTO(p))
	}
	c.JSON(http.StatusOK, gin.H{"permissions": dtos})
}

type assignPermissionRequest struct {
	PermissionCode string `json:"permission_code" binding:"required"`
}

// AssignPermission handles POST /api/v1/roles/:id/permissions.
// Idempotent: assigning a permission the role already has is not an error
// (the underlying query uses ON CONFLICT DO NOTHING).
func (h *Handler) AssignPermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role id.", nil)
		return
	}

	var req assignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "permission_code is required.", nil)
		return
	}

	perm, err := h.queries.GetPermissionByCode(c.Request.Context(), req.PermissionCode)
	if err != nil {
		apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Unknown permission code.", nil)
		return
	}

	if err := h.queries.AssignPermissionToRole(c.Request.Context(), db.AssignPermissionToRoleParams{
		RoleID:       roleID,
		PermissionID: perm.ID,
	}); err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not assign permission.", nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokePermission handles DELETE /api/v1/roles/:id/permissions/:permission_id.
// Also idempotent: revoking a permission the role doesn't have is not an
// error.
func (h *Handler) RevokePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role id.", nil)
		return
	}
	permissionID, err := uuid.Parse(c.Param("permission_id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid permission id.", nil)
		return
	}

	if err := h.queries.RevokePermissionFromRole(c.Request.Context(), db.RevokePermissionFromRoleParams{
		RoleID:       roleID,
		PermissionID: permissionID,
	}); err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not revoke permission.", nil)
		return
	}

	c.Status(http.StatusNoContent)
}
