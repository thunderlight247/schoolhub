package parentportal

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
	getParentOverviewFn func(ctx context.Context, userID uuid.UUID) (ParentOverviewData, error)
	getChildDetailFn    func(ctx context.Context, userID uuid.UUID, studentID uuid.UUID) (ChildDetail, error)
}

func (s *stubService) GetParentOverview(ctx context.Context, userID uuid.UUID) (ParentOverviewData, error) {
	if s.getParentOverviewFn == nil {
		panic("unexpected GetParentOverview call")
	}
	return s.getParentOverviewFn(ctx, userID)
}

func (s *stubService) GetChildDetail(ctx context.Context, userID uuid.UUID, studentID uuid.UUID) (ChildDetail, error) {
	if s.getChildDetailFn == nil {
		panic("unexpected GetChildDetail call")
	}
	return s.getChildDetailFn(ctx, userID, studentID)
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
	r.GET("/portal/parent/overview", h.GetParentOverview)
	r.GET("/portal/parent/children/:id", h.GetChildDetail)
	return r
}

func makeTestParent() auth.AuthenticatedUser {
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uuid.New(),
			Email:  "grace@schoolhub.test",
			Status: "ACTIVE",
		},
		Person: db.Person{
			FirstName: "Grace",
			LastName:  "Doe",
		},
		Roles:       []string{"PARENT"},
		Permissions: []string{"attendance.view", "result.view", "announcement.view"},
	}
}

func TestGetParentOverview_Success(t *testing.T) {
	parent := makeTestParent()
	phone := "+2348012345670"
	guardianID := uuid.New().String()
	classID := uuid.New().String()
	avg := 82.5

	svc := &stubService{
		getParentOverviewFn: func(ctx context.Context, userID uuid.UUID) (ParentOverviewData, error) {
			if userID != parent.User.ID {
				t.Fatalf("expected userID %s, got %s", parent.User.ID, userID)
			}
			return ParentOverviewData{
				Profile: GuardianProfile{
					GuardianID: guardianID,
					FirstName:  "Grace",
					LastName:   "Doe",
					Phone:      &phone,
				},
				Children: []ChildSummary{
					{
						StudentID:     uuid.New().String(),
						StudentNumber: "STU-101",
						FirstName:     "John",
						LastName:      "Doe",
						Relationship:  "Mother",
						StudentStatus: "ACTIVE",
						Enrollment: ChildEnrollmentContext{
							ClassID:          classID,
							ClassName:        "Grade 9A",
							AcademicYearName: "2026/2027",
						},
						AttendanceCounts: ChildAttendanceCounts{Present: 40, Absent: 2, Late: 1},
						ResultsCount:     8,
						AverageScore:     &avg,
					},
				},
				FeedAnnouncements: []ParentAnnouncement{
					{
						ID:         uuid.New().String(),
						Title:      "PTA Meeting",
						Content:    "Monthly PTA meeting on Friday.",
						Priority:   "NORMAL",
						TargetType: "ALL",
						AuthorName: "Principal",
						CreatedAt:  time.Now().Format(time.RFC3339),
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, parent)
	req := httptest.NewRequest(http.MethodGet, "/portal/parent/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data ParentOverviewData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.Profile.GuardianID != guardianID {
		t.Fatalf("expected guardian id %s, got %s", guardianID, res.Data.Profile.GuardianID)
	}
	if len(res.Data.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(res.Data.Children))
	}
	child := res.Data.Children[0]
	if child.Enrollment.ClassName != "Grade 9A" {
		t.Fatalf("expected class Grade 9A, got %q", child.Enrollment.ClassName)
	}
	if child.AttendanceCounts.Present != 40 {
		t.Fatalf("expected 40 present marks, got %d", child.AttendanceCounts.Present)
	}
	if child.AverageScore == nil || *child.AverageScore != 82.5 {
		t.Fatalf("expected average score 82.5, got %v", child.AverageScore)
	}
	if len(res.Data.FeedAnnouncements) != 1 || res.Data.FeedAnnouncements[0].Title != "PTA Meeting" {
		t.Fatalf("unexpected announcements: %+v", res.Data.FeedAnnouncements)
	}
}

func TestGetParentOverview_GuardianNotFound(t *testing.T) {
	parent := makeTestParent()

	svc := &stubService{
		getParentOverviewFn: func(ctx context.Context, userID uuid.UUID) (ParentOverviewData, error) {
			return ParentOverviewData{}, ErrGuardianNotFound
		},
	}

	router := setupTestRouter(svc, parent)
	req := httptest.NewRequest(http.MethodGet, "/portal/parent/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}

func TestGetChildDetail_Success(t *testing.T) {
	parent := makeTestParent()
	studentID := uuid.New()
	classID := uuid.New()
	score := 18.5

	svc := &stubService{
		getChildDetailFn: func(ctx context.Context, userID uuid.UUID, sid uuid.UUID) (ChildDetail, error) {
			if sid != studentID {
				t.Fatalf("expected studentID %s, got %s", studentID, sid)
			}
			return ChildDetail{
				Student: ChildSummary{
					StudentID:     studentID.String(),
					StudentNumber: "STU-101",
					FirstName:     "John",
					LastName:      "Doe",
					Relationship:  "Mother",
					StudentStatus: "ACTIVE",
					Enrollment: ChildEnrollmentContext{
						ClassID:          classID.String(),
						ClassName:        "Grade 9A",
						AcademicYearName: "2026/2027",
					},
					AttendanceCounts: ChildAttendanceCounts{Present: 12, Absent: 1},
				},
				AttendanceCounts: ChildAttendanceCounts{Present: 12, Absent: 1},
				RecentAttendance: []ChildAttendanceRecord{
					{
						RecordID:    uuid.New().String(),
						SessionDate: "2026-10-06",
						Status:      "PRESENT",
						ClassID:     classID.String(),
						ClassName:   "Grade 9A",
						SubjectID:   uuid.New().String(),
						SubjectName: "Mathematics",
						SubjectCode: "MATH09",
						TermName:    "Term 1",
					},
				},
				Results: []ChildResult{
					{
						ResultID:        uuid.New().String(),
						AssessmentID:    uuid.New().String(),
						AssessmentTitle: "Algebra Quiz 1",
						AssessmentType:  "QUIZ",
						SubjectName:     "Mathematics",
						SubjectCode:     "MATH09",
						ClassName:       "Grade 9A",
						TermName:        "Term 1",
						AssessmentDate:  "2026-10-05",
						MaxScore:        20,
						Score:           &score,
						Grade:           nil,
					},
				},
			}, nil
		},
	}

	router := setupTestRouter(svc, parent)
	req := httptest.NewRequest(http.MethodGet, "/portal/parent/children/"+studentID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res struct {
		Data ChildDetail `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Data.Student.StudentNumber != "STU-101" {
		t.Fatalf("expected student number STU-101, got %s", res.Data.Student.StudentNumber)
	}
	if len(res.Data.RecentAttendance) != 1 || res.Data.RecentAttendance[0].Status != "PRESENT" {
		t.Fatalf("unexpected attendance records: %+v", res.Data.RecentAttendance)
	}
	if len(res.Data.Results) != 1 || res.Data.Results[0].Score == nil || *res.Data.Results[0].Score != 18.5 {
		t.Fatalf("unexpected results: %+v", res.Data.Results)
	}
}

func TestGetChildDetail_AccessDeniedAndValidation(t *testing.T) {
	parent := makeTestParent()
	linkedStudentID := uuid.New()
	unlinkedStudentID := uuid.New()

	svc := &stubService{
		getChildDetailFn: func(ctx context.Context, userID uuid.UUID, sid uuid.UUID) (ChildDetail, error) {
			if sid == unlinkedStudentID {
				return ChildDetail{}, ErrChildAccessDenied
			}
			return ChildDetail{Student: ChildSummary{StudentID: sid.String()}}, nil
		},
	}

	router := setupTestRouter(svc, parent)

	// Unlinked child -> 403
	req := httptest.NewRequest(http.MethodGet, "/portal/parent/children/"+unlinkedStudentID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}

	// Linked child -> 200
	req = httptest.NewRequest(http.MethodGet, "/portal/parent/children/"+linkedStudentID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// Bad UUID -> 400
	req = httptest.NewRequest(http.MethodGet, "/portal/parent/children/invalid-uuid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}
}

func TestGetChildDetail_GuardianNotFound(t *testing.T) {
	parent := makeTestParent()

	svc := &stubService{
		getChildDetailFn: func(ctx context.Context, userID uuid.UUID, sid uuid.UUID) (ChildDetail, error) {
			return ChildDetail{}, ErrGuardianNotFound
		},
	}

	router := setupTestRouter(svc, parent)
	req := httptest.NewRequest(http.MethodGet, "/portal/parent/children/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", w.Code)
	}
}
