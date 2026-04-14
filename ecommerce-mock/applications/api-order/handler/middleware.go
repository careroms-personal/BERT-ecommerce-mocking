package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqID := uuid.New().String()
		c.Set("request_id", reqID)

		c.Next()

		status := c.Writer.Status()
		latency := time.Since(start)

		category := "ORDER"
		level := log.Info
		if status == 404 {
			category = "NOT_FOUND"
		} else if status == 409 {
			category = "CONFLICT_ERROR"
		} else if status >= 500 {
			category = "ERROR"
			level = log.Error
		}

		level("http request",
			zap.String("category", category),
			zap.String("request_id", reqID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
