package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"auction-system/internal/domain"
	"auction-system/pkg/logger"
	
	"github.com/go-redis/redis/v8"
)

type RedisEventSubscriber struct {
	client *redis.Client
	log    logger.Logger
}

func NewRedisEventSubscriber(client *redis.Client, log logger.Logger) *RedisEventSubscriber {
	return &RedisEventSubscriber{
		client: client,
		log:    log,
	}
}

func (r *RedisEventSubscriber) SubscribeToBidEvents(ctx context.Context, handler domain.EventHandler) error {
	pubsub := r.client.Subscribe(ctx, "auction_events")
	defer pubsub.Close()

	ch := pubsub.Channel()

	r.log.Info("Subscribed to auction events")

	for {
		select {
		case msg := <-ch:
			event, err := r.parseEventData(msg.Payload)
			if err != nil {
				r.log.Error("Failed to parse event", "payload", msg.Payload, "error", err)
				continue
			}

			if err := handler(event); err != nil {
				r.log.Error("Failed to handle event", "event", event, "error", err)
			}

		case <-ctx.Done():
			r.log.Info("Event subscriber stopped")
			return ctx.Err()
		}
	}
}

func (r *RedisEventSubscriber) parseEventData(payload string) (*domain.BidEvent, error) {
	// Parse "auctionID:eventType:userID:amount:timestamp"
	parts := strings.Split(payload, ":")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid event format: %s", payload)
	}

	amount, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return nil, err
	}

	timestamp, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return nil, err
	}

	return &domain.BidEvent{
		AuctionID: parts[0],
		Type:      domain.BidEventType(parts[1]),
		UserID:    parts[2],
		Amount:    amount,
		Timestamp: time.Unix(timestamp, 0),
	}, nil
}
