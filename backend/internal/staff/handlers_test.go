package staff

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// stubService embeds ServiceAPI: any method a test didn't wire up panics on
// the nil embedded interface, exactly like the rbac package's stub Querier —
// an unexpected call is a test failure, not silent success.
type stubService struct {
	ServiceAPI

	getStaffFn         func(ctx context.Context, id uuid.UUID) (StaffDetail, error)
	listStaffFn        func(ctx context.Context, f ListFilter) ([]db.ListStaffRow, int64, error)
	createStaffFn      func(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error)
	setStatusFn        func(ctx context.Context, actorID, id uuid.UUID, status string) (StaffDetail, error)
	createAccountFn    func(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error)
	setAccountRolesFn  func(ctx context.Context, actorID, staffID uuid.UUID, roleNames []string) ([]string, error)
	createDepartmentFn func(ctx context.Context, actorID uuid.UUID, name string, description *string) (db.Department, error)
	createPositionFn   func(ctx context.Context, actorID uuid.UUID, title, category string, description *string) (db.StaffPosition, error)
}

func (s *stubService) GetStaff(ctx context.Context, id uuid.UUID) (StaffDetail, error) {
	if s.getStaffFn == nil {
		panic("unexpected GetStaff call")
	}
	return s.getStaffFn(ctx, id)
}

func (s *stubService) ListStaff(ctx context.Context, f ListFilter) ([]db.ListStaffRow, int64, error) {
	if s.listStaffFn == nil {
		panic("unexpected ListStaff call")
	}
	return s.listStaffFn(ctx, f)
}

func (s *stubService) CreateStaff(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error) {
	if s.createStaffFn == nil {
		panic("unexpected CreateStaff call")
	}
	return s.createStaffFn(ctx, actorID, in)
}

func (s *stubService) SetEmploymentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (StaffDetail, error) {
	if s.setStatusFn == nil {
		panic("unexpected SetEmploymentStatus call")
	}
	return s.setStatusFn(ctx, actorID, id, status)
}

func (s *stubService) CreateAccount(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error) {
	if s.createAccountFn == nil {
		panic("unexpected CreateAccount call")
	}
	return s.createAccountFn(ctx, actorID, staffID, in)
}

func (s *stubService) SetAccountRoles(ctx context.Context, actorID, staffID uuid.UUID, roleNames []string) ([]string, error) {
	if s.setAccountRolesFn == nil {
		panic("unexpected SetAccountRoles call")
	}
	return s.setAccountRolesFn(ctx, actorID, staffID, roleNames)
}

func (s *stubService) CreateDepartment(ctx context.Context, actorID uuid.UUID, name string, description *string) (db.Department, error) {
	if s.createDepartmentFn == nil {
		panic("unexpected CreateDepartment call")
	}
	return s.createDepartmentFn(ctx, actorID, name, description)
}

func (s *stubService) CreateStaffPosition(ctx context.Context, actorID uuid.UUID, title, category string, description *string) (db.StaffPosition, error) {
	if s.createPositionFn == nil {
		panic("unexpected CreateStaffPosition call")
	}
	return s.createPositionFn(ctx, actorID, title, category, description)
}

const testActorID = "11111111-1111-1111-1111-111111111111"

// newTestRouter mounts every staff route with an injected authenticated
// user (auth.WithUser, the test helper from M4) — handler tests here cover
// handler behavior; the permission gates themselves are middleware, already
// covered by the M5 rbac tests.
func newTestRouter(svc ServiceAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)

	actor := uuid.MustParse(testActorID)
	r.Use(func(c *gin.Context) {
		auth.WithUser(c, auth.AuthenticatedUser{User: db.User{ID: actor}})
		c.Next()
	})

	r.GET("/staff", h.ListStaff)
	r.POST("/staff", h.CreateStaff)
	r.GET("/staff/:id", h.GetStaff)
	r.PATCH("/staff/:id", h.UpdateStaff)
	r.PUT("/staff/:id/assignment", h.SetAssignment)
	r.PUT("/staff/:id/status", h.SetEmploymentStatus)
	r.POST("/staff/:id/account", h.CreateAccount)
	r.PUT("/staff/:id/account/roles", h.SetAccountRoles)
	r.GET("/departments", h.ListDepartments)
	r.POST("/departments", h.CreateDepartment)
	r.GET("/staff-positions", h.ListStaffPositions)
	r.POST("/staff-positions", h.CreateStaffPosition)
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

// wantError asserts the one structured error shape (section 34).
func wantError(t *testing.T, out map[string]any, status int, code string) {
	t.Helper()
	errObj, ok := out["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %v", out)
	}
	if errObj["code"] != code {
		t.Fatalf("error code = %v, want %q", errObj["code"], code)
	}
} // sampleDetail builds a StaffDetail the stubs can return.
func sampleDetail(id uuid.UUID) StaffDetail {
	return StaffDetail{
		Row: db.GetStaffByIDRow{
			ID:               id,
			PersonID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			StaffNumber:      "STF-001",
			EmploymentStatus: StatusActive,
			FirstName:        "Ada",
			LastName:         "Obi",
			CreatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
			UpdatedAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}
}

func TestListStaff_ParsesFiltersAndClampsPagination(t *testing.T) {
	depID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	var got ListFilter
	svc := &stubService{listStaffFn: func(ctx context.Context, f ListFilter) ([]db.ListStaffRow, int64, error) {
		got = f
		return []db.ListStaffRow{}, 0, nil
	}}
	r := newTestRouter(svc)

	w, out := doJSON(t, r, http.MethodGet,
		"/staff?page=0&page_size=9999&search=%20%20ada%20%20&employment_status=ACTIVE&position_category=TEACHING&department_id="+depID.String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%v)", w.Code, out)
	}
	// page=0 clamps to 1, page_size=9999 clamps to MaxPageSize, search is trimmed.
	if got.Page != 0 || got.PageSize != 9999 {
		t.Fatalf("handler should pass raw ints to the service, got page=%d page_size=%d", got.Page, got.PageSize)
	}
	if got.Search != "ada" {
		t.Fatalf("search = %q, want \"ada\" (trimmed)", got.Search)
	}
	if got.EmploymentStatus != "ACTIVE" || got.PositionCategory != "TEACHING" {
		t.Fatalf("filters not passed through: %+v", got)
	}
	if got.DepartmentID == nil || *got.DepartmentID != depID {
		t.Fatalf("department_id not parsed: %+v", got.DepartmentID)
	}
	// The response carries the *clamped* pagination for the client.
	pagination := out["pagination"].(map[string]any)
	if pagination["page"] != float64(1) || pagination["page_size"] != float64(MaxPageSize) {
		t.Fatalf("pagination not clamped: %v", pagination)
	}
	if pagination["total"] != float64(0) || pagination["total_pages"] != float64(0) {
		t.Fatalf("empty result should report total=0/total_pages=0: %v", pagination)
	}
}

func TestListStaff_RejectsBadQueryParams(t *testing.T) {
	cases := []struct {
		name, query string
	}{
		{"non-integer page", "/staff?page=abc"},
		{"non-integer page_size", "/staff?page_size=big"},
		{"bad department uuid", "/staff?department_id=not-a-uuid"},
		{"unknown employment status", "/staff?employment_status=BANANA"},
		{"unknown position category", "/staff?position_category=XYZ"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{listStaffFn: func(ctx context.Context, f ListFilter) ([]db.ListStaffRow, int64, error) {
				t.Fatal("service must not be called for invalid input")
				return nil, 0, nil
			}}
			r := newTestRouter(svc)
			w, out := doJSON(t, r, http.MethodGet, tc.query, "")
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", w.Code)
			}
			wantError(t, out, w.Code, "VALIDATION_ERROR")
		})
	}
}

func TestCreateStaff(t *testing.T) {
	body := `{"staff_number":"STF-002","first_name":"Grace","last_name":"Eze","hired_date":"2026-09-01"}`

	t.Run("missing required fields is 400", func(t *testing.T) {
		svc := &stubService{createStaffFn: func(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error) {
			t.Fatal("service must not be called for invalid body")
			return StaffDetail{}, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff", `{"first_name":"Grace"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("bad hired_date format is 400", func(t *testing.T) {
		svc := &stubService{}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff",
			`{"staff_number":"STF-002","first_name":"Grace","last_name":"Eze","hired_date":"01/09/2026"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("service errors map to their codes", func(t *testing.T) {
		svc := &stubService{createStaffFn: func(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error) {
			return StaffDetail{}, ErrStaffNumberTaken
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff", body)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		wantError(t, out, w.Code, "STAFF_NUMBER_TAKEN")
	})

	t.Run("success parses inputs and returns 201", func(t *testing.T) {
		var gotIn CreateStaffInput
		var gotActor uuid.UUID
		id := uuid.MustParse("66666666-6666-6666-6666-666666666666")
		svc := &stubService{createStaffFn: func(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error) {
			gotIn = in
			gotActor = actorID
			return sampleDetail(id), nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff", body)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (%v)", w.Code, out)
		}
		if gotIn.StaffNumber != "STF-002" || gotIn.FirstName != "Grace" {
			t.Fatalf("input not passed through: %+v", gotIn)
		}
		if gotIn.HiredDate == nil || gotIn.HiredDate.Format("2006-01-02") != "2026-09-01" {
			t.Fatalf("hired_date not parsed: %+v", gotIn.HiredDate)
		}
		if gotActor.String() != testActorID {
			t.Fatalf("actor id = %s, want %s (audit attribution)", gotActor, testActorID)
		}
	})
}

func TestSetEmploymentStatus(t *testing.T) {
	id := uuid.MustParse("77777777-7777-7777-7777-777777777777")

	t.Run("invalid status is 400", func(t *testing.T) {
		svc := &stubService{setStatusFn: func(ctx context.Context, actorID, gotID uuid.UUID, status string) (StaffDetail, error) {
			return StaffDetail{}, ErrInvalidStatus
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPut, "/staff/"+id.String()+"/status", `{"employment_status":"FIRED"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("not found is 404", func(t *testing.T) {
		svc := &stubService{setStatusFn: func(ctx context.Context, actorID, gotID uuid.UUID, status string) (StaffDetail, error) {
			return StaffDetail{}, ErrStaffNotFound
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPut, "/staff/"+id.String()+"/status", `{"employment_status":"INACTIVE"}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		wantError(t, out, w.Code, "NOT_FOUND")
	})

	t.Run("success returns updated staff", func(t *testing.T) {
		var gotStatus string
		svc := &stubService{setStatusFn: func(ctx context.Context, actorID, gotID uuid.UUID, status string) (StaffDetail, error) {
			gotStatus = status
			detail := sampleDetail(gotID)
			detail.Row.EmploymentStatus = status
			return detail, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPut, "/staff/"+id.String()+"/status", `{"employment_status":"ON_LEAVE"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if gotStatus != "ON_LEAVE" {
			t.Fatalf("status arg = %q, want ON_LEAVE", gotStatus)
		}
		staffObj := out["staff"].(map[string]any)
		if staffObj["employment_status"] != "ON_LEAVE" {
			t.Fatalf("payload not updated: %v", staffObj)
		}
	})
}

func TestCreateAccount(t *testing.T) {
	id := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	validBody := `{"email":"ada@schoolhub.test","password":"ChangeMe123!","roles":["TEACHER"]}`

	t.Run("weak password is 400", func(t *testing.T) {
		svc := &stubService{createAccountFn: func(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error) {
			t.Fatal("service must not be called for a weak password")
			return Account{}, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff/"+id.String()+"/account",
			`{"email":"ada@schoolhub.test","password":"short","roles":["TEACHER"]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("empty roles is 400", func(t *testing.T) {
		svc := &stubService{}
		r := newTestRouter(svc)
		w, _ := doJSON(t, r, http.MethodPost, "/staff/"+id.String()+"/account",
			`{"email":"ada@schoolhub.test","password":"ChangeMe123!","roles":[]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (an account needs at least one role)", w.Code)
		}
	})

	t.Run("conflicts map to 409", func(t *testing.T) {
		cases := []struct {
			name string
			err  error
			code string
		}{
			{"account exists", ErrAccountExists, "ACCOUNT_EXISTS"},
			{"email taken", ErrEmailTaken, "EMAIL_TAKEN"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				svc := &stubService{createAccountFn: func(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error) {
					return Account{}, tc.err
				}}
				r := newTestRouter(svc)
				w, out := doJSON(t, r, http.MethodPost, "/staff/"+id.String()+"/account", validBody)
				if w.Code != http.StatusConflict {
					t.Fatalf("status = %d, want 409", w.Code)
				}
				wantError(t, out, w.Code, tc.code)
			})
		}
	})

	t.Run("unknown role is 400", func(t *testing.T) {
		svc := &stubService{createAccountFn: func(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error) {
			return Account{}, ErrUnknownRole
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff/"+id.String()+"/account", validBody)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("success returns account with roles", func(t *testing.T) {
		svc := &stubService{createAccountFn: func(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error) {
			if in.Email != "ada@schoolhub.test" || in.Password != "ChangeMe123!" {
				t.Fatalf("input not passed through: %+v", in)
			}
			return Account{ID: id, Email: in.Email, Status: "ACTIVE", Roles: in.RoleNames}, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff/"+id.String()+"/account", validBody)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		account := out["account"].(map[string]any)
		if account["email"] != "ada@schoolhub.test" {
			t.Fatalf("unexpected account payload: %v", account)
		}
	})
}

func TestSetAccountRoles(t *testing.T) {
	id := uuid.MustParse("99999999-9999-9999-9999-999999999999")

	t.Run("no account yet is 404", func(t *testing.T) {
		svc := &stubService{setAccountRolesFn: func(ctx context.Context, actorID, staffID uuid.UUID, roleNames []string) ([]string, error) {
			return nil, ErrUserNotFound
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPut, "/staff/"+id.String()+"/account/roles", `{"roles":["TEACHER"]}`)
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		wantError(t, out, w.Code, "NOT_FOUND")
	})

	t.Run("replace roles and return them", func(t *testing.T) {
		var gotRoles []string
		svc := &stubService{setAccountRolesFn: func(ctx context.Context, actorID, staffID uuid.UUID, roleNames []string) ([]string, error) {
			gotRoles = roleNames
			return []string{"ADMIN", "TEACHER"}, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPut, "/staff/"+id.String()+"/account/roles",
			`{"roles":["ADMIN","TEACHER"]}`)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if len(gotRoles) != 2 {
			t.Fatalf("roles not passed through: %v", gotRoles)
		}
		roles := out["roles"].([]any)
		if len(roles) != 2 || roles[0] != "ADMIN" {
			t.Fatalf("unexpected roles payload: %v", roles)
		}
	})
}

func TestCreateReferenceData(t *testing.T) {
	t.Run("department name taken is 409", func(t *testing.T) {
		svc := &stubService{createDepartmentFn: func(ctx context.Context, actorID uuid.UUID, name string, description *string) (db.Department, error) {
			return db.Department{}, ErrReferenceNameTaken
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/departments", `{"name":"Science"}`)
		if w.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", w.Code)
		}
		wantError(t, out, w.Code, "NAME_TAKEN")
	})

	t.Run("department created", func(t *testing.T) {
		svc := &stubService{createDepartmentFn: func(ctx context.Context, actorID uuid.UUID, name string, description *string) (db.Department, error) {
			return db.Department{ID: uuid.New(), Name: name}, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/departments", `{"name":"Science"}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		dept := out["department"].(map[string]any)
		if dept["name"] != "Science" {
			t.Fatalf("unexpected department payload: %v", dept)
		}
	})

	t.Run("position with unknown category is 400", func(t *testing.T) {
		svc := &stubService{createPositionFn: func(ctx context.Context, actorID uuid.UUID, title, category string, description *string) (db.StaffPosition, error) {
			return db.StaffPosition{}, ErrInvalidCategory
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff-positions", `{"title":"Bursar","category":"FINANCE"}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("position created", func(t *testing.T) {
		svc := &stubService{createPositionFn: func(ctx context.Context, actorID uuid.UUID, title, category string, description *string) (db.StaffPosition, error) {
			return db.StaffPosition{ID: uuid.New(), Title: title, Category: category}, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodPost, "/staff-positions", `{"title":"Bursar","category":"ACCOUNTS"}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		position := out["staff_position"].(map[string]any)
		if position["category"] != "ACCOUNTS" {
			t.Fatalf("unexpected position payload: %v", position)
		}
	})
}

func TestGetStaff(t *testing.T) {
	id := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	t.Run("bad uuid is 400", func(t *testing.T) {
		svc := &stubService{}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodGet, "/staff/nope", "")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
		wantError(t, out, w.Code, "VALIDATION_ERROR")
	})

	t.Run("not found is 404", func(t *testing.T) {
		svc := &stubService{getStaffFn: func(ctx context.Context, id uuid.UUID) (StaffDetail, error) {
			return StaffDetail{}, ErrStaffNotFound
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodGet, "/staff/"+id.String(), "")
		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", w.Code)
		}
		wantError(t, out, w.Code, "NOT_FOUND")
	})

	t.Run("success returns staff with account", func(t *testing.T) {
		detail := sampleDetail(id)
		detail.Account = &Account{
			ID:     uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			Email:  "ada@schoolhub.test",
			Status: "ACTIVE",
			Roles:  []string{"TEACHER"},
		}
		svc := &stubService{getStaffFn: func(ctx context.Context, got uuid.UUID) (StaffDetail, error) {
			if got != id {
				t.Fatalf("GetStaff got id %s, want %s", got, id)
			}
			return detail, nil
		}}
		r := newTestRouter(svc)
		w, out := doJSON(t, r, http.MethodGet, "/staff/"+id.String(), "")
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		staffObj := out["staff"].(map[string]any)
		if staffObj["staff_number"] != "STF-001" || staffObj["employment_status"] != "ACTIVE" {
			t.Fatalf("unexpected staff payload: %v", staffObj)
		}
		account := staffObj["account"].(map[string]any)
		if account["email"] != "ada@schoolhub.test" {
			t.Fatalf("unexpected account payload: %v", account)
		}
	})
}
