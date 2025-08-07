package main

import (
	"auction-system/internal/api/handlers"
	"auction-system/internal/api/middleware"
	"auction-system/internal/config"
	"auction-system/internal/infrastructure/leader"
	"auction-system/internal/infrastructure/mysql"
	"auction-system/internal/infrastructure/redis"
	"auction-system/internal/infrastructure/websocket"
	"auction-system/internal/services"
	"auction-system/pkg/logger"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	redisClient "github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func main() {
	log := logger.New()

	// Load configuration
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

	// Initialize repositories
	auctionRepo := mysql.NewMySQLAuctionRepository(db)
	//bidRepo := mysql.NewMySQLBidRepository(db)
	schedulerRepo := mysql.NewMySQLSchedulerRepository(db)

	// Initialize Redis services
	bidCache := redis.NewRedisBidCache(rdb)
	stateCache := redis.NewRedisStateCache(rdb)
	eventPublisher := redis.NewRedisEventPublisher(rdb)
	eventSubscriber := redis.NewRedisEventSubscriber(rdb, log)

	// Initialize validator
	validator := services.NewRedisBidValidator(rdb)
	if err := validator.LoadRules(ctx); err != nil {
		log.Error("Failed to load validation rules", "error", err)
		os.Exit(1)
	}

	// Initialize leader election
	leaderElection := leader.NewRedisLeaderElection(rdb, cfg.Leader.TTL)

	// Initialize connection manager
	connManager := websocket.NewConnectionManager(log)

	// Initialize notifiers
	userNotifier := websocket.NewWebSocketNotifier(connManager)
	auctionBroadcaster := websocket.NewWebSocketNotifier(connManager)

	//// Initialize auction manager
	auctionManager := services.NewAuctionManager(
		auctionRepo,
		stateCache,
		bidCache,
		eventPublisher,
		nil, // scheduler will be set later
		leaderElection,
		validator,
		cfg.Instance.ID,
		log,
	)

	// Initialize scheduler
	scheduler := services.NewCronAuctionScheduler(schedulerRepo, auctionManager, log)
	auctionManager.SetScheduler(scheduler) // Set circular dependency

	// Initialize bid service
	bidService := services.NewBidService(
		bidCache,
		stateCache,
		userNotifier,
		validator,
		auctionManager,
		log,
	)

	// Initialize event listener
	eventListener := services.NewEventListener(bidService, connManager, auctionBroadcaster, log)
	bidService.SetEventListener(eventListener)

	// Initialize handlers
	//auctionHandler := handlers.NewAuctionHandler(auctionManager, log)
	wsHandlers := handlers.NewWebSocketHandlers(bidService, auctionRepo, connManager, log)

	// Setup routes
	router := mux.NewRouter()
	router.Use(middleware.CORS)

	// API routes
	//api := router.PathPrefix("/api/v1").Subrouter()
	//api.HandleFunc("/auctions", auctionHandler.CreateAuction).Methods("POST")
	//api.HandleFunc("/auctions/{id}", auctionHandler.GetAuction).Methods("GET")
	//api.HandleFunc("/auctions/{id}/extend", auctionHandler.ExtendAuction).Methods("POST")

	// WebSocket routes
	router.HandleFunc("/ws/auction/{auctionID}", wsHandlers.HandleConnection)

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Start background services
	go func() {
		if err := scheduler.Start(context.Background()); err != nil {
			log.Error("Failed to start scheduler", "error", err)
		}
	}()

	go func() {
		if err := eventListener.Start(context.Background(), eventSubscriber); err != nil {
			log.Error("Failed to start event listener", "error", err)
		}
	}()

	// Try to become leader
	go func() {
		for {
			became, err := leaderElection.BecomeLeader(context.Background(), cfg.Instance.ID)
			if err != nil {
				log.Error("Failed to attempt leadership", "error", err)
				time.Sleep(5 * time.Second)
				continue
			}

			if became {
				log.Info("Became auction leader", "instance_id", cfg.Instance.ID)
			}

			time.Sleep(10 * time.Second)
		}
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Info("Starting auction service", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down auction service...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop scheduler
	scheduler.Stop()

	// Release leadership
	leaderElection.ReleaseLeadership(ctx, cfg.Instance.ID)

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Auction service stopped")
}
