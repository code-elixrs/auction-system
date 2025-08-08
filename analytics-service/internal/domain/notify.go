package domain

import "context"

// Notification interfaces
type UserNotifier interface {
	NotifyUser(ctx context.Context, userID string, message interface{}) error
}

type AuctionBroadcaster interface {
	BroadcastToAuction(ctx context.Context, auctionID string, message interface{}) error
}
