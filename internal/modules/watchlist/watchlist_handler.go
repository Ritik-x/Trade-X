package watchlist

import (
	"net/http"

	"tradex/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *WatchlistService
}

func NewHandler(service *WatchlistService) *Handler {
	return &Handler{
		service: service,
	}
}

type createWatchlistRequest struct {
	Name string `json:"name" binding:"required"`
}

type addStockRequest struct {
	InstrumentID uuid.UUID `json:"instrument_id" binding:"required"`
}

// POST /api/v1/watchlists
func (h *Handler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req createWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "watchlist name is required",
		})
		return
	}

	watchlist, err := h.service.Create(
		c.Request.Context(),
		userID,
		req.Name,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "watchlist created successfully",
		"data":    watchlist,
	})
}

// GET /api/v1/watchlists
func (h *Handler) GetAll(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	watchlists, err := h.service.GetByUserID(
		c.Request.Context(),
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not retrieve watchlists",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": watchlists,
	})
}

// POST /api/v1/watchlists/:id/stocks
func (h *Handler) AddStock(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	watchlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid watchlist ID",
		})
		return
	}

	var req addStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "valid instrument_id is required",
		})
		return
	}

	err = h.service.AddStock(
		c.Request.Context(),
		userID,
		watchlistID,
		req.InstrumentID,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "stock added to watchlist successfully",
	})
}

// DELETE /api/v1/watchlists/:id/stocks/:instrument_id
func (h *Handler) RemoveStock(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	watchlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid watchlist ID",
		})
		return
	}

	instrumentID, err := uuid.Parse(c.Param("instrument_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid instrument ID",
		})
		return
	}

	err = h.service.RemoveStock(
		c.Request.Context(),
		userID,
		watchlistID,
		instrumentID,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "stock removed from watchlist successfully",
	})
}

// GET /api/v1/watchlists/:id/stocks
func (h *Handler) GetStocks(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	watchlistID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid watchlist ID",
		})
		return
	}

	stocks, err := h.service.GetWatchlistStocks(
		c.Request.Context(),
		userID,
		watchlistID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not retrieve watchlist stocks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stocks,
	})
}
