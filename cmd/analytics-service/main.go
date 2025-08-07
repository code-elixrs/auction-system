package main

import (
	"auction-system/internal/config"
	"auction-system/internal/domain"
	"auction-system/internal/infrastructure/mysql"
	"auction-system/internal/infrastructure/redis"
	"auction-system/pkg/logger"
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	redisClient "github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
)

type AnalyticsService struct {
	subscriber *redis.RedisEventSubscriber
	bidRepo    *mysql.MySQLBidRepository
	log        logger.Logger
}

func NewAnalyticsService(subscriber *redis.RedisEventSubscriber, bidRepo *mysql.MySQLBidRepository, log logger.Logger) *AnalyticsService {
	return &AnalyticsService{
		subscriber: subscriber,
		bidRepo:    bidRepo,
		log:        log,
	}
}

func (as *AnalyticsService) Start(ctx context.Context) error {
	as.log.Info("Starting analytics service")

	return as.subscriber.SubscribeToBidEvents(ctx, func(event *domain.BidEvent) error {
		// Only store successful bid events
		if event.Type == domain.BidAccepted {
			as.log.Info("Storing bid event", "auction_id", event.AuctionID, "user_id", event.UserID, "amount", event.Amount)
			return as.bidRepo.SaveBidEvent(context.Background(), event)
		}
		return nil
	})
}

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

	// Test MySQL connection
	if err := db.PingContext(ctx); err != nil {
		log.Error("Failed to ping MySQL", "error", err)
		os.Exit(1)
	}

	// Initialize services
	eventSubscriber := redis.NewRedisEventSubscriber(rdb, log)
	bidRepo := mysql.NewMySQLBidRepository(db)

	analyticsService := NewAnalyticsService(eventSubscriber, bidRepo, log)

	// Start service
	go func() {
		if err := analyticsService.Start(context.Background()); err != nil {
			log.Error("Analytics service failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down analytics service...")
	log.Info("Analytics service stopped")
}
