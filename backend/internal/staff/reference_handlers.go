package staff

import (
	"net/http"

	"github.com/gin-gonic/gin"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
)

// Reference-data endpoints for the staff domain: departments and staff
// positions (section 12's assignment targets). Reading them needs
// staff.view (the dropdowns on staff screens); creating them needs
// staff.update — effectively an admin-only action under the default
// migration 000004 mapping, since no other role holds that code.

type departmentDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func toDepartmentDTO(d db.Department) departmentDTO {
	return departmentDTO{ID: d.ID.String(), Name: d.Name, Description: textPtr(d.Description)}
}

type staffPositionDTO struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Description *string `json:"description,omitempty"`
}

func toStaffPositionDTO(p db.StaffPosition) staffPositionDTO {
	return staffPositionDTO{ID: p.ID.String(), Title: p.Title, Category: p.Category, Description: textPtr(p.Description)}
}

// ListDepartments handles GET /departments.
func (h *Handler) ListDepartments(c *gin.Context) {
	departments, err := h.service.ListDepartments(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list departments.", nil)
		return
	}

	dtos := make([]departmentDTO, 0, len(departments))
	for _, d := range departments {
		dtos = append(dtos, toDepartmentDTO(d))
	}
	c.JSON(http.StatusOK, gin.H{"departments": dtos})
}

type createDepartmentRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
}

// CreateDepartment handles POST /departments.
func (h *Handler) CreateDepartment(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	var req createDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "name is required.", nil)
		return
	}

	dept, err := h.service.CreateDepartment(c.Request.Context(), act, req.Name, req.Description)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"department": toDepartmentDTO(dept)})
}

// ListStaffPositions handles GET /staff-positions.
func (h *Handler) ListStaffPositions(c *gin.Context) {
	positions, err := h.service.ListStaffPositions(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list staff positions.", nil)
		return
	}

	dtos := make([]staffPositionDTO, 0, len(positions))
	for _, p := range positions {
		dtos = append(dtos, toStaffPositionDTO(p))
	}
	c.JSON(http.StatusOK, gin.H{"staff_positions": dtos})
}

type createStaffPositionRequest struct {
	Title       string  `json:"title" binding:"required"`
	Category    string  `json:"category" binding:"required"`
	Description *string `json:"description"`
}

// CreateStaffPosition handles POST /staff-positions. An unknown category is
// a 400 from the service (the category set is business data), not a 500.
func (h *Handler) CreateStaffPosition(c *gin.Context) {
	act, ok := requireActor(c)
	if !ok {
		return
	}

	var req createStaffPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "title and category are required.", nil)
		return
	}

	position, err := h.service.CreateStaffPosition(c.Request.Context(), act, req.Title, req.Category, req.Description)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"staff_position": toStaffPositionDTO(position)})
}
