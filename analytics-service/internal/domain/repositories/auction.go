package repositories

import (
	"auction-system/internal/domain"
	"context"
)

type AuctionRepository interface {
	CreateAuction(ctx context.Context, auction *domain.Auction) error
	GetAuction(ctx context.Context, auctionID string) (*domain.Auction, error)
	UpdateAuctionStatus(ctx context.Context, auctionID string, status domain.AuctionStatus) error
	GetActiveAuctions(ctx context.Context) ([]*domain.Auction, error)
}
