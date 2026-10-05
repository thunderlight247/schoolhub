package academics

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
)

// Postgres error code for unique violation.
const pgErrUniqueViolation = "23505"

var (
	ErrAcademicYearNotFound  = errors.New("academic year not found")
	ErrAcademicYearNameTaken = errors.New("academic year name already in use")
	ErrInvalidDateRange      = errors.New("end date must be after or equal to start date")

	ErrTermNotFound  = errors.New("term not found")
	ErrTermNameTaken = errors.New("term name already exists in this academic year")

	ErrProgrammeNotFound  = errors.New("programme not found")
	ErrProgrammeCodeTaken = errors.New("programme code already in use")
	ErrProgrammeNameTaken = errors.New("programme name already in use")

	ErrSubjectNotFound      = errors.New("subject not found")
	ErrSubjectCodeTaken     = errors.New("subject code already in use")
	ErrInvalidSubjectType   = errors.New("subject type must be CORE or ELECTIVE")

	ErrClassNotFound        = errors.New("class not found")
	ErrClassNameTaken       = errors.New("class name already exists in this academic year")
	ErrInvalidCapacity      = errors.New("capacity must be a positive integer")
	ErrSubjectAlreadyAssigned = errors.New("subject already assigned to this class")
	ErrSubjectNotAssigned   = errors.New("subject is not assigned to this class")
)

type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: db.New(pool),
	}
}

// --- Helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

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

func toPgUUID(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func toPgInt4(i *int32) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *i, Valid: true}
}

// --- Academic Years ---

type CreateAcademicYearInput struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	IsCurrent bool
}

type UpdateAcademicYearInput struct {
	Name      *string
	StartDate *time.Time
	EndDate   *time.Time
}

func (s *Service) CreateAcademicYear(ctx context.Context, actorID uuid.UUID, in CreateAcademicYearInput) (db.AcademicYear, error) {
	if in.EndDate.Before(in.StartDate) {
		return db.AcademicYear{}, ErrInvalidDateRange
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.AcademicYear{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if in.IsCurrent {
		if err := q.UnsetCurrentAcademicYears(ctx); err != nil {
			return db.AcademicYear{}, err
		}
	}

	year, err := q.CreateAcademicYear(ctx, db.CreateAcademicYearParams{
		Name:      strings.TrimSpace(in.Name),
		StartDate: pgtype.Date{Time: in.StartDate, Valid: true},
		EndDate:   pgtype.Date{Time: in.EndDate, Valid: true},
		IsCurrent: in.IsCurrent,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.AcademicYear{}, ErrAcademicYearNameTaken
		}
		return db.AcademicYear{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAcademicYearCreated,
		ResourceType: audit.ResourceAcademicYear,
		ResourceID:   year.ID,
		Metadata:     map[string]any{"name": year.Name, "is_current": year.IsCurrent},
	}); err != nil {
		return db.AcademicYear{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.AcademicYear{}, err
	}

	return year, nil
}

func (s *Service) GetAcademicYear(ctx context.Context, id uuid.UUID) (db.AcademicYear, error) {
	year, err := s.queries.GetAcademicYearByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AcademicYear{}, ErrAcademicYearNotFound
		}
		return db.AcademicYear{}, err
	}
	return year, nil
}

func (s *Service) GetCurrentAcademicYear(ctx context.Context) (db.AcademicYear, error) {
	year, err := s.queries.GetCurrentAcademicYear(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AcademicYear{}, ErrAcademicYearNotFound
		}
		return db.AcademicYear{}, err
	}
	return year, nil
}

func (s *Service) ListAcademicYears(ctx context.Context) ([]db.AcademicYear, error) {
	return s.queries.ListAcademicYears(ctx)
}

func (s *Service) UpdateAcademicYear(ctx context.Context, actorID, id uuid.UUID, in UpdateAcademicYearInput) (db.AcademicYear, error) {
	current, err := s.GetAcademicYear(ctx, id)
	if err != nil {
		return db.AcademicYear{}, err
	}

	startDate := current.StartDate.Time
	endDate := current.EndDate.Time
	if in.StartDate != nil {
		startDate = *in.StartDate
	}
	if in.EndDate != nil {
		endDate = *in.EndDate
	}
	if endDate.Before(startDate) {
		return db.AcademicYear{}, ErrInvalidDateRange
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.AcademicYear{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	params := db.UpdateAcademicYearParams{
		ID:        id,
		Name:      toPgText(in.Name),
		StartDate: toPgDate(in.StartDate),
		EndDate:   toPgDate(in.EndDate),
	}

	year, err := q.UpdateAcademicYear(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.AcademicYear{}, ErrAcademicYearNameTaken
		}
		return db.AcademicYear{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAcademicYearUpdated,
		ResourceType: audit.ResourceAcademicYear,
		ResourceID:   year.ID,
		Metadata:     map[string]any{"name": year.Name},
	}); err != nil {
		return db.AcademicYear{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.AcademicYear{}, err
	}

	return year, nil
}

func (s *Service) SetCurrentAcademicYear(ctx context.Context, actorID, id uuid.UUID) (db.AcademicYear, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.AcademicYear{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.GetAcademicYearByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AcademicYear{}, ErrAcademicYearNotFound
		}
		return db.AcademicYear{}, err
	}

	if err := q.UnsetCurrentAcademicYears(ctx); err != nil {
		return db.AcademicYear{}, err
	}

	year, err := q.SetAcademicYearCurrent(ctx, id)
	if err != nil {
		return db.AcademicYear{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAcademicYearSetCurrent,
		ResourceType: audit.ResourceAcademicYear,
		ResourceID:   year.ID,
		Metadata:     map[string]any{"is_current": true},
	}); err != nil {
		return db.AcademicYear{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.AcademicYear{}, err
	}

	return year, nil
}

// --- Terms ---

type CreateTermInput struct {
	AcademicYearID uuid.UUID
	Name           string
	StartDate      time.Time
	EndDate        time.Time
	IsCurrent      bool
}

type UpdateTermInput struct {
	Name      *string
	StartDate *time.Time
	EndDate   *time.Time
}

func (s *Service) CreateTerm(ctx context.Context, actorID uuid.UUID, in CreateTermInput) (db.Term, error) {
	if in.EndDate.Before(in.StartDate) {
		return db.Term{}, ErrInvalidDateRange
	}

	// Verify academic year exists
	if _, err := s.queries.GetAcademicYearByID(ctx, in.AcademicYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Term{}, ErrAcademicYearNotFound
		}
		return db.Term{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Term{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if in.IsCurrent {
		if err := q.UnsetCurrentTerms(ctx); err != nil {
			return db.Term{}, err
		}
	}

	term, err := q.CreateTerm(ctx, db.CreateTermParams{
		AcademicYearID: in.AcademicYearID,
		Name:           strings.TrimSpace(in.Name),
		StartDate:      pgtype.Date{Time: in.StartDate, Valid: true},
		EndDate:        pgtype.Date{Time: in.EndDate, Valid: true},
		IsCurrent:      in.IsCurrent,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Term{}, ErrTermNameTaken
		}
		return db.Term{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionTermCreated,
		ResourceType: audit.ResourceTerm,
		ResourceID:   term.ID,
		Metadata:     map[string]any{"name": term.Name, "academic_year_id": term.AcademicYearID, "is_current": term.IsCurrent},
	}); err != nil {
		return db.Term{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Term{}, err
	}

	return term, nil
}

func (s *Service) GetTerm(ctx context.Context, id uuid.UUID) (db.GetTermByIDRow, error) {
	term, err := s.queries.GetTermByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTermByIDRow{}, ErrTermNotFound
		}
		return db.GetTermByIDRow{}, err
	}
	return term, nil
}

func (s *Service) GetCurrentTerm(ctx context.Context) (db.GetCurrentTermRow, error) {
	term, err := s.queries.GetCurrentTerm(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetCurrentTermRow{}, ErrTermNotFound
		}
		return db.GetCurrentTermRow{}, err
	}
	return term, nil
}

func (s *Service) ListTerms(ctx context.Context, academicYearID *uuid.UUID) ([]db.ListAllTermsRow, error) {
	if academicYearID != nil {
		rows, err := s.queries.ListTermsForAcademicYear(ctx, *academicYearID)
		if err != nil {
			return nil, err
		}
		all := make([]db.ListAllTermsRow, 0, len(rows))
		for _, r := range rows {
			all = append(all, db.ListAllTermsRow(r))
		}
		return all, nil
	}
	return s.queries.ListAllTerms(ctx)
}

func (s *Service) UpdateTerm(ctx context.Context, actorID, id uuid.UUID, in UpdateTermInput) (db.Term, error) {
	current, err := s.GetTerm(ctx, id)
	if err != nil {
		return db.Term{}, err
	}

	startDate := current.StartDate.Time
	endDate := current.EndDate.Time
	if in.StartDate != nil {
		startDate = *in.StartDate
	}
	if in.EndDate != nil {
		endDate = *in.EndDate
	}
	if endDate.Before(startDate) {
		return db.Term{}, ErrInvalidDateRange
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Term{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	term, err := q.UpdateTerm(ctx, db.UpdateTermParams{
		ID:        id,
		Name:      toPgText(in.Name),
		StartDate: toPgDate(in.StartDate),
		EndDate:   toPgDate(in.EndDate),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Term{}, ErrTermNameTaken
		}
		return db.Term{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionTermUpdated,
		ResourceType: audit.ResourceTerm,
		ResourceID:   term.ID,
		Metadata:     map[string]any{"name": term.Name},
	}); err != nil {
		return db.Term{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Term{}, err
	}

	return term, nil
}

func (s *Service) SetCurrentTerm(ctx context.Context, actorID, id uuid.UUID) (db.Term, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Term{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.GetTermByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Term{}, ErrTermNotFound
		}
		return db.Term{}, err
	}

	if err := q.UnsetCurrentTerms(ctx); err != nil {
		return db.Term{}, err
	}

	term, err := q.SetTermCurrent(ctx, id)
	if err != nil {
		return db.Term{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionTermSetCurrent,
		ResourceType: audit.ResourceTerm,
		ResourceID:   term.ID,
		Metadata:     map[string]any{"is_current": true},
	}); err != nil {
		return db.Term{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Term{}, err
	}

	return term, nil
}

// --- Programmes ---

type CreateProgrammeInput struct {
	Code        string
	Name        string
	Description *string
}

type UpdateProgrammeInput struct {
	Code        *string
	Name        *string
	Description *string
}

func (s *Service) CreateProgramme(ctx context.Context, actorID uuid.UUID, in CreateProgrammeInput) (db.Programme, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Programme{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	prog, err := q.CreateProgramme(ctx, db.CreateProgrammeParams{
		Code:        strings.ToUpper(strings.TrimSpace(in.Code)),
		Name:        strings.TrimSpace(in.Name),
		Description: toPgText(in.Description),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			if strings.Contains(pgErr.ConstraintName, "code") {
				return db.Programme{}, ErrProgrammeCodeTaken
			}
			return db.Programme{}, ErrProgrammeNameTaken
		}
		return db.Programme{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionProgrammeCreated,
		ResourceType: audit.ResourceProgramme,
		ResourceID:   prog.ID,
		Metadata:     map[string]any{"code": prog.Code, "name": prog.Name},
	}); err != nil {
		return db.Programme{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Programme{}, err
	}

	return prog, nil
}

func (s *Service) GetProgramme(ctx context.Context, id uuid.UUID) (db.Programme, error) {
	prog, err := s.queries.GetProgrammeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Programme{}, ErrProgrammeNotFound
		}
		return db.Programme{}, err
	}
	return prog, nil
}

func (s *Service) ListProgrammes(ctx context.Context) ([]db.Programme, error) {
	return s.queries.ListProgrammes(ctx)
}

func (s *Service) UpdateProgramme(ctx context.Context, actorID, id uuid.UUID, in UpdateProgrammeInput) (db.Programme, error) {
	if _, err := s.GetProgramme(ctx, id); err != nil {
		return db.Programme{}, err
	}

	var codePtr *string
	if in.Code != nil {
		c := strings.ToUpper(strings.TrimSpace(*in.Code))
		codePtr = &c
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Programme{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	prog, err := q.UpdateProgramme(ctx, db.UpdateProgrammeParams{
		ID:          id,
		Code:        toPgText(codePtr),
		Name:        toPgText(in.Name),
		Description: toPgText(in.Description),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			if strings.Contains(pgErr.ConstraintName, "code") {
				return db.Programme{}, ErrProgrammeCodeTaken
			}
			return db.Programme{}, ErrProgrammeNameTaken
		}
		return db.Programme{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionProgrammeUpdated,
		ResourceType: audit.ResourceProgramme,
		ResourceID:   prog.ID,
		Metadata:     map[string]any{"code": prog.Code, "name": prog.Name},
	}); err != nil {
		return db.Programme{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Programme{}, err
	}

	return prog, nil
}

// --- Subjects ---

type CreateSubjectInput struct {
	Code         string
	Name         string
	Description  *string
	Type         string
	DepartmentID *uuid.UUID
}

type UpdateSubjectInput struct {
	Code         *string
	Name         *string
	Description  *string
	Type         *string
	DepartmentID *uuid.UUID
}

type SubjectListFilter struct {
	Type         string
	DepartmentID *uuid.UUID
	Search       string
}

func (s *Service) CreateSubject(ctx context.Context, actorID uuid.UUID, in CreateSubjectInput) (db.Subject, error) {
	subType := strings.ToUpper(strings.TrimSpace(in.Type))
	if subType == "" {
		subType = "CORE"
	}
	if subType != "CORE" && subType != "ELECTIVE" {
		return db.Subject{}, ErrInvalidSubjectType
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Subject{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	subject, err := q.CreateSubject(ctx, db.CreateSubjectParams{
		Code:         strings.ToUpper(strings.TrimSpace(in.Code)),
		Name:         strings.TrimSpace(in.Name),
		Description:  toPgText(in.Description),
		Type:         subType,
		DepartmentID: toPgUUID(in.DepartmentID),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Subject{}, ErrSubjectCodeTaken
		}
		return db.Subject{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionSubjectCreated,
		ResourceType: audit.ResourceSubject,
		ResourceID:   subject.ID,
		Metadata:     map[string]any{"code": subject.Code, "name": subject.Name, "type": subject.Type},
	}); err != nil {
		return db.Subject{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Subject{}, err
	}

	return subject, nil
}

func (s *Service) GetSubject(ctx context.Context, id uuid.UUID) (db.GetSubjectByIDRow, error) {
	subject, err := s.queries.GetSubjectByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetSubjectByIDRow{}, ErrSubjectNotFound
		}
		return db.GetSubjectByIDRow{}, err
	}
	return subject, nil
}

func (s *Service) ListSubjects(ctx context.Context, f SubjectListFilter) ([]db.ListSubjectsRow, error) {
	return s.queries.ListSubjects(ctx, db.ListSubjectsParams{
		Type:         filterText(f.Type),
		DepartmentID: toPgUUID(f.DepartmentID),
		Search:       filterText(f.Search),
	})
}

func (s *Service) UpdateSubject(ctx context.Context, actorID, id uuid.UUID, in UpdateSubjectInput) (db.Subject, error) {
	if _, err := s.GetSubject(ctx, id); err != nil {
		return db.Subject{}, err
	}

	var codePtr *string
	if in.Code != nil {
		c := strings.ToUpper(strings.TrimSpace(*in.Code))
		codePtr = &c
	}

	if in.Type != nil {
		t := strings.ToUpper(strings.TrimSpace(*in.Type))
		if t != "CORE" && t != "ELECTIVE" {
			return db.Subject{}, ErrInvalidSubjectType
		}
		in.Type = &t
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Subject{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	subject, err := q.UpdateSubject(ctx, db.UpdateSubjectParams{
		ID:           id,
		Code:         toPgText(codePtr),
		Name:         toPgText(in.Name),
		Description:  toPgText(in.Description),
		Type:         toPgText(in.Type),
		DepartmentID: toPgUUID(in.DepartmentID),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Subject{}, ErrSubjectCodeTaken
		}
		return db.Subject{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionSubjectUpdated,
		ResourceType: audit.ResourceSubject,
		ResourceID:   subject.ID,
		Metadata:     map[string]any{"code": subject.Code, "name": subject.Name},
	}); err != nil {
		return db.Subject{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Subject{}, err
	}

	return subject, nil
}

// --- Classes ---

type CreateClassInput struct {
	AcademicYearID uuid.UUID
	ProgrammeID    *uuid.UUID
	Name           string
	Level          *string
	Capacity       *int32
}

type UpdateClassInput struct {
	ProgrammeID *uuid.UUID
	Name        *string
	Level       *string
	Capacity    *int32
}

type ClassListFilter struct {
	AcademicYearID *uuid.UUID
	ProgrammeID    *uuid.UUID
	Level          string
}

func (s *Service) CreateClass(ctx context.Context, actorID uuid.UUID, in CreateClassInput) (db.Class, error) {
	if in.Capacity != nil && *in.Capacity <= 0 {
		return db.Class{}, ErrInvalidCapacity
	}

	// Verify academic year exists
	if _, err := s.queries.GetAcademicYearByID(ctx, in.AcademicYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Class{}, ErrAcademicYearNotFound
		}
		return db.Class{}, err
	}

	// Verify programme if provided
	if in.ProgrammeID != nil {
		if _, err := s.queries.GetProgrammeByID(ctx, *in.ProgrammeID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.Class{}, ErrProgrammeNotFound
			}
			return db.Class{}, err
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Class{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	classRow, err := q.CreateClass(ctx, db.CreateClassParams{
		AcademicYearID: in.AcademicYearID,
		ProgrammeID:    toPgUUID(in.ProgrammeID),
		Name:           strings.TrimSpace(in.Name),
		Level:          toPgText(in.Level),
		Capacity:       toPgInt4(in.Capacity),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Class{}, ErrClassNameTaken
		}
		return db.Class{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionClassCreated,
		ResourceType: audit.ResourceClass,
		ResourceID:   classRow.ID,
		Metadata:     map[string]any{"name": classRow.Name, "academic_year_id": classRow.AcademicYearID},
	}); err != nil {
		return db.Class{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Class{}, err
	}

	return classRow, nil
}

func (s *Service) GetClass(ctx context.Context, id uuid.UUID) (db.GetClassByIDRow, error) {
	classRow, err := s.queries.GetClassByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetClassByIDRow{}, ErrClassNotFound
		}
		return db.GetClassByIDRow{}, err
	}
	return classRow, nil
}

func (s *Service) ListClasses(ctx context.Context, f ClassListFilter) ([]db.ListClassesRow, error) {
	return s.queries.ListClasses(ctx, db.ListClassesParams{
		AcademicYearID: toPgUUID(f.AcademicYearID),
		ProgrammeID:    toPgUUID(f.ProgrammeID),
		Level:          filterText(f.Level),
	})
}

func (s *Service) UpdateClass(ctx context.Context, actorID, id uuid.UUID, in UpdateClassInput) (db.Class, error) {
	if in.Capacity != nil && *in.Capacity <= 0 {
		return db.Class{}, ErrInvalidCapacity
	}

	if _, err := s.GetClass(ctx, id); err != nil {
		return db.Class{}, err
	}

	if in.ProgrammeID != nil {
		if _, err := s.queries.GetProgrammeByID(ctx, *in.ProgrammeID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.Class{}, ErrProgrammeNotFound
			}
			return db.Class{}, err
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Class{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	classRow, err := q.UpdateClass(ctx, db.UpdateClassParams{
		ID:          id,
		ProgrammeID: toPgUUID(in.ProgrammeID),
		Name:        toPgText(in.Name),
		Level:       toPgText(in.Level),
		Capacity:    toPgInt4(in.Capacity),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Class{}, ErrClassNameTaken
		}
		return db.Class{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionClassUpdated,
		ResourceType: audit.ResourceClass,
		ResourceID:   classRow.ID,
		Metadata:     map[string]any{"name": classRow.Name},
	}); err != nil {
		return db.Class{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Class{}, err
	}

	return classRow, nil
}

func (s *Service) AssignSubjectToClass(ctx context.Context, actorID, classID, subjectID uuid.UUID) (db.ClassSubject, error) {
	// Verify class and subject exist
	if _, err := s.queries.GetClassByID(ctx, classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ClassSubject{}, ErrClassNotFound
		}
		return db.ClassSubject{}, err
	}
	if _, err := s.queries.GetSubjectByID(ctx, subjectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ClassSubject{}, ErrSubjectNotFound
		}
		return db.ClassSubject{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.ClassSubject{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	cs, err := q.AssignSubjectToClass(ctx, db.AssignSubjectToClassParams{
		ClassID:   classID,
		SubjectID: subjectID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.ClassSubject{}, ErrSubjectAlreadyAssigned
		}
		return db.ClassSubject{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionClassSubjectAssigned,
		ResourceType: audit.ResourceClass,
		ResourceID:   classID,
		Metadata:     map[string]any{"subject_id": subjectID},
	}); err != nil {
		return db.ClassSubject{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.ClassSubject{}, err
	}

	return cs, nil
}

func (s *Service) RemoveSubjectFromClass(ctx context.Context, actorID, classID, subjectID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	n, err := q.RemoveSubjectFromClass(ctx, db.RemoveSubjectFromClassParams{
		ClassID:   classID,
		SubjectID: subjectID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSubjectNotAssigned
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionClassSubjectRemoved,
		ResourceType: audit.ResourceClass,
		ResourceID:   classID,
		Metadata:     map[string]any{"subject_id": subjectID},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) ListSubjectsForClass(ctx context.Context, classID uuid.UUID) ([]db.ListSubjectsForClassRow, error) {
	if _, err := s.queries.GetClassByID(ctx, classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrClassNotFound
		}
		return nil, err
	}
	return s.queries.ListSubjectsForClass(ctx, classID)
}
