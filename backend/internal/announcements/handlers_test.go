package announcements

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type stubService struct {
	ServiceAPI
	createAnnouncementFn    func(ctx context.Context, authorID uuid.UUID, in CreateAnnouncementInput) (db.GetAnnouncementByIDRow, error)
	getAnnouncementFn       func(ctx context.Context, id uuid.UUID) (db.GetAnnouncementByIDRow, error)
	listAnnouncementsFn     func(ctx context.Context, f ListFilter) ([]db.ListAnnouncementsRow, int64, error)
	listFeedAnnouncementsFn func(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]db.ListFeedAnnouncementsRow, int64, error)
	updateAnnouncementFn    func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID, in UpdateAnnouncementInput) (db.GetAnnouncementByIDRow, error)
	publishAnnouncementFn   func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error)
	archiveAnnouncementFn   func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error)
	deleteAnnouncementFn    func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) error
}

func (s *stubService) CreateAnnouncement(ctx context.Context, authorID uuid.UUID, in CreateAnnouncementInput) (db.GetAnnouncementByIDRow, error) {
	if s.createAnnouncementFn == nil {
		panic("unexpected CreateAnnouncement call")
	}
	return s.createAnnouncementFn(ctx, authorID, in)
}

func (s *stubService) GetAnnouncement(ctx context.Context, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
	if s.getAnnouncementFn == nil {
		panic("unexpected GetAnnouncement call")
	}
	return s.getAnnouncementFn(ctx, id)
}

func (s *stubService) ListAnnouncements(ctx context.Context, f ListFilter) ([]db.ListAnnouncementsRow, int64, error) {
	if s.listAnnouncementsFn == nil {
		panic("unexpected ListAnnouncements call")
	}
	return s.listAnnouncementsFn(ctx, f)
}

func (s *stubService) ListFeedAnnouncements(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]db.ListFeedAnnouncementsRow, int64, error) {
	if s.listFeedAnnouncementsFn == nil {
		panic("unexpected ListFeedAnnouncements call")
	}
	return s.listFeedAnnouncementsFn(ctx, userID, page, pageSize)
}

func (s *stubService) UpdateAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID, in UpdateAnnouncementInput) (db.GetAnnouncementByIDRow, error) {
	if s.updateAnnouncementFn == nil {
		panic("unexpected UpdateAnnouncement call")
	}
	return s.updateAnnouncementFn(ctx, actorID, isAdmin, id, in)
}

func (s *stubService) PublishAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
	if s.publishAnnouncementFn == nil {
		panic("unexpected PublishAnnouncement call")
	}
	return s.publishAnnouncementFn(ctx, actorID, isAdmin, id)
}

func (s *stubService) ArchiveAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
	if s.archiveAnnouncementFn == nil {
		panic("unexpected ArchiveAnnouncement call")
	}
	return s.archiveAnnouncementFn(ctx, actorID, isAdmin, id)
}

func (s *stubService) DeleteAnnouncement(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) error {
	if s.deleteAnnouncementFn == nil {
		panic("unexpected DeleteAnnouncement call")
	}
	return s.deleteAnnouncementFn(ctx, actorID, isAdmin, id)
}

func setupTestRouter(svc ServiceAPI, authUser auth.AuthenticatedUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, authUser)
		c.Next()
	})
	h := NewHandler(svc)
	r.GET("/announcements", h.ListAnnouncements)
	r.GET("/announcements/:id", h.GetAnnouncement)
	r.POST("/announcements", h.CreateAnnouncement)
	r.PATCH("/announcements/:id", h.UpdateAnnouncement)
	r.POST("/announcements/:id/publish", h.PublishAnnouncement)
	r.POST("/announcements/:id/archive", h.ArchiveAnnouncement)
	r.DELETE("/announcements/:id", h.DeleteAnnouncement)
	return r
}

func makeTestUser(roles ...string) auth.AuthenticatedUser {
	uid := uuid.New()
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uid,
			Email:  "test@schoolhub.test",
			Status: "ACTIVE",
		},
		Person: db.Person{
			FirstName: "Test",
			LastName:  "User",
		},
		Roles: roles,
	}
}

func TestListAnnouncements_Feed(t *testing.T) {
	u := makeTestUser("TEACHER")
	annID := uuid.New()
	now := time.Now().UTC()

	svc := &stubService{
		listFeedAnnouncementsFn: func(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]db.ListFeedAnnouncementsRow, int64, error) {
			if userID != u.User.ID {
				t.Fatalf("expected userID %s, got %s", u.User.ID, userID)
			}
			return []db.ListFeedAnnouncementsRow{
				{
					ID:              annID,
					Title:           "Welcome Back",
					Content:         "Welcome to the new school term!",
					AuthorID:        uuid.New(),
					AuthorEmail:     "principal@schoolhub.test",
					AuthorFirstName: "Jane",
					AuthorLastName:  "Doe",
					Status:          "PUBLISHED",
					TargetType:      "ALL",
					Priority:        "NORMAL",
					PublishedAt:     pgtype.Timestamptz{Time: now, Valid: true},
					CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
				},
			}, 1, nil
		},
	}

	router := setupTestRouter(svc, u)
	req := httptest.NewRequest(http.MethodGet, "/announcements?feed=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data []announcementDTO `json:"data"`
		Meta struct {
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
			Total    int64 `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(res.Data) != 1 || res.Data[0].Title != "Welcome Back" {
		t.Fatalf("unexpected data: %+v", res.Data)
	}
	if res.Meta.Total != 1 {
		t.Fatalf("expected total 1, got %d", res.Meta.Total)
	}
}

func TestListAnnouncements_AdminFilter(t *testing.T) {
	u := makeTestUser("ADMIN")
	annID := uuid.New()
	now := time.Now().UTC()

	svc := &stubService{
		listAnnouncementsFn: func(ctx context.Context, f ListFilter) ([]db.ListAnnouncementsRow, int64, error) {
			if f.Status == nil || *f.Status != "DRAFT" {
				t.Fatalf("expected status DRAFT filter, got %+v", f.Status)
			}
			return []db.ListAnnouncementsRow{
				{
					ID:              annID,
					Title:           "Staff Meeting Draft",
					Content:         "Agenda for Friday meeting",
					AuthorID:        u.User.ID,
					AuthorEmail:     u.User.Email,
					AuthorFirstName: "Test",
					AuthorLastName:  "User",
					Status:          "DRAFT",
					TargetType:      "ROLE",
					TargetRole:      pgtype.Text{String: "TEACHER", Valid: true},
					Priority:        "HIGH",
					CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
				},
			}, 1, nil
		},
	}

	router := setupTestRouter(svc, u)
	req := httptest.NewRequest(http.MethodGet, "/announcements?status=DRAFT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAnnouncement_SuccessAndNotFound(t *testing.T) {
	u := makeTestUser("TEACHER")
	annID := uuid.New()
	now := time.Now().UTC()

	svc := &stubService{
		getAnnouncementFn: func(ctx context.Context, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
			if id == annID {
				return db.GetAnnouncementByIDRow{
					ID:              annID,
					Title:           "Assembly",
					Content:         "Morning assembly at 8am",
					AuthorID:        uuid.New(),
					AuthorEmail:     "head@schoolhub.test",
					AuthorFirstName: "Head",
					AuthorLastName:  "Teacher",
					Status:          "PUBLISHED",
					TargetType:      "ALL",
					Priority:        "NORMAL",
					PublishedAt:     pgtype.Timestamptz{Time: now, Valid: true},
					CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
					UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
				}, nil
			}
			return db.GetAnnouncementByIDRow{}, ErrAnnouncementNotFound
		},
	}

	router := setupTestRouter(svc, u)

	// Success case
	req := httptest.NewRequest(http.MethodGet, "/announcements/"+annID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Not found case
	missingID := uuid.New()
	req = httptest.NewRequest(http.MethodGet, "/announcements/"+missingID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateAnnouncement_ValidationAndSuccess(t *testing.T) {
	u := makeTestUser("ADMIN")
	now := time.Now().UTC()

	svc := &stubService{
		createAnnouncementFn: func(ctx context.Context, authorID uuid.UUID, in CreateAnnouncementInput) (db.GetAnnouncementByIDRow, error) {
			if in.Title == "" {
				return db.GetAnnouncementByIDRow{}, ErrTitleRequired
			}
			return db.GetAnnouncementByIDRow{
				ID:              uuid.New(),
				Title:           in.Title,
				Content:         in.Content,
				AuthorID:        authorID,
				AuthorEmail:     u.User.Email,
				AuthorFirstName: u.Person.FirstName,
				AuthorLastName:  u.Person.LastName,
				Status:          "DRAFT",
				TargetType:      "ALL",
				Priority:        "NORMAL",
				CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	router := setupTestRouter(svc, u)

	// Validation failure: missing title
	body, _ := json.Marshal(map[string]any{"content": "No title here"})
	req := httptest.NewRequest(http.MethodPost, "/announcements", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	// Success
	body, _ = json.Marshal(map[string]any{
		"title":       "School Sports Day",
		"content":     "Sports day will take place on next Friday.",
		"target_type": "ALL",
	})
	req = httptest.NewRequest(http.MethodPost, "/announcements", bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAndPublishAnnouncement(t *testing.T) {
	u := makeTestUser("ADMIN")
	annID := uuid.New()
	now := time.Now().UTC()

	svc := &stubService{
		updateAnnouncementFn: func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID, in UpdateAnnouncementInput) (db.GetAnnouncementByIDRow, error) {
			return db.GetAnnouncementByIDRow{
				ID:              annID,
				Title:           *in.Title,
				Content:         "Updated content",
				AuthorID:        u.User.ID,
				AuthorEmail:     u.User.Email,
				AuthorFirstName: u.Person.FirstName,
				AuthorLastName:  u.Person.LastName,
				Status:          "DRAFT",
				TargetType:      "ALL",
				Priority:        "NORMAL",
				CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
		publishAnnouncementFn: func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) (db.GetAnnouncementByIDRow, error) {
			return db.GetAnnouncementByIDRow{
				ID:              annID,
				Title:           "Published Announcement",
				Content:         "Updated content",
				AuthorID:        u.User.ID,
				AuthorEmail:     u.User.Email,
				AuthorFirstName: u.Person.FirstName,
				AuthorLastName:  u.Person.LastName,
				Status:          "PUBLISHED",
				TargetType:      "ALL",
				Priority:        "NORMAL",
				PublishedAt:     pgtype.Timestamptz{Time: now, Valid: true},
				CreatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
			}, nil
		},
	}

	router := setupTestRouter(svc, u)

	// Update test
	newTitle := "Updated Title"
	body, _ := json.Marshal(map[string]any{"title": newTitle})
	req := httptest.NewRequest(http.MethodPatch, "/announcements/"+annID.String(), bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Publish test
	req = httptest.NewRequest(http.MethodPost, "/announcements/"+annID.String()+"/publish", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAnnouncement_ForbiddenAndSuccess(t *testing.T) {
	nonAuthor := makeTestUser("TEACHER")
	annID := uuid.New()

	svc := &stubService{
		deleteAnnouncementFn: func(ctx context.Context, actorID uuid.UUID, isAdmin bool, id uuid.UUID) error {
			if !isAdmin {
				return ErrUnauthorizedAction
			}
			return nil
		},
	}

	// Teacher trying to delete someone else's announcement -> 403 Forbidden
	router := setupTestRouter(svc, nonAuthor)
	req := httptest.NewRequest(http.MethodDelete, "/announcements/"+annID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// Admin deleting -> 204 No Content
	admin := makeTestUser("ADMIN")
	router = setupTestRouter(svc, admin)
	req = httptest.NewRequest(http.MethodDelete, "/announcements/"+annID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}
