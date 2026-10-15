package reports

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

type stubService struct {
	getEnrollmentReportFn       func(ctx context.Context, academicYearID uuid.UUID) (EnrollmentReport, error)
	getClassAttendanceReportFn  func(ctx context.Context, classID, termID uuid.UUID, startDate, endDate string) (ClassAttendanceReport, error)
	getClassPerformanceReportFn func(ctx context.Context, classID, termID, subjectID uuid.UUID) (ClassPerformanceReport, error)
	getStudentReportCardFn      func(ctx context.Context, studentID, termID uuid.UUID) (StudentReportCard, error)
}

func (s *stubService) GetEnrollmentReport(ctx context.Context, academicYearID uuid.UUID) (EnrollmentReport, error) {
	if s.getEnrollmentReportFn == nil {
		panic("unexpected GetEnrollmentReport call")
	}
	return s.getEnrollmentReportFn(ctx, academicYearID)
}

func (s *stubService) GetClassAttendanceReport(ctx context.Context, classID, termID uuid.UUID, startDate, endDate string) (ClassAttendanceReport, error) {
	if s.getClassAttendanceReportFn == nil {
		panic("unexpected GetClassAttendanceReport call")
	}
	return s.getClassAttendanceReportFn(ctx, classID, termID, startDate, endDate)
}

func (s *stubService) GetClassPerformanceReport(ctx context.Context, classID, termID, subjectID uuid.UUID) (ClassPerformanceReport, error) {
	if s.getClassPerformanceReportFn == nil {
		panic("unexpected GetClassPerformanceReport call")
	}
	return s.getClassPerformanceReportFn(ctx, classID, termID, subjectID)
}

func (s *stubService) GetStudentReportCard(ctx context.Context, studentID, termID uuid.UUID) (StudentReportCard, error) {
	if s.getStudentReportCardFn == nil {
		panic("unexpected GetStudentReportCard call")
	}
	return s.getStudentReportCardFn(ctx, studentID, termID)
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
	r.GET("/reports/enrollment", h.GetEnrollmentReport)
	r.GET("/reports/attendance/classes/:id", h.GetClassAttendanceReport)
	r.GET("/reports/performance/classes/:id", h.GetClassPerformanceReport)
	r.GET("/reports/students/:id/report-card", h.GetStudentReportCard)
	return r
}

func makeTestStaff() auth.AuthenticatedUser {
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uuid.New(),
			Email:  "staff@schoolhub.test",
			Status: "ACTIVE",
		},
		Person: db.Person{
			FirstName: "Sarah",
			LastName:  "Admin",
		},
		Roles:       []string{"STAFF"},
		Permissions: []string{"report.view"},
	}
}

func TestGetEnrollmentReport_Success(t *testing.T) {
	staff := makeTestStaff()
	yearID := uuid.New().String()
	util := 85.0

	svc := &stubService{
		getEnrollmentReportFn: func(ctx context.Context, ayID uuid.UUID) (EnrollmentReport, error) {
			return EnrollmentReport{
				AcademicYearID:     &yearID,
				AcademicYearName:   &yearID,
				TotalClasses:       2,
				TotalEnrolled:      50,
				TotalCapacity:      60,
				OverallUtilization: &util,
				TotalMale:          25,
				TotalFemale:        25,
				Classes: []EnrollmentClassSummary{
					{
						ClassID:         uuid.New().String(),
						ClassName:       "Grade 10A",
						EnrolledCount:   25,
						UtilizationRate: &util,
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/enrollment", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data EnrollmentReport `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.TotalEnrolled != 50 {
		t.Errorf("expected 50 total enrolled, got %d", resp.Data.TotalEnrolled)
	}
	if len(resp.Data.Classes) != 1 {
		t.Errorf("expected 1 class, got %d", len(resp.Data.Classes))
	}
}

func TestGetEnrollmentReport_InvalidYearUUID(t *testing.T) {
	staff := makeTestStaff()
	svc := &stubService{}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/enrollment?academic_year_id=not-a-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad year UUID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetEnrollmentReport_Unauthenticated(t *testing.T) {
	svc := &stubService{}
	router := setupTestRouter(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/reports/enrollment", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthenticated, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetEnrollmentReport_InternalError(t *testing.T) {
	staff := makeTestStaff()
	svc := &stubService{
		getEnrollmentReportFn: func(ctx context.Context, ayID uuid.UUID) (EnrollmentReport, error) {
			return EnrollmentReport{}, errors.New("database connection failed")
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/enrollment", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetClassAttendanceReport_Success(t *testing.T) {
	staff := makeTestStaff()
	classID := uuid.New()
	termID := uuid.New()

	svc := &stubService{
		getClassAttendanceReportFn: func(ctx context.Context, cID, tID uuid.UUID, sDate, eDate string) (ClassAttendanceReport, error) {
			if cID != classID {
				t.Fatalf("expected classID %s, got %s", classID, cID)
			}
			if tID != termID {
				t.Fatalf("expected termID %s, got %s", termID, tID)
			}
			return ClassAttendanceReport{
				ClassID:               classID.String(),
				ClassName:             "Grade 10A",
				TotalSessions:         20,
				TotalStudents:         1,
				AverageAttendanceRate: 95.0,
				Students: []ClassAttendanceStudentSummary{
					{
						StudentID:      uuid.New().String(),
						StudentNumber:  "STU-001",
						FirstName:      "John",
						LastName:       "Doe",
						PresentCount:   19,
						AbsentCount:    1,
						TotalRecords:   20,
						AttendanceRate: 95.0,
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/attendance/classes/"+classID.String()+"?term_id="+termID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data ClassAttendanceReport `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.ClassName != "Grade 10A" || len(resp.Data.Students) != 1 {
		t.Errorf("unexpected attendance report payload: %+v", resp.Data)
	}
}

func TestGetClassAttendanceReport_InvalidClassUUID(t *testing.T) {
	staff := makeTestStaff()
	svc := &stubService{}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/attendance/classes/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad class UUID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetClassAttendanceReport_InvalidDateRange(t *testing.T) {
	staff := makeTestStaff()
	classID := uuid.New()
	svc := &stubService{
		getClassAttendanceReportFn: func(ctx context.Context, cID, tID uuid.UUID, sDate, eDate string) (ClassAttendanceReport, error) {
			return ClassAttendanceReport{}, ErrInvalidDateRange
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/attendance/classes/"+classID.String()+"?start_date=2026-10-10&end_date=2026-10-01", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad date range, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetClassAttendanceReport_ClassNotFound(t *testing.T) {
	staff := makeTestStaff()
	classID := uuid.New()
	svc := &stubService{
		getClassAttendanceReportFn: func(ctx context.Context, cID, tID uuid.UUID, sDate, eDate string) (ClassAttendanceReport, error) {
			return ClassAttendanceReport{}, ErrClassNotFound
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/attendance/classes/"+classID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for class not found, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetClassPerformanceReport_Success(t *testing.T) {
	staff := makeTestStaff()
	classID := uuid.New()
	avg := 78.5

	svc := &stubService{
		getClassPerformanceReportFn: func(ctx context.Context, cID, tID, sID uuid.UUID) (ClassPerformanceReport, error) {
			return ClassPerformanceReport{
				ClassID:   classID.String(),
				ClassName: "Grade 10A",
				Assessments: []AssessmentPerformanceItem{
					{
						AssessmentID:      uuid.New().String(),
						AssessmentTitle:   "Midterm Exam",
						AssessmentType:    "EXAM",
						SubjectCode:       "MATH-101",
						SubjectName:       "Mathematics",
						MaxScore:          100,
						AverageScore:      78.5,
						AveragePercentage: &avg,
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/performance/classes/"+classID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data ClassPerformanceReport `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Data.Assessments) != 1 || resp.Data.Assessments[0].SubjectCode != "MATH-101" {
		t.Errorf("unexpected performance report payload: %+v", resp.Data)
	}
}

func TestGetStudentReportCard_Success(t *testing.T) {
	staff := makeTestStaff()
	studentID := uuid.New()
	termID := uuid.New()
	overall := 84.5
	grade := "A"

	svc := &stubService{
		getStudentReportCardFn: func(ctx context.Context, sID, tID uuid.UUID) (StudentReportCard, error) {
			if sID != studentID || tID != termID {
				t.Fatalf("unexpected IDs passed to service: sID=%s, tID=%s", sID, tID)
			}
			return StudentReportCard{
				StudentID:                studentID.String(),
				StudentNumber:            "STU-001",
				FirstName:                "Alex",
				LastName:                 "Smith",
				ClassName:                "Grade 10A",
				AcademicYearName:         "2026/2027",
				TermName:                 "Term 1",
				OverallAveragePercentage: &overall,
				OverallGrade:             &grade,
				Attendance: ReportCardAttendance{
					PresentCount:   38,
					AbsentCount:    2,
					TotalSessions:  40,
					AttendanceRate: 95.0,
				},
				Subjects: []ReportCardSubjectItem{
					{
						SubjectCode:       "ENG-101",
						SubjectName:       "English",
						AssessmentsTaken:  3,
						TotalScore:        260,
						TotalMaxScore:     300,
						AveragePercentage: 86.7,
						Grade:             "A",
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/students/"+studentID.String()+"/report-card?term_id="+termID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data StudentReportCard `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.StudentNumber != "STU-001" || resp.Data.Attendance.AttendanceRate != 95.0 {
		t.Errorf("unexpected report card: %+v", resp.Data)
	}
}

func TestGetStudentReportCard_MissingTermID(t *testing.T) {
	staff := makeTestStaff()
	studentID := uuid.New()
	svc := &stubService{}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/students/"+studentID.String()+"/report-card", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing term_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetStudentReportCard_StudentNotFound(t *testing.T) {
	staff := makeTestStaff()
	studentID := uuid.New()
	termID := uuid.New()
	svc := &stubService{
		getStudentReportCardFn: func(ctx context.Context, sID, tID uuid.UUID) (StudentReportCard, error) {
			return StudentReportCard{}, ErrStudentNotFound
		},
	}

	router := setupTestRouter(svc, &staff)
	req := httptest.NewRequest(http.MethodGet, "/reports/students/"+studentID.String()+"/report-card?term_id="+termID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for student not found, got %d: %s", w.Code, w.Body.String())
	}
}
