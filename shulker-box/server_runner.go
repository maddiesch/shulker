package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/angryboat/go-dispatch"
)

var rLog = log.New(os.Stderr, "[shulker-mc] ", logFlags)
var mLog = log.New(os.Stderr, "[minecraft] ", logFlags)

const (
	dispatchEventName_MinecraftStopped      = `minecraft.server_stopped`
	dispatchEventName_MinecraftStateChanged = `minecraft.server_state_changed`
	dispatchEventName_MinecraftCommand      = `minecraft.send_command`
)

func runMinecraftServer(cfg shulkerConfig) {
	defer dispatch.Send(dispatch.NullEvent(dispatchEventName_MinecraftStopped))

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

		killCancel := dispatch.Receive(context.Background(), dispatchEventName_Kill, func(ctx context.Context, e dispatch.Event) {
			rLog.Print(`Received Kill...`)
			cmd.Process.Kill()
		})
		defer killCancel()

		shutdownCancel := dispatch.Receive(context.Background(), dispatchEventName_Shutdown, func(ctx context.Context, e dispatch.Event) {
			rLog.Print(`Received Shutdown...`)

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

func (m *minecraftWriter) Write(p []byte) (int, error) {
	for _, v := range bytes.Split(bytes.TrimSpace(p), []byte{'\n'}) {
		mLog.Print(string(v))
	}

	return len(p), nil
}
