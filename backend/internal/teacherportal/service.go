// Package teacherportal implements the teacher portal domain (milestone M15).
package teacherportal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

var (
	ErrTeacherNotFound      = errors.New("teacher staff record not found for this account")
	ErrForbiddenClassAccess = errors.New("you are not authorized to view the roster for this class")
	ErrClassNotFound        = errors.New("class not found")
)

type TeacherProfile struct {
	StaffID          string  `json:"staff_id"`
	StaffNumber      string  `json:"staff_number"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Email            string  `json:"email"`
	Phone            *string `json:"phone,omitempty"`
	DepartmentName   *string `json:"department_name,omitempty"`
	PositionTitle    *string `json:"position_title,omitempty"`
	EmploymentStatus string  `json:"employment_status"`
}

type TeacherCounts struct {
	TotalClasses            int64 `json:"total_classes"`
	TotalSubjects           int64 `json:"total_subjects"`
	TotalStudents           int64 `json:"total_students"`
	TotalAttendanceSessions int64 `json:"total_attendance_sessions"`
	TotalAssessments        int64 `json:"total_assessments"`
}

type AcademicContext struct {
	AcademicYearID   *string `json:"academic_year_id,omitempty"`
	AcademicYearName *string `json:"academic_year_name,omitempty"`
	TermID           *string `json:"term_id,omitempty"`
	TermName         *string `json:"term_name,omitempty"`
}

type AssignedClass struct {
	AssignmentID          string  `json:"assignment_id"`
	ClassID               string  `json:"class_id"`
	ClassName             string  `json:"class_name"`
	ClassLevel            *string `json:"class_level,omitempty"`
	ClassCapacity         int32   `json:"class_capacity"`
	AcademicYearID        string  `json:"academic_year_id"`
	AcademicYearName      string  `json:"academic_year_name"`
	SubjectID             string  `json:"subject_id"`
	SubjectName           string  `json:"subject_name"`
	SubjectCode           string  `json:"subject_code"`
	SubjectType           string  `json:"subject_type"`
	EnrolledStudentsCount int64   `json:"enrolled_students_count"`
}

type RecentAttendanceSession struct {
	SessionID    string `json:"session_id"`
	SessionDate  string `json:"session_date"`
	ClassID      string `json:"class_id"`
	ClassName    string `json:"class_name"`
	SubjectID    string `json:"subject_id"`
	SubjectName  string `json:"subject_name"`
	SubjectCode  string `json:"subject_code"`
	TermName     string `json:"term_name"`
	PresentCount int64  `json:"present_count"`
	AbsentCount  int64  `json:"absent_count"`
	LateCount    int64  `json:"late_count"`
	ExcusedCount int64  `json:"excused_count"`
	TotalMarks   int64  `json:"total_marks"`
}

type RecentAssessment struct {
	AssessmentID       string   `json:"assessment_id"`
	Title              string   `json:"title"`
	AssessmentType     string   `json:"assessment_type"`
	MaxScore           float64  `json:"max_score"`
	Weight             *float64 `json:"weight,omitempty"`
	AssessmentDate     string   `json:"assessment_date"`
	ClassID            string   `json:"class_id"`
	ClassName          string   `json:"class_name"`
	SubjectID          string   `json:"subject_id"`
	SubjectName        string   `json:"subject_name"`
	SubjectCode        string   `json:"subject_code"`
	TermName           string   `json:"term_name"`
	ResultsCount       int64    `json:"results_count"`
	TotalStudentsCount int64    `json:"total_students_count"`
}

type TeacherAnnouncement struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	Priority    string  `json:"priority"`
	Status      string  `json:"status"`
	TargetType  string  `json:"target_type"`
	AuthorName  string  `json:"author_name"`
	PublishedAt *string `json:"published_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type StudentRosterItem struct {
	StudentID            string  `json:"student_id"`
	StudentNumber        string  `json:"student_number"`
	FirstName            string  `json:"first_name"`
	LastName             string  `json:"last_name"`
	Gender               *string `json:"gender,omitempty"`
	DateOfBirth          *string `json:"date_of_birth,omitempty"`
	EnrollmentStatus     string  `json:"enrollment_status"`
	EnrolledAt           string  `json:"enrolled_at"`
	GuardianName         *string `json:"guardian_name,omitempty"`
	GuardianPhone        *string `json:"guardian_phone,omitempty"`
	GuardianEmail        *string `json:"guardian_email,omitempty"`
	GuardianRelationship *string `json:"guardian_relationship,omitempty"`
}

type TeacherOverviewData struct {
	Profile          TeacherProfile            `json:"profile"`
	AcademicContext  AcademicContext           `json:"academic_context"`
	Counts           TeacherCounts             `json:"counts"`
	AssignedClasses  []AssignedClass           `json:"assigned_classes"`
	RecentAttendance []RecentAttendanceSession `json:"recent_attendance"`
	RecentAssessments []RecentAssessment       `json:"recent_assessments"`
	Announcements    []TeacherAnnouncement     `json:"announcements"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) GetTeacherOverview(ctx context.Context, userID uuid.UUID) (TeacherOverviewData, error) {
	q := db.New(s.pool)

	// 1. Resolve teacher profile
	profRow, err := q.GetTeacherProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeacherOverviewData{}, ErrTeacherNotFound
		}
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get profile: %w", err)
	}

	profile := TeacherProfile{
		StaffID:          profRow.StaffID.String(),
		StaffNumber:      profRow.StaffNumber,
		FirstName:        profRow.FirstName,
		LastName:         profRow.LastName,
		Email:            profRow.Email.String,
		EmploymentStatus: profRow.EmploymentStatus,
	}
	if profRow.Phone.Valid {
		p := profRow.Phone.String
		profile.Phone = &p
	}
	if profRow.DepartmentName.Valid {
		dn := profRow.DepartmentName.String
		profile.DepartmentName = &dn
	}
	if profRow.PositionTitle.Valid {
		pt := profRow.PositionTitle.String
		profile.PositionTitle = &pt
	}

	// 2. Resolve Academic Context
	var academicCtx AcademicContext
	ayRow, err := q.GetCurrentAcademicYearAndTerm(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get academic context: %w", err)
	} else if err == nil {
		ayID := ayRow.AcademicYearID.String()
		ayName := ayRow.AcademicYearName
		academicCtx.AcademicYearID = &ayID
		academicCtx.AcademicYearName = &ayName
		if ayRow.TermID.Valid {
			tID := uuid.UUID(ayRow.TermID.Bytes).String()
			academicCtx.TermID = &tID
		}
		if ayRow.TermName.Valid {
			tName := ayRow.TermName.String
			academicCtx.TermName = &tName
		}
	}

	// 3. Counts
	countsRow, err := q.GetTeacherDashboardCounts(ctx, profRow.StaffID)
	if err != nil {
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get counts: %w", err)
	}

	counts := TeacherCounts{
		TotalClasses:            countsRow.TotalClasses,
		TotalSubjects:           countsRow.TotalSubjects,
		TotalStudents:           countsRow.TotalStudents,
		TotalAttendanceSessions: countsRow.TotalAttendanceSessions,
		TotalAssessments:        countsRow.TotalAssessments,
	}

	// 4. Assigned classes with stats
	classRows, err := q.GetTeacherClassesWithStats(ctx, profRow.StaffID)
	if err != nil {
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get assigned classes: %w", err)
	}

	assignedClasses := make([]AssignedClass, 0, len(classRows))
	for _, c := range classRows {
		var capVal int32
		if c.ClassCapacity.Valid {
			capVal = c.ClassCapacity.Int32
		}
		item := AssignedClass{
			AssignmentID:          c.AssignmentID.String(),
			ClassID:               c.ClassID.String(),
			ClassName:             c.ClassName,
			ClassCapacity:         capVal,
			AcademicYearID:        c.AcademicYearID.String(),
			AcademicYearName:      c.AcademicYearName,
			SubjectID:             c.SubjectID.String(),
			SubjectName:           c.SubjectName,
			SubjectCode:           c.SubjectCode,
			SubjectType:           c.SubjectType,
			EnrolledStudentsCount: c.EnrolledStudentsCount,
		}
		if c.ClassLevel.Valid {
			lvl := c.ClassLevel.String
			item.ClassLevel = &lvl
		}
		assignedClasses = append(assignedClasses, item)
	}

	// 5. Recent attendance sessions (top 5)
	attRows, err := q.GetTeacherRecentAttendanceSessions(ctx, db.GetTeacherRecentAttendanceSessionsParams{
		TeacherID: profRow.StaffID,
		Limit:     5,
	})
	if err != nil {
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get recent attendance: %w", err)
	}

	recentAttendance := make([]RecentAttendanceSession, 0, len(attRows))
	for _, a := range attRows {
		recentAttendance = append(recentAttendance, RecentAttendanceSession{
			SessionID:    a.SessionID.String(),
			SessionDate:  a.SessionDate.Time.Format("2006-01-02"),
			ClassID:      a.ClassID.String(),
			ClassName:    a.ClassName,
			SubjectID:    a.SubjectID.String(),
			SubjectName:  a.SubjectName,
			SubjectCode:  a.SubjectCode,
			TermName:     a.TermName,
			PresentCount: a.PresentCount,
			AbsentCount:  a.AbsentCount,
			LateCount:    a.LateCount,
			ExcusedCount: a.ExcusedCount,
			TotalMarks:   a.TotalMarks,
		})
	}

	// 6. Recent assessments (top 5)
	assRows, err := q.GetTeacherAssessmentsSummary(ctx, db.GetTeacherAssessmentsSummaryParams{
		TeacherID: profRow.StaffID,
		Limit:     5,
	})
	if err != nil {
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get recent assessments: %w", err)
	}

	recentAssessments := make([]RecentAssessment, 0, len(assRows))
	for _, a := range assRows {
		maxScore, _ := a.MaxScore.Float64Value()
		item := RecentAssessment{
			AssessmentID:       a.AssessmentID.String(),
			Title:              a.Title,
			AssessmentType:     a.AssessmentType,
			MaxScore:           maxScore.Float64,
			AssessmentDate:     a.AssessmentDate.Time.Format("2006-01-02"),
			ClassID:            a.ClassID.String(),
			ClassName:          a.ClassName,
			SubjectID:          a.SubjectID.String(),
			SubjectName:        a.SubjectName,
			SubjectCode:        a.SubjectCode,
			TermName:           a.TermName,
			ResultsCount:       a.ResultsCount,
			TotalStudentsCount: a.TotalStudentsCount,
		}
		if a.Weight.Valid {
			w, _ := a.Weight.Float64Value()
			val := w.Float64
			item.Weight = &val
		}
		recentAssessments = append(recentAssessments, item)
	}

	// 7. Teacher Announcements Feed (top 5)
	annRows, err := q.ListFeedAnnouncements(ctx, db.ListFeedAnnouncementsParams{
		AuthorID: userID,
		Limit:    5,
		Offset:   0,
	})
	if err != nil {
		return TeacherOverviewData{}, fmt.Errorf("teacherportal: get announcements: %w", err)
	}

	announcements := make([]TeacherAnnouncement, 0, len(annRows))
	for _, a := range annRows {
		authorName := strings.TrimSpace(a.AuthorFirstName + " " + a.AuthorLastName)
		item := TeacherAnnouncement{
			ID:         a.ID.String(),
			Title:      a.Title,
			Content:    a.Content,
			Priority:   a.Priority,
			Status:     a.Status,
			TargetType: a.TargetType,
			AuthorName: authorName,
			CreatedAt:  a.CreatedAt.Time.Format(time.RFC3339),
		}
		if a.PublishedAt.Valid {
			pub := a.PublishedAt.Time.Format(time.RFC3339)
			item.PublishedAt = &pub
		}
		announcements = append(announcements, item)
	}

	return TeacherOverviewData{
		Profile:           profile,
		AcademicContext:   academicCtx,
		Counts:            counts,
		AssignedClasses:   assignedClasses,
		RecentAttendance:  recentAttendance,
		RecentAssessments: recentAssessments,
		Announcements:     announcements,
	}, nil
}

func (s *Service) GetTeacherClasses(ctx context.Context, userID uuid.UUID) ([]AssignedClass, error) {
	q := db.New(s.pool)

	profRow, err := q.GetTeacherProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeacherNotFound
		}
		return nil, fmt.Errorf("teacherportal: get profile: %w", err)
	}

	classRows, err := q.GetTeacherClassesWithStats(ctx, profRow.StaffID)
	if err != nil {
		return nil, fmt.Errorf("teacherportal: get assigned classes: %w", err)
	}

	assignedClasses := make([]AssignedClass, 0, len(classRows))
	for _, c := range classRows {
		var capVal int32
		if c.ClassCapacity.Valid {
			capVal = c.ClassCapacity.Int32
		}
		item := AssignedClass{
			AssignmentID:          c.AssignmentID.String(),
			ClassID:               c.ClassID.String(),
			ClassName:             c.ClassName,
			ClassCapacity:         capVal,
			AcademicYearID:        c.AcademicYearID.String(),
			AcademicYearName:      c.AcademicYearName,
			SubjectID:             c.SubjectID.String(),
			SubjectName:           c.SubjectName,
			SubjectCode:           c.SubjectCode,
			SubjectType:           c.SubjectType,
			EnrolledStudentsCount: c.EnrolledStudentsCount,
		}
		if c.ClassLevel.Valid {
			lvl := c.ClassLevel.String
			item.ClassLevel = &lvl
		}
		assignedClasses = append(assignedClasses, item)
	}

	return assignedClasses, nil
}

func (s *Service) GetClassRoster(ctx context.Context, userID uuid.UUID, classID uuid.UUID) ([]StudentRosterItem, error) {
	q := db.New(s.pool)

	profRow, err := q.GetTeacherProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTeacherNotFound
		}
		return nil, fmt.Errorf("teacherportal: get profile: %w", err)
	}

	// Verify teacher assignment to this class
	isAssigned, err := q.IsTeacherAssignedToClass(ctx, db.IsTeacherAssignedToClassParams{
		TeacherID: profRow.StaffID,
		ClassID:   classID,
	})
	if err != nil {
		return nil, fmt.Errorf("teacherportal: check assignment: %w", err)
	}
	if !isAssigned {
		return nil, ErrForbiddenClassAccess
	}

	rows, err := q.GetClassRosterForTeacher(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("teacherportal: get roster: %w", err)
	}

	roster := make([]StudentRosterItem, 0, len(rows))
	for _, r := range rows {
		item := StudentRosterItem{
			StudentID:        r.StudentID.String(),
			StudentNumber:    r.StudentNumber,
			FirstName:        r.FirstName,
			LastName:         r.LastName,
			EnrollmentStatus: r.EnrollmentStatus,
			EnrolledAt:       r.EnrolledAt.Time.Format(time.RFC3339),
		}
		if r.Gender.Valid {
			g := r.Gender.String
			item.Gender = &g
		}
		if r.DateOfBirth.Valid {
			dob := r.DateOfBirth.Time.Format("2006-01-02")
			item.DateOfBirth = &dob
		}
		if r.GuardianFirstName.Valid || r.GuardianLastName.Valid {
			gName := strings.TrimSpace(r.GuardianFirstName.String + " " + r.GuardianLastName.String)
			item.GuardianName = &gName
		}
		if r.GuardianPhone.Valid {
			gp := r.GuardianPhone.String
			item.GuardianPhone = &gp
		}
		if r.GuardianEmail.Valid {
			ge := r.GuardianEmail.String
			item.GuardianEmail = &ge
		}
		if r.GuardianRelationship.Valid {
			gr := r.GuardianRelationship.String
			item.GuardianRelationship = &gr
		}
		roster = append(roster, item)
	}

	return roster, nil
}
