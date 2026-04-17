package handler

// order-5x.go
// ─────────────────────────────────────────────────────────────────────────────
// Simulation endpoints that intentionally trigger 5xx responses.
// Routes are mounted under /orders/sim/ so they never conflict with
// the real /orders/:id param route.
//
// Endpoints
//   GET /orders/sim/bad-column   → SELECT not_existed FROM orders → 500

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type OrderSimHandler struct {
	db  *pgxpool.Pool
	log *zap.Logger
}

func NewOrderSimHandler(db *pgxpool.Pool, log *zap.Logger) *OrderSimHandler {
	return &OrderSimHandler{db: db, log: log}
}

// BadColumn runs a query that references a column that does not exist in the
// orders table. PostgreSQL will return:
//   ERROR: column "not_existed" does not exist (SQLSTATE 42703)
// The handler logs DB_ERROR and responds 500.
func (h *OrderSimHandler) BadColumn(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	h.log.Info("sim: bad-column query triggered",
		zap.String("category", "SIM"),
		zap.String("request_id", reqID.(string)),
	)

	if h.db == nil {
		h.log.Error("sim: database pool is nil",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "database pool unavailable",
			"sim":      "bad-column",
			"category": "DB_ERROR",
		})
		return
	}

	// Intentionally bad query — column "not_existed" does not exist.
	var val string
	err := h.db.QueryRow(context.Background(),
		`SELECT not_existed FROM orders LIMIT 1`,
	).Scan(&val)

	if err != nil {
		h.log.Error("sim: query failed as expected",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "database error",
			"sim":      "bad-column",
			"detail":   err.Error(),
			"category": "DB_ERROR",
		})
		return
	}

	// Should never reach here.
	h.log.Warn("sim: bad-column query unexpectedly succeeded",
		zap.String("category", "SIM"),
		zap.String("request_id", reqID.(string)),
		zap.String("value", val),
	)
	c.JSON(http.StatusOK, gin.H{"sim": "bad-column", "value": val})
}
