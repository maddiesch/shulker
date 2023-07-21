package shulker

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/do"
	"golang.org/x/exp/slog"
)

type ControlServerService struct {
	server *http.Server
	logger *slog.Logger
	bus    *EventBusService
}

func NewControlServerService(i *do.Injector) (*ControlServerService, error) {
	config, err := do.Invoke[Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config instance")
	}

	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get logger instance")
	}

	bus, err := do.Invoke[*EventBusService](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bus instance")
	}

	handler, err := do.Invoke[*ControlServerHandlerService](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handler instance")
	}

	return &ControlServerService{
		logger: logger.With(slog.String("subsystem", "control-server")),
		bus:    bus,
		server: &http.Server{
			Addr:    net.JoinHostPort(config.ServerAddress, config.ServerPort),
			Handler: handler,
		},
	}, nil
}

func (s *ControlServerService) Start() error {
	s.logger.Info("Starting HTTP Control Server", slog.String("addr", s.server.Addr))

	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return errors.Wrap(err, "failed to create tcp listener")
	}

	go func() {
		err := s.server.Serve(listener)

		if errors.Is(err, http.ErrServerClosed) {
			s.logger.Debug("HTTP Server Stopped with ErrServerClosed")
		} else if err != nil {
			s.bus.ShutdownWithError(err)
		}
	}()

	return nil
}

func (s *ControlServerService) Shutdown() error {
	s.logger.Debug("Shutdown server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
