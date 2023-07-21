package shulker

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	_ "embed"

	"github.com/maddiesch/go-raptor"
	"github.com/maddiesch/shulker/internal/shulker/model"
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
	injector := NewInjector(NewInjectorInput(in))

	return &App{
		injector: injector,
	}
}

type NewInjectorInput struct {
	*slog.Logger
	Config
}

func NewInjector(in NewInjectorInput) *do.Injector {
	injector := do.New()

	do.ProvideValue(injector, in.Logger)
	do.ProvideValue(injector, in.Config)
	do.Provide[*EventBusService](injector, NewEventBusService)
	do.Provide[*ControlServerService](injector, NewControlServerService)
	do.Provide[*ControlServerHandlerService](injector, NewControlServerHandlerService)
	do.Provide[*DatabaseService](injector, NewDatabaseService)

	return injector
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

const (
	kDBInitialSetup = ""
)

//go:embed new_user_warning.txt
var newUserWarning string

func (a *App) Start(ctx context.Context) error {
	log := do.MustInvoke[*slog.Logger](a.injector)
	log.Info("Starting Shulker")

	db := do.MustInvoke[*DatabaseService](a.injector)
	if err := db.ExecuteDatabaseMigration(ctx); err != nil {
		return err
	}

	if _, ok := db.Get(ctx, kDBInitialSetup); !ok {
		if err := performInitialDatabaseSetup(ctx, db, log); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, newUserWarning)
	}

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

	for name, err := range a.injector.HealthCheck() {
		if err != nil {
			log.Error("Service Health Check Failure", slog.String("service-name", name), slog.String("error", err.Error()))
			return err
		}
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

func performInitialDatabaseSetup(ctx context.Context, db *DatabaseService, log *slog.Logger) error {
	log.Info("Performing initial database setup")

	user := model.CreateUserParams{
		Username:    "admin",
		Password:    "password",
		Permissions: model.UserPermissionLogin | model.UserPermissionEditor | model.UserPermissionAdmin,
	}

	return db.conn.Transact(ctx, func(tx raptor.DB) error {
		if err := model.CreateUser(ctx, tx, user); err != nil {
			return err
		}
		return db.SetX(ctx, tx, kDBInitialSetup, []byte(time.Now().UTC().Format(time.RFC3339Nano)))
	})
}
