package shulker

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// NewLogger creates a new logger instance
func NewLogger(lc fx.Lifecycle) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	log, err := config.Build()
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return log.Sync()
		},
	})

	return log.Named("Shulker"), nil
}
