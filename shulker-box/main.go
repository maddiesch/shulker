package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/angryboat/go-dispatch"
	log "github.com/sirupsen/logrus"
)

var (
	serverRestartAttemptWait             = 1 * time.Second
	serverGracefulShutdownAttemptWait    = 24 * time.Second
	serverGracefulTimeoutKillAttemptWait = 100 * time.Millisecond
	serverRestartMaxAttempts             = 10
)

func main() {
	var updateFlagVal bool
	var configPathVal string
	flag.BoolVar(&updateFlagVal, "update", false, "specify if the server should be updated")
	flag.StringVar(&configPathVal, "config", "./config.shulker.hcl", "path to the shulker working dir")
	flag.Parse()

	ctx := context.Background()

	cfg, err := loadAndParseShulkerConfigAtFilePath(configPathVal)
	if err != nil {
		log.Fatal(err)
	}

	if err := performSetupWithForcedUpdate(cfg, updateFlagVal); err != nil {
		log.Fatal(err)
	}

	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go runMinecraftServer(cfg)
	go runControlServer(cfg)

	minecraftStopped := dispatch.Subscribe(dispatchEventName_MinecraftStopped)

	select {
	case <-signalChan:
		ctx, cancel := context.WithTimeout(ctx, serverGracefulShutdownAttemptWait)
		defer cancel()
		attemptGracefulShutdown(ctx, signalChan)
	case <-minecraftStopped.Receive():
		exitWithStatus(18) // TODO: - Shutdown Other
	}
}

var (
	dispatchEventName_Shutdown = `shulker.begin_shutdown`
	dispatchEventName_Kill     = `shulker.kill`
)

func attemptGracefulShutdown(ctx context.Context, sigChan <-chan os.Signal) {
	log.Print(`Attempting Graceful Shutdown`)

	dispatch.Send(dispatch.NullEvent(dispatchEventName_Shutdown))

	shutdown := awaitShutdownEvents()
	defer shutdown.Cancel()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(ctx, serverGracefulTimeoutKillAttemptWait)
		defer cancel()

		attemptKillShutdown(ctx)
	case <-shutdown.Receive():
		exitWithStatus(0)
	case <-sigChan:
		log.Print(`Gracefull shutdown interupt... killing`)
		ctx, cancel := context.WithTimeout(ctx, serverGracefulTimeoutKillAttemptWait)
		defer cancel()

		attemptKillShutdown(ctx)
	}
}

func attemptKillShutdown(ctx context.Context) {
	log.Print(`Shutdown Kill`)

	dispatch.Send(dispatch.NullEvent(dispatchEventName_Kill))

	shutdown := awaitShutdownEvents()
	defer shutdown.Cancel()

	select {
	case <-ctx.Done():
		log.Print(`Kill Timeout Exceeded`)
		exitWithStatus(8)
	case <-shutdown.Receive():
		log.Print(`Kill Successfull`)
		exitWithStatus(12)
	}
}

func awaitShutdownEvents() dispatch.Combine {
	return dispatch.Zip(
		dispatchEventName_MinecraftStopped,
		dispatchEventName_ControllerStopped,
	)
}

func exitWithStatus(status int) {
	if unsafeLogFilePtr != nil {
		unsafeLogFilePtr.Close()
	}
	os.Exit(status)
}
