package main

import (
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

	"auction-system/internal/api/handlers"
	"auction-system/internal/api/middleware"
	"auction-system/internal/config"
	"auction-system/internal/infrastructure/mysql"
	"auction-system/internal/infrastructure/redis"
	"auction-system/internal/infrastructure/websocket"
	"auction-system/internal/services"
	"auction-system/pkg/logger"

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
	db := utils.InitializeMysql(cfg, log, ctx)

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Error("Failed to close MySQL connection", "error", err)
		}
	}(db)

	// Initialize repositories
	auctionRepo := mysql.NewMySQLAuctionRepository(db)

	// Initialize Redis services
	bidCache := redis.NewBidCache(rdb)
	stateCache := redis.NewStateCache(rdb)
	eventSubscriber := redis.NewRedisEventSubscriber(rdb, log)

	// Initialize connection manager
	connManager := websocket.NewConnectionManager(log)

	// Initialize notifiers
	userNotifier := websocket.NewWebSocketNotifier(connManager)
	auctionBroadcaster := websocket.NewWebSocketNotifier(connManager)

	// Initialize bid service
	bidService := services.NewBidService(
		bidCache,
		stateCache,
		userNotifier,
		log,
	)

	// Initialize event listener
	eventListener := services.NewEventListener(bidService, connManager, auctionBroadcaster, log)

	// Initialize handlers
	wsHandlers := handlers.NewWebSocketHandlers(bidService, auctionRepo, connManager, log)

	// Setup routes
	router := mux.NewRouter()
	router.Use(middleware.CORS)

	// WebSocket routes
	router.HandleFunc("/ws/auction/{auctionID}", wsHandlers.HandleConnection)

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	go func() {

		// TODO: Subscribe the auction id related event stream in case auction info is being populated in local cache with failover mechanism.
		// TODO: Remove the global event stream and rely on auction specific event streams
		if err := eventListener.Start(context.Background(), eventSubscriber); err != nil {
			log.Error("Failed to start event listener", "error", err)
		}
	}()

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	go func() {
		log.Info("Starting auction service", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Auction service stopped")
}
