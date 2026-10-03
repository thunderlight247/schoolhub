// Package rbac implements authorization: permission and role checks, plus
// the admin endpoints for managing which permissions each role holds.
// Authentication (who is this user) is a separate concern — see
// internal/auth. This package depends on auth only for auth.CurrentUser,
// to read who the already-authenticated caller is.
package rbac

import (
	"context"

	"github.com/google/uuid"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

// Service answers authorization questions ("can this user do X", "does
// this user hold role Y"). It depends on db.Querier, not the concrete
// *db.Queries type, for the same testability reason as auth.Service.
type Service struct {
	queries db.Querier
}

func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

// HasPermission reports whether userID holds the given permission code,
// through any of their assigned roles. This is the check every
// RequirePermission-gated route ultimately relies on.
func (s *Service) HasPermission(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	return s.queries.UserHasPermission(ctx, db.UserHasPermissionParams{
		UserID:         userID,
		PermissionCode: code,
	})
}

// HasRole reports whether userID has been assigned the given role by name
// (e.g. "ADMIN"). Used sparingly — only for operations gated on "are you
// an admin" rather than a specific permission code, such as managing RBAC
// itself (see handlers.go): none of section 11's permission codes
// describe "manage permissions", since the spec didn't anticipate needing
// to secure its own permission-management endpoints.
func (s *Service) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	roles, err := s.queries.ListRolesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r.Name == roleName {
			return true, nil
		}
	}
	return false, nil
}
