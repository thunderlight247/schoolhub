package students

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	listStudentsFn              func(ctx context.Context, f ListFilter) ([]db.ListStudentsRow, int64, error)
	updateStudentFn             func(ctx context.Context, actorID, id uuid.UUID, in UpdateStudentInput) (StudentDetail, error)
	setStatusFn                 func(ctx context.Context, actorID, id uuid.UUID, status string) (StudentDetail, error)
	linkGuardianToStudentFn     func(ctx context.Context, actorID, studentID uuid.UUID, in LinkGuardianInput) ([]db.ListGuardiansForStudentRow, error)
	unlinkGuardianFromStudentFn func(ctx context.Context, actorID, studentID, guardianID uuid.UUID) ([]db.ListGuardiansForStudentRow, error)
	createGuardianFn            func(ctx context.Context, actorID uuid.UUID, in CreateGuardianInput) (GuardianDetail, error)
	getGuardianFn               func(ctx context.Context, id uuid.UUID) (GuardianDetail, error)
	listGuardiansFn             func(ctx context.Context, f GuardianListFilter) ([]db.ListGuardiansRow, int64, error)
	updateGuardianFn            func(ctx context.Context, actorID, id uuid.UUID, in UpdateGuardianInput) (GuardianDetail, error)
	createGuardianAccountFn     func(ctx context.Context, actorID, guardianID uuid.UUID, in GuardianAccountInput) error
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

func (s *stubService) CreateGuardianAccount(ctx context.Context, actorID, guardianID uuid.UUID, in GuardianAccountInput) error {
	if s.createGuardianAccountFn == nil {
		panic("unexpected CreateGuardianAccount call")
	}
	return s.createGuardianAccountFn(ctx, actorID, guardianID, in)
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
	r.POST("/guardians/:id/account", h.CreateGuardianAccount)

	return r
}

func TestCreateGuardianAccountHandler(t *testing.T) {
	actorID := uuid.MustParse(testActorID)
	guardianID := uuid.New()

	var gotActor, gotGuardian uuid.UUID
	var gotInput GuardianAccountInput
	svc := &stubService{
		createGuardianAccountFn: func(ctx context.Context, a, g uuid.UUID, in GuardianAccountInput) error {
			gotActor, gotGuardian, gotInput = a, g, in
			return nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/guardians/"+guardianID.String()+"/account",
		strings.NewReader(`{"email":"parent@example.test","password":"ParentPass2026!"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotActor != actorID || gotGuardian != guardianID {
		t.Fatalf("actor/guardian = %v/%v, want %v/%v", gotActor, gotGuardian, actorID, guardianID)
	}
	// Role is forced to PARENT regardless of what the client sends.
	if gotInput.Email != "parent@example.test" || gotInput.RoleName != "PARENT" {
		t.Fatalf("input = %+v", gotInput)
	}
}

func TestCreateGuardianAccountHandler_RejectsNonParentRole(t *testing.T) {
	svc := &stubService{
		createGuardianAccountFn: func(ctx context.Context, a, g uuid.UUID, in GuardianAccountInput) error {
			t.Error("service should not be reached")
			return nil
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/guardians/"+uuid.New().String()+"/account",
		strings.NewReader(`{"email":"parent@example.test","password":"ParentPass2026!","role":"ADMIN"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "VALIDATION_ERROR") {
		t.Fatalf("expected VALIDATION_ERROR, got %s", rec.Body.String())
	}
}

func TestCreateGuardianAccountHandler_MapsServiceErrors(t *testing.T) {
	svc := &stubService{
		createGuardianAccountFn: func(ctx context.Context, a, g uuid.UUID, in GuardianAccountInput) error {
			return ErrEmailTaken
		},
	}
	r := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/guardians/"+uuid.New().String()+"/account",
		strings.NewReader(`{"email":"parent@example.test","password":"ParentPass2026!"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "EMAIL_TAKEN") {
		t.Fatalf("expected EMAIL_TAKEN, got %s", rec.Body.String())
	}
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
		Students   []studentListItemDTO `json:"students"`
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
