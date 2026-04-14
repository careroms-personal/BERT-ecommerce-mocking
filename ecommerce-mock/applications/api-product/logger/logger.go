package logger

import "go.uber.org/zap"

func New() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	log, _ := cfg.Build(
		zap.Fields(zap.String("service", "api-product")),
	)
	return log
}
