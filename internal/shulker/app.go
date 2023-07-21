package shulker

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/samber/do"
	"golang.org/x/exp/slog"
)

type NewAppInput struct {
	*slog.Logger
	Config
}

type App struct {
	injector *do.Injector
}

func NewApp(in NewAppInput) *App {
	injector := do.New()

	do.ProvideValue(injector, in.Logger)
	do.ProvideValue(injector, in.Config)
	do.Provide[*EventBusService](injector, NewEventBusService)
	do.Provide[*ControlServerService](injector, NewControlServerService)

	return &App{
		injector: injector,
	}
}

func (a *App) Wait() <-chan error {
	bus := do.MustInvoke[*EventBusService](a.injector)
	event, cancel := bus.Sink()

	done := make(chan error, 1)

	go func() {
		for e := range event {
			if e.Name == EventNameShutdown {
				close(done)
				cancel()
				runtime.Goexit()
			}
		}
	}()

	return done
}

func (a *App) Start(_ context.Context) error {
	log := do.MustInvoke[*slog.Logger](a.injector)
	log.Info("Starting Shulker")

	bus, err := do.Invoke[*EventBusService](a.injector)
	if err != nil {
		return err
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		<-signalChan

		a.sendShutdownEvent()

		select {
		case <-time.After(30 * time.Second):
			os.Exit(15)
		case <-signalChan:
			os.Exit(16)
		}
	}()

	event, cancel := bus.Sink()

	go func() {
		defer cancel()

		for e := range event {
			switch e.Name {
			case EventNameShutdown:
				log.Debug("Shutdown event received... stopping error handler")
				runtime.Goexit()
			case EventNameShutdownError:
				log.Error("Received error event... starting shutdown")
				a.sendShutdownEvent()
				runtime.Goexit()
			}
		}
	}()

	server, err := do.Invoke[*ControlServerService](a.injector)
	if err != nil {
		return err
	}

	if err := server.Start(); err != nil {
		return err
	}

	return nil
}

func (a *App) sendShutdownEvent() {
	bus := do.MustInvoke[*EventBusService](a.injector)
	bus.Publish(Event{
		Name: EventNameShutdown,
	})
}

func (a *App) Stop(_ context.Context) error {
	log := do.MustInvoke[*slog.Logger](a.injector)
	log.Info("Stopping Shulker")

	return a.injector.Shutdown()
}
