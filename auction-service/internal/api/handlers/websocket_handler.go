package handlers

import (
	"auction-system/internal/domain/repositories"
	"net/http"

	"auction-system/internal/infrastructure/websocket"
	"auction-system/internal/services"
	"auction-system/pkg/logger"

	_ "github.com/gorilla/mux"
)

type WebSocketHandlers struct {
	wsHandler *websocket.WebSocketHandler
}

func NewWebSocketHandlers(bidService *services.BidService, auctionRepo repositories.AuctionRepository,
	connManager *websocket.ConnectionManager, log logger.Logger) *WebSocketHandlers {
	wsHandler := websocket.NewWebSocketHandler(bidService, auctionRepo, connManager, log)
	return &WebSocketHandlers{
		wsHandler: wsHandler,
	}
}

func (h *WebSocketHandlers) HandleConnection(w http.ResponseWriter, r *http.Request) {
	h.wsHandler.HandleConnection(w, r)
}
