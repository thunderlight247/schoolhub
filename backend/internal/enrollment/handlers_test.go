package enrollment

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
	listEnrollmentsFn        func(ctx context.Context, f ListFilter) ([]db.ListEnrollmentsRow, int64, error)
	createEnrollmentFn       func(ctx context.Context, actorID uuid.UUID, in CreateEnrollmentInput) (db.Enrollment, error)
	updateEnrollmentStatusFn func(ctx context.Context, actorID, id uuid.UUID, status string) (db.GetEnrollmentByIDRow, error)
	getEnrollmentFn          func(ctx context.Context, id uuid.UUID) (db.GetEnrollmentByIDRow, error)
}

func (s *stubService) ListEnrollments(ctx context.Context, f ListFilter) ([]db.ListEnrollmentsRow, int64, error) {
	if s.listEnrollmentsFn == nil {
		panic("unexpected ListEnrollments")
	}
	return s.listEnrollmentsFn(ctx, f)
}

func (s *stubService) CreateEnrollment(ctx context.Context, actorID uuid.UUID, in CreateEnrollmentInput) (db.Enrollment, error) {
	if s.createEnrollmentFn == nil {
		panic("unexpected CreateEnrollment")
	}
	return s.createEnrollmentFn(ctx, actorID, in)
}

func (s *stubService) UpdateEnrollmentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (db.GetEnrollmentByIDRow, error) {
	if s.updateEnrollmentStatusFn == nil {
		panic("unexpected UpdateEnrollmentStatus")
	}
	return s.updateEnrollmentStatusFn(ctx, actorID, id, status)
}

func (s *stubService) GetEnrollment(ctx context.Context, id uuid.UUID) (db.GetEnrollmentByIDRow, error) {
	if s.getEnrollmentFn == nil {
		panic("unexpected GetEnrollment")
	}
	return s.getEnrollmentFn(ctx, id)
}

func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{User: db.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}})
		c.Next()
	})

	h := NewHandler(svc)
	r.GET("/enrollments", h.ListEnrollments)
	r.GET("/enrollments/:id", h.GetEnrollment)
	r.POST("/enrollments", h.CreateEnrollment)
	r.PATCH("/enrollments/:id/status", h.UpdateEnrollmentStatus)
	return r
}

func TestListEnrollmentsHandler(t *testing.T) {
	classID := uuid.New()
	studentID := uuid.New()
	academicYearID := uuid.New()

	svc := &stubService{
		listEnrollmentsFn: func(ctx context.Context, f ListFilter) ([]db.ListEnrollmentsRow, int64, error) {
			return []db.ListEnrollmentsRow{{
				ID:               uuid.New(),
				StudentID:        studentID,
				AcademicYearID:   academicYearID,
				ClassID:          classID,
				Status:           StatusEnrolled,
				EnrollmentDate:   pgtype.Date{Time: time.Now(), Valid: true},
				StudentName:      "Alice Smith",
				ClassName:        "Grade 10-A",
				AcademicYearName: "2025/2026",
				CreatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}}, 1, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/enrollments?student_id="+studentID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Enrollments []enrollmentListItemDTO `json:"enrollments"`
		Pagination  struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Enrollments) != 1 || resp.Enrollments[0].Status != StatusEnrolled {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Pagination.Total)
	}
}

func TestCreateEnrollmentHandler(t *testing.T) {
	id := uuid.New()
	svc := &stubService{
		createEnrollmentFn: func(ctx context.Context, actorID uuid.UUID, in CreateEnrollmentInput) (db.Enrollment, error) {
			var notes pgtype.Text
			if in.Notes != nil {
				notes = pgtype.Text{String: *in.Notes, Valid: true}
			}
			return db.Enrollment{
				ID:             id,
				StudentID:      in.StudentID,
				AcademicYearID: in.AcademicYearID,
				ClassID:        in.ClassID,
				Status:         in.Status,
				EnrollmentDate: pgtype.Date{Time: in.EnrollmentDate, Valid: true},
				Notes:          notes,
				CreatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
				UpdatedAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil
		},
	}

	r := newTestRouter(svc)
	payload := `{"student_id":"` + uuid.New().String() + `","academic_year_id":"` + uuid.New().String() + `","class_id":"` + uuid.New().String() + `","status":"ENROLLED","enrollment_date":"2025-01-15"}`
	req := httptest.NewRequest(http.MethodPost, "/enrollments", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Enrollment enrollmentDTO `json:"enrollment"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Enrollment.Status != StatusEnrolled {
		t.Fatalf("expected status %q, got %q", StatusEnrolled, resp.Enrollment.Status)
	}
}
