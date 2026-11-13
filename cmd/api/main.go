package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"tradex/internal/config"

	"tradex/internal/database"

	"github.com/gin-gonic/gin"
)
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// 1. Load application configuration.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Connect to PostgreSQL.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	// 3. Initialize Gin.
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 4. Application health endpoint.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "tradex-api",
		})
	})

	// 5. Database health endpoint.
	router.GET("/health/db", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := db.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "disconnected",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	// 6. Start HTTP server.
	address := ":" + cfg.Port

	log.Printf("TradeX API starting on port %s", cfg.Port)

	if err := router.Run(address); err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}

	return nil
}