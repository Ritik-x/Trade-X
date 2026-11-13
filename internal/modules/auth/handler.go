package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest

	// 1. Parse and validate JSON.
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request payload",
		})
		return
	}

	// 2. Execute business logic.
	ctx := c.Request.Context()

	response, err := h.service.Register(ctx, req)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already registered",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to register user",
		})
		return
	}

	// 3. Return the created user.
	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"data":    response,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request payload",
		})
		return
	}

	response, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})

		case errors.Is(err, ErrAccountInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"error": "account is inactive",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to login",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "login successful",
		"data":    response,
	})
}
