package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"api-order/config"
	"api-order/handler"
	"api-order/logger"
	"api-order/repository"
)

func main() {
	cfg := config.Load()
	log := logger.New()
	defer log.Sync()

	log.Info("service starting",
		zap.String("category", "SYSTEM"),
		zap.String("version", "1.0.0"),
		zap.String("port", cfg.Port),
	)

	db, err := repository.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to parse database config, requests will fail",
			zap.String("category", "DB_ERROR"),
			zap.Error(err),
		)
	} else {
		defer db.Close()
		log.Info("database pool created (lazy connect)", zap.String("category", "SYSTEM"))
		if err := repository.Migrate(db); err != nil {
			log.Error("migration failed, continuing without schema",
				zap.String("category", "DB_ERROR"),
				zap.Error(err),
			)
		} else {
			log.Info("database migration ok", zap.String("category", "SYSTEM"))
		}
	}

	repo := repository.NewOrderRepository(db)
	h := handler.NewOrderHandler(repo, log, cfg.CartServiceURL, cfg.ProductServiceURL)
	sim := handler.NewOrderSimHandler(db, log)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(handler.RequestLogger(log))

	r.GET("/health", h.Health)
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/sim/bad-column", sim.BadColumn)
	r.POST("/orders/sim/bad-insert", sim.BadInsert)
	r.PUT("/orders/sim/bad-update", sim.BadUpdate)
	r.DELETE("/orders/sim/bad-delete", sim.BadDelete)
	r.GET("/orders/:id", h.GetOrder)
	r.GET("/orders/customer/:customerId", h.ListByCustomer)
	r.PUT("/orders/:id/status", h.UpdateStatus)
	r.DELETE("/orders/:id", h.CancelOrder)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("server listening",
			zap.String("category", "SYSTEM"),
			zap.String("addr", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error",
				zap.String("category", "ERROR"),
				zap.Error(err),
			)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutdown signal received", zap.String("category", "SYSTEM"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", zap.String("category", "ERROR"), zap.Error(err))
	}

	log.Info("service stopped", zap.String("category", "SYSTEM"))
}
