// Package students implements the students and guardians domain (sections
// 14-15): registering/editing/searching/filtering/archiving students, and
// managing guardians plus the student-guardian associations between them.
// Guardians live in this package rather than their own (see migration
// 000007's comment) for the same reason M6 folded departments and
// staff_positions into internal/staff.
//
// Layering matches internal/staff exactly: HTTP handlers (handlers.go)
// validate request shapes and map errors to responses; this Service owns
// business rules and transactions; persistence goes through sqlc-generated
// queries. Multi-step mutations run inside one transaction together with
// their audit row, so an action either fully happens with its audit entry
// or doesn't happen at all.
package students

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
	"github.com/thunderlight247/schoolhub/backend/internal/email"
)

// PostgreSQL error codes this service maps to friendlier errors. Duplicated
// from internal/staff rather than shared — matching that package's
// per-package convention (see its own comment on this choice).
const (
	pgErrUniqueViolation = "23505"
)

// Student statuses (section 14).
const (
	StatusActive   = "ACTIVE"
	StatusArchived = "ARCHIVED"
)

// StudentStatuses is the full set, for handler-side validation.
var StudentStatuses = []string{StatusActive, StatusArchived}

// Sentinel errors — handlers map these to specific HTTP responses; anything
// else is an unexpected failure (500) whose details are never exposed.
var (
	ErrStudentNotFound       = errors.New("students: student not found")
	ErrStudentNumberTaken    = errors.New("students: student number already in use")
	ErrInvalidStatus         = errors.New("students: invalid status")
	ErrGuardianNotFound      = errors.New("students: guardian not found")
	ErrGuardianAlreadyLinked = errors.New("students: guardian is already linked to this student")
	ErrGuardianLinkNotFound  = errors.New("students: guardian is not linked to this student")
)

// Service holds the pool (needed for transactions) and the query set, same
// shape as internal/staff.Service for the same reason: WithTx is only
// defined on the concrete *db.Queries type.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries

	// Optional outbound email + Clerk invitations, set once during wiring
	// (SetMailer/SetInviter); both flows are best-effort after commit.
	mailer      email.Sender
	inviter     *auth.ClerkInviter
	frontendURL string
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

// SetMailer and SetInviter attach the optional outbound-email provider and
// the optional Clerk invitation issuer (both no-ops before wiring). They
// are safe to call once during composition in cmd/server/main.go.
func (s *Service) SetMailer(mailer email.Sender, frontendURL string) {
	s.mailer = mailer
	if frontendURL != "" {
		s.frontendURL = frontendURL
	}
}

func (s *Service) SetInviter(inviter *auth.ClerkInviter) {
	s.inviter = inviter
}

// ErrEmailTaken / ErrAccountExists mirror internal/staff's account errors;
// the students package now provisions PARENT accounts too, and handlers
// map these to the same structured responses the staff endpoints return.
var (
	ErrEmailTaken    = errors.New("students: email already in use by an account")
	ErrAccountExists = errors.New("students: guardian already has a user account")
	ErrUnknownRole   = errors.New("students: unknown role name")
)

// StudentDetail is the full student profile: the joined person+student row
// plus their linked guardians.
type StudentDetail struct {
	Row       db.GetStudentByIDRow
	Guardians []db.ListGuardiansForStudentRow
}

// GuardianDetail is the full guardian profile: the joined person+guardian
// row plus the students they're linked to (and, when present, the roles
// of the guardian's login account, resolved fresh from user_roles).
type GuardianDetail struct {
	Row          db.GetGuardianByIDRow
	Students     []db.ListStudentsForGuardianRow
	AccountRoles []string
}

// CreateStudentInput registers a brand-new person as a student. Unlike
// CreateGuardian below, there is no "link to an existing person" path here
// — a newly admitted student is, in this milestone, always a new person
// record, matching how CreateStaff works in internal/staff.
type CreateStudentInput struct {
	StudentNumber string
	FirstName     string
	LastName      string
	OtherNames    *string
	Gender        *string
	DateOfBirth   *time.Time
	Phone         *string
	Email         *string
	Address       *string
	AdmissionDate *time.Time
}

// UpdateStudentInput carries PATCH semantics for person fields: nil = leave
// the current value unchanged. student_number is immutable after creation
// (same precedent as staff_number); admission_date has no update path in
// this milestone — a small follow-up if a real correction need shows up,
// not speculative scope now.
type UpdateStudentInput struct {
	FirstName   *string
	LastName    *string
	OtherNames  *string
	Gender      *string
	DateOfBirth *time.Time
	Phone       *string
	Email       *string
	Address     *string
}

// ListFilter is the backend-side filter/pagination request for students
// (section 25).
type ListFilter struct {
	Status   string
	Search   string
	Page     int
	PageSize int
}

// CreateGuardianInput registers a brand-new person as a guardian. Note: if
// the guardian is already a person in the system (e.g. an existing staff
// member who is also a parent — section 9's own example), this still
// creates a separate person row rather than detecting and linking to the
// existing one. Person deduplication/matching isn't addressed by the spec
// and would need its own design; flagging the limitation here rather than
// solving it.
type CreateGuardianInput struct {
	FirstName   string
	LastName    string
	OtherNames  *string
	Gender      *string
	DateOfBirth *time.Time
	Phone       *string
	Email       *string
	Address     *string
	Occupation  *string
}

// UpdateGuardianInput carries PATCH semantics for both person fields and
// the guardian-specific occupation field: nil = leave unchanged.
type UpdateGuardianInput struct {
	FirstName   *string
	LastName    *string
	OtherNames  *string
	Gender      *string
	DateOfBirth *time.Time
	Phone       *string
	Email       *string
	Address     *string
	Occupation  *string
}

// GuardianListFilter is the backend-side filter/pagination request for the
// guardians directory / "search existing guardian to link" workflow.
type GuardianListFilter struct {
	Search   string
	Page     int
	PageSize int
}

// LinkGuardianInput links an EXISTING guardian (by id) to a student. There
// is no "create and link in one step" endpoint — guardian creation
// (CreateGuardian) and linking (LinkGuardianToStudent) are deliberately
// separate steps, matching how internal/staff keeps CreateStaff,
// CreateAccount, and SetAssignment as separate operations rather than one
// combined endpoint.
type LinkGuardianInput struct {
	GuardianID   uuid.UUID
	Relationship string
}

// --- pgtype conversion helpers (duplicated from internal/staff; see that
// package's comment on why these aren't shared) ---

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

// --- pagination (duplicated from internal/staff) ---

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// NormalizePagination clamps client-provided pagination to sane bounds.
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

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// --- students: creation ---

// CreateStudent registers a new student: person row + student row + audit
// entry in one transaction. Status always starts ACTIVE.
func (s *Service) CreateStudent(ctx context.Context, actorID uuid.UUID, in CreateStudentInput) (StudentDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StudentDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

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
		return StudentDetail{}, err
	}

	studentRow, err := q.CreateStudent(ctx, db.CreateStudentParams{
		PersonID:      person.ID,
		StudentNumber: in.StudentNumber,
		AdmissionDate: toPgDate(in.AdmissionDate),
		Status:        StatusActive,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return StudentDetail{}, ErrStudentNumberTaken
		}
		return StudentDetail{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionStudentCreated,
		ResourceType: audit.ResourceStudent,
		ResourceID:   studentRow.ID,
		Metadata:     map[string]any{"student_number": in.StudentNumber},
	}); err != nil {
		return StudentDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StudentDetail{}, err
	}

	return s.GetStudent(ctx, studentRow.ID)
}

// --- students: reads ---

// GetStudent returns the full student profile including linked guardians.
func (s *Service) GetStudent(ctx context.Context, id uuid.UUID) (StudentDetail, error) {
	row, err := s.queries.GetStudentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StudentDetail{}, ErrStudentNotFound
		}
		return StudentDetail{}, err
	}

	guardians, err := s.queries.ListGuardiansForStudent(ctx, id)
	if err != nil {
		return StudentDetail{}, err
	}

	return StudentDetail{Row: row, Guardians: guardians}, nil
}

// ListStudents returns one page of the filtered student list plus the
// total matching count. Both queries apply the identical WHERE clause —
// see the sync note in students.sql.
func (s *Service) ListStudents(ctx context.Context, f ListFilter) ([]db.ListStudentsRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	listParams := db.ListStudentsParams{
		Status:     filterText(f.Status),
		Search:     filterText(f.Search),
		PageOffset: int32((page - 1) * pageSize),
		PageLimit:  int32(pageSize),
	}
	countParams := db.CountStudentsParams{
		Status: listParams.Status,
		Search: listParams.Search,
	}

	total, err := s.queries.CountStudents(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.queries.ListStudents(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// --- students: updates ---

// UpdateStudent applies PATCH semantics to a student's person fields: nil
// leaves the current value in place. Mirrors internal/staff.UpdateStaff's
// merge pattern exactly, including recording only the fields actually
// changed in the audit entry.
func (s *Service) UpdateStudent(ctx context.Context, actorID, id uuid.UUID, in UpdateStudentInput) (StudentDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StudentDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetStudentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StudentDetail{}, ErrStudentNotFound
		}
		return StudentDetail{}, err
	}

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
		return StudentDetail{}, err
	}

	if len(changed) > 0 {
		if err := audit.Record(ctx, q, audit.Entry{
			ActorUserID:  actorID,
			Action:       audit.ActionStudentUpdated,
			ResourceType: audit.ResourceStudent,
			ResourceID:   id,
			Metadata:     map[string]any{"fields": changed},
		}); err != nil {
			return StudentDetail{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return StudentDetail{}, err
	}

	return s.GetStudent(ctx, id)
}

// SetStudentStatus moves a student between ACTIVE and ARCHIVED (section
// 14's "archive/deactivate"). Gated by student.archive, stricter than
// student.update — mirrors staff.deactivate vs staff.update.
func (s *Service) SetStudentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (StudentDetail, error) {
	if !contains(StudentStatuses, status) {
		return StudentDetail{}, ErrInvalidStatus
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StudentDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetStudentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StudentDetail{}, ErrStudentNotFound
		}
		return StudentDetail{}, err
	}

	if _, err := q.SetStudentStatus(ctx, db.SetStudentStatusParams{
		Status: status,
		ID:     id,
	}); err != nil {
		return StudentDetail{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionStudentStatusChanged,
		ResourceType: audit.ResourceStudent,
		ResourceID:   id,
		Metadata:     map[string]any{"old": current.Status, "new": status},
	}); err != nil {
		return StudentDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return StudentDetail{}, err
	}

	return s.GetStudent(ctx, id)
}

// --- guardians ---

// CreateGuardian registers a new guardian (standalone, not yet linked to
// any student): person row + guardian row + audit entry in one
// transaction.
func (s *Service) CreateGuardian(ctx context.Context, actorID uuid.UUID, in CreateGuardianInput) (GuardianDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GuardianDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

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
		return GuardianDetail{}, err
	}

	guardianRow, err := q.CreateGuardian(ctx, db.CreateGuardianParams{
		PersonID:   person.ID,
		Occupation: toPgText(in.Occupation),
	})
	if err != nil {
		return GuardianDetail{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionGuardianCreated,
		ResourceType: audit.ResourceGuardian,
		ResourceID:   guardianRow.ID,
		Metadata:     map[string]any{},
	}); err != nil {
		return GuardianDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return GuardianDetail{}, err
	}

	return s.GetGuardian(ctx, guardianRow.ID)
}

// GetGuardian returns the full guardian profile including linked students.
func (s *Service) GetGuardian(ctx context.Context, id uuid.UUID) (GuardianDetail, error) {
	row, err := s.queries.GetGuardianByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GuardianDetail{}, ErrGuardianNotFound
		}
		return GuardianDetail{}, err
	}

	studentsForGuardian, err := s.queries.ListStudentsForGuardian(ctx, id)
	if err != nil {
		return GuardianDetail{}, err
	}

	// Roles for the account summary (empty when no login account exists).
	var accountRoles []string
	if row.AccountID.Valid {
		roleRows, err := s.queries.ListRolesForUser(ctx, uuid.UUID(row.AccountID.Bytes))
		if err != nil {
			return GuardianDetail{}, err
		}
		accountRoles = make([]string, 0, len(roleRows))
		for _, r := range roleRows {
			accountRoles = append(accountRoles, r.Name)
		}
	}

	return GuardianDetail{Row: row, Students: studentsForGuardian, AccountRoles: accountRoles}, nil
}

// ListGuardians returns one page of the filtered guardian list plus the
// total matching count — the "search existing guardian to link" workflow
// as well as a standalone directory.
func (s *Service) ListGuardians(ctx context.Context, f GuardianListFilter) ([]db.ListGuardiansRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	listParams := db.ListGuardiansParams{
		Search:     filterText(f.Search),
		PageOffset: int32((page - 1) * pageSize),
		PageLimit:  int32(pageSize),
	}
	total, err := s.queries.CountGuardians(ctx, listParams.Search)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.queries.ListGuardians(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// UpdateGuardian applies PATCH semantics to both a guardian's person fields
// and their occupation: nil leaves the current value unchanged. Mirrors
// UpdateStudent/UpdateStaff's merge-and-record-changed-fields pattern.
func (s *Service) UpdateGuardian(ctx context.Context, actorID, id uuid.UUID, in UpdateGuardianInput) (GuardianDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GuardianDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetGuardianByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GuardianDetail{}, ErrGuardianNotFound
		}
		return GuardianDetail{}, err
	}

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
		return GuardianDetail{}, err
	}

	if in.Occupation != nil {
		if _, err := q.UpdateGuardianOccupation(ctx, db.UpdateGuardianOccupationParams{
			ID:         id,
			Occupation: toPgText(in.Occupation),
		}); err != nil {
			return GuardianDetail{}, err
		}
		changed = append(changed, "occupation")
	}

	if len(changed) > 0 {
		if err := audit.Record(ctx, q, audit.Entry{
			ActorUserID:  actorID,
			Action:       audit.ActionGuardianUpdated,
			ResourceType: audit.ResourceGuardian,
			ResourceID:   id,
			Metadata:     map[string]any{"fields": changed},
		}); err != nil {
			return GuardianDetail{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return GuardianDetail{}, err
	}

	return s.GetGuardian(ctx, id)
}

// --- student-guardian links ---

// LinkGuardianToStudent attaches an existing guardian to an existing
// student with a relationship label. Both sides are verified to exist
// first so the admin gets a clear 404 instead of a raw FK violation; a
// duplicate link maps to ErrGuardianAlreadyLinked (the UNIQUE(student_id,
// guardian_id) constraint from migration 000007).
func (s *Service) LinkGuardianToStudent(ctx context.Context, actorID, studentID uuid.UUID, in LinkGuardianInput) ([]db.ListGuardiansForStudentRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.GetStudentByID(ctx, studentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	if _, err := q.GetGuardianByID(ctx, in.GuardianID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGuardianNotFound
		}
		return nil, err
	}

	if _, err := q.LinkGuardianToStudent(ctx, db.LinkGuardianToStudentParams{
		StudentID:    studentID,
		GuardianID:   in.GuardianID,
		Relationship: in.Relationship,
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return nil, ErrGuardianAlreadyLinked
		}
		return nil, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionGuardianLinked,
		ResourceType: audit.ResourceStudent,
		ResourceID:   studentID,
		Metadata:     map[string]any{"guardian_id": in.GuardianID.String(), "relationship": in.Relationship},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.queries.ListGuardiansForStudent(ctx, studentID)
}

// UnlinkGuardianFromStudent removes the association (not either record) —
// a guardian may still be linked to other students afterward.
func (s *Service) UnlinkGuardianFromStudent(ctx context.Context, actorID, studentID, guardianID uuid.UUID) ([]db.ListGuardiansForStudentRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	rows, err := q.UnlinkGuardianFromStudent(ctx, db.UnlinkGuardianFromStudentParams{
		StudentID:  studentID,
		GuardianID: guardianID,
	})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrGuardianLinkNotFound
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionGuardianUnlinked,
		ResourceType: audit.ResourceStudent,
		ResourceID:   studentID,
		Metadata:     map[string]any{"guardian_id": guardianID.String()},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.queries.ListGuardiansForStudent(ctx, studentID)
}

// --- guardian login accounts (expert-pass access-control follow-up) ---
//
// The parent-side mirror of internal/staff's CreateAccount: an admin
// provisions a PARENT login for a guardian record, the guardian receives
// a welcome email plus (when Clerk is configured) an official invitation
// to create their identity with that same email, and the existing bridge
// lets them into the parent portal on first sign-in.

// GuardianAccountInput mirrors staff.CreateAccountInput. Password policy
// is the handler's concern (binding tags); the service treats it as
// opaque. RoleName must be PARENT (this endpoint must not be able to
// mint admins or teachers — role changes for other audiences go through
// /staff/:id/account and RBAC).
type GuardianAccountInput struct {
	Email    string
	Password string
	RoleName string // always PARENT today; kept explicit
}

// CreateGuardianAccount creates the guardian's user account (Argon2id
// hash, PARENT role, audit entry) in one transaction, then fires the
// best-effort welcome email + Clerk invitation after commit.
func (s *Service) CreateGuardianAccount(ctx context.Context, actorID, guardianID uuid.UUID, in GuardianAccountInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	guardian, err := q.GetGuardianByID(ctx, guardianID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGuardianNotFound
		}
		return err
	}

	if _, err := q.GetUserByPersonID(ctx, guardian.PersonID); err == nil {
		return ErrAccountExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if _, err := q.GetUserByEmail(ctx, in.Email); err == nil {
		return ErrEmailTaken
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}

	roleName := in.RoleName
	if roleName == "" {
		roleName = "PARENT"
	}
	if roleName != "PARENT" {
		return ErrUnknownRole
	}
	role, err := q.GetRoleByName(ctx, roleName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnknownRole
		}
		return err
	}

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		PersonID:     guardian.PersonID,
		Email:        in.Email,
		PasswordHash: hash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			// Raced with another admin creating the same email.
			return ErrEmailTaken
		}
		return err
	}
	// M34: record who provisioned this parent account (inviter tracking).
	if err := q.SetUserInvitedBy(ctx, db.SetUserInvitedByParams{
		ID:        user.ID,
		InvitedBy: pgtype.UUID{Bytes: actorID, Valid: true},
	}); err != nil {
		return err
	}

	if err := q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: user.ID, RoleID: role.ID}); err != nil {
		return err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionUserCreated,
		ResourceType: audit.ResourceUser,
		ResourceID:   user.ID,
		Metadata:     map[string]any{"email": in.Email, "roles": []string{roleName}, "guardian_id": guardianID.String()},
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// The person row may not carry the email; the account email is the
	// authoritative address for the invitation.
	accountEmail := in.Email
	name := guardian.FirstName + " " + guardian.LastName

	// Best-effort Clerk invitation — the recipient gets the official
	// "create your password" email and lands on /register; the bridge
	// matches them by this same email on first sign-in.
	if s.inviter != nil && s.inviter.Enabled() {
		go func(addr string) {
			ictx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			s.inviter.Create(ictx, addr)
		}(accountEmail)
	}

	// Best-effort welcome email (same posture as the staff flow).
	if svc, ok := s.mailer.(*email.Service); ok && svc != nil && svc.Enabled() {
		to := accountEmail
		go func() {
			ectx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			signIn := s.frontendURL
			if signIn == "" {
				signIn = "http://localhost:3000/login"
			}
			_ = svc.SendWelcome(ectx, to, name, signIn)
		}()
	}

	return nil
}
