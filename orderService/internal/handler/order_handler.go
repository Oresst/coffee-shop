package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/order-service/internal/domain"
	"github.com/yourusername/order-service/internal/service"
	"github.com/yourusername/order-service/pkg/logger"
	"go.uber.org/zap"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	place := "[OrderHandler.CreateOrder]"

	var req domain.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Warn(fmt.Sprintf("%s Ошибка в методе ShouldBindJSON", place),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Log.Info(fmt.Sprintf("%s Получен запрос на создание заказа", place),
		zap.Int64("user_id", req.UserID),
		zap.String("request_id", req.RequestId),
	)

	ctx := c.Request.Context()
	order, err := h.orderService.CreateOrder(ctx, &req)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка создания заказа", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", req.UserID),
			zap.String("request_id", req.RequestId),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Log.Info(fmt.Sprintf("%s Заказ создан успешно", place),
		logger.WithTraceID(ctx),
		zap.Int64("user_id", req.UserID),
		zap.String("request_id", req.RequestId),
	)

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	ctx := c.Request.Context()
	order, err := h.orderService.GetOrder(ctx, id)
	if err != nil {
		logger.Log.Error("Failed to get order",
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("order_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	ctx := c.Request.Context()
	orders, err := h.orderService.GetUserOrders(ctx, userID)
	if err != nil {
		logger.Log.Error("Failed to get user orders",
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", userID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
