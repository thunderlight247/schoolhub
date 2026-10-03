package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

// stubQuerier is defined in middleware_test.go.

func TestServiceHasPermission(t *testing.T) {
	userID := uuid.New()

	t.Run("reports true and passes user id and code through to the query", func(t *testing.T) {
		var gotParams db.UserHasPermissionParams
		queries := &stubQuerier{
			userHasPermission: func(_ context.Context, p db.UserHasPermissionParams) (bool, error) {
				gotParams = p
				return true, nil
			},
		}
		svc := NewService(queries)

		got, err := svc.HasPermission(context.Background(), userID, "student.view")
		if err != nil {
			t.Fatalf("HasPermission returned error: %v", err)
		}
		if !got {
			t.Error("HasPermission = false, want true")
		}
		if gotParams.UserID != userID || gotParams.PermissionCode != "student.view" {
			t.Errorf("query params = %+v, want userID=%s, code=%q", gotParams, userID, "student.view")
		}
	})

	t.Run("reports false for a permission the user does not hold", func(t *testing.T) {
		queries := &stubQuerier{
			userHasPermission: func(context.Context, db.UserHasPermissionParams) (bool, error) {
				return false, nil
			},
		}
		svc := NewService(queries)

		got, err := svc.HasPermission(context.Background(), userID, "student.archive")
		if err != nil {
			t.Fatalf("HasPermission returned error: %v", err)
		}
		if got {
			t.Error("HasPermission = true, want false")
		}
	})

	t.Run("propagates query errors", func(t *testing.T) {
		wantErr := errors.New("db down")
		queries := &stubQuerier{
			userHasPermission: func(context.Context, db.UserHasPermissionParams) (bool, error) {
				return false, wantErr
			},
		}
		svc := NewService(queries)

		_, err := svc.HasPermission(context.Background(), userID, "student.view")
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}

func TestServiceHasRole(t *testing.T) {
	userID := uuid.New()

	t.Run("true when one of the user's roles matches", func(t *testing.T) {
		queries := &stubQuerier{
			listRolesForUser: func(context.Context, uuid.UUID) ([]db.Role, error) {
				return []db.Role{{Name: "STAFF"}, {Name: "ADMIN"}}, nil
			},
		}
		svc := NewService(queries)

		got, err := svc.HasRole(context.Background(), userID, "ADMIN")
		if err != nil {
			t.Fatalf("HasRole returned error: %v", err)
		}
		if !got {
			t.Error("HasRole(ADMIN) = false, want true")
		}
	})

	t.Run("false when none of the user's roles match", func(t *testing.T) {
		queries := &stubQuerier{
			listRolesForUser: func(context.Context, uuid.UUID) ([]db.Role, error) {
				return []db.Role{{Name: "PARENT"}}, nil
			},
		}
		svc := NewService(queries)

		got, err := svc.HasRole(context.Background(), userID, "ADMIN")
		if err != nil {
			t.Fatalf("HasRole returned error: %v", err)
		}
		if got {
			t.Error("HasRole(ADMIN) = true, want false")
		}
	})

	t.Run("false for a user with no roles", func(t *testing.T) {
		queries := &stubQuerier{
			listRolesForUser: func(context.Context, uuid.UUID) ([]db.Role, error) {
				return []db.Role{}, nil
			},
		}
		svc := NewService(queries)

		got, err := svc.HasRole(context.Background(), userID, "ADMIN")
		if err != nil {
			t.Fatalf("HasRole returned error: %v", err)
		}
		if got {
			t.Error("HasRole(ADMIN) = true, want false")
		}
	})

	t.Run("propagates query errors", func(t *testing.T) {
		wantErr := errors.New("db down")
		queries := &stubQuerier{
			listRolesForUser: func(context.Context, uuid.UUID) ([]db.Role, error) {
				return nil, wantErr
			},
		}
		svc := NewService(queries)

		_, err := svc.HasRole(context.Background(), userID, "ADMIN")
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
	})
}
