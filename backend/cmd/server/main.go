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
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
	"github.com/thunderlight247/schoolhub/backend/internal/database"
	"github.com/thunderlight247/schoolhub/backend/internal/enrollment"
	"github.com/thunderlight247/schoolhub/backend/internal/rbac"
	"github.com/thunderlight247/schoolhub/backend/internal/staff"
	"github.com/thunderlight247/schoolhub/backend/internal/students"
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
		Name:   sessionCookieName,
		Domain: cfg.CookieDomain,
		Secure: cfg.IsProduction(),
	})

	rbacService := rbac.NewService(queries)
	rbacHandler := rbac.NewHandler(queries)

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(corsMiddleware(cfg.FrontendURL))

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

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/logout", authHandler.Logout)
		v1.GET("/me", auth.RequireAuth(authService, sessionCookieName), authHandler.Me)
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
			createStudents.POST("/guardians", studentHandler.CreateGuardian)
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
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
