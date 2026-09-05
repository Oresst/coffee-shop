package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/saga-service/internal/domains"
	"github.com/yourusername/saga-service/internal/services"
	"net/http"
)

type OrderSagaHandler struct {
	OrderSagaService *services.OrderSagaService
}

func NewOrderSagaHandler(orderSagaService *services.OrderSagaService) *OrderSagaHandler {
	return &OrderSagaHandler{
		OrderSagaService: orderSagaService,
	}
}

func (h *OrderSagaHandler) StartSaga(c *gin.Context) {
	var request domains.CreateOrderSagaRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	result, err := h.OrderSagaService.StartSaga(ctx, request.UserId, request.Items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id": result.RequestID,
		"status":     "ok",
	})
}
