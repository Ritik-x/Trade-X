package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"tradex/internal/cache"
	"tradex/internal/config"
	"tradex/internal/middleware"
	"tradex/internal/modules/auth"
	"tradex/internal/modules/market"
	"tradex/internal/modules/watchlist"

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

	redisClient, err := cache.Connect(ctx, cfg.RedisAddr)
	if err != nil {
		return fmt.Errorf("initialize redis: %w", err)
	}
	defer redisClient.Close()
	gin.SetMode(gin.ReleaseMode)
	// 3. Initialize Gin.
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}

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

	// Initialize authentication dependencies.
	authRepository := auth.NewRepository(db)
	sessionRepository := auth.NewSessionRepository(db)

	authService := auth.NewService(
		authRepository,
		sessionRepository,
		cfg.JWTSecret,
	)

	authHandler := auth.NewHandler(authService)

	//Initialize market dependenices
	// Initialize market dependencies.
	marketRepository := market.NewRepository(db)
	quoteRepository := market.NewQuoteRepository(db)

	// Initialize Redis quote cache.
	quoteCache := cache.NewQuoteCache(redisClient)
	nseClient := market.NewNSEClient()

	marketService := market.NewService(
		marketRepository,
		quoteRepository,
		quoteCache,
		nseClient,
	)

	marketHandler := market.NewHandler(marketService)
	// Initialize watchlist dependencies.
	watchlistRepository := watchlist.NewWatchlistRepository(db)
	watchlistService := watchlist.NewWatchlistService(watchlistRepository)
	watchlistHandler := watchlist.NewHandler(watchlistService)
	// Register authentication routes.
	api := router.Group("/api/v1")

	// Public routes.
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.POST("/auth/logout", authHandler.Logout)

	api.GET("/stocks", marketHandler.GetAll)
	api.GET("/stocks/:exchange/:symbol", marketHandler.GetBySymbol)
	api.GET("/stocks/:exchange/:symbol/quote", marketHandler.GetQuote)
	api.POST(
		"/stocks/:exchange/:symbol/refresh",
		marketHandler.RefreshQuote,
	)
	// Protected routes.
	// Protected routes.
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	protected.GET("/users/me", authHandler.Me)

	// Watchlist routes.
	protected.POST("/watchlists", watchlistHandler.Create)
	protected.GET("/watchlists", watchlistHandler.GetAll)
	protected.POST("/watchlists/:id/stocks", watchlistHandler.AddStock)
	protected.DELETE("/watchlists/:id/stocks/:instrument_id", watchlistHandler.RemoveStock)
	protected.GET("/watchlists/:id/stocks", watchlistHandler.GetStocks)
	// 6. Start HTTP server.
	address := ":" + cfg.Port

	log.Printf("TradeX API starting on port %s", cfg.Port)

	if err := router.Run(address); err != nil {
		return fmt.Errorf("start HTTP server: %w", err)
	}

	return nil
}
