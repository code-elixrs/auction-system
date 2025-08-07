package redis

import (
	"auction-system/internal/domain"
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type EventPublisherImpl struct {
	client *redis.Client
}

func NewEventPublisher(client *redis.Client) *EventPublisherImpl {
	return &EventPublisherImpl{client: client}
}

func (r *EventPublisherImpl) PublishBiddingEvent(ctx context.Context, event *domain.BidEvent) error {
	eventData := fmt.Sprintf("%s:%s:%s:%.2f:%d",
		event.AuctionID, event.Type, event.UserID, event.Amount, event.Timestamp.Unix())

	return r.client.Publish(ctx, "auction_events", eventData).Err()
}
