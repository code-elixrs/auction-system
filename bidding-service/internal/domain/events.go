package domain

import "context"

// Event interfaces
type EventPublisher interface {
	PublishBiddingEvent(ctx context.Context, event *BidEvent) error
}

type EventSubscriber interface {
	SubscribeToBidEvents(ctx context.Context, handler EventHandler) error
}

type EventHandler func(event *BidEvent) error
