package shulker

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	m *Minecraft
}

type NewControllerInput struct {
	fx.In

	Log       *zap.Logger
	Minecraft *Minecraft
	Lifecycle fx.Lifecycle
}

func NewController(ctx context.Context, in NewControllerInput) (*Controller, error) {
	c := &Controller{
		m: in.Minecraft,
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return c, nil
}
