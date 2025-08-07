package services

import (
	"auction-system/internal/domain"
	"auction-system/pkg/logger"
	"auction-system/pkg/utils"
	"context"
	"sync"
	"time"
)

type AuctionManager struct {
	auctionRepo    domain.AuctionRepository
	stateCache     domain.AuctionStateCache
	bidCache       domain.BidCache
	eventPub       domain.EventPublisher
	scheduler      domain.AuctionScheduler
	leaderElection domain.LeaderElection
	biddingRuleDao domain.BiddingRuleDao
	instanceID     string
	log            logger.Logger
	auctionTimers  map[string]*time.Timer
	timerMutex     sync.RWMutex
}

func NewAuctionManager(
	auctionRepo domain.AuctionRepository,
	stateCache domain.AuctionStateCache,
	bidCache domain.BidCache,
	eventPub domain.EventPublisher,
	scheduler domain.AuctionScheduler,
	leaderElection domain.LeaderElection,
	biddingRuleDao domain.BiddingRuleDao,
	instanceID string,
	log logger.Logger,
) *AuctionManager {
	return &AuctionManager{
		auctionRepo:    auctionRepo,
		stateCache:     stateCache,
		bidCache:       bidCache,
		eventPub:       eventPub,
		scheduler:      scheduler,
		leaderElection: leaderElection,
		biddingRuleDao: biddingRuleDao,
		instanceID:     instanceID,
		log:            log,
		auctionTimers:  make(map[string]*time.Timer),
	}
}

func (am *AuctionManager) CreateAuction(ctx context.Context, startTime, endTime time.Time, startingBid float64) (*domain.Auction, error) {
	auction := &domain.Auction{
		ID:        utils.GenerateID("auction"),
		StartTime: startTime,
		EndTime:   endTime,
		Status:    domain.AuctionPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if err := am.auctionRepo.CreateAuction(ctx, auction); err != nil {
		return nil, err
	}

	// Initialize in Redis with starting bid and increment rule
	incrementRule := am.biddingRuleDao.GetIncrementRule(startingBid)
	if err := am.bidCache.InitializeBidding(ctx, auction.ID, startingBid, incrementRule); err != nil {
		return nil, err
	}

	// Schedule start and end
	if err := am.scheduler.ScheduleAuctionStart(ctx, auction.ID, startTime); err != nil {
		return nil, err
	}

	if err := am.scheduler.ScheduleAuctionEnd(ctx, auction.ID, endTime); err != nil {
		return nil, err
	}

	am.log.Info("Auction created", "auction_id", auction.ID)
	return auction, nil
}

func (am *AuctionManager) StartAuction(ctx context.Context, auctionID string) error {
	isLeader, err := am.leaderElection.IsLeader(ctx, am.instanceID)
	if err != nil || !isLeader {
		return err
	}

	am.log.Info("Starting auction", "auction_id", auctionID)

	if err := am.auctionRepo.UpdateAuctionStatus(ctx, auctionID, domain.AuctionActive); err != nil {
		return err
	}

	return am.stateCache.SetAuctionStatus(ctx, auctionID, domain.AuctionActive)
}

func (am *AuctionManager) EndAuction(ctx context.Context, auctionID string) error {
	isLeader, err := am.leaderElection.IsLeader(ctx, am.instanceID)
	if err != nil || !isLeader {
		return err
	}

	am.log.Info("Ending auction", "auction_id", auctionID)

	// Check current status to prevent double-ending
	currentStatus, err := am.stateCache.GetAuctionStatus(ctx, auctionID)
	if err != nil || currentStatus != domain.AuctionActive {
		return nil
	}

	// Update status
	if err := am.auctionRepo.UpdateAuctionStatus(ctx, auctionID, domain.AuctionEnded); err != nil {
		return err
	}

	if err := am.stateCache.SetAuctionStatus(ctx, auctionID, domain.AuctionEnded); err != nil {
		return err
	}

	// Cancel any pending timers
	am.cancelTimer(auctionID)

	// Publish end event
	return am.eventPub.PublishBiddingEvent(ctx, &domain.BidEvent{
		Type:      domain.AuctionEndedBidRejected,
		AuctionID: auctionID,
		Timestamp: time.Now(),
	})
}

func (am *AuctionManager) CheckAndExtendAuction(ctx context.Context, auctionID string, extensionDuration time.Duration) error {
	isLeader, err := am.leaderElection.IsLeader(ctx, am.instanceID)
	if err != nil || !isLeader {
		return err
	}

	// Get auction details
	auction, err := am.auctionRepo.GetAuction(ctx, auctionID)
	if err != nil {
		return err
	}

	// Check if auction ends within the extension window
	timeUntilEnd := time.Until(auction.EndTime)
	if timeUntilEnd <= extensionDuration && timeUntilEnd > 0 {
		newEndTime := time.Now().Add(extensionDuration)

		// Update end time
		auction.EndTime = newEndTime
		auction.UpdatedAt = time.Now()

		// Reschedule
		if err := am.scheduler.RescheduleAuctionEnd(ctx, auctionID, newEndTime); err != nil {
			return err
		}

		// Set new timer
		am.setEndTimer(auctionID, extensionDuration)

		// Publish extension event
		am.eventPub.PublishBiddingEvent(ctx, &domain.BidEvent{
			Type:      domain.AuctionExtended,
			AuctionID: auctionID,
			Timestamp: time.Now(),
		})

		am.log.Info("Auction extended", "auction_id", auctionID, "new_end_time", newEndTime)
	}

	return nil
}

func (am *AuctionManager) setEndTimer(auctionID string, duration time.Duration) {
	am.timerMutex.Lock()
	defer am.timerMutex.Unlock()

	// Cancel existing timer
	if timer, exists := am.auctionTimers[auctionID]; exists {
		timer.Stop()
	}

	// Set new timer
	am.auctionTimers[auctionID] = time.AfterFunc(duration, func() {
		am.EndAuction(context.Background(), auctionID)
	})
}

func (am *AuctionManager) cancelTimer(auctionID string) {
	am.timerMutex.Lock()
	defer am.timerMutex.Unlock()

	if timer, exists := am.auctionTimers[auctionID]; exists {
		timer.Stop()
		delete(am.auctionTimers, auctionID)
	}
}

func (am *AuctionManager) SetScheduler(scheduler domain.AuctionScheduler) {
	am.scheduler = scheduler
}
