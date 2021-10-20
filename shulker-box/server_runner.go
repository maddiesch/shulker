package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"shulker-box/eventbus"
	"time"
)

var rLog = log.New(os.Stderr, "[shulker-runner] ", log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Lmsgprefix)
var mLog = log.New(os.Stderr, "[minecraft] ", log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Lmsgprefix)

const (
	eventNameServerStopped = `server_stopped`
)

func startMinecraftServer(cfg shulkerConfig) {
	rLog.Println(`Starting Minecraft Subsystem`)

	defer eventbus.Dispatch(eventNameServerStopped, nil)

	stateListener := eventbus.Listen(`state_changed`, func(unsafeState interface{}) {
		rLog.Printf(`State Changed %v`, unsafeState)
	})
	defer stateListener.Close()

	execute := func() error {
		var cmdArgs []string
		cmdArgs = append(cmdArgs, cfg.Minecraft.Java.Flags...)
		cmdArgs = append(cmdArgs, `-jar`, cfg.Minecraft.Server.JarPath, `--nogui`)

		cmd := exec.Command(cfg.Minecraft.Java.Command, cmdArgs...)
		cmd.Dir = cfg.WorkingDir
		cmd.Stdout = io.MultiWriter(&minecraftWriter{}, &stateWriter{`stdout`})
		cmd.Stderr = io.MultiWriter(os.Stderr, &stateWriter{`stderr`})

		cmdIn, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		defer cmdIn.Close()

		kill := eventbus.Listen(`minecraft:kill_server`, func(interface{}) {
			cmd.Process.Kill()
		})
		defer kill.Close()

		command := eventbus.Listen(`minecraft:command`, func(unsafeCmd interface{}) {
			var val []byte
			switch v := unsafeCmd.(type) {
			case string:
				val = []byte(v)
			case []byte:
				val = v
			}

			if !bytes.HasSuffix(val, []byte{'\n', '\r'}) {
				val = append(val, '\n', '\r')
			}

			cmdIn.Write(val)
		})
		defer command.Close()

		return cmd.Run()
	}

	var attempts int

	for {
		attempts += 1

		rLog.Printf("Starting Minecraft Server (%d)", attempts)

		err := execute()

		if err != nil {
			eventbus.Dispatch(`minecraft:server_error`, err)
			runtime.Goexit()
		} else if cfg.Minecraft.AutoRestart {
			if attempts > serverRestartMaxAttempts {
				runtime.Goexit()
			}
			<-time.After(serverRestartAttemptWait)
		}
	}
}

var mcLaunchDoneRegexp = regexp.MustCompile(`:\sDone\s\(\d+\.\d+.\)!\sFor\shelp,\stype\s"help"`)
var mcClosedRegexp = regexp.MustCompile(`:\sClosing\sServer`)
var mcStartingRegexp = regexp.MustCompile(`:\sStarting\sMinecraft\sserver\son`)

type stateWriter struct {
	on string
}

func (w *stateWriter) Write(p []byte) (int, error) {
	if mcStartingRegexp.Match(p) {
		eventbus.Dispatch(`state_changed`, `starting`)
	}
	if mcClosedRegexp.Match(p) {
		eventbus.Dispatch(`state_changed`, `closing`)
	}
	if mcLaunchDoneRegexp.Match(p) {
		eventbus.Dispatch(`state_changed`, `started`)
	}

	return len(p), nil
}

type minecraftWriter struct{}

func (m *minecraftWriter) Write(p []byte) (int, error) {
	for _, v := range bytes.Split(bytes.TrimSpace(p), []byte{'\n'}) {
		mLog.Print(string(v))
	}

	return len(p), nil
}
