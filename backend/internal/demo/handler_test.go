package demo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	pgx "github.com/jackc/pgx/v5"
	"github.com/google/uuid"

	"github.com/thunderlight247/schoolhub/backend/internal/auth"
	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

// fakeLoginService records the role it was asked to resolve and returns a
// canned result, standing in for the database-backed *Service.
type fakeLoginService struct {
	gotRole Role
	token   string
	user    auth.AuthenticatedUser
	err     error
}

func (f *fakeLoginService) Login(ctx context.Context, role Role, userAgent, ipAddress string) (string, auth.AuthenticatedUser, error) {
	f.gotRole = role
	if f.err != nil {
		return "", auth.AuthenticatedUser{}, f.err
	}
	return f.token, f.user, nil
}

func fakeAuthUser() auth.AuthenticatedUser {
	return auth.AuthenticatedUser{
		User: db.User{
			ID:     uuid.MustParse("00000000-0000-0000-0000-00000000000a"),
			Email:  "ada.obi@demo.schoolhub",
			Status: "ACTIVE",
		},
		Person:      db.Person{FirstName: "Ada", LastName: "Obi"},
		Roles:       []string{"TEACHER"},
		Permissions: []string{"class.view"},
	}
}

func postDemoLogin(t *testing.T, svc *fakeLoginService, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/demo-login", NewHandler(svc, auth.CookieConfig{Name: "schoolhub_session"}).Login)

	req := httptest.NewRequest(http.MethodPost, "/demo-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestLogin_OK(t *testing.T) {
	svc := &fakeLoginService{token: "tok", user: fakeAuthUser()}
	rec := postDemoLogin(t, svc, `{"as":"teacher"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if svc.gotRole != RoleTeacher {
		t.Errorf("service received role %q, want teacher", svc.gotRole)
	}
	if !strings.Contains(rec.Body.String(), `"first_name":"Ada"`) {
		t.Errorf("body missing user payload: %s", rec.Body.String())
	}
	// The cookie must carry the raw session token exactly like the password
	// login sets it — the frontend's session handling relies on it.
	cookie := rec.Header().Get("Set-Cookie")
	if !strings.HasPrefix(cookie, "schoolhub_session=tok") {
		t.Errorf("session cookie not set correctly: %q", cookie)
	}
}

func TestLogin_UnknownRole(t *testing.T) {
	svc := &fakeLoginService{}
	rec := postDemoLogin(t, svc, `{"as":"student"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if svc.gotRole != "" {
		t.Errorf("service should not have been called for an unknown role, got %q", svc.gotRole)
	}
}

func TestLogin_MissingRole(t *testing.T) {
	svc := &fakeLoginService{}
	rec := postDemoLogin(t, svc, `{}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLogin_DemoDataMissing(t *testing.T) {
	svc := &fakeLoginService{err: pgx.ErrNoRows}
	rec := postDemoLogin(t, svc, `{"as":"parent"}`)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "DEMO_DATA_MISSING") {
		t.Errorf("body missing DEMO_DATA_MISSING code: %s", rec.Body.String())
	}
}

func TestLogin_UnexpectedServiceError(t *testing.T) {
	svc := &fakeLoginService{err: context.DeadlineExceeded}
	rec := postDemoLogin(t, svc, `{"as":"admin"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestParseRole(t *testing.T) {
	for _, r := range []string{"admin", "teacher", "parent"} {
		if _, ok := parseRole(r); !ok {
			t.Errorf("parseRole(%q) should be accepted", r)
		}
	}
	for _, r := range []string{"", "student", "master_admin", "ADMIN"} {
		if _, ok := parseRole(r); ok {
			t.Errorf("parseRole(%q) should be rejected", r)
		}
	}
}

// Mode handler: report the configured flag.
func TestMode(t *testing.T) {
	for _, want := range []bool{true, false} {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/demo-mode", Mode(want))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/demo-mode", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		body := rec.Body.String()
		if want != strings.Contains(body, `"demo_mode":true`) {
			t.Errorf("demo-mode body %s, want demo_mode true=%v", body, want)
		}
	}
}
