package main

import (
	"context"
	"flag"
	"os"

	"github.com/maddiesch/shulker/internal/shulker"
	"golang.org/x/exp/slog"
)

func main() {
	var config shulker.Config

	flag.StringVar(&config.ServerPort, "server-port", "3000", "control server port")
	flag.StringVar(&config.ServerAddress, "server-addr", "127.0.0.1", "control server address")
	flag.StringVar(&config.DatabasePath, "database", "./shulker.db", "path to the shulker database")

	flag.Parse()

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
