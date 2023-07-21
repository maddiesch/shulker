package main

import (
	"context"
	"os"

	"github.com/maddiesch/shulker/internal/shulker"
	"golang.org/x/exp/slog"
)

func main() {
	config := shulker.Config{
		ServerPort:    "3000",
		ServerAddress: "127.0.0.1",
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app := shulker.NewApp(shulker.NewAppInput{
		Logger: log,
		Config: config,
	})

	if err := app.Start(context.Background()); err != nil {
		log.Error("Failed to start app", slog.String("error", err.Error()))
		os.Exit(8)
	}

	err := <-app.Wait()
	if err != nil {
		log.Error("App Failed", slog.String("error", err.Error()))
	}

	if err := app.Stop(context.Background()); err != nil {
		log.Error("Failed to start app", slog.String("error", err.Error()))
		os.Exit(8)
	}
}
