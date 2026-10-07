// Package announcements HTTP handlers (M13). Routes are wired in
// cmd/server/main.go on the dedicated announcement.* permission codes:
//   - announcement.view: list/get announcements (and audience feeds)
//   - announcement.create: create announcement
//   - announcement.update: update / publish / archive / delete announcement
package announcements

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

type ServiceAPI interface {
	CreateAnnouncement(ctx context.Context, authorID uuid.UUID, in CreateAnnouncementInput) (db.GetAnnouncementByIDRow, error)
	GetAnnouncement(ctx context.Context, id uuid.UUID) (db.GetAnnouncementByIDRow, error)
	ListAnnouncements(ctx context.Context, f ListFilter) ([]db.ListAnnouncementsRow, int64, error)
	ListFeedAnnouncements(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]db.ListFeedAnnouncementsRow, int64, error)
	UpdateAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID, in UpdateAnnouncementInput) (db.GetAnnouncementByIDRow, error)
	PublishAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error)
	ArchiveAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error)
	DeleteAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) error
}

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

type announcementDTO struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Content         string  `json:"content"`
	AuthorID        string  `json:"author_id"`
	AuthorEmail     string  `json:"author_email"`
	AuthorName      string  `json:"author_name"`
	Status          string  `json:"status"`
	TargetType      string  `json:"target_type"`
	TargetRole      *string `json:"target_role,omitempty"`
	TargetClassID   *string `json:"target_class_id,omitempty"`
	TargetClassName *string `json:"target_class_name,omitempty"`
	Priority        string  `json:"priority"`
	PublishedAt     *string `json:"published_at,omitempty"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func timePtrString(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func uuidPtrString(v pgtype.UUID) *string {
	if !v.Valid {
		return nil
	}
	s := uuid.UUID(v.Bytes).String()
	return &s
}

func toAnnouncementDTO(row db.GetAnnouncementByIDRow) announcementDTO {
	authorName := strings.TrimSpace(row.AuthorFirstName + " " + row.AuthorLastName)
	return announcementDTO{
		ID:              row.ID.String(),
		Title:           row.Title,
		Content:         row.Content,
		AuthorID:        row.AuthorID.String(),
		AuthorEmail:     row.AuthorEmail,
		AuthorName:      authorName,
		Status:          row.Status,
		TargetType:      row.TargetType,
		TargetRole:      textPtr(row.TargetRole),
		TargetClassID:   uuidPtrString(row.TargetClassID),
		TargetClassName: textPtr(row.TargetClassName),
		Priority:        row.Priority,
		PublishedAt:     timePtrString(row.PublishedAt),
		ExpiresAt:       timePtrString(row.ExpiresAt),
		CreatedAt:       row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toAnnouncementListItemDTO(row db.ListAnnouncementsRow) announcementDTO {
	authorName := strings.TrimSpace(row.AuthorFirstName + " " + row.AuthorLastName)
	return announcementDTO{
		ID:              row.ID.String(),
		Title:           row.Title,
		Content:         row.Content,
		AuthorID:        row.AuthorID.String(),
		AuthorEmail:     row.AuthorEmail,
		AuthorName:      authorName,
		Status:          row.Status,
		TargetType:      row.TargetType,
		TargetRole:      textPtr(row.TargetRole),
		TargetClassID:   uuidPtrString(row.TargetClassID),
		TargetClassName: textPtr(row.TargetClassName),
		Priority:        row.Priority,
		PublishedAt:     timePtrString(row.PublishedAt),
		ExpiresAt:       timePtrString(row.ExpiresAt),
		CreatedAt:       row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toAnnouncementFeedItemDTO(row db.ListFeedAnnouncementsRow) announcementDTO {
	authorName := strings.TrimSpace(row.AuthorFirstName + " " + row.AuthorLastName)
	return announcementDTO{
		ID:              row.ID.String(),
		Title:           row.Title,
		Content:         row.Content,
		AuthorID:        row.AuthorID.String(),
		AuthorEmail:     row.AuthorEmail,
		AuthorName:      authorName,
		Status:          row.Status,
		TargetType:      row.TargetType,
		TargetRole:      textPtr(row.TargetRole),
		TargetClassID:   uuidPtrString(row.TargetClassID),
		TargetClassName: textPtr(row.TargetClassName),
		Priority:        row.Priority,
		PublishedAt:     timePtrString(row.PublishedAt),
		ExpiresAt:       timePtrString(row.ExpiresAt),
		CreatedAt:       row.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       row.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func parsePageParam(c *gin.Context, key string, fallback int) int {
	value := c.DefaultQuery(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func parseUUIDFromQuery(c *gin.Context, key string) (*uuid.UUID, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a valid UUID.", nil)
		return nil, false
	}
	return &id, true
}

func parseUUIDParam(c *gin.Context, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(key))
	if err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", key+" must be a valid UUID.", nil)
		return uuid.Nil, false
	}
	return id, true
}

func hasRole(u auth.AuthenticatedUser, role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

func (h *Handler) ListAnnouncements(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	page := parsePageParam(c, "page", 1)
	pageSize := parsePageParam(c, "page_size", 20)

	feedParam := strings.ToLower(strings.TrimSpace(c.Query("feed")))
	isAdmin := hasRole(authUser, "ADMIN")

	// If feed=true or user is non-admin without query filters, return the audience feed
	if feedParam == "true" || feedParam == "1" || (!isAdmin && c.Query("all") != "true" && c.Query("status") == "") {
		rows, total, err := h.service.ListFeedAnnouncements(c.Request.Context(), authUser.User.ID, page, pageSize)
		if err != nil {
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list announcements feed.", nil)
			return
		}
		data := make([]announcementDTO, 0, len(rows))
		for _, r := range rows {
			data = append(data, toAnnouncementFeedItemDTO(r))
		}
		c.JSON(http.StatusOK, gin.H{
			"data": data,
			"meta": gin.H{
				"page":      page,
				"page_size": pageSize,
				"total":     total,
			},
		})
		return
	}

	filter := ListFilter{
		Page:     page,
		PageSize: pageSize,
	}

	if s := strings.TrimSpace(c.Query("status")); s != "" {
		filter.Status = &s
	}
	if tt := strings.TrimSpace(c.Query("target_type")); tt != "" {
		filter.TargetType = &tt
	}
	if tr := strings.TrimSpace(c.Query("target_role")); tr != "" {
		filter.TargetRole = &tr
	}
	if cid, ok := parseUUIDFromQuery(c, "class_id"); ok && cid != nil {
		filter.TargetClassID = cid
	} else if !ok {
		return
	}
	if aid, ok := parseUUIDFromQuery(c, "author_id"); ok && aid != nil {
		filter.AuthorID = aid
	} else if !ok {
		return
	}
	if q := strings.TrimSpace(c.Query("search")); q != "" {
		filter.Search = &q
	}

	rows, total, err := h.service.ListAnnouncements(c.Request.Context(), filter)
	if err != nil {
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list announcements.", nil)
		return
	}

	data := make([]announcementDTO, 0, len(rows))
	for _, r := range rows {
		data = append(data, toAnnouncementListItemDTO(r))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		},
	})
}

func (h *Handler) GetAnnouncement(c *gin.Context) {
	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	row, err := h.service.GetAnnouncement(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrAnnouncementNotFound) {
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Announcement not found.", nil)
			return
		}
		apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve announcement.", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toAnnouncementDTO(row)})
}

func (h *Handler) CreateAnnouncement(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	var in CreateAnnouncementInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid announcement payload: "+err.Error(), nil)
		return
	}

	created, err := h.service.CreateAnnouncement(c.Request.Context(), authUser.User.ID, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrTitleRequired),
			errors.Is(err, ErrContentRequired),
			errors.Is(err, ErrInvalidTargetType),
			errors.Is(err, ErrInvalidTargetRole),
			errors.Is(err, ErrInvalidPriority),
			errors.Is(err, ErrInvalidStatus):
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, ErrClassNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create announcement.", nil)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": toAnnouncementDTO(created)})
}

func (h *Handler) UpdateAnnouncement(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var in UpdateAnnouncementInput
	if err := c.ShouldBindJSON(&in); err != nil {
		apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid update payload: "+err.Error(), nil)
		return
	}

	isAdmin := hasRole(authUser, "ADMIN")
	updated, err := h.service.UpdateAnnouncement(c.Request.Context(), authUser.User.ID, isAdmin, id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrAnnouncementNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Announcement not found.", nil)
		case errors.Is(err, ErrUnauthorizedAction):
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		case errors.Is(err, ErrTitleRequired),
			errors.Is(err, ErrContentRequired),
			errors.Is(err, ErrInvalidTargetType),
			errors.Is(err, ErrInvalidTargetRole),
			errors.Is(err, ErrInvalidPriority):
			apierror.Respond(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		case errors.Is(err, ErrClassNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update announcement.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toAnnouncementDTO(updated)})
}

func (h *Handler) PublishAnnouncement(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	isAdmin := hasRole(authUser, "ADMIN")
	published, err := h.service.PublishAnnouncement(c.Request.Context(), authUser.User.ID, isAdmin, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrAnnouncementNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Announcement not found.", nil)
		case errors.Is(err, ErrUnauthorizedAction):
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to publish announcement.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toAnnouncementDTO(published)})
}

func (h *Handler) ArchiveAnnouncement(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	isAdmin := hasRole(authUser, "ADMIN")
	archived, err := h.service.ArchiveAnnouncement(c.Request.Context(), authUser.User.ID, isAdmin, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrAnnouncementNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Announcement not found.", nil)
		case errors.Is(err, ErrUnauthorizedAction):
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive announcement.", nil)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toAnnouncementDTO(archived)})
}

func (h *Handler) DeleteAnnouncement(c *gin.Context) {
	authUser, ok := auth.CurrentUser(c)
	if !ok {
		apierror.Respond(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.", nil)
		return
	}

	id, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	isAdmin := hasRole(authUser, "ADMIN")
	err := h.service.DeleteAnnouncement(c.Request.Context(), authUser.User.ID, isAdmin, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrAnnouncementNotFound):
			apierror.Respond(c, http.StatusNotFound, "NOT_FOUND", "Announcement not found.", nil)
		case errors.Is(err, ErrUnauthorizedAction):
			apierror.Respond(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		default:
			apierror.Respond(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete announcement.", nil)
		}
		return
	}

	c.Status(http.StatusNoContent)
}
