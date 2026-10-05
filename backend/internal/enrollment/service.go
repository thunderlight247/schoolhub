package enrollment

import (
	"context"
	"encoding/json"
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

const pgErrUniqueViolation = "23505"

const (
	StatusEnrolled    = "ENROLLED"
	StatusCompleted   = "COMPLETED"
	StatusTransferred = "TRANSFERRED"
	StatusWithdrawn   = "WITHDRAWN"
)

var EnrollmentStatuses = []string{StatusEnrolled, StatusCompleted, StatusTransferred, StatusWithdrawn}

var (
	ErrEnrollmentNotFound        = errors.New("enrollment not found")
	ErrStudentNotFound           = errors.New("student not found")
	ErrAcademicYearNotFound      = errors.New("academic year not found")
	ErrClassNotFound             = errors.New("class not found")
	ErrClassYearMismatch         = errors.New("class does not belong to the selected academic year")
	ErrInvalidEnrollmentStatus   = errors.New("invalid enrollment status")
	ErrDuplicateActiveEnrollment = errors.New("student already has an active enrollment in this academic year")
)

type ListFilter struct {
	StudentID      *uuid.UUID
	AcademicYearID *uuid.UUID
	ClassID        *uuid.UUID
	Status         string
	Search         string
	Page           int
	PageSize       int
}

type CreateEnrollmentInput struct {
	StudentID      uuid.UUID
	AcademicYearID uuid.UUID
	ClassID        uuid.UUID
	Status         string
	EnrollmentDate time.Time
	Notes          *string
}

func (c *CreateEnrollmentInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		StudentID      string  `json:"student_id"`
		AcademicYearID string  `json:"academic_year_id"`
		ClassID        string  `json:"class_id"`
		Status         string  `json:"status"`
		EnrollmentDate string  `json:"enrollment_date"`
		Notes          *string `json:"notes"`
	}
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.StudentID != "" {
		id, err := uuid.Parse(aux.StudentID)
		if err != nil {
			return err
		}
		c.StudentID = id
	}
	if aux.AcademicYearID != "" {
		id, err := uuid.Parse(aux.AcademicYearID)
		if err != nil {
			return err
		}
		c.AcademicYearID = id
	}
	if aux.ClassID != "" {
		id, err := uuid.Parse(aux.ClassID)
		if err != nil {
			return err
		}
		c.ClassID = id
	}
	if aux.Status == "" {
		c.Status = StatusEnrolled
	} else {
		c.Status = aux.Status
	}
	if aux.Notes != nil {
		c.Notes = aux.Notes
	}
	if aux.EnrollmentDate != "" {
		parsed, err := time.Parse("2006-01-02", aux.EnrollmentDate)
		if err != nil {
			return err
		}
		c.EnrollmentDate = parsed
	} else {
		c.EnrollmentDate = time.Now()
	}
	return nil
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

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func filterText(s string) pgtype.Text {
	if strings.TrimSpace(s) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(s), Valid: true}
}

func normalizeEnrollmentStatus(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	if status == "" {
		return StatusEnrolled, nil
	}
	for _, allowed := range EnrollmentStatuses {
		if status == allowed {
			return status, nil
		}
	}
	return "", ErrInvalidEnrollmentStatus
}

func (s *Service) CreateEnrollment(ctx context.Context, actorID uuid.UUID, in CreateEnrollmentInput) (db.Enrollment, error) {
	status, err := normalizeEnrollmentStatus(in.Status)
	if err != nil {
		return db.Enrollment{}, err
	}
	if in.EnrollmentDate.IsZero() {
		in.EnrollmentDate = time.Now()
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Enrollment{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	if _, err := q.GetStudentByID(ctx, in.StudentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Enrollment{}, ErrStudentNotFound
		}
		return db.Enrollment{}, err
	}
	if _, err := q.GetAcademicYearByID(ctx, in.AcademicYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Enrollment{}, ErrAcademicYearNotFound
		}
		return db.Enrollment{}, err
	}
	classRow, err := q.GetClassByID(ctx, in.ClassID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Enrollment{}, ErrClassNotFound
		}
		return db.Enrollment{}, err
	}
	if classRow.AcademicYearID != in.AcademicYearID {
		return db.Enrollment{}, ErrClassYearMismatch
	}

	hasActive, err := q.HasActiveEnrollmentForStudentYear(ctx, db.HasActiveEnrollmentForStudentYearParams{
		StudentID:      in.StudentID,
		AcademicYearID: in.AcademicYearID,
	})
	if err != nil {
		return db.Enrollment{}, err
	}
	if hasActive {
		return db.Enrollment{}, ErrDuplicateActiveEnrollment
	}

	row, err := q.CreateEnrollment(ctx, db.CreateEnrollmentParams{
		StudentID:      in.StudentID,
		AcademicYearID: in.AcademicYearID,
		ClassID:        in.ClassID,
		Status:         status,
		EnrollmentDate: pgtype.Date{Time: in.EnrollmentDate, Valid: true},
		Notes:          toPgText(in.Notes),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.Enrollment{}, ErrDuplicateActiveEnrollment
		}
		return db.Enrollment{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionEnrollmentCreated,
		ResourceType: audit.ResourceEnrollment,
		ResourceID:   row.ID,
		Metadata:     map[string]any{"student_id": row.StudentID.String(), "class_id": row.ClassID.String(), "status": row.Status},
	}); err != nil {
		return db.Enrollment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Enrollment{}, err
	}
	return row, nil
}

func (s *Service) GetEnrollment(ctx context.Context, id uuid.UUID) (db.GetEnrollmentByIDRow, error) {
	row, err := s.queries.GetEnrollmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetEnrollmentByIDRow{}, ErrEnrollmentNotFound
		}
		return db.GetEnrollmentByIDRow{}, err
	}
	return row, nil
}

func (s *Service) ListEnrollments(ctx context.Context, f ListFilter) ([]db.ListEnrollmentsRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)
	status := ""
	var err error
	if f.Status != "" {
		status, err = normalizeEnrollmentStatus(f.Status)
		if err != nil {
			return nil, 0, err
		}
	}

	params := db.ListEnrollmentsParams{
		StudentID:      toPgUUIDPtr(f.StudentID),
		AcademicYearID: toPgUUIDPtr(f.AcademicYearID),
		ClassID:        toPgUUIDPtr(f.ClassID),
		Status:         filterText(status),
		Search:         filterText(f.Search),
		PageLimit:      int32(pageSize),
		PageOffset:     int32((page - 1) * pageSize),
	}
	rows, err := s.queries.ListEnrollments(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.queries.CountEnrollments(ctx, db.CountEnrollmentsParams{
		StudentID:      params.StudentID,
		AcademicYearID: params.AcademicYearID,
		ClassID:        params.ClassID,
		Status:         params.Status,
		Search:         params.Search,
	})
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Service) UpdateEnrollmentStatus(ctx context.Context, actorID, id uuid.UUID, status string) (db.GetEnrollmentByIDRow, error) {
	status, err := normalizeEnrollmentStatus(status)
	if err != nil {
		return db.GetEnrollmentByIDRow{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetEnrollmentByIDRow{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetEnrollmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetEnrollmentByIDRow{}, ErrEnrollmentNotFound
		}
		return db.GetEnrollmentByIDRow{}, err
	}
	if current.Status == status {
		return current, nil
	}

	if _, err := q.UpdateEnrollmentStatus(ctx, db.UpdateEnrollmentStatusParams{ID: id, Status: status}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.GetEnrollmentByIDRow{}, ErrDuplicateActiveEnrollment
		}
		return db.GetEnrollmentByIDRow{}, err
	}
	row, err := q.GetEnrollmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetEnrollmentByIDRow{}, ErrEnrollmentNotFound
		}
		return db.GetEnrollmentByIDRow{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionEnrollmentStatusUpdated,
		ResourceType: audit.ResourceEnrollment,
		ResourceID:   row.ID,
		Metadata:     map[string]any{"previous_status": current.Status, "new_status": row.Status},
	}); err != nil {
		return db.GetEnrollmentByIDRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetEnrollmentByIDRow{}, err
	}
	return row, nil
}

func toPgUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func toDateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func toTsString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func textPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}
