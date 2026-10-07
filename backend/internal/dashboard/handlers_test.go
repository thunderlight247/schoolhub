package dashboard

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
	getAdminDashboardFn func(ctx context.Context) (AdminDashboardData, error)
}

func (s *stubService) GetAdminDashboard(ctx context.Context) (AdminDashboardData, error) {
	if s.getAdminDashboardFn == nil {
		panic("unexpected GetAdminDashboard call")
	}
	return s.getAdminDashboardFn(ctx)
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
	r.GET("/dashboard/admin", h.GetAdminDashboard)
	return r
}

func makeTestAdmin() auth.AuthenticatedUser {
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uuid.New(),
			Email:  "admin@schoolhub.test",
			Status: "ACTIVE",
		},
		Person: db.Person{
			FirstName: "Admin",
			LastName:  "User",
		},
		Roles:       []string{"ADMIN"},
		Permissions: []string{"audit.view", "class.view", "staff.view", "student.view"},
	}
}

func TestGetAdminDashboard_Success(t *testing.T) {
	admin := makeTestAdmin()
	ayName := "2026/2027"
	tName := "Term 1"

	svc := &stubService{
		getAdminDashboardFn: func(ctx context.Context) (AdminDashboardData, error) {
			return AdminDashboardData{
				Counts: DashboardCounts{
					ActiveStudents:    150,
					ActiveStaff:       25,
					ActiveTeachers:    18,
					TotalClasses:      8,
					ActiveEnrollments: 145,
				},
				AcademicContext: AcademicContext{
					AcademicYearName: &ayName,
					TermName:         &tName,
				},
				Attendance: AttendanceSummary{
					Date:           "2026-10-06",
					TotalSessions:  8,
					PresentCount:   140,
					AbsentCount:    5,
					LateCount:      3,
					ExcusedCount:   2,
					TotalMarks:     150,
					AttendanceRate: 95.3,
				},
				RecentAnnouncements: []RecentAnnouncement{
					{
						ID:         uuid.New().String(),
						Title:      "Exam Schedule Released",
						Status:     "PUBLISHED",
						TargetType: "ALL",
						Priority:   "HIGH",
						CreatedAt:  time.Now().Format(time.RFC3339),
						AuthorName: "Principal Smith",
					},
				},
				RecentActivity: []RecentActivity{
					{
						ID:           uuid.New().String(),
						Action:       "STUDENT_CREATED",
						ResourceType: "student",
						ActorEmail:   "admin@schoolhub.test",
						ActorName:    "Admin User",
						CreatedAt:    time.Now().Format(time.RFC3339),
					},
				},
				QuickActions: defaultQuickActions,
			}, nil
		},
	}

	router := setupTestRouter(svc, admin)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data AdminDashboardData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.Counts.ActiveStudents != 150 {
		t.Fatalf("expected 150 active students, got %d", res.Data.Counts.ActiveStudents)
	}
	if res.Data.Attendance.AttendanceRate != 95.3 {
		t.Fatalf("expected 95.3 attendance rate, got %f", res.Data.Attendance.AttendanceRate)
	}
	if len(res.Data.QuickActions) != len(defaultQuickActions) {
		t.Fatalf("expected %d quick actions, got %d", len(defaultQuickActions), len(res.Data.QuickActions))
	}
}

func TestGetAdminDashboard_InternalError(t *testing.T) {
	admin := makeTestAdmin()

	svc := &stubService{
		getAdminDashboardFn: func(ctx context.Context) (AdminDashboardData, error) {
			return AdminDashboardData{}, errors.New("db connection failure")
		},
	}

	router := setupTestRouter(svc, admin)
	req := httptest.NewRequest(http.MethodGet, "/dashboard/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d", w.Code)
	}
}
