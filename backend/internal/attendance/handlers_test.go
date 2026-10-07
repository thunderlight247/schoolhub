package attendance

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
	createFn     func(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error)
	getFn        func(ctx context.Context, id uuid.UUID) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error)
	listFn       func(ctx context.Context, f ListFilter) ([]db.ListAttendanceSessionsRow, int64, error)
	updateFn     func(ctx context.Context, actorID, id uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error)
	studentSumFn func(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) (Summary, error)
	classSumFn   func(ctx context.Context, classID uuid.UUID, termID, subjectID *uuid.UUID) (Summary, error)
}

func (s *stubService) CreateSession(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
	if s.createFn == nil {
		panic("unexpected CreateSession")
	}
	return s.createFn(ctx, actorID, in)
}

func (s *stubService) GetSession(ctx context.Context, id uuid.UUID) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
	if s.getFn == nil {
		panic("unexpected GetSession")
	}
	return s.getFn(ctx, id)
}

func (s *stubService) ListSessions(ctx context.Context, f ListFilter) ([]db.ListAttendanceSessionsRow, int64, error) {
	if s.listFn == nil {
		panic("unexpected ListSessions")
	}
	return s.listFn(ctx, f)
}

func (s *stubService) UpdateRecord(ctx context.Context, actorID, id uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error) {
	if s.updateFn == nil {
		panic("unexpected UpdateRecord")
	}
	return s.updateFn(ctx, actorID, id, status, notes)
}

func (s *stubService) StudentSummary(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) (Summary, error) {
	if s.studentSumFn == nil {
		panic("unexpected StudentSummary")
	}
	return s.studentSumFn(ctx, studentID, termID, academicYearID)
}

func (s *stubService) ClassSummary(ctx context.Context, classID uuid.UUID, termID, subjectID *uuid.UUID) (Summary, error) {
	if s.classSumFn == nil {
		panic("unexpected ClassSummary")
	}
	return s.classSumFn(ctx, classID, termID, subjectID)
}

func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{User: db.User{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}})
		c.Next()
	})

	h := NewHandler(svc)
	r.GET("/attendance/sessions", h.ListSessions)
	r.GET("/attendance/sessions/:id", h.GetSession)
	r.GET("/students/:id/attendance/summary", h.StudentSummary)
	r.GET("/classes/:id/attendance/summary", h.ClassSummary)
	r.POST("/attendance/sessions", h.CreateSession)
	r.PATCH("/attendance/records/:id", h.UpdateRecord)
	return r
}

func sessionDetailRow(id, classID, subjectID, termID uuid.UUID) db.GetAttendanceSessionByIDRow {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return db.GetAttendanceSessionByIDRow{
		ID:               id,
		ClassID:          classID,
		SubjectID:        subjectID,
		TermID:           termID,
		SessionDate:      pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
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

func recordRow(sessionID, studentID uuid.UUID, status string) db.ListAttendanceSessionRecordsRow {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return db.ListAttendanceSessionRecordsRow{
		ID:            uuid.New(),
		SessionID:     sessionID,
		StudentID:     studentID,
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
		StudentNumber: "STU-001",
		StudentName:   "Ama Owusu",
	}
}

func recordDetailRow(id, sessionID, studentID uuid.UUID, status string) db.GetAttendanceRecordByIDRow {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return db.GetAttendanceRecordByIDRow{
		ID:          id,
		SessionID:   sessionID,
		StudentID:   studentID,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
		StudentName: "Ama Owusu",
		SessionDate: pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
		ClassID:     uuid.New(),
		ClassName:   "Grade 10-A",
		SubjectCode: "MATH",
		SubjectName: "Mathematics",
		TermID:      uuid.New(),
		TermName:    "Term 1",
	}
}

func TestCreateSessionHandler(t *testing.T) {
	classID := uuid.New()
	subjectID := uuid.New()
	termID := uuid.New()
	studentA := uuid.New()
	studentB := uuid.New()

	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			if in.ClassID != classID || in.SubjectID != subjectID || in.TermID != termID {
				t.Errorf("unexpected chain ids: %+v", in)
			}
			if in.SessionDate.Format("2006-01-02") != "2026-10-05" {
				t.Errorf("expected session date 2026-10-05, got %s", in.SessionDate)
			}
			if len(in.Records) != 2 {
				t.Fatalf("expected 2 records, got %d", len(in.Records))
			}
			if in.Records[0].StudentID != studentA || in.Records[0].Status != "PRESENT" {
				t.Errorf("unexpected record 0: %+v", in.Records[0])
			}
			if in.Records[1].StudentID != studentB || in.Records[1].Status != "ABSENT" {
				t.Errorf("unexpected record 1: %+v", in.Records[1])
			}
			if in.Records[1].Notes == nil || *in.Records[1].Notes != "sick" {
				t.Errorf("expected notes 'sick' on record 1, got %v", in.Records[1].Notes)
			}
			if actorID == uuid.Nil {
				t.Error("expected a non-nil actor id")
			}

			sessionID := uuid.New()
			return sessionDetailRow(sessionID, classID, subjectID, termID),
				[]db.ListAttendanceSessionRecordsRow{
					recordRow(sessionID, studentA, "PRESENT"),
					recordRow(sessionID, studentB, "ABSENT"),
				}, nil
		},
	}

	r := newTestRouter(svc)
	payload := `{"class_id":"` + classID.String() + `","subject_id":"` + subjectID.String() + `","term_id":"` + termID.String() + `","session_date":"2026-10-05","records":[{"student_id":"` + studentA.String() + `","status":"PRESENT"},{"student_id":"` + studentB.String() + `","status":"ABSENT","notes":"sick"}]}`
	req := httptest.NewRequest(http.MethodPost, "/attendance/sessions", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Session attendanceSessionDTO  `json:"session"`
		Records []attendanceRecordDTO `json:"records"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Session.SessionDate != "2026-10-05" || resp.Session.ClassName != "Grade 10-A" {
		t.Fatalf("unexpected session: %+v", resp.Session)
	}
	if len(resp.Records) != 2 || resp.Records[1].Status != "ABSENT" {
		t.Fatalf("unexpected records: %+v", resp.Records)
	}
}

func TestCreateSessionValidation(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			t.Error("service should not be called for an invalid payload")
			return db.GetAttendanceSessionByIDRow{}, nil, errors.New("unreachable")
		},
	}

	r := newTestRouter(svc)
	payload := `{"class_id":"` + uuid.New().String() + `","subject_id":"` + uuid.New().String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/attendance/sessions", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateSessionMalformedDate(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			t.Error("service should not be called for a malformed date")
			return db.GetAttendanceSessionByIDRow{}, nil, errors.New("unreachable")
		},
	}

	r := newTestRouter(svc)
	payload := `{"class_id":"` + uuid.New().String() + `","subject_id":"` + uuid.New().String() + `","term_id":"` + uuid.New().String() + `","session_date":"not-a-date","records":[{"student_id":"` + uuid.New().String() + `","status":"PRESENT"}]}`
	req := httptest.NewRequest(http.MethodPost, "/attendance/sessions", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateSessionConflict(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrDuplicateSession
		},
	}

	r := newTestRouter(svc)
	payload := `{"class_id":"` + uuid.New().String() + `","subject_id":"` + uuid.New().String() + `","term_id":"` + uuid.New().String() + `","records":[{"student_id":"` + uuid.New().String() + `","status":"PRESENT"}]}`
	req := httptest.NewRequest(http.MethodPost, "/attendance/sessions", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateSessionStudentNotEnrolled(t *testing.T) {
	svc := &stubService{
		createFn: func(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrStudentNotEnrolled
		},
	}

	r := newTestRouter(svc)
	payload := `{"class_id":"` + uuid.New().String() + `","subject_id":"` + uuid.New().String() + `","term_id":"` + uuid.New().String() + `","records":[{"student_id":"` + uuid.New().String() + `","status":"PRESENT"}]}`
	req := httptest.NewRequest(http.MethodPost, "/attendance/sessions", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListSessionsHandler(t *testing.T) {
	classID := uuid.New()
	termID := uuid.New()

	svc := &stubService{
		listFn: func(ctx context.Context, f ListFilter) ([]db.ListAttendanceSessionsRow, int64, error) {
			if f.ClassID == nil || *f.ClassID != classID {
				t.Errorf("expected class_id filter %s, got %v", classID, f.ClassID)
			}
			if f.TermID == nil || *f.TermID != termID {
				t.Errorf("expected term_id filter %s, got %v", termID, f.TermID)
			}
			sessionID := uuid.New()
			row := db.ListAttendanceSessionsRow{
				ID:               sessionID,
				ClassID:          classID,
				SubjectID:        uuid.New(),
				TermID:           termID,
				SessionDate:      pgtype.Date{Time: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), Valid: true},
				CreatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
				UpdatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
				ClassName:        "Grade 10-A",
				AcademicYearID:   uuid.New(),
				AcademicYearName: "2025/2026",
				SubjectCode:      "MATH",
				SubjectName:      "Mathematics",
				TermName:         "Term 1",
			}
			return []db.ListAttendanceSessionsRow{row}, 1, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/attendance/sessions?class_id="+classID.String()+"&term_id="+termID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Sessions   []attendanceSessionDTO `json:"sessions"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Sessions) != 1 || resp.Sessions[0].SessionDate != "2026-10-05" {
		t.Fatalf("unexpected sessions: %+v", resp.Sessions)
	}
	if resp.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Pagination.Total)
	}
}

func TestGetSessionHandler(t *testing.T) {
	id := uuid.New()
	studentID := uuid.New()

	svc := &stubService{
		getFn: func(ctx context.Context, got uuid.UUID) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			if got != id {
				t.Errorf("expected id %s, got %s", id, got)
			}
			return sessionDetailRow(id, uuid.New(), uuid.New(), uuid.New()),
				[]db.ListAttendanceSessionRecordsRow{recordRow(id, studentID, "LATE")}, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/attendance/sessions/"+id.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Session attendanceSessionDTO  `json:"session"`
		Records []attendanceRecordDTO `json:"records"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Session.ID != id.String() {
		t.Fatalf("expected session id %s, got %s", id, resp.Session.ID)
	}
	if len(resp.Records) != 1 || resp.Records[0].Status != "LATE" || resp.Records[0].StudentName != "Ama Owusu" {
		t.Fatalf("unexpected records: %+v", resp.Records)
	}
}

func TestGetSessionNotFound(t *testing.T) {
	svc := &stubService{
		getFn: func(ctx context.Context, id uuid.UUID) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrSessionNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/attendance/sessions/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRecordHandler(t *testing.T) {
	id := uuid.New()
	sessionID := uuid.New()
	studentID := uuid.New()

	svc := &stubService{
		updateFn: func(ctx context.Context, actorID, gotID uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error) {
			if gotID != id {
				t.Errorf("expected record id %s, got %s", id, gotID)
			}
			if status != "ABSENT" {
				t.Errorf("expected status ABSENT, got %s", status)
			}
			if notes != nil {
				t.Errorf("expected nil notes to be passed through, got %v", *notes)
			}
			return recordDetailRow(id, sessionID, studentID, "ABSENT"), nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/attendance/records/"+id.String(), bytes.NewBufferString(`{"status":"ABSENT"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Record attendanceRecordDetailDTO `json:"record"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Record.Status != "ABSENT" || resp.Record.SessionDate != "2026-10-05" {
		t.Fatalf("unexpected record: %+v", resp.Record)
	}
}

func TestUpdateRecordInvalidStatus(t *testing.T) {
	svc := &stubService{
		updateFn: func(ctx context.Context, actorID, id uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error) {
			return db.GetAttendanceRecordByIDRow{}, ErrInvalidStatus
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/attendance/records/"+uuid.New().String(), bytes.NewBufferString(`{"status":"MAYBE"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateRecordNotFound(t *testing.T) {
	svc := &stubService{
		updateFn: func(ctx context.Context, actorID, id uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error) {
			return db.GetAttendanceRecordByIDRow{}, ErrRecordNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/attendance/records/"+uuid.New().String(), bytes.NewBufferString(`{"status":"PRESENT"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStudentSummaryHandler(t *testing.T) {
	studentID := uuid.New()
	termID := uuid.New()

	svc := &stubService{
		studentSumFn: func(ctx context.Context, got uuid.UUID, gotTerm, gotYear *uuid.UUID) (Summary, error) {
			if got != studentID {
				t.Errorf("expected student %s, got %s", studentID, got)
			}
			if gotTerm == nil || *gotTerm != termID {
				t.Errorf("expected term filter %s, got %v", termID, gotTerm)
			}
			if gotYear != nil {
				t.Errorf("expected no year filter, got %v", *gotYear)
			}
			counts := map[string]int64{
				StatusPresent: 18,
				StatusAbsent:  1,
				StatusLate:    2,
				StatusExcused: 0,
			}
			return Summary{StudentID: &studentID, TermID: gotTerm, Counts: counts, TotalRecords: 21}, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/students/"+studentID.String()+"/attendance/summary?term_id="+termID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Summary summaryDTO `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Summary.StudentID != studentID.String() {
		t.Fatalf("unexpected summary: %+v", resp.Summary)
	}
	if resp.Summary.Counts["PRESENT"] != 18 || resp.Summary.Counts["LATE"] != 2 {
		t.Fatalf("unexpected counts: %+v", resp.Summary.Counts)
	}
	if resp.Summary.TotalRecords != 21 {
		t.Fatalf("expected total_records 21, got %d", resp.Summary.TotalRecords)
	}
}

func TestStudentSummaryNotFound(t *testing.T) {
	svc := &stubService{
		studentSumFn: func(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) (Summary, error) {
			return Summary{}, ErrStudentNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/students/"+uuid.New().String()+"/attendance/summary", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestClassSummaryHandler(t *testing.T) {
	classID := uuid.New()
	subjectID := uuid.New()

	svc := &stubService{
		classSumFn: func(ctx context.Context, got uuid.UUID, gotTerm, gotSubject *uuid.UUID) (Summary, error) {
			if got != classID {
				t.Errorf("expected class %s, got %s", classID, got)
			}
			if gotSubject == nil || *gotSubject != subjectID {
				t.Errorf("expected subject filter %s, got %v", subjectID, gotSubject)
			}
			counts := map[string]int64{
				StatusPresent: 30,
				StatusAbsent:  2,
				StatusLate:    1,
				StatusExcused: 0,
			}
			return Summary{ClassID: &classID, SubjectID: gotSubject, Counts: counts, TotalRecords: 33}, nil
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/classes/"+classID.String()+"/attendance/summary?subject_id="+subjectID.String(), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Summary summaryDTO `json:"summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Summary.ClassID != classID.String() || resp.Summary.TotalRecords != 33 {
		t.Fatalf("unexpected summary: %+v", resp.Summary)
	}
}

func TestClassSummaryNotFound(t *testing.T) {
	svc := &stubService{
		classSumFn: func(ctx context.Context, classID uuid.UUID, termID, subjectID *uuid.UUID) (Summary, error) {
			return Summary{}, ErrClassNotFound
		},
	}

	r := newTestRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/classes/"+uuid.New().String()+"/attendance/summary", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
