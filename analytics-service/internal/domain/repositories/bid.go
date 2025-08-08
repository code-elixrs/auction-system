package repositories

import (
	"auction-system/internal/domain"
	"context"
)

type BidRepository interface {
	SaveBidEvent(ctx context.Context, event *domain.BidEvent) error
	GetBidHistory(ctx context.Context, auctionID string) ([]*domain.BidEvent, error)
}
