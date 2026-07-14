package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/models"
	"github.com/THEGunDevil/NEXTJS-CRYPTO-PLATFORM-BACKEND/internal/service"
)

// OrderHandler holds dependencies for order-related HTTP endpoints.
type OrderHandler struct {
	orderSvc *service.OrderService
}

// NewOrderHandler creates a new OrderHandler with the injected service.
func NewOrderHandler(orderSvc *service.OrderService) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
}

// POST /api/orders
func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Calling PlaceOrder on the injected orderSvc instance
	order, err := h.orderSvc.PlaceOrder(c.Request.Context(), userID, req)
	if err != nil {
		switch err {
		case service.ErrInsufficientBalance:
			service.AbortWithError(c, http.StatusBadRequest, "insufficient balance")
		case service.ErrInvalidOrderType:
			service.AbortWithError(c, http.StatusBadRequest, "limit orders require a price")
		default:
			service.AbortWithError(c, http.StatusInternalServerError, "failed to place order")
		}
		return
	}

	service.WriteJSON(c, http.StatusCreated, order)
}

// DELETE /api/orders/:id
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		service.AbortWithError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	orderID, err := service.ParseUUIDParam(c, "id")
	if err != nil {
		service.AbortWithError(c, http.StatusBadRequest, "invalid order id")
		return
	}

	// Calling CancelOrder on the injected orderSvc instance
	if err := h.orderSvc.CancelOrder(c.Request.Context(), orderID, userID); err != nil {
		service.AbortWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}