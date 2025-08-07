package websocket

import (
	"auction-system/internal/domain"
	"context"
)

type WebSocketNotifier struct {
	connManager domain.ConnectionManager
}

func NewWebSocketNotifier(connManager domain.ConnectionManager) *WebSocketNotifier {
	return &WebSocketNotifier{connManager: connManager}
}

func (n *WebSocketNotifier) NotifyUser(ctx context.Context, userID string, message interface{}) error {
	return n.connManager.NotifyUser(userID, message)
}

func (n *WebSocketNotifier) BroadcastToAuction(ctx context.Context, auctionID string, message interface{}) error {
	return n.connManager.BroadcastToAuction(auctionID, message)
}
