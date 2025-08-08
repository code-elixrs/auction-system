package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"auction-system/internal/domain"
	"auction-system/pkg/logger"
)

type BidService struct {
	bidCache     domain.BidCache
	stateCache   domain.AuctionStateCache
	userNotifier domain.UserNotifier
	localCache   map[string]*domain.LocalAuctionCache
	cacheMutex   sync.RWMutex
	log          logger.Logger
}

func NewBidService(
	bidCache domain.BidCache,
	stateCache domain.AuctionStateCache,
	userNotifier domain.UserNotifier,
	log logger.Logger,
) *BidService {
	service := &BidService{
		bidCache:     bidCache,
		stateCache:   stateCache,
		userNotifier: userNotifier,
		localCache:   make(map[string]*domain.LocalAuctionCache),
		log:          log,
	}

	return service
}

func (s *BidService) PlaceBid(ctx context.Context, auctionID, userID string, amount float64) error {
	s.log.Info("Placing bid", "auction_id", auctionID, "user_id", userID, "amount", amount)

	// Check auction status first
	status, err := s.stateCache.GetAuctionStatus(ctx, auctionID)
	if err != nil {
		return err
	}

	if status != domain.AuctionActive {
		err := s.userNotifier.NotifyUser(ctx, userID, map[string]interface{}{
			"type":   "bid_rejected",
			"reason": "auction_not_active",
			"status": status.String(),
		})
		if err != nil {
			s.log.Error("Failed to notify user", "auction_id", auctionID, "user_id", userID)
			return err
		}
		return nil
	}

	// Initialize auction cache if not exists
	if err := s.ensureAuctionCached(ctx, auctionID); err != nil {
		return err
	}

	// Atomic Redis update
	_, err = s.bidCache.AtomicBidUpdate(ctx, auctionID, userID, amount)
	if err != nil {
		s.log.Error("Failed to update bid", "error", err)
		err := s.userNotifier.NotifyUser(ctx, userID, map[string]interface{}{
			"type":           "bid_rejected",
			"reason":         err.Error(),
			"current_bid":    amount,
			"current_winner": userID,
		})
		if err != nil {
			return err
		}
		return err
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
