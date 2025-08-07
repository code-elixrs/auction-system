package websocket

import (
	"auction-system/internal/domain"
	"auction-system/pkg/logger"
	"encoding/json"
	"sync"
)

type ConnectionManager struct {
	connections map[string]map[string]domain.WebSocketConnection // auctionID -> userID -> connection
	userConns   map[string][]domain.WebSocketConnection          // userID -> connections
	mutex       sync.RWMutex
	log         logger.Logger
}

func NewConnectionManager(log logger.Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]map[string]domain.WebSocketConnection),
		userConns:   make(map[string][]domain.WebSocketConnection),
		log:         log,
	}
}

func (cm *ConnectionManager) RegisterConnection(userID, auctionID string, conn domain.WebSocketConnection) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Register by auction
	if cm.connections[auctionID] == nil {
		cm.connections[auctionID] = make(map[string]domain.WebSocketConnection)
	}
	cm.connections[auctionID][userID] = conn

	// Register by user
	cm.userConns[userID] = append(cm.userConns[userID], conn)

	cm.log.Info("Connection registered", "user_id", userID, "auction_id", auctionID)
	return nil
}

func (cm *ConnectionManager) UnregisterConnection(userID, auctionID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Remove from auction connections
	if auctionConns, exists := cm.connections[auctionID]; exists {
		delete(auctionConns, userID)
		if len(auctionConns) == 0 {
			delete(cm.connections, auctionID)
		}
	}

	// Remove from user connections
	if userConnections, exists := cm.userConns[userID]; exists {
		var newConns []domain.WebSocketConnection
		for _, existingConn := range userConnections {
			if existingConn.AuctionID() != auctionID {
				newConns = append(newConns, existingConn)
			}
		}

		if len(newConns) == 0 {
			delete(cm.userConns, userID)
		} else {
			cm.userConns[userID] = newConns
		}
	}

	cm.log.Info("Connection unregistered", "user_id", userID, "auction_id", auctionID)
	return nil
}

func (cm *ConnectionManager) CloseAndUnregisterConnections(auctionID string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Remove from auction connections
	if auctionConns, exists := cm.connections[auctionID]; exists {
		for userID, conn := range auctionConns {
			if err := conn.Close(); err != nil {
				cm.log.Error("Failed to close connection", "user_id", userID,
					"auction_id", auctionID, "error", err)
			} else {
				cm.log.Info("Closed connection", "user_id", userID, "auction_id", auctionID)
			}

			// Remove this connection from user connections
			if userConnections, exists := cm.userConns[userID]; exists {
				var newConns []domain.WebSocketConnection
				for _, existingConn := range userConnections {
					if existingConn.AuctionID() != auctionID {
						newConns = append(newConns, existingConn)
					}
				}

				if len(newConns) == 0 {
					delete(cm.userConns, userID)
				} else {
					cm.userConns[userID] = newConns
				}
			}
		}
		delete(cm.connections, auctionID)
	}

	cm.log.Info("Connections closed for auction", "auction_id", auctionID)
	return nil
}

func (cm *ConnectionManager) GetConnectionsForAuction(auctionID string) []domain.WebSocketConnection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	var connections []domain.WebSocketConnection
	if auctionConns, exists := cm.connections[auctionID]; exists {
		for _, conn := range auctionConns {
			connections = append(connections, conn)
		}
	}

	return connections
}

func (cm *ConnectionManager) GetConnectionsForUser(userID string) []domain.WebSocketConnection {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if connections, exists := cm.userConns[userID]; exists {
		return connections
	}

	return nil
}

func (cm *ConnectionManager) BroadcastToAuction(auctionID string, message interface{}) error {
	cm.log.Info("Broadcasting to auction", "auctionId", auctionID, ", message:", message)
	connections := cm.GetConnectionsForAuction(auctionID)
	cm.log.Info("No. of Connections:", len(connections))
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	for _, conn := range connections {
		if err := conn.Send(messageBytes); err != nil {
			cm.log.Error("Failed to send message", "user_id", conn.UserID(),
				"message", string(messageBytes), "error", err)
			// Continue to other connections
		} else {
			cm.log.Info("Sent message", "user_id", conn.UserID(), "message", string(messageBytes))
		}
	}

	return nil
}

func (cm *ConnectionManager) NotifyUser(userID string, message interface{}) error {
	connections := cm.GetConnectionsForUser(userID)

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	for _, conn := range connections {
		if err := conn.Send(messageBytes); err != nil {
			cm.log.Error("Failed to send message", "user_id", userID, "error", err)
		}
	}

	return nil
}
