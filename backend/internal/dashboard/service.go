// Package dashboard implements the admin dashboard domain (milestone M14).
package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

type DashboardCounts struct {
	ActiveStudents    int64 `json:"active_students"`
	ActiveStaff       int64 `json:"active_staff"`
	ActiveTeachers    int64 `json:"active_teachers"`
	TotalClasses      int64 `json:"total_classes"`
	ActiveEnrollments int64 `json:"active_enrollments"`
}

type AcademicContext struct {
	AcademicYearID   *string `json:"academic_year_id,omitempty"`
	AcademicYearName *string `json:"academic_year_name,omitempty"`
	TermID           *string `json:"term_id,omitempty"`
	TermName         *string `json:"term_name,omitempty"`
}

type AttendanceSummary struct {
	Date           string  `json:"date"`
	TotalSessions  int64   `json:"total_sessions"`
	PresentCount   int64   `json:"present_count"`
	AbsentCount    int64   `json:"absent_count"`
	LateCount      int64   `json:"late_count"`
	ExcusedCount   int64   `json:"excused_count"`
	TotalMarks     int64   `json:"total_marks"`
	AttendanceRate float64 `json:"attendance_rate"`
}

type RecentAnnouncement struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Status          string  `json:"status"`
	TargetType      string  `json:"target_type"`
	TargetRole      *string `json:"target_role,omitempty"`
	TargetClassName *string `json:"target_class_name,omitempty"`
	Priority        string  `json:"priority"`
	PublishedAt     *string `json:"published_at,omitempty"`
	CreatedAt       string  `json:"created_at"`
	AuthorName      string  `json:"author_name"`
}

type RecentActivity struct {
	ID           string         `json:"id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	ActorEmail   string         `json:"actor_email"`
	ActorName    string         `json:"actor_name"`
	CreatedAt    string         `json:"created_at"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type QuickAction struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Href        string `json:"href"`
	Category    string `json:"category"`
}

type AdminDashboardData struct {
	Counts              DashboardCounts      `json:"counts"`
	AcademicContext     AcademicContext      `json:"academic_context"`
	Attendance          AttendanceSummary    `json:"attendance"`
	RecentAnnouncements []RecentAnnouncement `json:"recent_announcements"`
	RecentActivity      []RecentActivity     `json:"recent_activity"`
	QuickActions        []QuickAction        `json:"quick_actions"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

var defaultQuickActions = []QuickAction{
	{
		ID:          "students_new",
		Title:       "Register Student",
		Description: "Add a new student profile and link guardians",
		Href:        "/students/new",
		Category:    "STUDENTS",
	},
	{
		ID:          "staff_new",
		Title:       "Add Staff Member",
		Description: "Create a new teaching or administrative staff record",
		Href:        "/staff/new",
		Category:    "STAFF",
	},
	{
		ID:          "attendance_record",
		Title:       "Record Attendance",
		Description: "Take class attendance register for today",
		Href:        "/attendance",
		Category:    "ACADEMICS",
	},
	{
		ID:          "assessments_new",
		Title:       "Create Assessment",
		Description: "Schedule tests, assignments, or midterm exams",
		Href:        "/assessments",
		Category:    "ACADEMICS",
	},
	{
		ID:          "announcements_new",
		Title:       "New Announcement",
		Description: "Broadcast an announcement school-wide or to specific classes",
		Href:        "/announcements",
		Category:    "COMMUNICATION",
	},
	{
		ID:          "classes_manage",
		Title:       "Manage Classes",
		Description: "View and configure class cohorts and subject allocations",
		Href:        "/classes",
		Category:    "ACADEMICS",
	},
	{
		ID:          "audit_logs",
		Title:       "Audit Logs",
		Description: "Review administrative actions and security log trail",
		Href:        "/audit",
		Category:    "ADMIN",
	},
}

func (s *Service) GetAdminDashboard(ctx context.Context) (AdminDashboardData, error) {
	q := db.New(s.pool)

	// 1. Institutional Counts
	countsRow, err := q.GetAdminDashboardCounts(ctx)
	if err != nil {
		return AdminDashboardData{}, fmt.Errorf("dashboard: get counts: %w", err)
	}

	counts := DashboardCounts{
		ActiveStudents:    countsRow.ActiveStudents,
		ActiveStaff:       countsRow.ActiveStaff,
		ActiveTeachers:    countsRow.ActiveTeachers,
		TotalClasses:      countsRow.TotalClasses,
		ActiveEnrollments: countsRow.ActiveEnrollments,
	}

	// 2. Current Academic Context
	var academicCtx AcademicContext
	ayRow, err := q.GetCurrentAcademicYearAndTerm(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AdminDashboardData{}, fmt.Errorf("dashboard: get academic context: %w", err)
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

	// 3. Attendance Summary
	attRow, err := q.GetLatestAttendanceSummary(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AdminDashboardData{}, fmt.Errorf("dashboard: get attendance summary: %w", err)
	}

	var attSummary AttendanceSummary
	if err == nil {
		dateStr := time.Now().UTC().Format("2006-01-02")
		if attRow.SessionDate.Valid {
			dateStr = attRow.SessionDate.Time.Format("2006-01-02")
		}
		attSummary.Date = dateStr
		attSummary.TotalSessions = attRow.TotalSessions
		attSummary.PresentCount = attRow.PresentCount
		attSummary.AbsentCount = attRow.AbsentCount
		attSummary.LateCount = attRow.LateCount
		attSummary.ExcusedCount = attRow.ExcusedCount
		attSummary.TotalMarks = attRow.TotalMarks

		if attRow.TotalMarks > 0 {
			rate := (float64(attRow.PresentCount+attRow.LateCount) / float64(attRow.TotalMarks)) * 100.0
			attSummary.AttendanceRate = math.Round(rate*10) / 10
		}
	} else {
		attSummary.Date = time.Now().UTC().Format("2006-01-02")
	}

	// 4. Recent Announcements (top 5)
	annRows, err := q.GetRecentAnnouncementsForDashboard(ctx, 5)
	if err != nil {
		return AdminDashboardData{}, fmt.Errorf("dashboard: get recent announcements: %w", err)
	}
	recentAnnouncements := make([]RecentAnnouncement, 0, len(annRows))
	for _, a := range annRows {
		authorName := strings.TrimSpace(a.AuthorFirstName + " " + a.AuthorLastName)
		item := RecentAnnouncement{
			ID:         a.ID.String(),
			Title:      a.Title,
			Status:     a.Status,
			TargetType: a.TargetType,
			Priority:   a.Priority,
			CreatedAt:  a.CreatedAt.Time.Format(time.RFC3339),
			AuthorName: authorName,
		}
		if a.TargetRole.Valid {
			r := a.TargetRole.String
			item.TargetRole = &r
		}
		if a.TargetClassName.Valid {
			cn := a.TargetClassName.String
			item.TargetClassName = &cn
		}
		if a.PublishedAt.Valid {
			p := a.PublishedAt.Time.Format(time.RFC3339)
			item.PublishedAt = &p
		}
		recentAnnouncements = append(recentAnnouncements, item)
	}

	// 5. Recent Activity / Audit Logs (top 10)
	auditRows, err := q.GetRecentAuditLogsForDashboard(ctx, 10)
	if err != nil {
		return AdminDashboardData{}, fmt.Errorf("dashboard: get recent audit logs: %w", err)
	}
	recentActivity := make([]RecentActivity, 0, len(auditRows))
	for _, l := range auditRows {
		actorName := strings.TrimSpace(l.ActorFirstName + " " + l.ActorLastName)
		if actorName == "" {
			actorName = l.ActorEmail
		}
		act := RecentActivity{
			ID:           l.ID.String(),
			Action:       l.Action,
			ResourceType: l.ResourceType,
			ActorEmail:   l.ActorEmail,
			ActorName:    actorName,
			CreatedAt:    l.CreatedAt.Time.Format(time.RFC3339),
		}
		if l.ResourceID.Valid {
			resID := uuid.UUID(l.ResourceID.Bytes).String()
			act.ResourceID = &resID
		}
		if len(l.Metadata) > 0 {
			var meta map[string]any
			if err := json.Unmarshal(l.Metadata, &meta); err == nil {
				act.Metadata = meta
			}
		}
		recentActivity = append(recentActivity, act)
	}

	return AdminDashboardData{
		Counts:              counts,
		AcademicContext:     academicCtx,
		Attendance:          attSummary,
		RecentAnnouncements: recentAnnouncements,
		RecentActivity:      recentActivity,
		QuickActions:        defaultQuickActions,
	}, nil
}
