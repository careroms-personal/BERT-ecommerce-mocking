package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"api-order/model"
	"api-order/repository"
)

type OrderHandler struct {
	repo              *repository.OrderRepository
	log               *zap.Logger
	cartServiceURL    string
	productServiceURL string
	httpClient        *http.Client
}

func NewOrderHandler(repo *repository.OrderRepository, log *zap.Logger, cartURL, productURL string) *OrderHandler {
	return &OrderHandler{
		repo:              repo,
		log:               log,
		cartServiceURL:    cartURL,
		productServiceURL: productURL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *OrderHandler) Health(c *gin.Context) {
	h.log.Info("health check", zap.String("category", "SYSTEM"), zap.String("status", "ok"))
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-order"})
}

// POST /orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid create order request",
			zap.String("category", "ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("creating order",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("customer_id", req.CustomerID),
		zap.Int("item_count", len(req.Items)),
	)

	// Deduct stock for each item via api-product
	for _, item := range req.Items {
		url := fmt.Sprintf("%s/products/%s/stock", h.productServiceURL, item.ProductID)
		stockReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
		resp, err := h.httpClient.Do(stockReq)
		if err != nil || resp.StatusCode != http.StatusOK {
			h.log.Error("stock check failed during order creation",
				zap.String("category", "DB_TIMEOUT"),
				zap.String("request_id", reqID.(string)),
				zap.String("product_id", item.ProductID),
			)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "stock service unavailable"})
			return
		}
		resp.Body.Close()
	}

	// Create with timeout to simulate DB_TIMEOUT chaos
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	order, err := h.repo.Create(ctx, req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			h.log.Error("order creation timed out",
				zap.String("category", "DB_TIMEOUT"),
				zap.String("request_id", reqID.(string)),
				zap.String("customer_id", req.CustomerID),
			)
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "database timeout"})
			return
		}
		h.log.Error("failed to create order",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("customer_id", req.CustomerID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("order created",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("order_id", order.ID),
		zap.String("customer_id", order.CustomerID),
		zap.String("status", string(order.Status)),
		zap.Float64("total_amount", order.TotalAmount),
	)
	c.JSON(http.StatusCreated, order)
}

// GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	id := c.Param("id")

	h.log.Info("fetching order",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("order_id", id),
	)

	order, err := h.repo.GetByID(c.Request.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		h.log.Warn("order not found",
			zap.String("category", "NOT_FOUND"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found", "order_id": id})
		return
	}
	if err != nil {
		h.log.Error("failed to fetch order",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("order fetched",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("order_id", order.ID),
		zap.String("status", string(order.Status)),
	)
	c.JSON(http.StatusOK, order)
}

// GET /orders/customer/:customerId
func (h *OrderHandler) ListByCustomer(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	customerID := c.Param("customerId")

	h.log.Info("listing orders for customer",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("customer_id", customerID),
	)

	orders, err := h.repo.ListByCustomer(c.Request.Context(), customerID)
	if err != nil {
		h.log.Error("failed to list orders",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("customer_id", customerID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("orders listed",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("customer_id", customerID),
		zap.Int("count", len(orders)),
	)
	c.JSON(http.StatusOK, gin.H{"data": orders, "count": len(orders)})
}

// PUT /orders/:id/status
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	id := c.Param("id")

	var req model.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid update status request",
			zap.String("category", "ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("updating order status",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("order_id", id),
		zap.String("new_status", req.Status),
	)

	order, err := h.repo.UpdateStatus(c.Request.Context(), id, model.OrderStatus(req.Status))
	if errors.Is(err, repository.ErrNotFound) {
		h.log.Warn("order not found for status update",
			zap.String("category", "NOT_FOUND"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found", "order_id": id})
		return
	}
	if err != nil {
		h.log.Error("failed to update order status",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if req.Status == string(model.StatusConfirmed) {
		h.log.Info("order confirmed",
			zap.String("category", "ORDER"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", order.ID),
			zap.String("customer_id", order.CustomerID),
			zap.Float64("total_amount", order.TotalAmount),
		)
	} else {
		h.log.Info("order status updated",
			zap.String("category", "ORDER"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", order.ID),
			zap.String("status", string(order.Status)),
		)
	}

	c.JSON(http.StatusOK, order)
}

// DELETE /orders/:id
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	id := c.Param("id")

	h.log.Info("cancel order requested",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("order_id", id),
	)

	order, err := h.repo.GetByID(c.Request.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		h.log.Warn("order not found for cancellation",
			zap.String("category", "NOT_FOUND"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found", "order_id": id})
		return
	}
	if err != nil {
		h.log.Error("failed to fetch order for cancellation",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if !order.CanCancel() {
		h.log.Warn("order cannot be cancelled",
			zap.String("category", "CONFLICT_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.String("current_status", string(order.Status)),
		)
		c.JSON(http.StatusConflict, gin.H{
			"error":          "order cannot be cancelled",
			"order_id":       id,
			"current_status": order.Status,
		})
		return
	}

	if order.CancelIsConflict() {
		h.log.Warn("cancelling confirmed order — payment may have been processed",
			zap.String("category", "CONFLICT_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.String("current_status", string(order.Status)),
			zap.Float64("total_amount", order.TotalAmount),
		)
	}

	cancelled, err := h.repo.UpdateStatus(c.Request.Context(), id, model.StatusCancelled)
	if err != nil {
		h.log.Error("failed to cancel order",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("order_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("order cancelled",
		zap.String("category", "ORDER"),
		zap.String("request_id", reqID.(string)),
		zap.String("order_id", cancelled.ID),
		zap.String("customer_id", cancelled.CustomerID),
	)
	c.JSON(http.StatusOK, cancelled)
}
