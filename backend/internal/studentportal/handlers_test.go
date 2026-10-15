package studentportal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type stubService struct {
	getStudentOverviewFn        func(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error)
	listStudentAttendanceFn     func(ctx context.Context, userID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error)
	listStudentResultsHistoryFn func(ctx context.Context, userID uuid.UUID, termID, academicYearID uuid.UUID) ([]StudentResult, error)
	listStudentEnrollmentsFn    func(ctx context.Context, userID uuid.UUID) ([]StudentEnrollmentHistoryItem, error)
}

func (s *stubService) GetStudentOverview(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error) {
	if s.getStudentOverviewFn == nil {
		panic("unexpected GetStudentOverview call")
	}
	return s.getStudentOverviewFn(ctx, userID)
}

func (s *stubService) ListStudentAttendance(ctx context.Context, userID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error) {
	if s.listStudentAttendanceFn == nil {
		panic("unexpected ListStudentAttendance call")
	}
	return s.listStudentAttendanceFn(ctx, userID, limit)
}

func (s *stubService) ListStudentResultsHistory(ctx context.Context, userID uuid.UUID, termID, academicYearID uuid.UUID) ([]StudentResult, error) {
	if s.listStudentResultsHistoryFn == nil {
		panic("unexpected ListStudentResultsHistory call")
	}
	return s.listStudentResultsHistoryFn(ctx, userID, termID, academicYearID)
}

func (s *stubService) ListStudentEnrollments(ctx context.Context, userID uuid.UUID) ([]StudentEnrollmentHistoryItem, error) {
	if s.listStudentEnrollmentsFn == nil {
		panic("unexpected ListStudentEnrollments call")
	}
	return s.listStudentEnrollmentsFn(ctx, userID)
}

func setupTestRouter(svc ServiceAPI, authUser *auth.AuthenticatedUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	if authUser != nil {
		r.Use(func(c *gin.Context) {
			auth.WithUser(c, *authUser)
			c.Next()
		})
	}
	h := NewHandler(svc)
	r.GET("/portal/student/overview", h.GetStudentOverview)
	r.GET("/portal/student/attendance", h.GetStudentAttendance)
	r.GET("/portal/student/results", h.GetStudentResultsHistory)
	r.GET("/portal/student/enrollments", h.GetStudentEnrollments)
	return r
}

func makeTestStudent() auth.AuthenticatedUser {
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uuid.New(),
			Email:  "student@schoolhub.test",
			Status: "ACTIVE",
		},
		Person: db.Person{
			FirstName: "Alex",
			LastName:  "Smith",
		},
		Roles:       []string{"STUDENT"},
		Permissions: []string{"attendance.view", "result.view", "announcement.view"},
	}
}

func TestGetStudentOverview_Success(t *testing.T) {
	student := makeTestStudent()
	studentID := uuid.New().String()
	avg := 88.5
	progName := "General Science"

	svc := &stubService{
		getStudentOverviewFn: func(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error) {
			if userID != student.User.ID {
				t.Fatalf("expected userID %s, got %s", student.User.ID, userID)
			}
			return StudentOverviewData{
				Profile: StudentProfile{
					StudentID:     studentID,
					StudentNumber: "STU-2026-001",
					FirstName:     "Alex",
					LastName:      "Smith",
					StudentStatus: "ACTIVE",
				},
				Enrollment: &StudentEnrollmentContext{
					EnrollmentID:     uuid.New().String(),
					EnrollmentStatus: "ENROLLED",
					EnrollmentDate:   "2026-09-01",
					ClassID:          uuid.New().String(),
					ClassName:        "Grade 10A",
					ProgrammeName:    &progName,
					AcademicYearName: "2026/2027",
				},
				EnrollmentHistory: []StudentEnrollmentHistoryItem{
					{
						EnrollmentID:     uuid.New().String(),
						EnrollmentStatus: "ENROLLED",
						EnrollmentDate:   "2026-09-01",
						EnrolledAt:       time.Now().Format(time.RFC3339),
						ClassName:        "Grade 10A",
						AcademicYearName: "2026/2027",
					},
				},
				Attendance: AttendanceCounts{
					Present: 45,
					Absent:  2,
					Late:    1,
					Excused: 0,
				},
				Results: ResultsSummary{
					ResultsCount: 6,
					AverageScore: &avg,
				},
				RecentAttendance: []StudentAttendanceRecord{
					{
						RecordID:    uuid.New().String(),
						SessionDate: "2026-10-05",
						Status:      "PRESENT",
						ClassName:   "Grade 10A",
						SubjectName: "Mathematics",
						SubjectCode: "MATH-101",
						TermName:    "Term 1",
					},
				},
				RecentResults: []StudentResult{
					{
						ResultID:        uuid.New().String(),
						AssessmentTitle: "Midterm Exam",
						AssessmentType:  "EXAM",
						SubjectName:     "Mathematics",
						SubjectCode:     "MATH-101",
						ClassName:       "Grade 10A",
						TermName:        "Term 1",
						AssessmentDate:  "2026-10-04",
						MaxScore:        100,
						Score:           &avg,
					},
				},
				Announcements: []StudentAnnouncement{
					{
						ID:         uuid.New().String(),
						Title:      "School Assembly",
						Content:    "Assembly on Friday at 8am.",
						Priority:   "NORMAL",
						TargetType: "ALL",
						AuthorName: "Principal Davis",
						CreatedAt:  time.Now().Format(time.RFC3339),
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data StudentOverviewData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.Profile.StudentID != studentID {
		t.Errorf("expected student_id %s, got %s", studentID, resp.Data.Profile.StudentID)
	}
	if resp.Data.Profile.StudentNumber != "STU-2026-001" {
		t.Errorf("expected student_number STU-2026-001, got %s", resp.Data.Profile.StudentNumber)
	}
	if resp.Data.Enrollment == nil || resp.Data.Enrollment.ClassName != "Grade 10A" {
		t.Errorf("expected enrollment class Grade 10A, got %+v", resp.Data.Enrollment)
	}
	if len(resp.Data.EnrollmentHistory) != 1 {
		t.Errorf("expected 1 enrollment history item, got %d", len(resp.Data.EnrollmentHistory))
	}
	if resp.Data.Attendance.Present != 45 {
		t.Errorf("expected 45 present, got %d", resp.Data.Attendance.Present)
	}
	if resp.Data.Results.ResultsCount != 6 || resp.Data.Results.AverageScore == nil || *resp.Data.Results.AverageScore != avg {
		t.Errorf("unexpected results summary: %+v", resp.Data.Results)
	}
	if len(resp.Data.RecentAttendance) != 1 {
		t.Errorf("expected 1 attendance record, got %d", len(resp.Data.RecentAttendance))
	}
	if len(resp.Data.RecentResults) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Data.RecentResults))
	}
	if len(resp.Data.Announcements) != 1 {
		t.Errorf("expected 1 announcement, got %d", len(resp.Data.Announcements))
	}
}

func TestGetStudentOverview_NotFound(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{
		getStudentOverviewFn: func(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error) {
			return StudentOverviewData{}, ErrStudentNotFound
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStudentOverview_InternalError(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{
		getStudentOverviewFn: func(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error) {
			return StudentOverviewData{}, errors.New("db error")
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStudentOverview_Unauthenticated(t *testing.T) {
	svc := &stubService{}
	router := setupTestRouter(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStudentAttendance_Success(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{
		listStudentAttendanceFn: func(ctx context.Context, userID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error) {
			if limit != 10 {
				t.Fatalf("expected limit 10, got %d", limit)
			}
			return []StudentAttendanceRecord{
				{
					RecordID:    uuid.New().String(),
					SessionDate: "2026-10-05",
					Status:      "PRESENT",
					ClassName:   "Grade 10A",
					SubjectName: "Mathematics",
					SubjectCode: "MATH-101",
					TermName:    "Term 1",
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/attendance?limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []StudentAttendanceRecord `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 record, got %d", len(resp.Data))
	}
}

func TestGetStudentAttendance_InvalidLimit(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/attendance?limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStudentAttendance_NotFound(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{
		listStudentAttendanceFn: func(ctx context.Context, userID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error) {
			return nil, ErrStudentNotFound
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/attendance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStudentResultsHistory_Success(t *testing.T) {
	student := makeTestStudent()
	termID := uuid.New()
	score := 92.0
	svc := &stubService{
		listStudentResultsHistoryFn: func(ctx context.Context, userID uuid.UUID, tID, ayID uuid.UUID) ([]StudentResult, error) {
			if tID != termID {
				t.Fatalf("expected termID %s, got %s", termID, tID)
			}
			return []StudentResult{
				{
					ResultID:        uuid.New().String(),
					AssessmentTitle: "Midterm Exam",
					AssessmentType:  "EXAM",
					SubjectName:     "Physics",
					SubjectCode:     "PHY-101",
					ClassName:       "Grade 10A",
					TermName:        "Term 1",
					AssessmentDate:  "2026-10-04",
					MaxScore:        100,
					Score:           &score,
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/results?term_id="+termID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []StudentResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].SubjectCode != "PHY-101" {
		t.Fatalf("unexpected results payload: %+v", resp.Data)
	}
}

func TestGetStudentResultsHistory_InvalidUUID(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/results?term_id=not-a-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/portal/student/results?academic_year_id=not-a-uuid", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad academic_year_id, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestGetStudentEnrollments_Success(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{
		listStudentEnrollmentsFn: func(ctx context.Context, userID uuid.UUID) ([]StudentEnrollmentHistoryItem, error) {
			return []StudentEnrollmentHistoryItem{
				{
					EnrollmentID:     uuid.New().String(),
					EnrollmentStatus: "ENROLLED",
					EnrollmentDate:   "2026-09-01",
					EnrolledAt:       time.Now().Format(time.RFC3339),
					ClassName:        "Grade 10A",
					AcademicYearName: "2026/2027",
				},
				{
					EnrollmentID:     uuid.New().String(),
					EnrollmentStatus: "COMPLETED",
					EnrollmentDate:   "2025-09-01",
					EnrolledAt:       time.Now().Add(-365 * 24 * time.Hour).Format(time.RFC3339),
					ClassName:        "Grade 9A",
					AcademicYearName: "2025/2026",
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/enrollments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data []StudentEnrollmentHistoryItem `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 enrollments, got %d", len(resp.Data))
	}
	if resp.Data[0].ClassName != "Grade 10A" || resp.Data[1].ClassName != "Grade 9A" {
		t.Errorf("unexpected enrollment classes: %+v", resp.Data)
	}
}

func TestGetStudentEnrollments_NotFound(t *testing.T) {
	student := makeTestStudent()
	svc := &stubService{
		listStudentEnrollmentsFn: func(ctx context.Context, userID uuid.UUID) ([]StudentEnrollmentHistoryItem, error) {
			return nil, ErrStudentNotFound
		},
	}

	router := setupTestRouter(svc, &student)
	req := httptest.NewRequest(http.MethodGet, "/portal/student/enrollments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
