package main

import (
	"flag"
	"log"
	"os"
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
			case <-time.After(serverGracefulTimeoutKillAttemptWait):
				os.Exit(16)
			}
		}
	case err := <-eventbus.Subscribe(eventNameServerStopped).Receive():
		spew.Dump(err)
	}
}
