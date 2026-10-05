package students

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

	createStudentFn             func(ctx context.Context, actorID uuid.UUID, in CreateStudentInput) (StudentDetail, error)
	getStudentFn                func(ctx context.Context, id uuid.UUID) (StudentDetail, error)
	listStudentsFn               func(ctx context.Context, f ListFilter) ([]db.ListStudentsRow, int64, error)
	updateStudentFn             func(ctx context.Context, actorID, id uuid.UUID, in UpdateStudentInput) (StudentDetail, error)
	setStatusFn                 func(ctx context.Context, actorID, id uuid.UUID, status string) (StudentDetail, error)
	linkGuardianToStudentFn     func(ctx context.Context, actorID, studentID uuid.UUID, in LinkGuardianInput) ([]db.ListGuardiansForStudentRow, error)
	unlinkGuardianFromStudentFn func(ctx context.Context, actorID, studentID, guardianID uuid.UUID) ([]db.ListGuardiansForStudentRow, error)
	createGuardianFn            func(ctx context.Context, actorID uuid.UUID, in CreateGuardianInput) (GuardianDetail, error)
	getGuardianFn               func(ctx context.Context, id uuid.UUID) (GuardianDetail, error)
	listGuardiansFn             func(ctx context.Context, f GuardianListFilter) ([]db.ListGuardiansRow, int64, error)
	updateGuardianFn            func(ctx context.Context, actorID, id uuid.UUID, in UpdateGuardianInput) (GuardianDetail, error)
}

func (s *stubService) CreateStudent(ctx context.Context, actorID uuid.UUID, in CreateStudentInput) (StudentDetail, error) {
	if s.createStudentFn == nil {
		panic("unexpected CreateStudent call")
	}
	return s.createStudentFn(ctx, actorID, in)
}

func (s *stubService) GetStudent(ctx context.Context, id uuid.UUID) (StudentDetail, error) {
	if s.getStudentFn == nil {
		panic("unexpected GetStudent call")
	}
	return s.getStudentFn(ctx, id)
}

func (s *stubService) ListStudents(ctx context.Context, f ListFilter) ([]db.ListStudentsRow, int64, error) {
	if s.listStudentsFn == nil {
		panic("unexpected ListStudents call")
	}
	return s.listStudentsFn(ctx, f)
}

func (s *stubService) UpdateStudent(ctx context.Context, actorID, id uuid.UUID, in UpdateStudentInput) (StudentDetail, error) {
	if s.updateStudentFn == nil {
		panic("unexpected UpdateStudent call")
	}
	return s.updateStudentFn(ctx, actorID, id, in)
}

func (s *stubService) SetStudentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (StudentDetail, error) {
	if s.setStatusFn == nil {
		panic("unexpected SetStudentStatus call")
	}
	return s.setStatusFn(ctx, actorID, id, status)
}

func (s *stubService) LinkGuardianToStudent(ctx context.Context, actorID, studentID uuid.UUID, in LinkGuardianInput) ([]db.ListGuardiansForStudentRow, error) {
	if s.linkGuardianToStudentFn == nil {
		panic("unexpected LinkGuardianToStudent call")
	}
	return s.linkGuardianToStudentFn(ctx, actorID, studentID, in)
}

func (s *stubService) UnlinkGuardianFromStudent(ctx context.Context, actorID, studentID, guardianID uuid.UUID) ([]db.ListGuardiansForStudentRow, error) {
	if s.unlinkGuardianFromStudentFn == nil {
		panic("unexpected UnlinkGuardianFromStudent call")
	}
	return s.unlinkGuardianFromStudentFn(ctx, actorID, studentID, guardianID)
}

func (s *stubService) CreateGuardian(ctx context.Context, actorID uuid.UUID, in CreateGuardianInput) (GuardianDetail, error) {
	if s.createGuardianFn == nil {
		panic("unexpected CreateGuardian call")
	}
	return s.createGuardianFn(ctx, actorID, in)
}

func (s *stubService) GetGuardian(ctx context.Context, id uuid.UUID) (GuardianDetail, error) {
	if s.getGuardianFn == nil {
		panic("unexpected GetGuardian call")
	}
	return s.getGuardianFn(ctx, id)
}

func (s *stubService) ListGuardians(ctx context.Context, f GuardianListFilter) ([]db.ListGuardiansRow, int64, error) {
	if s.listGuardiansFn == nil {
		panic("unexpected ListGuardians call")
	}
	return s.listGuardiansFn(ctx, f)
}

func (s *stubService) UpdateGuardian(ctx context.Context, actorID, id uuid.UUID, in UpdateGuardianInput) (GuardianDetail, error) {
	if s.updateGuardianFn == nil {
		panic("unexpected UpdateGuardian call")
	}
	return s.updateGuardianFn(ctx, actorID, id, in)
}

const testActorID = "11111111-1111-1111-1111-111111111111"

func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{
			User: db.User{ID: uuid.MustParse(testActorID)},
		})
		c.Next()
	})

	h := NewHandler(svc)
	r.GET("/students", h.ListStudents)
	r.GET("/students/:id", h.GetStudent)
	r.POST("/students", h.CreateStudent)
	r.PATCH("/students/:id", h.UpdateStudent)
	r.PUT("/students/:id/status", h.SetStudentStatus)
	r.POST("/students/:id/guardians", h.LinkGuardian)
	r.DELETE("/students/:id/guardians/:guardian_id", h.UnlinkGuardian)

	r.GET("/guardians", h.ListGuardians)
	r.GET("/guardians/:id", h.GetGuardian)
	r.POST("/guardians", h.CreateGuardian)
	r.PATCH("/guardians/:id", h.UpdateGuardian)

	return r
}

func TestListStudentsHandler(t *testing.T) {
	studentID := uuid.New()
	svc := &stubService{
		listStudentsFn: func(ctx context.Context, f ListFilter) ([]db.ListStudentsRow, int64, error) {
			return []db.ListStudentsRow{
				{
					ID:            studentID,
					StudentNumber: "STU-001",
					FirstName:     "Alice",
					LastName:      "Smith",
					Status:        "ACTIVE",
					CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				},
			}, 1, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/students?page=1&page_size=20", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Students []studentListItemDTO `json:"students"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Students) != 1 || resp.Students[0].StudentNumber != "STU-001" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Pagination.Total)
	}
}

func TestCreateStudentHandler(t *testing.T) {
	studentID := uuid.New()
	svc := &stubService{
		createStudentFn: func(ctx context.Context, actorID uuid.UUID, in CreateStudentInput) (StudentDetail, error) {
			return StudentDetail{
				Row: db.GetStudentByIDRow{
					ID:            studentID,
					StudentNumber: in.StudentNumber,
					FirstName:     in.FirstName,
					LastName:      in.LastName,
					Status:        "ACTIVE",
					CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
					UpdatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
				},
			}, nil
		},
	}

	r := newTestRouter(svc)
	payload := `{"student_number":"STU-100","first_name":"Bob","last_name":"Brown"}`
	req := httptest.NewRequest(http.MethodPost, "/students", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Student studentDTO `json:"student"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Student.StudentNumber != "STU-100" {
		t.Fatalf("expected STU-100, got %s", resp.Student.StudentNumber)
	}
}

func TestGetStudentNotFound(t *testing.T) {
	svc := &stubService{
		getStudentFn: func(ctx context.Context, id uuid.UUID) (StudentDetail, error) {
			return StudentDetail{}, ErrStudentNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/students/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
