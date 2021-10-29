package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"shulker-box/config"
	"shulker-box/logger"
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

var logFileWriter io.WriteCloser

func main() {
	var updateFlagVal bool
	var configPathVal string
	var logFilePathVal string
	var logLevelVal string

	flag.BoolVar(&updateFlagVal, "update", false, "specify if the server should be updated")
	flag.StringVar(&configPathVal, "config", "./config.shulker.hcl", "path to the shulker working dir")
	flag.StringVar(&logFilePathVal, "log", "./shulker.log", "path to the shulker log")
	flag.StringVar(&logLevelVal, "loglevel", "info", "logging level")
	flag.Parse()

	var err error

	logFileWriter, err = logger.CreateLog(logFilePathVal)
	if err != nil {
		failWithError(err)
	}

	logLevel, err := log.ParseLevel(logLevelVal)
	if err != nil {
		failWithError(err)
	}

	logger.L.SetOutput(io.MultiWriter(os.Stdout, logFileWriter))
	logger.L.SetLevel(logLevel)
	logger.L.Debugf(`Logger setup with level %s`, logLevel)

	ctx := context.Background()

	cfg, err := config.Load(ctx, configPathVal)
	if err != nil {
		failWithError(err)
	}

	if err := prepareShulkerForRunning(ctx, cfg, updateFlagVal); err != nil {
		failWithError(err)
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
	logger.L.Info(`Attempting Graceful Shutdown`)

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
	logger.L.Warn(`Shutdown Kill`)

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
	if logFileWriter != nil {
		logFileWriter.Close()
	}
	os.Exit(status)
}

func failWithError(err error) {
	logger.L.Errorf(`Fatal Error: %v`, err)

	exitWithStatus(44)
}
