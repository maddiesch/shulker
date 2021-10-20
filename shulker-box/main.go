package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"shulker-box/eventbus"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
)

var (
	serverRestartAttemptWait             = 1 * time.Second
	serverGracefulShutdownAttemptWait    = 24 * time.Second
	serverGracefulTimeoutKillAttemptWait = 100 * time.Millisecond
	serverRestartMaxAttempts             = 10

	errServerShouldBadExit = errors.New("server should exit with bad error code")
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC | log.Lmsgprefix)
	log.SetPrefix("[shulker] ")
}

func main() {
	var updateFlagVal bool
	var configPathVal string
	flag.BoolVar(&updateFlagVal, "update", false, "specify if the server should be updated")
	flag.StringVar(&configPathVal, "config", "./config.shulker.hcl", "path to the shulker working dir")
	flag.Parse()

	cfg, err := loadAndParseShulkerConfigAtFilePath(configPathVal)
	if err != nil {
		log.Fatal(err)
	}

	if err := performSetupWithForcedUpdate(cfg, updateFlagVal); err != nil {
		log.Fatal(err)
	}

	eventbus.InstallSignalEvent(eventbus.Default, os.Interrupt, syscall.SIGTERM)

	go startMinecraftServer(cfg)

	select {
	case <-eventbus.Subscribe(eventbus.SignalEvent).Receive():
		eventbus.Dispatch(`minecraft:command`, `stop`)

		select {
		case event := <-eventbus.Subscribe(eventNameServerStopped).Receive():
			if val := event.Value(); val != nil {
				log.Fatal(val)
			} else {
				os.Exit(0)
			}
		case <-time.After(serverGracefulShutdownAttemptWait):
			eventbus.Dispatch(`minecraft:kill_server`, nil)

			select {
			case <-eventbus.Subscribe(eventNameServerStopped).Receive():
			case <-time.After(1 * time.Second):
				os.Exit(16)
			}
		}
	case err := <-eventbus.Subscribe(eventNameServerStopped).Receive():
		spew.Dump(err)
	}

	/**

	serverCommandChan := make(chan []byte, 1)
	serverSignalChan := make(chan serverSig, 1)
	serverChan := make(chan error, 1)
	interuptChan := make(chan os.Signal, 1)

	signal.Notify(interuptChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		defer close(serverChan)
		defer close(serverSignalChan)
		defer close(serverCommandChan)

		var attempts int

		for {
			attempts += 1

			if err := runMinecraftServer(ctx, serverSignalChan, serverCommandChan, cfg); err != nil {
				if !errors.Is(err, errServerStopped) {
					serverChan <- err
				}
				runtime.Goexit()
			} else if !cfg.Minecraft.AutoRestart {
				runtime.Goexit()
			} else if attempts >= serverRestartMaxAttempts {
				log.Printf("Failed to start Minecraft Server after %d attempts... We're gonna crash now :(", attempts)
				serverChan <- errServerShouldBadExit
				runtime.Goexit()
			}
			log.Println(`Minecraft Server Shutdown. Attempting Restart`)
			<-time.After(serverRestartAttemptWait)
		}
	}()

	select {
	case <-ctx.Done():
	case <-interuptChan:
		log.Printf(`Server Interupt Attempting Graceful Shutdown (%s)`, serverGracefulShutdownAttemptWait)

		go func() {
			log.Println(`Sending stop signal to Minecraft Server Runner`)
			serverSignalChan <- serverSig_Stop
			log.Println(`Minecraft Runner Handled Stop Signal`)
		}()

		select {
		case <-time.After(serverGracefulShutdownAttemptWait):
			go func() {
				log.Println(`Server Shutdown Failed ... Killing`)
				serverSignalChan <- serverSig_Kill
				log.Println(`Server Shutdown Killed`)
			}()
			<-time.After(serverGracefulTimeoutKillAttemptWait)
			os.Exit(64)
		case err := <-serverChan:
			handleServerChanResult(err)
		}
	case err := <-serverChan:
		handleServerChanResult(err)
	}

	*/
}

type serverSig uint8

const (
	_ serverSig = iota
	serverSig_Stop
	serverSig_Kill
)

func handleServerChanResult(err error) {
	var osExitErr *exec.ExitError

	if errors.Is(err, errServerShouldBadExit) {
		os.Exit(100)
	} else if errors.As(err, &osExitErr) {
		switch osExitErr.ExitCode() {
		case 130:
			log.Printf("{MINECRAFT_EXIT} - 130")
		default:
			log.Fatal(osExitErr)
		}
	} else if err != nil {
		log.Fatal(err)
	}

	log.Println("Minecraft server shutdown complete")
	os.Exit(0)
}
