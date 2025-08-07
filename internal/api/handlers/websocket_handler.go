package handlers

import (
	"net/http"

	"auction-system/internal/domain"
	"auction-system/internal/infrastructure/websocket"
	"auction-system/internal/services"
	"auction-system/pkg/logger"

	_ "github.com/gorilla/mux"
)

type WebSocketHandlers struct {
	wsHandler *websocket.WebSocketHandler
}

func NewWebSocketHandlers(bidService *services.BidService, auctionRepo domain.AuctionRepository,
	connManager *websocket.ConnectionManager, log logger.Logger) *WebSocketHandlers {
	wsHandler := websocket.NewWebSocketHandler(bidService, auctionRepo, connManager, log)
	return &WebSocketHandlers{
		wsHandler: wsHandler,
	}
}

func (h *WebSocketHandlers) HandleConnection(w http.ResponseWriter, r *http.Request) {
	h.wsHandler.HandleConnection(w, r)
}
