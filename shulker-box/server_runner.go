package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/davecgh/go-spew/spew"
)

var rLog = log.New(os.Stderr, "[shulker-runner] ", 0)

var errServerStopped = errors.New("minecraft server stopped")

var mcLaunchDoneRegexp = regexp.MustCompile(`:\sDone\s\(\d+\.\d+.\)!\sFor\shelp,\stype\s"help"`)
var mcClosedRegexp = regexp.MustCompile(`:\sClosing\sServer`)

type stateWriter struct {
	on    string
	state *uint64
}

func (w *stateWriter) Write(p []byte) (int, error) {
	if mcClosedRegexp.Match(p) {
		atomic.StoreUint64(w.state, 2)
	}
	if mcLaunchDoneRegexp.Match(p) {
		atomic.StoreUint64(w.state, 1)
	}
	return len(p), nil
}

func runMinecraftServer(ctx context.Context, sig <-chan serverSig, msg <-chan []byte, cfg shulkerConfig) error {
	rLog.Println(`Starting Minecraft Server`)

	var mu sync.Mutex

	var currentState uint64

	var cmdArgs []string
	cmdArgs = append(cmdArgs, cfg.Minecraft.Java.Flags...)
	cmdArgs = append(cmdArgs, `-jar`, cfg.Minecraft.Server.JarPath, `--nogui`)

	cmd := exec.Command(cfg.Minecraft.Java.Command, cmdArgs...)
	cmd.Dir = cfg.WorkingDir
	cmd.Stdout = io.MultiWriter(os.Stdout, &stateWriter{`stdout`, &currentState})
	cmd.Stderr = io.MultiWriter(os.Stderr, &stateWriter{`stderr`, &currentState})

	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer cmdIn.Close()

	runningChan := make(chan struct{})
	defer close(runningChan)

	var gotStopSig bool

	var sendMu sync.Mutex

	send := func(m []byte) {
		sendMu.Lock()
		defer sendMu.Unlock()

		if !bytes.HasSuffix(m, []byte{'\n', '\r'}) {
			m = append(m, '\n', '\r')
		}

		_, err := cmdIn.Write(m)
		if err != nil {
			rLog.Printf("Failed to write to server sub-process (%v)", err)
		}
	}

	go func() {
		defer rLog.Print(`  Server Command Handler Stopped`)

		for {
			select {
			case <-runningChan:
				runtime.Goexit()
			case message := <-msg:
				send(message)
			}
		}
	}()

	go func() {
		defer rLog.Print(`  Server Signal Handler Stopped`)

		for {
			select {
			case <-runningChan:
				runtime.Goexit()
			case code := <-sig:
				mu.Lock()

				rLog.Printf("(signal) %d", code)

				switch code {
				case serverSig_Stop:
					send([]byte("stop"))
					gotStopSig = true
				case serverSig_Kill:
					cmd.Process.Kill()
				default:
					log.Printf(`unexpected signal: %d`, code)
				}

				mu.Unlock()
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return err
	}

	err = cmd.Wait()

	mu.Lock()
	defer mu.Unlock()

	spew.Dump(atomic.LoadUint64(&currentState))

	var osExitErr *exec.ExitError

	if errors.As(err, &osExitErr) {
		if osExitErr.ExitCode() == 130 && gotStopSig {
			return errServerStopped
		}
	}

	return err
}
