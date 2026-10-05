// Package assignments implements the teacher assignment domain (M10):
// which teacher teaches which subject in which class. An assignment is a
// thin mapping over records owned by other domains — staff (M6), classes
// and subjects (M8) — so creation validates the whole chain before
// inserting, and every mutation writes an audit row in the same
// transaction.
package assignments

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
)

const pgErrUniqueViolation = "23505"

var (
	ErrAssignmentNotFound  = errors.New("teacher assignment not found")
	ErrTeacherNotFound     = errors.New("teacher not found")
	ErrClassNotFound       = errors.New("class not found")
	ErrSubjectNotFound     = errors.New("subject not found")
	ErrSubjectNotInClass   = errors.New("subject is not assigned to this class")
	ErrDuplicateAssignment = errors.New("teacher already has an assignment for this class and subject")
)

// ListFilter carries the optional list filters. Each nil pointer disables
// its condition at the SQL layer; pagination is normalized before it
// reaches the queries.
type ListFilter struct {
	TeacherID      *uuid.UUID
	ClassID        *uuid.UUID
	SubjectID      *uuid.UUID
	AcademicYearID *uuid.UUID
	Search         string
	Page           int
	PageSize       int
}

type CreateTeacherAssignmentInput struct {
	// TeacherID is the staff record (M6) of the teacher being assigned.
	TeacherID uuid.UUID `json:"teacher_id"`
	ClassID   uuid.UUID `json:"class_id"`
	SubjectID uuid.UUID `json:"subject_id"`
}

type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

func NormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func toPgUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func filterText(s string) pgtype.Text {
	if strings.TrimSpace(s) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(s), Valid: true}
}

// CreateTeacherAssignment validates the teacher/class/subject chain, then
// inserts the assignment and its audit entry in one transaction. The
// subject must already be part of the class's curriculum (class_subjects,
// M8) — an assignment covers teaching, it never widens the curriculum.
func (s *Service) CreateTeacherAssignment(ctx context.Context, actorID uuid.UUID, in CreateTeacherAssignmentInput) (db.GetTeacherAssignmentByIDRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetTeacherAssignmentByIDRow{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.GetStaffByID(ctx, in.TeacherID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTeacherAssignmentByIDRow{}, ErrTeacherNotFound
		}
		return db.GetTeacherAssignmentByIDRow{}, err
	}
	if _, err := q.GetClassByID(ctx, in.ClassID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTeacherAssignmentByIDRow{}, ErrClassNotFound
		}
		return db.GetTeacherAssignmentByIDRow{}, err
	}
	if _, err := q.GetSubjectByID(ctx, in.SubjectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTeacherAssignmentByIDRow{}, ErrSubjectNotFound
		}
		return db.GetTeacherAssignmentByIDRow{}, err
	}

	hasPairing, err := q.HasClassSubject(ctx, db.HasClassSubjectParams{
		ClassID:   in.ClassID,
		SubjectID: in.SubjectID,
	})
	if err != nil {
		return db.GetTeacherAssignmentByIDRow{}, err
	}
	if !hasPairing {
		return db.GetTeacherAssignmentByIDRow{}, ErrSubjectNotInClass
	}

	created, err := q.CreateTeacherAssignment(ctx, db.CreateTeacherAssignmentParams{
		TeacherID: in.TeacherID,
		ClassID:   in.ClassID,
		SubjectID: in.SubjectID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.GetTeacherAssignmentByIDRow{}, ErrDuplicateAssignment
		}
		return db.GetTeacherAssignmentByIDRow{}, err
	}

	// Re-fetch through the detail join so the caller gets the same enriched
	// shape as GetTeacherAssignment without a second round trip after commit.
	row, err := q.GetTeacherAssignmentByID(ctx, created.ID)
	if err != nil {
		return db.GetTeacherAssignmentByIDRow{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionTeacherAssignmentCreated,
		ResourceType: audit.ResourceTeacherAssignment,
		ResourceID:   row.ID,
		Metadata: map[string]any{
			"teacher_id": row.TeacherID.String(),
			"class_id":   row.ClassID.String(),
			"subject_id": row.SubjectID.String(),
		},
	}); err != nil {
		return db.GetTeacherAssignmentByIDRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetTeacherAssignmentByIDRow{}, err
	}
	return row, nil
}

func (s *Service) GetTeacherAssignment(ctx context.Context, id uuid.UUID) (db.GetTeacherAssignmentByIDRow, error) {
	row, err := s.queries.GetTeacherAssignmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTeacherAssignmentByIDRow{}, ErrAssignmentNotFound
		}
		return db.GetTeacherAssignmentByIDRow{}, err
	}
	return row, nil
}

func (s *Service) ListTeacherAssignments(ctx context.Context, f ListFilter) ([]db.ListTeacherAssignmentsRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	params := db.ListTeacherAssignmentsParams{
		TeacherID:      toPgUUIDPtr(f.TeacherID),
		ClassID:        toPgUUIDPtr(f.ClassID),
		SubjectID:      toPgUUIDPtr(f.SubjectID),
		AcademicYearID: toPgUUIDPtr(f.AcademicYearID),
		Search:         filterText(f.Search),
		PageLimit:      int32(pageSize),
		PageOffset:     int32((page - 1) * pageSize),
	}
	rows, err := s.queries.ListTeacherAssignments(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.queries.CountTeacherAssignments(ctx, db.CountTeacherAssignmentsParams{
		TeacherID:      params.TeacherID,
		ClassID:        params.ClassID,
		SubjectID:      params.SubjectID,
		AcademicYearID: params.AcademicYearID,
		Search:         params.Search,
	})
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// DeleteTeacherAssignment removes the mapping and records the removal with
// enough metadata to reconstruct what was deleted, since the row itself is
// gone afterwards.
func (s *Service) DeleteTeacherAssignment(ctx context.Context, actorID, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetTeacherAssignmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssignmentNotFound
		}
		return err
	}

	if err := q.DeleteTeacherAssignment(ctx, id); err != nil {
		return err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionTeacherAssignmentRemoved,
		ResourceType: audit.ResourceTeacherAssignment,
		ResourceID:   current.ID,
		Metadata: map[string]any{
			"teacher_id": current.TeacherID.String(),
			"class_id":   current.ClassID.String(),
			"subject_id": current.SubjectID.String(),
		},
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
