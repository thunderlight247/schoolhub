// Package studentportal implements the student portal domain (milestone M17).
// It is a read-only, self-scoped view over the student's own profile,
// enrollment and academic context, attendance, results, and announcements —
// mirroring the parentportal (M16) pattern, with the row-level boundary
// (a STUDENT account reads only its own records) enforced by resolving the
// student row from the authenticated user's person_id; no endpoint accepts a
// student id at all.
package studentportal

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

// ErrStudentNotFound maps a STUDENT-role account with no students row to a
// friendly 404 (mirrors ErrGuardianNotFound in parentportal).
var ErrStudentNotFound = errors.New("student profile not found for this account")

// StudentProfile is the student's own identity block, mirroring the profile
// blocks of the other portals.
type StudentProfile struct {
	StudentID     string  `json:"student_id"`
	StudentNumber string  `json:"student_number"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	OtherNames    *string `json:"other_names,omitempty"`
	Gender        *string `json:"gender,omitempty"`
	DateOfBirth   *string `json:"date_of_birth,omitempty"`
	Email         *string `json:"email,omitempty"`
	Phone         *string `json:"phone,omitempty"`
	Address       *string `json:"address,omitempty"`
	AdmissionDate *string `json:"admission_date,omitempty"`
	StudentStatus string  `json:"student_status"`
}

// AcademicContext is the active academic calendar, mirroring the teacher
// portal's context block. Nil fields mean no current year/term is configured.
type AcademicContext struct {
	AcademicYearID   *string `json:"academic_year_id,omitempty"`
	AcademicYearName *string `json:"academic_year_name,omitempty"`
	TermID           *string `json:"term_id,omitempty"`
	TermName         *string `json:"term_name,omitempty"`
}

// StudentEnrollmentContext is the student's currently-active class membership
// with programme context. Omitted from the JSON entirely when the student has
// no ENROLLED history row right now (transferred out, graduated cohort).
type StudentEnrollmentContext struct {
	EnrollmentID     string  `json:"enrollment_id"`
	EnrollmentStatus string  `json:"enrollment_status"`
	EnrollmentDate   string  `json:"enrollment_date"`
	ClassID          string  `json:"class_id"`
	ClassName        string  `json:"class_name"`
	ClassLevel       *string `json:"class_level,omitempty"`
	ClassCapacity    *int32  `json:"class_capacity,omitempty"`
	ProgrammeID      *string `json:"programme_id,omitempty"`
	ProgrammeCode    *string `json:"programme_code,omitempty"`
	ProgrammeName    *string `json:"programme_name,omitempty"`
	AcademicYearID   string  `json:"academic_year_id"`
	AcademicYearName string  `json:"academic_year_name"`
}

// StudentEnrollmentHistoryItem represents one historical or active enrollment
// row for the student across academic years.
type StudentEnrollmentHistoryItem struct {
	EnrollmentID     string  `json:"enrollment_id"`
	EnrollmentStatus string  `json:"enrollment_status"`
	EnrollmentDate   string  `json:"enrollment_date"`
	ExitDate         *string `json:"exit_date,omitempty"`
	EnrolledAt       string  `json:"enrolled_at"`
	ClassID          string  `json:"class_id"`
	ClassName        string  `json:"class_name"`
	ClassLevel       *string `json:"class_level,omitempty"`
	ProgrammeID      *string `json:"programme_id,omitempty"`
	ProgrammeCode    *string `json:"programme_code,omitempty"`
	ProgrammeName    *string `json:"programme_name,omitempty"`
	AcademicYearID   string  `json:"academic_year_id"`
	AcademicYearName string  `json:"academic_year_name"`
}

// AttendanceCounts is the per-status lifetime mark breakdown rendered by the
// overview attendance card.
type AttendanceCounts struct {
	Present int64 `json:"present"`
	Absent  int64 `json:"absent"`
	Late    int64 `json:"late"`
	Excused int64 `json:"excused"`
}

// ResultsSummary is the results headline for the overview card. AverageScore
// stays nil when there are no results, keeping "no results" and "0 average"
// distinguishable.
type ResultsSummary struct {
	ResultsCount int64    `json:"results_count"`
	AverageScore *float64 `json:"average_score,omitempty"`
}

// StudentAttendanceRecord is one attendance mark with full session context —
// the register rows in the overview and attendance views.
type StudentAttendanceRecord struct {
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

// StudentResult is one result row with assessment context — the rows in the
// results history table.
type StudentResult struct {
	ResultID        string   `json:"result_id"`
	AssessmentID    string   `json:"assessment_id"`
	AssessmentTitle string   `json:"assessment_title"`
	AssessmentType  string   `json:"assessment_type"`
	SubjectName     string   `json:"subject_name"`
	SubjectCode     string   `json:"subject_code"`
	ClassName       string   `json:"class_name"`
	TermName        string   `json:"term_name"`
	AssessmentDate  string   `json:"assessment_date"`
	MaxScore        float64  `json:"max_score"`
	Score           *float64 `json:"score,omitempty"`
	Grade           *string  `json:"grade,omitempty"`
	Remarks         *string  `json:"remarks,omitempty"`
}

// StudentAnnouncement is one feed announcement for the student, mirroring the
// parent portal's announcement item (M13's audience query already understands
// students via enrollments).
type StudentAnnouncement struct {
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

// StudentOverviewData is the /portal/student/overview payload.
type StudentOverviewData struct {
	Profile           StudentProfile                 `json:"profile"`
	AcademicContext   AcademicContext                `json:"academic_context"`
	Enrollment        *StudentEnrollmentContext       `json:"enrollment,omitempty"`
	EnrollmentHistory []StudentEnrollmentHistoryItem `json:"enrollment_history"`
	Attendance        AttendanceCounts               `json:"attendance_counts"`
	Results           ResultsSummary                 `json:"results_summary"`
	RecentAttendance  []StudentAttendanceRecord       `json:"recent_attendance"`
	RecentResults     []StudentResult                `json:"recent_results"`
	Announcements     []StudentAnnouncement          `json:"announcements"`
}

// Service resolves student portal context against the database.
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

func uuidPtr(u pgtype.UUID) *string {
	if u.Valid {
		v := uuid.UUID(u.Bytes).String()
		return &v
	}
	return nil
}

func dateStr(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	v := d.Time.Format("2006-01-02")
	return &v
}

func numericFloat(n pgtype.Numeric) (float64, bool) {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0, false
	}
	return f.Float64, true
}

// resolveStudent returns the student row behind the authenticated account.
func (s *Service) resolveStudent(ctx context.Context, q *db.Queries, userID uuid.UUID) (db.GetStudentProfileByUserIDRow, error) {
	row, err := q.GetStudentProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetStudentProfileByUserIDRow{}, ErrStudentNotFound
		}
		return db.GetStudentProfileByUserIDRow{}, fmt.Errorf("studentportal: get profile: %w", err)
	}
	return row, nil
}

func profileFromRow(row db.GetStudentProfileByUserIDRow) StudentProfile {
	return StudentProfile{
		StudentID:     row.StudentID.String(),
		StudentNumber: row.StudentNumber,
		FirstName:     row.FirstName,
		LastName:      row.LastName,
		OtherNames:    textPtr(row.OtherNames),
		Gender:        textPtr(row.Gender),
		DateOfBirth:   dateStr(row.DateOfBirth),
		Email:         textPtr(row.Email),
		Phone:         textPtr(row.Phone),
		Address:       textPtr(row.Address),
		AdmissionDate: dateStr(row.AdmissionDate),
		StudentStatus: row.StudentStatus,
	}
}

// attendanceCountsForStudent assembles the lifetime per-status breakdown.
func attendanceCountsForStudent(ctx context.Context, q *db.Queries, studentID uuid.UUID) (AttendanceCounts, error) {
	counts, err := q.GetStudentAttendanceCounts(ctx, studentID)
	if err != nil {
		return AttendanceCounts{}, fmt.Errorf("studentportal: get attendance counts: %w", err)
	}
	return AttendanceCounts{
		Present: counts.PresentCount,
		Absent:  counts.AbsentCount,
		Late:    counts.LateCount,
		Excused: counts.ExcusedCount,
	}, nil
}

// attendanceRecordsForStudent maps the shared record query rows into the
// portal payload shape, capped by limit.
func attendanceRecordsForStudent(ctx context.Context, q *db.Queries, studentID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error) {
	rows, err := q.GetStudentRecentAttendanceRecords(ctx, db.GetStudentRecentAttendanceRecordsParams{
		StudentID: studentID,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("studentportal: get attendance records: %w", err)
	}
	records := make([]StudentAttendanceRecord, 0, len(rows))
	for _, a := range rows {
		records = append(records, StudentAttendanceRecord{
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
		})
	}
	return records, nil
}

// resultsForStudent maps the shared M12 results history query (already
// understood by parentportal) into the portal payload shape. When termID or
// academicYearID is set (their pgtype.UUID Valid), the query scopes to that
// window; unset means the full history.
func resultsForStudent(ctx context.Context, q *db.Queries, studentID uuid.UUID, termID, academicYearID uuid.UUID) ([]StudentResult, error) {
	params := db.ListStudentResultsParams{StudentID: studentID}
	if termID != uuid.Nil {
		params.TermID = pgtype.UUID{Bytes: termID, Valid: true}
	}
	if academicYearID != uuid.Nil {
		params.AcademicYearID = pgtype.UUID{Bytes: academicYearID, Valid: true}
	}
	rows, err := q.ListStudentResults(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("studentportal: get results: %w", err)
	}
	results := make([]StudentResult, 0, len(rows))
	for _, r := range rows {
		item := StudentResult{
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
	return results, nil
}

// academicContext resolves the active calendar, tolerating a system with no
// current year/term configured (zero-value context, not an error) — the same
// tolerance teacherportal applies to GetCurrentAcademicYearAndTerm.
func academicContext(ctx context.Context, q *db.Queries) (AcademicContext, error) {
	var ctxOut AcademicContext
	row, err := q.GetCurrentAcademicYearAndTerm(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ctxOut, nil
		}
		return AcademicContext{}, fmt.Errorf("studentportal: get academic context: %w", err)
	}
	yearID := row.AcademicYearID.String()
	yearName := row.AcademicYearName
	ctxOut.AcademicYearID = &yearID
	ctxOut.AcademicYearName = &yearName
	if row.TermID.Valid {
		termID := uuid.UUID(row.TermID.Bytes).String()
		ctxOut.TermID = &termID
	}
	if row.TermName.Valid {
		termName := row.TermName.String
		ctxOut.TermName = &termName
	}
	return ctxOut, nil
}

// enrollmentContext maps the current-enrollment row; nil when the student has
// no active enrollment.
func enrollmentContext(row db.GetCurrentEnrollmentForStudentRow, err error) (*StudentEnrollmentContext, error) {
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("studentportal: get enrollment: %w", err)
	}
	enrollmentCtx := &StudentEnrollmentContext{
		EnrollmentID:     row.EnrollmentID.String(),
		EnrollmentStatus: row.EnrollmentStatus,
		EnrollmentDate:   row.EnrollmentDate.Time.Format("2006-01-02"),
		ClassID:          row.ClassID.String(),
		ClassName:        row.ClassName,
		AcademicYearID:   row.AcademicYearID.String(),
		AcademicYearName: row.AcademicYearName,
	}
	enrollmentCtx.ClassLevel = textPtr(row.ClassLevel)
	if row.ClassCapacity.Valid {
		capacity := row.ClassCapacity.Int32
		enrollmentCtx.ClassCapacity = &capacity
	}
	enrollmentCtx.ProgrammeID = uuidPtr(row.ProgrammeID)
	enrollmentCtx.ProgrammeCode = textPtr(row.ProgrammeCode)
	enrollmentCtx.ProgrammeName = textPtr(row.ProgrammeName)
	return enrollmentCtx, nil
}

// enrollmentHistoryForStudent loads the student's complete enrollment history.
func enrollmentHistoryForStudent(ctx context.Context, q *db.Queries, studentID uuid.UUID) ([]StudentEnrollmentHistoryItem, error) {
	rows, err := q.ListStudentEnrollments(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("studentportal: get enrollment history: %w", err)
	}
	history := make([]StudentEnrollmentHistoryItem, 0, len(rows))
	for _, row := range rows {
		item := StudentEnrollmentHistoryItem{
			EnrollmentID:     row.EnrollmentID.String(),
			EnrollmentStatus: row.EnrollmentStatus,
			EnrollmentDate:   row.EnrollmentDate.Time.Format("2006-01-02"),
			EnrolledAt:       row.EnrolledAt.Time.Format(time.RFC3339),
			ClassID:          row.ClassID.String(),
			ClassName:        row.ClassName,
			AcademicYearID:   row.AcademicYearID.String(),
			AcademicYearName: row.AcademicYearName,
		}
		item.ExitDate = dateStr(row.ExitDate)
		item.ClassLevel = textPtr(row.ClassLevel)
		item.ProgrammeID = uuidPtr(row.ProgrammeID)
		item.ProgrammeCode = textPtr(row.ProgrammeCode)
		item.ProgrammeName = textPtr(row.ProgrammeName)
		history = append(history, item)
	}
	return history, nil
}

func (s *Service) GetStudentOverview(ctx context.Context, userID uuid.UUID) (StudentOverviewData, error) {
	q := db.New(s.pool)

	// 1. Self profile (404 when this account has no students row).
	profRow, err := s.resolveStudent(ctx, q, userID)
	if err != nil {
		return StudentOverviewData{}, err
	}

	// 2. Academic calendar context (tolerates none configured).
	academicCtx, err := academicContext(ctx, q)
	if err != nil {
		return StudentOverviewData{}, err
	}

	// 3. Current enrollment & full enrollment history.
	enrollRow, err := q.GetCurrentEnrollmentForStudent(ctx, profRow.StudentID)
	enrollment, err := enrollmentContext(enrollRow, err)
	if err != nil {
		return StudentOverviewData{}, err
	}
	enrollmentHistory, err := enrollmentHistoryForStudent(ctx, q, profRow.StudentID)
	if err != nil {
		return StudentOverviewData{}, err
	}

	// 4. Lifetime aggregates.
	attendance, err := attendanceCountsForStudent(ctx, q, profRow.StudentID)
	if err != nil {
		return StudentOverviewData{}, err
	}
	summaryRow, err := q.GetStudentResultsSummary(ctx, profRow.StudentID)
	if err != nil {
		return StudentOverviewData{}, fmt.Errorf("studentportal: get results summary: %w", err)
	}
	summary := ResultsSummary{ResultsCount: summaryRow.ResultsCount}
	if summaryRow.ResultsCount > 0 {
		avg := summaryRow.AverageScore
		summary.AverageScore = &avg
	}

	// 5. Recent activity (top 5 each, mirroring the other portals' caps).
	recentAttendance, err := attendanceRecordsForStudent(ctx, q, profRow.StudentID, 5)
	if err != nil {
		return StudentOverviewData{}, err
	}
	recentResults, err := resultsForStudent(ctx, q, profRow.StudentID, uuid.Nil, uuid.Nil)
	if err != nil {
		return StudentOverviewData{}, err
	}
	if len(recentResults) > 5 {
		recentResults = recentResults[:5]
	}

	// 6. Announcements feed (M13's audience query understands students).
	announcements, err := announcementsForStudent(ctx, q, userID, 5)
	if err != nil {
		return StudentOverviewData{}, err
	}

	return StudentOverviewData{
		Profile:           profileFromRow(profRow),
		AcademicContext:   academicCtx,
		Enrollment:        enrollment,
		EnrollmentHistory: enrollmentHistory,
		Attendance:        attendance,
		Results:           summary,
		RecentAttendance:  recentAttendance,
		RecentResults:     recentResults,
		Announcements:     announcements,
	}, nil
}

// announcementsForStudent maps the shared M13 feed query (already understood
// by parentportal) into the portal payload shape.
func announcementsForStudent(ctx context.Context, q *db.Queries, userID uuid.UUID, limit int32) ([]StudentAnnouncement, error) {
	rows, err := q.ListFeedAnnouncements(ctx, db.ListFeedAnnouncementsParams{
		AuthorID: userID,
		Limit:    limit,
		Offset:   0,
	})
	if err != nil {
		return nil, fmt.Errorf("studentportal: get announcements: %w", err)
	}
	announcements := make([]StudentAnnouncement, 0, len(rows))
	for _, a := range rows {
		item := StudentAnnouncement{
			ID:         a.ID.String(),
			Title:      a.Title,
			Content:    a.Content,
			Priority:   a.Priority,
			TargetType: a.TargetType,
			AuthorName: a.AuthorFirstName + " " + a.AuthorLastName,
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
	return announcements, nil
}

// ListStudentAttendance is the fuller attendance view behind
// GET /portal/student/attendance (the overview caps itself at 5).
func (s *Service) ListStudentAttendance(ctx context.Context, userID uuid.UUID, limit int32) ([]StudentAttendanceRecord, error) {
	q := db.New(s.pool)

	profRow, err := s.resolveStudent(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	return attendanceRecordsForStudent(ctx, q, profRow.StudentID, limit)
}

// ListStudentResultsHistory is the full results record behind
// GET /portal/student/results. Optional term/year filters scope the window;
// absent means everything (the shared M12 query's contract).
func (s *Service) ListStudentResultsHistory(ctx context.Context, userID uuid.UUID, termID, academicYearID uuid.UUID) ([]StudentResult, error) {
	q := db.New(s.pool)

	profRow, err := s.resolveStudent(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	return resultsForStudent(ctx, q, profRow.StudentID, termID, academicYearID)
}

// ListStudentEnrollments returns the full enrollment history for the student
// behind GET /portal/student/enrollments.
func (s *Service) ListStudentEnrollments(ctx context.Context, userID uuid.UUID) ([]StudentEnrollmentHistoryItem, error) {
	q := db.New(s.pool)

	profRow, err := s.resolveStudent(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	return enrollmentHistoryForStudent(ctx, q, profRow.StudentID)
}
