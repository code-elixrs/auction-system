package domain

import (
	"context"
	"time"
)

// Scheduler interface
type AuctionScheduler interface {
	ScheduleAuctionStart(ctx context.Context, auctionID string, startTime time.Time) error
	ScheduleAuctionEnd(ctx context.Context, auctionID string, endTime time.Time) error
	RescheduleAuctionEnd(ctx context.Context, auctionID string, newEndTime time.Time) error
	CancelSchedule(ctx context.Context, auctionID string) error
	Start(ctx context.Context) error
	Stop() error
}
