// Package assessments implements the assessments and results domain (M12):
// graded events per class/subject/term and per-student results with score,
// optional letter grade, and remarks. Creation validates the whole chain —
// subject in the class curriculum (M8), term in the class's academic year,
// date inside the term — and every mutation writes its audit row in the
// same transaction. Scores are bounded against the assessment's max_score
// in the service, since a CHECK constraint cannot reference another table.
package assessments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

// Assessment types (CHECK constraint in 000012).
const (
	TypeQuiz       = "QUIZ"
	TypeTest       = "TEST"
	TypeAssignment = "ASSIGNMENT"
	TypeProject    = "PROJECT"
	TypeMidterm    = "MIDTERM"
	TypeFinal      = "FINAL"
)

var AssessmentTypes = []string{TypeQuiz, TypeTest, TypeAssignment, TypeProject, TypeMidterm, TypeFinal}

var (
	ErrAssessmentNotFound  = errors.New("assessment not found")
	ErrResultNotFound      = errors.New("result not found")
	ErrClassNotFound       = errors.New("class not found")
	ErrSubjectNotFound     = errors.New("subject not found")
	ErrTermNotFound        = errors.New("term not found")
	ErrStudentNotFound     = errors.New("student not found")
	ErrSubjectNotInClass   = errors.New("subject is not assigned to this class")
	ErrTermYearMismatch    = errors.New("term does not belong to the class's academic year")
	ErrDateOutsideTerm     = errors.New("assessment date falls outside the term's start and end dates")
	ErrTitleRequired       = errors.New("title is required")
	ErrInvalidType         = errors.New("invalid assessment type")
	ErrInvalidMaxScore     = errors.New("max_score must be greater than zero")
	ErrInvalidWeight       = errors.New("weight must be between 0 and 100")
	ErrDuplicateAssessment = errors.New("an assessment with this title already exists for this class, subject, and term")
	ErrStudentNotEnrolled  = errors.New("student is not enrolled in this class")
	ErrDuplicateStudent    = errors.New("a student appears more than once in the score sheet")
	ErrNoResults           = errors.New("at least one result is required")
	ErrInvalidScore        = errors.New("score must be between 0 and the assessment's max_score")
)

// ListFilter carries the optional assessment-list filters.
type ListFilter struct {
	ClassID   *uuid.UUID
	SubjectID *uuid.UUID
	TermID    *uuid.UUID
	Type      string
	Page      int
	PageSize  int
}

// CreateAssessmentInput is the assessment header payload.
type CreateAssessmentInput struct {
	ClassID        uuid.UUID
	SubjectID      uuid.UUID
	TermID         uuid.UUID
	Title          string
	Type           string
	MaxScore       float64
	Weight         *float64
	AssessmentDate time.Time
}

func (c *CreateAssessmentInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		ClassID        string   `json:"class_id"`
		SubjectID      string   `json:"subject_id"`
		TermID         string   `json:"term_id"`
		Title          string   `json:"title"`
		Type           string   `json:"type"`
		MaxScore       *float64 `json:"max_score"`
		Weight         *float64 `json:"weight"`
		AssessmentDate string   `json:"assessment_date"`
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
	c.Title = aux.Title
	c.Type = aux.Type
	if aux.MaxScore != nil {
		c.MaxScore = *aux.MaxScore
	}
	c.Weight = aux.Weight
	if aux.AssessmentDate != "" {
		parsed, err := time.Parse("2006-01-02", aux.AssessmentDate)
		if err != nil {
			return err
		}
		c.AssessmentDate = parsed
	}
	return nil
}

// UpdateAssessmentInput is the partial-edit payload: nil fields keep their
// current values (NULL narg COALESCE at the SQL layer).
type UpdateAssessmentInput struct {
	Title          *string
	Type           *string
	MaxScore       *float64
	Weight         *float64
	AssessmentDate *time.Time
}

func (u *UpdateAssessmentInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		Title          *string  `json:"title"`
		Type           *string  `json:"type"`
		MaxScore       *float64 `json:"max_score"`
		Weight         *float64 `json:"weight"`
		AssessmentDate *string  `json:"assessment_date"`
	}
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.Title = aux.Title
	u.Type = aux.Type
	u.MaxScore = aux.MaxScore
	u.Weight = aux.Weight
	if aux.AssessmentDate != nil && *aux.AssessmentDate != "" {
		parsed, err := time.Parse("2006-01-02", *aux.AssessmentDate)
		if err != nil {
			return err
		}
		u.AssessmentDate = &parsed
	}
	return nil
}

// ResultInput is one student's mark inside a score-sheet entry payload.
// Score is optional: a row can be created before it is marked (NULL score).
type ResultInput struct {
	StudentID uuid.UUID
	Score     *float64
	Grade     *string
	Remarks   *string
}

func (r *ResultInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		StudentID string   `json:"student_id"`
		Score     *float64 `json:"score"`
		Grade     *string  `json:"grade"`
		Remarks   *string  `json:"remarks"`
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
	r.Score = aux.Score
	r.Grade = aux.Grade
	r.Remarks = aux.Remarks
	return nil
}

// UpdateResultInput is the single-result correction payload: nil fields
// keep their current values.
type UpdateResultInput struct {
	Score   *float64
	Grade   *string
	Remarks *string
}

func (u *UpdateResultInput) UnmarshalJSON(data []byte) error {
	type alias struct {
		Score   *float64 `json:"score"`
		Grade   *string  `json:"grade"`
		Remarks *string  `json:"remarks"`
	}
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	u.Score = aux.Score
	u.Grade = aux.Grade
	u.Remarks = aux.Remarks
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

func normalizeType(raw string) (string, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return TypeTest, nil
	}
	for _, allowed := range AssessmentTypes {
		if t == allowed {
			return t, nil
		}
	}
	return "", ErrInvalidType
}

func toMidnightUTC(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// NumericToFloat renders a non-null pgtype.Numeric (score/max_score/weight)
// for the DTO layer; sqlc maps NUMERIC to pgtype.Numeric whose Value() is a
// decimal string.
func NumericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	v, err := n.Value()
	if err != nil {
		return 0
	}
	s, _ := v.(string)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// NumericToFloatPtr renders a nullable Numeric: nil when NULL.
func NumericToFloatPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f := NumericToFloat(n)
	return &f
}

func floatToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func floatPtrToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	return floatToNumeric(*f)
}

func toPgUUIDPtr(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func toPgTextPtr(s *string) pgtype.Text {
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

// validateScoreBounds enforces 0 <= score <= max_score. The DB CHECK can't
// see the assessment's max_score (CHECKs can't reference other tables), so
// entry and correction enforce it here.
func validateScoreBounds(score, maxScore float64) error {
	if score < 0 || score > maxScore {
		return ErrInvalidScore
	}
	return nil
}

// CreateAssessment validates the whole chain (subject in class curriculum,
// term in academic year, date in term) and creates the assessment header
// with its audit entry atomically.
func (s *Service) CreateAssessment(ctx context.Context, actorID uuid.UUID, in CreateAssessmentInput) (db.GetAssessmentByIDRow, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return db.GetAssessmentByIDRow{}, ErrTitleRequired
	}
	typ, err := normalizeType(in.Type)
	if err != nil {
		return db.GetAssessmentByIDRow{}, err
	}
	if in.MaxScore <= 0 {
		return db.GetAssessmentByIDRow{}, ErrInvalidMaxScore
	}
	if in.Weight != nil && (*in.Weight < 0 || *in.Weight > 100) {
		return db.GetAssessmentByIDRow{}, ErrInvalidWeight
	}

	assessmentDate := in.AssessmentDate
	if assessmentDate.IsZero() {
		assessmentDate = time.Now()
	}
	assessmentDate = toMidnightUTC(assessmentDate)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAssessmentByIDRow{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	classRow, err := q.GetClassByID(ctx, in.ClassID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAssessmentByIDRow{}, ErrClassNotFound
		}
		return db.GetAssessmentByIDRow{}, err
	}
	if _, err := q.GetSubjectByID(ctx, in.SubjectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAssessmentByIDRow{}, ErrSubjectNotFound
		}
		return db.GetAssessmentByIDRow{}, err
	}
	termRow, err := q.GetTermByID(ctx, in.TermID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAssessmentByIDRow{}, ErrTermNotFound
		}
		return db.GetAssessmentByIDRow{}, err
	}

	if termRow.AcademicYearID != classRow.AcademicYearID {
		return db.GetAssessmentByIDRow{}, ErrTermYearMismatch
	}

	hasPairing, err := q.HasClassSubject(ctx, db.HasClassSubjectParams{
		ClassID:   in.ClassID,
		SubjectID: in.SubjectID,
	})
	if err != nil {
		return db.GetAssessmentByIDRow{}, err
	}
	if !hasPairing {
		return db.GetAssessmentByIDRow{}, ErrSubjectNotInClass
	}

	if assessmentDate.Before(termRow.StartDate.Time) || assessmentDate.After(termRow.EndDate.Time) {
		return db.GetAssessmentByIDRow{}, ErrDateOutsideTerm
	}

	created, err := q.CreateAssessment(ctx, db.CreateAssessmentParams{
		ClassID:        in.ClassID,
		SubjectID:      in.SubjectID,
		TermID:         in.TermID,
		Title:          title,
		Type:           typ,
		MaxScore:       floatToNumeric(in.MaxScore),
		Weight:         floatPtrToNumeric(in.Weight),
		AssessmentDate: pgtype.Date{Time: assessmentDate, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.GetAssessmentByIDRow{}, ErrDuplicateAssessment
		}
		return db.GetAssessmentByIDRow{}, err
	}

	metadata := map[string]any{
		"class_id":        in.ClassID.String(),
		"subject_id":      in.SubjectID.String(),
		"term_id":         in.TermID.String(),
		"title":           title,
		"type":            typ,
		"max_score":       in.MaxScore,
		"assessment_date": assessmentDate.Format("2006-01-02"),
	}
	if in.Weight != nil {
		metadata["weight"] = *in.Weight
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAssessmentCreated,
		ResourceType: audit.ResourceAssessment,
		ResourceID:   created.ID,
		Metadata:     metadata,
	}); err != nil {
		return db.GetAssessmentByIDRow{}, err
	}

	row, err := q.GetAssessmentByID(ctx, created.ID)
	if err != nil {
		return db.GetAssessmentByIDRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAssessmentByIDRow{}, err
	}
	return row, nil
}

func (s *Service) GetAssessment(ctx context.Context, id uuid.UUID) (db.GetAssessmentByIDRow, error) {
	row, err := s.queries.GetAssessmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAssessmentByIDRow{}, ErrAssessmentNotFound
		}
		return db.GetAssessmentByIDRow{}, err
	}
	return row, nil
}

func (s *Service) ListAssessments(ctx context.Context, f ListFilter) ([]db.ListAssessmentsRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	var typePg pgtype.Text
	if f.Type != "" {
		typ, err := normalizeType(f.Type)
		if err != nil {
			return nil, 0, err
		}
		typePg = pgtype.Text{String: typ, Valid: true}
	}

	params := db.ListAssessmentsParams{
		ClassID:    toPgUUIDPtr(f.ClassID),
		SubjectID:  toPgUUIDPtr(f.SubjectID),
		TermID:     toPgUUIDPtr(f.TermID),
		Type:       typePg,
		PageLimit:  int32(pageSize),
		PageOffset: int32((page - 1) * pageSize),
	}
	rows, err := s.queries.ListAssessments(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.queries.CountAssessments(ctx, db.CountAssessmentsParams{
		ClassID:   params.ClassID,
		SubjectID: params.SubjectID,
		TermID:    params.TermID,
		Type:      params.Type,
	})
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Service) UpdateAssessment(ctx context.Context, actorID, id uuid.UUID, in UpdateAssessmentInput) (db.GetAssessmentByIDRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetAssessmentByIDRow{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetAssessmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAssessmentByIDRow{}, ErrAssessmentNotFound
		}
		return db.GetAssessmentByIDRow{}, err
	}

	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return db.GetAssessmentByIDRow{}, ErrTitleRequired
		}
		in.Title = &t
	}
	if in.Type != nil {
		typ, err := normalizeType(*in.Type)
		if err != nil {
			return db.GetAssessmentByIDRow{}, err
		}
		in.Type = &typ
	}
	if in.MaxScore != nil && *in.MaxScore <= 0 {
		return db.GetAssessmentByIDRow{}, ErrInvalidMaxScore
	}
	if in.Weight != nil && (*in.Weight < 0 || *in.Weight > 100) {
		return db.GetAssessmentByIDRow{}, ErrInvalidWeight
	}
	if in.AssessmentDate != nil {
		d := toMidnightUTC(*in.AssessmentDate)
		termRow, err := q.GetTermByID(ctx, current.TermID)
		if err != nil {
			return db.GetAssessmentByIDRow{}, err
		}
		if d.Before(termRow.StartDate.Time) || d.After(termRow.EndDate.Time) {
			return db.GetAssessmentByIDRow{}, ErrDateOutsideTerm
		}
		in.AssessmentDate = &d
	}

	if _, err := q.UpdateAssessment(ctx, db.UpdateAssessmentParams{
		ID:             id,
		Title:          toPgTextPtr(in.Title),
		Type:           toPgTextPtr(in.Type),
		MaxScore:       floatPtrToNumeric(in.MaxScore),
		Weight:         floatPtrToNumeric(in.Weight),
		AssessmentDate: toPgDatePtr(in.AssessmentDate),
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return db.GetAssessmentByIDRow{}, ErrDuplicateAssessment
		}
		return db.GetAssessmentByIDRow{}, err
	}

	row, err := q.GetAssessmentByID(ctx, id)
	if err != nil {
		return db.GetAssessmentByIDRow{}, err
	}

	metadata := map[string]any{
		"class_id":   row.ClassID.String(),
		"subject_id": row.SubjectID.String(),
		"term_id":    row.TermID.String(),
		"title":      row.Title,
	}
	if in.Title != nil {
		metadata["new_title"] = *in.Title
	}
	if in.Type != nil {
		metadata["new_type"] = *in.Type
	}
	if in.MaxScore != nil {
		metadata["new_max_score"] = *in.MaxScore
	}
	if in.Weight != nil {
		metadata["new_weight"] = *in.Weight
	}
	if in.AssessmentDate != nil {
		metadata["new_assessment_date"] = in.AssessmentDate.Format("2006-01-02")
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAssessmentUpdated,
		ResourceType: audit.ResourceAssessment,
		ResourceID:   id,
		Metadata:     metadata,
	}); err != nil {
		return db.GetAssessmentByIDRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetAssessmentByIDRow{}, err
	}
	return row, nil
}

func (s *Service) DeleteAssessment(ctx context.Context, actorID, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetAssessmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssessmentNotFound
		}
		return err
	}

	if err := q.DeleteAssessment(ctx, id); err != nil {
		return err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionAssessmentDeleted,
		ResourceType: audit.ResourceAssessment,
		ResourceID:   id,
		Metadata: map[string]any{
			"class_id":   current.ClassID.String(),
			"subject_id": current.SubjectID.String(),
			"term_id":    current.TermID.String(),
			"title":      current.Title,
		},
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RecordResults enters or batches student scores for an assessment. It
// verifies that every student is actively enrolled in the assessment's class,
// and enforces 0 <= score <= max_score.
func (s *Service) RecordResults(ctx context.Context, actorID, assessmentID uuid.UUID, in []ResultInput) ([]db.ListAssessmentResultsRow, error) {
	if len(in) == 0 {
		return nil, ErrNoResults
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	assessment, err := q.GetAssessmentByID(ctx, assessmentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssessmentNotFound
		}
		return nil, err
	}

	maxScore := NumericToFloat(assessment.MaxScore)
	seen := make(map[uuid.UUID]struct{}, len(in))
	for _, r := range in {
		if _, dup := seen[r.StudentID]; dup {
			return nil, ErrDuplicateStudent
		}
		seen[r.StudentID] = struct{}{}

		if r.Score != nil {
			if err := validateScoreBounds(*r.Score, maxScore); err != nil {
				return nil, err
			}
		}

		enrolled, err := q.HasActiveClassEnrollment(ctx, db.HasActiveClassEnrollmentParams{
			StudentID: r.StudentID,
			ClassID:   assessment.ClassID,
		})
		if err != nil {
			return nil, err
		}
		if !enrolled {
			return nil, fmt.Errorf("%w: %s", ErrStudentNotEnrolled, r.StudentID)
		}
	}

	for _, r := range in {
		if _, err := q.CreateResult(ctx, db.CreateResultParams{
			AssessmentID: assessmentID,
			StudentID:    r.StudentID,
			Score:        floatPtrToNumeric(r.Score),
			Grade:        toPgTextPtr(r.Grade),
			Remarks:      toPgTextPtr(r.Remarks),
		}); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
				return nil, ErrDuplicateStudent
			}
			return nil, err
		}
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionResultsRecorded,
		ResourceType: audit.ResourceAssessment,
		ResourceID:   assessmentID,
		Metadata: map[string]any{
			"assessment_title": assessment.Title,
			"result_count":     len(in),
		},
	}); err != nil {
		return nil, err
	}

	rows, err := q.ListAssessmentResults(ctx, assessmentID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Service) ListAssessmentResults(ctx context.Context, assessmentID uuid.UUID) ([]db.ListAssessmentResultsRow, error) {
	if _, err := s.queries.GetAssessmentByID(ctx, assessmentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssessmentNotFound
		}
		return nil, err
	}
	return s.queries.ListAssessmentResults(ctx, assessmentID)
}

func (s *Service) GetResult(ctx context.Context, id uuid.UUID) (db.GetResultByIDRow, error) {
	row, err := s.queries.GetResultByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetResultByIDRow{}, ErrResultNotFound
		}
		return db.GetResultByIDRow{}, err
	}
	return row, nil
}

func (s *Service) UpdateResult(ctx context.Context, actorID, id uuid.UUID, in UpdateResultInput) (db.GetResultByIDRow, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.GetResultByIDRow{}, err
	}
	defer tx.Rollback(ctx)

	q := s.queries.WithTx(tx)

	current, err := q.GetResultByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetResultByIDRow{}, ErrResultNotFound
		}
		return db.GetResultByIDRow{}, err
	}

	maxScore := NumericToFloat(current.MaxScore)
	if in.Score != nil {
		if err := validateScoreBounds(*in.Score, maxScore); err != nil {
			return db.GetResultByIDRow{}, err
		}
	}

	scoreChanged := in.Score != nil && (!current.Score.Valid || *in.Score != NumericToFloat(current.Score))
	gradeChanged := in.Grade != nil && (!current.Grade.Valid || *in.Grade != current.Grade.String)
	remarksChanged := in.Remarks != nil && (!current.Remarks.Valid || *in.Remarks != current.Remarks.String)

	if !scoreChanged && !gradeChanged && !remarksChanged {
		return current, nil
	}

	if _, err := q.UpdateResult(ctx, db.UpdateResultParams{
		ID:      id,
		Score:   floatPtrToNumeric(in.Score),
		Grade:   toPgTextPtr(in.Grade),
		Remarks: toPgTextPtr(in.Remarks),
	}); err != nil {
		return db.GetResultByIDRow{}, err
	}

	row, err := q.GetResultByID(ctx, id)
	if err != nil {
		return db.GetResultByIDRow{}, err
	}

	if err := audit.Record(ctx, q, audit.Entry{
		ActorUserID:  actorID,
		Action:       audit.ActionResultUpdated,
		ResourceType: audit.ResourceResult,
		ResourceID:   id,
		Metadata: map[string]any{
			"assessment_id":   row.AssessmentID.String(),
			"student_id":      row.StudentID.String(),
			"score_changed":   scoreChanged,
			"grade_changed":   gradeChanged,
			"remarks_changed": remarksChanged,
		},
	}); err != nil {
		return db.GetResultByIDRow{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.GetResultByIDRow{}, err
	}
	return row, nil
}

func (s *Service) ListStudentResults(ctx context.Context, studentID uuid.UUID, termID, academicYearID *uuid.UUID) ([]db.ListStudentResultsRow, error) {
	if _, err := s.queries.GetStudentByID(ctx, studentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStudentNotFound
		}
		return nil, err
	}
	return s.queries.ListStudentResults(ctx, db.ListStudentResultsParams{
		StudentID:      studentID,
		TermID:         toPgUUIDPtr(termID),
		AcademicYearID: toPgUUIDPtr(academicYearID),
	})
}
