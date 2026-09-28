// Command server is the entrypoint for the SchoolHub API. At this milestone
// it does nothing beyond: load config, connect to PostgreSQL, and expose a
// health-check endpoint. Business logic (auth, students, staff, ...) is
// added in later milestones under internal/.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/yourusername/schoolhub/backend/config"
	"github.com/yourusername/schoolhub/backend/internal/database"
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

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

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

	addr := ":" + cfg.Port
	log.Printf("SchoolHub API listening on %s (env=%s)", addr, cfg.Environment)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
