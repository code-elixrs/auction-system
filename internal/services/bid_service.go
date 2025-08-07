package services

import (
	"auction-system/internal/domain"
	"auction-system/pkg/logger"
	"context"
	"fmt"
	"sync"
	"time"
)

type BidService struct {
	bidCache       domain.BidCache
	stateCache     domain.AuctionStateCache
	userNotifier   domain.UserNotifier
	validator      domain.BidValidator
	localCache     map[string]*domain.LocalAuctionCache
	cacheMutex     sync.RWMutex
	eventListener  *EventListener
	auctionManager *AuctionManager
	log            logger.Logger
}

func NewBidService(
	bidCache domain.BidCache,
	stateCache domain.AuctionStateCache,
	userNotifier domain.UserNotifier,
	validator domain.BidValidator,
	auctionManager *AuctionManager,
	log logger.Logger,
) *BidService {
	service := &BidService{
		bidCache:       bidCache,
		stateCache:     stateCache,
		userNotifier:   userNotifier,
		validator:      validator,
		localCache:     make(map[string]*domain.LocalAuctionCache),
		auctionManager: auctionManager,
		log:            log,
	}

	return service
}

func (s *BidService) SetEventListener(eventListener *EventListener) {
	s.eventListener = eventListener
}

func (s *BidService) PlaceBid(ctx context.Context, auctionID, userID string, amount float64) error {
	s.log.Info("Placing bid", "auction_id", auctionID, "user_id", userID, "amount", amount)

	// Check auction status first
	status, err := s.stateCache.GetAuctionStatus(ctx, auctionID)
	if err != nil {
		return err
	}

	if status != domain.AuctionActive {
		s.userNotifier.NotifyUser(ctx, userID, map[string]interface{}{
			"type":   "bid_rejected",
			"reason": "auction_not_active",
			"status": status.String(),
		})
		return nil
	}

	// Initialize auction cache if not exists
	if err := s.ensureAuctionCached(ctx, auctionID); err != nil {
		return err
	}

	// Quick local validation
	s.cacheMutex.RLock()
	cachedAuction := s.localCache[auctionID]
	s.cacheMutex.RUnlock()

	if !s.validator.ValidateIncrement(cachedAuction.CurrentBid, amount) {
		s.userNotifier.NotifyUser(ctx, userID, map[string]interface{}{
			"type":             "bid_rejected",
			"reason":           "insufficient_increment",
			"current_bid":      cachedAuction.CurrentBid,
			"current_winner":   cachedAuction.WinnerID,
			"required_minimum": s.validator.GetMinimumBid(cachedAuction.CurrentBid),
		})
		return nil
	}

	// Atomic Redis update
	success, err := s.bidCache.AtomicBidUpdate(ctx, auctionID, userID, amount)
	if err != nil {
		s.log.Error("Failed to update bid", "error", err)
		return err
	}

	// Check if we need to extend auction (30-second rule)
	if success {
		go s.checkAuctionExtension(auctionID)
	}

	return nil
}

func (s *BidService) ensureAuctionCached(ctx context.Context, auctionID string) error {
	s.cacheMutex.RLock()
	_, exists := s.localCache[auctionID]
	s.cacheMutex.RUnlock()

	if !exists {
		currentBid, err := s.bidCache.GetCurrentBid(ctx, auctionID)
		if err != nil {
			return err
		}

		s.cacheMutex.Lock()
		s.localCache[auctionID] = currentBid
		s.cacheMutex.Unlock()
	}

	return nil
}

func (s *BidService) checkAuctionExtension(auctionID string) {
	// Check if auction ends within 30 seconds
	// If so, extend by 30 seconds
	s.auctionManager.CheckAndExtendAuction(context.Background(), auctionID, 30*time.Second)
}

func (s *BidService) UpdateLocalCache(auctionID string, bid float64, winnerID string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	//TODO fix this using logger
	fmt.Printf("Updating local cache for auction %s: bid=%.2f, winner=%s\n", auctionID, bid, winnerID)
	s.localCache[auctionID] = &domain.LocalAuctionCache{
		CurrentBid:  bid,
		WinnerID:    winnerID,
		LastUpdated: time.Now(),
	}
}

func (s *BidService) RemoveFromCache(auctionID string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	delete(s.localCache, auctionID)
}

func (s *BidService) HandleWebSocketConnection(userID, auctionID string) error {
	return s.ensureAuctionCached(context.Background(), auctionID)
}
