package assessments

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
	createAssessmentFn      func(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error)
	getAssessmentFn         func(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error)
	listAssessmentsFn       func(ctx context.Context, f ListFilter) ([]db.ListAssessmentsRow, int64, error)
	updateAssessmentFn      func(ctx context.Context, actorID, id uuid.UUID, in UpdateAssessmentInput) (db.GetAssessmentByIDRow, error)
	deleteAssessmentFn      func(ctx context.Context, actorID, id uuid.UUID) error
	recordResultsFn         func(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error)
	listAssessmentResultsFn func(ctx context.Context, assessmentID uuid.UUID) ([]db.ListAssessmentResultsRow, error)
	getResultFn             func(ctx context.Context, id uuid.UUID) (db.GetResultByIDRow, error)
	updateResultFn          func(ctx context.Context, actorID, id uuid.UUID, in UpdateResultInput) (db.GetResultByIDRow, error)
	listStudentResultsFn    func(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) ([]db.ListStudentResultsRow, error)
}

func (s *stubService) CreateAssessment(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error) {
	if s.createAssessmentFn == nil {
		panic("unexpected CreateAssessment")
	}
	return s.createAssessmentFn(ctx, actorID, in)
}

func (s *stubService) GetAssessment(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error) {
	if s.getAssessmentFn == nil {
		panic("unexpected GetAssessment")
	}
	return s.getAssessmentFn(ctx, id)
}

func (s *stubService) ListAssessments(ctx context.Context, f ListFilter) ([]db.ListAssessmentsRow, int64, error) {
	if s.listAssessmentsFn == nil {
		panic("unexpected ListAssessments")
	}
	return s.listAssessmentsFn(ctx, f)
}

func (s *stubService) UpdateAssessment(ctx context.Context, actorID, id uuid.UUID, in UpdateAssessmentInput) (db.GetAssessmentByIDRow, error) {
	if s.updateAssessmentFn == nil {
		panic("unexpected UpdateAssessment")
	}
	return s.updateAssessmentFn(ctx, actorID, id, in)
}

func (s *stubService) DeleteAssessment(ctx context.Context, actorID, id uuid.UUID) error {
	if s.deleteAssessmentFn == nil {
		panic("unexpected DeleteAssessment")
	}
	return s.deleteAssessmentFn(ctx, actorID, id)
}

func (s *stubService) RecordResults(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error) {
	if s.recordResultsFn == nil {
		panic("unexpected RecordResults")
	}
	return s.recordResultsFn(ctx, actorID, assessmentID, in)
}

func (s *stubService) ListAssessmentResults(ctx context.Context, assessmentID uuid.UUID) ([]db.ListAssessmentResultsRow, error) {
	if s.listAssessmentResultsFn == nil {
		panic("unexpected ListAssessmentResults")
	}
	return s.listAssessmentResultsFn(ctx, assessmentID)
}

func (s *stubService) GetResult(ctx context.Context, id uuid.UUID) (db.GetResultByIDRow, error) {
	if s.getResultFn == nil {
		panic("unexpected GetResult")
	}
	return s.getResultFn(ctx, id)
}

func (s *stubService) UpdateResult(ctx context.Context, actorID, id uuid.UUID, in UpdateResultInput) (db.GetResultByIDRow, error) {
	if s.updateResultFn == nil {
		panic("unexpected UpdateResult")
	}
	return s.updateResultFn(ctx, actorID, id, in)
}

func (s *stubService) ListStudentResults(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) ([]db.ListStudentResultsRow, error) {
	if s.listStudentResultsFn == nil {
		panic("unexpected ListStudentResults")
	}
	return s.listStudentResultsFn(ctx, studentID, termID, academicYearID)
}

func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{User: db.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}})
		c.Next()
	})

	h := NewHandler(svc)
	r.GET("/assessments", h.ListAssessments)
	r.GET("/assessments/:id", h.GetAssessment)
	r.POST("/assessments", h.CreateAssessment)
	r.PATCH("/assessments/:id", h.UpdateAssessment)
	r.DELETE("/assessments/:id", h.DeleteAssessment)

	r.GET("/assessments/:id/results", h.ListAssessmentResults)
	r.POST("/assessments/:id/results", h.RecordResults)
	r.GET("/results/:id", h.GetResult)
	r.PATCH("/results/:id", h.UpdateResult)
	r.GET("/students/:id/results", h.ListStudentResults)

	return r
}

func assessmentDetailRow(id, classID, subjectID, termID uuid.UUID, title string) db.GetAssessmentByIDRow {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return db.GetAssessmentByIDRow{
		ID:               id,
		ClassID:          classID,
		SubjectID:        subjectID,
		TermID:           termID,
		Title:            title,
		Type:             TypeTest,
		MaxScore:         floatToNumeric(100),
		Weight:           floatPtrToNumeric(ptr(20.0)),
		AssessmentDate:   pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
		CreatedAt:        now,
		UpdatedAt:        now,
		ClassName:        "Grade 10-A",
		AcademicYearID:   uuid.New(),
		AcademicYearName: "2025/2026",
		SubjectCode:      "MATH",
		SubjectName:      "Mathematics",
		TermName:         "Term 1",
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestCreateAssessmentHandler(t *testing.T) {
	classID := uuid.New()
	subjectID := uuid.New()
	termID := uuid.New()

	tests := []struct {
		name       string
		body       any
		serviceFn  func(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error)
		wantStatus int
		wantCode   string
	}{
		{
			name: "success 201",
			body: map[string]any{
				"class_id":        classID.String(),
				"subject_id":      subjectID.String(),
				"term_id":         termID.String(),
				"title":           "Midterm Exam",
				"type":            "MIDTERM",
				"max_score":       100.0,
				"weight":          30.0,
				"assessment_date": "2026-10-05",
			},
			serviceFn: func(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error) {
				return assessmentDetailRow(uuid.New(), in.ClassID, in.SubjectID, in.TermID, in.Title), nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing class_id",
			body:       map[string]any{"subject_id": subjectID.String(), "term_id": termID.String(), "title": "Test"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "duplicate title conflict 409",
			body: map[string]any{
				"class_id":        classID.String(),
				"subject_id":      subjectID.String(),
				"term_id":         termID.String(),
				"title":           "Existing Test",
				"max_score":       50.0,
				"assessment_date": "2026-10-05",
			},
			serviceFn: func(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error) {
				return db.GetAssessmentByIDRow{}, ErrDuplicateAssessment
			},
			wantStatus: http.StatusConflict,
			wantCode:   "ALREADY_EXISTS",
		},
		{
			name: "class not found 404",
			body: map[string]any{
				"class_id":        classID.String(),
				"subject_id":      subjectID.String(),
				"term_id":         termID.String(),
				"title":           "Test",
				"max_score":       50.0,
				"assessment_date": "2026-10-05",
			},
			serviceFn: func(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error) {
				return db.GetAssessmentByIDRow{}, ErrClassNotFound
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name: "date outside term 400",
			body: map[string]any{
				"class_id":        classID.String(),
				"subject_id":      subjectID.String(),
				"term_id":         termID.String(),
				"title":           "Test",
				"max_score":       50.0,
				"assessment_date": "2026-10-05",
			},
			serviceFn: func(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error) {
				return db.GetAssessmentByIDRow{}, ErrDateOutsideTerm
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{createAssessmentFn: tc.serviceFn}
			r := newTestRouter(svc)

			payload, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/assessments", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantCode != "" {
				var errResp struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
				if errResp.Error.Code != tc.wantCode {
					t.Errorf("error code %q, want %q", errResp.Error.Code, tc.wantCode)
				}
			}
		})
	}
}

func TestGetAssessmentHandler(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		idParam    string
		serviceFn  func(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error)
		wantStatus int
	}{
		{
			name:    "success 200",
			idParam: id.String(),
			serviceFn: func(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error) {
				return assessmentDetailRow(id, uuid.New(), uuid.New(), uuid.New(), "Quiz 1"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid uuid 400",
			idParam:    "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "not found 404",
			idParam: id.String(),
			serviceFn: func(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error) {
				return db.GetAssessmentByIDRow{}, ErrAssessmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{getAssessmentFn: tc.serviceFn}
			r := newTestRouter(svc)

			req := httptest.NewRequest(http.MethodGet, "/assessments/"+tc.idParam, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestListAssessmentsHandler(t *testing.T) {
	svc := &stubService{
		listAssessmentsFn: func(ctx context.Context, f ListFilter) ([]db.ListAssessmentsRow, int64, error) {
			now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
			row := db.ListAssessmentsRow{
				ID:               uuid.New(),
				ClassID:          uuid.New(),
				SubjectID:        uuid.New(),
				TermID:           uuid.New(),
				Title:            "Midterm Exam",
				Type:             TypeMidterm,
				MaxScore:         floatToNumeric(100),
				Weight:           floatPtrToNumeric(ptr(30.0)),
				AssessmentDate:   pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
				CreatedAt:        now,
				UpdatedAt:        now,
				ClassName:        "Grade 10-A",
				AcademicYearID:   uuid.New(),
				AcademicYearName: "2025/2026",
				SubjectCode:      "MATH",
				SubjectName:      "Mathematics",
				TermName:         "Term 1",
			}
			return []db.ListAssessmentsRow{row}, 1, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/assessments?page=1&page_size=10&type=MIDTERM", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Assessments []assessmentDTO `json:"assessments"`
		Pagination  struct {
			Page  int   `json:"page"`
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Assessments) != 1 || resp.Pagination.Total != 1 {
		t.Fatalf("expected 1 assessment, got %+v", resp)
	}
}

func TestUpdateAssessmentHandler(t *testing.T) {
	id := uuid.New()
	svc := &stubService{
		updateAssessmentFn: func(ctx context.Context, actorID, id uuid.UUID, in UpdateAssessmentInput) (db.GetAssessmentByIDRow, error) {
			return assessmentDetailRow(id, uuid.New(), uuid.New(), uuid.New(), *in.Title), nil
		},
	}

	r := newTestRouter(svc)
	payload, _ := json.Marshal(map[string]any{"title": "Updated Title"})
	req := httptest.NewRequest(http.MethodPatch, "/assessments/"+id.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAssessmentHandler(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		serviceFn  func(ctx context.Context, actorID, id uuid.UUID) error
		wantStatus int
	}{
		{
			name: "success 200",
			serviceFn: func(ctx context.Context, actorID, id uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not found 404",
			serviceFn: func(ctx context.Context, actorID, id uuid.UUID) error {
				return ErrAssessmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{deleteAssessmentFn: tc.serviceFn}
			r := newTestRouter(svc)

			req := httptest.NewRequest(http.MethodDelete, "/assessments/"+id.String(), nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRecordResultsHandler(t *testing.T) {
	assessmentID := uuid.New()
	studentA := uuid.New()

	tests := []struct {
		name       string
		body       any
		serviceFn  func(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error)
		wantStatus int
		wantCode   string
	}{
		{
			name: "success 201",
			body: map[string]any{
				"results": []map[string]any{
					{"student_id": studentA.String(), "score": 85.5, "grade": "A", "remarks": "Excellent"},
				},
			},
			serviceFn: func(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error) {
				now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
				return []db.ListAssessmentResultsRow{
					{
						ID:            uuid.New(),
						AssessmentID:  assessmentID,
						StudentID:     studentA,
						Score:         floatPtrToNumeric(ptr(85.5)),
						Grade:         pgtype.Text{String: "A", Valid: true},
						Remarks:       pgtype.Text{String: "Excellent", Valid: true},
						CreatedAt:     now,
						UpdatedAt:     now,
						StudentNumber: "STU-001",
						StudentName:   "Ama Owusu",
					},
				}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "duplicate student 400",
			body: map[string]any{
				"results": []map[string]any{
					{"student_id": studentA.String(), "score": 80.0},
					{"student_id": studentA.String(), "score": 90.0},
				},
			},
			serviceFn: func(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error) {
				return nil, ErrDuplicateStudent
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "student not enrolled 400",
			body: map[string]any{
				"results": []map[string]any{
					{"student_id": studentA.String(), "score": 80.0},
				},
			},
			serviceFn: func(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error) {
				return nil, ErrStudentNotEnrolled
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "invalid score bounds 400",
			body: map[string]any{
				"results": []map[string]any{
					{"student_id": studentA.String(), "score": 150.0},
				},
			},
			serviceFn: func(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error) {
				return nil, ErrInvalidScore
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{recordResultsFn: tc.serviceFn}
			r := newTestRouter(svc)

			payload, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/assessments/"+assessmentID.String()+"/results", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantCode != "" {
				var errResp struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
				if errResp.Error.Code != tc.wantCode {
					t.Errorf("error code %q, want %q", errResp.Error.Code, tc.wantCode)
				}
			}
		})
	}
}

func TestListAssessmentResultsHandler(t *testing.T) {
	assessmentID := uuid.New()
	svc := &stubService{
		listAssessmentResultsFn: func(ctx context.Context, id uuid.UUID) ([]db.ListAssessmentResultsRow, error) {
			now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
			return []db.ListAssessmentResultsRow{
				{
					ID:            uuid.New(),
					AssessmentID:  id,
					StudentID:     uuid.New(),
					Score:         floatPtrToNumeric(ptr(92.0)),
					Grade:         pgtype.Text{String: "A+", Valid: true},
					StudentNumber: "STU-001",
					StudentName:   "Ama Owusu",
					CreatedAt:     now,
					UpdatedAt:     now,
				},
			}, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/assessments/"+assessmentID.String()+"/results", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestGetResultHandler(t *testing.T) {
	id := uuid.New()
	svc := &stubService{
		getResultFn: func(ctx context.Context, id uuid.UUID) (db.GetResultByIDRow, error) {
			now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
			return db.GetResultByIDRow{
				ID:              id,
				AssessmentID:    uuid.New(),
				StudentID:       uuid.New(),
				Score:           floatPtrToNumeric(ptr(88.0)),
				Grade:           pgtype.Text{String: "A", Valid: true},
				AssessmentTitle: "Midterm Exam",
				AssessmentType:  TypeMidterm,
				MaxScore:        floatToNumeric(100),
				AssessmentDate:  pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
				ClassID:         uuid.New(),
				ClassName:       "Grade 10-A",
				SubjectCode:     "MATH",
				SubjectName:     "Mathematics",
				TermID:          uuid.New(),
				TermName:        "Term 1",
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/results/"+id.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateResultHandler(t *testing.T) {
	id := uuid.New()
	svc := &stubService{
		updateResultFn: func(ctx context.Context, actorID, id uuid.UUID, in UpdateResultInput) (db.GetResultByIDRow, error) {
			now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
			return db.GetResultByIDRow{
				ID:              id,
				AssessmentID:    uuid.New(),
				StudentID:       uuid.New(),
				Score:           floatPtrToNumeric(in.Score),
				Grade:           pgtype.Text{String: *in.Grade, Valid: true},
				AssessmentTitle: "Midterm Exam",
				AssessmentType:  TypeMidterm,
				MaxScore:        floatToNumeric(100),
				AssessmentDate:  pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
				ClassID:         uuid.New(),
				ClassName:       "Grade 10-A",
				SubjectCode:     "MATH",
				SubjectName:     "Mathematics",
				TermID:          uuid.New(),
				TermName:        "Term 1",
				CreatedAt:       now,
				UpdatedAt:       now,
			}, nil
		},
	}

	r := newTestRouter(svc)
	newScore := 95.0
	newGrade := "A+"
	payload, _ := json.Marshal(map[string]any{"score": newScore, "grade": newGrade})
	req := httptest.NewRequest(http.MethodPatch, "/results/"+id.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestListStudentResultsHandler(t *testing.T) {
	studentID := uuid.New()
	svc := &stubService{
		listStudentResultsFn: func(ctx context.Context, id uuid.UUID, termID, academicYearID *uuid.UUID) ([]db.ListStudentResultsRow, error) {
			now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
			return []db.ListStudentResultsRow{
				{
					ID:              uuid.New(),
					AssessmentID:    uuid.New(),
					StudentID:       id,
					Score:           floatPtrToNumeric(ptr(91.0)),
					Grade:           pgtype.Text{String: "A", Valid: true},
					AssessmentTitle: "Midterm Exam",
					AssessmentType:  TypeMidterm,
					MaxScore:        floatToNumeric(100),
					AssessmentDate:  pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
					ClassName:       "Grade 10-A",
					SubjectCode:     "MATH",
					SubjectName:     "Mathematics",
					TermID:          uuid.New(),
					TermName:        "Term 1",
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			}, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/students/"+studentID.String()+"/results", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []studentResultDTO `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
}
