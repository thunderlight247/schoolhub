package academics

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

	createAcademicYearFn    func(ctx context.Context, actorID uuid.UUID, in CreateAcademicYearInput) (db.AcademicYear, error)
	getAcademicYearFn       func(ctx context.Context, id uuid.UUID) (db.AcademicYear, error)
	getCurrentAcademicYearFn func(ctx context.Context) (db.AcademicYear, error)
	listAcademicYearsFn     func(ctx context.Context) ([]db.AcademicYear, error)
	updateAcademicYearFn    func(ctx context.Context, actorID, id uuid.UUID, in UpdateAcademicYearInput) (db.AcademicYear, error)
	setCurrentAcademicYearFn func(ctx context.Context, actorID, id uuid.UUID) (db.AcademicYear, error)

	createTermFn    func(ctx context.Context, actorID uuid.UUID, in CreateTermInput) (db.Term, error)
	getTermFn       func(ctx context.Context, id uuid.UUID) (db.GetTermByIDRow, error)
	getCurrentTermFn func(ctx context.Context) (db.GetCurrentTermRow, error)
	listTermsFn     func(ctx context.Context, academicYearID *uuid.UUID) ([]db.ListAllTermsRow, error)
	updateTermFn    func(ctx context.Context, actorID, id uuid.UUID, in UpdateTermInput) (db.Term, error)
	setCurrentTermFn func(ctx context.Context, actorID, id uuid.UUID) (db.Term, error)

	createProgrammeFn func(ctx context.Context, actorID uuid.UUID, in CreateProgrammeInput) (db.Programme, error)
	getProgrammeFn    func(ctx context.Context, id uuid.UUID) (db.Programme, error)
	listProgrammesFn  func(ctx context.Context) ([]db.Programme, error)
	updateProgrammeFn func(ctx context.Context, actorID, id uuid.UUID, in UpdateProgrammeInput) (db.Programme, error)

	createSubjectFn func(ctx context.Context, actorID uuid.UUID, in CreateSubjectInput) (db.Subject, error)
	getSubjectFn    func(ctx context.Context, id uuid.UUID) (db.GetSubjectByIDRow, error)
	listSubjectsFn  func(ctx context.Context, f SubjectListFilter) ([]db.ListSubjectsRow, error)
	updateSubjectFn func(ctx context.Context, actorID, id uuid.UUID, in UpdateSubjectInput) (db.Subject, error)

	createClassFn           func(ctx context.Context, actorID uuid.UUID, in CreateClassInput) (db.Class, error)
	getClassFn              func(ctx context.Context, id uuid.UUID) (db.GetClassByIDRow, error)
	listClassesFn           func(ctx context.Context, f ClassListFilter) ([]db.ListClassesRow, error)
	updateClassFn           func(ctx context.Context, actorID, id uuid.UUID, in UpdateClassInput) (db.Class, error)
	assignSubjectToClassFn  func(ctx context.Context, actorID, classID, subjectID uuid.UUID) (db.ClassSubject, error)
	removeSubjectFromClassFn func(ctx context.Context, actorID, classID, subjectID uuid.UUID) error
	listSubjectsForClassFn  func(ctx context.Context, classID uuid.UUID) ([]db.ListSubjectsForClassRow, error)
}

func (s *stubService) CreateAcademicYear(ctx context.Context, actorID uuid.UUID, in CreateAcademicYearInput) (db.AcademicYear, error) {
	if s.createAcademicYearFn == nil {
		panic("unexpected CreateAcademicYear")
	}
	return s.createAcademicYearFn(ctx, actorID, in)
}

func (s *stubService) GetAcademicYear(ctx context.Context, id uuid.UUID) (db.AcademicYear, error) {
	if s.getAcademicYearFn == nil {
		panic("unexpected GetAcademicYear")
	}
	return s.getAcademicYearFn(ctx, id)
}

func (s *stubService) GetCurrentAcademicYear(ctx context.Context) (db.AcademicYear, error) {
	if s.getCurrentAcademicYearFn == nil {
		panic("unexpected GetCurrentAcademicYear")
	}
	return s.getCurrentAcademicYearFn(ctx)
}

func (s *stubService) ListAcademicYears(ctx context.Context) ([]db.AcademicYear, error) {
	if s.listAcademicYearsFn == nil {
		panic("unexpected ListAcademicYears")
	}
	return s.listAcademicYearsFn(ctx)
}

func (s *stubService) UpdateAcademicYear(ctx context.Context, actorID, id uuid.UUID, in UpdateAcademicYearInput) (db.AcademicYear, error) {
	if s.updateAcademicYearFn == nil {
		panic("unexpected UpdateAcademicYear")
	}
	return s.updateAcademicYearFn(ctx, actorID, id, in)
}

func (s *stubService) SetCurrentAcademicYear(ctx context.Context, actorID, id uuid.UUID) (db.AcademicYear, error) {
	if s.setCurrentAcademicYearFn == nil {
		panic("unexpected SetCurrentAcademicYear")
	}
	return s.setCurrentAcademicYearFn(ctx, actorID, id)
}

func (s *stubService) CreateTerm(ctx context.Context, actorID uuid.UUID, in CreateTermInput) (db.Term, error) {
	if s.createTermFn == nil {
		panic("unexpected CreateTerm")
	}
	return s.createTermFn(ctx, actorID, in)
}

func (s *stubService) GetTerm(ctx context.Context, id uuid.UUID) (db.GetTermByIDRow, error) {
	if s.getTermFn == nil {
		panic("unexpected GetTerm")
	}
	return s.getTermFn(ctx, id)
}

func (s *stubService) GetCurrentTerm(ctx context.Context) (db.GetCurrentTermRow, error) {
	if s.getCurrentTermFn == nil {
		panic("unexpected GetCurrentTerm")
	}
	return s.getCurrentTermFn(ctx)
}

func (s *stubService) ListTerms(ctx context.Context, academicYearID *uuid.UUID) ([]db.ListAllTermsRow, error) {
	if s.listTermsFn == nil {
		panic("unexpected ListTerms")
	}
	return s.listTermsFn(ctx, academicYearID)
}

func (s *stubService) UpdateTerm(ctx context.Context, actorID, id uuid.UUID, in UpdateTermInput) (db.Term, error) {
	if s.updateTermFn == nil {
		panic("unexpected UpdateTerm")
	}
	return s.updateTermFn(ctx, actorID, id, in)
}

func (s *stubService) SetCurrentTerm(ctx context.Context, actorID, id uuid.UUID) (db.Term, error) {
	if s.setCurrentTermFn == nil {
		panic("unexpected SetCurrentTerm")
	}
	return s.setCurrentTermFn(ctx, actorID, id)
}

func (s *stubService) CreateProgramme(ctx context.Context, actorID uuid.UUID, in CreateProgrammeInput) (db.Programme, error) {
	if s.createProgrammeFn == nil {
		panic("unexpected CreateProgramme")
	}
	return s.createProgrammeFn(ctx, actorID, in)
}

func (s *stubService) GetProgramme(ctx context.Context, id uuid.UUID) (db.Programme, error) {
	if s.getProgrammeFn == nil {
		panic("unexpected GetProgramme")
	}
	return s.getProgrammeFn(ctx, id)
}

func (s *stubService) ListProgrammes(ctx context.Context) ([]db.Programme, error) {
	if s.listProgrammesFn == nil {
		panic("unexpected ListProgrammes")
	}
	return s.listProgrammesFn(ctx)
}

func (s *stubService) UpdateProgramme(ctx context.Context, actorID, id uuid.UUID, in UpdateProgrammeInput) (db.Programme, error) {
	if s.updateProgrammeFn == nil {
		panic("unexpected UpdateProgramme")
	}
	return s.updateProgrammeFn(ctx, actorID, id, in)
}

func (s *stubService) CreateSubject(ctx context.Context, actorID uuid.UUID, in CreateSubjectInput) (db.Subject, error) {
	if s.createSubjectFn == nil {
		panic("unexpected CreateSubject")
	}
	return s.createSubjectFn(ctx, actorID, in)
}

func (s *stubService) GetSubject(ctx context.Context, id uuid.UUID) (db.GetSubjectByIDRow, error) {
	if s.getSubjectFn == nil {
		panic("unexpected GetSubject")
	}
	return s.getSubjectFn(ctx, id)
}

func (s *stubService) ListSubjects(ctx context.Context, f SubjectListFilter) ([]db.ListSubjectsRow, error) {
	if s.listSubjectsFn == nil {
		panic("unexpected ListSubjects")
	}
	return s.listSubjectsFn(ctx, f)
}

func (s *stubService) UpdateSubject(ctx context.Context, actorID, id uuid.UUID, in UpdateSubjectInput) (db.Subject, error) {
	if s.updateSubjectFn == nil {
		panic("unexpected UpdateSubject")
	}
	return s.updateSubjectFn(ctx, actorID, id, in)
}

func (s *stubService) CreateClass(ctx context.Context, actorID uuid.UUID, in CreateClassInput) (db.Class, error) {
	if s.createClassFn == nil {
		panic("unexpected CreateClass")
	}
	return s.createClassFn(ctx, actorID, in)
}

func (s *stubService) GetClass(ctx context.Context, id uuid.UUID) (db.GetClassByIDRow, error) {
	if s.getClassFn == nil {
		panic("unexpected GetClass")
	}
	return s.getClassFn(ctx, id)
}

func (s *stubService) ListClasses(ctx context.Context, f ClassListFilter) ([]db.ListClassesRow, error) {
	if s.listClassesFn == nil {
		panic("unexpected ListClasses")
	}
	return s.listClassesFn(ctx, f)
}

func (s *stubService) UpdateClass(ctx context.Context, actorID, id uuid.UUID, in UpdateClassInput) (db.Class, error) {
	if s.updateClassFn == nil {
		panic("unexpected UpdateClass")
	}
	return s.updateClassFn(ctx, actorID, id, in)
}

func (s *stubService) AssignSubjectToClass(ctx context.Context, actorID, classID, subjectID uuid.UUID) (db.ClassSubject, error) {
	if s.assignSubjectToClassFn == nil {
		panic("unexpected AssignSubjectToClass")
	}
	return s.assignSubjectToClassFn(ctx, actorID, classID, subjectID)
}

func (s *stubService) RemoveSubjectFromClass(ctx context.Context, actorID, classID, subjectID uuid.UUID) error {
	if s.removeSubjectFromClassFn == nil {
		panic("unexpected RemoveSubjectFromClass")
	}
	return s.removeSubjectFromClassFn(ctx, actorID, classID, subjectID)
}

func (s *stubService) ListSubjectsForClass(ctx context.Context, classID uuid.UUID) ([]db.ListSubjectsForClassRow, error) {
	if s.listSubjectsForClassFn == nil {
		panic("unexpected ListSubjectsForClass")
	}
	return s.listSubjectsForClassFn(ctx, classID)
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

	r.GET("/academic-years", h.ListAcademicYears)
	r.GET("/academic-years/current", h.GetCurrentAcademicYear)
	r.GET("/academic-years/:id", h.GetAcademicYear)
	r.POST("/academic-years", h.CreateAcademicYear)
	r.PATCH("/academic-years/:id", h.UpdateAcademicYear)
	r.PUT("/academic-years/:id/current", h.SetCurrentAcademicYear)

	r.GET("/terms", h.ListTerms)
	r.GET("/terms/current", h.GetCurrentTerm)
	r.GET("/terms/:id", h.GetTerm)
	r.POST("/terms", h.CreateTerm)
	r.PATCH("/terms/:id", h.UpdateTerm)
	r.PUT("/terms/:id/current", h.SetCurrentTerm)

	r.GET("/programmes", h.ListProgrammes)
	r.GET("/programmes/:id", h.GetProgramme)
	r.POST("/programmes", h.CreateProgramme)
	r.PATCH("/programmes/:id", h.UpdateProgramme)

	r.GET("/subjects", h.ListSubjects)
	r.GET("/subjects/:id", h.GetSubject)
	r.POST("/subjects", h.CreateSubject)
	r.PATCH("/subjects/:id", h.UpdateSubject)

	r.GET("/classes", h.ListClasses)
	r.GET("/classes/:id", h.GetClass)
	r.POST("/classes", h.CreateClass)
	r.PATCH("/classes/:id", h.UpdateClass)
	r.GET("/classes/:id/subjects", h.ListSubjectsForClass)
	r.POST("/classes/:id/subjects", h.AssignSubjectToClass)
	r.DELETE("/classes/:id/subjects/:subject_id", h.RemoveSubjectFromClass)

	return r
}

func TestAcademicYearsHandlers(t *testing.T) {
	yearID := uuid.New()
	now := time.Now()

	svc := &stubService{
		listAcademicYearsFn: func(ctx context.Context) ([]db.AcademicYear, error) {
			return []db.AcademicYear{
				{
					ID:        yearID,
					Name:      "2025/2026",
					StartDate: pgtype.Date{Time: now, Valid: true},
					EndDate:   pgtype.Date{Time: now.AddDate(1, 0, 0), Valid: true},
					IsCurrent: true,
				},
			}, nil
		},
		createAcademicYearFn: func(ctx context.Context, actorID uuid.UUID, in CreateAcademicYearInput) (db.AcademicYear, error) {
			return db.AcademicYear{
				ID:        yearID,
				Name:      in.Name,
				StartDate: pgtype.Date{Time: in.StartDate, Valid: true},
				EndDate:   pgtype.Date{Time: in.EndDate, Valid: true},
				IsCurrent: in.IsCurrent,
			}, nil
		},
		getAcademicYearFn: func(ctx context.Context, id uuid.UUID) (db.AcademicYear, error) {
			if id == yearID {
				return db.AcademicYear{
					ID:        yearID,
					Name:      "2025/2026",
					StartDate: pgtype.Date{Time: now, Valid: true},
					EndDate:   pgtype.Date{Time: now.AddDate(1, 0, 0), Valid: true},
					IsCurrent: true,
				}, nil
			}
			return db.AcademicYear{}, ErrAcademicYearNotFound
		},
	}

	r := newTestRouter(svc)

	// List
	req := httptest.NewRequest(http.MethodGet, "/academic-years", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create
	body := `{"name":"2025/2026","start_date":"2025-09-01","end_date":"2026-06-30","is_current":true}`
	req = httptest.NewRequest(http.MethodPost, "/academic-years", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get 404
	req = httptest.NewRequest(http.MethodGet, "/academic-years/"+uuid.New().String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestTermsHandlers(t *testing.T) {
	termID := uuid.New()
	yearID := uuid.New()
	now := time.Now()

	svc := &stubService{
		createTermFn: func(ctx context.Context, actorID uuid.UUID, in CreateTermInput) (db.Term, error) {
			return db.Term{
				ID:             termID,
				AcademicYearID: in.AcademicYearID,
				Name:           in.Name,
				StartDate:      pgtype.Date{Time: in.StartDate, Valid: true},
				EndDate:        pgtype.Date{Time: in.EndDate, Valid: true},
				IsCurrent:      in.IsCurrent,
			}, nil
		},
		getCurrentTermFn: func(ctx context.Context) (db.GetCurrentTermRow, error) {
			return db.GetCurrentTermRow{
				ID:               termID,
				AcademicYearID:   yearID,
				AcademicYearName: "2025/2026",
				Name:             "Term 1",
				StartDate:        pgtype.Date{Time: now, Valid: true},
				EndDate:          pgtype.Date{Time: now.AddDate(0, 3, 0), Valid: true},
				IsCurrent:        true,
			}, nil
		},
	}

	r := newTestRouter(svc)

	// Create
	body := `{"academic_year_id":"` + yearID.String() + `","name":"Term 1","start_date":"2025-09-01","end_date":"2025-12-15","is_current":true}`
	req := httptest.NewRequest(http.MethodPost, "/terms", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Current
	req = httptest.NewRequest(http.MethodGet, "/terms/current", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProgrammesAndSubjectsHandlers(t *testing.T) {
	progID := uuid.New()
	subID := uuid.New()

	svc := &stubService{
		createProgrammeFn: func(ctx context.Context, actorID uuid.UUID, in CreateProgrammeInput) (db.Programme, error) {
			return db.Programme{
				ID:   progID,
				Code: in.Code,
				Name: in.Name,
			}, nil
		},
		createSubjectFn: func(ctx context.Context, actorID uuid.UUID, in CreateSubjectInput) (db.Subject, error) {
			return db.Subject{
				ID:   subID,
				Code: in.Code,
				Name: in.Name,
				Type: in.Type,
			}, nil
		},
	}

	r := newTestRouter(svc)

	// Programme Create
	body := `{"code":"SCI","name":"General Science"}`
	req := httptest.NewRequest(http.MethodPost, "/programmes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Subject Create
	body = `{"code":"MATH101","name":"Core Mathematics","type":"CORE"}`
	req = httptest.NewRequest(http.MethodPost, "/subjects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestClassesAndClassSubjectsHandlers(t *testing.T) {
	classID := uuid.New()
	subID := uuid.New()
	yearID := uuid.New()

	svc := &stubService{
		createClassFn: func(ctx context.Context, actorID uuid.UUID, in CreateClassInput) (db.Class, error) {
			return db.Class{
				ID:             classID,
				AcademicYearID: in.AcademicYearID,
				Name:           in.Name,
			}, nil
		},
		assignSubjectToClassFn: func(ctx context.Context, actorID, cID, sID uuid.UUID) (db.ClassSubject, error) {
			return db.ClassSubject{
				ID:        uuid.New(),
				ClassID:   cID,
				SubjectID: sID,
			}, nil
		},
		removeSubjectFromClassFn: func(ctx context.Context, actorID, cID, sID uuid.UUID) error {
			return nil
		},
		listSubjectsForClassFn: func(ctx context.Context, cID uuid.UUID) ([]db.ListSubjectsForClassRow, error) {
			return []db.ListSubjectsForClassRow{
				{
					ID:   subID,
					Code: "ENG",
					Name: "English Language",
					Type: "CORE",
				},
			}, nil
		},
	}

	r := newTestRouter(svc)

	// Create class
	body := `{"academic_year_id":"` + yearID.String() + `","name":"Form 1A"}`
	req := httptest.NewRequest(http.MethodPost, "/classes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Assign subject
	body = `{"subject_id":"` + subID.String() + `"}`
	req = httptest.NewRequest(http.MethodPost, "/classes/"+classID.String()+"/subjects", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// List subjects for class
	req = httptest.NewRequest(http.MethodGet, "/classes/"+classID.String()+"/subjects", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Subjects []classSubjectDTO `json:"subjects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Subjects) != 1 || resp.Subjects[0].Code != "ENG" {
		t.Fatalf("unexpected subjects: %+v", resp.Subjects)
	}

	// Remove subject
	req = httptest.NewRequest(http.MethodDelete, "/classes/"+classID.String()+"/subjects/"+subID.String(), nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
