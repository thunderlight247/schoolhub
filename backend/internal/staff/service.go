// Package staff implements staff management (section 12): adding, editing,
// searching, filtering, activating/deactivating staff, assigning department
// and position, and creating user accounts with system roles for them.
//
// Layering: HTTP handlers (handlers.go, reference_handlers.go) validate
// request shapes and map errors to responses; this Service owns business
// rules and transactions; persistence goes through sqlc-generated queries.
// Multi-step mutations (person + staff, user + role) run inside one
// transaction together with their audit row (internal/audit), so an action
// either fully happens with its audit entry or doesn't happen at all.
package staff

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thunderlight247/schoolhub/backend/internal/email"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
)

// PostgreSQL error codes this service maps to friendlier errors (a subset
// of the pgerrcode list; kept as literals to avoid a dependency for two
// constants).
const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
)

// Employment statuses (section 12).
const (
	StatusActive    = "ACTIVE"
	StatusInactive  = "INACTIVE"
	StatusSuspended = "SUSPENDED"
	StatusOnLeave   = "ON_LEAVE"
)

// EmploymentStatuses is the full set, for handler-side validation.
var EmploymentStatuses = []string{StatusActive, StatusInactive, StatusSuspended, StatusOnLeave}

// PositionCategories mirrors the staff_positions.category CHECK constraint
// in migration 000005 (section 12's category list). Kept in sync manually;
// a mismatch surfaces as a rejected create, not silently wrong data.
var PositionCategories = []string{
	"TEACHING", "ADMINISTRATION", "ICT", "ACCOUNTS", "MANAGEMENT", "SUPPORT",
}

// Sentinel errors — handlers map these to specific HTTP responses; anything
// else is an unexpected failure (500) whose details are never exposed.
var (
	ErrStaffNotFound      = errors.New("staff: staff member not found")
	ErrStaffNumberTaken   = errors.New("staff: staff number already in use")
	ErrEmailTaken         = errors.New("staff: email already in use by an account")
	ErrAccountExists      = errors.New("staff: staff member already has a user account")
	ErrUnknownDepartment  = errors.New("staff: unknown department")
	ErrUnknownPosition    = errors.New("staff: unknown staff position")
	ErrUnknownRole        = errors.New("staff: unknown role")
	ErrUserNotFound       = errors.New("staff: user account not found")
	ErrInvalidStatus      = errors.New("staff: invalid employment status")
	ErrInvalidCategory    = errors.New("staff: invalid position category")
	ErrReferenceNameTaken = errors.New("staff: department or position name already in use")
)

// Service holds the pool (needed for transactions) and the query set.
// It depends on the concrete *db.Queries rather than the db.Querier
// interface because WithTx is only defined on the concrete type — the
// trade-off for correct transactional behavior; the transactional paths
// are covered by end-to-end tests rather than unit tests.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	// mailer is optional (nil in tests and in deployments without an
	// email provider): a broken email provider must never prevent an
	// admin from creating an account — the email send is best-effort and
	// lives AFTER the transaction commits.
	mailer email.Sender
	// inviter is optional (nil without Clerk): when set, account creation
	// also asks Clerk to email an official invitation for the same
	// address — best-effort, after commit.
	inviter *auth.ClerkInviter
	// frontendURL is used to build the sign-in link in the welcome email.
	frontendURL string
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

// SetMailer attaches an optional email sender (and the frontend URL used
// in welcome-email links). Safe to call once during wiring.
func (s *Service) SetMailer(mailer email.Sender, frontendURL string) {
	s.mailer = mailer
	if frontendURL != "" {
		s.frontendURL = frontendURL
	}
}

// SetInviter attaches the optional Clerk invitation issuer.
func (s *Service) SetInviter(inviter *auth.ClerkInviter) {
	s.inviter = inviter
}

// Account is the user-account view attached to a staff member (nil until
// an admin creates one). Roles are resolved fresh from user_roles.
type Account struct {
	ID     uuid.UUID
	Email  string
	Status string
	Roles  []string
}

// StaffDetail is the full staff profile: the joined employment row plus the
// (optional) account.
type StaffDetail struct {
	Row     db.GetStaffByIDRow
	Account *Account
}

// CreateStaffInput is the fully-validated input for registering a new staff
// member. Optional person/employment fields arrive as pointers (nil = not
// provided). Employment status always starts ACTIVE.
type CreateStaffInput struct {
	StaffNumber  string
	FirstName    string
	LastName     string
	OtherNames   *string
	Gender       *string
	DateOfBirth  *time.Time
	Phone        *string
	Email        *string
	Address      *string
	DepartmentID *uuid.UUID
	PositionID   *uuid.UUID
	HiredDate    *time.Time
}

// UpdateStaffInput carries PATCH semantics for person fields: nil = leave
// the current value unchanged. Assignment fields are NOT here — they are a
// full-replace operation on their own endpoint (SetAssignment), matching
// section 12's "assign department/position" actions.
type UpdateStaffInput struct {
	FirstName   *string
	LastName    *string
	OtherNames  *string
	Gender      *string
	DateOfBirth *time.Time
	Phone       *string
	Email       *string
	Address     *string
}

// ListFilter is the backend-side filter/pagination request (section 25).
// Empty strings mean "no filter"; the service clamps pagination.
type ListFilter struct {
	DepartmentID     *uuid.UUID
	EmploymentStatus string
	PositionCategory string
	Search           string
	Page             int
	PageSize         int
}

// --- pgtype conversion helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// filterText maps "" (no filter) to SQL NULL.
func filterText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func toPgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

// --- creation ---

// CreateStaff registers a new staff member: person row + staff employment
// row + audit entry in one transaction. The lookups for department/position
// are verified first so the admin gets a clear error instead of a raw FK
// violation. Status always starts ACTIVE; status changes are a separate,
// separately-gated operation.
func (s *Service) CreateStaff(ctx context.Context, actorID uuid.UUID, in CreateStaffInput) (StaffDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StaffDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if in.DepartmentID != nil {
		if _, err := q.GetDepartmentByID(ctx, *in.DepartmentID); err != nil {
			return StaffDetail{}, ErrUnknownDepartment
		}
	}
	if in.PositionID != nil {
		if _, err := q.GetStaffPositionByID(ctx, *in.PositionID); err != nil {
			return StaffDetail{}, ErrUnknownPosition
		}
	}

	person, err := q.CreatePerson(ctx, db.CreatePersonParams{
		FirstName:   in.FirstName,
		LastName:    in.LastName,
		OtherNames:  toPgText(in.OtherNames),
		DateOfBirth: toPgDate(in.DateOfBirth),
		Gender:      toPgText(in.Gender),
		Phone:       toPgText(in.Phone),
		Email:       toPgText(in.Email),
		Address:     toPgText(in.Address),
	})
	if err != nil {
		return StaffDetail{}, err
	}

	staffRow, err := q.CreateStaff(ctx, db.CreateStaffParams{
		PersonID:         person.ID,
		StaffNumber:      in.StaffNumber,
		DepartmentID:     toPgUUID(in.DepartmentID),
		PositionID:       toPgUUID(in.PositionID),
		EmploymentStatus: StatusActive,
		HiredDate:        toPgDate(in.HiredDate),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return StaffDetail{}, ErrStaffNumberTaken
		}
		return StaffDetail{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionStaffCreated,
		ResourceType: audit.ResourceStaff,
		ResourceID:   staffRow.ID,
		Metadata:     map[string]any{"staff_number": in.StaffNumber},
	}); err != nil {
		return StaffDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StaffDetail{}, err
	}

	return s.GetStaff(ctx, staffRow.ID)
}

// --- reads ---

// GetStaff returns the full staff profile. The optional user account is
// resolved separately (pgx.ErrNoRows means "no account", not an error) so
// a staff member without a login is a normal state, not a failure.
func (s *Service) GetStaff(ctx context.Context, id uuid.UUID) (StaffDetail, error) {
	row, err := s.queries.GetStaffByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StaffDetail{}, ErrStaffNotFound
		}
		return StaffDetail{}, err
	}

	detail := StaffDetail{Row: row}

	user, err := s.queries.GetUserByPersonID(ctx, row.PersonID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return StaffDetail{}, err
	}
	if err == nil {
		roles, err := s.queries.ListRolesForUser(ctx, user.ID)
		if err != nil {
			return StaffDetail{}, err
		}
		roleNames := make([]string, 0, len(roles))
		for _, r := range roles {
			roleNames = append(roleNames, r.Name)
		}
		detail.Account = &Account{ID: user.ID, Email: user.Email, Status: user.Status, Roles: roleNames}
	}

	return detail, nil
}

// ListStaff returns one page of the filtered staff list plus the total
// matching count (for the pagination metadata). Both queries apply the
// identical WHERE clause — see the sync note on ListStaff in staff.sql.
func (s *Service) ListStaff(ctx context.Context, f ListFilter) ([]db.ListStaffRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	listParams := db.ListStaffParams{
		DepartmentID:     toPgUUID(f.DepartmentID),
		EmploymentStatus: filterText(f.EmploymentStatus),
		PositionCategory: filterText(f.PositionCategory),
		Search:           filterText(f.Search),
		PageOffset:       int32((page - 1) * pageSize),
		PageLimit:        int32(pageSize),
	}
	countParams := db.CountStaffParams{
		DepartmentID:     listParams.DepartmentID,
		EmploymentStatus: listParams.EmploymentStatus,
		PositionCategory: listParams.PositionCategory,
		Search:           listParams.Search,
	}

	total, err := s.queries.CountStaff(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.queries.ListStaff(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// --- pagination ---

// Default and maximum list page sizes (section 25).
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// NormalizePagination clamps client-provided pagination to sane bounds so
// one request can't pull the whole table. Handlers parse the raw strings;
// these clamped values are what reach the queries.
func NormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// contains reports whether want is in values. Small enough that a
// dependency on the slices package isn't warranted.
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// --- updates (section 12: "Edit staff", "Assign department/position") ---

// SetAssignmentInput is the full-replace input for the assignment endpoint.
// Every field is sent explicitly and a nil clears the current value — a
// staff record with no department/position assigned yet is a legitimate
// state (see the migration 000005 comment on the nullable FK columns).
type SetAssignmentInput struct {
	DepartmentID *uuid.UUID
	PositionID   *uuid.UUID
	HiredDate    *time.Time
}

// UpdateStaff applies PATCH semantics to a staff member's person fields:
// nil leaves the current value in place. The person update and its audit
// entry run in one transaction. Assignment fields and employment status
// are deliberately separate endpoints (SetAssignment, SetEmploymentStatus)
// because they carry their own permissions and audit events.
func (s *Service) UpdateStaff(ctx context.Context, actorID, id uuid.UUID, in UpdateStaffInput) (StaffDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StaffDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetStaffByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StaffDetail{}, ErrStaffNotFound
		}
		return StaffDetail{}, err
	}

	// Merge: nil fields keep the current person values. The "changed" list
	// names only the fields actually sent, so the audit row records what
	// was edited, not the whole record.
	var changed []string
	firstName := current.FirstName
	if in.FirstName != nil {
		firstName = *in.FirstName
		changed = append(changed, "first_name")
	}
	lastName := current.LastName
	if in.LastName != nil {
		lastName = *in.LastName
		changed = append(changed, "last_name")
	}
	otherNames := current.OtherNames
	if in.OtherNames != nil {
		otherNames = pgtype.Text{String: *in.OtherNames, Valid: true}
		changed = append(changed, "other_names")
	}
	dateOfBirth := current.DateOfBirth
	if in.DateOfBirth != nil {
		dateOfBirth = pgtype.Date{Time: *in.DateOfBirth, Valid: true}
		changed = append(changed, "date_of_birth")
	}
	gender := current.Gender
	if in.Gender != nil {
		gender = pgtype.Text{String: *in.Gender, Valid: true}
		changed = append(changed, "gender")
	}
	phone := current.Phone
	if in.Phone != nil {
		phone = pgtype.Text{String: *in.Phone, Valid: true}
		changed = append(changed, "phone")
	}
	email := current.Email
	if in.Email != nil {
		email = pgtype.Text{String: *in.Email, Valid: true}
		changed = append(changed, "email")
	}
	address := current.Address
	if in.Address != nil {
		address = pgtype.Text{String: *in.Address, Valid: true}
		changed = append(changed, "address")
	}

	if _, err := q.UpdatePerson(ctx, db.UpdatePersonParams{
		ID:          current.PersonID,
		FirstName:   firstName,
		LastName:    lastName,
		OtherNames:  otherNames,
		DateOfBirth: dateOfBirth,
		Gender:      gender,
		Phone:       phone,
		Email:       email,
		Address:     address,
	}); err != nil {
		return StaffDetail{}, err
	}

	if len(changed) > 0 {
		if err := audit.Record(ctx, q, audit.Entry{
			ActorUserID:  actorID,
			Action:       audit.ActionStaffUpdated,
			ResourceType: audit.ResourceStaff,
			ResourceID:   id,
			Metadata:     map[string]any{"fields": changed},
		}); err != nil {
			return StaffDetail{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return StaffDetail{}, err
	}

	return s.GetStaff(ctx, id)
}

// SetEmploymentStatus moves a staff member between ACTIVE / INACTIVE /
// SUSPENDED / ON_LEAVE (section 12's activate/deactivate). The route is
// gated by staff.deactivate; the audit entry captures the old→new
// transition, which is exactly what section 29's log exists for.
func (s *Service) SetEmploymentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (StaffDetail, error) {
	if !contains(EmploymentStatuses, status) {
		return StaffDetail{}, ErrInvalidStatus
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StaffDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetStaffByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StaffDetail{}, ErrStaffNotFound
		}
		return StaffDetail{}, err
	}

	if _, err := q.SetStaffEmploymentStatus(ctx, db.SetStaffEmploymentStatusParams{
		EmploymentStatus: status,
		ID:               id,
	}); err != nil {
		return StaffDetail{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionStaffStatusChanged,
		ResourceType: audit.ResourceStaff,
		ResourceID:   id,
		Metadata:     map[string]any{"old": current.EmploymentStatus, "new": status},
	}); err != nil {
		return StaffDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StaffDetail{}, err
	}

	return s.GetStaff(ctx, id)
}

// --- user accounts for staff (section 12: "Create user accounts with
// system roles") ---

// CreateAccountInput registers the login account for a staff member.
// Password policy is enforced by the handler's binding tags; the service
// treats it as opaque. RoleNames must be known role names (ADMIN, TEACHER,
// STAFF, PARENT, STUDENT — fixed by migration 000002's CHECK).
type CreateAccountInput struct {
	Email     string
	Password  string
	RoleNames []string
}

// CreateAccount creates the user account (Argon2id-hashed password) and
// assigns the requested roles, with audit entries, all in one transaction.
// Login-ability is controlled by users.status and later account actions;
// employment status and account status are intentionally independent
// (see migration 000005's comment).
func (s *Service) CreateAccount(ctx context.Context, actorID, staffID uuid.UUID, in CreateAccountInput) (Account, error) {
	detail, err := s.GetStaff(ctx, staffID)
	if err != nil {
		return Account{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.GetUserByPersonID(ctx, detail.Row.PersonID); err == nil {
		return Account{}, ErrAccountExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Account{}, err
	}

	if _, err := q.GetUserByEmail(ctx, in.Email); err == nil {
		return Account{}, ErrEmailTaken
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Account{}, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return Account{}, err
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		PersonID:     detail.Row.PersonID,
		Email:        in.Email,
		PasswordHash: hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			// Raced with another admin creating the same email between the
			// check above and the insert — same user-visible answer.
			return Account{}, ErrEmailTaken
		}
		return Account{}, err
	}
	// M34: record who provisioned this staff account (inviter tracking).
	if err := q.SetUserInvitedBy(ctx, db.SetUserInvitedByParams{
		ID:        user.ID,
		InvitedBy: pgtype.UUID{Bytes: actorID, Valid: true},
	}); err != nil {
		return Account{}, err
	}

	for _, name := range in.RoleNames {
		role, err := q.GetRoleByName(ctx, name)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Account{}, ErrUnknownRole
			}
			return Account{}, err
		}
		if err := q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: user.ID, RoleID: role.ID}); err != nil {
			return Account{}, err
		}
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionUserCreated,
		ResourceType: audit.ResourceUser,
		ResourceID:   user.ID,
		Metadata:     map[string]any{"email": in.Email, "roles": in.RoleNames, "staff_id": staffID},
	}); err != nil {
		return Account{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Account{}, err
	}

	detail, err = s.GetStaff(ctx, staffID)
	if err != nil {
		return Account{}, err
	}
	if detail.Account == nil {
		return Account{}, ErrUserNotFound
	}

	// Best-effort welcome email (after commit — never blocks or fails the
	// account creation itself). No-op when no provider is configured.
	if svc, ok := s.mailer.(*email.Service); ok && svc != nil && svc.Enabled() {
		firstName := detail.Row.FirstName
		lastName := detail.Row.LastName
		to := in.Email
		go func() {
			mailCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			signIn := s.frontendURL
			if signIn == "" {
				signIn = "http://localhost:3000/login"
			}
			if err := svc.SendWelcome(mailCtx, to, firstName+" "+lastName, signIn); err != nil {
				// Already logged inside the service; nothing more to do.
			}
		}()
	}

	// Best-effort Clerk invitation (after commit): the recipient gets the
	// official "create your password" email for this same address; the
	// bridge matches them by email on first sign-in. No-op when Clerk is
	// not configured.
	if s.inviter != nil && s.inviter.Enabled() {
		go func(addr string) {
			ictx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			s.inviter.Create(ictx, addr)
		}(in.Email)
	}

	return *detail.Account, nil
}

// SetAccountRoles replaces the staff member's account roles wholesale with
// the given list; an empty list removes all roles. The account must exist
// first (roles attach to users, not to staff employment records).
func (s *Service) SetAccountRoles(ctx context.Context, actorID, staffID uuid.UUID, roleNames []string) ([]string, error) {
	detail, err := s.GetStaff(ctx, staffID)
	if err != nil {
		return nil, err
	}
	if detail.Account == nil {
		return nil, ErrUserNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if err := q.ClearUserRoles(ctx, detail.Account.ID); err != nil {
		return nil, err
	}

	assigned := make([]string, 0, len(roleNames))
	seen := map[string]bool{}
	for _, name := range roleNames {
		if seen[name] {
			continue // duplicates in the request are harmless, not errors
		}
		seen[name] = true

		role, err := q.GetRoleByName(ctx, name)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrUnknownRole
			}
			return nil, err
		}
		if err := q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: detail.Account.ID, RoleID: role.ID}); err != nil {
			return nil, err
		}
		assigned = append(assigned, role.Name)
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionUserRolesUpdated,
		ResourceType: audit.ResourceUser,
		ResourceID:   detail.Account.ID,
		Metadata:     map[string]any{"roles": assigned},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	detail, err = s.GetStaff(ctx, staffID)
	if err != nil {
		return nil, err
	}
	if detail.Account == nil {
		return nil, ErrUserNotFound
	}
	return detail.Account.Roles, nil
}

// metadata helpers for audit entries: uuidOrNil/dateOrNil map Go optionals
// to JSON values (nil serializes as null, matching the unassigned state).

func uuidOrNil(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return id.String()
}

func dateOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

// --- reference data: departments and staff positions ---

// ListDepartments returns the departments for assignment dropdowns and
// admin list screens. Reads produce no audit rows — section 29 covers
// actions, not views.
func (s *Service) ListDepartments(ctx context.Context) ([]db.Department, error) {
	return s.queries.ListDepartments(ctx)
}

// ListStaffPositions returns the position catalog, grouped by category
// (the query orders by category, title).
func (s *Service) ListStaffPositions(ctx context.Context) ([]db.StaffPosition, error) {
	return s.queries.ListStaffPositions(ctx)
}

// CreateDepartment adds a department (admin reference data, section 12).
// departments.name is UNIQUE at the database level; a collision maps to
// ErrReferenceNameTaken so the admin gets a 409 instead of a 500.
func (s *Service) CreateDepartment(ctx context.Context, actorID uuid.UUID, name string, description *string) (db.Department, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Department{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	dept, err := q.CreateDepartment(ctx, db.CreateDepartmentParams{
		Name:        name,
		Description: toPgText(description),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Department{}, ErrReferenceNameTaken
		}
		return db.Department{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionDepartmentCreated,
		ResourceType: audit.ResourceDepartment,
		ResourceID:   dept.ID,
		Metadata:     map[string]any{"name": name},
	}); err != nil {
		return db.Department{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Department{}, err
	}
	return dept, nil
}

// CreateStaffPosition adds a position to the catalog. The category is
// validated here (not just in the handler) because the set is business
// data mirrored from migration 000005's CHECK constraint.
func (s *Service) CreateStaffPosition(ctx context.Context, actorID uuid.UUID, title, category string, description *string) (db.StaffPosition, error) {
	if !contains(PositionCategories, category) {
		return db.StaffPosition{}, ErrInvalidCategory
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.StaffPosition{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	position, err := q.CreateStaffPosition(ctx, db.CreateStaffPositionParams{
		Title:       title,
		Category:    category,
		Description: toPgText(description),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.StaffPosition{}, ErrReferenceNameTaken
		}
		return db.StaffPosition{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionPositionCreated,
		ResourceType: audit.ResourceStaffPosition,
		ResourceID:   position.ID,
		Metadata:     map[string]any{"title": title, "category": category},
	}); err != nil {
		return db.StaffPosition{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.StaffPosition{}, err
	}
	return position, nil
}

// SetAssignment replaces the department/position/hired-date triple in full
// and records a STAFF_UPDATED audit entry. Lookups are verified first so
// the admin gets a clear 400 instead of a raw FK violation.
func (s *Service) SetAssignment(ctx context.Context, actorID, id uuid.UUID, in SetAssignmentInput) (StaffDetail, error) {
	if in.DepartmentID != nil {
		if _, err := s.queries.GetDepartmentByID(ctx, *in.DepartmentID); err != nil {
			return StaffDetail{}, ErrUnknownDepartment
		}
	}
	if in.PositionID != nil {
		if _, err := s.queries.GetStaffPositionByID(ctx, *in.PositionID); err != nil {
			return StaffDetail{}, ErrUnknownPosition
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StaffDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.UpdateStaffAssignment(ctx, db.UpdateStaffAssignmentParams{
		DepartmentID: toPgUUID(in.DepartmentID),
		PositionID:   toPgUUID(in.PositionID),
		HiredDate:    toPgDate(in.HiredDate),
		ID:           id,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StaffDetail{}, ErrStaffNotFound
		}
		return StaffDetail{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionStaffUpdated,
		ResourceType: audit.ResourceStaff,
		ResourceID:   id,
		Metadata: map[string]any{
			"department_id": uuidOrNil(in.DepartmentID),
			"position_id":   uuidOrNil(in.PositionID),
			"hired_date":    dateOrNil(in.HiredDate),
		},
	}); err != nil {
		return StaffDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StaffDetail{}, err
	}

	return s.GetStaff(ctx, id)
}
