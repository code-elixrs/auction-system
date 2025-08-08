package handlers

import (
	"auction-system/internal/services"
	"auction-system/pkg/logger"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type AuctionHandler struct {
	auctionManager *services.AuctionManager
	log            logger.Logger
}

type CreateAuctionRequest struct {
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	StartingBid float64   `json:"starting_bid"`
}

type CreateAuctionResponse struct {
	AuctionID   string    `json:"auction_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	StartingBid float64   `json:"starting_bid"`
	Status      string    `json:"status"`
}

func NewAuctionHandler(auctionManager *services.AuctionManager, log logger.Logger) *AuctionHandler {
	return &AuctionHandler{
		auctionManager: auctionManager,
		log:            log,
	}
}

func (h *AuctionHandler) CreateAuction(c echo.Context) error {
	h.log.Info("CreateAuction endpoint called",
		"method", c.Request().Method,
		"remote_addr", c.RealIP(),
		"user_agent", c.Request().UserAgent(),
		"content_type", c.Request().Header.Get("Content-Type"))

	var req CreateAuctionRequest
	if err := c.Bind(&req); err != nil {
		h.log.Error("Failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validation
	if req.StartTime.Before(time.Now()) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Start time must be in the future"})
	}

	if req.EndTime.Before(req.StartTime) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "End time must be after start time"})
	}

	if req.StartingBid <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Starting bid must be positive"})
	}

	auction, err := h.auctionManager.CreateAuction(c.Request().Context(), req.StartTime, req.EndTime, req.StartingBid)
	if err != nil {
		h.log.Error("Failed to create auction", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create auction"})
	}

	response := CreateAuctionResponse{
		AuctionID:   auction.ID,
		StartTime:   auction.StartTime,
		EndTime:     auction.EndTime,
		StartingBid: req.StartingBid,
		Status:      auction.Status.String(),
	}

	h.log.Info("Auction created successfully", "auction_id", auction.ID)
	return c.JSON(http.StatusCreated, response)
}

func (h *AuctionHandler) GetAuction(c echo.Context) error {
	auctionID := c.Param("id")
	h.log.Info("GetAuction endpoint called", "auction_id", auctionID)

	return c.JSON(http.StatusOK, map[string]string{
		"auction_id": auctionID,
		"message":    "Auction details would be here",
	})
}

func (h *AuctionHandler) ExtendAuction(c echo.Context) error {
	auctionID := c.Param("id")
	extensionStr := c.QueryParam("seconds")

	h.log.Info("ExtendAuction endpoint called", "auction_id", auctionID, "seconds", extensionStr)

	if extensionStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Extension duration required"})
	}

	extensionDuration := 30 * time.Second
	if err := h.auctionManager.CheckAndExtendAuction(c.Request().Context(), auctionID, extensionDuration); err != nil {
		h.log.Error("Failed to extend auction", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to extend auction"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Auction extended successfully",
	})
}
