// Package announcements implements the announcements domain (M13):
// institutional communication with school-wide broadcasts and targeted
// messages (by role or by class cohort), draft/published/archived status
// flows, audience resolution, and transactional audit logging.
package announcements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
	"github.com/thunderlight247/schoolhub/backend/internal/email"
)

// Target types.
const (
	TargetAll   = "ALL"
	TargetRole  = "ROLE"
	TargetClass = "CLASS"
)

// Announcement statuses.
const (
	StatusDraft     = "DRAFT"
	StatusPublished = "PUBLISHED"
	StatusArchived  = "ARCHIVED"
)

// Announcement priorities.
const (
	PriorityLow    = "LOW"
	PriorityNormal = "NORMAL"
	PriorityHigh   = "HIGH"
	PriorityUrgent = "URGENT"
)

var ValidRoles = map[string]bool{
	"ADMIN":   true,
	"TEACHER": true,
	"STAFF":   true,
	"PARENT":  true,
	"STUDENT": true,
}

var (
	ErrAnnouncementNotFound = errors.New("announcement not found")
	ErrTitleRequired        = errors.New("title is required")
	ErrContentRequired      = errors.New("content is required")
	ErrInvalidTargetType    = errors.New("invalid target type")
	ErrInvalidTargetRole    = errors.New("invalid target role")
	ErrInvalidPriority      = errors.New("invalid priority")
	ErrInvalidStatus        = errors.New("invalid status")
	ErrClassNotFound        = errors.New("target class not found")
	ErrUnauthorizedAction   = errors.New("not authorized to modify this announcement")
)

type ListFilter struct {
	Status        *string
	TargetType    *string
	TargetRole    *string
	TargetClassID *uuid.UUID
	AuthorID      *uuid.UUID
	Search        *string
	Page          int
	PageSize      int
}

type CreateAnnouncementInput struct {
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	TargetType    string     `json:"target_type"`
	TargetRole    *string    `json:"target_role,omitempty"`
	TargetClassID *uuid.UUID `json:"target_class_id,omitempty"`
	Priority      string     `json:"priority,omitempty"`
	Status        string     `json:"status,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

type UpdateAnnouncementInput struct {
	Title         *string    `json:"title,omitempty"`
	Content       *string    `json:"content,omitempty"`
	TargetType    *string    `json:"target_type,omitempty"`
	TargetRole    *string    `json:"target_role,omitempty"`
	TargetClassID *uuid.UUID `json:"target_class_id,omitempty"`
	Priority      *string    `json:"priority,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

type Service struct {
	pool *pgxpool.Pool
	// mailer is optional (nil by default): when an email provider is
	// configured, publishing a school-wide announcement notifies staff by
	// email on a best-effort basis after commit.
	mailer *email.Service
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// SetMailer attaches an optional email service used for publish
// notifications. Safe to call once during wiring.
func (s *Service) SetMailer(mailer *email.Service) {
	s.mailer = mailer
}

func (s *Service) CreateAnnouncement(ctx context.Context, authorID uuid.UUID, in CreateAnnouncementInput) (db.GetAnnouncementByIDRow, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return db.GetAnnouncementByIDRow{}, ErrTitleRequired
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return db.GetAnnouncementByIDRow{}, ErrContentRequired
	}

	targetType := strings.ToUpper(strings.TrimSpace(in.TargetType))
	if targetType == "" {
		targetType = TargetAll
	}
	if targetType != TargetAll && targetType != TargetRole && targetType != TargetClass {
		return db.GetAnnouncementByIDRow{}, ErrInvalidTargetType
	}

	var targetRole pgtype.Text
	var targetClassID pgtype.UUID

	switch targetType {
	case TargetAll:
		targetRole = pgtype.Text{Valid: false}
		targetClassID = pgtype.UUID{Valid: false}
	case TargetRole:
		if in.TargetRole == nil || strings.TrimSpace(*in.TargetRole) == "" {
			return db.GetAnnouncementByIDRow{}, ErrInvalidTargetRole
		}
		role := strings.ToUpper(strings.TrimSpace(*in.TargetRole))
		if !ValidRoles[role] {
			return db.GetAnnouncementByIDRow{}, ErrInvalidTargetRole
		}
		targetRole = pgtype.Text{String: role, Valid: true}
		targetClassID = pgtype.UUID{Valid: false}
	case TargetClass:
		if in.TargetClassID == nil || *in.TargetClassID == uuid.Nil {
			return db.GetAnnouncementByIDRow{}, ErrClassNotFound
		}
		targetClassID = pgtype.UUID{Bytes: *in.TargetClassID, Valid: true}
		targetRole = pgtype.Text{Valid: false}
	}

	priority := strings.ToUpper(strings.TrimSpace(in.Priority))
	if priority == "" {
		priority = PriorityNormal
	}
	if priority != PriorityLow && priority != PriorityNormal && priority != PriorityHigh && priority != PriorityUrgent {
		return db.GetAnnouncementByIDRow{}, ErrInvalidPriority
	}

	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = StatusDraft
	}
	if status != StatusDraft && status != StatusPublished && status != StatusArchived {
		return db.GetAnnouncementByIDRow{}, ErrInvalidStatus
	}

	var publishedAt pgtype.Timestamptz
	if status == StatusPublished {
		publishedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}

	var expiresAt pgtype.Timestamptz
	if in.ExpiresAt != nil && !in.ExpiresAt.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: in.ExpiresAt.UTC(), Valid: true}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)

	if targetType == TargetClass {
		if _, err := q.GetClassByID(ctx, *in.TargetClassID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.GetAnnouncementByIDRow{}, ErrClassNotFound
			}
			return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: check class: %w", err)
		}
	}

	created, err := q.CreateAnnouncement(ctx, db.CreateAnnouncementParams{
		Title:         title,
		Content:       content,
		AuthorID:      authorID,
		Status:        status,
		TargetType:    targetType,
		TargetRole:    targetRole,
		TargetClassID: targetClassID,
		Priority:      priority,
		PublishedAt:   publishedAt,
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: insert: %w", err)
	}

	metadata := map[string]any{
		"title":       title,
		"status":      status,
		"target_type": targetType,
		"priority":    priority,
	}
	if targetRole.Valid {
		metadata["target_role"] = targetRole.String
	}
	if targetClassID.Valid {
		metadata["target_class_id"] = in.TargetClassID.String()
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  authorID,
		Action:       audit.ActionAnnouncementCreated,
		ResourceType: audit.ResourceAnnouncement,
		ResourceID:   created.ID,
		Metadata:     metadata,
	}); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: commit: %w", err)
	}

	return s.GetAnnouncement(ctx, created.ID)
}

func (s *Service) GetAnnouncement(ctx context.Context, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
	q := db.New(s.pool)
	row, err := q.GetAnnouncementByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAnnouncementByIDRow{}, ErrAnnouncementNotFound
		}
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: get by id: %w", err)
	}
	return row, nil
}

func (s *Service) ListAnnouncements(ctx context.Context, f ListFilter) ([]db.ListAnnouncementsRow, int64, error) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var statusParam pgtype.Text
	if f.Status != nil && strings.TrimSpace(*f.Status) != "" {
		statusParam = pgtype.Text{String: strings.ToUpper(strings.TrimSpace(*f.Status)), Valid: true}
	}
	var targetTypeParam pgtype.Text
	if f.TargetType != nil && strings.TrimSpace(*f.TargetType) != "" {
		targetTypeParam = pgtype.Text{String: strings.ToUpper(strings.TrimSpace(*f.TargetType)), Valid: true}
	}
	var targetRoleParam pgtype.Text
	if f.TargetRole != nil && strings.TrimSpace(*f.TargetRole) != "" {
		targetRoleParam = pgtype.Text{String: strings.ToUpper(strings.TrimSpace(*f.TargetRole)), Valid: true}
	}
	var targetClassParam pgtype.UUID
	if f.TargetClassID != nil && *f.TargetClassID != uuid.Nil {
		targetClassParam = pgtype.UUID{Bytes: *f.TargetClassID, Valid: true}
	}
	var authorParam pgtype.UUID
	if f.AuthorID != nil && *f.AuthorID != uuid.Nil {
		authorParam = pgtype.UUID{Bytes: *f.AuthorID, Valid: true}
	}
	var searchParam pgtype.Text
	if f.Search != nil && strings.TrimSpace(*f.Search) != "" {
		searchParam = pgtype.Text{String: strings.TrimSpace(*f.Search), Valid: true}
	}

	q := db.New(s.pool)

	total, err := q.CountAnnouncements(ctx, db.CountAnnouncementsParams{
		Status:        statusParam,
		TargetType:    targetTypeParam,
		TargetRole:    targetRoleParam,
		TargetClassID: targetClassParam,
		AuthorID:      authorParam,
		Search:        searchParam,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("announcements: count: %w", err)
	}

	rows, err := q.ListAnnouncements(ctx, db.ListAnnouncementsParams{
		Limit:         int32(pageSize),
		Offset:        int32(offset),
		Status:        statusParam,
		TargetType:    targetTypeParam,
		TargetRole:    targetRoleParam,
		TargetClassID: targetClassParam,
		AuthorID:      authorParam,
		Search:        searchParam,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("announcements: list: %w", err)
	}

	return rows, total, nil
}

func (s *Service) ListFeedAnnouncements(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]db.ListFeedAnnouncementsRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	q := db.New(s.pool)

	total, err := q.CountFeedAnnouncements(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("announcements: count feed: %w", err)
	}

	rows, err := q.ListFeedAnnouncements(ctx, db.ListFeedAnnouncementsParams{
		AuthorID: userID,
		Limit:    int32(pageSize),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("announcements: list feed: %w", err)
	}

	return rows, total, nil
}

func (s *Service) UpdateAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID, in UpdateAnnouncementInput) (db.GetAnnouncementByIDRow, error) {
	existing, err := s.GetAnnouncement(ctx, id)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, err
	}

	if !isAdmin && existing.AuthorID != actorID {
		return db.GetAnnouncementByIDRow{}, ErrUnauthorizedAction
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)

	var titleParam pgtype.Text
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return db.GetAnnouncementByIDRow{}, ErrTitleRequired
		}
		titleParam = pgtype.Text{String: t, Valid: true}
	}

	var contentParam pgtype.Text
	if in.Content != nil {
		c := strings.TrimSpace(*in.Content)
		if c == "" {
			return db.GetAnnouncementByIDRow{}, ErrContentRequired
		}
		contentParam = pgtype.Text{String: c, Valid: true}
	}

	targetType := existing.TargetType
	if in.TargetType != nil {
		targetType = strings.ToUpper(strings.TrimSpace(*in.TargetType))
		if targetType != TargetAll && targetType != TargetRole && targetType != TargetClass {
			return db.GetAnnouncementByIDRow{}, ErrInvalidTargetType
		}
	}

	var targetRoleParam pgtype.Text
	var targetClassParam pgtype.UUID

	switch targetType {
	case TargetAll:
		targetRoleParam = pgtype.Text{Valid: false}
		targetClassParam = pgtype.UUID{Valid: false}
	case TargetRole:
		role := ""
		if in.TargetRole != nil {
			role = strings.ToUpper(strings.TrimSpace(*in.TargetRole))
		} else if existing.TargetRole.Valid {
			role = existing.TargetRole.String
		}
		if role == "" || !ValidRoles[role] {
			return db.GetAnnouncementByIDRow{}, ErrInvalidTargetRole
		}
		targetRoleParam = pgtype.Text{String: role, Valid: true}
		targetClassParam = pgtype.UUID{Valid: false}
	case TargetClass:
		var classID uuid.UUID
		if in.TargetClassID != nil {
			classID = *in.TargetClassID
		} else if existing.TargetClassID.Valid {
			classID = existing.TargetClassID.Bytes
		}
		if classID == uuid.Nil {
			return db.GetAnnouncementByIDRow{}, ErrClassNotFound
		}
		if _, err := q.GetClassByID(ctx, classID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.GetAnnouncementByIDRow{}, ErrClassNotFound
			}
			return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: check class: %w", err)
		}
		targetClassParam = pgtype.UUID{Bytes: classID, Valid: true}
		targetRoleParam = pgtype.Text{Valid: false}
	}

	var priorityParam pgtype.Text
	if in.Priority != nil {
		p := strings.ToUpper(strings.TrimSpace(*in.Priority))
		if p != PriorityLow && p != PriorityNormal && p != PriorityHigh && p != PriorityUrgent {
			return db.GetAnnouncementByIDRow{}, ErrInvalidPriority
		}
		priorityParam = pgtype.Text{String: p, Valid: true}
	}

	var expiresAtParam pgtype.Timestamptz
	if in.ExpiresAt != nil {
		if !in.ExpiresAt.IsZero() {
			expiresAtParam = pgtype.Timestamptz{Time: in.ExpiresAt.UTC(), Valid: true}
		} else {
			expiresAtParam = pgtype.Timestamptz{Valid: false}
		}
	} else {
		expiresAtParam = existing.ExpiresAt
	}

	updated, err := q.UpdateAnnouncement(ctx, db.UpdateAnnouncementParams{
		ID:            id,
		Title:         titleParam,
		Content:       contentParam,
		TargetType:    pgtype.Text{String: targetType, Valid: true},
		TargetRole:    targetRoleParam,
		TargetClassID: targetClassParam,
		Priority:      priorityParam,
		ExpiresAt:     expiresAtParam,
	})
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: update: %w", err)
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAnnouncementUpdated,
		ResourceType: audit.ResourceAnnouncement,
		ResourceID:   id,
		Metadata: map[string]any{
			"updated_fields": in,
		},
	}); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: commit: %w", err)
	}

	return s.GetAnnouncement(ctx, updated.ID)
}

func (s *Service) PublishAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
	existing, err := s.GetAnnouncement(ctx, id)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, err
	}

	if !isAdmin && existing.AuthorID != actorID {
		return db.GetAnnouncementByIDRow{}, ErrUnauthorizedAction
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)

	now := time.Now().UTC()
	published, err := q.UpdateAnnouncementStatus(ctx, db.UpdateAnnouncementStatusParams{
		ID:          id,
		Status:      StatusPublished,
		PublishedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: publish: %w", err)
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAnnouncementPublished,
		ResourceType: audit.ResourceAnnouncement,
		ResourceID:   id,
		Metadata: map[string]any{
			"published_at": now.Format(time.RFC3339),
			"previous":     existing.Status,
		},
	}); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: commit: %w", err)
	}

	// Best-effort email notification for school-wide announcements (after
	// commit; never blocks or fails the publish itself). No-op when no
	// provider is configured or the announcement is role/class-targeted.
	if s.mailer != nil && s.mailer.Enabled() && published.TargetType == TargetAll {
		title := published.Title
		body := ""
		if published.Content != "" {
			body = published.Content
			if len(body) > 400 {
				body = body[:400]
			}
		}
		go func() {
			mailCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			recipients, err := s.pool.Query(mailCtx,
				`SELECT DISTINCT u.email FROM users u
				 JOIN user_roles ur ON ur.user_id = u.id
				 JOIN roles r ON r.id = ur.role_id
				 WHERE u.status = 'ACTIVE' AND r.name IN ('ADMIN','TEACHER','STAFF')`)
			if err != nil {
				return
			}
			defer recipients.Close()
			for recipients.Next() {
				var to string
				if err := recipients.Scan(&to); err != nil {
					continue
				}
				_ = s.mailer.SendAnnouncement(mailCtx, to, title, body)
			}
		}()
	}

	return s.GetAnnouncement(ctx, published.ID)
}

func (s *Service) ArchiveAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
	existing, err := s.GetAnnouncement(ctx, id)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, err
	}

	if !isAdmin && existing.AuthorID != actorID {
		return db.GetAnnouncementByIDRow{}, ErrUnauthorizedAction
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)

	archived, err := q.UpdateAnnouncementStatus(ctx, db.UpdateAnnouncementStatusParams{
		ID:          id,
		Status:      StatusArchived,
		PublishedAt: existing.PublishedAt,
	})
	if err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: archive: %w", err)
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAnnouncementArchived,
		ResourceType: audit.ResourceAnnouncement,
		ResourceID:   id,
		Metadata: map[string]any{
			"archived_at": time.Now().UTC().Format(time.RFC3339),
			"previous":    existing.Status,
		},
	}); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAnnouncementByIDRow{}, fmt.Errorf("announcements: commit: %w", err)
	}

	return s.GetAnnouncement(ctx, archived.ID)
}

func (s *Service) DeleteAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) error {
	existing, err := s.GetAnnouncement(ctx, id)
	if err != nil {
		return err
	}

	if !isAdmin && existing.AuthorID != actorID {
		return ErrUnauthorizedAction
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("announcements: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)

	if err := q.DeleteAnnouncement(ctx, id); err != nil {
		return fmt.Errorf("announcements: delete: %w", err)
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAnnouncementDeleted,
		ResourceType: audit.ResourceAnnouncement,
		ResourceID:   id,
		Metadata: map[string]any{
			"title":  existing.Title,
			"status": existing.Status,
		},
	}); err != nil {
		return fmt.Errorf("announcements: audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// UnmarshalJSON helper for CreateAnnouncementInput to handle optional string/times cleanly.
func (c *CreateAnnouncementInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		Title         string  `json:"title"`
		Content       string  `json:"content"`
		TargetType    string  `json:"target_type"`
		TargetRole    *string `json:"target_role"`
		TargetClassID *string `json:"target_class_id"`
		Priority      string  `json:"priority"`
		Status        string  `json:"status"`
		ExpiresAt     *string `json:"expires_at"`
	}
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.Title = aux.Title
	c.Content = aux.Content
	c.TargetType = aux.TargetType
	c.TargetRole = aux.TargetRole
	c.Priority = aux.Priority
	c.Status = aux.Status

	if aux.TargetClassID != nil && *aux.TargetClassID != "" {
		id, err := uuid.Parse(*aux.TargetClassID)
		if err != nil {
			return fmt.Errorf("invalid target_class_id UUID: %w", err)
		}
		c.TargetClassID = &id
	}

	if aux.ExpiresAt != nil && *aux.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *aux.ExpiresAt)
		if err != nil {
			// Also support simple date YYYY-MM-DD
			t, err = time.Parse("2006-01-02", *aux.ExpiresAt)
			if err != nil {
				return fmt.Errorf("invalid expires_at date: %w", err)
			}
		}
		c.ExpiresAt = &t
	}
	return nil
}

// UnmarshalJSON helper for UpdateAnnouncementInput.
func (u *UpdateAnnouncementInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		Title         *string `json:"title"`
		Content       *string `json:"content"`
		TargetType    *string `json:"target_type"`
		TargetRole    *string `json:"target_role"`
		TargetClassID *string `json:"target_class_id"`
		Priority      *string `json:"priority"`
		ExpiresAt     *string `json:"expires_at"`
	}
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.Title = aux.Title
	u.Content = aux.Content
	u.TargetType = aux.TargetType
	u.TargetRole = aux.TargetRole
	u.Priority = aux.Priority

	if aux.TargetClassID != nil {
		if *aux.TargetClassID == "" {
			nilUUID := uuid.Nil
			u.TargetClassID = &nilUUID
		} else {
			id, err := uuid.Parse(*aux.TargetClassID)
			if err != nil {
				return fmt.Errorf("invalid target_class_id UUID: %w", err)
			}
			u.TargetClassID = &id
		}
	}

	if aux.ExpiresAt != nil {
		if *aux.ExpiresAt == "" {
			zeroTime := time.Time{}
			u.ExpiresAt = &zeroTime
		} else {
			t, err := time.Parse(time.RFC3339, *aux.ExpiresAt)
			if err != nil {
				t, err = time.Parse("2006-01-02", *aux.ExpiresAt)
				if err != nil {
					return fmt.Errorf("invalid expires_at date: %w", err)
				}
			}
			u.ExpiresAt = &t
		}
	}
	return nil
}
