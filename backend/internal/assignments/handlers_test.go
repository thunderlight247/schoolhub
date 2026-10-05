package assignments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	listFn   func(ctx context.Context, f ListFilter) ([]db.ListTeacherAssignmentsRow, int64, error)
	getFn    func(ctx context.Context, id uuid.UUID) (db.GetTeacherAssignmentByIDRow, error)
	createFn func(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error)
	deleteFn func(ctx context.Context, actorID, id uuid.UUID) error
}

func (s *stubService) ListTeacherAssignments(ctx context.Context, f ListFilter) ([]db.ListTeacherAssignmentsRow, int64, error) {
	if s.listFn == nil {
		panic("unexpected ListTeacherAssignments")
	}
	return s.listFn(ctx, f)
}

func (s *stubService) GetTeacherAssignment(ctx context.Context, id uuid.UUID) (db.GetTeacherAssignmentByIDRow, error) {
	if s.getFn == nil {
		panic("unexpected GetTeacherAssignment")
	}
	return s.getFn(ctx, id)
}

func (s *stubService) CreateTeacherAssignment(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error) {
	if s.createFn == nil {
		panic("unexpected CreateTeacherAssignment")
	}
	return s.createFn(ctx, actorID, in)
}

func (s *stubService) DeleteTeacherAssignment(ctx context.Context, actorID, id uuid.UUID) error {
	if s.deleteFn == nil {
		panic("unexpected DeleteTeacherAssignment")
	}
	return s.deleteFn(ctx, actorID, id)
}

func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{User: db.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}})
		c.Next()
	})

	h := NewHandler(svc)
	r.GET("/teacher-assignments", h.ListTeacherAssignments)
	r.GET("/teacher-assignments/:id", h.GetTeacherAssignment)
	r.GET("/teachers/:id/assignments", h.ListTeacherAssignments)
	r.POST("/teacher-assignments", h.CreateTeacherAssignment)
	r.POST("/teachers/:id/assignments", h.CreateTeacherAssignment)
	r.DELETE("/teacher-assignments/:id", h.DeleteTeacherAssignment)
	return r
}

func detailRow(id, teacherID, classID, subjectID uuid.UUID) db.GetTeacherAssignmentByIDRow {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return db.GetTeacherAssignmentByIDRow{
		ID:               id,
		TeacherID:        teacherID,
		ClassID:          classID,
		SubjectID:        subjectID,
		CreatedAt:        now,
		UpdatedAt:        now,
		StaffNumber:      "STF-001",
		TeacherName:      "John Mwangi",
		AcademicYearID:   uuid.New(),
		AcademicYearName: "2025/2026",
		ClassName:        "Grade 10-A",
		SubjectCode:      "MATH",
		SubjectName:      "Mathematics",
	}
}

func TestListTeacherAssignmentsHandler(t *testing.T) {
	teacherID := uuid.New()
	classID := uuid.New()
	subjectID := uuid.New()

	svc := &stubService{
		listFn: func(ctx context.Context, f ListFilter) ([]db.ListTeacherAssignmentsRow, int64, error) {
			if f.TeacherID == nil || *f.TeacherID != teacherID {
				t.Errorf("expected teacher_id filter %s, got %v", teacherID, f.TeacherID)
			}
			row := db.ListTeacherAssignmentsRow{
				ID:               uuid.New(),
				TeacherID:        teacherID,
				ClassID:          classID,
				SubjectID:        subjectID,
				CreatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
				UpdatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
				StaffNumber:      "STF-001",
				TeacherName:      "John Mwangi",
				AcademicYearID:   uuid.New(),
				AcademicYearName: "2025/2026",
				ClassName:        "Grade 10-A",
				SubjectCode:      "MATH",
				SubjectName:      "Mathematics",
			}
			return []db.ListTeacherAssignmentsRow{row}, 1, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/teacher-assignments?teacher_id="+teacherID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Assignments []teacherAssignmentListItemDTO `json:"assignments"`
		Pagination  struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(resp.Assignments))
	}
	if resp.Assignments[0].TeacherName != "John Mwangi" || resp.Assignments[0].ClassName != "Grade 10-A" {
		t.Fatalf("unexpected item: %+v", resp.Assignments[0])
	}
	if resp.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Pagination.Total)
	}
}

func TestListTeacherAssignmentsNestedRouteUsesPathTeacher(t *testing.T) {
	teacherID := uuid.New()

	svc := &stubService{
		listFn: func(ctx context.Context, f ListFilter) ([]db.ListTeacherAssignmentsRow, int64, error) {
			if f.TeacherID == nil || *f.TeacherID != teacherID {
				t.Errorf("expected path teacher %s, got %v", teacherID, f.TeacherID)
			}
			return []db.ListTeacherAssignmentsRow{}, 0, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/teachers/"+teacherID.String()+"/assignments", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTeacherAssignmentHandler(t *testing.T) {
	teacherID := uuid.New()
	classID := uuid.New()
	subjectID := uuid.New()

	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error) {
			if in.TeacherID != teacherID || in.ClassID != classID || in.SubjectID != subjectID {
				t.Errorf("unexpected input: %+v", in)
			}
			return detailRow(uuid.New(), in.TeacherID, in.ClassID, in.SubjectID), nil
		},
	}

	r := newTestRouter(svc)
	payload := `{"teacher_id":"` + teacherID.String() + `","class_id":"` + classID.String() + `","subject_id":"` + subjectID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Assignment teacherAssignmentDTO `json:"assignment"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Assignment.TeacherID != teacherID.String() || resp.Assignment.SubjectName != "Mathematics" {
		t.Fatalf("unexpected assignment: %+v", resp.Assignment)
	}
}

func TestCreateTeacherAssignmentNestedRouteOverridesBody(t *testing.T) {
	pathTeacher := uuid.New()
	bodyTeacher := uuid.New()

	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error) {
			if in.TeacherID != pathTeacher {
				t.Errorf("expected path teacher %s to win, got %s", pathTeacher, in.TeacherID)
			}
			return detailRow(uuid.New(), in.TeacherID, in.ClassID, in.SubjectID), nil
		},
	}

	r := newTestRouter(svc)
	payload := `{"teacher_id":"` + bodyTeacher.String() + `","class_id":"` + uuid.New().String() + `","subject_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/teachers/"+pathTeacher.String()+"/assignments", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTeacherAssignmentValidation(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error) {
			t.Error("service should not be called for an invalid payload")
			return db.GetTeacherAssignmentByIDRow{}, errors.New("unreachable")
		},
	}

	r := newTestRouter(svc)
	payload := `{"teacher_id":"` + uuid.New().String() + `","class_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetTeacherAssignmentHandler(t *testing.T) {
	id := uuid.New()
	svc := &stubService{
		getFn: func(ctx context.Context, got uuid.UUID) (db.GetTeacherAssignmentByIDRow, error) {
			if got != id {
				t.Errorf("expected id %s, got %s", id, got)
			}
			return detailRow(id, uuid.New(), uuid.New(), uuid.New()), nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/teacher-assignments/"+id.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Assignment teacherAssignmentDTO `json:"assignment"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Assignment.ID != id.String() {
		t.Fatalf("expected id %s, got %s", id, resp.Assignment.ID)
	}
}

func TestGetTeacherAssignmentNotFound(t *testing.T) {
	svc := &stubService{
		getFn: func(ctx context.Context, id uuid.UUID) (db.GetTeacherAssignmentByIDRow, error) {
			return db.GetTeacherAssignmentByIDRow{}, ErrAssignmentNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/teacher-assignments/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTeacherAssignmentConflict(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error) {
			return db.GetTeacherAssignmentByIDRow{}, ErrDuplicateAssignment
		},
	}

	r := newTestRouter(svc)
	payload := `{"teacher_id":"` + uuid.New().String() + `","class_id":"` + uuid.New().String() + `","subject_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/teacher-assignments", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteTeacherAssignmentHandler(t *testing.T) {
	id := uuid.New()
	var gotActor, gotID uuid.UUID

	svc := &stubService{
		deleteFn: func(ctx context.Context, actorID, target uuid.UUID) error {
			gotActor, gotID = actorID, target
			return nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/teacher-assignments/"+id.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotID != id {
		t.Errorf("expected delete of %s, got %s", id, gotID)
	}
	if gotActor == uuid.Nil {
		t.Error("expected a non-nil actor id from the authenticated user")
	}
}

func TestDeleteTeacherAssignmentNotFound(t *testing.T) {
	svc := &stubService{
		deleteFn: func(ctx context.Context, actorID, id uuid.UUID) error {
			return ErrAssignmentNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/teacher-assignments/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
