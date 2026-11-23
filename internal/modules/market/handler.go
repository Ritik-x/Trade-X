package market

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetAll(c *gin.Context) {
	instruments, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not retrieve instruments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": instruments,
	})
}

func (h *Handler) GetBySymbol(c *gin.Context) {
	exchange := c.Param("exchange")
	symbol := c.Param("symbol")

	instrument, err := h.service.GetBySymbol(
		c.Request.Context(),
		exchange,
		symbol,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "instrument not found",
			})
			return
		}

		if err.Error() == "unsupported exchange" || err.Error() == "symbol is required" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not retrieve instrument",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": instrument,
	})
}

func (h *Handler) GetQuote(c *gin.Context) {
	quote, err := h.service.GetQuote(
		c.Request.Context(),
		c.Param("exchange"),
		c.Param("symbol"),
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "instrument not found",
			})
			return
		}

		if err.Error() == "quote not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "quote not available",
			})
			return
		}

		if err.Error() == "unsupported exchange" || err.Error() == "symbol is required" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not retrieve market quote",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": quote,
	})
}

func (h *Handler) RefreshQuote(c *gin.Context) {
	quote, err := h.service.FetchAndStoreQuote(
		c.Request.Context(),
		c.Param("exchange"),
		c.Param("symbol"),
	)

	if err != nil {
		log.Printf(
			"REFRESH QUOTE ERROR | exchange=%s | symbol=%s | error=%+v",
			c.Param("exchange"),
			c.Param("symbol"),
			err,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "instrument not found",
			})
			return
		}

		if err.Error() == "unsupported exchange" ||
			err.Error() == "symbol is required" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"error": "could not refresh market quote from NSE",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "market quote refreshed successfully",
		"data":    quote,
	})
}
