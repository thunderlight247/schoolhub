package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// stubQuerier lets tests stub exactly the Querier methods the code under
// test should call. The embedded db.Querier interface value is nil, so any
// method that isn't explicitly overridden panics when called — an
// unexpected query fails the test loudly instead of returning zero values
// and passing silently. The service tests in service_test.go reuse it.
type stubQuerier struct {
	db.Querier
	userHasPermission func(context.Context, db.UserHasPermissionParams) (bool, error)
	listRolesForUser  func(context.Context, uuid.UUID) ([]db.Role, error)
}

func (s *stubQuerier) UserHasPermission(ctx context.Context, p db.UserHasPermissionParams) (bool, error) {
	return s.userHasPermission(ctx, p)
}

func (s *stubQuerier) ListRolesForUser(ctx context.Context, userID uuid.UUID) ([]db.Role, error) {
	return s.listRolesForUser(ctx, userID)
}

// newTestContext builds a Gin test context with (or without) an
// authenticated user installed exactly the way auth.RequireAuth would have
// installed one, so the middleware under test runs against realistic input.
func newTestContext(t *testing.T, authUser *auth.AuthenticatedUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if authUser != nil {
		auth.WithUser(c, *authUser)
	}
	return c, w
}

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name          string
		authUser      *auth.AuthenticatedUser
		hasPermission bool
		queryErr      error
		wantStatus    int
		wantAbort     bool
	}{
		{
			name:       "rejects unauthenticated request with 401 without touching the database",
			authUser:   nil,
			wantStatus: http.StatusUnauthorized,
			wantAbort:  true,
		},
		{
			name:          "allows a user who holds the permission",
			authUser:      &auth.AuthenticatedUser{},
			hasPermission: true,
			wantStatus:    http.StatusOK,
			wantAbort:     false,
		},
		{
			name:       "rejects a user without the permission with 403",
			authUser:   &auth.AuthenticatedUser{},
			wantStatus: http.StatusForbidden,
			wantAbort:  true,
		},
		{
			name:       "fails closed on query error with 500",
			authUser:   &auth.AuthenticatedUser{},
			queryErr:   errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantAbort:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queryCalls int
			var gotCode string
			queries := &stubQuerier{
				userHasPermission: func(_ context.Context, p db.UserHasPermissionParams) (bool, error) {
					queryCalls++
					gotCode = p.PermissionCode
					return tt.hasPermission, tt.queryErr
				},
			}
			svc := NewService(queries)

			c, w := newTestContext(t, tt.authUser)
			RequirePermission(svc, "student.view")(c)

			if tt.wantAbort && !c.IsAborted() {
				t.Fatal("expected request to be aborted, but it passed through")
			}
			if !tt.wantAbort && c.IsAborted() {
				t.Fatalf("expected request to pass through, but it was aborted with status %d", w.Code)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantAbort && tt.authUser != nil && queryCalls != 1 {
				t.Errorf("query called %d times for an authenticated request, want 1", queryCalls)
			}
			if !tt.wantAbort && gotCode != "student.view" {
				t.Errorf("permission code checked = %q, want %q", gotCode, "student.view")
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		authUser   *auth.AuthenticatedUser
		roles      []db.Role
		queryErr   error
		wantStatus int
		wantAbort  bool
	}{
		{
			name:       "rejects unauthenticated request with 401",
			authUser:   nil,
			wantStatus: http.StatusUnauthorized,
			wantAbort:  true,
		},
		{
			name:       "allows a user who holds the required role",
			authUser:   &auth.AuthenticatedUser{},
			roles:      []db.Role{{Name: "ADMIN"}},
			wantStatus: http.StatusOK,
			wantAbort:  false,
		},
		{
			name:       "rejects a user with only a different role with 403",
			authUser:   &auth.AuthenticatedUser{},
			roles:      []db.Role{{Name: "TEACHER"}},
			wantStatus: http.StatusForbidden,
			wantAbort:  true,
		},
		{
			name:       "rejects a user with no roles with 403",
			authUser:   &auth.AuthenticatedUser{},
			roles:      []db.Role{},
			wantStatus: http.StatusForbidden,
			wantAbort:  true,
		},
		{
			name:       "fails closed on query error with 500",
			authUser:   &auth.AuthenticatedUser{},
			queryErr:   errors.New("db down"),
			wantStatus: http.StatusInternalServerError,
			wantAbort:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queryCalls int
			queries := &stubQuerier{
				listRolesForUser: func(context.Context, uuid.UUID) ([]db.Role, error) {
					queryCalls++
					return tt.roles, tt.queryErr
				},
			}
			svc := NewService(queries)

			c, w := newTestContext(t, tt.authUser)
			RequireRole(svc, "ADMIN")(c)

			if tt.wantAbort && !c.IsAborted() {
				t.Fatal("expected request to be aborted, but it passed through")
			}
			if !tt.wantAbort && c.IsAborted() {
				t.Fatalf("expected request to pass through, but it was aborted with status %d", w.Code)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantAbort && tt.authUser != nil && queryCalls != 1 {
				t.Errorf("query called %d times for an authenticated request, want 1", queryCalls)
			}
		})
	}
}
