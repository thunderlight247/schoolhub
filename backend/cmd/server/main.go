// Command server is the entrypoint for the SchoolHub API. It loads config,
// connects to PostgreSQL, wires up each domain's handlers, and starts the
// HTTP server. Each milestone adds its own wiring here as new domains gain
// real implementations (M4: auth; M5: RBAC middleware; M6+: everything
// else) — this file stays a thin composition root, not where business
// logic lives.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/thunderlight247/schoolhub/backend/config"
	db "github.com/thunderlight247/schoolhub/backend/db/generated"
	"github.com/thunderlight247/schoolhub/backend/internal/academics"
	"github.com/thunderlight247/schoolhub/backend/internal/adminaccounts"
	"github.com/thunderlight247/schoolhub/backend/internal/announcements"
	"github.com/thunderlight247/schoolhub/backend/internal/assessments"
	"github.com/thunderlight247/schoolhub/backend/internal/assignments"
	"github.com/thunderlight247/schoolhub/backend/internal/attendance"
	"github.com/thunderlight247/schoolhub/backend/internal/audit"
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
	"github.com/thunderlight247/schoolhub/backend/internal/dashboard"
	"github.com/thunderlight247/schoolhub/backend/internal/database"
	"github.com/thunderlight247/schoolhub/backend/internal/email"
	"github.com/thunderlight247/schoolhub/backend/internal/enrollment"
	"github.com/thunderlight247/schoolhub/backend/internal/parentportal"
	"github.com/thunderlight247/schoolhub/backend/internal/ratelimit"
	"github.com/thunderlight247/schoolhub/backend/internal/rbac"
	"github.com/thunderlight247/schoolhub/backend/internal/reports"
	"github.com/thunderlight247/schoolhub/backend/internal/staff"
	"github.com/thunderlight247/schoolhub/backend/internal/studentportal"
	"github.com/thunderlight247/schoolhub/backend/internal/students"
	"github.com/thunderlight247/schoolhub/backend/internal/teacherportal"
)

func main() {
	// godotenv.Load is a no-op (returns an error we ignore) if no .env file
	// is present, e.g. in production where env vars are injected directly.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)
	authService := auth.NewService(queries)

	const sessionCookieName = "schoolhub_session"
	authHandler := auth.NewHandler(authService, auth.CookieConfig{
		Name:     sessionCookieName,
		Domain:   cfg.CookieDomain,
		Secure:   cfg.IsProduction(),
		SameSite: cfg.CookieSameSiteMode(),
	})

	// Optional Clerk bridge (hybrid auth): when CLERK_SECRET_KEY is set the
	// frontend can exchange a verified Clerk session JWT for our session
	// cookie. Classic email/password login always remains available.
	clerkVerifier := auth.NewClerkVerifier(cfg.ClerkSecretKey)
	clerkAuthHandler := auth.NewHandlerWithClerk(authService, auth.CookieConfig{
		Name:     sessionCookieName,
		Domain:   cfg.CookieDomain,
		Secure:   cfg.IsProduction(),
		SameSite: cfg.CookieSameSiteMode(),
	}, clerkVerifier)

	// M34: forced first-login password change + the Clerk-mode counterpart
	// that just clears the flag once the frontend rotated the password in
	// Clerk's own credential store.
	passwordChangeService := auth.NewPasswordChangeService(pool)
	passwordChangeHandler := auth.NewChangeHandler(passwordChangeService)
	clerkPasswordChangeHandler := auth.NewClerkPasswordChangedHandler(passwordChangeService)

	rbacService := rbac.NewService(queries)
	rbacHandler := rbac.NewHandler(queries)

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(corsMiddleware(cfg.FrontendURL))
	router.Use(securityHeadersMiddleware(cfg.IsProduction()))

	// /api/v1/health lets the frontend, deployment platform, and load
	// balancers all verify the API is up and can reach the database.
	router.GET("/api/v1/health", func(c *gin.Context) {
		healthCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(healthCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "error",
				"database": "unreachable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "connected",
		})
	})

	// M20: 10 login attempts per minute per IP. Scoped to login only — see
	// internal/ratelimit's package comment for why not applied more
	// broadly.
	loginLimiter := ratelimit.New(10, time.Minute)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/login", loginLimiter.Middleware(), authHandler.Login)
		v1.POST("/auth/logout", authHandler.Logout)
		v1.GET("/me", auth.RequireAuth(authService, sessionCookieName), authHandler.Me)
		if clerkVerifier.Enabled() {
			v1.POST("/auth/clerk-session", clerkAuthHandler.ClerkSession)
		}

		// M34 forced first-login password change. ChangePassword re-checks
		// the CURRENT password, so it shares the login rate limiter's per-IP
		// bucket; ClerkPasswordChanged requires only a valid session (Clerk
		// already proved identity) and needs no limiter. Both live behind
		// RequireAuth, whose PASSWORD_CHANGE_REQUIRED allowlist lets flagged
		// accounts reach these three endpoints and nothing else.
		requireAuth := auth.RequireAuth(authService, sessionCookieName)
		v1.POST("/auth/change-password", requireAuth, loginLimiter.Middleware(), passwordChangeHandler.ChangePassword)
		v1.POST("/auth/clerk-password-changed", requireAuth, clerkPasswordChangeHandler.ClerkPasswordChanged)
	}

	// RBAC management: viewing roles/permissions and customizing which
	// permissions a role holds (on top of the default mapping migration
	// 000004 seeds). Gated on the ADMIN role directly, not a permission
	// code — see rbac.RequireRole's doc comment for why.
	rbacRoutes := v1.Group("")
	rbacRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	rbacRoutes.Use(rbac.RequireRole(rbacService, "ADMIN"))
	{
		rbacRoutes.GET("/roles", rbacHandler.ListRoles)
		rbacRoutes.GET("/permissions", rbacHandler.ListPermissions)
		rbacRoutes.GET("/roles/:id/permissions", rbacHandler.ListRolePermissions)
		rbacRoutes.POST("/roles/:id/permissions", rbacHandler.AssignPermission)
		rbacRoutes.DELETE("/roles/:id/permissions/:permission_id", rbacHandler.RevokePermission)
	}

	// Staff management (M6): the first real consumers of RequirePermission.
	// All routes sit behind RequireAuth (the audit trail needs to know who
	// acted); each group then gates on the specific staff.* permission code
	// from section 11. staff.deactivate is deliberately stricter than
	// staff.update — suspending someone's employment is a bigger deal than
	// fixing their phone number.
	staffService := staff.NewService(pool)

	// Optional outbound email (Resend). Every mail path is best-effort:
	// providers are enhancement, never a hard dependency.
	mailer := email.New(cfg.ResendAPIKey, cfg.EmailFromAddress)
	staffService.SetMailer(mailer, cfg.FrontendURL)
	// Same optional Clerk invitation as the parent flow: provisioning a
	// staff/teacher account emails the official Clerk invite.
	staffService.SetInviter(auth.NewClerkInviter(cfg.ClerkSecretKey, cfg.FrontendURL+"/register"))
	staffHandler := staff.NewHandler(staffService)

	staffRoutes := v1.Group("")
	staffRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewStaff := staffRoutes.Group("")
		viewStaff.Use(rbac.RequirePermission(rbacService, "staff.view"))
		{
			viewStaff.GET("/staff", staffHandler.ListStaff)
			viewStaff.GET("/staff/:id", staffHandler.GetStaff)
			viewStaff.GET("/departments", staffHandler.ListDepartments)
			viewStaff.GET("/staff-positions", staffHandler.ListStaffPositions)
		}

		createStaff := staffRoutes.Group("")
		createStaff.Use(rbac.RequirePermission(rbacService, "staff.create"))
		{
			createStaff.POST("/staff", staffHandler.CreateStaff)
		}

		updateStaff := staffRoutes.Group("")
		updateStaff.Use(rbac.RequirePermission(rbacService, "staff.update"))
		{
			updateStaff.PATCH("/staff/:id", staffHandler.UpdateStaff)
			updateStaff.PUT("/staff/:id/assignment", staffHandler.SetAssignment)
			updateStaff.POST("/staff/:id/account", staffHandler.CreateAccount)
			updateStaff.PUT("/staff/:id/account/roles", staffHandler.SetAccountRoles)
			updateStaff.POST("/departments", staffHandler.CreateDepartment)
			updateStaff.POST("/staff-positions", staffHandler.CreateStaffPosition)
		}

		deactivateStaff := staffRoutes.Group("")
		deactivateStaff.Use(rbac.RequirePermission(rbacService, "staff.deactivate"))
		{
			deactivateStaff.PUT("/staff/:id/status", staffHandler.SetEmploymentStatus)
		}
	}

	// Students and guardians management (M7).
	// All routes sit behind RequireAuth; each group gates on the specific
	// student.* permission code. Guardians reuse the student.* permissions
	// (section 11 has no separate guardian.* codes).
	studentService := students.NewService(pool)
	studentService.SetMailer(mailer, cfg.FrontendURL)
	// Optional Clerk invitations: when configured, provisioning a parent or
	// staff account also emails an official Clerk "create your password"
	// invitation for the same address (best-effort).
	studentService.SetInviter(auth.NewClerkInviter(cfg.ClerkSecretKey, cfg.FrontendURL+"/register"))
	studentHandler := students.NewHandler(studentService)

	studentRoutes := v1.Group("")
	studentRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewStudents := studentRoutes.Group("")
		viewStudents.Use(rbac.RequirePermission(rbacService, "student.view"))
		{
			viewStudents.GET("/students", studentHandler.ListStudents)
			viewStudents.GET("/students/:id", studentHandler.GetStudent)
			viewStudents.GET("/guardians", studentHandler.ListGuardians)
			viewStudents.GET("/guardians/:id", studentHandler.GetGuardian)
		}

		createStudents := studentRoutes.Group("")
		createStudents.Use(rbac.RequirePermission(rbacService, "student.create"))
		{
			createStudents.POST("/students", studentHandler.CreateStudent)
		}

		// Inviting a guardian is its own power (M34: "teachers invite
		// parents") — separate from student minting, which stays admin-only.
		createGuardians := studentRoutes.Group("")
		createGuardians.Use(rbac.RequirePermission(rbacService, "guardian.create"))
		{
			createGuardians.POST("/guardians", studentHandler.CreateGuardian)
		}

		// Parent access provisioning (expert-pass follow-up): the "Create
		// parent access" action on a guardian's profile. Retitles to the
		// guardian.create code in M34 so the INVITER role (teacher) can
		// provision the parent too — admins keep it via the same code.
		provisionParents := studentRoutes.Group("")
		provisionParents.Use(rbac.RequirePermission(rbacService, "guardian.create"))
		{
			provisionParents.POST("/guardians/:id/account", studentHandler.CreateGuardianAccount)
		}

		updateStudents := studentRoutes.Group("")
		updateStudents.Use(rbac.RequirePermission(rbacService, "student.update"))
		{
			updateStudents.PATCH("/students/:id", studentHandler.UpdateStudent)
			updateStudents.POST("/students/:id/guardians", studentHandler.LinkGuardian)
			updateStudents.DELETE("/students/:id/guardians/:guardian_id", studentHandler.UnlinkGuardian)
			updateStudents.PATCH("/guardians/:id", studentHandler.UpdateGuardian)
		}

		archiveStudents := studentRoutes.Group("")
		archiveStudents.Use(rbac.RequirePermission(rbacService, "student.archive"))
		{
			archiveStudents.PUT("/students/:id/status", studentHandler.SetStudentStatus)
		}
	}

	// Academic structure management (M8).
	// Gated on class.view (read), class.create (create), and class.update (mutate/assign).
	academicsService := academics.NewService(pool)
	academicsHandler := academics.NewHandler(academicsService)

	academicRoutes := v1.Group("")
	academicRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewAcademics := academicRoutes.Group("")
		viewAcademics.Use(rbac.RequirePermission(rbacService, "class.view"))
		{
			// Academic Years
			viewAcademics.GET("/academic-years", academicsHandler.ListAcademicYears)
			viewAcademics.GET("/academic-years/current", academicsHandler.GetCurrentAcademicYear)
			viewAcademics.GET("/academic-years/:id", academicsHandler.GetAcademicYear)

			// Terms
			viewAcademics.GET("/terms", academicsHandler.ListTerms)
			viewAcademics.GET("/terms/current", academicsHandler.GetCurrentTerm)
			viewAcademics.GET("/terms/:id", academicsHandler.GetTerm)

			// Programmes
			viewAcademics.GET("/programmes", academicsHandler.ListProgrammes)
			viewAcademics.GET("/programmes/:id", academicsHandler.GetProgramme)

			// Subjects
			viewAcademics.GET("/subjects", academicsHandler.ListSubjects)
			viewAcademics.GET("/subjects/:id", academicsHandler.GetSubject)

			// Classes
			viewAcademics.GET("/classes", academicsHandler.ListClasses)
			viewAcademics.GET("/classes/:id", academicsHandler.GetClass)
			viewAcademics.GET("/classes/:id/subjects", academicsHandler.ListSubjectsForClass)
		}

		createAcademics := academicRoutes.Group("")
		createAcademics.Use(rbac.RequirePermission(rbacService, "class.create"))
		{
			createAcademics.POST("/academic-years", academicsHandler.CreateAcademicYear)
			createAcademics.POST("/terms", academicsHandler.CreateTerm)
			createAcademics.POST("/programmes", academicsHandler.CreateProgramme)
			createAcademics.POST("/subjects", academicsHandler.CreateSubject)
			createAcademics.POST("/classes", academicsHandler.CreateClass)
		}

		updateAcademics := academicRoutes.Group("")
		updateAcademics.Use(rbac.RequirePermission(rbacService, "class.update"))
		{
			updateAcademics.PATCH("/academic-years/:id", academicsHandler.UpdateAcademicYear)
			updateAcademics.PUT("/academic-years/:id/current", academicsHandler.SetCurrentAcademicYear)
			updateAcademics.PATCH("/terms/:id", academicsHandler.UpdateTerm)
			updateAcademics.PUT("/terms/:id/current", academicsHandler.SetCurrentTerm)
			updateAcademics.PATCH("/programmes/:id", academicsHandler.UpdateProgramme)
			updateAcademics.PATCH("/subjects/:id", academicsHandler.UpdateSubject)
			updateAcademics.PATCH("/classes/:id", academicsHandler.UpdateClass)
			updateAcademics.POST("/classes/:id/subjects", academicsHandler.AssignSubjectToClass)
			updateAcademics.DELETE("/classes/:id/subjects/:subject_id", academicsHandler.RemoveSubjectFromClass)
		}
	}

	// Enrollment history (M9): student membership in academic years/classes.
	enrollmentService := enrollment.NewService(pool)
	enrollmentHandler := enrollment.NewHandler(enrollmentService)
	enrollmentRoutes := v1.Group("")
	enrollmentRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewEnrollments := enrollmentRoutes.Group("")
		viewEnrollments.Use(rbac.RequirePermission(rbacService, "class.view"))
		{
			viewEnrollments.GET("/enrollments", enrollmentHandler.ListEnrollments)
			viewEnrollments.GET("/enrollments/:id", enrollmentHandler.GetEnrollment)
			// The nested route must reuse the :id wildcard name — gin panics at
			// registration time when the same path position carries two
			// differently-named wildcards (:id above, :student_id here).
			viewEnrollments.GET("/students/:id/enrollments", enrollmentHandler.ListEnrollments)
		}

		createEnrollments := enrollmentRoutes.Group("")
		createEnrollments.Use(rbac.RequirePermission(rbacService, "class.create"))
		{
			createEnrollments.POST("/enrollments", enrollmentHandler.CreateEnrollment)
			createEnrollments.POST("/students/:id/enrollments", enrollmentHandler.CreateEnrollment)
		}

		updateEnrollments := enrollmentRoutes.Group("")
		updateEnrollments.Use(rbac.RequirePermission(rbacService, "class.update"))
		{
			updateEnrollments.PATCH("/enrollments/:id/status", enrollmentHandler.UpdateEnrollmentStatus)
		}
	}

	// Teacher assignments (M10): which teacher teaches which subject in
	// which class. Read access follows classes (class.view); creating and
	// removing assignments is a teaching-coverage change, gated like the
	// class-subject mappings in M8 (class.update).
	assignmentsService := assignments.NewService(pool)
	assignmentsHandler := assignments.NewHandler(assignmentsService)
	assignmentRoutes := v1.Group("")
	assignmentRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewAssignments := assignmentRoutes.Group("")
		viewAssignments.Use(rbac.RequirePermission(rbacService, "class.view"))
		{
			viewAssignments.GET("/teacher-assignments", assignmentsHandler.ListTeacherAssignments)
			viewAssignments.GET("/teacher-assignments/:id", assignmentsHandler.GetTeacherAssignment)
			viewAssignments.GET("/teachers/:id/assignments", assignmentsHandler.ListTeacherAssignments)
		}

		updateAssignments := assignmentRoutes.Group("")
		updateAssignments.Use(rbac.RequirePermission(rbacService, "class.update"))
		{
			updateAssignments.POST("/teacher-assignments", assignmentsHandler.CreateTeacherAssignment)
			updateAssignments.POST("/teachers/:id/assignments", assignmentsHandler.CreateTeacherAssignment)
			updateAssignments.DELETE("/teacher-assignments/:id", assignmentsHandler.DeleteTeacherAssignment)
		}
	}

	// Attendance (M11): teacher-facing register capture (one session per
	// class/subject/term/day, one mark per student) plus the per-status
	// summaries the parent/student portals (M16/M17) will read. Gated on the
	// dedicated attendance.* permission codes from section 11: attendance.view
	// (also held by PARENT/STUDENT for the portals), attendance.record for
	// taking a register, and attendance.update for mark corrections.
	attendanceService := attendance.NewService(pool)
	attendanceHandler := attendance.NewHandler(attendanceService)
	attendanceRoutes := v1.Group("")
	attendanceRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewAttendance := attendanceRoutes.Group("")
		viewAttendance.Use(rbac.RequirePermission(rbacService, "attendance.view"))
		{
			viewAttendance.GET("/attendance/sessions", attendanceHandler.ListSessions)
			viewAttendance.GET("/attendance/sessions/:id", attendanceHandler.GetSession)
			viewAttendance.GET("/students/:id/attendance/summary", attendanceHandler.StudentSummary)
			viewAttendance.GET("/classes/:id/attendance/summary", attendanceHandler.ClassSummary)
		}

		recordAttendance := attendanceRoutes.Group("")
		recordAttendance.Use(rbac.RequirePermission(rbacService, "attendance.record"))
		{
			recordAttendance.POST("/attendance/sessions", attendanceHandler.CreateSession)
		}

		correctAttendance := attendanceRoutes.Group("")
		correctAttendance.Use(rbac.RequirePermission(rbacService, "attendance.update"))
		{
			correctAttendance.PATCH("/attendance/records/:id", attendanceHandler.UpdateRecord)
		}
	}

	// Assessments and results (M12): graded events per class/subject/term
	// and per-student results (score, grade, remarks). Permissions are
	// assessment.view, assessment.create, assessment.update, result.view,
	// result.create, and result.update.
	assessmentsService := assessments.NewService(pool)
	assessmentsHandler := assessments.NewHandler(assessmentsService)
	assessmentRoutes := v1.Group("")
	assessmentRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewAssessments := assessmentRoutes.Group("")
		viewAssessments.Use(rbac.RequirePermission(rbacService, "assessment.view"))
		{
			viewAssessments.GET("/assessments", assessmentsHandler.ListAssessments)
			viewAssessments.GET("/assessments/:id", assessmentsHandler.GetAssessment)
		}

		createAssessments := assessmentRoutes.Group("")
		createAssessments.Use(rbac.RequirePermission(rbacService, "assessment.create"))
		{
			createAssessments.POST("/assessments", assessmentsHandler.CreateAssessment)
		}

		updateAssessments := assessmentRoutes.Group("")
		updateAssessments.Use(rbac.RequirePermission(rbacService, "assessment.update"))
		{
			updateAssessments.PATCH("/assessments/:id", assessmentsHandler.UpdateAssessment)
			updateAssessments.DELETE("/assessments/:id", assessmentsHandler.DeleteAssessment)
		}

		viewResults := assessmentRoutes.Group("")
		viewResults.Use(rbac.RequirePermission(rbacService, "result.view"))
		{
			viewResults.GET("/assessments/:id/results", assessmentsHandler.ListAssessmentResults)
			viewResults.GET("/results/:id", assessmentsHandler.GetResult)
			viewResults.GET("/students/:id/results", assessmentsHandler.ListStudentResults)
		}

		recordResults := assessmentRoutes.Group("")
		recordResults.Use(rbac.RequirePermission(rbacService, "result.create"))
		{
			recordResults.POST("/assessments/:id/results", assessmentsHandler.RecordResults)
		}

		updateResults := assessmentRoutes.Group("")
		updateResults.Use(rbac.RequirePermission(rbacService, "result.update"))
		{
			updateResults.PATCH("/results/:id", assessmentsHandler.UpdateResult)
		}
	}

	// Announcements (M13): school-wide broadcasts and targeted announcements
	// (per role or per class cohort), draft/published/archived status flow,
	// and audience feed filtering. Permissions are announcement.view,
	// announcement.create, and announcement.update.
	announcementsService := announcements.NewService(pool)
	announcementsService.SetMailer(mailer)
	announcementsHandler := announcements.NewHandler(announcementsService)
	announcementRoutes := v1.Group("")
	announcementRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		viewAnnouncements := announcementRoutes.Group("")
		viewAnnouncements.Use(rbac.RequirePermission(rbacService, "announcement.view"))
		{
			viewAnnouncements.GET("/announcements", announcementsHandler.ListAnnouncements)
			viewAnnouncements.GET("/announcements/:id", announcementsHandler.GetAnnouncement)
		}

		createAnnouncements := announcementRoutes.Group("")
		createAnnouncements.Use(rbac.RequirePermission(rbacService, "announcement.create"))
		{
			createAnnouncements.POST("/announcements", announcementsHandler.CreateAnnouncement)
		}

		updateAnnouncements := announcementRoutes.Group("")
		updateAnnouncements.Use(rbac.RequirePermission(rbacService, "announcement.update"))
		{
			updateAnnouncements.PATCH("/announcements/:id", announcementsHandler.UpdateAnnouncement)
			updateAnnouncements.POST("/announcements/:id/publish", announcementsHandler.PublishAnnouncement)
			updateAnnouncements.POST("/announcements/:id/archive", announcementsHandler.ArchiveAnnouncement)
			updateAnnouncements.DELETE("/announcements/:id", announcementsHandler.DeleteAnnouncement)
		}
	}

	// Admin dashboard (M14): institutional overview aggregating student/staff/class
	// KPIs, academic calendar context, daily attendance rates, recent announcements,
	// system audit log activity, and quick actions. Gated on the ADMIN role.
	dashboardService := dashboard.NewService(pool)
	dashboardHandler := dashboard.NewHandler(dashboardService)
	dashboardRoutes := v1.Group("")
	dashboardRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	dashboardRoutes.Use(rbac.RequireRole(rbacService, "ADMIN"))
	{
		dashboardRoutes.GET("/dashboard/admin", dashboardHandler.GetAdminDashboard)
	}

	// Teacher portal (M15): teacher-specific operational context (assigned classes
	// and subjects, student rosters with guardian contacts, academic calendar status,
	// recent attendance registers, assessment score entry tracking, and teacher announcements).
	// Gated on the TEACHER role.
	teacherPortalService := teacherportal.NewService(pool)
	teacherPortalHandler := teacherportal.NewHandler(teacherPortalService)
	teacherPortalRoutes := v1.Group("")
	teacherPortalRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	teacherPortalRoutes.Use(rbac.RequireRole(rbacService, "TEACHER"))
	{
		teacherPortalRoutes.GET("/portal/teacher/overview", teacherPortalHandler.GetTeacherOverview)
		teacherPortalRoutes.GET("/portal/teacher/classes", teacherPortalHandler.GetTeacherClasses)
		teacherPortalRoutes.GET("/portal/teacher/classes/:id/roster", teacherPortalHandler.GetClassRoster)
	}

	// Parent portal (M16): parent-specific read-only context (guardian profile,
	// linked children with active enrollment and lifetime attendance/results
	// aggregates, and per-child detail with recent attendance marks and full
	// results history). Row-level scoping ("only their own children") is
	// enforced in the service via student_guardians, per migration 000004's
	// note that RBAC cannot express row ownership. Gated on the PARENT role.
	parentPortalService := parentportal.NewService(pool)
	parentPortalHandler := parentportal.NewHandler(parentPortalService)
	parentPortalRoutes := v1.Group("")
	parentPortalRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	parentPortalRoutes.Use(rbac.RequireRole(rbacService, "PARENT"))
	{
		parentPortalRoutes.GET("/portal/parent/overview", parentPortalHandler.GetParentOverview)
		parentPortalRoutes.GET("/portal/parent/children/:id", parentPortalHandler.GetChildDetail)
	}

	// Student portal (M17): student-specific read-only context (own profile,
	// active enrollment and academic calendar context, lifetime attendance and
	// results aggregates, recent attendance marks, results history, and targeted
	// announcements). Row-level scoping ("only their own records") is enforced
	// in the service by resolving the student row from the authenticated user's
	// person_id; no endpoint accepts a student id. Gated on the STUDENT role.
	studentPortalService := studentportal.NewService(pool)
	studentPortalHandler := studentportal.NewHandler(studentPortalService)
	studentPortalRoutes := v1.Group("")
	studentPortalRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	studentPortalRoutes.Use(rbac.RequireRole(rbacService, "STUDENT"))
	{
		studentPortalRoutes.GET("/portal/student/overview", studentPortalHandler.GetStudentOverview)
		studentPortalRoutes.GET("/portal/student/attendance", studentPortalHandler.GetStudentAttendance)
		studentPortalRoutes.GET("/portal/student/results", studentPortalHandler.GetStudentResultsHistory)
		studentPortalRoutes.GET("/portal/student/enrollments", studentPortalHandler.GetStudentEnrollments)
	}

	// Reports (M18): institutional and academic analytical outputs (enrollment summaries,
	// class attendance analytics, class performance distribution, and student report cards).
	// Gated on the report.view permission.
	reportsService := reports.NewService(pool)
	reportsHandler := reports.NewHandler(reportsService)
	reportRoutes := v1.Group("")
	reportRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	reportRoutes.Use(rbac.RequirePermission(rbacService, "report.view"))
	{
		reportRoutes.GET("/reports/enrollment", reportsHandler.GetEnrollmentReport)
		reportRoutes.GET("/reports/attendance/classes/:id", reportsHandler.GetClassAttendanceReport)
		reportRoutes.GET("/reports/performance/classes/:id", reportsHandler.GetClassPerformanceReport)
		reportRoutes.GET("/reports/students/:id/report-card", reportsHandler.GetStudentReportCard)
	}

	// Audit logs (M19): authorized admin review of the audit trail every
	// mutation since M6 has been writing (internal/audit.Record). Read-only;
	// gated on the audit.view permission.
	auditLogService := audit.NewService(pool)
	auditLogHandler := audit.NewHandler(auditLogService)
	auditLogRoutes := v1.Group("")
	auditLogRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	auditLogRoutes.Use(rbac.RequirePermission(rbacService, "audit.view"))
	{
		auditLogRoutes.GET("/audit-logs", auditLogHandler.ListAuditLogs)
		auditLogRoutes.GET("/audit-logs/:id", auditLogHandler.GetAuditLog)
		auditLogRoutes.GET("/audit-logs/actions", auditLogHandler.ListActions)
	}

	// Admin/account access control (expert pass follow-up).
	//
	// Two audiences, two gates:
	//   * Managing ADMINS (list admins, revoke the ADMIN role) is
	//     master-admin business: both admin.manage permission AND the
	//     MASTER_ADMIN role are required (the permission exists so RBAC
	//     masters can express the concept; the second, role-based gate
	//     means granting admin.manage to a plain admin still can't promote
	//     them into a position to revoke their peers).
	//   * Blocking/unblocking TEACHER/STAFF/PARENT/STUDENT accounts is
	//     every admin's power, gated on account.block. Blocks kill live
	//     sessions immediately (auth.ValidateSession re-checks status).
	adminAccountsService := adminaccounts.NewService(pool)
	// M34: newly added admins activate through the Clerk invitation email,
	// same best-effort bridge staff/parent provisioning already uses.
	adminAccountsService.SetInviter(auth.NewClerkInviter(cfg.ClerkSecretKey, cfg.FrontendURL+"/register"), cfg.FrontendURL)
	adminAccountsHandler := adminaccounts.NewHandler(adminAccountsService)
	adminAccountsRoutes := v1.Group("")
	adminAccountsRoutes.Use(auth.RequireAuth(authService, sessionCookieName))
	{
		adminManage := adminAccountsRoutes.Group("")
		adminManage.Use(rbac.RequirePermission(rbacService, "admin.manage"))
		adminManage.Use(rbac.RequireRole(rbacService, "MASTER_ADMIN"))
		{
			adminManage.GET("/admins", adminAccountsHandler.ListAdmins)
			adminManage.PUT("/users/:id/admin-role", adminAccountsHandler.RevokeAdmin)
			// M34: master-admin-only admin provisioning (person + user + ADMIN
			// role + Clerk invite in one transaction).
			adminManage.POST("/admins", adminAccountsHandler.CreateAdmin)
		}

		blockable := adminAccountsRoutes.Group("")
		blockable.Use(rbac.RequirePermission(rbacService, "account.block"))
		{
			blockable.GET("/accounts", adminAccountsHandler.ListAccountsByRole)
			blockable.PUT("/users/:id/status", adminAccountsHandler.Block)
		}

		// "Every inviter can see invitees" (M34): any authenticated user —
		// teachers included — lists the accounts THEY minted. Access control
		// is the service filter (users.invited_by = caller), not a permission.
		invitees := adminAccountsRoutes.Group("")
		{
			invitees.GET("/my-invitees", adminAccountsHandler.ListMyInvitees)
			// Inviter-side suspend/restore, strictly scoped in the service to
			// accounts the CALLER provisioned (the account.block route stays
			// for admin-level use).
			invitees.PUT("/my-invitees/:id/status", adminAccountsHandler.InviteeStatus)
		}
	}

	addr := ":" + cfg.Port
	log.Printf("SchoolHub API listening on %s (env=%s)", addr, cfg.Environment)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// corsMiddleware allows exactly one origin (the frontend) to make
// credentialed requests — session cookies only work cross-origin (the
// frontend on :3000, the API on :8080 even in local dev, are different
// origins) if the server explicitly echoes back that one origin and sets
// Access-Control-Allow-Credentials. A wildcard "*" origin is rejected by
// browsers when credentials are involved, so this can't just be gin's
// default permissive CORS setup.
func corsMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// securityHeadersMiddleware sets baseline HTTP security headers (section
// 28). This is a JSON API, not a page-serving app, so most of the
// classic browser-security headers (CSP, X-XSS-Protection) don't have
// much to guard here — these three are the ones that still matter: don't
// let a browser guess/sniff a response's content type into something
// executable, don't let this origin be framed, and don't leak the full
// referrer URL (which can include query strings with tokens/ids) to
// third parties. HSTS is production-only: sending it over local HTTP dev
// would make the browser force HTTPS for localhost going forward, which
// actively breaks local testing rather than protecting anything.
func securityHeadersMiddleware(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if isProduction {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}
