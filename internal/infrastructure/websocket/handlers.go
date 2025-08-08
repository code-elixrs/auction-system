package websocket

import (
	"auction-system/internal/domain/repositories"
	"context"
	"net/http"
	"strconv"
	"time"

	"auction-system/internal/domain"
	"auction-system/internal/services"
	"auction-system/pkg/logger"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type WebSocketHandler struct {
	bidService  *services.BidService
	auctionRepo repositories.AuctionRepository
	connManager domain.ConnectionManager
	log         logger.Logger
}

func NewWebSocketHandler(bidService *services.BidService,
	auctionRepo repositories.AuctionRepository,
	connManager domain.ConnectionManager, log logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		bidService:  bidService,
		connManager: connManager,
		auctionRepo: auctionRepo,
		log:         log,
	}
}

func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	auctionID := vars["auctionID"]

	// TODO this is not working, both for not allowing before auction start as well as after auction end
	// Check if auction exists and is active
	auction, err := h.auctionRepo.GetAuction(r.Context(), auctionID)
	if err != nil {
		h.log.Error("Failed to find auction", "error", err, "auctionID", auctionID)
		http.Error(w, "auction not found", http.StatusNotFound)
		return
	}

	// Check auction status
	now := time.Now()

	if now.After(auction.EndTime) {
		h.log.Info("Rejected connection - auction has ended", "auctionID", auctionID)
		http.Error(w, "auction has already ended", http.StatusForbidden)
		return
	}

	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("Failed to upgrade connection", "error", err)
		return
	}

	wsConn := NewWebSocketConnection(conn, userID, auctionID, h.log)

	// Register connection
	if err := h.connManager.RegisterConnection(userID, auctionID, wsConn); err != nil {
		h.log.Error("Failed to register connection", "error", err)
		err := conn.Close()
		if err != nil {
			return
		}
		return
	}

	// Initialize auction cache for this connection
	if err := h.bidService.HandleWebSocketConnection(userID, auctionID); err != nil {
		h.log.Error("Failed to initialize auction cache", "error", err)
	}

	// Start message handling
	go h.handleMessages(wsConn, userID, auctionID)
}

func (h *WebSocketHandler) handleMessages(conn *WebSocketConnection, userID, auctionID string) {
	defer func() {
		h.connManager.UnregisterConnection(userID, auctionID)
		conn.Close()
	}()

	for {
		if auction, err := h.auctionRepo.GetAuction(context.Background(), auctionID); err != nil {
			if auction.EndTime.Before(time.Now()) {
				break
			}
		}
		var msg map[string]interface{}
		err := conn.conn.ReadJSON(&msg)
		if err != nil {
			h.log.Error("Failed to read message", "error", err)
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "place_bid":
			h.handleBidMessage(conn, userID, auctionID, msg)
		case "ping":
			conn.Send(map[string]string{"type": "pong"})
		}
	}
}

func (h *WebSocketHandler) handleBidMessage(conn *WebSocketConnection, userID, auctionID string, msg map[string]interface{}) {
	amountStr, ok := msg["amount"].(string)
	if !ok {
		conn.Send(map[string]string{"type": "error", "message": "invalid amount"})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		conn.Send(map[string]string{"type": "error", "message": "invalid amount format"})
		return
	}

	if err := h.bidService.PlaceBid(context.Background(), auctionID, userID, amount); err != nil {
		h.log.Error("Failed to place bid", "error", err)
		conn.Send(map[string]string{"type": "error", "message": "failed to place bid"})
	}
}

type WebSocketConnection struct {
	conn      *websocket.Conn
	userID    string
	auctionID string
	log       logger.Logger
}

func NewWebSocketConnection(conn *websocket.Conn, userID, auctionID string, log logger.Logger) *WebSocketConnection {
	return &WebSocketConnection{
		conn:      conn,
		userID:    userID,
		auctionID: auctionID,
		log:       log,
	}
}

func (wsc *WebSocketConnection) Send(message interface{}) error {
	return wsc.conn.WriteJSON(message)
}

func (wsc *WebSocketConnection) Close() error {
	return wsc.conn.Close()
}

func (wsc *WebSocketConnection) UserID() string {
	return wsc.userID
}

func (wsc *WebSocketConnection) AuctionID() string {
	return wsc.auctionID
}
