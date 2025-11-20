package http

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"time"
	"wb_l0/internal/domain"
	"wb_l0/internal/usecase"
	"wb_l0/pkg/prometheus"
)

type OrderHandler struct {
	uc  *usecase.OrderUsecase
	log *slog.Logger
}

type OrderStatus string

const (
	OrderStatusSuccess       OrderStatus = "success"
	OrderStatusBadRequest    OrderStatus = "bad_request"
	OrderStatusInvalidUID    OrderStatus = "invalid_uid"
	OrderStatusNotFound      OrderStatus = "not_found"
	OrderStatusInternalError OrderStatus = "internal_error"
)

func NewOrderHandler(uc *usecase.OrderUsecase, logger *slog.Logger) *OrderHandler {
	return &OrderHandler{
		uc:  uc,
		log: logger,
	}
}

// GetOrderByUID возвращает заказ по order_uid
// @Summary Get order by UID
// @Description Get order details by order_uid
// @Tags orders
// @Produce json
// @Param order_uid path string true "Order UID"
// @Success 200 {object} domain.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /order/{order_uid} [get]
func (h *OrderHandler) GetOrderByUID(c *gin.Context) {
	startTime := time.Now()
	orderUID := c.Param("order_uid")

	status := OrderStatusSuccess

	defer func() {
		prometheus.OrdersProcessed.WithLabelValues(string(status)).Inc()
		prometheus.OrderProcessingDuration.Observe(time.Since(startTime).Seconds())

		h.log.Info("Order request completed",
			"order_uid", orderUID,
			"status", status,
			"duration_ms", time.Since(startTime).Milliseconds(),
		)
	}()

	if orderUID == "" {
		status = OrderStatusBadRequest
		h.log.Error("Order_uid is empty")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "order_uid is required",
		})
		return
	}

	if len(orderUID) != 20 {
		status = OrderStatusInvalidUID
		h.log.Error("Order_uid is invalid")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_order_uid",
			"message": "order_uid must be 20 characters long",
		})
		return
	}

	order, err := h.uc.GetOrder(c.Request.Context(), orderUID)
	if err != nil {
		if err == domain.ErrRecordNotFound {
			status = OrderStatusNotFound
			h.log.Error("Order not found", "orderUID", orderUID)
			c.JSON(http.StatusNotFound, gin.H{
				"error":     "not_found",
				"message":   "order not found",
				"order_uid": orderUID,
			})
			return
		}

		h.log.Error("Failed to get order", "error", err, "orderUID", orderUID)
		status = OrderStatusInternalError
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "failed to retrieve order",
		})
		return
	}

	duration := time.Since(startTime)
	h.log.Info("Order retrieved", "order_uid", orderUID, "duration", duration)

	c.Header("X-Execution-Time-MS", fmt.Sprintf("%d", time.Since(startTime).Milliseconds()))
	c.Header("X-Server-Timestamp", time.Now().Format(time.RFC3339))

	c.JSON(http.StatusOK, order)
}

// HealthCheck endpoint
// @Summary Health check
// @Description Check if service is healthy
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *OrderHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "order-api",
	})
}
