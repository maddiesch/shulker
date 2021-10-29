package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"shulker-box/config"
	"shulker-box/logger"
	"strings"
	"sync/atomic"
	"time"

	"github.com/angryboat/go-dispatch"
)

var rLog = logger.L.WithField(`subsystem`, `minecraft-runner`)
var mLog = logger.L.WithField(`subsystem`, `minecraft`)

const (
	dispatchEventName_MinecraftStopped      = `minecraft.server_stopped`
	dispatchEventName_MinecraftStateChanged = `minecraft.server_state_changed`
	dispatchEventName_MinecraftCommand      = `minecraft.send_command`
)

func runMinecraftServer(cfg config.Config) {
	defer dispatch.Send(dispatch.NullEvent(dispatchEventName_MinecraftStopped))

	javaCommand, err := cfg.JavaCommand()
	if err != nil {
		panic(err) // This is validate during setup
	}

	var receivedShutdown int32

	execute := func() error {
		var cmdArgs []string
		cmdArgs = append(cmdArgs, cfg.Minecraft.Java.Flags...)
		cmdArgs = append(cmdArgs, `-jar`, cfg.ServerJar(), `--nogui`)

		cmd := exec.Command(javaCommand, cmdArgs...)
		cmd.Dir = cfg.WorkingDir
		cmd.Stdout = io.MultiWriter(&minecraftWriter{}, &stateWriter{`stdout`})
		cmd.Stderr = io.MultiWriter(os.Stderr, &stateWriter{`stderr`})

		cmdIn, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		defer cmdIn.Close()

		killCancel := dispatch.Receive(context.Background(), dispatchEventName_Kill, func(ctx context.Context, e dispatch.Event) {
			rLog.Print(`Received Kill...`)
			atomic.AddInt32(&receivedShutdown, 1)

			cmd.Process.Kill()
		})
		defer killCancel()

		shutdownCancel := dispatch.Receive(context.Background(), dispatchEventName_Shutdown, func(ctx context.Context, e dispatch.Event) {
			rLog.Print(`Received Shutdown...`)

			atomic.AddInt32(&receivedShutdown, 1)

			go sendServerCommandEvent(`stop`)
		})
		defer shutdownCancel()

		commandCancel := dispatch.Receive(context.Background(), dispatchEventName_MinecraftCommand, func(ctx context.Context, e dispatch.Event) {
			rLog.Print(`Received Command...`)

			switch val := e.Value().(type) {
			case []byte:
				cmdIn.Write(val)
			case string:
				cmdIn.Write([]byte(val))
			default:
				rLog.Printf(`failed to send command with type - %t`, val)
			}
		})
		defer commandCancel()

		return cmd.Run()
	}

	var attempts int

	for {
		attempts += 1

		rLog.Printf("Starting Minecraft Server (%d/%d)", attempts, serverRestartMaxAttempts)

		err := execute()

		if atomic.SwapInt32(&receivedShutdown, 0) > 0 {
			rLog.Print(`Mincraft Server Shutdown Complete`)
			runtime.Goexit()
		}

		if err != nil {
			runtime.Goexit()
		} else if cfg.Minecraft.AutoRestart {
			if attempts > serverRestartMaxAttempts {
				runtime.Goexit()
			}
			<-time.After(serverRestartAttemptWait)
		}
	}
}

func sendServerCommandEvent(input string) {
	if !strings.HasSuffix(input, "\n\r") {
		input += "\n\r"
	}

	dispatch.Send(dispatch.ValueEvent(dispatchEventName_MinecraftCommand, []byte(input)))
}

var mcLaunchDoneRegexp = regexp.MustCompile(`:\sDone\s\(\d+\.\d+.\)!\sFor\shelp,\stype\s"help"`)
var mcClosedRegexp = regexp.MustCompile(`:\sClosing\sServer`)
var mcStartingRegexp = regexp.MustCompile(`:\sStarting\sMinecraft\sserver\son`)

type stateWriter struct {
	on string
}

func (w *stateWriter) Write(p []byte) (int, error) {
	if mcStartingRegexp.Match(p) {
		dispatch.Send(dispatch.ValueEvent(dispatchEventName_MinecraftStateChanged, `starting`))
	}
	if mcClosedRegexp.Match(p) {
		dispatch.Send(dispatch.ValueEvent(dispatchEventName_MinecraftStateChanged, `closing`))
	}
	if mcLaunchDoneRegexp.Match(p) {
		dispatch.Send(dispatch.ValueEvent(dispatchEventName_MinecraftStateChanged, `started`))
	}

	return len(p), nil
}

type minecraftWriter struct{}

var minecraftLogCleanupRegexp = regexp.MustCompile(`\A\[\d{2}:\d{2}:\d{2}\s+[\w]+\]:\s+`)

func (m *minecraftWriter) Write(p []byte) (int, error) {
	for _, v := range bytes.Split(bytes.TrimSpace(p), []byte{'\n'}) {
		finalOut := minecraftLogCleanupRegexp.ReplaceAllString(string(v), "")

		mLog.Info(finalOut)
	}

	return len(p), nil
}
