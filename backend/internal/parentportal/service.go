package parentportal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

var (
	// ErrGuardianNotFound maps a PARENT-role account with no guardians row
	// to a friendly 404 (mirrors ErrTeacherNotFound in teacherportal).
	ErrGuardianNotFound = errors.New("guardian profile not found for this account")
	// ErrChildAccessDenied marks per-child requests for students the
	// guardian is not linked to — the row-level boundary migration 000004's
	// comments flagged as this portal's job.
	ErrChildAccessDenied = errors.New("this student is not linked to your guardian account")
)

// GuardianProfile is the parent's own information, mirroring the teacher
// portal's profile block.
type GuardianProfile struct {
	GuardianID  string  `json:"guardian_id"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Occupation  *string `json:"occupation,omitempty"`
}

// ChildEnrollmentContext is the child's currently-active class membership.
// EnrollmentID is uuid.Nil when the child has no ENROLLED history row right
// now (graduated cohort, mid-transfer) — the JSON fields stay omitted then.
type ChildEnrollmentContext struct {
	EnrollmentID     string `json:"enrollment_id,omitempty"`
	ClassID          string `json:"class_id,omitempty"`
	ClassName        string `json:"class_name,omitempty"`
	ClassLevel       *string `json:"class_level,omitempty"`
	AcademicYearID   string `json:"academic_year_id,omitempty"`
	AcademicYearName string `json:"academic_year_name,omitempty"`
}

// ChildSummary is one linked child in the overview: identity, relationship,
// active enrollment context, and lifetime attendance/results aggregates for
// the per-child cards.
type ChildSummary struct {
	StudentID        string                  `json:"student_id"`
	StudentNumber    string                  `json:"student_number"`
	FirstName        string                  `json:"first_name"`
	LastName         string                  `json:"last_name"`
	Gender           *string                 `json:"gender,omitempty"`
	DateOfBirth      *string                 `json:"date_of_birth,omitempty"`
	Relationship     string                  `json:"relationship"`
	StudentStatus    string                  `json:"student_status"`
	Enrollment       ChildEnrollmentContext  `json:"enrollment"`
	AttendanceCounts ChildAttendanceCounts   `json:"attendance_counts"`
	ResultsCount     int64                   `json:"results_count"`
	AverageScore     *float64                `json:"average_score,omitempty"`
}

// ChildAttendanceCounts is the per-status lifetime mark breakdown rendered
// by the overview cards and the child detail header.
type ChildAttendanceCounts struct {
	Present  int64 `json:"present"`
	Absent   int64 `json:"absent"`
	Late     int64 `json:"late"`
	Excused  int64 `json:"excused"`
}

// ChildAttendanceRecord is one attendance mark for a child, with full session
// context — the register rows in the child detail view.
type ChildAttendanceRecord struct {
	RecordID    string  `json:"record_id"`
	SessionDate string  `json:"session_date"`
	Status      string  `json:"status"`
	Notes       *string `json:"notes,omitempty"`
	ClassID     string  `json:"class_id"`
	ClassName   string  `json:"class_name"`
	SubjectID   string  `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	SubjectCode string  `json:"subject_code"`
	TermName    string  `json:"term_name"`
}

// ChildResult is one result row for a child, with assessment context — the
// rows in the child's results history table.
type ChildResult struct {
	ResultID       string   `json:"result_id"`
	AssessmentID   string   `json:"assessment_id"`
	AssessmentTitle string  `json:"assessment_title"`
	AssessmentType string   `json:"assessment_type"`
	SubjectName    string   `json:"subject_name"`
	SubjectCode    string   `json:"subject_code"`
	ClassName      string   `json:"class_name"`
	TermName       string   `json:"term_name"`
	AssessmentDate string   `json:"assessment_date"`
	MaxScore       float64  `json:"max_score"`
	Score          *float64 `json:"score,omitempty"`
	Grade          *string  `json:"grade,omitempty"`
	Remarks        *string  `json:"remarks,omitempty"`
}

// ChildDetail is the per-child payload: identity summary plus the aggregate,
// recent marks, and results history the tabs render.
type ChildDetail struct {
	Student          ChildSummary           `json:"student"`
	AttendanceCounts ChildAttendanceCounts  `json:"attendance_counts"`
	RecentAttendance []ChildAttendanceRecord `json:"recent_attendance"`
	Results          []ChildResult          `json:"results"`
}

// ParentOverviewData is the /portal/parent/overview payload.
type ParentOverviewData struct {
	Profile GuardianProfile `json:"profile"`
	Children []ChildSummary `json:"children"`
	FeedAnnouncements []ParentAnnouncement `json:"announcements"`
}

// ParentAnnouncement is one feed announcement for the parent, mirroring the
// teacher portal's announcement item.
type ParentAnnouncement struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	Priority    string  `json:"priority"`
	TargetType  string  `json:"target_type"`
	TargetClass *string `json:"target_class,omitempty"`
	AuthorName  string  `json:"author_name"`
	PublishedAt *string `json:"published_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// Service resolves parent portal context against the database.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func textPtr(t pgtype.Text) *string {
	if t.Valid {
		v := t.String
		return &v
	}
	return nil
}

func numericFloat(n pgtype.Numeric) (float64, bool) {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0, false
	}
	return f.Float64, true
}

func (s *Service) GetParentOverview(ctx context.Context, userID uuid.UUID) (ParentOverviewData, error) {
	q := db.New(s.pool)

	// 1. Guardian profile (404 when this account has no guardians row).
	profRow, err := q.GetGuardianProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ParentOverviewData{}, ErrGuardianNotFound
		}
		return ParentOverviewData{}, fmt.Errorf("parentportal: get profile: %w", err)
	}

	profile := GuardianProfile{
		GuardianID: profRow.GuardianID.String(),
		FirstName:  profRow.FirstName,
		LastName:   profRow.LastName,
		Email:      textPtr(profRow.Email),
		Phone:      textPtr(profRow.Phone),
		Occupation: textPtr(profRow.Occupation),
	}

	// 2. Children with enrollment context and lifetime aggregates.
	children, err := s.buildChildSummaries(ctx, q, profRow.GuardianID)
	if err != nil {
		return ParentOverviewData{}, err
	}

	// 3. Announcements feed (M13's audience query already understands
	// guardians via student_guardians; reuse it, capped at 5 like the
	// other portals).
	annRows, err := q.ListFeedAnnouncements(ctx, db.ListFeedAnnouncementsParams{
		AuthorID: userID,
		Limit:    5,
		Offset:   0,
	})
	if err != nil {
		return ParentOverviewData{}, fmt.Errorf("parentportal: get announcements: %w", err)
	}
	announcements := make([]ParentAnnouncement, 0, len(annRows))
	for _, a := range annRows {
		authorName := a.AuthorFirstName + " " + a.AuthorLastName
		item := ParentAnnouncement{
			ID:         a.ID.String(),
			Title:      a.Title,
			Content:    a.Content,
			Priority:   a.Priority,
			TargetType: a.TargetType,
			AuthorName: authorName,
			CreatedAt:  a.CreatedAt.Time.Format(time.RFC3339),
		}
		if v := textPtr(a.TargetClassName); v != nil {
			item.TargetClass = v
		}
		if a.PublishedAt.Valid {
			pub := a.PublishedAt.Time.Format(time.RFC3339)
			item.PublishedAt = &pub
		}
		announcements = append(announcements, item)
	}

	return ParentOverviewData{
		Profile:           profile,
		Children:          children,
		FeedAnnouncements: announcements,
	}, nil
}

func (s *Service) GetChildDetail(ctx context.Context, userID uuid.UUID, studentID uuid.UUID) (ChildDetail, error) {
	q := db.New(s.pool)

	profRow, err := q.GetGuardianProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChildDetail{}, ErrGuardianNotFound
		}
		return ChildDetail{}, fmt.Errorf("parentportal: get profile: %w", err)
	}

	// Access gate: only linked children are readable (row-level boundary
	// from migration 000004; RBAC cannot express this).
	linked, err := q.IsGuardianLinkedToStudent(ctx, db.IsGuardianLinkedToStudentParams{
		GuardianID: profRow.GuardianID,
		StudentID:  studentID,
	})
	if err != nil {
		return ChildDetail{}, fmt.Errorf("parentportal: check link: %w", err)
	}
	if !linked {
		return ChildDetail{}, ErrChildAccessDenied
	}

	// Identity + lifetime aggregates.
	summaries, err := s.buildChildSummaries(ctx, q, profRow.GuardianID)
	if err != nil {
		return ChildDetail{}, err
	}
	var summary *ChildSummary
	for i := range summaries {
		if summaries[i].StudentID == studentID.String() {
			summary = &summaries[i]
			break
		}
	}
	if summary == nil {
		// Linked but nothing to show (e.g. no ENROLLED history). Still a
		// valid read: return the child shell with zeroed aggregates.
		idRow, err := q.GetStudentIdentityForGuardian(ctx, db.GetStudentIdentityForGuardianParams{
			GuardianID: profRow.GuardianID,
			StudentID:  studentID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ChildDetail{}, ErrChildAccessDenied
			}
			return ChildDetail{}, fmt.Errorf("parentportal: get child identity: %w", err)
		}
		summary = &ChildSummary{
			StudentID:     studentID.String(),
			StudentNumber: idRow.StudentNumber,
			FirstName:     idRow.FirstName,
			LastName:      idRow.LastName,
			Relationship:  idRow.Relationship,
			StudentStatus: idRow.StudentStatus,
			Enrollment:    ChildEnrollmentContext{},
			AttendanceCounts: ChildAttendanceCounts{},
		}
	}

	// Recent attendance marks (top 10).
	attRows, err := q.GetChildRecentAttendanceRecords(ctx, db.GetChildRecentAttendanceRecordsParams{
		StudentID: studentID,
		Limit:     10,
	})
	if err != nil {
		return ChildDetail{}, fmt.Errorf("parentportal: get child attendance: %w", err)
	}
	recentAttendance := make([]ChildAttendanceRecord, 0, len(attRows))
	for _, a := range attRows {
		record := ChildAttendanceRecord{
			RecordID:    a.RecordID.String(),
			SessionDate: a.SessionDate.Time.Format("2006-01-02"),
			Status:      a.Status,
			Notes:       textPtr(a.Notes),
			ClassID:     a.ClassID.String(),
			ClassName:   a.ClassName,
			SubjectID:   a.SubjectID.String(),
			SubjectName: a.SubjectName,
			SubjectCode: a.SubjectCode,
			TermName:    a.TermName,
		}
		recentAttendance = append(recentAttendance, record)
	}

	// Results history (term/year unfiltered — full record).
	resRows, err := q.ListStudentResults(ctx, db.ListStudentResultsParams{
		StudentID: studentID,
	})
	if err != nil {
		return ChildDetail{}, fmt.Errorf("parentportal: get child results: %w", err)
	}
	results := make([]ChildResult, 0, len(resRows))
	for _, r := range resRows {
		item := ChildResult{
			ResultID:        r.ID.String(),
			AssessmentID:    r.AssessmentID.String(),
			AssessmentTitle: r.AssessmentTitle,
			AssessmentType:  r.AssessmentType,
			SubjectName:     r.SubjectName,
			SubjectCode:     r.SubjectCode,
			ClassName:       r.ClassName,
			TermName:        r.TermName,
			AssessmentDate:  r.AssessmentDate.Time.Format("2006-01-02"),
		}
		if maxScore, ok := numericFloat(r.MaxScore); ok {
			item.MaxScore = maxScore
		}
		if score, ok := numericFloat(r.Score); ok {
			item.Score = &score
		}
		item.Grade = textPtr(r.Grade)
		item.Remarks = textPtr(r.Remarks)
		results = append(results, item)
	}

	detail := ChildDetail{
		Student:          *summary,
		AttendanceCounts: summary.AttendanceCounts,
		RecentAttendance: recentAttendance,
		Results:          results,
	}
	// Recompute the score average over the results just fetched so the
	// detail header matches the visible table even when the lifetime
	// aggregate predates new rows.
	if len(results) > 0 {
		sum := 0.0
		count := 0
		for _, r := range results {
			if r.Score != nil {
				sum += *r.Score
				count++
			}
		}
		if count > 0 {
			avg := sum / float64(count)
			detail.Student.AverageScore = &avg
		}
	}

	return detail, nil
}

// buildChildSummaries assembles the per-child overview list for one guardian.
func (s *Service) buildChildSummaries(ctx context.Context, q *db.Queries, guardianID uuid.UUID) ([]ChildSummary, error) {
	rows, err := q.ListChildrenForGuardianUser(ctx, guardianID)
	if err != nil {
		return nil, fmt.Errorf("parentportal: get children: %w", err)
	}

	children := make([]ChildSummary, 0, len(rows))
	for _, ch := range rows {
		summary := ChildSummary{
			StudentID:     ch.StudentID.String(),
			StudentNumber: ch.StudentNumber,
			FirstName:     ch.FirstName,
			LastName:      ch.LastName,
			Gender:        textPtr(ch.Gender),
			Relationship:  ch.Relationship,
			StudentStatus: ch.StudentStatus,
			AttendanceCounts: ChildAttendanceCounts{
				Present: ch.PresentCount,
				Absent:  ch.AbsentCount,
				Late:    ch.LateCount,
				Excused: ch.ExcusedCount,
			},
			ResultsCount: ch.ResultsCount,
		}
		if dob := ch.DateOfBirth; dob.Valid {
			d := dob.Time.Format("2006-01-02")
			summary.DateOfBirth = &d
		}
		// uuid.Nil marks "no active enrollment" (COALESCEd in SQL); omit
		// the enrollment context in JSON then.
		if ch.EnrollmentID != uuid.Nil {
			summary.Enrollment = ChildEnrollmentContext{
				EnrollmentID:     ch.EnrollmentID.String(),
				ClassID:          ch.ClassID.String(),
				ClassName:        ch.ClassName,
				ClassLevel:       textPtr(ch.ClassLevel),
				AcademicYearID:   ch.AcademicYearID.String(),
				AcademicYearName: ch.AcademicYearName,
			}
		}
		if ch.ResultsCount > 0 {
			avg := ch.AverageScore
			summary.AverageScore = &avg
		}
		children = append(children, summary)
	}
	return children, nil
}
