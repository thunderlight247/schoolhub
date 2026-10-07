// Package attendance implements the attendance domain (M11): teacher-facing
// register capture per class/subject/term and the per-status summaries the
// parent/student portals (M16/M17) will read.
//
// The model has two levels (migration 000011): an attendance_sessions row is
// one register-taking event (class, subject, term, date); attendance_records
// hold each student's mark. Capture validates the whole chain before
// inserting — the subject must be in the class curriculum (M8), the term
// must belong to the class's academic year, the date must fall inside the
// term, and every marked student must be actively enrolled in the class
// (M9). Every mutation writes its audit row in the same transaction.
package attendance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Mark values for an attendance record (CHECK constraint in 000011).
const (
	StatusPresent = "PRESENT"
	StatusAbsent  = "ABSENT"
	StatusLate    = "LATE"
	StatusExcused = "EXCUSED"
)

var AttendanceStatuses = []string{StatusPresent, StatusAbsent, StatusLate, StatusExcused}

var (
	ErrSessionNotFound    = errors.New("attendance session not found")
	ErrRecordNotFound     = errors.New("attendance record not found")
	ErrClassNotFound      = errors.New("class not found")
	ErrSubjectNotFound    = errors.New("subject not found")
	ErrTermNotFound       = errors.New("term not found")
	ErrStudentNotFound    = errors.New("student not found")
	ErrSubjectNotInClass  = errors.New("subject is not assigned to this class")
	ErrTermYearMismatch   = errors.New("term does not belong to the class's academic year")
	ErrDateOutsideTerm    = errors.New("session date falls outside the term's start and end dates")
	ErrInvalidStatus      = errors.New("invalid attendance status")
	ErrDuplicateSession   = errors.New("attendance has already been captured for this class, subject, term, and date")
	ErrStudentNotEnrolled = errors.New("student is not enrolled in this class")
	ErrDuplicateStudent   = errors.New("a student appears more than once in the register")
	ErrNoRecords          = errors.New("at least one attendance record is required")
)

// ListFilter carries the optional session-list filters. Each nil pointer
// disables its condition at the SQL layer; pagination is normalized before
// it reaches the queries.
type ListFilter struct {
	ClassID   *uuid.UUID
	SubjectID *uuid.UUID
	TermID    *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
	Page      int
	PageSize  int
}

// RecordInput is one student's mark inside a capture payload.
type RecordInput struct {
	StudentID uuid.UUID
	Status    string
	Notes     *string
}

// CreateSessionInput is the whole-register capture payload. The custom
// unmarshal keeps the same JSON conventions as the enrollment domain
// (dates as YYYY-MM-DD, UUIDs as strings, optional notes as *string).
type CreateSessionInput struct {
	ClassID     uuid.UUID
	SubjectID   uuid.UUID
	TermID      uuid.UUID
	SessionDate time.Time
	Records     []RecordInput
}

func (c *CreateSessionInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		ClassID     string        `json:"class_id"`
		SubjectID   string        `json:"subject_id"`
		TermID      string        `json:"term_id"`
		SessionDate string        `json:"session_date"`
		Records     []RecordInput `json:"records"`
	}
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.ClassID != "" {
		id, err := uuid.Parse(aux.ClassID)
		if err != nil {
			return err
		}
		c.ClassID = id
	}
	if aux.SubjectID != "" {
		id, err := uuid.Parse(aux.SubjectID)
		if err != nil {
			return err
		}
		c.SubjectID = id
	}
	if aux.TermID != "" {
		id, err := uuid.Parse(aux.TermID)
		if err != nil {
			return err
		}
		c.TermID = id
	}
	if aux.SessionDate != "" {
		parsed, err := time.Parse("2006-01-02", aux.SessionDate)
		if err != nil {
			return err
		}
		c.SessionDate = parsed
	}
	c.Records = aux.Records
	return nil
}

// UnmarshalJSON for RecordInput mirrors the same conventions.
func (r *RecordInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		StudentID string  `json:"student_id"`
		Status    string  `json:"status"`
		Notes     *string `json:"notes"`
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
		r.StudentID = id
	}
	r.Status = aux.Status
	r.Notes = aux.Notes
	return nil
}

// Summary is the per-status count aggregation behind the student and class
// summary endpoints. Counts always contain all four statuses (zero-filled)
// so clients can render a stable shape.
type Summary struct {
	StudentID      *uuid.UUID
	ClassID        *uuid.UUID
	SubjectID      *uuid.UUID
	TermID         *uuid.UUID
	AcademicYearID *uuid.UUID
	Counts         map[string]int64
	TotalRecords   int64
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

func normalizeStatus(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	for _, allowed := range AttendanceStatuses {
		if status == allowed {
			return status, nil
		}
	}
	return "", ErrInvalidStatus
}

func toPgUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func toPgDatePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: toMidnightUTC(*t), Valid: true}
}

// toMidnightUTC collapses a timestamp to its calendar date in UTC so
// comparisons against DATE columns (term bounds, session_date) never flip
// on clock time or timezone.
func toMidnightUTC(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// CreateSession captures one whole register: it validates the
// class/subject/term chain, the term window, and every student's active
// enrollment, then inserts the session, all marks, and the audit entry in
// one transaction. A register either fully lands or not at all.
func (s *Service) CreateSession(ctx context.Context, actorID uuid.UUID, in CreateSessionInput) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
	if len(in.Records) == 0 {
		return db.GetAttendanceSessionByIDRow{}, nil, ErrNoRecords
	}

	statuses := make([]string, len(in.Records))
	for i, r := range in.Records {
		st, err := normalizeStatus(r.Status)
		if err != nil {
			return db.GetAttendanceSessionByIDRow{}, nil, err
		}
		statuses[i] = st
	}

	sessionDate := in.SessionDate
	if sessionDate.IsZero() {
		sessionDate = time.Now()
	}
	sessionDate = toMidnightUTC(sessionDate)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	classRow, err := q.GetClassByID(ctx, in.ClassID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrClassNotFound
		}
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	if _, err := q.GetSubjectByID(ctx, in.SubjectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrSubjectNotFound
		}
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	termRow, err := q.GetTermByID(ctx, in.TermID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrTermNotFound
		}
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}

	// The term must belong to the class's academic year — registers are
	// taken inside the year the class itself lives in (000008/000011).
	if termRow.AcademicYearID != classRow.AcademicYearID {
		return db.GetAttendanceSessionByIDRow{}, nil, ErrTermYearMismatch
	}

	// The subject must be part of the class curriculum (class_subjects,
	// M8) — same rule as teacher assignments (M10).
	hasPairing, err := q.HasClassSubject(ctx, db.HasClassSubjectParams{
		ClassID:   in.ClassID,
		SubjectID: in.SubjectID,
	})
	if err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	if !hasPairing {
		return db.GetAttendanceSessionByIDRow{}, nil, ErrSubjectNotInClass
	}

	if sessionDate.Before(termRow.StartDate.Time) || sessionDate.After(termRow.EndDate.Time) {
		return db.GetAttendanceSessionByIDRow{}, nil, ErrDateOutsideTerm
	}

	// One mark per student per register: reject duplicates in the payload
	// itself, then verify every marked student is actively enrolled (M9).
	// COMPLETED/TRANSFERRED memberships are historical — past marks stay
	// readable, but they earn no new ones.
	seen := make(map[uuid.UUID]struct{}, len(in.Records))
	for _, r := range in.Records {
		if _, dup := seen[r.StudentID]; dup {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrDuplicateStudent
		}
		seen[r.StudentID] = struct{}{}
	}
	for studentID := range seen {
		enrolled, err := q.HasActiveClassEnrollment(ctx, db.HasActiveClassEnrollmentParams{
			StudentID: studentID,
			ClassID:   in.ClassID,
		})
		if err != nil {
			return db.GetAttendanceSessionByIDRow{}, nil, err
		}
		if !enrolled {
			return db.GetAttendanceSessionByIDRow{}, nil, fmt.Errorf("%w: %s", ErrStudentNotEnrolled, studentID)
		}
	}

	created, err := q.CreateAttendanceSession(ctx, db.CreateAttendanceSessionParams{
		ClassID:     in.ClassID,
		SubjectID:   in.SubjectID,
		TermID:      in.TermID,
		SessionDate: pgtype.Date{Time: sessionDate, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrDuplicateSession
		}
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}

	statusCounts := make(map[string]int, len(AttendanceStatuses))
	for i, r := range in.Records {
		if _, err := q.CreateAttendanceRecord(ctx, db.CreateAttendanceRecordParams{
			SessionID: created.ID,
			StudentID: r.StudentID,
			Status:    statuses[i],
			Notes:     toPgText(r.Notes),
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
				return db.GetAttendanceSessionByIDRow{}, nil, ErrDuplicateStudent
			}
			return db.GetAttendanceSessionByIDRow{}, nil, err
		}
		statusCounts[statuses[i]]++
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAttendanceRecorded,
		ResourceType: audit.ResourceAttendanceSession,
		ResourceID:   created.ID,
		Metadata: map[string]any{
			"class_id":      in.ClassID.String(),
			"subject_id":    in.SubjectID.String(),
			"term_id":       in.TermID.String(),
			"session_date":  sessionDate.Format("2006-01-02"),
			"record_count":  len(in.Records),
			"status_counts": statusCounts,
		},
	}); err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}

	// Re-fetch through the detail joins so the caller gets the same
	// enriched shapes as GetSession without extra round trips after commit.
	sessionRow, err := q.GetAttendanceSessionByID(ctx, created.ID)
	if err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	records, err := q.ListAttendanceSessionRecords(ctx, created.ID)
	if err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	return sessionRow, records, nil
}

func (s *Service) GetSession(ctx context.Context, id uuid.UUID) (db.GetAttendanceSessionByIDRow, []db.ListAttendanceSessionRecordsRow, error) {
	sessionRow, err := s.queries.GetAttendanceSessionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAttendanceSessionByIDRow{}, nil, ErrSessionNotFound
		}
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	records, err := s.queries.ListAttendanceSessionRecords(ctx, id)
	if err != nil {
		return db.GetAttendanceSessionByIDRow{}, nil, err
	}
	return sessionRow, records, nil
}

func (s *Service) ListSessions(ctx context.Context, f ListFilter) ([]db.ListAttendanceSessionsRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	params := db.ListAttendanceSessionsParams{
		ClassID:    toPgUUIDPtr(f.ClassID),
		SubjectID:  toPgUUIDPtr(f.SubjectID),
		TermID:     toPgUUIDPtr(f.TermID),
		DateFrom:   toPgDatePtr(f.DateFrom),
		DateTo:     toPgDatePtr(f.DateTo),
		PageLimit:  int32(pageSize),
		PageOffset: int32((page - 1) * pageSize),
	}
	rows, err := s.queries.ListAttendanceSessions(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.queries.CountAttendanceSessions(ctx, db.CountAttendanceSessionsParams{
		ClassID:   params.ClassID,
		SubjectID: params.SubjectID,
		TermID:    params.TermID,
		DateFrom:  params.DateFrom,
		DateTo:    params.DateTo,
	})
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// UpdateRecord corrects a single mark. The status must always be given (a
// mark is changed, never unset); notes are left untouched when omitted. An
// update that changes nothing is a no-op without an audit row, mirroring
// the enrollment status flow (M9).
func (s *Service) UpdateRecord(ctx context.Context, actorID, id uuid.UUID, status string, notes *string) (db.GetAttendanceRecordByIDRow, error) {
	status, err := normalizeStatus(status)
	if err != nil {
		return db.GetAttendanceRecordByIDRow{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAttendanceRecordByIDRow{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetAttendanceRecordByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAttendanceRecordByIDRow{}, ErrRecordNotFound
		}
		return db.GetAttendanceRecordByIDRow{}, err
	}

	notesUpdated := notes != nil && (!current.Notes.Valid || *notes != current.Notes.String)
	if current.Status == status && !notesUpdated {
		return current, nil
	}

	if _, err := q.UpdateAttendanceRecord(ctx, db.UpdateAttendanceRecordParams{
		Status: status,
		Notes:  toPgText(notes),
		ID:     id,
	}); err != nil {
		return db.GetAttendanceRecordByIDRow{}, err
	}
	row, err := q.GetAttendanceRecordByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAttendanceRecordByIDRow{}, ErrRecordNotFound
		}
		return db.GetAttendanceRecordByIDRow{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAttendanceRecordUpdated,
		ResourceType: audit.ResourceAttendanceRecord,
		ResourceID:   row.ID,
		Metadata: map[string]any{
			"session_id":      row.SessionID.String(),
			"student_id":      row.StudentID.String(),
			"previous_status": current.Status,
			"new_status":      row.Status,
			"notes_updated":   notesUpdated,
		},
	}); err != nil {
		return db.GetAttendanceRecordByIDRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAttendanceRecordByIDRow{}, err
	}
	return row, nil
}

// StudentSummary aggregates one student's marks per status — the read
// groundwork the parent (M16) and student (M17) portals render. Optional
// term/year filters scope the window.
func (s *Service) StudentSummary(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) (Summary, error) {
	if _, err := s.queries.GetStudentByID(ctx, studentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Summary{}, ErrStudentNotFound
		}
		return Summary{}, err
	}

	rows, err := s.queries.SummarizeStudentAttendance(ctx, db.SummarizeStudentAttendanceParams{
		StudentID:      studentID,
		TermID:         toPgUUIDPtr(termID),
		AcademicYearID: toPgUUIDPtr(academicYearID),
	})
	if err != nil {
		return Summary{}, err
	}
	pairs := make([]statusCount, len(rows))
	for i, r := range rows {
		pairs[i] = statusCount{Status: r.Status, Total: r.Total}
	}
	return buildSummary(pairs, func(sum *Summary) {
		sum.StudentID = &studentID
		sum.TermID = termID
		sum.AcademicYearID = academicYearID
	}), nil
}

// ClassSummary aggregates one class's marks per status — the teacher's
// register overview, optionally narrowed to a term and/or a single
// teaching slot (subject).
func (s *Service) ClassSummary(ctx context.Context, classID uuid.UUID, termID, subjectID *uuid.UUID) (Summary, error) {
	if _, err := s.queries.GetClassByID(ctx, classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Summary{}, ErrClassNotFound
		}
		return Summary{}, err
	}

	rows, err := s.queries.SummarizeClassAttendance(ctx, db.SummarizeClassAttendanceParams{
		ClassID:   classID,
		TermID:    toPgUUIDPtr(termID),
		SubjectID: toPgUUIDPtr(subjectID),
	})
	if err != nil {
		return Summary{}, err
	}
	pairs := make([]statusCount, len(rows))
	for i, r := range rows {
		pairs[i] = statusCount{Status: r.Status, Total: r.Total}
	}
	return buildSummary(pairs, func(sum *Summary) {
		sum.ClassID = &classID
		sum.TermID = termID
		sum.SubjectID = subjectID
	}), nil
}

// buildSummary zero-fills all four statuses, applies the rows' counts, and
// stamps the summary's scope fields via the caller's hook. The sqlc GROUP
// BY queries return two distinct row types with identical shapes, so the
// callers normalize them into statusCount first.
type statusCount struct {
	Status string
	Total  int64
}

func buildSummary(rows []statusCount, scope func(*Summary)) Summary {
	counts := make(map[string]int64, len(AttendanceStatuses))
	for _, st := range AttendanceStatuses {
		counts[st] = 0
	}
	var total int64
	for _, r := range rows {
		counts[r.Status] = r.Total
		total += r.Total
	}
	sum := Summary{Counts: counts, TotalRecords: total}
	scope(&sum)
	return sum
}
