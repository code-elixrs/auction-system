package main

import (
	"auction-system/internal/api/handlers"
	"auction-system/pkg/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auction-system/internal/config"
	"auction-system/internal/infrastructure/leader"
	"auction-system/internal/infrastructure/mysql"
	"auction-system/internal/infrastructure/redis"
	"auction-system/internal/services"
	"auction-system/pkg/logger"

	redisClient "github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	log := logger.New()
	log.Info("Starting Auction Manager Service")

	// Load configuration
	// TODO: This should be service specific
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize Redis
	rdb := redisClient.NewClient(&redisClient.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	log.Info("Connected to Redis", "address", cfg.Redis.Address)

	// Initialize MySQL
	db := utils.InitializeMysql(cfg, log, ctx)

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Error("Failed to close MySQL connection", "error", err)
		}
	}(db)

	// TODO: Add loggers and proper logging in each components
	// Initialize repositories
	auctionRepo := mysql.NewMySQLAuctionRepository(db)
	schedulerRepo := mysql.NewMySQLSchedulerRepository(db)

	// Initialize Redis based components
	bidCache := redis.NewBidCache(rdb)
	stateCache := redis.NewStateCache(rdb)
	eventPublisher := redis.NewEventPublisher(rdb)

	//Initialize validator
	biddingRuleDao := services.NewBiddingRuleDao(rdb)
	if err := biddingRuleDao.LoadRules(ctx); err != nil {
		log.Error("Failed to load validation rules", "error", err)
		os.Exit(1)
	}

	// Initialize leader election
	leaderElection := leader.NewRedisLeaderElection(rdb, cfg.Leader.TTL)

	// Initialize auction manager
	//TODO: Remove this cyclic dependency later!!
	auctionManager := services.NewAuctionManager(
		auctionRepo,
		stateCache,
		bidCache,
		eventPublisher,
		nil, // scheduler will be set below
		leaderElection,
		biddingRuleDao,
		cfg.Instance.ID,
		log,
	)

	// Initialize scheduler
	scheduler := services.NewCronAuctionScheduler(schedulerRepo, auctionManager, log)

	auctionManager.SetScheduler(scheduler)

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","id":"${id}","remote_ip":"${remote_ip}","host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}","status":${status},"error":"${error}","latency":${latency},"latency_human":"${latency_human}","bytes_in":${bytes_in},"bytes_out":${bytes_out}}` + "\n",
	}))
	e.Use(middleware.Recover())

	addCorsDebuggingMiddleware(e, log)

	// Initialize handlers
	auctionHandler := handlers.NewAuctionHandler(auctionManager, log)

	// API routes
	api := e.Group("/api/v1")
	api.POST("/auctions", auctionHandler.CreateAuction)
	api.GET("/auctions/:id", auctionHandler.GetAuction)
	api.POST("/auctions/:id/extend", auctionHandler.ExtendAuction)

	// Health check endpoint
	e.GET("/health", healthStatusHandler(cfg))

	corsDebuggingEndpoints(e, cfg)

	// Start background services
	go func() {
		if err := scheduler.Start(context.Background()); err != nil {
			log.Error("Failed to start scheduler", "error", err)
		}
	}()

	go func() {
		for {
			became, err := leaderElection.BecomeLeader(context.Background(), cfg.Instance.ID)
			if err != nil {
				log.Error("Failed to attempt leadership", "error", err)
				time.Sleep(5 * time.Second)
				continue
			}
			if became {
				log.Info("Became auction manager leader", "instance_id", cfg.Instance.ID)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// Start server with CORRECT PORT
	serverAddr := fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port)
	log.Info("Starting auction manager server", "address", serverAddr, "port", cfg.Server.Port)

	go func() {
		if err := e.Start(serverAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down auction manager service...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := scheduler.Stop(); err != nil {
		log.Error("Failed to stop scheduler", "error", err)
	}
	if err := leaderElection.ReleaseLeadership(ctx, cfg.Instance.ID); err != nil {
		log.Error("Failed to release leadership", "error", err)
	}

	if err := e.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Auction manager service stopped")
}

func corsDebuggingEndpoints(e *echo.Echo, cfg *config.Config) {
	// TODO: Remove after development!!
	// Debug CORS endpoint
	e.GET("/debug/cors", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "CORS is working",
			"method":  c.Request().Method,
			"origin":  c.Request().Header.Get("Origin"),
			"port":    cfg.Server.Port,
		})
	})

	e.POST("/debug/cors", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "POST CORS is working",
			"method":  c.Request().Method,
			"origin":  c.Request().Header.Get("Origin"),
		})
	})
}

func addCorsDebuggingMiddleware(e *echo.Echo, log logger.Logger) {
	// CORS Middleware - For local testing
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			echo.GET, echo.HEAD, echo.PUT, echo.PATCH,
			echo.POST, echo.DELETE, echo.OPTIONS,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			echo.HeaderXRequestedWith,
			echo.HeaderAccessControlRequestMethod,
			echo.HeaderAccessControlRequestHeaders,
		},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Request debugging middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			log.Info("Request received",
				"method", req.Method,
				"path", req.URL.Path,
				"remote_addr", c.RealIP(),
				"origin", req.Header.Get("Origin"),
				"content_type", req.Header.Get("Content-Type"))
			return next(c)
		}
	})
}

func healthStatusHandler(cfg *config.Config) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"service":   "auction-manager",
			"timestamp": time.Now().Format(time.RFC3339),
			"port":      cfg.Server.Port,
			"version":   "1.0.0",
		})
	}
}
