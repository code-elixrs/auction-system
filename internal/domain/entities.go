package domain

import (
	"time"
)

type Auction struct {
	ID        string
	StartTime time.Time
	EndTime   time.Time
	Status    AuctionStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AuctionStatus int

const (
	AuctionPending AuctionStatus = iota
	AuctionActive
	AuctionEnded
	AuctionCancelled
)

func (s AuctionStatus) String() string {
	switch s {
	case AuctionPending:
		return "pending"
	case AuctionActive:
		return "active"
	case AuctionEnded:
		return "ended"
	case AuctionCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

type LocalAuctionCache struct {
	AuctionID     string
	CurrentBid    float64
	WinnerID      string
	IncrementRule float64
	LastUpdated   time.Time
}

type BidEvent struct {
	Type      BidEventType `json:"type"`
	AuctionID string       `json:"auction_id"`
	UserID    string       `json:"user_id"`
	Amount    float64      `json:"amount"`
	Timestamp time.Time    `json:"timestamp"`
}

type BidEventType string

const (
	BidAccepted             BidEventType = "bid_accepted"
	BidRejected             BidEventType = "bid_rejected"
	AuctionEndedBidRejected BidEventType = "auction_ended"
	AuctionExtended         BidEventType = "auction_extended"
)

type BidValidationRules struct {
	Rules map[string]float64 `json:"rules"`
}

type ScheduledJob struct {
	ID        string
	AuctionID string
	JobType   JobType
	RunAt     time.Time
	Status    JobStatus
	CreatedAt time.Time
}

type JobType string

const (
	JobStartAuction JobType = "start_auction"
	JobEndAuction   JobType = "end_auction"
)

type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobExecuted  JobStatus = "executed"
	JobCancelled JobStatus = "cancelled"
)
