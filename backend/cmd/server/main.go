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
	"github.com/thunderlight247/schoolhub/backend/internal/auth"
	"github.com/thunderlight247/schoolhub/backend/internal/database"
	"github.com/thunderlight247/schoolhub/backend/internal/rbac"
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
