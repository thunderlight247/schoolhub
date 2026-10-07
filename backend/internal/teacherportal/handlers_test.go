package teacherportal

import (
	"context"
	"encoding/json"
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
	getTeacherOverviewFn func(ctx context.Context, userID uuid.UUID) (TeacherOverviewData, error)
	getTeacherClassesFn  func(ctx context.Context, userID uuid.UUID) ([]AssignedClass, error)
	getClassRosterFn     func(ctx context.Context, userID uuid.UUID, classID uuid.UUID) ([]StudentRosterItem, error)
}

func (s *stubService) GetTeacherOverview(ctx context.Context, userID uuid.UUID) (TeacherOverviewData, error) {
	if s.getTeacherOverviewFn == nil {
		panic("unexpected GetTeacherOverview call")
	}
	return s.getTeacherOverviewFn(ctx, userID)
}

func (s *stubService) GetTeacherClasses(ctx context.Context, userID uuid.UUID) ([]AssignedClass, error) {
	if s.getTeacherClassesFn == nil {
		panic("unexpected GetTeacherClasses call")
	}
	return s.getTeacherClassesFn(ctx, userID)
}

func (s *stubService) GetClassRoster(ctx context.Context, userID uuid.UUID, classID uuid.UUID) ([]StudentRosterItem, error) {
	if s.getClassRosterFn == nil {
		panic("unexpected GetClassRoster call")
	}
	return s.getClassRosterFn(ctx, userID, classID)
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
	r.GET("/portal/teacher/overview", h.GetTeacherOverview)
	r.GET("/portal/teacher/classes", h.GetTeacherClasses)
	r.GET("/portal/teacher/classes/:id/roster", h.GetClassRoster)
	return r
}

func makeTestTeacher() auth.AuthenticatedUser {
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uuid.New(),
			Email:  "ada@schoolhub.test",
			Status: "ACTIVE",
		},
		Person: db.Person{
			FirstName: "Ada",
			LastName:  "Obi",
		},
		Roles:       []string{"TEACHER"},
		Permissions: []string{"class.view", "attendance.view", "attendance.record", "assessment.view", "assessment.create"},
	}
}

func TestGetTeacherOverview_Success(t *testing.T) {
	teacher := makeTestTeacher()
	ayName := "2026/2027"
	tName := "Term 1"
	dept := "Mathematics"
	pos := "Senior Teacher"

	svc := &stubService{
		getTeacherOverviewFn: func(ctx context.Context, userID uuid.UUID) (TeacherOverviewData, error) {
			if userID != teacher.User.ID {
				t.Fatalf("expected userID %s, got %s", teacher.User.ID, userID)
			}
			return TeacherOverviewData{
				Profile: TeacherProfile{
					StaffID:          uuid.New().String(),
					StaffNumber:      "STF-001",
					FirstName:        teacher.Person.FirstName,
					LastName:         teacher.Person.LastName,
					Email:            teacher.User.Email,
					DepartmentName:   &dept,
					PositionTitle:    &pos,
					EmploymentStatus: "ACTIVE",
				},
				AcademicContext: AcademicContext{
					AcademicYearName: &ayName,
					TermName:         &tName,
				},
				Counts: TeacherCounts{
					TotalClasses:            3,
					TotalSubjects:           2,
					TotalStudents:           75,
					TotalAttendanceSessions: 12,
					TotalAssessments:        4,
				},
				AssignedClasses: []AssignedClass{
					{
						AssignmentID:          uuid.New().String(),
						ClassID:               uuid.New().String(),
						ClassName:             "Grade 10A",
						ClassCapacity:         30,
						AcademicYearID:        uuid.New().String(),
						AcademicYearName:      ayName,
						SubjectID:             uuid.New().String(),
						SubjectName:           "Mathematics",
						SubjectCode:           "MATH10",
						SubjectType:           "CORE",
						EnrolledStudentsCount: 28,
					},
				},
				RecentAttendance: []RecentAttendanceSession{
					{
						SessionID:    uuid.New().String(),
						SessionDate:  "2026-10-06",
						ClassID:      uuid.New().String(),
						ClassName:    "Grade 10A",
						SubjectID:    uuid.New().String(),
						SubjectName:  "Mathematics",
						SubjectCode:  "MATH10",
						TermName:     tName,
						PresentCount: 26,
						AbsentCount:  1,
						LateCount:    1,
						ExcusedCount: 0,
						TotalMarks:   28,
					},
				},
				RecentAssessments: []RecentAssessment{
					{
						AssessmentID:       uuid.New().String(),
						Title:              "Algebra Quiz 1",
						AssessmentType:     "QUIZ",
						MaxScore:           20.0,
						AssessmentDate:     "2026-10-05",
						ClassID:            uuid.New().String(),
						ClassName:          "Grade 10A",
						SubjectID:          uuid.New().String(),
						SubjectName:        "Mathematics",
						SubjectCode:        "MATH10",
						TermName:           tName,
						ResultsCount:       28,
						TotalStudentsCount: 28,
					},
				},
				Announcements: []TeacherAnnouncement{
					{
						ID:         uuid.New().String(),
						Title:      "Staff Meeting on Friday",
						Content:    "All staff members please attend.",
						Priority:   "NORMAL",
						Status:     "PUBLISHED",
						TargetType: "ROLE",
						AuthorName: "Principal",
						CreatedAt:  time.Now().Format(time.RFC3339),
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, teacher)
	req := httptest.NewRequest(http.MethodGet, "/portal/teacher/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data TeacherOverviewData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.Profile.StaffNumber != "STF-001" {
		t.Fatalf("expected staff number STF-001, got %s", res.Data.Profile.StaffNumber)
	}
	if res.Data.Counts.TotalStudents != 75 {
		t.Fatalf("expected 75 total students, got %d", res.Data.Counts.TotalStudents)
	}
	if len(res.Data.AssignedClasses) != 1 || res.Data.AssignedClasses[0].ClassName != "Grade 10A" {
		t.Fatalf("unexpected assigned classes: %+v", res.Data.AssignedClasses)
	}
}

func TestGetTeacherOverview_NotFound(t *testing.T) {
	teacher := makeTestTeacher()

	svc := &stubService{
		getTeacherOverviewFn: func(ctx context.Context, userID uuid.UUID) (TeacherOverviewData, error) {
			return TeacherOverviewData{}, ErrTeacherNotFound
		},
	}

	router := setupTestRouter(svc, teacher)
	req := httptest.NewRequest(http.MethodGet, "/portal/teacher/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestGetTeacherClasses_Success(t *testing.T) {
	teacher := makeTestTeacher()
	classID := uuid.New()

	svc := &stubService{
		getTeacherClassesFn: func(ctx context.Context, userID uuid.UUID) ([]AssignedClass, error) {
			return []AssignedClass{
				{
					AssignmentID:          uuid.New().String(),
					ClassID:               classID.String(),
					ClassName:             "Grade 11B",
					ClassCapacity:         35,
					AcademicYearID:        uuid.New().String(),
					AcademicYearName:      "2026/2027",
					SubjectID:             uuid.New().String(),
					SubjectName:           "Physics",
					SubjectCode:           "PHYS11",
					SubjectType:           "CORE",
					EnrolledStudentsCount: 32,
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, teacher)
	req := httptest.NewRequest(http.MethodGet, "/portal/teacher/classes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res struct {
		Data []AssignedClass `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(res.Data) != 1 || res.Data[0].ClassName != "Grade 11B" {
		t.Fatalf("unexpected data: %+v", res.Data)
	}
}

func TestGetClassRoster_SuccessAndForbidden(t *testing.T) {
	teacher := makeTestTeacher()
	assignedClassID := uuid.New()
	unassignedClassID := uuid.New()
	phone := "+1234567890"
	rel := "Mother"
	guardianName := "Grace Doe"

	svc := &stubService{
		getClassRosterFn: func(ctx context.Context, userID uuid.UUID, classID uuid.UUID) ([]StudentRosterItem, error) {
			if classID == unassignedClassID {
				return nil, ErrForbiddenClassAccess
			}
			return []StudentRosterItem{
				{
					StudentID:            uuid.New().String(),
					StudentNumber:        "STU-001",
					FirstName:            "John",
					LastName:             "Doe",
					EnrollmentStatus:     "ENROLLED",
					EnrolledAt:           time.Now().Format(time.RFC3339),
					GuardianName:         &guardianName,
					GuardianPhone:        &phone,
					GuardianRelationship: &rel,
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, teacher)

	// Success case
	req := httptest.NewRequest(http.MethodGet, "/portal/teacher/classes/"+assignedClassID.String()+"/roster", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data []StudentRosterItem `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(res.Data) != 1 || *res.Data[0].GuardianName != "Grace Doe" {
		t.Fatalf("unexpected roster data: %+v", res.Data)
	}

	// Forbidden case: teacher not assigned to class
	req = httptest.NewRequest(http.MethodGet, "/portal/teacher/classes/"+unassignedClassID.String()+"/roster", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}

	// Bad UUID case
	req = httptest.NewRequest(http.MethodGet, "/portal/teacher/classes/invalid-uuid/roster", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}
}
