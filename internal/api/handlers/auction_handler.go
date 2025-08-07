package handlers

//
//import (
//	"auction-system/internal/services"
//	"auction-system/pkg/logger"
//	"encoding/json"
//	"net/http"
//	"strconv"
//	"time"
//
//	"github.com/gorilla/mux"
//)
//
//type AuctionHandler struct {
//	auctionManager *services.AuctionManager
//	log            logger.Logger
//}
//
//type CreateAuctionRequest struct {
//	StartTime   time.Time `json:"start_time"`
//	EndTime     time.Time `json:"end_time"`
//	StartingBid float64   `json:"starting_bid"`
//}
//
//type CreateAuctionResponse struct {
//	AuctionID   string    `json:"auction_id"`
//	StartTime   time.Time `json:"start_time"`
//	EndTime     time.Time `json:"end_time"`
//	StartingBid float64   `json:"starting_bid"`
//	Status      string    `json:"status"`
//}
//
//func NewAuctionHandler(auctionManager *services.AuctionManager, log logger.Logger) *AuctionHandler {
//	return &AuctionHandler{
//		auctionManager: auctionManager,
//		log:            log,
//	}
//}
//
//func (h *AuctionHandler) CreateAuction(w http.ResponseWriter, r *http.Request) {
//	var req CreateAuctionRequest
//	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
//		http.Error(w, "Invalid request body", http.StatusBadRequest)
//		return
//	}
//
//	// Validation
//	if req.StartTime.Before(time.Now()) {
//		http.Error(w, "Start time must be in the future", http.StatusBadRequest)
//		return
//	}
//
//	if req.EndTime.Before(req.StartTime) {
//		http.Error(w, "End time must be after start time", http.StatusBadRequest)
//		return
//	}
//
//	if req.StartingBid <= 0 {
//		http.Error(w, "Starting bid must be positive", http.StatusBadRequest)
//		return
//	}
//
//	auction, err := h.auctionManager.CreateAuction(r.Context(), req.StartTime, req.EndTime, req.StartingBid)
//	if err != nil {
//		h.log.Error("Failed to create auction", "error", err)
//		http.Error(w, "Failed to create auction", http.StatusInternalServerError)
//		return
//	}
//
//	response := CreateAuctionResponse{
//		AuctionID:   auction.ID,
//		StartTime:   auction.StartTime,
//		EndTime:     auction.EndTime,
//		StartingBid: req.StartingBid,
//		Status:      auction.Status.String(),
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(http.StatusCreated)
//	json.NewEncoder(w).Encode(response)
//}
//
//func (h *AuctionHandler) GetAuction(w http.ResponseWriter, r *http.Request) {
//	vars := mux.Vars(r)
//	auctionID := vars["id"]
//
//	// This would require adding GetAuction method to AuctionManager
//	// For now, return a simple response
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(map[string]string{
//		"auction_id": auctionID,
//		"message":    "Auction details would be here",
//	})
//}
//
//func (h *AuctionHandler) CreateAuctionWithLogging(w http.ResponseWriter, r *http.Request) {
//	h.log.Info("CreateAuction called",
//		"method", r.Method,
//		"content_type", r.Header.Get("Content-Type"),
//		"content_length", r.Header.Get("Content-Length"))
//
//	// Set CORS headers explicitly (belt and suspenders approach)
//	w.Header().Set("Access-Control-Allow-Origin", "*")
//	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
//	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
//
//	if r.Method == "OPTIONS" {
//		h.log.Info("Handling OPTIONS preflight request")
//		w.WriteHeader(http.StatusOK)
//		return
//	}
//
//	// Call the original CreateAuction method
//	h.CreateAuction(w, r)
//}
//
//func (h *AuctionHandler) ExtendAuction(w http.ResponseWriter, r *http.Request) {
//	vars := mux.Vars(r)
//	auctionID := vars["id"]
//
//	extensionStr := r.URL.Query().Get("seconds")
//	if extensionStr == "" {
//		http.Error(w, "Extension duration required", http.StatusBadRequest)
//		return
//	}
//
//	extensionSeconds, err := strconv.Atoi(extensionStr)
//	if err != nil || extensionSeconds <= 0 {
//		http.Error(w, "Invalid extension duration", http.StatusBadRequest)
//		return
//	}
//
//	extensionDuration := time.Duration(extensionSeconds) * time.Second
//
//	if err := h.auctionManager.CheckAndExtendAuction(r.Context(), auctionID, extensionDuration); err != nil {
//		h.log.Error("Failed to extend auction", "error", err)
//		http.Error(w, "Failed to extend auction", http.StatusInternalServerError)
//		return
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(map[string]string{
//		"message": "Auction extended successfully",
//	})
//}
