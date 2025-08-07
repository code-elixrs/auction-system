package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

type AuctionHandler struct {
	auctionManager *services.AuctionManager
	log            logger.Logger
}

type CreateAuctionRequest struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	StartingBid float64   `json:"starting_bid"`
}

type CreateAuctionResponse struct {
	AuctionID   string    `json:"auction_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	StartingBid float64   `json:"starting_bid"`
	Status      string    `json:"status"`
}

func NewAuctionHandler(auctionManager *services.AuctionManager, log logger.Logger) *AuctionHandler {
	return &AuctionHandler{
		auctionManager: auctionManager,
		log:            log,
	}
}

func (h *AuctionHandler) CreateAuction(c echo.Context) error {
	h.log.Info("CreateAuction endpoint called",
		"method", c.Request().Method,
		"remote_addr", c.RealIP(),
		"user_agent", c.Request().UserAgent(),
		"content_type", c.Request().Header.Get("Content-Type"))

	var req CreateAuctionRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error("Failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validation
	if req.StartTime.Before(time.Now()) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Start time must be in the future"})
	}

	if req.EndTime.Before(req.StartTime) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "End time must be after start time"})
	}

	if req.StartingBid <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Starting bid must be positive"})
	}

	auction, err := h.auctionManager.CreateAuction(c.Request().Context(), req.StartTime, req.EndTime, req.StartingBid)
	if err != nil {
		h.log.Error("Failed to create auction", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create auction"})
	}

	response := CreateAuctionResponse{
		AuctionID:   auction.ID,
		StartTime:   auction.StartTime,
		EndTime:     auction.EndTime,
		StartingBid: req.StartingBid,
		Status:      auction.Status.String(),
	}

	h.log.Info("Auction created successfully", "auction_id", auction.ID)
	return c.JSON(http.StatusCreated, response)
}

func (h *AuctionHandler) GetAuction(c echo.Context) error {
	auctionID := c.Param("id")
	h.log.Info("GetAuction endpoint called", "auction_id", auctionID)

	return c.JSON(http.StatusOK, map[string]string{
		"auction_id": auctionID,
		"message":    "Auction details would be here",
	})
}

func (h *AuctionHandler) ExtendAuction(c echo.Context) error {
	auctionID := c.Param("id")
	extensionStr := c.QueryParam("seconds")

	h.log.Info("ExtendAuction endpoint called", "auction_id", auctionID, "seconds", extensionStr)

	if extensionStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Extension duration required"})
	}

	extensionDuration := 30 * time.Second
	if err := h.auctionManager.CheckAndExtendAuction(c.Request().Context(), auctionID, extensionDuration); err != nil {
		h.log.Error("Failed to extend auction", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to extend auction"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Auction extended successfully",
	})
}

func main() {
	log := logger.New()
	log.Info("Starting Auction Manager Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// FORCE PORT CONFIGURATION FOR AUCTION MANAGER
	managerPort := 8081
	if portEnv := os.Getenv("MANAGER_PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			managerPort = p
		}
	}

	log.Info("Configuration loaded",
		"redis_address", cfg.Redis.Address,
		"mysql_dsn", "***hidden***",
		"instance_id", cfg.Instance.ID,
		"manager_port", managerPort)

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
	db, err := sql.Open("mysql", cfg.MySQL.DSN)
	if err != nil {
		log.Error("Failed to connect to MySQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime)

	// Test MySQL connection
	if err := db.PingContext(ctx); err != nil {
		log.Error("Failed to ping MySQL", "error", err)
		os.Exit(1)
	}
	log.Info("Connected to MySQL")

	// Initialize repositories
	auctionRepo := mysql.NewMySQLAuctionRepository(db)
	schedulerRepo := mysql.NewMySQLSchedulerRepository(db)

	// Initialize Redis services
	bidCache := redis.NewRedisBidCache(rdb)
	stateCache := redis.NewRedisStateCache(rdb)
	eventPublisher := redis.NewRedisEventPublisher(rdb)

	// Initialize validator
	validator := services.NewRedisBidValidator(rdb)
	if err := validator.LoadRules(ctx); err != nil {
		log.Error("Failed to load validation rules", "error", err)
		os.Exit(1)
	}

	// Initialize leader election
	leaderElection := leader.NewRedisLeaderElection(rdb, cfg.Leader.TTL)

	// Initialize auction manager
	auctionManager := services.NewAuctionManager(
		auctionRepo,
		stateCache,
		bidCache,
		eventPublisher,
		nil, // scheduler will be set below
		leaderElection,
		validator,
		cfg.Instance.ID+"-manager",
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

	// CORS Middleware - Very permissive for debugging
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

	// Initialize handlers
	auctionHandler := NewAuctionHandler(auctionManager, log)

	// API routes
	api := e.Group("/api/v1")
	api.POST("/auctions", auctionHandler.CreateAuction)
	api.GET("/auctions/:id", auctionHandler.GetAuction)
	api.POST("/auctions/:id/extend", auctionHandler.ExtendAuction)

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"service":   "auction-manager",
			"timestamp": time.Now().Format(time.RFC3339),
			"port":      managerPort,
			"version":   "1.0.0",
		})
	})

	// Debug CORS endpoint
	e.GET("/debug/cors", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "CORS is working",
			"method":  c.Request().Method,
			"origin":  c.Request().Header.Get("Origin"),
			"port":    managerPort,
		})
	})

	e.POST("/debug/cors", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "POST CORS is working",
			"method":  c.Request().Method,
			"origin":  c.Request().Header.Get("Origin"),
		})
	})

	// Start background services
	go func() {
		if err := scheduler.Start(context.Background()); err != nil {
			log.Error("Failed to start scheduler", "error", err)
		}
	}()

	// Try to become leader
	go func() {
		instanceID := cfg.Instance.ID + "-manager"
		for {
			became, err := leaderElection.BecomeLeader(context.Background(), instanceID)
			if err != nil {
				log.Error("Failed to attempt leadership", "error", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if became {
				log.Info("Became auction manager leader", "instance_id", instanceID)
			}

			time.Sleep(10 * time.Second)
		}
	}()

	// Start server with CORRECT PORT
	serverAddr := fmt.Sprintf("0.0.0.0:%d", managerPort)
	log.Info("Starting auction manager server", "address", serverAddr, "port", managerPort)

	go func() {
		if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
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

	scheduler.Stop()
	leaderElection.ReleaseLeadership(ctx, cfg.Instance.ID+"-manager")

	if err := e.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Auction manager service stopped")
}
