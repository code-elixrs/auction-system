package domain

import "context"

// Cache interfaces
type BidCache interface {
	AtomicBidUpdate(ctx context.Context, auctionID, userID string, amount float64) (bool, error)
	GetCurrentBid(ctx context.Context, auctionID string) (*LocalAuctionCache, error)
	SetBiddingIncrementRule(ctx context.Context, auctionID string, rule float64) error
	InitializeBidding(ctx context.Context, auctionID string, startingBid float64, incrementRule float64) error
}

type AuctionStateCache interface {
	SetAuctionStatus(ctx context.Context, auctionID string, status AuctionStatus) error
	GetAuctionStatus(ctx context.Context, auctionID string) (AuctionStatus, error)
}
