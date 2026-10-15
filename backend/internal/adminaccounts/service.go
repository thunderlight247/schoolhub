// Package adminaccounts implements the account-access control layer added
// in the expert pass: a MASTER_ADMIN role above ADMIN that is the only one
// allowed to grant or revoke the ADMIN role itself, plus the ability for
// any admin to suspend ("block") any parent/teacher/staff account —
// including one that is signed in right now (auth.ValidateSession
// re-checks users.status on every request, so flipping status to SUSPENDED
// kills the live session's next call).
//
// Layering matches the other domain packages: HTTP handlers validate and
// map errors; this Service owns business rules and transactions; the
// rbac.RequirePermission gates in cmd/server/main.go decide who may even
// reach these endpoints (admin.manage for admin management, account.block
// for suspending/restoring regular accounts).
//
// Guards the service enforces (defense in depth — the permission gate
// alone doesn't express these):
//   - Only MASTER_ADMIN users may revoke another user's ADMIN role. A
//     plain ADMIN holding admin.manage (possible via RBAC customization)
//     is still refused. Self-service demotion is refused too: a master
//     admin removing their own admin role would strand the school.
//   - No revoke that would leave zero active admins — otherwise a single
//     mistake locks the school out of its own management screens.
//   - Blocking never applies to admin accounts; admins are managed
//     through role revocation instead, one mechanism per audience.
//   - Every mutation writes an audit row in the same transaction.
package adminaccounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// Role names fixed by migration 000002/000014's CHECK on roles.name.
const (
	RoleAdmin       = "ADMIN"
	RoleMasterAdmin = "MASTER_ADMIN"
)

// Account statuses (migration 000002's CHECK on users.status).
const (
	StatusActive    = "ACTIVE"
	StatusSuspended = "SUSPENDED"
)

// Sentinel errors — handlers map these to specific HTTP responses;
// anything else is an unexpected failure (500) whose details are never
// exposed.
var (
	ErrUserNotFound      = errors.New("adminaccounts: user not found")
	ErrNotAdmin          = errors.New("adminaccounts: target user holds no admin role")
	ErrLastAdmin         = errors.New("adminaccounts: refusing to remove the last active admin")
	ErrEmailTaken        = errors.New("adminaccounts: email already has an account")
	ErrSelfDemotion      = errors.New("adminaccounts: master admins cannot revoke their own admin role")
	ErrSelfBlock         = errors.New("adminaccounts: cannot block your own account")
	ErrAdminNotBlockable = errors.New("adminaccounts: admin accounts can only be blocked by a MASTER_ADMIN or the account's inviter")
	ErrLastMaster        = errors.New("adminaccounts: cannot block the last ACTIVE master admin")
	ErrNotYourInvitee    = errors.New("adminaccounts: not an account you provisioned")
	ErrUnknownRole       = errors.New("adminaccounts: unknown role name")
)

// Service owns the access-control mutations. It holds the pool (like the
// other domain services) because every mutation runs inside a transaction
// that also writes its audit entry.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	// inviter (optional, M34) is the best-effort Clerk invitation issuer
	// for newly added admins; the recipient activates via the emailed link,
	// so the backend never stores a human-chosen admin password.
	inviter     *auth.ClerkInviter
	frontendURL string
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

// SetInviter attaches the optional Clerk invitation issuer (M34).
func (s *Service) SetInviter(inviter *auth.ClerkInviter, frontendURL string) {
	s.inviter = inviter
	s.frontendURL = frontendURL
}

// AdminView is one row of the master-admin's "manage admins" list.
type AdminView struct {
	UserID      uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	Status      string
	IsMaster    bool
	LastLoginAt *time.Time
}

// AccountView is one row of the per-role account directory (the rows the
// block/unblock toggle applies to).
type AccountView struct {
	UserID      uuid.UUID
	PersonID    uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	Status      string
	LastLoginAt *time.Time
}

// IsMasterAdmin reports whether userID holds the MASTER_ADMIN role. Both
// the service guards below and the frontend's "Manage admins" UI section
// key off this.
func (s *Service) IsMasterAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	roles, err := s.queries.ListRolesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r.Name == RoleMasterAdmin {
			return true, nil
		}
	}
	return false, nil
}

// ListAdmins returns every ADMIN/MASTER_ADMIN holder for the master-admin
// management screen. Raises ErrNotMasterAdmin in the handler when the
// caller isn't one — enforced at the route layer (main.go).
func (s *Service) ListAdmins(ctx context.Context) ([]AdminView, error) {
	rows, err := s.queries.ListAdmins(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]AdminView, 0, len(rows))
	for _, r := range rows {
		views = append(views, AdminView{
			UserID:      r.UserID,
			Email:       r.Email,
			FirstName:   r.FirstName,
			LastName:    r.LastName,
			Status:      r.Status,
			IsMaster:    r.IsMaster == RoleMasterAdmin,
			LastLoginAt: tsOrNil(r.LastLoginAt),
		})
	}
	return views, nil
}

// Invitee is one row of GET /my-invitees — an account the caller
// personally provisioned (inviter tracking, migration 000016).
type Invitee struct {
	UserID    uuid.UUID
	Email     string
	FirstName string
	LastName  string
	Roles     string // comma-joined role names
	Status    string
}

// ListInvitees returns the accounts the caller minted themselves — the
// "every inviter can see invitees" clause. Empty slice, not nil, when the
// caller never provisioned anything.
func (s *Service) ListInvitees(ctx context.Context, inviterID uuid.UUID) ([]Invitee, error) {
	rows, err := s.queries.ListUsersInvitedBy(ctx, pgtype.UUID{Bytes: inviterID, Valid: true})
	if err != nil {
		return nil, err
	}
	invitees := make([]Invitee, 0, len(rows))
	for _, r := range rows {
		roleStr, _ := r.Roles.(string)
		invitees = append(invitees, Invitee{
			UserID:    r.ID,
			Email:     r.Email,
			FirstName: r.FirstName,
			LastName:  r.LastName,
			Roles:     roleStr,
			Status:    r.Status,
		})
	}
	return invitees, nil
}

// BlockInvitee/UnblockInvitee: the "every inviter can block their invitees"
// clause, for callers WITHOUT the account.block permission (teachers). The
// scoping is strict — the target must have been minted by THE CALLER; anyone
// else keeps using the account.block-gated /users/:id/status route.
func (s *Service) BlockInvitee(ctx context.Context, inviterID, targetUserID uuid.UUID) error {
	return s.setInviteeStatus(ctx, inviterID, targetUserID, StatusSuspended)
}

func (s *Service) UnblockInvitee(ctx context.Context, inviterID, targetUserID uuid.UUID) error {
	return s.setInviteeStatus(ctx, inviterID, targetUserID, StatusActive)
}

func (s *Service) setInviteeStatus(ctx context.Context, inviterID, targetUserID uuid.UUID, newStatus string) error {
	// Own-invitee scoping BEFORE anything else: this path is only for
	// callers acting on an account they themselves provisioned.
	target, err := s.queries.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if !target.InvitedBy.Valid || uuid.UUID(target.InvitedBy.Bytes) != inviterID {
		return ErrNotYourInvitee
	}
	if inviterID == targetUserID {
		return ErrSelfBlock
	}
	return s.setAccountStatus(ctx, inviterID, targetUserID, newStatus)
}

// ListAccountsByRole returns every account holding the given role
// (TEACHER/STAFF/PARENT/STUDENT — the roles the block button applies to).
func (s *Service) ListAccountsByRole(ctx context.Context, roleName string) ([]AccountView, error) {
	rows, err := s.queries.ListAccountsWithRole(ctx, roleName)
	if err != nil {
		return nil, err
	}
	views := make([]AccountView, 0, len(rows))
	for _, r := range rows {
		views = append(views, AccountView{
			UserID:      r.UserID,
			PersonID:    r.PersonID,
			Email:       r.Email,
			FirstName:   r.FirstName,
			LastName:    r.LastName,
			Status:      r.Status,
			LastLoginAt: tsOrNil(r.LastLoginAt),
		})
	}
	return views, nil
}

// RevokeAdmin removes the ADMIN role (and MASTER_ADMIN, if held) from the
// target user, provided the caller is themselves a MASTER_ADMIN, the
// target isn't the caller, and enough active admins remain afterwards.
// A still-active MASTER_ADMIN can't be left idle alongside a demoted
// plain admin, so both roles go together.
func (s *Service) RevokeAdmin(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	if actorID == targetUserID {
		return ErrSelfDemotion
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	targetRoles, err := q.ListRolesForUser(ctx, targetUserID)
	if err != nil {
		return err
	}
	targetIsAdmin := false
	for _, r := range targetRoles {
		if r.Name == RoleAdmin || r.Name == RoleMasterAdmin {
			targetIsAdmin = true
		}
	}
	if !targetIsAdmin {
		return ErrNotAdmin
	}

	// Last-active-admin guard: the target must not be the only ACTIVE
	// holder remaining. Non-active targets can always be demoted (they
	// hold no live power).
	n, err := q.CountAdmins(ctx)
	if err != nil {
		return err
	}
	target, err := q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if n <= 1 && target.Status == StatusActive {
		return ErrLastAdmin
	}

	adminRole, err := q.GetRoleByName(ctx, RoleAdmin)
	if err != nil {
		return err
	}
	if err := q.RevokeRoleFromUser(ctx, db.RevokeRoleFromUserParams{
		UserID: targetUserID,
		RoleID: adminRole.ID,
	}); err != nil {
		return err
	}
	// Revoking a link that doesn't exist (no MASTER_ADMIN role held) is a
	// no-op DELETE of zero rows — safe to run unconditionally.
	masterRole, err := q.GetRoleByName(ctx, RoleMasterAdmin)
	if err != nil {
		return err
	}
	if err := q.RevokeRoleFromUser(ctx, db.RevokeRoleFromUserParams{
		UserID: targetUserID,
		RoleID: masterRole.ID,
	}); err != nil {
		return err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAdminAccessRevoked,
		ResourceType: audit.ResourceUser,
		ResourceID:   targetUserID,
		Metadata:     map[string]any{"revoked_roles": []string{RoleAdmin, RoleMasterAdmin}},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Block flips an ACTIVE account to SUSPENDED. auth.ValidateSession then
// fails closed on every subsequent request, so the user is locked out
// immediately — even mid-session. Idempotent: blocking an already-
// suspended account is a no-op, not an error.
func (s *Service) Block(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	return s.setAccountStatus(ctx, actorID, targetUserID, StatusSuspended)
}

// Unblock restores a SUSPENDED account to ACTIVE.
func (s *Service) Unblock(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	return s.setAccountStatus(ctx, actorID, targetUserID, StatusActive)
}

// CreateAdminInput is POST /admins' payload (M34): only the master admin
// reaches this endpoint, and it is the ONLY path that grants ADMIN start
// to zero (ordinary admins can invite teachers/parents through their own
// domain flows, not each other).
type CreateAdminInput struct {
	Email     string
	FirstName string
	LastName  string
}

// CreateAdmin provisions a new ADMIN account: person record, user record
// with an unusable placeholder password (the real credential comes from
// the Clerk invitation's "create your password" flow; in classic mode the
// master admin notices the pending state in the admins list and can
// revoke), and the ADMIN role — all in one transaction with the audit
// entry. The Clerk invitation itself is best-effort after commit.
func (s *Service) CreateAdmin(ctx context.Context, actorID uuid.UUID, in CreateAdminInput) (AdminView, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	firstName := strings.TrimSpace(in.FirstName)
	lastName := strings.TrimSpace(in.LastName)
	if email == "" || strings.Count(email, "@") != 1 || strings.HasSuffix(email, "@") || strings.HasPrefix(email, "@") {
		return AdminView{}, ErrUnknownRole // validation happens in the handler; belt-and-braces here
	}
	if firstName == "" || lastName == "" {
		return AdminView{}, ErrUnknownRole
	}

	if _, err := s.queries.GetUserByEmail(ctx, email); err == nil {
		return AdminView{}, ErrEmailTaken
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AdminView{}, err
	}

	// The placeholder password is sha-mapped to a value no human knows: a
	// fresh random token, hashed, discarded. Sign-in for admins is meant
	// to arrive through the Clerk bridge after invitation; the classic
	// mode login will simply reject the placeholder.
	placeholderToken, err := auth.GenerateToken()
	if err != nil {
		return AdminView{}, err
	}
	placeholderHash, err := auth.HashPassword(placeholderToken + "-admin-bootstrap")
	if err != nil {
		return AdminView{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AdminView{}, err
	}
	defer tx.Rollback(ctx)
	q := s.queries.WithTx(tx)

	person, err := q.CreatePerson(ctx, db.CreatePersonParams{
		FirstName: firstName,
		LastName:  lastName,
	})
	if err != nil {
		return AdminView{}, err
	}
	user, err := q.CreateUser(ctx, db.CreateUserParams{
		PersonID:     person.ID,
		Email:        email,
		PasswordHash: placeholderHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Raced the pre-check — same user-visible error either way.
			return AdminView{}, ErrEmailTaken
		}
		return AdminView{}, err
	}
	// M34: record who minted this admin — the power behind "inviter sees
	// their invitees" and the master-admin blocking rules.
	if err := q.SetUserInvitedBy(ctx, db.SetUserInvitedByParams{
		ID:        user.ID,
		InvitedBy: pgtype.UUID{Bytes: actorID, Valid: true},
	}); err != nil {
		return AdminView{}, err
	}
	role, err := q.GetRoleByName(ctx, RoleAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminView{}, ErrUnknownRole
		}
		return AdminView{}, err
	}
	if err := q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: user.ID, RoleID: role.ID}); err != nil {
		return AdminView{}, err
	}
	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionUserCreated,
		ResourceType: audit.ResourceUser,
		ResourceID:   user.ID,
		Metadata:     map[string]any{"email": email, "roles": []string{RoleAdmin}, "via": "add-admin"},
	}); err != nil {
		return AdminView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdminView{}, err
	}

	// Best-effort Clerk invitation after commit: the recipient creates
	// their own password through the emailed link and the session bridge
	// matches them by email on first sign-in. No-op when Clerk is off.
	if s.inviter != nil && s.inviter.Enabled() {
		go func(addr string) {
			ictx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			s.inviter.Create(ictx, addr)
		}(email)
	}

	return AdminView{
		UserID:    user.ID,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Status:    StatusActive,
	}, nil
}

// setAccountStatus is Block/Unblock's shared body. Governance (M34):
//
//	* nobody changes their own account status — a suspended account can't
//	  hold a live session anyway, so Unblock needs a second actor too.
//	* admin targets are blockable in exactly two cases: the caller is a
//	  MASTER_ADMIN (the explicit user requirement — blocking is a reversible
//	  suspend/restore; stripping privileges stays the role-revocation path),
//	  or the caller is the account's inviter (enforceable for every role).
//	* suspending a MASTER target additionally requires another ACTIVE
//	  master to exist — the guard runs for EVERY actor (a demoted inviter
//	  could otherwise lock the school out), not just master callers.
//	* non-admin targets: any caller with account.block may block.
func (s *Service) setAccountStatus(ctx context.Context, actorID, targetUserID uuid.UUID, newStatus string) error {
	if actorID == targetUserID {
		return ErrSelfBlock
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	target, err := q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	targetRoles, err := q.ListRolesForUser(ctx, targetUserID)
	if err != nil {
		return err
	}
	// Single pass over the target's roles: flags first, gates second —
	// breaking on the first matching role (ADMIN sorts before MASTER_ADMIN)
	// would otherwise leave targetIsMaster unset for dual-role admins and
	// silently skip the last-master guard below.
	targetIsMaster := false
	targetIsAdmin := false
	for _, r := range targetRoles {
		switch r.Name {
		case RoleMasterAdmin:
			targetIsMaster = true
			targetIsAdmin = true
		case RoleAdmin:
			targetIsAdmin = true
		}
	}
	if targetIsAdmin {
		// Admin target: gate on MASTER_ADMIN or inviter-of-target.
		actorRoles, err := q.ListRolesForUser(ctx, actorID)
		if err != nil {
			return err
		}
		actorIsMaster := false
		for _, ar := range actorRoles {
			if ar.Name == RoleMasterAdmin {
				actorIsMaster = true
			}
		}
		invitedByCaller := target.InvitedBy.Valid &&
			uuid.UUID(target.InvitedBy.Bytes) == actorID
		if !actorIsMaster && !invitedByCaller {
			return ErrAdminNotBlockable
		}
	}
	if targetIsMaster && newStatus == StatusSuspended {
		// Never lose the LAST ACTIVE master — the guard runs for every
		// actor, because a demoted ex-master INVITER could otherwise
		// suspend the school's only remaining master and lock it out.
		actives, err := countActiveMasterAdmins(ctx, targetUserID, q)
		if err != nil {
			return err
		}
		if actives == 0 { // (target excluded; see helper)
			return ErrLastMaster
		}
	}
	if target.Status == newStatus {
		return nil // idempotent: already in the requested state
	}

	if err := q.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
		Status: newStatus,
		ID:     targetUserID,
	}); err != nil {
		return err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAccountStatusChanged,
		ResourceType: audit.ResourceUser,
		ResourceID:   targetUserID,
		Metadata:     map[string]any{"old": target.Status, "new": newStatus},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// countActiveMasterAdmins reports how many ACTIVE MASTER_ADMIN users exist
// OTHER than excludeUserID (the candidate target). The suspend guard calls
// it before blocking: zero remaining actives means the target is the only
// ACTIVE master, which must not be blocked.
func countActiveMasterAdmins(ctx context.Context, excludeUserID uuid.UUID, q *db.Queries) (int, error) {
	count, err := q.CountActiveMasterAdminsExcept(ctx, excludeUserID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// --- pgtype helpers (per-package convention: duplicated, not shared) ---

func tsOrNil(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tm := t.Time
	return &tm
}
