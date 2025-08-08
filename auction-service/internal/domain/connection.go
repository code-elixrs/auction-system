package domain

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
