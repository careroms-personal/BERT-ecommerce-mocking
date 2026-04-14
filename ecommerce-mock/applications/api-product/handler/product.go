package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"api-product/model"
	"api-product/repository"
)

type ProductHandler struct {
	repo *repository.ProductRepository
	log  *zap.Logger
}

func NewProductHandler(repo *repository.ProductRepository, log *zap.Logger) *ProductHandler {
	return &ProductHandler{repo: repo, log: log}
}

func (h *ProductHandler) Health(c *gin.Context) {
	h.log.Info("health check",
		zap.String("category", "SYSTEM"),
		zap.String("status", "ok"),
	)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-product"})
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	page := parseIntParam(c.Query("page"), 1)
	limit := parseIntParam(c.Query("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	category := c.Query("category")

	h.log.Info("listing products",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.String("filter_category", category),
	)

	products, total, err := h.repo.List(c.Request.Context(), page, limit, category)
	if err != nil {
		h.log.Error("failed to list products",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("products listed",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.Int("count", len(products)),
		zap.Int("total", total),
		zap.Int("page", page),
	)
	c.JSON(http.StatusOK, model.ListResponse{
		Data:  products,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	id := c.Param("id")

	h.log.Info("fetching product",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", id),
	)

	product, err := h.repo.GetByID(c.Request.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		h.log.Warn("product not found",
			zap.String("category", "NOT_FOUND"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found", "product_id": id})
		return
	}
	if err != nil {
		h.log.Error("failed to fetch product",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("product found",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", product.ID),
		zap.String("product_name", product.Name),
		zap.Float64("price", product.Price),
	)
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	var req model.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid create product request",
			zap.String("category", "ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("creating product",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.String("name", req.Name),
		zap.Float64("price", req.Price),
		zap.String("product_category", req.Category),
		zap.Int("initial_stock", req.Stock),
	)

	product, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		h.log.Error("failed to create product",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("name", req.Name),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("product created",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", product.ID),
		zap.String("name", product.Name),
		zap.Float64("price", product.Price),
	)
	c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	id := c.Param("id")

	var req model.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid update product request",
			zap.String("category", "ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("updating product",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", id),
	)

	product, err := h.repo.Update(c.Request.Context(), id, req)
	if errors.Is(err, repository.ErrNotFound) {
		h.log.Warn("product not found for update",
			zap.String("category", "NOT_FOUND"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found", "product_id": id})
		return
	}
	if err != nil {
		h.log.Error("failed to update product",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	h.log.Info("product updated",
		zap.String("category", "PRODUCT"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", product.ID),
		zap.String("name", product.Name),
		zap.Float64("price", product.Price),
	)
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) GetStock(c *gin.Context) {
	reqID, _ := c.Get("request_id")
	id := c.Param("id")

	h.log.Info("checking stock",
		zap.String("category", "STOCK"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", id),
	)

	stock, err := h.repo.GetStock(c.Request.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		h.log.Warn("product not found for stock check",
			zap.String("category", "NOT_FOUND"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found", "product_id": id})
		return
	}
	if err != nil {
		h.log.Error("failed to check stock",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if stock == 0 {
		h.log.Warn("product out of stock",
			zap.String("category", "STOCK"),
			zap.String("request_id", reqID.(string)),
			zap.String("product_id", id),
			zap.Int("stock", stock),
		)
		c.JSON(http.StatusOK, gin.H{"product_id": id, "stock": stock, "available": false})
		return
	}

	h.log.Info("stock level ok",
		zap.String("category", "STOCK"),
		zap.String("request_id", reqID.(string)),
		zap.String("product_id", id),
		zap.Int("stock", stock),
	)
	c.JSON(http.StatusOK, gin.H{"product_id": id, "stock": stock, "available": true})
}

func parseIntParam(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return fallback
	}
	return v
}
