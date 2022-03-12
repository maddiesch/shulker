package main

import (
	"context"
	"flag"
	"os"
	"shulker-box/shulker"
	"shulker-box/shulker/utility"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	var params shulker.Params

	flag.StringVar(&params.ConfigFile, "config", "config.shulker.hcl", "path to the shulker configuration file")

	flag.Parse()

	ctx := context.Background()

	app := fx.New(
		shulker.Module,
		fx.Supply(fx.Annotate(ctx, fx.As(new(context.Context)))),
		fx.Supply(params),
		fx.StartTimeout(1*time.Minute),
		fx.StopTimeout(2*time.Minute),
		fx.Invoke(SetupEnvironment),
		fx.Invoke(Run),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),
		fx.NopLogger,
	)

	app.Run()
}

type RunInput struct {
	fx.In

	Log       *zap.Logger
	Minecraft *shulker.Minecraft
	// Controller *shulker.Controller
}

func Run(in RunInput) {
	in.Log.Info("Running Shulker")
}

type SetupEnvironmentInput struct {
	fx.In

	Log    *zap.Logger
	Config shulker.Config
}

func SetupEnvironment(in SetupEnvironmentInput) error {
	in.Log.Info("Setup Shulker Environment")

	if !utility.FileExists(in.Log, in.Config.WorkingDir) {
		in.Log.Info("Creating Shulker Working Directory", zap.String("path", in.Config.WorkingDir))

		if err := os.MkdirAll(in.Config.WorkingDir, 0744); err != nil {
			in.Log.Error("Failed to create working directory", zap.String("path", in.Config.WorkingDir), zap.Error(err))
			return err
		}
	}

	if !utility.FileExists(in.Log, in.Config.ServerJar()) {
		in.Log.Info("Downloading ServerJar", zap.String("path", in.Config.ServerJar()))

		if err := utility.DownloadFile(in.Log, in.Config.Minecraft.Server.DownloadURL, in.Config.ServerJar()); err != nil {
			in.Log.Error("Error downloading server jar", zap.Error(err))
			return err
		}
	}

	return nil
}
