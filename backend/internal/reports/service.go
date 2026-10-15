// Package reports implements analytical and summary reports (milestone M18).
package reports

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

var (
	ErrClassNotFound        = errors.New("class not found")
	ErrStudentNotFound      = errors.New("student not found")
	ErrTermNotFound         = errors.New("term not found")
	ErrNoActiveEnrollment   = errors.New("student has no active enrollment for this term")
	ErrInvalidDateRange     = errors.New("start_date cannot be after end_date")
)

// Helper to convert percentage to letter grade.
func calculateGrade(pct float64) string {
	switch {
	case pct >= 90:
		return "A+"
	case pct >= 80:
		return "A"
	case pct >= 70:
		return "B"
	case pct >= 60:
		return "C"
	case pct >= 50:
		return "D"
	default:
		return "F"
	}
}

func textPtr(t pgtype.Text) *string {
	if t.Valid {
		v := t.String
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

func round(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// EnrollmentClassSummary models one class's enrollment numbers and demographics.
type EnrollmentClassSummary struct {
	ClassID          string   `json:"class_id"`
	ClassName        string   `json:"class_name"`
	ClassLevel       *string  `json:"class_level,omitempty"`
	Capacity         *int32   `json:"capacity,omitempty"`
	AcademicYearID   string   `json:"academic_year_id"`
	AcademicYearName string   `json:"academic_year_name"`
	ProgrammeName    *string  `json:"programme_name,omitempty"`
	EnrolledCount    int64    `json:"enrolled_count"`
	CompletedCount   int64    `json:"completed_count"`
	TransferredCount int64    `json:"transferred_count"`
	WithdrawnCount   int64    `json:"withdrawn_count"`
	MaleCount        int64    `json:"male_count"`
	FemaleCount      int64    `json:"female_count"`
	OtherGenderCount int64    `json:"other_gender_count"`
	UtilizationRate  *float64 `json:"utilization_rate,omitempty"`
}

// EnrollmentReport aggregates school-wide enrollment metrics across classes.
type EnrollmentReport struct {
	AcademicYearID     *string                  `json:"academic_year_id,omitempty"`
	AcademicYearName   *string                  `json:"academic_year_name,omitempty"`
	TotalClasses       int                      `json:"total_classes"`
	TotalEnrolled      int64                    `json:"total_enrolled"`
	TotalCapacity      int64                    `json:"total_capacity"`
	OverallUtilization *float64                 `json:"overall_utilization,omitempty"`
	TotalMale          int64                    `json:"total_male"`
	TotalFemale        int64                    `json:"total_female"`
	TotalOtherGender   int64                    `json:"total_other_gender"`
	Classes            []EnrollmentClassSummary `json:"classes"`
}

// ClassAttendanceStudentSummary represents one student's attendance metrics in a class.
type ClassAttendanceStudentSummary struct {
	StudentID      string  `json:"student_id"`
	StudentNumber  string  `json:"student_number"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	PresentCount   int64   `json:"present_count"`
	AbsentCount    int64   `json:"absent_count"`
	LateCount      int64   `json:"late_count"`
	ExcusedCount   int64   `json:"excused_count"`
	TotalRecords   int64   `json:"total_records"`
	AttendanceRate float64 `json:"attendance_rate"`
}

// ClassAttendanceReport is the full analytical output for class attendance.
type ClassAttendanceReport struct {
	ClassID               string                          `json:"class_id"`
	ClassName             string                          `json:"class_name"`
	TermID                *string                         `json:"term_id,omitempty"`
	TermName              *string                         `json:"term_name,omitempty"`
	StartDate             *string                         `json:"start_date,omitempty"`
	EndDate               *string                         `json:"end_date,omitempty"`
	TotalSessions         int64                           `json:"total_sessions"`
	TotalStudents         int                             `json:"total_students"`
	AverageAttendanceRate float64                         `json:"average_attendance_rate"`
	Students              []ClassAttendanceStudentSummary `json:"students"`
}

// AssessmentPerformanceItem is performance statistics for one assessment.
type AssessmentPerformanceItem struct {
	AssessmentID      string   `json:"assessment_id"`
	AssessmentTitle   string   `json:"assessment_title"`
	AssessmentType    string   `json:"assessment_type"`
	MaxScore          float64  `json:"max_score"`
	Weight            float64  `json:"weight"`
	AssessmentDate    string   `json:"assessment_date"`
	SubjectID         string   `json:"subject_id"`
	SubjectCode       string   `json:"subject_code"`
	SubjectName       string   `json:"subject_name"`
	ResultsCount      int64    `json:"results_count"`
	AverageScore      float64  `json:"average_score"`
	HighestScore      float64  `json:"highest_score"`
	LowestScore       float64  `json:"lowest_score"`
	AveragePercentage *float64 `json:"average_percentage,omitempty"`
}

// ClassPerformanceReport aggregates assessment performances across a class.
type ClassPerformanceReport struct {
	ClassID     string                      `json:"class_id"`
	ClassName   string                      `json:"class_name"`
	TermID      *string                     `json:"term_id,omitempty"`
	TermName    *string                     `json:"term_name,omitempty"`
	SubjectID   *string                     `json:"subject_id,omitempty"`
	SubjectName *string                     `json:"subject_name,omitempty"`
	Assessments []AssessmentPerformanceItem `json:"assessments"`
}

// ReportCardAttendance models attendance statistics on a student's report card.
type ReportCardAttendance struct {
	PresentCount   int64   `json:"present_count"`
	AbsentCount    int64   `json:"absent_count"`
	LateCount      int64   `json:"late_count"`
	ExcusedCount   int64   `json:"excused_count"`
	TotalSessions  int64   `json:"total_sessions"`
	AttendanceRate float64 `json:"attendance_rate"`
}

// ReportCardSubjectItem is one subject's aggregate performance on a student's report card.
type ReportCardSubjectItem struct {
	SubjectID         string  `json:"subject_id"`
	SubjectCode       string  `json:"subject_code"`
	SubjectName       string  `json:"subject_name"`
	SubjectType       string  `json:"subject_type"`
	AssessmentsTaken  int64   `json:"assessments_taken"`
	TotalScore        float64 `json:"total_score"`
	TotalMaxScore     float64 `json:"total_max_score"`
	AveragePercentage float64 `json:"average_percentage"`
	Grade             string  `json:"grade"`
}

// StudentReportCard is the terminal academic report card for a student in a term.
type StudentReportCard struct {
	StudentID                string                  `json:"student_id"`
	StudentNumber            string                  `json:"student_number"`
	FirstName                string                  `json:"first_name"`
	LastName                 string                  `json:"last_name"`
	Gender                   *string                 `json:"gender,omitempty"`
	DateOfBirth              *string                 `json:"date_of_birth,omitempty"`
	ClassID                  string                  `json:"class_id"`
	ClassName                string                  `json:"class_name"`
	ClassLevel               *string                 `json:"class_level,omitempty"`
	AcademicYearID           string                  `json:"academic_year_id"`
	AcademicYearName         string                  `json:"academic_year_name"`
	TermID                   string                  `json:"term_id"`
	TermName                 string                  `json:"term_name"`
	Attendance               ReportCardAttendance    `json:"attendance"`
	OverallAveragePercentage *float64                `json:"overall_average_percentage,omitempty"`
	OverallGrade             *string                 `json:"overall_grade,omitempty"`
	Subjects                 []ReportCardSubjectItem `json:"subjects"`
}

// Service provides reports data access and analytical aggregation.
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// GetEnrollmentReport returns the school-wide enrollment metrics and class utilization.
func (s *Service) GetEnrollmentReport(ctx context.Context, academicYearID uuid.UUID) (EnrollmentReport, error) {
	q := db.New(s.pool)

	var ayParam pgtype.UUID
	if academicYearID != uuid.Nil {
		ayParam = pgtype.UUID{Bytes: academicYearID, Valid: true}
	}

	rows, err := q.GetEnrollmentReport(ctx, ayParam)
	if err != nil {
		return EnrollmentReport{}, fmt.Errorf("reports: get enrollment report: %w", err)
	}

	report := EnrollmentReport{
		Classes: make([]EnrollmentClassSummary, 0, len(rows)),
	}

	var totalEnrolled, totalCapacity, totalMale, totalFemale, totalOther int64
	var yearIDStr, yearNameStr string

	for _, r := range rows {
		totalEnrolled += r.EnrolledCount
		totalMale += r.MaleCount
		totalFemale += r.FemaleCount
		totalOther += r.OtherGenderCount

		summary := EnrollmentClassSummary{
			ClassID:          r.ClassID.String(),
			ClassName:        r.ClassName,
			AcademicYearID:   r.AcademicYearID.String(),
			AcademicYearName: r.AcademicYearName,
			EnrolledCount:    r.EnrolledCount,
			CompletedCount:   r.CompletedCount,
			TransferredCount: r.TransferredCount,
			WithdrawnCount:   r.WithdrawnCount,
			MaleCount:        r.MaleCount,
			FemaleCount:      r.FemaleCount,
			OtherGenderCount: r.OtherGenderCount,
		}

		if yearIDStr == "" {
			yearIDStr = r.AcademicYearID.String()
			yearNameStr = r.AcademicYearName
		}

		if r.ClassLevel.Valid {
			lvl := r.ClassLevel.String
			summary.ClassLevel = &lvl
		}
		if r.Capacity.Valid {
			capVal := r.Capacity.Int32
			summary.Capacity = &capVal
			totalCapacity += int64(capVal)
			if capVal > 0 {
				util := round((float64(r.EnrolledCount)/float64(capVal))*100, 1)
				summary.UtilizationRate = &util
			}
		}
		if r.ProgrammeName != "" {
			pName := r.ProgrammeName
			summary.ProgrammeName = &pName
		}

		report.Classes = append(report.Classes, summary)
	}

	report.TotalClasses = len(report.Classes)
	report.TotalEnrolled = totalEnrolled
	report.TotalCapacity = totalCapacity
	report.TotalMale = totalMale
	report.TotalFemale = totalFemale
	report.TotalOtherGender = totalOther

	if yearIDStr != "" {
		report.AcademicYearID = &yearIDStr
		report.AcademicYearName = &yearNameStr
	}

	if totalCapacity > 0 {
		overallUtil := round((float64(totalEnrolled)/float64(totalCapacity))*100, 1)
		report.OverallUtilization = &overallUtil
	}

	return report, nil
}

// GetClassAttendanceReport generates attendance metrics for all students in a class.
func (s *Service) GetClassAttendanceReport(ctx context.Context, classID, termID uuid.UUID, startDate, endDate string) (ClassAttendanceReport, error) {
	q := db.New(s.pool)

	// Verify class existence
	classRow, err := q.GetClassByID(ctx, classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClassAttendanceReport{}, ErrClassNotFound
		}
		return ClassAttendanceReport{}, fmt.Errorf("reports: verify class: %w", err)
	}

	var termParam pgtype.UUID
	var termNamePtr *string
	if termID != uuid.Nil {
		termParam = pgtype.UUID{Bytes: termID, Valid: true}
		termRow, err := q.GetTermByID(ctx, termID)
		if err == nil {
			tName := termRow.Name
			termNamePtr = &tName
		}
	}

	var startParam, endParam pgtype.Date
	if startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return ClassAttendanceReport{}, fmt.Errorf("invalid start_date format, expected YYYY-MM-DD: %w", err)
		}
		startParam = pgtype.Date{Time: t, Valid: true}
	}
	if endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return ClassAttendanceReport{}, fmt.Errorf("invalid end_date format, expected YYYY-MM-DD: %w", err)
		}
		endParam = pgtype.Date{Time: t, Valid: true}
	}

	if startParam.Valid && endParam.Valid && startParam.Time.After(endParam.Time) {
		return ClassAttendanceReport{}, ErrInvalidDateRange
	}

	// Session count
	sessionCount, err := q.GetClassAttendanceSessionCount(ctx, db.GetClassAttendanceSessionCountParams{
		ClassID:   classID,
		TermID:    termParam,
		StartDate: startParam,
		EndDate:   endParam,
	})
	if err != nil {
		return ClassAttendanceReport{}, fmt.Errorf("reports: get session count: %w", err)
	}

	// Students breakdown
	rows, err := q.GetClassAttendanceSummaryReport(ctx, db.GetClassAttendanceSummaryReportParams{
		ClassID:   classID,
		TermID:    termParam,
		StartDate: startParam,
		EndDate:   endParam,
	})
	if err != nil {
		return ClassAttendanceReport{}, fmt.Errorf("reports: get student attendance breakdown: %w", err)
	}

	report := ClassAttendanceReport{
		ClassID:       classRow.ID.String(),
		ClassName:     classRow.Name,
		TotalSessions: sessionCount,
		TotalStudents: len(rows),
		Students:      make([]ClassAttendanceStudentSummary, 0, len(rows)),
	}

	if termID != uuid.Nil {
		tIDStr := termID.String()
		report.TermID = &tIDStr
		report.TermName = termNamePtr
	}
	if startDate != "" {
		report.StartDate = &startDate
	}
	if endDate != "" {
		report.EndDate = &endDate
	}

	var rateSum float64
	for _, r := range rows {
		totalMarks := r.PresentCount + r.AbsentCount + r.LateCount + r.ExcusedCount
		var rate float64
		if totalMarks > 0 {
			rate = round((float64(r.PresentCount)/float64(totalMarks))*100, 1)
		}
		rateSum += rate

		report.Students = append(report.Students, ClassAttendanceStudentSummary{
			StudentID:      r.StudentID.String(),
			StudentNumber:  r.StudentNumber,
			FirstName:      r.FirstName,
			LastName:       r.LastName,
			PresentCount:   r.PresentCount,
			AbsentCount:    r.AbsentCount,
			LateCount:      r.LateCount,
			ExcusedCount:   r.ExcusedCount,
			TotalRecords:   r.TotalRecords,
			AttendanceRate: rate,
		})
	}

	if len(rows) > 0 {
		report.AverageAttendanceRate = round(rateSum/float64(len(rows)), 1)
	}

	return report, nil
}

// GetClassPerformanceReport aggregates assessment performances across a class and term.
func (s *Service) GetClassPerformanceReport(ctx context.Context, classID, termID, subjectID uuid.UUID) (ClassPerformanceReport, error) {
	q := db.New(s.pool)

	classRow, err := q.GetClassByID(ctx, classID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClassPerformanceReport{}, ErrClassNotFound
		}
		return ClassPerformanceReport{}, fmt.Errorf("reports: verify class: %w", err)
	}

	var termParam, subParam pgtype.UUID
	var termNamePtr, subNamePtr *string

	if termID != uuid.Nil {
		termParam = pgtype.UUID{Bytes: termID, Valid: true}
		termRow, err := q.GetTermByID(ctx, termID)
		if err == nil {
			tName := termRow.Name
			termNamePtr = &tName
		}
	}
	if subjectID != uuid.Nil {
		subParam = pgtype.UUID{Bytes: subjectID, Valid: true}
		subRow, err := q.GetSubjectByID(ctx, subjectID)
		if err == nil {
			sName := subRow.Name
			subNamePtr = &sName
		}
	}

	rows, err := q.GetClassAssessmentPerformanceReport(ctx, db.GetClassAssessmentPerformanceReportParams{
		ClassID:   classID,
		TermID:    termParam,
		SubjectID: subParam,
	})
	if err != nil {
		return ClassPerformanceReport{}, fmt.Errorf("reports: get assessment performance: %w", err)
	}

	report := ClassPerformanceReport{
		ClassID:     classRow.ID.String(),
		ClassName:   classRow.Name,
		Assessments: make([]AssessmentPerformanceItem, 0, len(rows)),
	}

	if termID != uuid.Nil {
		tIDStr := termID.String()
		report.TermID = &tIDStr
		report.TermName = termNamePtr
	}
	if subjectID != uuid.Nil {
		sIDStr := subjectID.String()
		report.SubjectID = &sIDStr
		report.SubjectName = subNamePtr
	}

	for _, r := range rows {
		item := AssessmentPerformanceItem{
			AssessmentID:    r.AssessmentID.String(),
			AssessmentTitle: r.AssessmentTitle,
			AssessmentType:  r.AssessmentType,
			AssessmentDate:  r.AssessmentDate.Time.Format("2006-01-02"),
			SubjectID:       r.SubjectID.String(),
			SubjectCode:     r.SubjectCode,
			SubjectName:     r.SubjectName,
			ResultsCount:    r.ResultsCount,
			AverageScore:    round(r.AverageScore, 2),
			HighestScore:    round(r.HighestScore, 2),
			LowestScore:     round(r.LowestScore, 2),
		}

		if maxScore, ok := numericFloat(r.MaxScore); ok {
			item.MaxScore = maxScore
			if maxScore > 0 {
				avgPct := round((r.AverageScore/maxScore)*100, 1)
				item.AveragePercentage = &avgPct
			}
		}
		if weight, ok := numericFloat(r.Weight); ok {
			item.Weight = weight
		}

		report.Assessments = append(report.Assessments, item)
	}

	return report, nil
}

// GetStudentReportCard produces the terminal transcript summary report card.
func (s *Service) GetStudentReportCard(ctx context.Context, studentID, termID uuid.UUID) (StudentReportCard, error) {
	q := db.New(s.pool)

	prof, err := q.GetStudentReportCardProfile(ctx, db.GetStudentReportCardProfileParams{
		StudentID: studentID,
		TermID:    termID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if student exists
			_, sErr := q.GetStudentByID(ctx, studentID)
			if sErr != nil {
				return StudentReportCard{}, ErrStudentNotFound
			}
			return StudentReportCard{}, ErrNoActiveEnrollment
		}
		return StudentReportCard{}, fmt.Errorf("reports: get report card profile: %w", err)
	}

	// Attendance in term
	attRow, err := q.GetStudentAttendanceForTerm(ctx, db.GetStudentAttendanceForTermParams{
		StudentID: studentID,
		TermID:    termID,
	})
	if err != nil {
		return StudentReportCard{}, fmt.Errorf("reports: get attendance for term: %w", err)
	}

	totalMarks := attRow.PresentCount + attRow.AbsentCount + attRow.LateCount + attRow.ExcusedCount
	var attRate float64
	if totalMarks > 0 {
		attRate = round((float64(attRow.PresentCount)/float64(totalMarks))*100, 1)
	}

	att := ReportCardAttendance{
		PresentCount:   attRow.PresentCount,
		AbsentCount:    attRow.AbsentCount,
		LateCount:      attRow.LateCount,
		ExcusedCount:   attRow.ExcusedCount,
		TotalSessions:  attRow.TotalSessions,
		AttendanceRate: attRate,
	}

	// Subject results
	subRows, err := q.GetStudentSubjectResultsForReportCard(ctx, db.GetStudentSubjectResultsForReportCardParams{
		TermID:    termID,
		StudentID: studentID,
		ClassID:   prof.ClassID,
	})
	if err != nil {
		return StudentReportCard{}, fmt.Errorf("reports: get subject results for report card: %w", err)
	}

	subjects := make([]ReportCardSubjectItem, 0, len(subRows))
	var pctSum float64
	var subjectsWithAssessments int

	for _, sub := range subRows {
		avgPct := round(sub.AveragePercentage, 1)
		grade := calculateGrade(avgPct)
		if sub.AssessmentsTaken == 0 {
			grade = "N/A"
		} else {
			pctSum += avgPct
			subjectsWithAssessments++
		}

		subjects = append(subjects, ReportCardSubjectItem{
			SubjectID:         sub.SubjectID.String(),
			SubjectCode:       sub.SubjectCode,
			SubjectName:       sub.SubjectName,
			SubjectType:       sub.SubjectType,
			AssessmentsTaken:  sub.AssessmentsTaken,
			TotalScore:        round(sub.TotalScore, 2),
			TotalMaxScore:     round(sub.TotalMaxScore, 2),
			AveragePercentage: avgPct,
			Grade:             grade,
		})
	}

	reportCard := StudentReportCard{
		StudentID:        prof.StudentID.String(),
		StudentNumber:    prof.StudentNumber,
		FirstName:        prof.FirstName,
		LastName:         prof.LastName,
		ClassID:          prof.ClassID.String(),
		ClassName:        prof.ClassName,
		AcademicYearID:   prof.AcademicYearID.String(),
		AcademicYearName: prof.AcademicYearName,
		TermID:           prof.TermID.String(),
		TermName:         prof.TermName,
		Attendance:       att,
		Subjects:         subjects,
	}

	reportCard.Gender = textPtr(prof.Gender)
	reportCard.DateOfBirth = dateStr(prof.DateOfBirth)
	reportCard.ClassLevel = textPtr(prof.ClassLevel)

	if subjectsWithAssessments > 0 {
		overallAvg := round(pctSum/float64(subjectsWithAssessments), 1)
		reportCard.OverallAveragePercentage = &overallAvg
		overallGrade := calculateGrade(overallAvg)
		reportCard.OverallGrade = &overallGrade
	}

	return reportCard, nil
}
