package services

import (
	"context"
	"errors"
	"fmt"

	"auction-system/internal/domain"
	"auction-system/pkg/logger"
)

type EventListener struct {
	bidService        *BidService
	broadcaster       domain.AuctionBroadcaster
	connectionManager domain.ConnectionManager
	log               logger.Logger
}

func NewEventListener(bidService *BidService, connectionManager domain.ConnectionManager,
	broadcaster domain.AuctionBroadcaster, log logger.Logger) *EventListener {
	return &EventListener{
		bidService:        bidService,
		broadcaster:       broadcaster,
		connectionManager: connectionManager,
		log:               log,
	}
}

func (el *EventListener) Start(ctx context.Context, subscriber domain.EventSubscriber) error {
	el.log.Info("Starting event listener")
	return subscriber.SubscribeToBidEvents(ctx, el.handleBidEvent)
}

func (el *EventListener) handleBidEvent(event *domain.BidEvent) error {
	el.log.Info("Handling bid event", "type", event.Type, "auction_id", event.AuctionID)

	switch event.Type {
	case domain.BidAccepted:
		return el.handleBidAccepted(event)
	case domain.BidRejected:
		return el.handleBidRejected(event)
	case domain.AuctionEndedBidRejected:
		return el.handleAuctionEnded(event)
	case domain.AuctionExtended:
		return el.handleAuctionExtended(event)
	}

	return errors.New(fmt.Sprintf("unknown event type %+v", *event))
}

func (el *EventListener) handleBidAccepted(event *domain.BidEvent) error {
	// Update local cache
	el.bidService.UpdateLocalCache(event.AuctionID, event.Amount, event.UserID)

	// Broadcast to all connected users for this auction
	return el.broadcaster.BroadcastToAuction(context.Background(), event.AuctionID, map[string]interface{}{
		"type":           "bid_update",
		"current_bid":    event.Amount,
		"current_winner": event.UserID,
		"timestamp":      event.Timestamp,
	})
}

func (el *EventListener) handleBidRejected(event *domain.BidEvent) error {

	return nil
}

func (el *EventListener) handleAuctionEnded(event *domain.BidEvent) error {
	// Get final state from cache
	// Remove from local cache
	el.bidService.RemoveFromCache(event.AuctionID)

	// Final broadcast
	if err := el.broadcaster.BroadcastToAuction(context.Background(), event.AuctionID, map[string]interface{}{
		"type":      "auction_ended",
		"timestamp": event.Timestamp,
	}); err != nil {
		el.log.Error("Failed to broadcast auction ended event", "error", err)
		return err
	}

	if err := el.connectionManager.CloseAndUnregisterConnections(event.AuctionID); err != nil {
		el.log.Error("Failed to finalize connections for auction", "auction_id",
			event.AuctionID, "error", err)
		return err
	}
	return nil
}

func (el *EventListener) handleAuctionExtended(event *domain.BidEvent) error {
	return el.broadcaster.BroadcastToAuction(context.Background(), event.AuctionID, map[string]interface{}{
		"type":      "auction_extended",
		"timestamp": event.Timestamp,
	})
}
