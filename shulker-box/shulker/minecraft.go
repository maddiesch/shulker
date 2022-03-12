package shulker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type NewMinecraftInput struct {
	fx.In

	Config    Config
	Lifecycle fx.Lifecycle
	Runtime   fx.Shutdowner
	Log       *zap.Logger
}

func NewMinecraft(ctx context.Context, in NewMinecraftInput) (*Minecraft, error) {
	m := &Minecraft{
		autoRestart: in.Config.Minecraft.AutoRestart,
		log:         in.Log.Named("Minecraft"),
		commandChan: make(chan []byte, 8),
		doneChan:    make(chan struct{}),
		restartChan: make(chan struct{}),
	}

	in.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return m.start(ctx, in.Config, in.Runtime)
		},
		OnStop: func(ctx context.Context) error {
			return m.Shutdown(ctx)
		},
	})

	return m, nil
}

type Minecraft struct {
	mu             sync.Mutex
	log            *zap.Logger
	commandChan    chan []byte
	doneChan       chan struct{}
	restartChan    chan struct{}
	shuttingDown   bool
	proc           *os.Process
	hasMissingEula bool
	autoRestart    bool
}

func (m *Minecraft) Command(s []byte) {
	if !bytes.HasSuffix(s, []byte{'\r', '\n'}) {
		s = append(s, '\r', '\n')
	}

	m.log.Debug("Sending Command", zap.ByteString("command", s))

	m.commandChan <- s
}

func (m *Minecraft) Shutdown(ctx context.Context) error {
	m.log.Debug("Shutdown")

	m.mu.Lock()
	m.shuttingDown = true
	m.mu.Unlock()

	m.Command([]byte("stop"))

	select {
	case <-m.doneChan:
		return nil
	case <-ctx.Done():
		m.mu.Lock()
		if m.proc != nil {
			m.log.Error("Failed to stop the process")
			m.proc.Kill()
		}
		m.mu.Unlock()
		return ctx.Err()
	}
}

func (m *Minecraft) start(ctx context.Context, config Config, app fx.Shutdowner) error {
	if os.Getenv("DISABLE_MINECRAFT_PROCESS") == "true" {
		close(m.doneChan)
		return nil
	}

	javaCommand, err := config.JavaCommand()
	if err != nil {
		return err
	}

	m.log.Debug("Starting Minecraft", zap.String("java", javaCommand))

	go func() {
		defer func() {
			m.mu.Lock()
			m.proc = nil
			m.mu.Unlock()
			close(m.doneChan)
		}()

		for m.running() {
			if err := m.run(ctx, javaCommand, config); err != nil {
				app.Shutdown()
				runtime.Goexit()
			} else if !m.autoRestart {
				app.Shutdown()
				runtime.Goexit()
			}
		}
	}()

	return nil
}

func (m *Minecraft) run(ctx context.Context, javaPath string, config Config) error {
	m.mu.Lock()

	if m.hasMissingEula {
		m.mu.Unlock()
		return errors.New("missing eula")
	}

	if m.restartChan != nil {
		close(m.restartChan)
	}
	m.restartChan = make(chan struct{})

	m.mu.Unlock()

	var cmdArgs []string
	cmdArgs = append(cmdArgs, config.Java.Flags...)
	cmdArgs = append(cmdArgs, `-jar`, config.ServerJar(), `--nogui`)

	cmd := exec.Command(javaPath, cmdArgs...)
	cmd.Dir = config.WorkingDir
	cmd.Stdout = io.MultiWriter(&minecraftLogWatcher{m}, &zapMinecraftOutput{m.log.Named("Stdout")})
	cmd.Stderr = io.MultiWriter(&minecraftLogWatcher{m}, &zapMinecraftOutput{m.log.Named("Stderr")})

	cmdIn, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			m.log.Debug("Stopped Command Processing")
		}()

		for {
			select {
			case <-m.doneChan:
				runtime.Goexit()
			case <-m.restartChan:
				runtime.Goexit()
			case cmd := <-m.commandChan:
				if _, err := cmdIn.Write(cmd); err != nil {
					m.log.Error("Failed to write to Minecraft Process", zap.Error(err))
				}
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		m.log.Error("Failed to start Minecraft Process", zap.Error(err))
		return err
	}

	m.mu.Lock()
	m.proc = cmd.Process
	m.mu.Unlock()

	return cmd.Wait()
}

func (m *Minecraft) running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return !m.shuttingDown
}

func (m *Minecraft) setEulaMissing() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Warn("Minecraft Missing EULA file")

	m.hasMissingEula = true
}

func (m *Minecraft) setServerProbablyReady() {
	m.log.Debug("Minecraft Server _Probably_ Ready")
}

func (m *Minecraft) setServerProbablyStopping() {
	m.log.Debug("Minecraft Server _Probably_ Stopping")
}

type minecraftLogWatcher struct {
	m *Minecraft
}

var (
	minecraftMissingEulaLogContent    = []byte("You need to agree to the EULA in order to run the server")
	minecraftServerStartedLogContent  = []byte(`For help, type "help"`)
	minecraftServerStoppingLogContent = []byte("Stopping the server")
)

func (w *minecraftLogWatcher) Write(p []byte) (int, error) {
	if bytes.Contains(p, minecraftMissingEulaLogContent) {
		w.m.setEulaMissing()
	}
	if bytes.Contains(p, minecraftServerStartedLogContent) {
		w.m.setServerProbablyReady()
	}
	if bytes.Contains(p, minecraftServerStoppingLogContent) {
		w.m.setServerProbablyStopping()
	}
	return len(p), nil
}

type zapMinecraftOutput struct {
	log *zap.Logger
}

func (z *zapMinecraftOutput) Write(p []byte) (int, error) {
	z.log.Info(string(bytes.TrimSpace(p)))

	return len(p), nil
}
