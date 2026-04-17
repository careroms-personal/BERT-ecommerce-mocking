package handler

// product-5x.go
// ─────────────────────────────────────────────────────────────────────────────
// Simulation endpoints that intentionally trigger 5xx responses.
// Routes are mounted under /products/sim/ so they never conflict with
// the real /products/:id param route.
//
// Endpoints
//   GET    /products/sim/bad-column   → SELECT not_existed FROM products → 500
//   POST   /products/sim/bad-insert   → INSERT INTO products (not_existed) → 500
//   DELETE /products/sim/bad-delete   → DELETE FROM products WHERE not_existed → 500

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type SimHandler struct {
	db  *pgxpool.Pool
	log *zap.Logger
}

func NewSimHandler(db *pgxpool.Pool, log *zap.Logger) *SimHandler {
	return &SimHandler{db: db, log: log}
}

// BadColumn runs a query that references a column that does not exist in the
// products table. PostgreSQL will return:
//   ERROR: column "not_existed" does not exist (SQLSTATE 42703)
// The handler logs DB_ERROR and responds 500.
func (h *SimHandler) BadColumn(c *gin.Context) {
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
		`SELECT not_existed FROM products LIMIT 1`,
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

	// Should never reach here — included for completeness.
	h.log.Warn("sim: bad-column query unexpectedly succeeded",
		zap.String("category", "SIM"),
		zap.String("request_id", reqID.(string)),
		zap.String("value", val),
	)
	c.JSON(http.StatusOK, gin.H{"sim": "bad-column", "value": val})
}

// BadInsert runs an INSERT that references a column that does not exist in the
// products table. PostgreSQL will return:
//   ERROR: column "not_existed" of relation "products" does not exist (SQLSTATE 42703)
func (h *SimHandler) BadInsert(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	h.log.Info("sim: bad-insert triggered",
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
			"sim":      "bad-insert",
			"category": "DB_ERROR",
		})
		return
	}

	// Intentionally bad INSERT — column "not_existed" does not exist.
	_, err := h.db.Exec(context.Background(),
		`INSERT INTO products (not_existed) VALUES ('sim')`,
	)

	if err != nil {
		h.log.Error("sim: insert failed as expected",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "database error",
			"sim":      "bad-insert",
			"detail":   err.Error(),
			"category": "DB_ERROR",
		})
		return
	}

	// Should never reach here.
	h.log.Warn("sim: bad-insert unexpectedly succeeded",
		zap.String("category", "SIM"),
		zap.String("request_id", reqID.(string)),
	)
	c.JSON(http.StatusOK, gin.H{"sim": "bad-insert", "result": "unexpected_success"})
}

// BadDelete runs a DELETE that references a column that does not exist in the
// products table. PostgreSQL will return:
//   ERROR: column "not_existed" does not exist (SQLSTATE 42703)
func (h *SimHandler) BadDelete(c *gin.Context) {
	reqID, _ := c.Get("request_id")

	h.log.Info("sim: bad-delete triggered",
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
			"sim":      "bad-delete",
			"category": "DB_ERROR",
		})
		return
	}

	// Intentionally bad DELETE — column "not_existed" does not exist.
	_, err := h.db.Exec(context.Background(),
		`DELETE FROM products WHERE not_existed = 'sim'`,
	)

	if err != nil {
		h.log.Error("sim: delete failed as expected",
			zap.String("category", "DB_ERROR"),
			zap.String("request_id", reqID.(string)),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "database error",
			"sim":      "bad-delete",
			"detail":   err.Error(),
			"category": "DB_ERROR",
		})
		return
	}

	// Should never reach here.
	h.log.Warn("sim: bad-delete unexpectedly succeeded",
		zap.String("category", "SIM"),
		zap.String("request_id", reqID.(string)),
	)
	c.JSON(http.StatusOK, gin.H{"sim": "bad-delete", "result": "unexpected_success"})
}
