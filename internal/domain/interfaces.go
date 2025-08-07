package domain

import (
	"context"
	"time"
)

// Repository interfaces
type AuctionRepository interface {
	CreateAuction(ctx context.Context, auction *Auction) error
	GetAuction(ctx context.Context, auctionID string) (*Auction, error)
	UpdateAuctionStatus(ctx context.Context, auctionID string, status AuctionStatus) error
	GetActiveAuctions(ctx context.Context) ([]*Auction, error)
}

type BidRepository interface {
	SaveBidEvent(ctx context.Context, event *BidEvent) error
	GetBidHistory(ctx context.Context, auctionID string) ([]*BidEvent, error)
}

type SchedulerRepository interface {
	CreateJob(ctx context.Context, job *ScheduledJob) error
	GetPendingJobs(ctx context.Context, before time.Time) ([]*ScheduledJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error
	CancelJobsForAuction(ctx context.Context, auctionID string) error
}

// Cache interfaces
type BidCache interface {
	AtomicBidUpdate(ctx context.Context, auctionID, userID string, amount float64) (bool, error)
	GetCurrentBid(ctx context.Context, auctionID string) (*LocalAuctionCache, error)
	SetAuctionIncrementRule(ctx context.Context, auctionID string, rule float64) error
	InitializeAuction(ctx context.Context, auctionID string, startingBid float64, incrementRule float64) error
}

type AuctionStateCache interface {
	SetAuctionStatus(ctx context.Context, auctionID string, status AuctionStatus) error
	GetAuctionStatus(ctx context.Context, auctionID string) (AuctionStatus, error)
}

// Event interfaces
type EventPublisher interface {
	PublishBidEvent(ctx context.Context, event *BidEvent) error
}

type EventSubscriber interface {
	SubscribeToBidEvents(ctx context.Context, handler EventHandler) error
}

type EventHandler func(event *BidEvent) error

// Notification interfaces
type UserNotifier interface {
	NotifyUser(ctx context.Context, userID string, message interface{}) error
}

type AuctionBroadcaster interface {
	BroadcastToAuction(ctx context.Context, auctionID string, message interface{}) error
}

// Validation interface
type BidValidator interface {
	ValidateIncrement(currentAmount, newAmount float64) bool
	GetMinimumBid(currentAmount float64) float64
	GetIncrementRule(amount float64) float64
	LoadRules(ctx context.Context) error
}

// Leader election interface
type LeaderElection interface {
	BecomeLeader(ctx context.Context, instanceID string) (bool, error)
	IsLeader(ctx context.Context, instanceID string) (bool, error)
	ReleaseLeadership(ctx context.Context, instanceID string) error
}

// Scheduler interface
type AuctionScheduler interface {
	ScheduleAuctionStart(ctx context.Context, auctionID string, startTime time.Time) error
	ScheduleAuctionEnd(ctx context.Context, auctionID string, endTime time.Time) error
	RescheduleAuctionEnd(ctx context.Context, auctionID string, newEndTime time.Time) error
	CancelSchedule(ctx context.Context, auctionID string) error
	Start(ctx context.Context) error
	Stop() error
}

// WebSocket interfaces
type WebSocketConnection interface {
	Send(message interface{}) error
	Close() error
	UserID() string
	AuctionID() string
}

type ConnectionManager interface {
	RegisterConnection(userID, auctionID string, conn WebSocketConnection) error
	UnregisterConnection(userID, auctionID string) error
	GetConnectionsForAuction(auctionID string) []WebSocketConnection
	GetConnectionsForUser(userID string) []WebSocketConnection
	BroadcastToAuction(auctionID string, message interface{}) error
	NotifyUser(userID string, message interface{}) error
	CloseAndUnregisterConnections(auctionID string) error
}
