package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/apierror"
)

// ServiceAPI is what the handlers need from the read service — same
// interface-for-testability pattern as every other domain's handlers.go.
type ServiceAPI interface {
	List(ctx context.Context, f ListFilter) ([]db.ListAuditLogsRow, int64, error)
	Get(ctx context.Context, id uuid.UUID) (db.GetAuditLogByIDRow, error)
	ListActions(ctx context.Context) ([]string, error)
}

// Handler exposes the audit log read endpoints (section 19/29). Route
// mounting and the audit.view gate happen in cmd/server/main.go — every
// handler here can assume both auth.RequireAuth and
// rbac.RequirePermission(..., "audit.view") already ran, so unlike some
// earlier handlers there's no redundant auth.CurrentUser check here.
type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

// --- pgtype -> JSON helpers (duplicated per package; see internal/staff's
// comment on this convention) ---

func uuidPtr(t pgtype.UUID) *string {
	if !t.Valid {
		return nil
	}
	s := uuid.UUID(t.Bytes).String()
	return &s
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func tsString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

// metadataMap decodes the stored JSONB metadata. Record (audit.go) always
// writes valid JSON (defaulting to "{}"), so a decode failure here would
// indicate the column was written by something other than Record — fail
// soft to an empty map rather than breaking the whole list response over
// one bad row.
func metadataMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	return m
}

// --- DTO ---

// entryDTO is shared by both the list and detail endpoints — ListAuditLogs
// and GetAuditLogByID select identical columns, just via two sqlc-generated
// row types.
type entryDTO struct {
	ID             string         `json:"id"`
	ActorUserID    *string        `json:"actor_user_id,omitempty"`
	ActorEmail     *string        `json:"actor_email,omitempty"`
	ActorFirstName *string        `json:"actor_first_name,omitempty"`
	ActorLastName  *string        `json:"actor_last_name,omitempty"`
	Action         string         `json:"action"`
	ResourceType   string         `json:"resource_type"`
	ResourceID     *string        `json:"resource_id,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      string         `json:"created_at"`
}

func toEntryDTO(r db.ListAuditLogsRow) entryDTO {
	return entryDTO{
		ID:             r.ID.String(),
		ActorUserID:    uuidPtr(r.ActorUserID),
		ActorEmail:     textPtr(r.ActorEmail),
		ActorFirstName: textPtr(r.ActorFirstName),
		ActorLastName:  textPtr(r.ActorLastName),
		Action:         r.Action,
		ResourceType:   r.ResourceType,
		ResourceID:     uuidPtr(r.ResourceID),
		Metadata:       metadataMap(r.Metadata),
		CreatedAt:      tsString(r.CreatedAt),
	}
}

func toEntryDetailDTO(r db.GetAuditLogByIDRow) entryDTO {
	return entryDTO{
		ID:             r.ID.String(),
		ActorUserID:    uuidPtr(r.ActorUserID),
		ActorEmail:     textPtr(r.ActorEmail),
		ActorFirstName: textPtr(r.ActorFirstName),
		ActorLastName:  textPtr(r.ActorLastName),
		Action:         r.Action,
		ResourceType:   r.ResourceType,
		ResourceID:     uuidPtr(r.ResourceID),
		Metadata:       metadataMap(r.Metadata),
		CreatedAt:      tsString(r.CreatedAt),
	}
}

// ListAuditLogs handles GET /audit-logs?page=&page_size=&actor_user_id=&action=&resource_type=&resource_id=&date_from=&date_to=
// date_from/date_to are RFC3339 timestamps (not YYYY-MM-DD like other
// domains' date filters) — audit_logs.created_at is a precise timestamptz,
// not a DATE column, so a bare date would be ambiguous about which
// timezone's midnight it means.
func (h *Handler) ListAuditLogs(c *gin.Context) {
	filter := ListFilter{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
	}

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
	if raw := c.Query("actor_user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "actor_user_id must be a UUID.", nil)
			return
		}
		filter.ActorUserID = &id
	}
	if raw := c.Query("resource_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "resource_id must be a UUID.", nil)
			return
		}
		filter.ResourceID = &id
	}
	if raw := c.Query("date_from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "date_from must be an RFC3339 timestamp.", nil)
			return
		}
		filter.DateFrom = &t
	}
	if raw := c.Query("date_to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "date_to must be an RFC3339 timestamp.", nil)
			return
		}
		filter.DateTo = &t
	}

	rows, total, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list audit logs.", nil)
		return
	}

	page, pageSize := NormalizePagination(filter.Page, filter.PageSize)
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	dtos := make([]entryDTO, 0, len(rows))
	for _, r := range rows {
		dtos = append(dtos, toEntryDTO(r))
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": dtos,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetAuditLog handles GET /audit-logs/:id.
func (h *Handler) GetAuditLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a UUID.", nil)
		return
	}

	row, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrAuditLogNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Audit log entry not found.", nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not get audit log entry.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_log": toEntryDetailDTO(row)})
}

// ListActions handles GET /audit-logs/actions — backs a filter dropdown.
func (h *Handler) ListActions(c *gin.Context) {
	actions, err := h.service.ListActions(c.Request.Context())
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not list audit actions.", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"actions": actions})
}
