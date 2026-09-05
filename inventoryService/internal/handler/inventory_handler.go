package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-service/internal/domain"
	"github.com/yourusername/inventory-service/internal/service"
	"github.com/yourusername/inventory-service/pkg/logger"
	"go.uber.org/zap"
)

type InventoryHandler struct {
	inventoryService *service.InventoryService
}

func NewInventoryHandler(inventoryService *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventoryService: inventoryService}
}

func (h *InventoryHandler) CheckAvailability(c *gin.Context) {
	var req domain.CheckAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	available, err := h.inventoryService.CheckAvailability(ctx, req.Items)
	if err != nil {
		logger.Log.Error("Check availability failed",
			logger.WithTraceID(ctx),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !available {
		c.JSON(http.StatusConflict, gin.H{"available": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"available": true})
}

func (h *InventoryHandler) ReserveItems(c *gin.Context) {
	place := "[InventoryHandler.ReserveItems]"

	var req domain.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.inventoryService.ReserveItems(ctx, &req)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("%s Ошибка резервации товаров", place),
			logger.WithTraceID(ctx),
			zap.Error(err),
			zap.Int64("user_id", req.UserId),
			zap.String("request_id", req.RequestID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusConflict, resp)
		return
	}

	logger.Log.Info(fmt.Sprintf("%s Товары успешно зарезервированы", place),
		logger.WithTraceID(ctx),
		zap.Int64("user_id", req.UserId),
		zap.String("request_id", req.RequestID),
	)

	c.JSON(http.StatusOK, resp)
}

func (h *InventoryHandler) CancelReservation(c *gin.Context) {
	var req struct {
		RequestID string `json:"request_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.inventoryService.CancelReservation(ctx, req.RequestID); err != nil {
		logger.Log.Error("Cancel reservation failed",
			logger.WithTraceID(ctx),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (h *InventoryHandler) ConfirmReservation(c *gin.Context) {
	var req struct {
		RequestID string `json:"request_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.inventoryService.ConfirmReservation(ctx, req.RequestID); err != nil {
		logger.Log.Error("Confirm reservation failed",
			logger.WithTraceID(ctx),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}
