package adminaccounts

// Handler tests for the access-control endpoints, mirroring internal/
// staff's handlers_test.go: stub ServiceAPI, inject an authenticated user
// via auth.WithUser, and assert request shapes/error mapping. Permission
// and role gating (admin.manage, MASTER_ADMIN, account.block) are RBAC
// middleware concerns already covered by the M5 rbac tests — these tests
// assume the handler is reached with an authenticated actor.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// stubService embeds ServiceAPI: any method a test didn't wire up panics
// on the nil embedded interface — an unexpected call is a test failure,
// not silent success.
type stubService struct {
	ServiceAPI

	listAdminsFn  func(ctx context.Context) ([]AdminView, error)
	listByRoleFn  func(ctx context.Context, roleName string) ([]AccountView, error)
	revokeAdminFn func(ctx context.Context, actorID, targetUserID uuid.UUID) error
	blockFn       func(ctx context.Context, actorID, targetUserID uuid.UUID) error
	unblockFn     func(ctx context.Context, actorID, targetUserID uuid.UUID) error
}

func (s *stubService) ListAdmins(ctx context.Context) ([]AdminView, error) {
	if s.listAdminsFn == nil {
		panic("unexpected ListAdmins call")
	}
	return s.listAdminsFn(ctx)
}

func (s *stubService) ListAccountsByRole(ctx context.Context, roleName string) ([]AccountView, error) {
	if s.listByRoleFn == nil {
		panic("unexpected ListAccountsByRole call")
	}
	return s.listByRoleFn(ctx, roleName)
}

func (s *stubService) RevokeAdmin(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	if s.revokeAdminFn == nil {
		panic("unexpected RevokeAdmin call")
	}
	return s.revokeAdminFn(ctx, actorID, targetUserID)
}

func (s *stubService) Block(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	if s.blockFn == nil {
		panic("unexpected Block call")
	}
	return s.blockFn(ctx, actorID, targetUserID)
}

func (s *stubService) Unblock(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	if s.unblockFn == nil {
		panic("unexpected Unblock call")
	}
	return s.unblockFn(ctx, actorID, targetUserID)
}

const testActorID = "11111111-1111-1111-1111-111111111111"

func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)

	actor := uuid.MustParse(testActorID)
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{User: db.User{ID: actor}})
		c.Next()
	})

	r.GET("/admins", h.ListAdmins)
	r.GET("/accounts", h.ListAccountsByRole)
	r.PUT("/users/:id/admin-role", h.RevokeAdmin)
	r.PUT("/users/:id/status", h.Block)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var out map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("response is not JSON: %v (%s)", err, w.Body.String())
		}
	}
	return w, out
}

func wantError(t *testing.T, out map[string]any, status int, code string) {
	t.Helper()
	errObj, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %v", out)
	}
	if errObj["code"] != code {
		t.Fatalf("error code = %v, want %q", errObj["code"], code)
	}
}

func TestListAdmins_ReturnsViews(t *testing.T) {
	masterID := uuid.MustParse(testActorID)
	other := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	svc := &stubService{listAdminsFn: func(ctx context.Context) ([]AdminView, error) {
		return []AdminView{
			{UserID: masterID, Email: "master@schoolhub.test", FirstName: "Ada", LastName: "Obi", Status: "ACTIVE", IsMaster: true},
			{UserID: other, Email: "deputy@schoolhub.test", FirstName: "Ben", LastName: "Ito", Status: "SUSPENDED", IsMaster: false},
		}, nil
	}}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodGet, "/admins", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%v)", w.Code, out)
	}
	admins := out["admins"].([]any)
	if len(admins) != 2 {
		t.Fatalf("len(admins) = %d, want 2", len(admins))
	}
	first := admins[0].(map[string]any)
	if first["is_master"] != true || first["status"] != "ACTIVE" {
		t.Fatalf("first admin row wrong: %v", first)
	}
}

func TestListAccountsByRole_RejectsAdminRoles(t *testing.T) {
	svc := &stubService{
		listByRoleFn: func(ctx context.Context, roleName string) ([]AccountView, error) {
			t.Errorf("unexpected call for role %q", roleName)
			return nil, nil
		},
	}
	r := newTestRouter(svc)

	for _, role := range []string{"ADMIN", "MASTER_ADMIN", "bogus", ""} {
		w, out := doJSON(t, r, http.MethodGet, "/accounts?role="+role, "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("role=%q status = %d, want 400 (%v)", role, w.Code, out)
		}
		wantError(t, out, http.StatusBadRequest, "VALIDATION_ERROR")
	}
}

func TestListAccountsByRole_PassesRoleThrough(t *testing.T) {
	var gotRole string
	svc := &stubService{listByRoleFn: func(ctx context.Context, roleName string) ([]AccountView, error) {
		gotRole = roleName
		return []AccountView{}, nil
	}}
	r := newTestRouter(svc)

	w, _ := doJSON(t, r, http.MethodGet, "/accounts?role=PARENT", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if gotRole != "PARENT" {
		t.Fatalf("role = %q, want PARENT", gotRole)
	}
}

func TestBlock_SuspendsAndUnblocks(t *testing.T) {
	actor := uuid.MustParse(testActorID)
	target := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	var gotStatus string
	svc := &stubService{
		blockFn: func(ctx context.Context, a, u uuid.UUID) error {
			if a != actor || u != target {
				t.Fatalf("actor/target = %v/%v, want %v/%v", a, u, actor, target)
			}
			gotStatus = "SUSPENDED"
			return nil
		},
		unblockFn: func(ctx context.Context, a, u uuid.UUID) error {
			gotStatus = "ACTIVE"
			return nil
		},
	}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodPut, "/users/"+target.String()+"/status", `{"status":"SUSPENDED"}`)
	if w.Code != http.StatusNoContent || gotStatus != "SUSPENDED" {
		t.Fatalf("block: status = %d (%v), gotStatus %q", w.Code, out, gotStatus)
	}

	w, out = doJSON(t, r, http.MethodPut, "/users/"+target.String()+"/status", `{"status":"ACTIVE"}`)
	if w.Code != http.StatusNoContent || gotStatus != "ACTIVE" {
		t.Fatalf("unblock: status = %d (%v), gotStatus %q", w.Code, out, gotStatus)
	}
}

func TestBlock_MapsServiceErrors(t *testing.T) {
	target := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	svc := &stubService{
		blockFn: func(ctx context.Context, a, u uuid.UUID) error {
			return ErrSelfBlock
		},
	}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodPut, "/users/"+target.String()+"/status", `{"status":"SUSPENDED"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%v)", w.Code, out)
	}
	wantError(t, out, http.StatusConflict, "SELF_BLOCK")
}

func TestBlock_RejectsBadStatus(t *testing.T) {
	target := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	svc := &stubService{
		blockFn: func(ctx context.Context, a, u uuid.UUID) error {
			t.Error("unexpected Block call")
			return nil
		},
	}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodPut, "/users/"+target.String()+"/status", `{"status":"DELETED"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (%v)", w.Code, out)
	}
	wantError(t, out, http.StatusBadRequest, "VALIDATION_ERROR")
}

func TestRevokeAdmin_PassesActorAndTarget(t *testing.T) {
	actor := uuid.MustParse(testActorID)
	target := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	var gotActor, gotTarget uuid.UUID
	svc := &stubService{revokeAdminFn: func(ctx context.Context, a, u uuid.UUID) error {
		gotActor, gotTarget = a, u
		return nil
	}}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodPut, "/users/"+target.String()+"/admin-role", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%v)", w.Code, out)
	}
	if gotActor != actor || gotTarget != target {
		t.Fatalf("actor/target = %v/%v, want %v/%v", gotActor, gotTarget, actor, target)
	}
}

func TestRevokeAdmin_MapsLastAdmin(t *testing.T) {
	target := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	svc := &stubService{revokeAdminFn: func(ctx context.Context, a, u uuid.UUID) error {
		return ErrLastAdmin
	}}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodPut, "/users/"+target.String()+"/admin-role", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%v)", w.Code, out)
	}
	wantError(t, out, http.StatusConflict, "LAST_ADMIN")
}

func TestRevokeAdmin_MapsSelfDemotion(t *testing.T) {
	svc := &stubService{revokeAdminFn: func(ctx context.Context, a, u uuid.UUID) error {
		return ErrSelfDemotion
	}}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodPut, "/users/"+testActorID+"/admin-role", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%v)", w.Code, out)
	}
	wantError(t, out, http.StatusConflict, "SELF_DEMOTION")
}
